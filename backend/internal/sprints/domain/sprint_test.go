package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNewSprintValidatesAndNormalizesDetailsAndRelations_AC7To10(t *testing.T) {
	now := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
	sprint, err := NewSprint("sprint-1", "  August Sprint  ", now, now, []string{" member-1 ", "member-1"}, []string{"task-1", "task-1"}, now)
	if err != nil {
		t.Fatalf("create one-day sprint: %v", err)
	}
	if sprint.Name != "August Sprint" || sprint.NormalizedName != "august sprint" {
		t.Fatalf("unexpected normalized name: %#v", sprint)
	}
	if len(sprint.MemberIDs) != 1 || len(sprint.TaskIDs) != 1 || sprint.Status != StatusPlanned || sprint.Version != 1 {
		t.Fatalf("unexpected initial sprint: %#v", sprint)
	}

	tests := []struct {
		name       string
		inputName  string
		start, end time.Time
		members    []string
		want       error
	}{
		{name: "blank name", inputName: "  ", start: now, end: now, members: []string{"m"}, want: ErrNameRequired},
		{name: "long name", inputName: strings.Repeat("x", 201), start: now, end: now, members: []string{"m"}, want: ErrNameTooLong},
		{name: "backwards dates", inputName: "Sprint", start: now, end: now.AddDate(0, 0, -1), members: []string{"m"}, want: ErrDateRangeInvalid},
		{name: "no members", inputName: "Sprint", start: now, end: now, want: ErrMemberRequired},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewSprint("id", test.inputName, test.start, test.end, test.members, nil, now)
			if !errors.Is(err, test.want) {
				t.Fatalf("got %v, want %v", err, test.want)
			}
		})
	}
}

func TestOverlapUsesInclusiveDatesAndSharedMembers_AC59To63(t *testing.T) {
	day := func(value int) time.Time { return time.Date(2026, 8, value, 0, 0, 0, 0, time.UTC) }
	other := Sprint{StartDate: day(10), EndDate: day(20), MemberIDs: []string{"member-1"}}
	if !Overlaps(day(1), day(10), []string{"member-1"}, other) {
		t.Fatal("boundary contact for a shared member must overlap")
	}
	if Overlaps(day(10), day(20), []string{"member-2"}, other) {
		t.Fatal("same dates for disjoint members must be allowed")
	}
	if Overlaps(day(21), day(22), []string{"member-1"}, other) {
		t.Fatal("disjoint dates must not overlap")
	}
}

func TestStartIsOneWayMetadataTransition_AC66To69(t *testing.T) {
	now := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	sprint, err := NewSprint("id", "Sprint", now, now, []string{"member"}, []string{"task"}, now)
	if err != nil {
		t.Fatal(err)
	}
	startedAt := now.Add(time.Hour)
	if err := sprint.Start(startedAt); err != nil {
		t.Fatal(err)
	}
	if sprint.Status != StatusStarted || sprint.StartedAt == nil || !sprint.StartedAt.Equal(startedAt) || sprint.Version != 2 {
		t.Fatalf("unexpected started sprint: %#v", sprint)
	}
	if err := sprint.Start(startedAt.Add(time.Hour)); !errors.Is(err, ErrAlreadyStarted) {
		t.Fatalf("got %v, want already started", err)
	}
	if sprint.Version != 2 || !sprint.StartedAt.Equal(startedAt) {
		t.Fatal("repeated start mutated metadata")
	}
}

func TestSuggestionIncludesAllMandatoryThenWholeFillTasksDeterministically_AC24To32(t *testing.T) {
	end := time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)
	candidates := []Candidate{
		{TaskID: "fill-later", MemberID: "member", ExecutionEnd: end.AddDate(0, 0, 2), ProjectPriority: 3, WBSOrder: 1, InSprintAllocationMinutes: 180},
		{TaskID: "mandatory-zero", MemberID: "member", ExecutionEnd: end.AddDate(0, 0, -10), ProjectPriority: 1, WBSOrder: 2},
		{TaskID: "fill-first", MemberID: "member", ExecutionEnd: end.AddDate(0, 0, 1), ProjectPriority: 2, WBSOrder: 2, InSprintAllocationMinutes: 180},
		{TaskID: "mandatory-end", MemberID: "member", ExecutionEnd: end, ProjectPriority: 1, WBSOrder: 1, InSprintAllocationMinutes: 60},
		{TaskID: "later-zero", MemberID: "member", ExecutionEnd: end.AddDate(0, 0, 1), ProjectPriority: 9, WBSOrder: 1},
	}
	want := []SuggestedTask{
		{TaskID: "mandatory-zero", Reason: SuggestionMandatory},
		{TaskID: "mandatory-end", Reason: SuggestionMandatory},
		{TaskID: "fill-first", Reason: SuggestionCapacityFill},
		{TaskID: "fill-later", Reason: SuggestionCapacityFill},
	}
	got := Suggest([]string{"member"}, end, map[string]int64{"member": 300}, candidates)
	if len(got) != len(want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("at %d got %#v, want %#v", index, got[index], want[index])
		}
	}
}

func TestSuggestionUsesLowerProjectPriorityValueFirst_AC28And32(t *testing.T) {
	end := time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)
	candidates := []Candidate{
		{TaskID: "lower-priority", MemberID: "member", ExecutionEnd: end.AddDate(0, 0, 1), ProjectPriority: 8, WBSOrder: 1, InSprintAllocationMinutes: 120},
		{TaskID: "higher-priority", MemberID: "member", ExecutionEnd: end.AddDate(0, 0, 1), ProjectPriority: 1, WBSOrder: 9, InSprintAllocationMinutes: 120},
	}
	got := Suggest([]string{"member"}, end, map[string]int64{"member": 60}, candidates)
	if len(got) != 1 || got[0].TaskID != "higher-priority" {
		t.Fatalf("got %#v, want higher-priority Project Task first", got)
	}
}
