# Sprint Daily Working-Plan Implementation Evidence

## Scope and validation status

This document maps the US-8.1 daily working-plan delta to production code and
authored evidence. The configured validation mode is `USER_LOCAL_VALIDATION`.
No test, build, lint, typecheck, formatter, service, database, or browser command
was executed while preparing this change.

- Code Inspection: `IMPLEMENTED BY CODE INSPECTION`
- Unit/Integration: `AUTHORED — NOT RUN — LOCAL VALIDATION REQUIRED`
- Acceptance-Level: `AUTHORED — NOT RUN — LOCAL VALIDATION REQUIRED`
- Overall: `IMPLEMENTED — LOCAL VALIDATION REQUIRED`

## Requirement delta traceability

| Delta | Affected ACs | Production implementation | Unit/integration evidence | Acceptance-level evidence |
|---|---|---|---|---|
| SPD-01 Flat canonical daily-plan order | AC34, AC54, AC78 | `backend/internal/sprints/domain/daily_plan.go` (`CompareDailyPlanTask`); `backend/internal/sprints/infrastructure/gormrepo/daily_projection.go` (`sortDailyPlanTasks`); `projection.go` and `suggestion.go`; `frontend/src/features/sprints/presentation/SprintTaskReview.tsx` (`sortDailyPlanTasks`, flat Member filtering) | `backend/internal/sprints/domain/daily_plan_test.go` — `TestCompareDailyPlanTaskOrdersCanonicalDailyWorkingPlan_SPD01_AC34And78`; `backend/internal/sprints/infrastructure/gormrepo/repository_test.go` — `TestDetailOrdersFlatDailyPlanAcrossProjectsAndWBS_SPD01And02And17_AC34And36And47And74And78` | `frontend/src/features/sprints/presentation/SprintTaskReview.test.tsx` — `renders a flat daily plan across Projects...`, `orders unfinished readable Tasks before completed and unreadable Tasks...`, and `keeps rendering bounded...` |
| SPD-02 Complete Task execution context | AC34, AC36, AC76 | `application/readmodel.go` (`ProjectPriority`, `WBSPath`, `WBSRank`, `DailyPlanOrderDate`); `gormrepo/wbs_context.go`; HTTP DTO/mappers in `transport/http/handler.go`; frontend domain/gateway/task rows. WBS/order metadata remains available for ordering but is hidden from Sprint Planning. | Repository ordering test above; `frontend/src/features/sprints/infrastructure/httpSprintsGateway.test.ts` — `normalizes nullable Task collections before Sprint Planning rendering` | HTTP contract tests preserve Project/Parent/WBS/order inputs; component tests verify visible immediate Parent Name, owning Project status, live planning metadata, and allocation, omit Assignee inside normal Member groups, retain it under Needs Review, and keep WBS plus aggregate allocation metadata absent. |
| SPD-03 Per-Member, per-Date variance without cross-Date netting | AC17–AC21, AC35, AC79 | `application.DailySummary`; `gormrepo/finalizeMemberProjection`; local `deriveReviewProjection`/`dailySummary` | `TestDetailComposesLiveCapacityOutsideAllocationAndNeedsReview_SPD17_AC17To21And34To55`; `TestDetailTreatsWeekendAllocationAsFullOvercapacity_SPD03_AC19`; frontend `deriveReviewProjection` test | HTTP tests preserve no-netting calculations. Component tests preserve the per-Date capacity read model while rendering usage-versus-capacity Member summaries and verifying the zero-capacity state without Remaining/Overcapacity summary rows. |
| SPD-04 Sprint daily/period totals preserve cross-Member variance | AC19, AC79 | `gormrepo/composeSprintDailyTotals`; detail/suggestion total accumulation; frontend `aggregateDailySummaries` | `TestDetailPreservesCrossMemberRemainingAndOvercapacity_SPD03And04_AC19And79`; frontend `deriveReviewProjection` test | HTTP no-netting tests preserve the read-model contract; Sprint Planning intentionally does not render Sprint daily/period totals. |
| SPD-05 Needs Review daily allocation is separate | AC49–AC53, AC55, AC57, AC79 | `projection.go` (`needsReviewByDate`); `Totals.NeedsReviewDailyAllocation`; HTTP/frontend mapping; `NeedsReviewDailyPlan` | `TestDetailComposesLiveCapacityOutsideAllocationAndNeedsReview_SPD17_AC17To21And34To55` verifies reassignment separation; candidate/readability regressions | `TestHTTPMemberDeleteRetainsSprintTaskAsNeedsReview_AC57`; component tests verify retained unreadable Tasks in a separate group without an aggregate Needs Review allocation row. |
| SPD-06 Accessible daily grid | AC36–AC39, AC76 | `SprintTaskReview.tsx`: `reviewDates`, `DailyPlanTable`, `DateHeader`, focusable horizontal scroll regions, and accessible Task/Capacity cells | Component tests render the complete Sprint Date range in one grid and verify stable row order without Date navigation controls. | The same workflow verifies `DD MMM YYYY` headers, Sprint boundary labels, and no Previous/Next Date controls. |
| SPD-09 Simplified Member daily-plan presentation | AC35–AC39, AC76 | `SprintTaskReview.tsx`: removes `SummaryGrid`/`DateWindowControls`, renders one Member summary plus per-Date Capacity, hides WBS/aggregate allocation metadata, wraps Task names at `50ch`, and derives a whole-column `No capacity` marker directly from `member.dailyCapacity` | `SprintTaskReview.test.tsx` — `renders Member daily plans directly...`, `renders only Sprint Dates and ignores outside allocation columns...`, and `formats dates, wraps long Task names...` | Component workflows verify the complete observable UI simplification and that header, Capacity, and Task cells are marked only when Member Daily Capacity is `0h`, not when Remaining Capacity is `0h`. |
| SPD-10 Sprint-period Date columns only | AC37, AC39, AC53, AC76 | `SprintTaskReview.tsx`: `reviewDates` returns only the inclusive Sprint Start-to-End range; Task/Capacity cells no longer expose outside-Sprint column semantics | `SprintTaskReview.test.tsx` — `renders only Sprint Dates and ignores outside allocation columns_SPD10`; ordering regressions retain overdue/after-Sprint rows while asserting outside allocation cells are absent | The page-level Sprint Planning shows all Sprint Dates, keeps deterministic Task membership/order, and creates no Date column for positive allocation before Sprint Start or after Sprint End. |
| SPD-11 Shared Task edit inside Sprint Planning | AC80 | `App.tsx` supplies the existing Project/WBS/Dependency/Role/Member gateways to `SprintsPage`; `SprintTaskReview.tsx` renders the same `WBSPanel` direct-entry controller used by Home, keeps the Sprints hash unchanged, performs no Sprint action on dialog Close, and invokes the existing suggestion read after `WBSPanel.onMutated` | Existing `WBSPanel` and `WBSDetailDialog` suites remain the component-level evidence for Task loading/edit persistence; `SprintTaskReview.test.tsx` statically composes those same production components and mocks only their gateways | `opens the shared Home Task editor and closes back to the same Sprint Planning without action_SPD11_AC80` and `regenerates Sprint Planning usage and live Parent Name after saving through the shared Home Task editor_SPD11_SPD12_SPD17_AC47And80` verify visible Task-name activation, actual Edit Task dialog workflow, no hash navigation, no suggestion on Close, suggestion after Save, refreshed local usage, preserved Task identity, and no Sprint persistence before `Save Sprint Planning` |
| SPD-12 Local usage-versus-capacity summary | AC19, AC35, AC38, AC42–AC44, AC80 | `deriveReviewProjection` recalculates each selected Member's uncapped in-range allocation from the current local Task selection; `MemberDailyPlan` renders `Capacity: {Usage} of {Total}` with screen-reader clarification for Usage and Execution Capacity. Needs Review and outside-Sprint allocation are excluded. | `SprintTaskReview.test.tsx` — `derives uncapped local Sprint usage from selected in-range allocations only_SPD12_AC19And35` and `formats decimal Sprint usage and Execution Capacity in the Member summary_SPD12_AC35And38`; Add/Remove/regeneration regressions assert `0h`, overcapacity, decimal formatting, stale-response protection, and immediate local refresh. | `renders Member daily plans directly...`, `shows Add Task results inline...`, `removes a Task...`, and the shared Task-edit regeneration scenario prove the visible summary refreshes without Save. |
| SPD-13 Live Effort and Commitment projection | AC36, AC47, AC74–AC75 | `application.TaskProjection`, GORM detail/suggestion/candidate selects, projection token, HTTP Task DTO, frontend domain, and HTTP gateway carry canonical `effortMinutes`, `commitmentStart`, and `commitmentEnd`; existing Execution fields remain live. No Sprint persistence or scheduler input is added. | `repository_test.go` verifies detail, suggestion, draft candidate, saved candidate, token drift, and unchanged usage; `httpSprintsGateway.test.ts` verifies field mapping. | HTTP detail/suggestion/candidate contract tests assert all live planning metadata; Sprint component metadata scenario verifies observable rendering. |
| SPD-14 Context-sensitive Assignee presentation | AC34, AC36, AC50–AC52 | `TaskMetadata` omits Assignee inside selected Member groups and adds `Assignee: {Name|Unassigned}` only under `NeedsReviewDailyPlan`. | Component metadata tests inspect normal and Needs Review Task rows. | `renders Member daily plans directly...` proves a reassigned non-member shows Tony only under Needs Review; the dedicated metadata workflow proves `Unassigned`. |
| SPD-15 Compact complete metadata formatting | AC36, AC76 | `formatTaskDateRange`, `formatEffort`, `MetadataRow`, and the compact `<dl>` implement whole/decimal hours, `Not set`, all four compact range variants, `Not scheduled`, identical ranges, and screen-reader-only complete start/end dates. | `SprintTaskReview.test.tsx` — `renders immediate Parent Name, compact planning metadata, and Needs Review ownership_SPD13To17_AC34And36`. | The same rendered workflow verifies visible scan order and accessible full-date text without hover or tooltip. |
| SPD-16 Documentation and regression reconciliation | AC81 and documentation gate | `US-8.1`, `docs/project/architecture.md`, `docs/project/schedmind-context.md`, this evidence file, and `docs/project/README.md` describe usage, live metadata, context-sensitive Assignee, compact dates, and unchanged scheduling/persistence boundaries. | Static consistency review plus the tests mapped above. | Manual Sprint Planning review steps below cover the complete observable delta. |
| SPD-17 Immediate Parent Task context | AC34, AC36, AC47, AC74–AC76 | `gormrepo/wbs_context.go` resolves each Task's direct Parent Name in the existing set-based WBS read, falling back to owning Project Name only for root WBS nodes; `TaskProjection`, HTTP DTO, frontend domain/gateway, and `TaskRow` carry/render the value. WBS row Name participates in the projection token so parent rename/move cannot be restored as stale context. | Repository tests verify child and root parent resolution plus token refresh after parent rename; gateway tests verify mapping. | HTTP detail/suggestion/candidate contracts expose `parentName`; Sprint Planning component tests show Parent Name instead of Project Name, keep Project status, update after regeneration, and use Parent Name in accessible allocation-cell labels. |
| SPD-07 Coherent API order, recoverable reads, and projection inputs | AC47, AC74–AC75, AC77 | Set-based WBS context/allocation/capacity reads; `projectionToken` includes Sprint, Member, Task, Project, WBS ancestor, override, allocation, and holiday inputs; gateway maps new fields; `SprintTaskReview` request-generation guards and active request cancellation reject stale suggestion/candidate responses after a newer detail projection; `application.Service.Suggest` normalizes repository read/integrity failure to `ErrSuggestionUnavailable` | `backend/internal/sprints/application/service_test.go` — `TestSuggestMapsRepositoryReadFailureToRecoverableContract_SPD07_AC77`; `backend/internal/sprints/infrastructure/gormrepo/repository_test.go` — `TestSuggestionRejectsUnreadableCanonicalAllocationWithoutPersisting_SPD07_AC77`; repository ordering test mutates ancestor WBS position and verifies both path and token change; gateway mapping test covers new Task/Member/Totals fields | `backend/internal/sprints/transport/http/handler_test.go` — `TestHTTPSuggestionReturnsStableUnavailableErrorForUnreadableAllocation_SPD07_AC77`; HTTP detail/suggestion/candidate contract assertions; `backend/internal/sprints/transport/http/handler_test.go` — `TestHTTPDetailUsesBoundedSetBasedQueriesAsTaskCountGrows_SPD07_AC75` proves the public detail workflow keeps a constant read-query count as Sprint Task cardinality grows; `SprintTaskReview.test.tsx` — `preserves reviewed membership when suggestion generation fails_SPD07_AC77`, `ignores a stale suggestion response after a newer Sprint projection is reviewed_SPD07_AC74` and `ignores a stale candidate response after Sprint detail changes_SPD07_AC74` control response order, assert the obsolete requests are aborted, and prove stale Task projections cannot overwrite or reopen the current review |
| SPD-08 Earlier daily-plan documentation reconciliation | Documentation gate | `docs/project/architecture.md`, `docs/project/schedmind-context.md`, this evidence file, and `docs/project/README.md` | Static review only | Static review only |

