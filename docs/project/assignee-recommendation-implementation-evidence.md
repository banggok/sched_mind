# US-6.5 Assignee Recommendation Implementation Evidence

## Scope and validation status

This document records traceability for `user_story/US-6.5-recommend-assignee.md`
and the resulting updates to US-4.1, US-6.1, and US-6.3. It covers the batch
recommendation API, Automatic Scheduling `ON` rollback-only simulation,
Automatic Scheduling `OFF` advisory simulation, deterministic ranking,
percentage preservation, frontend freshness/fallback behavior, and the
side-effect boundary.

The configured validation mode for this implementation is
`USER_LOCAL_VALIDATION`. Automated commands were intentionally not executed.
The statuses used below are therefore:

- Code Inspection: `IMPLEMENTED BY CODE INSPECTION`
- Unit/Integration: `AUTHORED — NOT RUN — LOCAL VALIDATION REQUIRED`
- Acceptance-Level: `AUTHORED — NOT RUN — LOCAL VALIDATION REQUIRED`
- Overall: `IMPLEMENTED — LOCAL VALIDATION REQUIRED`

## Production implementation map

| Boundary | Production path and symbol | Implemented responsibility |
|---|---|---|
| Domain | `backend/internal/scheduling/domain/recommendation.go`: `AssigneeRecommendationInput.Validate`, `RankAssigneeRecommendations` | Required-input invariant, recommendation modes/groups, deterministic group/end/remainder/name/ID order. |
| Scheduling application | `backend/internal/scheduling/application/service.go`: `Service.RecommendAssignees` | One validated batch delegation and nil-result dependency contract. |
| WBS application | `backend/internal/wbs/application/service.go`: `AssigneeRecommender`, `Service.RecommendAssignees` | Confirmed Project/Task identity injection and application-clock ownership. |
| Scheduling persistence | `backend/internal/scheduling/infrastructure/gormrepo/recommendation.go`: `Repository.RecommendAssignees` | Serialized, locked, rollback-only shared snapshot; active Role candidates; lifecycle checks; edited-Task removal; ON/OFF simulation; baseline reuse; candidate isolation; metrics; stable failure mapping. |
| Scheduler metric | `backend/internal/scheduling/infrastructure/gormrepo/allocator.go`: `timelineResult.completionMetrics`, `remainingForCandidate` | Captures residual capacity immediately after candidate allocation and before lower-priority work. |
| WBS persistence | `backend/internal/wbs/infrastructure/gormrepo/repository.go`: `Repository.UpdateExecutable` | Preserves Task-owned percentage when Assignee is first selected, changed, or cleared unless an explicit valid percentage is supplied. |
| HTTP | `backend/internal/wbs/transport/http/handler.go`: `recommendAssignees`, `parseAssigneeRecommendationRequest`, `mapAssigneeRecommendation`, `writeError` | Strict batch request contract, stable response DTO, and safe 400/409/503 error codes. |
| Runtime wiring | `backend/cmd/api/main.go`: WBS `NewServiceWithDependencies` construction | Supplies backend `APP_TIMEZONE` clock and concrete scheduler/recommender. |
| Frontend gateway | `frontend/src/features/wbs/application/wbsGateway.ts`; `frontend/src/features/wbs/infrastructure/httpWBSGateway.ts`: `recommendAssignees`, `mapAssigneeRecommendation` | One batch request, strict untrusted-response validation, and confirmed-projection stale rejection. |
| Frontend workflow | `frontend/src/features/wbs/presentation/WBSDetailDialog.tsx`: `recommendationInput`, `loadRecommendations`, option assembly and Assignee interaction handlers | Prerequisite gate; Role/Assignee placement immediately after Lag; deterministic fallback; percentage preservation; mode labels; mismatch visibility; abort/sequence protection; open-order freeze; no-reservation selection. |

## Acceptance Criteria traceability

Every row below has authored evidence at all three required levels. No test in
this matrix was executed in this task.

