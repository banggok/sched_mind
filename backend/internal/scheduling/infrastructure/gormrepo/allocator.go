package gormrepo

import (
	"fmt"
	"math/big"
	"sort"
	"time"

	schedulingdomain "github.com/banggok/sched_mind/backend/internal/scheduling/domain"
)

type dailyAllocation struct {
	TaskID   string
	MemberID string
	Date     time.Time
	Minutes  *big.Rat
	Sequence int
	Fixed    bool
}

type taskSchedule struct {
	TaskID              string
	Start               *time.Time
	End                 *time.Time
	Reason              *string
	Allocations         []dailyAllocation
	ManualCandidateDate *time.Time
}

type timelineResult struct {
	timeline         schedulingdomain.Timeline
	schedules        map[string]taskSchedule
	automaticBlocker map[string]string
	calendar         *allocationCalendar
}

type allocationCalendar struct {
	state       *portfolioState
	timeline    schedulingdomain.Timeline
	allocations map[string]map[string][]dailyAllocation
	sequence    int
}

func newAllocationCalendar(state *portfolioState, timeline schedulingdomain.Timeline) *allocationCalendar {
	return &allocationCalendar{
		state:       state,
		timeline:    timeline,
		allocations: make(map[string]map[string][]dailyAllocation),
	}
}

func (state *portfolioState) scheduleTimeline(timeline schedulingdomain.Timeline) (*timelineResult, error) {
	result := &timelineResult{
		timeline:         timeline,
		schedules:        make(map[string]taskSchedule),
		automaticBlocker: make(map[string]string),
		calendar:         newAllocationCalendar(state, timeline),
	}
	if err := result.reserveFixedTasks(); err != nil {
		return nil, err
	}

	pending := make(map[string]taskModel)
	for _, task := range state.tasks {
		project := state.projects[task.ProjectID]
		if _, leaf := state.leafOrder[task.ID]; !leaf {
			continue
		}
		if task.ActualEnd != nil || project.Status != "open" || !project.AutomaticScheduling {
			continue
		}
		if project.SchedulingStartDate == nil {
			result.schedules[task.ID] = unscheduled(task.ID, reasonMissingAnchor)
			continue
		}
		if task.AssigneeID == nil || *task.AssigneeID == "" {
			result.schedules[task.ID] = unscheduled(task.ID, reasonMissingAssignee)
			continue
		}
		if task.EffortMinutes == nil || *task.EffortMinutes < 30 || *task.EffortMinutes%30 != 0 {
			result.schedules[task.ID] = unscheduled(task.ID, reasonMissingEffort)
			continue
		}
		if _, exists := state.members[*task.AssigneeID]; !exists {
			result.schedules[task.ID] = unscheduled(task.ID, reasonMissingAssignee)
			continue
		}
		pending[task.ID] = task
	}

	for len(pending) > 0 {
		ready := make([]taskModel, 0)
		for taskID, task := range pending {
			if result.dependenciesResolved(taskID, pending) {
				ready = append(ready, task)
			}
		}
		if len(ready) == 0 {
			for taskID := range pending {
				result.schedules[taskID] = unscheduled(taskID, reasonBlocked)
			}
			break
		}
		sort.Slice(ready, func(left, right int) bool {
			return compareTaskOrder(state, ready[left], ready[right])
		})
		selected := ready[0]
		schedule, displaced, err := result.allocateTask(selected)
		if err != nil {
			return nil, err
		}
		result.invalidateDisplaced(displaced, pending)
		schedule.Allocations = result.calendar.commit(selected, schedule.Allocations)
		result.schedules[selected.ID] = schedule
		delete(pending, selected.ID)
	}

	if timeline == schedulingdomain.Execution {
		result.deriveAutomaticBlockers()
	}
	return result, nil
}

func unscheduled(taskID, reason string) taskSchedule {
	return taskSchedule{TaskID: taskID, Reason: stringPointer(reason), Allocations: []dailyAllocation{}}
}

func (result *timelineResult) dependenciesResolved(taskID string, pending map[string]taskModel) bool {
	for _, blockerID := range result.calendar.state.blockers(taskID) {
		if _, stillPending := pending[blockerID]; stillPending {
			return false
		}
	}
	return true
}