## Regression boundaries

- Suggestion eligibility and fill order remain owned by the existing suggestion
  algorithm. Only the returned review presentation order changes to canonical
  daily-plan order.
- Sprint still stores only aggregate fields and Member/Task relations. Capacity,
  immediate Parent/Project/WBS context, Effort, Execution/Commitment dates,
  schedule, and allocation remain live projections.
- Add/Remove changes local reviewed membership only. It does not alter Task,
  Project, schedule, Assignee, allocation, or capacity data. It does immediately
  recalculate the visible uncapped Member usage from selected in-range allocation.
- Overcapacity remains allowed and does not block Save.
- Completed Tasks remain retained. Readable completed Tasks are displayed after
  unfinished daily-plan Tasks; completion alone is not a Needs Review condition.
- Closed, reassigned-to-non-member, cleared-Assignee, unscheduled, or allocation-
  inconsistent retained Tasks remain visible under Needs Review rather than being
  silently removed. A new suggestion with unreadable canonical allocation fails
  through the stable recoverable `SPRINT_SUGGESTION_UNAVAILABLE` contract and
  persists no Sprint rows.
- Positive outside-Sprint allocation remains excluded from utilization and from
  the visible Date grid. It remains a live canonical ordering/read-model input
  and is not mutated or deleted.
