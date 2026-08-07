package schedulingimpact

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGuardRejectsConfirmedTokenWhenHiddenRecalculationSignatureChanges_US62_D04_AC24(t *testing.T) {
	ctx := requestContext(t, "")
	ctx = WithOperation(ctx, "owner", ModeOrdinary)
	projects := []Project{
		{ID: "owner", Name: "Owner", Version: 3},
		{ID: "beta", Name: "Beta", Version: 7},
		{ID: "alpha", Name: "Alpha", Version: 5},
	}

	err := Guard(ctx, "schedule-signature", nil, projects)
	var impact Error
	if !errors.As(err, &impact) {
		t.Fatalf("expected impact error, got %v", err)
	}
	if impact.Kind != ConfirmationRequired || impact.Token == "" {
		t.Fatalf("impact = %#v", impact)
	}
	if len(impact.OpenProjects) != 2 || impact.OpenProjects[0].ID != "alpha" || impact.OpenProjects[1].ID != "beta" {
		t.Fatalf("open projects = %#v", impact.OpenProjects)
	}

	confirmed := WithOperation(requestContext(t, impact.Token), "owner", ModeOrdinary)
	if err := Guard(confirmed, "schedule-signature", nil, projects); err != nil {
		t.Fatalf("confirmed guard = %v", err)
	}

	stale := WithOperation(requestContext(t, impact.Token), "owner", ModeOrdinary)
	err = Guard(stale, "changed-signature", nil, projects)
	if !errors.As(err, &impact) || impact.Kind != StaleImpact {
		t.Fatalf("stale guard = %#v, %v", impact, err)
	}
}

func TestGuardBlocksOrdinaryLockedImpactButAllowsActualDateConfirmation(t *testing.T) {
	locked := []Project{{ID: "locked", Name: "Locked", Status: "locked", Version: 11}}

	ordinary := WithOperation(requestContext(t, ""), "owner", ModeOrdinary)
	err := Guard(ordinary, "signature", locked, nil)
	var impact Error
	if !errors.As(err, &impact) || impact.Kind != LockedProjectImpact {
		t.Fatalf("ordinary guard = %#v, %v", impact, err)
	}

	actual := WithOperation(requestContext(t, ""), "owner", ModeActualDate)
	err = Guard(actual, "signature", locked, nil)
	if !errors.As(err, &impact) || impact.Kind != ConfirmationRequired {
		t.Fatalf("actual-date preview = %#v, %v", impact, err)
	}
	confirmed := WithOperation(requestContext(t, impact.Token), "owner", ModeActualDate)
	if err := Guard(confirmed, "signature", locked, nil); err != nil {
		t.Fatalf("actual-date confirmation = %v", err)
	}
}

func TestGuardSkipsBulkReopenAndOwnerOnlyImpact(t *testing.T) {
	owner := []Project{{ID: "owner", Name: "Owner", Version: 1}}
	if err := Guard(WithOperation(context.Background(), "owner", ModeBulkReopen), "signature", owner, owner); err != nil {
		t.Fatalf("bulk reopen guard = %v", err)
	}
	if err := Guard(WithOperation(context.Background(), "owner", ModeOrdinary), "signature", nil, owner); err != nil {
		t.Fatalf("owner-only guard = %v", err)
	}
}

func TestGuardAllowsLatestNoTimelineImpactWithoutObsoleteConfirmation_US62_D04_AC24(t *testing.T) {
	ctx := WithOperation(requestContext(t, "obsolete-token"), "owner", ModeOrdinary)
	if err := Guard(ctx, "latest-hidden-state", nil, nil); err != nil {
		t.Fatalf("latest simulation without timeline impact must not require obsolete confirmation: %v", err)
	}
}

func TestWithPreviewOperationIgnoresSuppliedConfirmationToken_US62_D07_AC25(t *testing.T) {
	confirmed := WithOperation(requestContext(t, "client-token"), "owner", ModeBulkReopen)
	preview := WithPreviewOperation(confirmed, "owner", ModeOrdinary)

	enabled, owner, mode, token := Operation(preview)
	if !enabled || owner != "owner" || mode != ModeOrdinary || token != "" {
		t.Fatalf("preview operation = enabled=%v owner=%q mode=%q token=%q", enabled, owner, mode, token)
	}
}

func requestContext(t *testing.T, token string) context.Context {
	t.Helper()
	var captured context.Context
	handler := CaptureToken(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		captured = request.Context()
	}))
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	if token != "" {
		request.Header.Set(ConfirmationTokenHeader, token)
	}
	handler.ServeHTTP(httptest.NewRecorder(), request)
	if captured == nil {
		t.Fatal("request context was not captured")
	}
	return captured
}