func (result *timelineResult) reserveFixedTasks() error {
	values := append([]taskModel{}, result.calendar.state.fixed[result.timeline]...)
	sort.Slice(values, func(left, right int) bool {
		leftStart, _ := timelineDates(values[left], result.timeline)
		rightStart, _ := timelineDates(values[right], result.timeline)
		if leftStart == nil || rightStart == nil {
			return compareTaskOrder(result.calendar.state, values[left], values[right])
		}
		if !leftStart.Equal(*rightStart) {
			return leftStart.Before(*rightStart)
		}
		return compareTaskOrder(result.calendar.state, values[left], values[right])
	})
	for _, task := range values {
		if _, leaf := result.calendar.state.leafOrder[task.ID]; !leaf || task.AssigneeID == nil || task.EffortMinutes == nil {
			continue
		}
		start, end := timelineDates(task, result.timeline)
		if start == nil || end == nil {
			continue
		}
		if task.ActualEnd != nil && end.After(*task.ActualEnd) {
			end = datePointer(*task.ActualEnd)
		}
		allocations, usedProjection, err := result.calendar.reserveProjectedFixed(task, *start, *end)
		if err != nil {
			return err
		}
		if !usedProjection {
			allocations, err = result.calendar.reconstructFixed(task, *start, *end)
			if err != nil {
				return err
			}
		}
		result.schedules[task.ID] = taskSchedule{TaskID: task.ID, Start: start, End: end, Allocations: allocations}
	}
	return nil
}

func timelineDates(task taskModel, timeline schedulingdomain.Timeline) (*time.Time, *time.Time) {
	if timeline == schedulingdomain.Execution {
		return task.ExecutionStart, task.ExecutionEnd
	}
	return task.CommitmentStart, task.CommitmentEnd
}

func (calendar *allocationCalendar) reserveProjectedFixed(
	task taskModel,
	start time.Time,
	end time.Time,
) ([]dailyAllocation, bool, error) {
	rows := calendar.state.existingAllocations[calendar.timeline][task.ID]
	if len(rows) == 0 {
		return nil, false, nil
	}
	if task.AssigneeID == nil || task.EffortMinutes == nil {
		return nil, false, nil
	}
	total := new(big.Rat)
	for _, row := range rows {
		date := schedulingdomain.DateOnly(row.AllocationDate)
		if row.AssigneeID != *task.AssigneeID || date.Before(schedulingdomain.DateOnly(start)) || date.After(schedulingdomain.DateOnly(end)) {
			return nil, false, nil
		}
		minutes, err := schedulingdomain.ParseDecimal(row.AllocatedMinutes)
		if err != nil || minutes.Sign() <= 0 {
			return nil, false, fmt.Errorf("%w: invalid persisted allocation for fixed task %s", schedulingdomain.ErrDataIntegrity, task.ID)
		}
		total.Add(total, minutes)
	}
	if total.Cmp(big.NewRat(int64(*task.EffortMinutes), 1)) != 0 {
		return nil, false, nil
	}

	allocations := make([]dailyAllocation, 0, len(rows))
	for _, row := range rows {
		minutes, _ := schedulingdomain.ParseDecimal(row.AllocatedMinutes)
		allocation := calendar.add(task.ID, *task.AssigneeID, row.AllocationDate, minutes, true)
		allocations = append(allocations, allocation)
	}
	return allocations, true, nil
}

func (calendar *allocationCalendar) reconstructFixed(task taskModel, start, end time.Time) ([]dailyAllocation, error) {
	start = schedulingdomain.DateOnly(start)
	end = schedulingdomain.DateOnly(end)
	if end.Before(start) {
		return nil, fmt.Errorf("%w: fixed task %s has end before start", schedulingdomain.ErrDataIntegrity, task.ID)
	}

	remaining := big.NewRat(int64(*task.EffortMinutes), 1)
	allocations := make([]dailyAllocation, 0)
	for date := start; !date.After(end) && remaining.Sign() > 0; date = date.AddDate(0, 0, 1) {
		available, err := calendar.remaining(*task.AssigneeID, task.ProjectID, date)
		if err != nil {
			return nil, err
		}
		if available.Sign() <= 0 {
			continue
		}
		allocated := minRat(remaining, available)
		allocation := calendar.add(task.ID, *task.AssigneeID, date, allocated, true)
		allocations = append(allocations, allocation)
		remaining.Sub(remaining, allocated)
	}
	if remaining.Sign() > 0 {
		if task.ActualEnd != nil {
			// Completion is immutable historical evidence, not a claim that the
			// Task's planned Effort still fits today's capacity configuration.
			// Keep any unresolved historical Effort on Actual End so the Task
			// consumes no future capacity while the completed timeline remains
			// usable by dependency and same-assignee readiness calculations.
			allocation := calendar.add(task.ID, *task.AssigneeID, end, remaining, true)
			allocations = append(allocations, allocation)
			return allocations, nil
		}
		return nil, fmt.Errorf("%w: fixed task %s exceeds its locked/manual timeline capacity", schedulingdomain.ErrDataIntegrity, task.ID)
	}
	return allocations, nil
}