| AC | Code inspection evidence | Unit/integration evidence | Acceptance-level evidence | Actual status |
|---|---|---|---|---|
| AC-1 | WBS dialog exposes recommendation for editable unfinished leaves; repository rejects Locked, Closed, started, completed, or non-leaf contexts, including an explicit Closed-Project lookup because Closed projects are excluded from the scheduling snapshot; ordinary Name-only create path is unchanged. | `TestRecommendAssigneesRejectsInvalidRoleAndNonRecommendableLifecycle_D03_AC3_AC1`; existing `TestCreateTaskAcceptanceDefaultsCapacityPercentageAndSkipsUnneededScheduler_US63_AC1_US6_AC29_US4_AC23`. | `TestAssigneeRecommendationHTTPAcceptanceRanksFromConcreteSchedulerWithoutSideEffectsAndPreservesPercentageOnSave_US65_AC1_AC4_AC6_AC12_AC13_AC14_AC15_AC25_AC29_AC30_AC33`; `TestAssigneeRecommendationClosedProjectHTTPAcceptanceRejectsLifecycle_US65_AC1`; existing Name-only create HTTP acceptance. | CI implemented; U/I authored not run; A authored not run. |
| AC-2 | `WBSDetailDialog` renders Effort, percentage, Lag, Role, then Assignee; timeline and dependency controls follow Assignee, and native controls preserve keyboard order. | `D02 AC2 renders Role and Assignee immediately after Lag and before timeline and dependency controls`. | Same rendered Task Details workflow verifies visible and DOM/tab sequence. | CI implemented; U/I authored not run; A authored not run. |
| AC-3 | `recommendationInput`, transport parser, domain validation, Role existence check, and confirmed Project/Task context validation reject incomplete/invalid drafts without simulation. | `TestAssigneeRecommendationInputValidate_D03_AC3`; `TestRecommendAssigneesRejectsInvalidInputBeforeStore_D03_AC3`; `TestAssigneeRecommendationEndpointRejectsIncompleteOrBrowserToday_D03_D17_AC3_AC34`; repository invalid-context test. | `D03 AC3 keeps alphabetical options and skips API while prerequisites are incomplete`. | CI implemented; U/I authored not run; A authored not run. |
| AC-4 | `prepareRecommendationState` removes the edited Task from all existing allocation timelines and clears generated projection before Automatic ON baseline/candidate runs. | `TestRecommendAssigneesAutomaticUsesSharedSchedulerAndRollsBack_D04_D05_D06_D07_D10_D11_AC4_AC6_AC11_AC13_AC18_AC19_AC29_AC30_AC33`. | First HTTP acceptance starts from a persisted old allocation and proves current Assignee is not double-counted. | CI implemented; U/I authored not run; A authored not run. |
| AC-5 | The same preparation removes the edited manual Task while `reclassifyRecommendationState` preserves every other manual/fixed/completed/Locked reservation. | `TestRecommendAssigneesManualUsesDraftStartKeepsOtherFixedAndIgnoresManualEnd_D04_D08_D09_D10_AC5_AC8_AC9_AC10_AC34`. | `TestAssigneeRecommendationManualHTTPAcceptanceUsesAdvisoryAnchorAndKeepsManualDates_US65_AC5_AC8_AC9_AC10_AC25_AC34`. | CI implemented; U/I authored not run; A authored not run. |
| AC-6 | Automatic mode invokes the existing `portfolioState.scheduleTimeline(Execution)` under the normal scheduling serialization/lock and deliberate rollback sentinel; no alternate allocator exists. | Automatic shared-scheduler repository test and rollback-failure repository test. | First HTTP acceptance proves concrete priority/capacity dates and persisted state remains unchanged. | CI implemented; U/I authored not run; A authored not run. |
| AC-7 | Automatic mode with missing Project scheduling anchor returns stable no-completion rows and never enters manual/today anchor logic. Frontend converts this result to alphabetical options with prerequisite text. | `TestRecommendAssigneesAutomaticWithoutAnchorDoesNotInventToday_D07_AC7`. | `TestAssigneeRecommendationAutomaticMissingAnchorHTTPAcceptanceReturnsPrerequisiteWithoutToday_US65_AC7_AC23_AC33`; `D07 AC7 keeps candidates alphabetical when Automatic Scheduling has no anchor`. | CI implemented; U/I authored not run; A authored not run. |
| AC-8 | Manual advisory preparation gives explicit draft Execution Start precedence and only changes the cloned Project/task state. | Manual draft-start repository test. | Manual HTTP acceptance verifies an explicit `2026-08-12` start produces an estimated finish on that Date while stored manual dates remain unchanged; manual component workflow sends `executionStart`. | CI implemented; U/I authored not run; A authored not run. |
| AC-9 | Manual advisory anchor is explicit start, else `max(Project Start, CalculatedOn)`, else `CalculatedOn`; `CalculatedOn` is injected by WBS application clock. | `TestRecommendAssigneesManualFallsBackToApplicationToday_D08_D09_AC9_AC34`; `TestRecommendAssigneesManualUsesLaterOfProjectStartAndApplicationToday_D08_D09_AC9_AC34`; WBS application clock test. | Manual HTTP acceptance covers future Project Start, no Project Start, and past Project Start against the same backend application Date. | CI implemented; U/I authored not run; A authored not run. |
| AC-10 | The HTTP DTO accepts and date-validates optional draft Execution End, but deliberately omits it from the domain simulation input; confirmed Execution End is cleared only in the cloned edited Task and never affects ranking or warning UI. | Manual repository test with a wide confirmed manual range plus handler contract coverage for optional `executionEnd`. | Manual HTTP acceptance proves advisory finish differs from stored manual End without mutation; manual component workflow renders only advisory metadata. | CI implemented; U/I authored not run; A authored not run. |
| AC-11 | One transaction loads one `portfolioState`, captures one Project-version map, computes one baseline, and clones that in-memory state for every candidate. | Automatic shared-snapshot repository test; transport one-batch/exact-draft test. | First HTTP acceptance returns all candidates with one snapshot version identity. | CI implemented; U/I authored not run; A authored not run. |
| AC-12 | Frontend issues one `recommendAssignees` call; backend candidate list is loaded once and looped in memory. | Scheduling application one-batch test; Role-filter repository query test; gateway one-batch mapping test; handler one-batch test. | First HTTP acceptance receives all candidates from one request. | CI implemented; U/I authored not run; A authored not run. |
| AC-13 | Assignee selection handler does not change percentage; simulation input uses current visible percentage; WBS persistence omission preserves stored percentage. | WBS percentage-preservation integration test; automatic repository test simulates at `40%`; component batch test. | First HTTP acceptance simulates and then first-selects at `40%`, re-reading `40%`. | CI implemented; U/I authored not run; A authored not run. |
| AC-14 | Reassignment no longer resets draft or persisted percentage; explicit percentage remains the only replacement path. | `TestUpdateExecutablePreservesPercentageOnFirstSelectionChangeAndClear_US65_AC13_AC14_AC15`; component preservation test. | First HTTP acceptance reassigns with omitted percentage and re-reads `40%`. | CI implemented; U/I authored not run; A authored not run. |
| AC-15 | Clear handler and repository omission retain percentage; existing missing-Assignee scheduler behavior remains unchanged. | Same WBS integration test and component preservation test. | First HTTP acceptance clears Assignee and re-reads nil Assignee with `40%`. | CI implemented; U/I authored not run; A authored not run. |
| AC-16 | Domain rank group order is feasible, overcapacity, no-completion; incremental metric determines first two groups before finish ordering. | `TestRankAssigneeRecommendations_D10_D11_D12_D13_AC16_AC17_AC18_AC20_AC21_AC22_AC23`; ranking repository test. | `TestAssigneeRecommendationRankingHTTPAcceptanceGroupsIncrementalOvercapacityAndCandidateFailure_US65_AC16_AC20_AC21_AC23_AC32`. | CI implemented; U/I authored not run; A authored not run. |
| AC-17 | `RankAssigneeRecommendations` compares Execution End before remainder inside completion groups. | Domain ranking test. | First HTTP acceptance ranks earlier feasible B before later feasible A despite lower work; returned dates are asserted. | CI implemented; U/I authored not run; A authored not run. |
| AC-18 | Domain ranker compares larger `RemainingExecutionCapacityMinutes` after equal end; scheduler records that metric per Task. | Domain ranking test; automatic repository shared-scheduler test. | `TestAssigneeRecommendationTieBreakHTTPAcceptanceUsesRemainingCapacityThenNormalizedNameAndID_US65_AC18_AC22`. | CI implemented; U/I authored not run; A authored not run. |
| AC-19 | `scheduleTimeline` captures `remainingForCandidate` immediately after candidate commit and before scheduling the next lower-priority Task. | Automatic repository test includes lower-priority same-member work and asserts preserved residual metric. | First HTTP acceptance includes lower-priority B work yet returns B's immediate `6h` remainder. | CI implemented; U/I authored not run; A authored not run. |
| AC-20 | `incrementalOvercapacityMinutes` subtracts per-Date baseline overcapacity and sums only positive increases. | Direct incremental-overcapacity test plus ranking repository test with pre-existing baseline overcapacity. | Ranking HTTP acceptance classifies the candidate that does not worsen baseline overcapacity as feasible. | CI implemented; U/I authored not run; A authored not run. |
| AC-21 | Per-Date positive increases are accumulated independently; negative changes on another Date are never netted against them. | `TestIncrementalOvercapacityCountsOnlyPositivePerDateIncrease_D12_AC20_AC21`. | Ranking HTTP acceptance returns the worsening candidate as overcapacity with exact `4h` increment. | CI implemented; U/I authored not run; A authored not run. |
| AC-22 | Final comparisons normalize trimmed lower-case name and then stable Member ID; frontend duplicate labels disclose role/short ID without changing backend order. | Domain ranking test. | Tie-break HTTP acceptance verifies duplicate case-insensitive names resolve by ID after all metrics tie. | CI implemented; U/I authored not run; A authored not run. |
| AC-23 | Candidate-specific zero capacity/blocked/unavailable outcomes become selectable `no-completion` items ordered last; automatic missing-anchor frontend deliberately drops fake ranking metadata but retains alphabetic selection. | Domain ranking and repository candidate-isolation tests. | Ranking HTTP acceptance returns a no-capacity candidate last; missing-anchor HTTP/component workflows keep candidates selectable/alphabetical. | CI implemented; U/I authored not run; A authored not run. |
| AC-24 | Role change no longer clears current Assignee; option assembly appends a mismatched current Assignee last with text warning; normal backend Role/Assignee validation remains authoritative on Save. | `D09 D14 AC24 preserves a mismatched current Assignee visibly and last`. | Same rendered Task workflow exercises the user-visible mismatch exception and replacement/clear path. | CI implemented; U/I authored not run; A authored not run. |
| AC-25 | API returns `mode`; option formatter uses `finishes` for automatic and `estimated` for manual, with remaining/overcapacity/no-completion text and no manual-End warning. | Gateway mapping test; automatic and manual component metadata tests; handler DTO test. | First automatic HTTP acceptance plus manual HTTP/component acceptance prove mode-specific metadata. | CI implemented; U/I authored not run; A authored not run. |
| AC-26 | Draft or confirmed changes abort/invalidate accepted ranking; opening without a current result requests one latest batch; stale options are disabled while pending. | Gateway projection-version test; `D16 AC26 AC28 disables stale rows and ignores an older response`. | Same rendered workflow proves stale rows cannot be selected before latest response. | CI implemented; U/I authored not run; A authored not run. |
| AC-27 | While Assignee interaction is open, a newer accepted response is held; confirmed-state invalidation marks existing options stale and applies changes only after close/reopen. Manual mode refreshes on each open, preventing midnight reorder during the open interaction. | `D16 AC27 freezes visible order across a confirmed mutation until close and reopen`. | Same rendered native-select workflow observes stable option order until close. | CI implemented; U/I authored not run; A authored not run. |
| AC-28 | AbortController, request sequence, and draft version checks discard late older responses without mutating selection or newest result. | Gateway stale-response test; `D16 AC26 AC28 disables stale rows and ignores an older response`. | Rendered overlapping-request workflow verifies newest ranking and selected Assignee survive older completion. | CI implemented; U/I authored not run; A authored not run. |
| AC-29 | Recommendation path never calls Task mutation/cache invalidation; selection only changes local draft; rollback prevents capacity reservation. | Automatic repository rollback test and component batch/selection test. | First HTTP acceptance re-reads unchanged Task/allocation/version immediately after recommendation, before explicit Save. | CI implemented; U/I authored not run; A authored not run. |
| AC-30 | Save remains the existing `updateExecutable` command and concrete latest-state scheduling transaction; no recommendation token/age is accepted by Save. | Automatic repository test and WBS percentage-preservation integration test. | First HTTP acceptance performs Save only after recommendation and verifies confirmed result through the ordinary executable endpoint. | CI implemented; U/I authored not run; A authored not run. |
| AC-31 | Shared failure maps to stable unavailable error; gateway rejects malformed batches; dialog preserves draft/selection, switches to alphabetic options, exposes Retry, and leaves Save available. | Scheduling nil/unavailable tests; rollback-failure repository test; handler stable-error test; gateway malformed-response test; component fallback test. | `D16 D17 AC31 falls back alphabetically, preserves selection, and retries latest batch` is the highest practical user workflow. | CI implemented; U/I authored not run; A authored not run. |
| AC-32 | Per-candidate schedule/metric error is caught inside the loop and converted to a stable no-completion reason without aborting other items. | Ranking repository candidate-isolation test. | Ranking HTTP acceptance returns successful candidates plus one candidate-specific no-completion item. | CI implemented; U/I authored not run; A authored not run. |
| AC-33 | Deliberate transaction rollback encloses success and failure; cloned state protects in-memory baseline; frontend read does not advance confirmed projection cache. | Automatic rollback test; shared scheduler failure readback test; handler exact-batch test. | First and missing-anchor HTTP acceptances re-read unchanged Task, dates, allocation, Project anchor/version; frontend gateway test verifies no confirmed invalidation. | CI implemented; U/I authored not run; A authored not run. |
| AC-34 | Runtime WBS service receives `time.Now().In(config.location)`; browser today is absent from contract; manual anchor uses backend `CalculatedOn`. | WBS application clock test; handler rejects unknown/browser-today input; manual fallback repository tests. | Manual HTTP acceptance uses a fixed `APP_TIMEZONE` clock and covers explicit, future, absent, and past anchors with one calculated Date. | CI implemented; U/I authored not run; A authored not run. |
| AC-35 | This file maps every AC to exact production, unit/integration, acceptance, commands, and non-fabricated status. | All tests named above are authored in the repository. | Backend HTTP and frontend rendered workflows are explicitly identified above. | CI implemented; U/I authored not run; A authored not run; overall local validation required. |

