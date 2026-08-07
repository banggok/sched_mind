package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	schedulingdomain "github.com/banggok/sched_mind/backend/internal/scheduling/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/identity"
	"github.com/banggok/sched_mind/backend/internal/shared/schedulingimpact"
	"github.com/banggok/sched_mind/backend/internal/wbs/domain"
)

type GroupSchedulingInput struct {
	ExpectedVersion     int64
	Name                *string
	Source              string
	AutomaticScheduling *bool
	SchedulingStartDate *time.Time
}

type WriteExecutableInput struct {
	Name                         *string
	RoleID, AssigneeID           *string
	EffortMinutes                *int
	LagDays                      int
	CapacityAllocationPercentage *int
	Execution, Commitment        domain.Timeline
}

type PreviewExecutableInput struct {
	RoleID, AssigneeID           *string
	EffortMinutes                *int
	LagDays                      int
	CapacityAllocationPercentage int
}

type SchedulePreview struct {
	Task *domain.Node
}

type AllocationRow struct {
	Date                         time.Time
	AllocatedMinutes             int
	CapacityMinutes              int
	RemainingMinutes             int
	OvercapacityMinutes          int
	CapacityAllocationPercentage int
	TaskDailyLimitMinutes        int
}

type AllocationGroups struct {
	Execution  []AllocationRow
	Commitment []AllocationRow
	Actual     []AllocationRow
}

func (input PreviewExecutableInput) Validate() error {
	if input.CapacityAllocationPercentage < 0 || input.CapacityAllocationPercentage > 100 {
		return domain.ErrCapacityAllocationInvalid
	}
	if input.LagDays < 0 {
		return domain.ErrLagInvalid
	}
	if input.RoleID == nil || strings.TrimSpace(*input.RoleID) == "" ||
		input.EffortMinutes == nil {
		return domain.ErrSchedulePreviewIncomplete
	}
	return nil
}

type Store interface {
	Tree(context.Context, string) ([]domain.Node, error)
	Find(context.Context, string, string) (*domain.Node, error)
	Allocations(context.Context, string, string) (*AllocationGroups, error)
	Create(context.Context, string, string, *string, string, bool, time.Time, func(context.Context, string) error, func(context.Context, []string) error) (*domain.Node, error)
	CreateSibling(context.Context, string, string, string, string, time.Time, func(context.Context, string) error) (*domain.Node, error)
	Rename(context.Context, string, string, string, time.Time) (*domain.Node, error)
	UpdateGroupScheduling(context.Context, string, string, GroupSchedulingInput, time.Time, func(context.Context, string) error) (*domain.Node, error)
	ChangeGroupStatus(context.Context, string, string, string, int64, time.Time, func(context.Context, string) error) (*domain.Node, error)
	BulkReopenGroup(context.Context, string, string, string, int64, time.Time, func(context.Context, string) error) (*domain.Node, error)
	UpdateExecutable(context.Context, string, string, WriteExecutableInput, time.Time, func(context.Context, string) error) (*domain.Node, error)
	PreviewExecutableSchedule(context.Context, string, string, PreviewExecutableInput, time.Time, func(context.Context, string) error) (*SchedulePreview, error)
	Complete(context.Context, string, string, time.Time, time.Time, time.Time, func(context.Context, []string) error) (*domain.Node, error)
	Reopen(context.Context, string, string, time.Time, func(context.Context, []string) error) (*domain.Node, error)
	Reorder(context.Context, string, string, domain.Direction, time.Time, func(context.Context, string) error) error
	Place(context.Context, string, string, string, domain.Placement, time.Time, func(context.Context, string) error) error
	Move(context.Context, string, string, string, *string, bool, time.Time, func(context.Context, string) error, func(context.Context, []string) error) error
	Delete(context.Context, string, string, time.Time, func(context.Context, string) error, func(context.Context, []string) error) error
}

type AssigneeRecommender interface {
	RecommendAssignees(context.Context, schedulingdomain.AssigneeRecommendationInput) (*schedulingdomain.AssigneeRecommendationResult, error)
}

type Scheduler interface {
	RecalculateProjectSchedule(context.Context, string) error
	RecalculateProjectForecast(context.Context, string) error
	InvalidatePortfolio(context.Context, []string) error
}
type NoopScheduler struct{}

func (NoopScheduler) RecalculateProjectSchedule(context.Context, string) error { return nil }
func (NoopScheduler) RecalculateProjectForecast(context.Context, string) error { return nil }
func (NoopScheduler) InvalidatePortfolio(context.Context, []string) error      { return nil }

type Service struct {
	store       Store
	scheduler   Scheduler
	recommender AssigneeRecommender
	now         func() time.Time
	newID       func() (string, error)
}

func NewService(store Store, scheduler Scheduler) *Service {
	if scheduler == nil {
		scheduler = NoopScheduler{}
	}
	recommender, _ := scheduler.(AssigneeRecommender)
	return &Service{store: store, scheduler: scheduler, recommender: recommender, now: func() time.Time { return time.Now().UTC() }, newID: identity.NewUUID}
}
func NewServiceWithDependencies(store Store, scheduler Scheduler, now func() time.Time, newID func() (string, error)) *Service {
	recommender, _ := scheduler.(AssigneeRecommender)
	return &Service{store: store, scheduler: scheduler, recommender: recommender, now: now, newID: newID}
}