func (result *timelineResult) allocateTask(task taskModel) (taskSchedule, []string, error) {
	candidate, _, blocked, err := result.readiness(task)
	if err != nil {
		return taskSchedule{}, nil, err
	}
	if blocked {
		return unscheduled(task.ID, reasonBlocked), nil, nil
	}
	manualCandidate := schedulingdomain.DateOnly(candidate)

	remaining := big.NewRat(int64(*task.EffortMinutes), 1)
	for attempts := 0; attempts < maximumScheduleDays; attempts++ {
		allocations, displaced, nextCandidate, completed, err := result.calendar.simulateContiguous(task, candidate, remaining)
		if err != nil {
			return taskSchedule{}, nil, err
		}
		if completed {
			start := allocations[0].Date
			end := allocations[len(allocations)-1].Date
			return taskSchedule{
				TaskID:              task.ID,
				Start:               datePointer(start),
				End:                 datePointer(end),
				Allocations:         allocations,
				ManualCandidateDate: datePointer(manualCandidate),
			}, displaced, nil
		}
		if nextCandidate.IsZero() {
			return unscheduled(task.ID, reasonZeroCapacity), nil, nil
		}
		candidate = nextCandidate
	}
	return unscheduled(task.ID, reasonZeroCapacity), nil, nil
}

func (result *timelineResult) invalidateDisplaced(taskIDs []string, pending map[string]taskModel) {
	if len(taskIDs) == 0 {
		return
	}
	state := result.calendar.state
	queue := append([]string{}, taskIDs...)
	seen := make(map[string]bool)
	for len(queue) > 0 {
		taskID := queue[0]
		queue = queue[1:]
		if seen[taskID] {
			continue
		}
		seen[taskID] = true
		task, exists := state.tasks[taskID]
		if !exists {
			continue
		}
		project := state.projects[task.ProjectID]
		if task.ActualEnd != nil || project.Status != "open" || !project.AutomaticScheduling {
			continue
		}
		result.calendar.removeTask(taskID)
		delete(result.schedules, taskID)
		pending[taskID] = task
		for candidateID := range result.schedules {
			for _, blockerID := range state.blockers(candidateID) {
				if blockerID == taskID {
					queue = append(queue, candidateID)
					break
				}
			}
		}
	}
}

func (result *timelineResult) readiness(task taskModel) (time.Time, bool, bool, error) {
	state := result.calendar.state
	project := state.projects[task.ProjectID]
	candidate := schedulingdomain.DateOnly(*project.SchedulingStartDate).AddDate(0, 0, task.LagDays)
	blockers := state.blockers(task.ID)
	if len(blockers) == 0 {
		return candidate, false, false, nil
	}

	var controlling time.Time
	strictNextDay := false
	for _, blockerID := range blockers {
		blocker, exists := state.tasks[blockerID]
		if !exists {
			return time.Time{}, false, true, nil
		}
		anchor := blocker.ActualEnd
		if anchor == nil {
			schedule, scheduled := result.schedules[blockerID]
			if !scheduled || schedule.End == nil {
				return time.Time{}, false, true, nil
			}
			anchor = schedule.End
		}
		anchorDate := schedulingdomain.DateOnly(*anchor)
		if controlling.IsZero() || anchorDate.After(controlling) {
			controlling = anchorDate
			strictNextDay = blocker.AssigneeID == nil || task.AssigneeID == nil || *blocker.AssigneeID != *task.AssigneeID
		} else if anchorDate.Equal(controlling) && (blocker.AssigneeID == nil || task.AssigneeID == nil || *blocker.AssigneeID != *task.AssigneeID) {
			strictNextDay = true
		}
	}
	candidate = controlling.AddDate(0, 0, task.LagDays)
	if task.LagDays > 0 {
		strictNextDay = false
	} else if strictNextDay {
		candidate = controlling.AddDate(0, 0, 1)
	}
	return candidate, !strictNextDay && task.LagDays == 0, false, nil
}