## Query and transaction review

### Candidate query

`loadRecommendationCandidates` executes one bounded Role cohort read:

```sql
SELECT ...
FROM team_members
WHERE role_id = $1
  AND deleted_at IS NULL
ORDER BY LOWER(name), id;
```

Review findings:

- Equality predicate: `role_id`.
- Soft-delete predicate: `deleted_at IS NULL`.
- Deterministic order: normalized name then stable ID.
- Expected cardinality: active Members in one selected Role, not the whole
  portfolio and not one query per candidate.
- Existing support: `team_members_role_id_idx` supports Role equality and the
  result cohort is bounded before the normalized-name sort. The primary key
  supports stable identity but not the sort itself.
- No new index was added. A composite partial expression index would add write
  and storage cost for a form interaction whose result is already bounded by
  Role. That decision must be revisited only if representative production
  cardinality or a PostgreSQL plan demonstrates a material sort/scan cost.
- Query-plan verification was not run under `USER_LOCAL_VALIDATION`.

The Role existence check is one primary-key equality count. The scheduler state
load remains the existing bounded active-portfolio/horizon read owned by
US-6.1. Candidate simulations perform no additional database reads.

### Transaction, concurrency, and rollback

- Recommendation uses the same process serialization and database advisory lock
  as schedule mutation, preventing a mixed confirmed snapshot.