- Sprint Planning reuses Home's `WBSPanel`; it does not introduce a second Task
  form, Task mutation path, or navigation route. Closing the editor leaves local
  selection untouched. Saving a Task regenerates local suggestion, usage, and
  metadata state but does not persist Sprint membership.
- Immediate Parent Name is display-only live hierarchy context. Parent rename or
  move refreshes the projection token and visible row context but does not affect
  suggestion, grouping, usage, capacity, allocation, Date columns, or row order.
  Root WBS nodes use owning Project Name as their direct level-0 Parent Name.
- Effort and Commitment are display-only live metadata. They do not affect
  suggestion, grouping, usage, capacity, allocation, Date columns, or row order.
- Normal Member rows preserve grouped ownership context by omitting Assignee; only
  Needs Review adds current Assignee or `Unassigned`.

## Static query and index review

The implementation adds no per-Task or per-Date query loop. Detail and suggestion
use these bounded set-based shapes:

1. selected Sprint Members;
2. selected/candidate Task rows joined to Project and Assignee, including existing
   Effort and Execution/Commitment columns in the same select;
3. all WBS nodes for the represented Project IDs in one query, selecting Name and
   hierarchy keys and composing immediate Parent Name plus depth-first path/rank in
   memory;
4. all canonical Execution allocation rows for the Task ID set in one query;
5. all overlapping capacity overrides for the Member/date set;
6. all holiday Dates for the Sprint range.