func (calendar *allocationCalendar) simulateContiguous(
	task taskModel,
	candidate time.Time,
	effort *big.Rat,
) ([]dailyAllocation, []string, time.Time, bool, error) {
	remaining := schedulingdomain.CloneRat(effort)
	allocations := make([]dailyAllocation, 0)
	displaced := make(map[string]struct{})
	started := false
	date := schedulingdomain.DateOnly(candidate)
	for days := 0; days < maximumScheduleDays; days++ {
		base, err := calendar.capacity(*task.AssigneeID, task.ProjectID, date)
		if err != nil {
			return nil, nil, time.Time{}, false, err
		}
		if base.Sign() <= 0 {
			date = date.AddDate(0, 0, 1)
			continue
		}

		existing := calendar.dayExcluding(*task.AssigneeID, date, displaced)
		for _, allocation := range existing {
			if allocation.Fixed {
				continue
			}
			existingTask, exists := calendar.state.tasks[allocation.TaskID]
			if exists &&
				!calendar.state.isEffectiveBlocker(task.ID, allocation.TaskID) &&
				compareTaskOrder(calendar.state, task, existingTask) {
				displaced[allocation.TaskID] = struct{}{}
			}
		}
		existing = calendar.dayExcluding(*task.AssigneeID, date, displaced)

		if len(existing) > 0 && started {
			nextCandidate := date
			for _, allocation := range existing {
				end := calendar.taskEnd(allocation.TaskID)
				if end.After(nextCandidate) {
					nextCandidate = end
				}
			}
			return nil, nil, nextCandidate, false, nil
		}

		remainingCapacity := schedulingdomain.CloneRat(base)
		for _, allocation := range existing {
			remainingCapacity.Sub(remainingCapacity, allocation.Minutes)
		}
		if remainingCapacity.Sign() <= 0 {
			date = date.AddDate(0, 0, 1)
			continue
		}
		allocated := minRat(remaining, remainingCapacity)
		if allocated.Sign() > 0 {
			started = true
			allocations = append(allocations, dailyAllocation{
				TaskID: task.ID, MemberID: *task.AssigneeID, Date: date, Minutes: schedulingdomain.CloneRat(allocated),
			})
			remaining.Sub(remaining, allocated)
			if remaining.Sign() == 0 {
				return allocations, sortedKeys(displaced), time.Time{}, true, nil
			}
		}
		date = date.AddDate(0, 0, 1)
	}
	return nil, nil, time.Time{}, false, nil
}

func (calendar *allocationCalendar) commit(task taskModel, allocations []dailyAllocation) []dailyAllocation {
	committed := make([]dailyAllocation, len(allocations))
	for index, allocation := range allocations {
		committed[index] = calendar.add(
			task.ID,
			*task.AssigneeID,
			allocation.Date,
			allocation.Minutes,
			false,
		)
	}
	return committed
}

func (calendar *allocationCalendar) removeTask(taskID string) {
	for memberID, byDate := range calendar.allocations {
		for dateKey, values := range byDate {
			kept := values[:0]
			for _, allocation := range values {
				if allocation.TaskID != taskID || allocation.Fixed {
					kept = append(kept, allocation)
				}
			}
			if len(kept) == 0 {
				delete(byDate, dateKey)
			} else {
				byDate[dateKey] = kept
			}
		}
		if len(byDate) == 0 {
			delete(calendar.allocations, memberID)
		}
	}
}

func (calendar *allocationCalendar) dayExcluding(memberID string, date time.Time, excluded map[string]struct{}) []dailyAllocation {
	values := calendar.day(memberID, date)
	if len(excluded) == 0 {
		return values
	}
	out := make([]dailyAllocation, 0, len(values))
	for _, allocation := range values {
		if _, skip := excluded[allocation.TaskID]; !skip {
			out = append(out, allocation)
		}
	}
	return out
}