- One database transaction loads state, Role candidates, and Project versions.
- One baseline is calculated after the edited Task is removed.
- Every candidate starts from a deep clone of the same prepared snapshot.
- Success exits with a deliberate sentinel error so the transaction rolls back.
- Shared scheduler/query failure returns a stable whole-batch error after
  rollback; candidate-only simulation failure remains an item-level result.
- No Task, dependency, allocation, Project version, cache version, reservation,
  or mutation event is persisted by recommendation.

## Local validation commands

Run from the repository root unless a command includes `cd`.

### Backend targeted behavior

```sh
cd backend
go test ./internal/scheduling/domain -run 'TestAssigneeRecommendation'
go test ./internal/scheduling/application -run 'TestRecommendAssignees'
go test ./internal/scheduling/infrastructure/gormrepo -run 'TestRecommendAssignees|TestIncrementalOvercapacity|TestAssigneeRecommendation'
go test ./internal/wbs/application -run 'TestRecommendAssignees'
go test ./internal/wbs/infrastructure/gormrepo -run 'TestUpdateExecutablePreservesPercentage'
go test ./internal/wbs/transport/http -run 'TestAssigneeRecommendation'
```

Expected success criterion: every listed command exits `0` with no failing test.
These commands cover D-01 through D-18, including the actual HTTP acceptance
workflows in the scheduling repository package.