The WBS context query matches `wbs_nodes_project_tree_idx
(project_id, parent_key, position, id)`. Allocation reads use the primary key
`(task_id, timeline, allocation_date)`; candidate readability checks are
correlated by Task/timeline/Assignee and the existing
`task_schedule_allocations_member_date_idx` remains available for member/date
access. The Parent Name and planning-metadata projection requires no additional query,
join, or index. No schema change is required. PostgreSQL plan execution was not performed
under `USER_LOCAL_VALIDATION`; measured plan evidence remains a local validation
responsibility if dataset characteristics materially differ from the existing
US-8.1 plan gate.

## Local validation commands

Run from the repository root.

### Targeted backend

```sh
cd backend
go test ./internal/sprints/application -run 'TestSuggestMapsRepositoryReadFailureToRecoverableContract'
go test ./internal/sprints/domain -run 'TestCompareDailyPlanTaskOrdersCanonicalDailyWorkingPlan'
go test ./internal/sprints/infrastructure/gormrepo -run 'TestDetailComposesLiveCapacityOutsideAllocationAndNeedsReview|TestDetailOrdersFlatDailyPlanAcrossProjectsAndWBS|TestDetailPreservesCrossMemberRemainingAndOvercapacity|TestDetailTreatsWeekendAllocationAsFullOvercapacity|TestSuggestionSelectsMandatoryZeroAllocationThenFillsWholeTasks|TestSuggestionRejectsUnreadableCanonicalAllocationWithoutPersisting|TestCandidatesReturnOnlyUnselectedEligibleTasksIncludingZeroInSprintAllocation'
go test ./internal/sprints/transport/http -run 'TestHTTPMemberDeleteRetainsSprintTaskAsNeedsReview|TestHTTPCreateDetailStartAndDeletePreservesOwningEntities|TestHTTPSuggestionAndCandidatePickerExposeCanonicalAllocationWithoutWriting|TestHTTPSuggestionReturnsStableUnavailableErrorForUnreadableAllocation|TestHTTPDetailUsesBoundedSetBasedQueriesAsTaskCountGrows|TestHTTPDetailPreservesDailyCrossMemberVarianceWithoutNetting|TestHTTPDetailPreservesCrossDateVarianceWithoutNetting|TestHTTPAllowsZeroCapacityAllocationAndReportsFullOvercapacity'
cd ..
```

