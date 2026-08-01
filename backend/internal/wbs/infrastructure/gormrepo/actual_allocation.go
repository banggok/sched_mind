package gormrepo

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"time"

	schedulingdomain "github.com/banggok/sched_mind/backend/internal/scheduling/domain"
	"github.com/banggok/sched_mind/backend/internal/wbs/application"
	"github.com/banggok/sched_mind/backend/internal/wbs/domain"
	"gorm.io/gorm"
)

type actualAllocationModel struct {
	TaskID                   string
	AssigneeID               string
	Timeline                 string
	AllocationDate           time.Time
	AllocatedMinutes         string
	RemainingCapacityMinutes string
	Sequence                 int
}

func (actualAllocationModel) TableName() string { return "task_schedule_allocations" }

type allocationMemberModel struct {
	ID               string
	DailyCapacity    string
	BufferPercentage string
}

func (allocationMemberModel) TableName() string { return "team_members" }

type allocationOverrideModel struct {
	TeamMemberID string
	StartDate    time.Time
	EndDate      time.Time
	Capacity     string
	DeletedAt    *time.Time
}

func (allocationOverrideModel) TableName() string { return "capacity_overrides" }

type allocationHolidayModel struct {
	Date time.Time
}

func (allocationHolidayModel) TableName() string { return "public_holiday_dates" }

type actualAllocationValue struct {
	Date                time.Time
	AllocatedMinutes    int
	BAUCapacityMinutes  int
	RemainingMinutes    int
	OvercapacityMinutes int
}

func replaceActualAllocations(
	tx *gorm.DB,
	task nodeModel,
	baselineExecutionStart *time.Time,
	actualStart time.Time,
	actualEnd time.Time,
) ([]actualAllocationValue, error) {
	if err := tx.Where("task_id = ? AND timeline = ?", task.ID, string(schedulingdomain.Actual)).Delete(&actualAllocationModel{}).Error; err != nil {
		return nil, fmt.Errorf("replace actual allocations: %w", err)
	}
	if task.AssigneeID == nil || task.EffortMinutes == nil || *task.EffortMinutes <= 0 {
		return []actualAllocationValue{}, nil
	}

	calendar, err := loadActualAllocationCalendar(tx, *task.AssigneeID, task.ID)
	if err != nil {
		return nil, err
	}
	allocationStart := schedulingdomain.DateOnly(actualStart)
	if baselineExecutionStart != nil {
		baseline := schedulingdomain.DateOnly(*baselineExecutionStart)
		if baseline.Before(allocationStart) {
			allocationStart = baseline
		}
	}
	actualStart = schedulingdomain.DateOnly(actualStart)
	actualEnd = schedulingdomain.DateOnly(actualEnd)

	remaining := *task.EffortMinutes
	allocated := make(map[string]int)
	orderedDates := make([]time.Time, 0)
	appendAllocation := func(date time.Time, minutes int) {
		if minutes <= 0 {
			return
		}
		key := schedulingdomain.DateKey(date)
		if _, exists := allocated[key]; !exists {
			orderedDates = append(orderedDates, schedulingdomain.DateOnly(date))
		}
		allocated[key] += minutes
	}

	for day := allocationStart; day.Before(actualStart) && remaining > 0; day = day.AddDate(0, 0, 1) {
		if !calendar.working(day) {
			continue
		}
		available := calendar.remaining(day)
		if available <= 0 {
			continue
		}
		take := minInt(remaining, available)
		appendAllocation(day, take)
		remaining -= take
	}

	actualWorkingDates := make([]time.Time, 0)
	for day := actualStart; !day.After(actualEnd); day = day.AddDate(0, 0, 1) {
		if calendar.working(day) {
			actualWorkingDates = append(actualWorkingDates, day)
		}
	}
	if remaining > 0 && len(actualWorkingDates) > 0 {
		for _, day := range actualWorkingDates {
			if remaining == 0 {
				break
			}
			available := calendar.remaining(day)
			take := minInt(remaining, available)
			appendAllocation(day, take)
			remaining -= take
		}
		if remaining > 0 {
			perDate := remaining / len(actualWorkingDates)
			remainder := remaining % len(actualWorkingDates)
			for index, day := range actualWorkingDates {
				extra := perDate
				if index < remainder {
					extra++
				}
				appendAllocation(day, extra)
			}
			remaining = 0
		}
	}
	if remaining > 0 && len(actualWorkingDates) == 0 {
		day := actualEnd.AddDate(0, 0, 1)
		for !calendar.working(day) {
			day = day.AddDate(0, 0, 1)
		}
		appendAllocation(day, remaining)
		remaining = 0
	}

	sort.Slice(orderedDates, func(left, right int) bool { return orderedDates[left].Before(orderedDates[right]) })
	rows := make([]actualAllocationModel, 0, len(orderedDates))
	values := make([]actualAllocationValue, 0, len(orderedDates))
	for index, day := range orderedDates {
		key := schedulingdomain.DateKey(day)
		minutes := allocated[key]
		capacity := calendar.capacity(day)
		immutable := calendar.immutable[key]
		remainingCapacity := capacity - immutable - minutes
		overcapacity := 0
		if remainingCapacity < 0 {
			overcapacity = -remainingCapacity
			remainingCapacity = 0
		}
		rows = append(rows, actualAllocationModel{
			TaskID:                   task.ID,
			AssigneeID:               *task.AssigneeID,
			Timeline:                 string(schedulingdomain.Actual),
			AllocationDate:           day,
			AllocatedMinutes:         fmt.Sprintf("%d", minutes),
			RemainingCapacityMinutes: fmt.Sprintf("%d", remainingCapacity),
			Sequence:                 index + 1,
		})
		values = append(values, actualAllocationValue{
			Date:                day,
			AllocatedMinutes:    minutes,
			BAUCapacityMinutes:  capacity,
			RemainingMinutes:    remainingCapacity,
			OvercapacityMinutes: overcapacity,
		})
	}
	if len(rows) > 0 {
		if err := tx.Create(&rows).Error; err != nil {
			return nil, fmt.Errorf("persist actual allocations: %w", err)
		}
	}
	return values, nil
}