func TestGuardScopesGroupOperationExcludesOnlyOwnerGroupAndKeepsSiblingGroup_US44_AC22_AC23(t *testing.T) {
	ctx := WithGroupOperation(requestContext(t, ""), "project", "owner-group", ModeOrdinary)
	groups := []Group{
		{ID: "owner-group", ProjectID: "project", Name: "Owner", Path: "Alpha / Owner", Version: 1},
		{ID: "sibling-group", ProjectID: "project", Name: "Sibling", Path: "Alpha / Sibling", Version: 2},
		{ID: "other-group", ProjectID: "other", Name: "Other", Path: "Beta / Other", Version: 3},
	}

	err := GuardScopes(ctx, "timeline-signature", nil, nil, nil, groups)
	var impact Error
	if !errors.As(err, &impact) || impact.Kind != ConfirmationRequired {
		t.Fatalf("impact=%#v err=%v, want confirmation", impact, err)
	}
	if len(impact.OpenGroups) != 2 || impact.OpenGroups[0].ID != "sibling-group" || impact.OpenGroups[1].ID != "other-group" {
		t.Fatalf("open groups=%#v, want same-Project sibling plus other Project Group", impact.OpenGroups)
	}
	for _, group := range impact.OpenGroups {
		if group.ID == "owner-group" {
			t.Fatalf("owner Group leaked into impact list: %#v", impact.OpenGroups)
		}
	}
}

func TestGuardScopesBlocksTimelineImpactToLockedGroup_US44_AC24(t *testing.T) {
	ctx := WithGroupOperation(requestContext(t, ""), "project", "owner-group", ModeOrdinary)
	locked := []Group{{ID: "locked-group", ProjectID: "project", Name: "Locked", Path: "Alpha / Locked", Status: "locked", Version: 11}}

	err := GuardScopes(ctx, "timeline-signature", nil, nil, locked, nil)
	var impact Error
	if !errors.As(err, &impact) || impact.Kind != LockedScopeImpact {
		t.Fatalf("impact=%#v err=%v, want SCHEDULING_LOCKED_SCOPE_IMPACT", impact, err)
	}
	if len(impact.LockedGroups) != 1 || impact.LockedGroups[0].Path != "Alpha / Locked" {
		t.Fatalf("locked groups=%#v", impact.LockedGroups)
	}
}

func TestGuardScopesKeepsDuplicateGroupNamesDistinctByQualifiedPath_US44_AC22(t *testing.T) {
	ctx := WithGroupOperation(requestContext(t, ""), "project", "owner-group", ModeOrdinary)
	groups := []Group{
		{ID: "api-a", ProjectID: "project", Name: "API", Path: "Alpha / Parent A / API", Version: 4},
		{ID: "api-b", ProjectID: "project", Name: "API", Path: "Alpha / Parent B / API", Version: 5},
	}

	err := GuardScopes(ctx, "timeline-signature", nil, nil, nil, groups)
	var impact Error
	if !errors.As(err, &impact) || impact.Kind != ConfirmationRequired {
		t.Fatalf("impact=%#v err=%v, want confirmation", impact, err)
	}
	if len(impact.OpenGroups) != 2 {
		t.Fatalf("open groups=%#v, want both duplicate-name Groups", impact.OpenGroups)
	}
	if impact.OpenGroups[0].Path == impact.OpenGroups[1].Path || impact.OpenGroups[0].ID == impact.OpenGroups[1].ID {
		t.Fatalf("duplicate-name Groups collapsed: %#v", impact.OpenGroups)
	}
	if impact.OpenGroups[0].Path != "Alpha / Parent A / API" || impact.OpenGroups[1].Path != "Alpha / Parent B / API" {
		t.Fatalf("qualified paths=%#v", impact.OpenGroups)
	}
}

func TestGuardScopesGroupPreviewRejectsChangedEffectiveState_US44_AC37(t *testing.T) {
	preview := WithGroupOperation(requestContext(t, ""), "project", "owner-group", ModeOrdinary)
	groups := []Group{{ID: "sibling", ProjectID: "project", Name: "Sibling", Path: "Alpha / Sibling", Version: 2}}

	err := GuardScopes(preview, "parent-anchor:2026-08-10", nil, nil, nil, groups)
	var impact Error
	if !errors.As(err, &impact) || impact.Kind != ConfirmationRequired || impact.Token == "" {
		t.Fatalf("preview impact=%#v err=%v", impact, err)
	}

	confirmed := WithGroupOperation(requestContext(t, impact.Token), "project", "owner-group", ModeOrdinary)
	err = GuardScopes(confirmed, "parent-anchor:2026-08-11", nil, nil, nil, groups)
	if !errors.As(err, &impact) || impact.Kind != StaleImpact {
		t.Fatalf("stale impact=%#v err=%v, want SCHEDULING_IMPACT_STALE", impact, err)
	}
}