func sortedKeys(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func (calendar *allocationCalendar) taskEnd(taskID string) time.Time {
	var latest time.Time
	for _, byDate := range calendar.allocations {
		for _, values := range byDate {
			for _, allocation := range values {
				if allocation.TaskID == taskID && (latest.IsZero() || allocation.Date.After(latest)) {
					latest = allocation.Date
				}
			}
		}
	}
	return latest
}

func (calendar *allocationCalendar) add(taskID, memberID string, date time.Time, minutes *big.Rat, fixed bool) dailyAllocation {
	calendar.sequence++
	allocation := dailyAllocation{
		TaskID: taskID, MemberID: memberID, Date: schedulingdomain.DateOnly(date),
		Minutes: schedulingdomain.CloneRat(minutes), Sequence: calendar.sequence, Fixed: fixed,
	}
	key := schedulingdomain.DateKey(date)
	if calendar.allocations[memberID] == nil {
		calendar.allocations[memberID] = make(map[string][]dailyAllocation)
	}
	calendar.allocations[memberID][key] = append(calendar.allocations[memberID][key], allocation)
	return allocation
}

func (calendar *allocationCalendar) day(memberID string, date time.Time) []dailyAllocation {
	return calendar.allocations[memberID][schedulingdomain.DateKey(date)]
}

func (calendar *allocationCalendar) remaining(memberID, projectID string, date time.Time) (*big.Rat, error) {
	capacity, err := calendar.capacity(memberID, projectID, date)
	if err != nil {
		return nil, err
	}
	remaining := schedulingdomain.CloneRat(capacity)
	allReservationsFixed := true
	for _, allocation := range calendar.day(memberID, date) {
		remaining.Sub(remaining, allocation.Minutes)
		if !allocation.Fixed {
			allReservationsFixed = false
		}
	}
	if remaining.Sign() < 0 {
		// Completed, locked, and manual schedules are immutable reservations.
		// A later capacity reduction must not invalidate their historical
		// projection; it simply leaves no capacity for newly generated work.
		if allReservationsFixed {
			return new(big.Rat), nil
		}
		return nil, fmt.Errorf("%w: member %s is overallocated on %s", schedulingdomain.ErrDataIntegrity, memberID, schedulingdomain.DateKey(date))
	}
	return remaining, nil
}

func (calendar *allocationCalendar) capacity(memberID, projectID string, date time.Time) (*big.Rat, error) {
	state := calendar.state
	member, exists := state.members[memberID]
	if !exists {
		return new(big.Rat), nil
	}
	project, exists := state.projects[projectID]
	if !exists {
		return nil, schedulingdomain.ErrDataIntegrity
	}
	resolved := new(big.Rat)
	if !schedulingdomain.IsWeekend(date) {
		if _, holiday := state.holidays[schedulingdomain.DateKey(date)]; !holiday {
			capacityValue := member.DailyCapacity
			for _, override := range state.overrides[memberID] {
				day := schedulingdomain.DateOnly(date)
				if !day.Before(schedulingdomain.DateOnly(override.StartDate)) && !day.After(schedulingdomain.DateOnly(override.EndDate)) {
					capacityValue = override.Capacity
					break
				}
			}
			value, err := schedulingdomain.ParseDecimal(capacityValue)
			if err != nil {
				return nil, err
			}
			resolved = value
		}
	}
	return schedulingdomain.CapacityMinutes(resolved, member.BufferPercentage, project.ProjectBuffer, calendar.timeline)
}

func minRat(left, right *big.Rat) *big.Rat {
	if left.Cmp(right) <= 0 {
		return schedulingdomain.CloneRat(left)
	}
	return schedulingdomain.CloneRat(right)
}

func (result *timelineResult) deriveAutomaticBlockers() {
	state := result.calendar.state
	byMember := make(map[string][]taskSchedule)
	for taskID, schedule := range result.schedules {
		task := state.tasks[taskID]
		if schedule.Start == nil || schedule.End == nil || len(schedule.Allocations) == 0 || task.AssigneeID == nil {
			continue
		}
		byMember[*task.AssigneeID] = append(byMember[*task.AssigneeID], schedule)
	}
	for memberID := range byMember {
		values := byMember[memberID]
		sort.Slice(values, func(left, right int) bool {
			if !values[left].Start.Equal(*values[right].Start) {
				return values[left].Start.Before(*values[right].Start)
			}
			return values[left].Allocations[0].Sequence < values[right].Allocations[0].Sequence
		})
		for index, schedule := range values {
			task := state.tasks[schedule.TaskID]
			project := state.projects[task.ProjectID]
			if index == 0 || schedule.ManualCandidateDate == nil || schedule.Start == nil || task.ActualEnd != nil || project.Status != "open" || !project.AutomaticScheduling {
				continue
			}
			previous := values[index-1]
			if previous.End == nil {
				continue
			}
			resourceDelayed := schedule.Start.After(*schedule.ManualCandidateDate)
			if !resourceDelayed && schedule.Start.Equal(*schedule.ManualCandidateDate) {
				resourceDelayed = previous.End.Equal(*schedule.Start) && previous.Allocations[len(previous.Allocations)-1].Sequence < schedule.Allocations[0].Sequence
			}
			if resourceDelayed {
				result.automaticBlocker[schedule.TaskID] = previous.TaskID
			}
		}
	}
}
