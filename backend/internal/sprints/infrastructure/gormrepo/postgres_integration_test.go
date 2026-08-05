package gormrepo

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/banggok/sched_mind/backend/internal/sprints/application"
	"github.com/banggok/sched_mind/backend/internal/sprints/domain"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestPostgresConcurrentOverlappingCreatesHaveExactlyOneWinner_AC64(t *testing.T) {
	databaseURL := os.Getenv("SPRINT_POSTGRES_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("SPRINT_POSTGRES_TEST_DATABASE_URL is not configured")
	}
	admin, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open PostgreSQL test database: %v", err)
	}
	schema := fmt.Sprintf("sprint_test_%d", time.Now().UnixNano())
	if err := admin.Exec(`CREATE SCHEMA "` + schema + `"`).Error; err != nil {
		t.Fatalf("create isolated schema: %v", err)
	}
	t.Cleanup(func() {
		if cleanupErr := admin.Exec(`DROP SCHEMA "` + schema + `" CASCADE`).Error; cleanupErr != nil {
			t.Errorf("drop isolated schema: %v", cleanupErr)
		}
	})
	scopedURL, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatalf("parse PostgreSQL test URL: %v", err)
	}
	query := scopedURL.Query()
	query.Set("search_path", schema)
	scopedURL.RawQuery = query.Encode()
	database, err := gorm.Open(postgres.Open(scopedURL.String()), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open isolated PostgreSQL schema: %v", err)
	}
	if err := database.Exec(`
		CREATE TABLE team_members (id UUID PRIMARY KEY, name TEXT NOT NULL, role_id UUID NOT NULL,
			daily_capacity NUMERIC NOT NULL, buffer_percentage NUMERIC NOT NULL, updated_at TIMESTAMPTZ NOT NULL, deleted_at TIMESTAMPTZ);
		CREATE TABLE projects (id UUID PRIMARY KEY, name TEXT NOT NULL, status TEXT NOT NULL, priority INTEGER NOT NULL, schedule_version BIGINT NOT NULL);
		CREATE TABLE wbs_nodes (id UUID PRIMARY KEY, project_id UUID NOT NULL, parent_key TEXT NOT NULL DEFAULT '', parent_id UUID,
			position INTEGER NOT NULL, name TEXT NOT NULL, assignee_id UUID, execution_start DATE, execution_end DATE, actual_start DATE, actual_end DATE, updated_at TIMESTAMPTZ NOT NULL);
		CREATE TABLE sprints (id UUID PRIMARY KEY, name TEXT NOT NULL, name_key TEXT NOT NULL UNIQUE, start_date DATE NOT NULL, end_date DATE NOT NULL,
			status TEXT NOT NULL, version BIGINT NOT NULL, started_at TIMESTAMPTZ, created_at TIMESTAMPTZ NOT NULL, updated_at TIMESTAMPTZ NOT NULL);
		CREATE TABLE sprint_members (sprint_id UUID NOT NULL, member_id UUID NOT NULL, PRIMARY KEY (sprint_id, member_id));
		CREATE TABLE sprint_tasks (sprint_id UUID NOT NULL, task_id UUID NOT NULL, PRIMARY KEY (sprint_id, task_id));
	`).Error; err != nil {
		t.Fatalf("create isolated Sprint schema: %v", err)
	}
	memberID := "00000000-0000-0000-0000-000000000001"
	projectID := "00000000-0000-0000-0000-000000000002"
	taskID := "00000000-0000-0000-0000-000000000003"
	day := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	if err := database.Exec(`INSERT INTO team_members (id, name, role_id, daily_capacity, buffer_percentage, updated_at)
		VALUES (?, 'Harry', '00000000-0000-0000-0000-000000000004', 8, 20, ?)`, memberID, day).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Exec(`INSERT INTO projects (id, name, status, priority, schedule_version) VALUES (?, 'Alpha', 'open', 1, 1)`, projectID).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Exec(`INSERT INTO wbs_nodes (id, project_id, position, name, assignee_id, execution_start, execution_end, updated_at)
		VALUES (?, ?, 1, 'Task', ?, ?, ?, ?)`, taskID, projectID, memberID, day, day, day).Error; err != nil {
		t.Fatal(err)
	}
	first, err := domain.NewSprint("00000000-0000-0000-0000-000000000010", "First", day, day.AddDate(0, 0, 7), []string{memberID}, []string{taskID}, day)
	if err != nil {
		t.Fatal(err)
	}
	second, err := domain.NewSprint("00000000-0000-0000-0000-000000000011", "Second", day, day.AddDate(0, 0, 7), []string{memberID}, []string{taskID}, day)
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	for _, candidate := range []*domain.Sprint{first, second} {
		go func(sprint domain.Sprint) {
			ready.Done()
			<-start
			results <- New(database).Create(context.Background(), sprint)
		}(*candidate)
	}
	ready.Wait()
	close(start)
	successes, conflicts := 0, 0
	for range 2 {
		switch result := <-results; {
		case result == nil:
			successes++
		case errors.Is(result, application.ErrMemberOverlap):
			conflicts++
		default:
			t.Fatalf("unexpected concurrent result: %v", result)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes=%d conflicts=%d", successes, conflicts)
	}
}
