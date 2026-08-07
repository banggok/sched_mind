package application

import (
	"context"
	"errors"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/banggok/sched_mind/backend/internal/roles/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/listing"
)

type memoryRepository struct {
	mu          sync.Mutex
	roles       map[string]domain.Role
	inUse       map[string]bool
	memberUsage map[string][]MemberUsage
	findNil     bool
	barrier     chan struct{}
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{
		roles:       make(map[string]domain.Role),
		inUse:       make(map[string]bool),
		memberUsage: make(map[string][]MemberUsage),
	}
}

func (repository *memoryRepository) List(_ context.Context, query listing.Query) (listing.Page[domain.Role], error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()

	roles := make([]domain.Role, 0, len(repository.roles))
	for _, role := range repository.roles {
		roles = append(roles, role)
	}
	sort.Slice(roles, func(left, right int) bool {
		return domain.NormalizedNameKey(roles[left].Name) <
			domain.NormalizedNameKey(roles[right].Name)
	})
	start := (query.Page - 1) * query.PageSize
	if start > len(roles) {
		start = len(roles)
	}
	end := start + query.PageSize
	if end > len(roles) {
		end = len(roles)
	}
	return listing.Page[domain.Role]{
		Items: roles[start:end], Page: query.Page, PageSize: query.PageSize, Total: int64(len(roles)),
	}, nil
}

func (repository *memoryRepository) ListMembers(
	_ context.Context,
	roleID string,
	query listing.Query,
) (listing.Page[MemberUsage], error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	items := append([]MemberUsage(nil), repository.memberUsage[roleID]...)
	return listing.Page[MemberUsage]{
		Items: items, Page: query.Page, PageSize: query.PageSize, Total: int64(len(items)),
	}, nil
}

func (repository *memoryRepository) FindByID(_ context.Context, id string) (*domain.Role, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()

	role, found := repository.roles[id]
	if !found {
		return nil, domain.ErrNotFound
	}
	if repository.findNil {
		return nil, nil
	}
	return &role, nil
}

func (repository *memoryRepository) NameExists(
	_ context.Context,
	nameKey string,
	excludeID string,
) (bool, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()

	for id, role := range repository.roles {
		if id != excludeID && domain.NormalizedNameKey(role.Name) == nameKey {
			return true, nil
		}
	}
	return false, nil
}

func (repository *memoryRepository) Create(_ context.Context, role domain.Role) error {
	repository.mu.Lock()
	defer repository.mu.Unlock()

	for _, existing := range repository.roles {
		if domain.NormalizedNameKey(existing.Name) == domain.NormalizedNameKey(role.Name) {
			return domain.ErrNameExists
		}
	}
	repository.roles[role.ID] = role
	return nil
}

func (repository *memoryRepository) Update(_ context.Context, role domain.Role) error {
	repository.mu.Lock()
	defer repository.mu.Unlock()

	if _, found := repository.roles[role.ID]; !found {
		return domain.ErrNotFound
	}
	for id, existing := range repository.roles {
		if id != role.ID &&
			domain.NormalizedNameKey(existing.Name) == domain.NormalizedNameKey(role.Name) {
			return domain.ErrNameExists
		}
	}
	repository.roles[role.ID] = role
	return nil
}

func (repository *memoryRepository) Delete(_ context.Context, id string) error {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if _, found := repository.roles[id]; !found {
		return domain.ErrNotFound
	}
	if repository.inUse[id] {
		return domain.ErrInUse
	}
	delete(repository.roles, id)
	return nil
}