### Frontend targeted behavior

```sh
cd frontend
npm test -- src/features/wbs/infrastructure/httpWBSGateway.test.ts src/features/wbs/presentation/WBSDetailDialog.test.tsx
```

Expected success criterion: the gateway batch/stale-contract tests and rendered
Task Details recommendation workflows pass without unhandled React warnings.

### Repository-required full validation

```sh
cd backend
go fmt ./...
go vet ./...
go test ./...
go test -race ./...

cd ../frontend
npm run format:check
npm run lint
npm run typecheck
npm test
npm run build
```

Expected success criterion: every command exits `0`. Because backend code or
configuration changed, local runtime validation must then restart the backend
and smoke-test:

```text
POST /api/projects/{projectId}/wbs/{wbsId}/assignee-recommendations
```

with both an automatic Project and a manual Project. Confirm that response mode,
ordered items, calculation Date, and snapshot versions are present and that a
subsequent Task read shows no mutation before Save.

### Optional PostgreSQL query-plan check

Against representative active-member cardinality, substitute a real Role UUID:

```sql
EXPLAIN (ANALYZE, BUFFERS)
SELECT id, name, role_id, daily_capacity, buffer_percentage, deleted_at
FROM team_members
WHERE role_id = '<role-uuid>'
  AND deleted_at IS NULL
ORDER BY LOWER(name), id;
```