func (s *Service) Tree(ctx context.Context, projectID string) ([]domain.Node, error) {
	value, err := s.store.Tree(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("get WBS tree: %w", err)
	}
	return value, nil
}
func (s *Service) Get(ctx context.Context, projectID, id string) (*domain.Node, error) {
	value, err := s.store.Find(ctx, projectID, id)
	if err != nil {
		return nil, fmt.Errorf("get WBS: %w", err)
	}
	if value == nil {
		return nil, errors.New("get WBS: store returned nil")
	}
	return value, nil
}
func (s *Service) Allocations(ctx context.Context, projectID, id string) (*AllocationGroups, error) {
	value, err := s.store.Allocations(ctx, projectID, id)
	if err != nil {
		return nil, fmt.Errorf("get WBS allocations: %w", err)
	}
	if value == nil {
		return nil, errors.New("get WBS allocations: store returned nil")
	}
	return value, nil
}
func (s *Service) Create(ctx context.Context, projectID string, parentID *string, name string, confirm bool) (*domain.Node, error) {
	ctx = schedulingimpact.WithOperation(ctx, projectID, schedulingimpact.ModeOrdinary)
	if _, err := domain.NormalizeName(name); err != nil {
		return nil, err
	}
	id, err := s.newID()
	if err != nil {
		return nil, fmt.Errorf("generate WBS ID: %w", err)
	}
	value, err := s.store.Create(ctx, id, projectID, parentID, name, confirm, s.now(), s.scheduler.RecalculateProjectSchedule, s.scheduler.InvalidatePortfolio)
	if err != nil {
		return nil, fmt.Errorf("create WBS: %w", err)
	}
	if value == nil {
		return nil, errors.New("create WBS: store returned nil")
	}
	return value, nil
}
func (s *Service) CreateSibling(ctx context.Context, projectID, insertAfterID, name string) (*domain.Node, error) {
	ctx = schedulingimpact.WithOperation(ctx, projectID, schedulingimpact.ModeOrdinary)
	if strings.TrimSpace(insertAfterID) == "" {
		return nil, domain.ErrCreatePositionInvalid
	}
	if _, err := domain.NormalizeName(name); err != nil {
		return nil, err
	}
	id, err := s.newID()
	if err != nil {
		return nil, fmt.Errorf("generate WBS ID: %w", err)
	}
	value, err := s.store.CreateSibling(ctx, id, projectID, insertAfterID, name, s.now(), s.scheduler.RecalculateProjectSchedule)
	if err != nil {
		return nil, fmt.Errorf("create sibling WBS: %w", err)
	}
	if value == nil {
		return nil, errors.New("create sibling WBS: store returned nil")
	}
	return value, nil
}
func (s *Service) Rename(ctx context.Context, p, id, name string) (*domain.Node, error) {
	value, err := s.store.Rename(ctx, p, id, name, s.now())
	if err != nil {
		return nil, fmt.Errorf("rename WBS: %w", err)
	}
	if value == nil {
		return nil, errors.New("rename WBS: store returned nil")
	}
	return value, nil
}
func (s *Service) UpdateGroupScheduling(ctx context.Context, p, id string, input GroupSchedulingInput) (*domain.Node, error) {
	ctx = schedulingimpact.WithGroupOperation(ctx, p, id, schedulingimpact.ModeOrdinary)
	value, err := s.store.UpdateGroupScheduling(ctx, p, id, input, s.now(), s.scheduler.RecalculateProjectSchedule)
	if err != nil {
		return nil, fmt.Errorf("update Group scheduling: %w", err)
	}
	if value == nil {
		return nil, errors.New("update Group scheduling: store returned nil")
	}
	return value, nil
}
func (s *Service) ChangeGroupStatus(ctx context.Context, p, id, target string, expectedVersion int64) (*domain.Node, error) {
	ctx = schedulingimpact.WithGroupOperation(ctx, p, id, schedulingimpact.ModeOrdinary)
	value, err := s.store.ChangeGroupStatus(ctx, p, id, target, expectedVersion, s.now(), s.scheduler.RecalculateProjectSchedule)
	if err != nil {
		return nil, fmt.Errorf("change Group status: %w", err)
	}
	if value == nil {
		return nil, errors.New("change Group status: store returned nil")
	}
	return value, nil
}
func (s *Service) BulkReopenGroup(ctx context.Context, p, id, token string, expectedVersion int64) (*domain.Node, error) {
	ctx = schedulingimpact.WithGroupOperation(ctx, p, id, schedulingimpact.ModeBulkReopen)
	value, err := s.store.BulkReopenGroup(ctx, p, id, token, expectedVersion, s.now(), s.scheduler.RecalculateProjectSchedule)
	if err != nil {
		return nil, fmt.Errorf("bulk reopen Group scope: %w", err)
	}
	if value == nil {
		return nil, errors.New("bulk reopen Group scope: store returned nil")
	}
	return value, nil
}
func (s *Service) UpdateExecutable(ctx context.Context, p, id string, input WriteExecutableInput) (*domain.Node, error) {
	ctx = schedulingimpact.WithOperation(ctx, p, schedulingimpact.ModeOrdinary)
	value, err := s.store.UpdateExecutable(ctx, p, id, input, s.now(), s.scheduler.RecalculateProjectSchedule)
	if err != nil {
		return nil, fmt.Errorf("update executable WBS: %w", err)
	}
	if value == nil {
		return nil, errors.New("update executable WBS: store returned nil")
	}
	return value, nil
}

