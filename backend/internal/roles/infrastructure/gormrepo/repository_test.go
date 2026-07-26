package gormrepo

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/banggok/sched_mind/backend/internal/roles/application"
	"github.com/banggok/sched_mind/backend/internal/roles/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/listing"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openTestDatabase(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf(
		"file:%s?mode=memory&cache=shared&_foreign_keys=on&_busy_timeout=5000",
		t.Name(),
	)
	database, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open SQLite: %v", err)
	}
	sqlDatabase, err := database.DB()
	if err != nil {
		t.Fatalf("get SQL database: %v", err)
	}
	sqlDatabase.SetMaxOpenConns(1)

	if err := MigrateTestSchema(database); err != nil {
		t.Fatalf("migrate test schema: %v", err)
	}

	t.Cleanup(func() {
		_ = sqlDatabase.Close()
	})

	return database
}

func TestRepositorySortsRolesAndEnforcesCaseInsensitiveUniqueness(t *testing.T) {
	t.Parallel()

	database := openTestDatabase(t)
	repository := New(database)
	now := time.Now().UTC()
	for index, name := range []string{"QA", "backend", "Frontend"} {
		role, err := domain.NewRole(fmt.Sprintf("role-%d", index), name, now)
		if err != nil {
			t.Fatalf("NewRole() error = %v", err)
		}
		if role == nil {
			t.Fatal("NewRole() result = nil, want role")
		}
		if err := repository.Create(context.Background(), *role); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	roles, err := repository.List(context.Background(), listing.Query{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	got := []string{roles.Items[0].Name, roles.Items[1].Name, roles.Items[2].Name}
	want := []string{"backend", "Frontend", "QA"}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("roles[%d] = %q, want %q", index, got[index], want[index])
		}
	}
	if roles.Total != 3 || roles.Page != 1 || roles.PageSize != 20 {
		t.Fatalf("page metadata = %#v, want page 1, size 20, total 3", roles)
	}

	secondPage, err := repository.List(context.Background(), listing.Query{Page: 2, PageSize: 2})
	if err != nil {
		t.Fatalf("List() second page error = %v", err)
	}
	if secondPage.Total != 3 || len(secondPage.Items) != 1 || secondPage.Items[0].Name != "QA" {
		t.Fatalf("second page = %#v, want QA with total 3", secondPage)
	}

	searchResult, err := repository.List(context.Background(), listing.Query{
		Search: "BA",
		Page:   1, PageSize: 20,
	})
	if err != nil {
		t.Fatalf("List() search error = %v", err)
	}
	if searchResult.Total != 1 || len(searchResult.Items) != 1 || searchResult.Items[0].Name != "backend" {
		t.Fatalf("search result = %#v, want backend only", searchResult)
	}

	duplicate, err := domain.NewRole("duplicate", "BACKEND", now)
	if err != nil {
		t.Fatalf("NewRole() error = %v", err)
	}
	if duplicate == nil {
		t.Fatal("NewRole() result = nil, want role")
	}
	if err := repository.Create(context.Background(), *duplicate); !errors.Is(err, domain.ErrNameExists) {
		t.Fatalf("Create() duplicate error = %v, want ErrNameExists", err)
	}
}

func TestRepositoryPreventsDeletingAssignedRole(t *testing.T) {
	t.Parallel()

	database := openTestDatabase(t)
	repository := New(database)
	role, err := domain.NewRole("role-id", "Backend", time.Now().UTC())
	if err != nil {
		t.Fatalf("NewRole() error = %v", err)
	}
	if role == nil {
		t.Fatal("NewRole() result = nil, want role")
	}
	if err := repository.Create(context.Background(), *role); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := AssignRoleForTest(database, "member-id", role.ID); err != nil {
		t.Fatalf("AssignRoleForTest() error = %v", err)
	}

	if err := repository.Delete(context.Background(), role.ID); !errors.Is(err, domain.ErrInUse) {
		t.Fatalf("Delete() error = %v, want ErrInUse", err)
	}
	if _, err := repository.FindByID(context.Background(), role.ID); err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
}

func TestRepositoryTreatsSoftDeletedMemberAsHistoricalRoleReference(t *testing.T) {
	t.Parallel()

	database := openTestDatabase(t)
	repository := New(database)
	role, err := domain.NewRole("role-id", "Backend", time.Now().UTC())
	if err != nil || role == nil {
		t.Fatalf("NewRole() role = %#v, error = %v", role, err)
	}
	if err := repository.Create(context.Background(), *role); err != nil {
		t.Fatal(err)
	}
	if err := AssignRoleForTest(database, "member-id", role.ID); err != nil {
		t.Fatal(err)
	}
	if err := database.Delete(&teamMemberModel{}, "id = ?", "member-id").Error; err != nil {
		t.Fatal(err)
	}

	if err := repository.Delete(context.Background(), role.ID); !errors.Is(err, domain.ErrInUse) {
		t.Fatalf("Delete() error = %v, want ErrInUse", err)
	}
}

func TestRepositoryFindReturnsNilWhenRoleDoesNotExist(t *testing.T) {
	t.Parallel()

	repository := New(openTestDatabase(t))
	role, err := repository.FindByID(context.Background(), "missing")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("FindByID() error = %v, want ErrNotFound", err)
	}
	if role != nil {
		t.Fatalf("FindByID() result = %#v, want nil on error", role)
	}
}

func TestDatabaseProtectsConcurrentDuplicateCreation(t *testing.T) {
	database := openTestDatabase(t)
	repository := New(database)
	var sequence atomic.Int64
	service := application.NewServiceWithDependencies(
		repository,
		func() time.Time { return time.Now().UTC() },
		func() (string, error) {
			return fmt.Sprintf("role-%d", sequence.Add(1)), nil
		},
	)

	start := make(chan struct{})
	results := make(chan error, 2)
	var waitGroup sync.WaitGroup
	for range 2 {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			<-start
			_, err := service.Create(context.Background(), "Backend")
			results <- err
		}()
	}
	close(start)
	waitGroup.Wait()
	close(results)

	var succeeded, conflicted int
	for err := range results {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, domain.ErrNameExists):
			conflicted++
		default:
			t.Fatalf("Create() error = %v", err)
		}
	}
	if succeeded != 1 || conflicted != 1 {
		t.Fatalf("successes = %d, conflicts = %d; want 1 and 1", succeeded, conflicted)
	}
}