type actualCalendar struct {
	dailyCapacity *big.Rat
	memberBuffer  string
	overrides     []allocationOverrideModel
	holidays      map[string]struct{}
	immutable     map[string]int
}

func loadActualAllocationCalendar(tx *gorm.DB, memberID, taskID string) (*actualCalendar, error) {
	var member allocationMemberModel
	if err := tx.Where("id = ?", memberID).First(&member).Error; err != nil {
		return nil, fmt.Errorf("load actual allocation member: %w", err)
	}
	dailyCapacity, err := schedulingdomain.ParseDecimal(member.DailyCapacity)
	if err != nil {
		return nil, fmt.Errorf("load actual allocation daily capacity: %w", err)
	}
	var overrides []allocationOverrideModel
	if err := tx.Where("team_member_id = ? AND deleted_at IS NULL", memberID).Order("start_date ASC, end_date ASC, capacity ASC").Find(&overrides).Error; err != nil {
		return nil, fmt.Errorf("load actual allocation overrides: %w", err)
	}
	var holidayRows []allocationHolidayModel
	if err := tx.Find(&holidayRows).Error; err != nil {
		return nil, fmt.Errorf("load actual allocation holidays: %w", err)
	}
	holidays := make(map[string]struct{}, len(holidayRows))
	for _, holiday := range holidayRows {
		holidays[schedulingdomain.DateKey(holiday.Date)] = struct{}{}
	}

	type immutableRow struct {
		AllocationDate time.Time
		Minutes        int
	}
	var immutableRows []immutableRow
	query := `
		SELECT allocation.allocation_date,
		       CAST(SUM(CAST(allocation.allocated_minutes AS NUMERIC)) AS INTEGER) AS minutes
		FROM task_schedule_allocations AS allocation
		JOIN wbs_nodes AS task ON task.id = allocation.task_id
		JOIN projects AS project ON project.id = task.project_id
		WHERE allocation.assignee_id = ?
		  AND allocation.task_id <> ?
		  AND (
		        allocation.timeline = 'actual'
		        OR (allocation.timeline = 'execution' AND project.status = 'locked' AND task.actual_start IS NULL AND task.actual_end IS NULL)
		      )
		GROUP BY allocation.allocation_date`
	if err := tx.Raw(query, memberID, taskID).Scan(&immutableRows).Error; err != nil {
		return nil, fmt.Errorf("load immutable actual allocation reservations: %w", err)
	}
	immutable := make(map[string]int, len(immutableRows))
	for _, row := range immutableRows {
		immutable[schedulingdomain.DateKey(row.AllocationDate)] = row.Minutes
	}
	return &actualCalendar{dailyCapacity: dailyCapacity, memberBuffer: member.BufferPercentage, overrides: overrides, holidays: holidays, immutable: immutable}, nil
}

func (calendar *actualCalendar) working(date time.Time) bool {
	if schedulingdomain.IsWeekend(date) {
		return false
	}
	_, holiday := calendar.holidays[schedulingdomain.DateKey(date)]
	return !holiday
}

func (calendar *actualCalendar) resolvedHours(date time.Time) *big.Rat {
	if !calendar.working(date) {
		return new(big.Rat)
	}
	resolved := new(big.Rat).Set(calendar.dailyCapacity)
	var minimumOverride *big.Rat
	for _, override := range calendar.overrides {
		day := schedulingdomain.DateOnly(date)
		if day.Before(schedulingdomain.DateOnly(override.StartDate)) || day.After(schedulingdomain.DateOnly(override.EndDate)) {
			continue
		}
		candidate, err := schedulingdomain.ParseDecimal(override.Capacity)
		if err == nil && (minimumOverride == nil || candidate.Cmp(minimumOverride) < 0) {
			minimumOverride = candidate
		}
	}
	if minimumOverride != nil {
		resolved = minimumOverride
	}
	return resolved
}