func (s *Service) PreviewExecutableSchedule(ctx context.Context, p, id string, input PreviewExecutableInput) (*SchedulePreview, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	value, err := s.store.PreviewExecutableSchedule(ctx, p, id, input, s.now(), s.scheduler.RecalculateProjectSchedule)
	if err != nil {
		return nil, fmt.Errorf("preview executable WBS schedule: %w", err)
	}
	if value == nil || value.Task == nil {
		return nil, errors.New("preview executable WBS schedule: store returned nil")
	}
	return value, nil
}
func (s *Service) RecommendAssignees(ctx context.Context, projectID, taskID string, input schedulingdomain.AssigneeRecommendationInput) (*schedulingdomain.AssigneeRecommendationResult, error) {
	if s.recommender == nil {
		return nil, schedulingdomain.ErrAssigneeRecommendationUnavailable
	}
	input.ProjectID = projectID
	input.TaskID = taskID
	input.CalculatedOn = s.now()
	if err := input.Validate(); err != nil {
		return nil, err
	}
	value, err := s.recommender.RecommendAssignees(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("recommend assignees: %w", err)
	}
	if value == nil {
		return nil, errors.New("recommend assignees: scheduler returned nil")
	}
	return value, nil
}

func (s *Service) Complete(ctx context.Context, p, id string, actualStart, actualEnd time.Time) (*domain.Node, error) {
	ctx = schedulingimpact.WithOperation(ctx, p, schedulingimpact.ModeActualDate)
	value, err := s.store.Complete(ctx, p, id, actualStart, actualEnd, s.now(), s.scheduler.InvalidatePortfolio)
	if err != nil {
		return nil, fmt.Errorf("complete WBS: %w", err)
	}
	if value == nil {
		return nil, errors.New("complete WBS: store returned nil")
	}
	return value, nil
}
func (s *Service) Reopen(ctx context.Context, p, id string) (*domain.Node, error) {
	ctx = schedulingimpact.WithOperation(ctx, p, schedulingimpact.ModeOrdinary)
	value, err := s.store.Reopen(ctx, p, id, s.now(), s.scheduler.InvalidatePortfolio)
	if err != nil {
		return nil, fmt.Errorf("reopen WBS: %w", err)
	}
	if value == nil {
		return nil, errors.New("reopen WBS: store returned nil")
	}
	return value, nil
}
func (s *Service) Reorder(ctx context.Context, p, id string, d domain.Direction) error {
	ctx = schedulingimpact.WithOperation(ctx, p, schedulingimpact.ModeOrdinary)
	if d != domain.MoveUp && d != domain.MoveDown {
		return domain.ErrMoveNotAllowed
	}
	if err := s.store.Reorder(ctx, p, id, d, s.now(), s.scheduler.RecalculateProjectSchedule); err != nil {
		return fmt.Errorf("reorder WBS: %w", err)
	}
	return nil
}
func (s *Service) Place(ctx context.Context, p, id, targetID string, placement domain.Placement) error {
	ctx = schedulingimpact.WithOperation(ctx, p, schedulingimpact.ModeOrdinary)
	if strings.TrimSpace(targetID) == "" || targetID == id || (placement != domain.PlaceBefore && placement != domain.PlaceAfter) {
		return domain.ErrReorderTargetInvalid
	}
	if err := s.store.Place(ctx, p, id, targetID, placement, s.now(), s.scheduler.RecalculateProjectSchedule); err != nil {
		return fmt.Errorf("place WBS: %w", err)
	}
	return nil
}
func (s *Service) Move(ctx context.Context, p, id string, parent *string, confirm bool) error {
	ctx = schedulingimpact.WithOperation(ctx, p, schedulingimpact.ModeOrdinary)
	conversionID, err := s.newID()
	if err != nil {
		return fmt.Errorf("generate conversion WBS ID: %w", err)
	}
	if err := s.store.Move(ctx, p, id, conversionID, parent, confirm, s.now(), s.scheduler.RecalculateProjectSchedule, s.scheduler.InvalidatePortfolio); err != nil {
		return fmt.Errorf("move WBS: %w", err)
	}
	return nil
}
func (s *Service) Delete(ctx context.Context, p, id string) error {
	ctx = schedulingimpact.WithOperation(ctx, p, schedulingimpact.ModeOrdinary)
	if err := s.store.Delete(ctx, p, id, s.now(), s.scheduler.RecalculateProjectSchedule, s.scheduler.InvalidatePortfolio); err != nil {
		return fmt.Errorf("delete WBS: %w", err)
	}
	return nil
}