Success criteria: every command exits `0`; no scenario reports reordered rows,
netted variance, missing Parent/WBS path/order date, missing live planning metadata,
stale projection token, altered usage, or changed Sprint ownership behaviour.

### Targeted frontend

```sh
cd frontend
npm test -- src/features/sprints/presentation/SprintTaskReview.test.tsx
npm test -- src/features/sprints/infrastructure/httpSprintsGateway.test.ts
npm test -- src/features/sprints/presentation/SprintFormDialog.test.tsx
npm test -- src/features/sprints/presentation/SprintsPage.test.tsx
cd ..
```

Success criteria: every command exits `0`; the component exposes canonical row
order, one-grid Sprint-period allocation, uncapped usage-versus-capacity summaries,
immediate Parent Name, compact Effort/Execution/Commitment metadata,
context-sensitive Assignee, zero-capacity markers, Add/Remove behavior, focus
preservation, shared Task editing without navigation, post-save usage/Parent
regeneration, and mapped API fields.

### Repository-required full validation

```sh
cd backend
go fmt ./...
go vet ./...
go test ./...
go test -race ./...
cd ..

cd frontend
npm run format:check
npm run lint
npm run typecheck
npm test
npm run build
cd ..
```

Success criteria: every command exits `0` with no formatter, vet, race, lint,
type, test, accessibility, or build failure. Because `go fmt ./...` writes files,
review its resulting diff before committing.

## Manual review

1. Open a Sprint containing Tasks from multiple Projects with the same Assignee.
2. Confirm canonical row order still uses earliest positive allocation Date, then
   Project Priority, WBS path/rank, and Task ID, without displaying WBS.
3. Confirm completed Tasks follow unfinished readable Tasks and unreadable Tasks
   remain under Needs Review.
4. Confirm there is no Sprint daily summary and each Member shows
   `Capacity: {Usage} of {Total}` plus per-Date Capacity and Task allocation rows.
   Verify `0h` for an empty Member and an uncapped value above total capacity.
5. Confirm Previous/Next Date controls are absent, the first column is Sprint
   Start, the last column is Sprint End, and positive allocation outside that
   range creates no visible Date column.
6. Confirm Date headers use `DD MMM YYYY`; Task metadata shows Effort plus compact
   same-Date, same-month, cross-month, and cross-year Execution/Commitment ranges.
   Verify `Not set`, `Not scheduled`, identical ranges, assistive full dates, `50ch`
   wrapping, and absence of WBS/aggregate allocation/First planned Date.
7. Set one Member Daily Capacity to `0h` through holiday or capacity override and
   confirm the complete Member Date column is marked, one textual `No capacity`
   state appears in its header, a different Date with `0h` Remaining but positive
   Daily Capacity is not marked, and Save remains allowed.
8. Add and remove a Task before Save; confirm Task rows and Usage change
   immediately while total Member capacity remains unchanged and only Sprint Task
   membership is submitted.
9. Make canonical allocation unreadable in a controlled fixture; confirm suggestion
   returns `SPRINT_SUGGESTION_UNAVAILABLE`, the review remains unchanged, and no
   Sprint row is persisted.
10. Click a Task Name from Sprint Planning and confirm the same Edit Task dialog
    used by Home opens without changing `#sprints`. Close it and confirm the local
    selection is unchanged. Reopen, Save a Task change, and confirm the dialog
    closes, Sprint Planning remains visible, suggestion, Usage, Effort, and
    Execution/Commitment metadata refresh, and Sprint Task membership is still not
    persisted until Save Sprint Planning.
11. Confirm a nested Task row shows its immediate Parent Name instead of its
    Project Name while retaining the owning Project status. Confirm a root WBS Task
    uses Project Name as its Parent Name. Rename or move a parent, regenerate/read the
    Sprint, and confirm the visible Parent context updates without changing membership,
    usage, or row order.
12. Confirm a normal Member-row Task does not repeat Assignee, while Needs Review
    shows the current Assignee or `Unassigned`.