func (calendar *actualCalendar) capacity(date time.Time) int {
	minutes := schedulingdomain.BAUCapacityMinutes(calendar.resolvedHours(date))
	return int(new(big.Int).Quo(minutes.Num(), minutes.Denom()).Int64())
}

func (calendar *actualCalendar) remaining(date time.Time) int {
	remaining := calendar.capacity(date) - calendar.immutable[schedulingdomain.DateKey(date)]
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (r *Repository) Allocations(ctx context.Context, projectID, taskID string) (*application.AllocationGroups, error) {
	if _, err := loadProject(r.db.WithContext(ctx), projectID, false); err != nil {
		return nil, err
	}
	task, err := findNode(r.db.WithContext(ctx), projectID, taskID, false)
	if err != nil {
		return nil, err
	}
	children, err := childCount(r.db.WithContext(ctx), projectID, taskID)
	if err != nil {
		return nil, err
	}
	if children > 0 {
		return nil, domain.ErrExecutableOnly
	}
	groups := &application.AllocationGroups{
		Execution:  []application.AllocationRow{},
		Commitment: []application.AllocationRow{},
		Actual:     []application.AllocationRow{},
	}
	if task.AssigneeID == nil {
		return groups, nil
	}
	calendar, err := loadActualAllocationCalendar(r.db.WithContext(ctx), *task.AssigneeID, taskID)
	if err != nil {
		return nil, err
	}
	var project projectModel
	if err := r.db.WithContext(ctx).Select("id, project_buffer").First(&project, "id = ?", projectID).Error; err != nil {
		return nil, fmt.Errorf("load allocation project: %w", err)
	}
	var rows []actualAllocationModel
	if err := r.db.WithContext(ctx).Where("task_id = ?", taskID).Order("timeline ASC, allocation_date ASC, sequence ASC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("load task allocations: %w", err)
	}
	for _, persisted := range rows {
		allocated, err := allocationInteger(persisted.AllocatedMinutes)
		if err != nil {
			return nil, fmt.Errorf("read allocated minutes: %w", err)
		}
		timeline := schedulingdomain.Timeline(persisted.Timeline)
		capacity, err := allocationCapacity(calendar, project.ProjectBuffer, persisted.AllocationDate, timeline)
		if err != nil {
			return nil, err
		}
		remaining, overcapacity := 0, 0
		if timeline == schedulingdomain.Actual {
			remaining = capacity - calendar.immutable[schedulingdomain.DateKey(persisted.AllocationDate)] - allocated
			if remaining < 0 {
				overcapacity = -remaining
				remaining = 0
			}
		} else {
			remaining, err = allocationInteger(persisted.RemainingCapacityMinutes)
			if err != nil {
				return nil, fmt.Errorf("read remaining capacity: %w", err)
			}
		}
		row := application.AllocationRow{
			Date:                schedulingdomain.DateOnly(persisted.AllocationDate),
			AllocatedMinutes:    allocated,
			CapacityMinutes:     capacity,
			RemainingMinutes:    remaining,
			OvercapacityMinutes: overcapacity,
		}
		switch timeline {
		case schedulingdomain.Execution:
			groups.Execution = append(groups.Execution, row)
		case schedulingdomain.Commitment:
			groups.Commitment = append(groups.Commitment, row)
		case schedulingdomain.Actual:
			groups.Actual = append(groups.Actual, row)
		default:
			return nil, fmt.Errorf("%w: unsupported allocation timeline %q", schedulingdomain.ErrDataIntegrity, persisted.Timeline)
		}
	}
	return groups, nil
}

func allocationCapacity(calendar *actualCalendar, projectBuffer int, date time.Time, timeline schedulingdomain.Timeline) (int, error) {
	if timeline == schedulingdomain.Actual {
		return calendar.capacity(date), nil
	}
	minutes, err := schedulingdomain.CapacityMinutes(calendar.resolvedHours(date), calendar.memberBuffer, projectBuffer, timeline)
	if err != nil {
		return 0, fmt.Errorf("resolve allocation capacity: %w", err)
	}
	return int(new(big.Int).Quo(minutes.Num(), minutes.Denom()).Int64()), nil
}

func allocationInteger(value string) (int, error) {
	decimal, err := schedulingdomain.ParseDecimal(value)
	if err != nil {
		return 0, err
	}
	if decimal.Denom().Cmp(big.NewInt(1)) != 0 {
		return 0, schedulingdomain.ErrDataIntegrity
	}
	return int(decimal.Num().Int64()), nil
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}