func TestServiceRoleLifecycle(t *testing.T) {
	t.Parallel()

	repository := newMemoryRepository()
	now := time.Date(2026, 7, 24, 10, 0, 0, 0, time.UTC)
	service := NewServiceWithDependencies(
		repository,
		func() time.Time { return now },
		func() (string, error) { return "role-id", nil },
	)

	created, err := service.Create(context.Background(), " Backend ")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created == nil {
		t.Fatal("Create() result = nil, want role")
	}
	if created.Name != "Backend" {
		t.Fatalf("created.Name = %q, want Backend", created.Name)
	}

	now = now.Add(time.Hour)
	updated, err := service.Update(context.Background(), created.ID, "backend")
	if err != nil {
		t.Fatalf("Update() same role error = %v", err)
	}
	if updated.ID != created.ID {
		t.Fatalf("updated.ID = %q, want %q", updated.ID, created.ID)
	}

	if err := service.Delete(context.Background(), created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := repository.FindByID(context.Background(), created.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("FindByID() error = %v, want ErrNotFound", err)
	}
}

func TestServiceRejectsNilRepositoryResult(t *testing.T) {
	t.Parallel()

	repository := newMemoryRepository()
	repository.roles["role-id"] = domain.RehydrateRole(
		"role-id",
		"Backend",
		time.Now(),
		time.Now(),
	)
	repository.findNil = true
	service := NewService(repository)

	updated, err := service.Update(context.Background(), "role-id", "QA")
	if err == nil {
		t.Fatal("Update() error = nil, want repository contract error")
	}
	if updated != nil {
		t.Fatalf("Update() result = %#v, want nil", updated)
	}

}

func TestServiceRejectsDuplicateAndRoleInUse(t *testing.T) {
	t.Parallel()

	repository := newMemoryRepository()
	service := NewServiceWithDependencies(
		repository,
		time.Now,
		func() (string, error) { return "new-id", nil },
	)
	existing, err := domain.NewRole("existing-id", "Backend", time.Now())
	if err != nil {
		t.Fatalf("NewRole() error = %v", err)
	}
	repository.roles[existing.ID] = *existing
	repository.inUse[existing.ID] = true

	created, err := service.Create(context.Background(), "backend")
	if !errors.Is(err, domain.ErrNameExists) {
		t.Fatalf("Create() error = %v, want ErrNameExists", err)
	}
	if created != nil {
		t.Fatalf("Create() result = %#v, want nil on error", created)
	}
	if err := service.Delete(context.Background(), existing.ID); !errors.Is(err, domain.ErrInUse) {
		t.Fatalf("Delete() error = %v, want ErrInUse", err)
	}
	updated, err := service.Update(context.Background(), "missing", "QA")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Update() error = %v, want ErrNotFound", err)
	}
	if updated != nil {
		t.Fatalf("Update() result = %#v, want nil on error", updated)
	}
}

func TestConcurrentDuplicateCreation(t *testing.T) {
	t.Parallel()

	repository := newMemoryRepository()
	nextID := 0
	var idMutex sync.Mutex
	service := NewServiceWithDependencies(repository, time.Now, func() (string, error) {
		idMutex.Lock()
		defer idMutex.Unlock()
		nextID++
		return string(rune('0' + nextID)), nil
	})

	start := make(chan struct{})
	results := make(chan error, 2)
	for range 2 {
		go func() {
			<-start
			_, err := service.Create(context.Background(), "Backend")
			results <- err
		}()
	}
	close(start)

	var succeeded, conflicted int
	for range 2 {
		err := <-results
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, domain.ErrNameExists):
			conflicted++
		default:
			t.Fatalf("Create() unexpected error = %v", err)
		}
	}

	if succeeded != 1 || conflicted != 1 {
		t.Fatalf("successes = %d, conflicts = %d; want 1 and 1", succeeded, conflicted)
	}
}

func TestServiceListsMembersForExistingRole_US11_AC16(t *testing.T) {
	t.Parallel()

	repository := newMemoryRepository()
	repository.roles["role-id"] = domain.RehydrateRole(
		"role-id",
		"Backend",
		time.Now(),
		time.Now(),
	)
	repository.memberUsage["role-id"] = []MemberUsage{{ID: "member-1", Name: "Ayu"}}
	service := NewService(repository)

	result, err := service.ListMembers(
		context.Background(),
		"role-id",
		listing.Query{Page: 1, PageSize: 10},
	)
	if err != nil {
		t.Fatalf("ListMembers() error = %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 || result.Items[0].Name != "Ayu" {
		t.Fatalf("ListMembers() result = %#v, want Ayu", result)
	}
}

func TestServiceListMembersRequiresExistingRole(t *testing.T) {
	t.Parallel()

	repository := newMemoryRepository()
	service := NewService(repository)
	result, err := service.ListMembers(context.Background(), "missing", listing.Query{Page: 1, PageSize: 10})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("ListMembers() error = %v, want ErrNotFound", err)
	}
	if result.Total != 0 || len(result.Items) != 0 {
		t.Fatalf("ListMembers() result = %#v, want empty result on error", result)
	}
}