Expected review criterion: Role filtering uses an appropriate indexed path or a
planner-justified sequential scan for a small table; sort and returned row count
remain bounded to one Role cohort.

## Static review result

- Obsolete rule: backend and frontend Assignee-triggered `100%` resets were
  removed; default `100%` remains only Task creation/default behavior.
- Architecture: domain owns validation/ranking, application owns use-case clock
  and identity, persistence owns snapshot/scheduler/query/rollback, transport
  owns DTO/errors, and frontend owns interaction/fallback presentation.
- Transaction/rollback: one locked rollback-only transaction; no candidate
  writes or per-candidate reads.
- Concurrency/stale state: process/database scheduling locks prevent mixed
  backend snapshots; projection version, abort, sequence, and draft version
  prevent stale frontend acceptance.
- Query/index: bounded Role lookup uses existing Role index; no unproven index or
  migration was added.
- Cache: recommendation read does not invalidate confirmed cache; confirmed
  mutations invalidate/disable stale recommendation results.
- Identity/field/dependency preservation: Task identity/order, unrelated fields,
  explicit dependency endpoints, manual/completed/Locked allocations, and
  Project versions remain unchanged by simulation.
- Accessibility: visible text carries ranking meaning, native labels/select
  preserve keyboard behavior, status text is live, and mismatch/failure do not
  rely on color alone.
- Test quality: tests assert returned/persisted state, rollback readback,
  deterministic order, exact endpoints/DTOs, and stale response ordering rather
  than mock calls alone.
- Documentation: US-4.1, US-6.1, US-6.3, project context, architecture, and
  previous capacity evidence now consistently identify US-6.5 ownership.

## Final status

`IMPLEMENTED — LOCAL VALIDATION REQUIRED`
