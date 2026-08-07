# Current Project Architecture

> **US-6.4 architecture revision (2026-08-03):** Dependency rows are manual-only
> Finish-to-Start prerequisites. Scheduler and preview read the manual graph but
> never infer or mutate relations. Migration 000023 deletes automatic-only rows,
> preserves manual/shared identity, drops ownership schema, and performs full
> active-portfolio recalculation atomically. This supersedes ownership and
> reconciliation text retained below as historical context.

This document records technical decisions specific to this repository. It
applies the reusable [architecture baseline](../architecture.md) and is not safe
to copy unchanged into another project.

## Repository layout

```text
backend/       Go API
frontend/      React web application
docs/          reusable and project-specific documentation
scripts/       local run scripts
user_story/    authoritative approved feature requirements
```

## Backend stack and composition

The backend uses Go 1.24, GORM, PostgreSQL for local/runtime persistence, and
SQLite only for isolated automated tests. The executable entry point is
`backend/cmd/api/main.go`; `backend/internal/httpapi` composes feature HTTP
transports.

Business features are organized under `backend/internal/<feature>/` with
`domain`, `application`, `transport`, and `infrastructure` packages when those
responsibilities exist. Technical health handling intentionally has only a
transport package because it has no business model or application workflow.
Shared pagination types and standard HTTP list-query parsing live under
`internal/shared/listing`. Other context-free technical helpers live in
narrowly scoped packages: `internal/shared/identity` owns UUID v4 generation,
`internal/shared/persistence` owns SQL `LIKE` pattern escaping, and
`internal/shared/httpjson` owns JSON response serialization. Feature handlers
retain request/response DTOs and error mapping. Application services continue
to receive ID generator functions so tests remain deterministic; feature
domains do not depend on the generator implementation.

The API performs graceful shutdown for interrupt and termination signals using
the environment-configured timeout. It stops HTTP work before closing the owned
database connection.

## Persistence decisions

- PostgreSQL 16 runs through Docker Compose for local development.
- SQLite is limited to isolated automated-test repositories so tests do not
  modify developer PostgreSQL data.
- GORM APIs are required for runtime `SELECT`, `INSERT`, `UPDATE`, and `DELETE`.
  Handwritten SQL is allowed only for version-controlled DDL migrations and
  isolated test DDL that the ORM cannot express.
- Runtime configuration comes from `DATABASE_URL`, `HTTP_ADDRESS`, and
  `HTTP_SHUTDOWN_TIMEOUT`.
- Migrations run when the API starts.

Name-list queries use case-insensitive prefix search with deterministic ordering
and backend pagination. PostgreSQL functional `LOWER(name) text_pattern_ops`
indexes support those queries. A future MySQL adapter must provide an equivalent
indexed path appropriate to its supported version without changing domain or
application contracts.

Dependency candidate search uses case-insensitive substring matching against
Task Name and Project Name because users may search using any meaningful part
of either name. Other list endpoints retain their existing prefix-search
contract unless their product requirements explicitly require substring search.

Every production query change is reviewed with its effective GORM scope,
filters, ordering, pagination, locking, expected cardinality, and supporting
index. Non-trivial PostgreSQL paths are checked with `EXPLAIN`. Indexes without
a confirmed query are removed rather than retained speculatively.

Capacity Override writes lock the parent Member row before checking exact
duplicate identity and persisting within a transaction. Overlapping ranges are
valid. Exact duplicate identity is normalized `(team_member_id, start_date,
end_date, capacity)` for active records; Description is excluded. Parent-scoped list and
inclusive effective-date queries use the `(team_member_id, start_date,
end_date, id)` index. Scheduler resolution loads all active overrides for one
Member/Date and selects minimum Capacity. The filter is applied before count,
limit, and offset. Create/update/delete compares before/after per-Date minimum;
only an effective-capacity change enters US-6.2 impact simulation.

Role deletion is a single repository transaction. It locks the Role row, rejects
active `team_members.role_id` references, rejects any `wbs_nodes.role_id` Task
reference with a distinct business error, detaches only soft-deleted Member Role
references, and then physically deletes the Role. `team_members.role_id` is
nullable only for soft-deleted history; a database check keeps it mandatory for
active Members. `wbs_nodes_role_id_idx` supports the Task-reference guard. The
Role-member audit endpoint reads only active Members, filters by Role and optional
case-insensitive name prefix, orders by `LOWER(name), id`, and paginates. The
existing `team_members_role_id_idx` bounds that lookup by team-sized Role
cardinality, so no additional Member audit index is required.

Member deletion soft-deletes the Member and all owned Capacity Overrides in one
transaction. Direct Capacity Override deletion remains a hard delete. Default
GORM scopes exclude deleted records; historical readers must opt in explicitly.
Active Member name search uses a PostgreSQL partial prefix index; names remain
non-unique and can also be reused after deletion. Member deletion checks
`wbs_nodes` joined to `projects`, rejecting an assigned executable leaf in an
Open or Locked Project while allowing assignments that exist only in Closed
history. The selective `wbs_nodes_assignee_id_idx`, Project primary key, and
bounded leaf anti-join support this query; no additional index is required.

## Backend API conventions

WBS `name` and normalized `name_key` support up to 200 Unicode characters in
the domain/API contract and use `VARCHAR(200)` persistence columns. Existing
sibling uniqueness and tree indexes remain unchanged.

Business transports expose JSON APIs below `/api`. List endpoints accept backend
search/filter and pagination as applicable and return `page`, `pageSize`, and
`total`. The default page size is `5`; transport caps page size at `100`.
Structured errors expose stable codes and safe field information without
database details.

Capacity Override is a nested Member resource rather than a top-level resource:

```text
/api/team-members/{teamMemberId}/capacity-overrides
```

Capacity Override stores a required, trimmed `description` (maximum 100
characters) as the user-managed reason for the temporary capacity change. The
frontend presents it as the list-item title, with date range and daily capacity
as secondary information.

The shared calendar popover chooses an above placement only when the complete
calendar fits there. When neither side has sufficient viewport space, it opens
below with a constrained, scrollable body so its header is not clipped.
The selected placement remains stable for the complete open interaction and is
recalculated only on the next open.

Transient success feedback uses the shared `Toast` presentation primitive.
The primitive owns consistent notification layering, status semantics, manual
dismissal, timer cleanup, and the default five-second auto-dismiss lifecycle;
features own only the message and confirmed operation state.

Detailed contracts remain authoritative in the applicable user story and root
developer README.

Public Holiday is a global scheduling feature with independent backend and
frontend feature boundaries. Roles, Members, and Public Holidays share the
**Team Configuration** navigation group, while Public Holiday remains a
standalone page and is not owned by Member or Project. A Public Holiday is an
aggregate: `public_holidays` stores its description and inclusive range, while
`public_holiday_dates` stores one row for every weekday in that range. Weekend
dates are omitted because Saturday and Sunday are holidays by default. The
unique child-date index enforces one Public Holiday per weekday and supports
exact scheduler/calendar lookups without expanding ranges at read time.
Mutations replace the aggregate atomically, so a conflict on any date rolls
back the complete range. Lists place ranges whose `end_date` is on or after
today first, followed by expired ranges; each segment orders by `start_date`,
`end_date`, then `id` ascending. Today is evaluated in `APP_TIMEZONE`.

The Public Holidays frontend request-cache identity includes `holidayDate`,
`page`, and `pageSize`. Confirmed mutations invalidate every filtered and
unfiltered Public Holiday cache entry, using versioned invalidation so older
in-flight responses cannot repopulate stale data. The application exposes an
exact-date and bounded calendar consumers for capacity resolution and shared
calendar markings. Public Holiday
has precedence over every overlapping Capacity Override and Daily Capacity.
On a non-holiday working Date, the minimum Capacity among all active overrides
for the Member becomes Resolved Daily Capacity; if none exists, Daily Capacity
is used. Member Buffer is then applied for Execution and Project Buffer after it
for Commitment. Capacity Override mutations that leave this per-Date minimum
unchanged persist without scheduler invocation. Public Holiday mutations follow
the US-6.2 impact guard because adding, changing, or removing a holiday changes
Resolved Daily Capacity for its derived working Dates.

Public Holiday endpoints are top-level global resources:

```text
/api/public-holidays
```

Project Management is an independent `projects` feature boundary. A Project is
WBS level `0`; creation does not create a separate WBS row. PostgreSQL stores a
normalized `name_key` for case-insensitive uniqueness and indexed prefix
search, a unique positive integer Priority, nullable derived dates, lifecycle
status, Closed At, a monotonic `schedule_version`, and nullable locked baseline
snapshots. Lock and Close traverse descendant executable leaves from
`wbs_nodes`; zero-leaf transitions are rejected. Lock serializes the current
Execution and Commitment leaf timelines into immutable snapshots before the
portfolio scheduler treats the Project as fixed reservations.

Active Projects (`open` and `locked`) precede Closed Projects and order by
Priority. Closed Projects order by Closed At descending and ID ascending.
Priority Move Up/Down locks the active ordering, swaps with the adjacent active
Project in one transaction, and invokes the concrete portfolio scheduler before
commit. Lock, Close, and Closed-to-Open status changes use the same schedule
mutation lock and transaction context. Scheduler failure rolls back the status,
Closed At, snapshots, Priority, generated dates, dependency ownership, and daily
allocation projection together.

Project endpoints are:

```text
GET    /api/projects
GET    /api/projects/{projectId}
POST   /api/projects
PUT    /api/projects/{projectId}
POST   /api/projects/{projectId}/status
POST   /api/projects/{projectId}/priority
PATCH  /api/projects/{projectId}/settings
DELETE /api/projects/{projectId}
```

`PUT /api/projects/{projectId}` has two explicit contracts. A payload containing
only `name` invokes the Project rename path, which is valid for Open and Locked
Projects and updates only `name`, normalized `name_key`, and `updated_at`.
Supplying scheduling fields requires the complete settings tuple and invokes
the existing full Open-Project update transaction. Partial tuples are rejected,
and a mixed Locked payload is rejected atomically. The Locked name-only path
does not acquire the schedule-mutation lock, emit scheduling impact, invoke the
scheduler, or change schedule version and protected snapshots.

Project Settings persist `automatic_scheduling` (default `true`), nullable
`scheduling_start_date` as SQL `DATE`, and integer `project_buffer` (default
`20`, constrained to `0..100`). The API represents the anchor as `YYYY-MM-DD`.
Only Open Projects may change settings; Locked and Closed settings are read-only.
A scheduling-relevant Project settings change always enters the concrete
portfolio scheduler inside the same settings transaction. The scheduler then
resolves each Task's effective Group/Project configuration, so an inherited
subtree follows the changed Project value while a custom Group override can
remain automatic even when the raw Project setting is OFF. Scheduler failure
rolls back the settings update and all derived projections atomically.

Group scheduling is persisted on grouping `wbs_nodes` as a local source
(`inherit` or `override`), optional Automatic Scheduling override, optional
Scheduling Start Date override, local lifecycle, monotonic scheduling version,
and frozen effective scheduling values while locally Locked. A shared
`groupscheduling` resolver walks the nearest Group ancestors and then the
Project. `override` always owns Automatic Scheduling; a null Group start-date
override continues to inherit the nearest parent Group date and finally the
Project date. Project lifecycle and locked ancestors are hard ceilings over a
Group's local lifecycle. Lock freezes the resolved values, including an
explicit no-anchor result, while Reopen re-resolves inheritance. Group
scheduling/lifecycle writes use optimistic `expectedVersion` checks in addition
to the schedule-impact token used by confirmation workflows. The Home Group dialog
uses one ordinary atomic write for Group Name plus scheduling source/overrides;
summary fields are never persisted, and lifecycle Lock/Reopen remains a separate
state transition. A legacy Group rename also advances `group_scheduling_version` so
a stale unified draft cannot overwrite a newer confirmed name.

Scheduling Start Date remains the Project-level default anchor, but an Open
Group may supply a nearer non-null override. Automatic Scheduling may be
effective ON without any resolved anchor; in that case the scheduler never
invents Task dates and stores the existing explicit unscheduled reason.
Executable WBS contains no Earliest Start, Start Constraint, or Task Anchor.
Lag belongs to the Task and is applied after the effective Group/Project and
dependency readiness anchor while zero-capacity dates are skipped.

The production composition creates `internal/scheduling/application.Service`
over the GORM scheduler repository and injects that same service into Project,
WBS, and Dependency application services. Mutation repositories propagate their
GORM transaction through `shared/persistence.WithTransaction`, so nested
scheduler work uses a savepoint on the same database transaction rather than an
independent commit.

The WBS create endpoint stores only structural input (`Name`, optional parent,
and conversion confirmation). A root or child create that does not convert an
existing executable node or retarget dependency endpoints therefore does not
invoke portfolio scheduling and returns the new Task with empty generated
dates. Create remains an atomic scheduling trigger when it performs executable-
to-group conversion and moves executable state or dependency endpoints; a
scheduler failure rolls back that conversion.

WBS delete uses the same scheduling-impact boundary. Deleting an unfinished
Name-only or Role-only leaf with default Lag, no generated or unscheduled
projection, and no dependency endpoints is a pure structural mutation. It does
not invoke portfolio scheduling, even when the Project contains completed
Tasks. Delete still coordinates affected-scope scheduling atomically when the
removed Task has Assignee, Effort, non-zero Lag, generated projection, automatic
unscheduled projection, or dependency endpoints; scheduling failure rolls back
the deletion.

The scheduler loads the transitive recalculation scope, validates unique Priority and WBS ordering, resolves effective Group scheduling/lifecycle together with manual and retained automatic dependency edges, and allocates Execution and Commitment independently. Recalculation propagates on schedule-relevant date, allocation, capacity, readiness, effective configuration, and unscheduled-state changes; warning classification is a later, narrower step. Effectively Open unfinished automatic Tasks are mutable outputs; effective manual Tasks are fixed manual reservations; Locked Groups and Locked Projects are immutable scheduling anchors; unrelated Projects are not recalculated or version-updated. Capacity arithmetic uses
`math/big.Rat`: weekend/Public Holiday resolves to zero, otherwise the minimum
active Capacity Override replaces Member Daily Capacity, Member Buffer produces
raw Execution Capacity and rounds it to the nearest `0.5` hour. Commitment
Capacity is calculated independently after both Member Buffer and Project
Buffer, then rounded to the nearest `0.5` hour. US-6.3 then applies persisted
Task Capacity Allocation Percentage (`1–100`, default `100`) to each final
timeline capacity and rounds the Task Daily Limit to `0.5` hour with a minimum
positive `0.5` hour. Dependency-ready Tasks remain ordered by Project Priority
then visual WBS order, but same-assignee Tasks may receive positive allocation
on the same Date. Later Tasks use only residual capacity; mutable automatic
allocation remains priority-safe. Completed Actual Allocation and Locked
baseline are absolute reservations. Fixed manual allocation is immutable but
uses Project Priority for capacity precedence: higher-priority manual rows reduce
lower-priority automatic capacity, while lower-priority manual rows may overlap
higher-priority automatic rows without warning or later capacity debt.

US-6.2 defines Actual Date, Actual Allocation, Locked Project, and generic cross-project impact coordination; US-4.4 extends the same protection model to Group scope. Completion persists required Actual Start/Actual End together. Open completion moves each planned Start earlier only when Actual Start is earlier and always sets each End to Actual End. Actual Allocation ignores planned Task percentage and uses only eligible Dates inside Actual Start–Actual End. It competes only with other completed Actual Allocation for the same Assignee/Date, distributes Effort in `0.5`-hour balanced shares, recalculates remaining shares after constrained Dates, backfills spare BAU capacity, and then levels unavoidable total historical overcapacity with a latest-Date tie-breaker. Existing completed Actual rows remain immutable; planned automatic/manual/Locked rows are ignored for Actual head-to-head construction. Historical overcapacity never carries debt to later Dates. Locked Projects and Groups remain immutable scheduler outputs, but Actual Date keeps the factual-data exception while the Project is not Closed. Cross-scope warning is emitted only when another Executable Task changes Execution Start/End or Commitment Start/End; allocation, capacity, readiness, or unscheduled-reason changes without a Task date delta may still persist and version without warning. A Group mutation excludes only its owning subtree from warning classification, so affected sibling Groups in the same Project remain visible. Ordinary planning mutations block on a Locked Group/Project only when counterfactual simulation would require one of those protected Task dates to change. Project and Group Reopen use the same fixed-point closure rule and may atomically require a mixed set of locked Groups and Projects to avoid mutual-lock dead ends.

Execution, Commitment, and Actual daily allocations are separate projections. Actual Allocation rows are canonical and support both Task-centric and assignee-centric queries; current UI exposes the Task-centric read-only verification section while assignee analytics remains deferred. Impact simulation is version-bound: preview returns only scopes whose descendant Task timelines actually change. Project operations keep the established Project warning boundary; Group operations exclude only the owner subtree and may return qualified sibling Group paths from the same or another Project. The confirmation token is bound to the full schedule-relevant recalculation signature and Project/Group versions, including non-warning scopes. Confirm re-simulates under scheduling locks, so hidden allocation/readiness/capacity changes can stale an otherwise identical visible warning set and stale impact never persists. Group `Reopen All` additionally fingerprints current Project scheduling/lifecycle state and the complete involved WBS hierarchy plus Group configuration/lifecycle/version state, so a hierarchy or parent-configuration change between preview and confirmation invalidates the closure token before lifecycle mutation.

`task_schedule_allocations` stores exact decimal allocation and remaining
capacity by Task, timeline, assignee, and date. Its primary key prevents duplicate
Task/date projection. `projects.schedule_version` and Group
`wbs_nodes.group_scheduling_version` provide optimistic scheduling-state versions.
Every scheduling mutation first acquires the process-wide re-entrant
serialization context, then opens or reuses its database transaction, then takes
the PostgreSQL transaction-level advisory lock before row locks. Nested scheduler
callbacks inherit both the serialization marker and transaction context, avoiding
mutex/database lock-order inversion. SQLite test infrastructure intentionally
treats the advisory lock as a no-op.
Frontend gateways share a projection clock so an older in-flight Project, WBS,
or Dependency response cannot repopulate state after a scheduling mutation.

## Frontend stack and structure

The frontend uses React 19, TypeScript, Vite, Tailwind CSS 4, Vitest, jsdom, and
Testing Library. ESLint performs static analysis, Prettier owns formatting, and
axe-core provides automated accessibility assertions in component tests. Its
structure is:

```text
frontend/src/
├── app/
├── features/<feature>/{domain,application,infrastructure,presentation}
└── shared/{application,infrastructure,presentation}
```

Layers are created only when the feature has the corresponding responsibility.
The application composition root is `src/app/App.tsx`. `AppShell` owns stable
global chrome; `ApplicationSidebar`, `TopBar`, and page content remain mounted
across feature navigation. `src/app/navigation.ts` is the source of truth for
page identifiers, labels, hashes, ordering, and breadcrumb/sidebar metadata.

The Role, Member, and Capacity Override features separate domain/application
models from HTTP DTOs. Capacity Overrides are presented inside the Member
workflow rather than as standalone navigation. The Roles page uses the narrower
`RoleAuditGateway` capability to open a searchable, paginated active-Member usage
dialog without widening Role contracts consumed by WBS, Sprint, or Member
selectors.

The WBS domain remains the only work-item model. Its frontend presents an
Executable WBS as **Task** and a Grouping WBS as **Group**; these labels are
derived from child existence and never create or persist a separate type field.
Home is the canonical active-WBS surface. `App.tsx` keeps Home mounted while it
opens the shared Project controller or a direct-entry WBS dialog for Group,
Task, Add Child, Add Sibling, and Move to. Add Sibling carries only an
insert-after WBS identity; the backend transaction resolves the anchor's current
parent and full persisted sibling order, shifts later positions atomically, and
creates an empty executable Task. The standalone Project Structure panel has
been removed; the WBS controller accepts only a direct Home action and renders
the corresponding shared dialog. Direct-entry dialogs fetch the authoritative
Project tree, so Move to destinations and sibling anchors are not inferred from
the current Home Role projection.

The Home portfolio is rendered as a synchronized split grid/timeline. WBS,
Name, Role, Assignee, Effort, Start, and End remain individually resizable data
columns. Their latest bounded widths are stored as a browser-local layout
preference and restored when Home is mounted again; malformed or unavailable
storage falls back per column without blocking current-session resize. The Name
header also owns one contextual icon-only Collapse All/Expand All control. It
operates on every expandable row in the currently loaded Role-adjusted
hierarchy, is absent when no row is expandable, and does not call the backend or
scheduler. Rows retain their existing single-line height at rest. On hover or
keyboard focus,
only a row with at least one eligible creation action temporarily expands. Its
labelled Add Sibling/Add Child controls render as a content-width floating second
line logically owned by Name, above following Project Grid separators and
clipped at the Project Grid/Timeline boundary. Rows without eligible creation
actions do not expand. The independently right-anchored overflow trigger remains
at the far right edge of the primary Name line; no separate Actions column
exists. The matching timeline row, virtual spacers, dependency geometry, and
vertical scroll bounds share the same temporary height so both panes remain
aligned. Eligible non-Project rows expose a dedicated leading drag handle, while
the remaining row surface is not draggable. Drag submits source identity, target
sibling identity, and
before/after placement to the same serialized WBS reorder transaction used by
the adjacent Move Up/Down fallback. The backend rejects stale, cross-parent,
cross-Project, or self placement, preserves Group descendants as one subtree,
and restores confirmed order on scheduler failure. Structural conflicts
invalidate the versioned WBS cache before the UI reloads authoritative rows, so
an older in-flight response cannot restore stale order. A restrictive Role filter
disables both drag and Move Up/Down because hidden siblings make placement
ambiguous. Name is the only row edit activation. Project and Group Role cells
are blank, while Task Role remains direct data. Start and End display
timezone-stable `D Mon YYYY` values. The timeline header uses three rows for working-day
sequence, grouped month/year, and calendar date. Timeline bars and other cells
are read-only.
Collapsed Project/Group identities use a third browser-local presentation
preference. The state initializes before the first hierarchy render, defaults to
fully expanded when storage is absent or invalid, and is written after
individual, bulk, or required ancestor-expansion changes. Unknown, unloaded,
filtered, or no-longer-expandable identities remain harmless and are ignored by
the current projection. Bulk updates add or remove only identities in the
currently loaded hierarchy, preserving preferences for Projects outside the
active filter. Storage failure never blocks current-session hierarchy use.

Structural actions reuse the WBS gateway commands, and Project lifecycle
actions reuse the same `ProjectsPage` controller used by Projects. Confirmed
mutations advance the shared schedule projection clock or explicitly reload the
portfolio projection while retaining selected filters and expansion IDs.

The Home Execution/Commitment projection is also a browser-local presentation
preference. The frontend reads only the supported projection enum at Home state
initialization, defaults safely to Execution when storage is missing, invalid,
or unreadable, and writes only after Apply or an applying Save As succeeds.
Storage failure never blocks the in-memory projection and does not persist saved
filter identity, Project/Role selections, range, expansion, scroll, or business
data. Column widths use a separate browser-local key and lifecycle; they never
enter the projection preference or backend saved-filter contract. Collapsed row
identities use another separate key and likewise never enter saved-filter,
projection, column-width, WBS, or schedule persistence.

The insert-after and target-placement write paths reuse the existing serialized
WBS transaction boundary. Each command first resolves a Project-scoped node by
the globally unique WBS primary key, then locks and orders the bounded sibling
set by `(project_id, parent_key, position, id)`. Insert-after shifts only the
affected position suffix; target placement rewrites only that sibling set after
moving the source identity in memory. The existing
`wbs_sibling_position_unique (project_id, parent_key, position)` constraint
protects persisted cardinality, while `wbs_nodes_project_tree_idx
(project_id, parent_key, position, id)` supports the locked sibling read,
deterministic ordering, and position predicates. No new query shape, join, or
index is required. Representative PostgreSQL lock/plan and concurrent mutation
validation remains `AUTHORED — NOT RUN — LOCAL VALIDATION REQUIRED`.

The approved US-4.3 implementation uses one frontend-only recursive read model over current confirmed WBS roots. View Group passes the selected subtree; Edit Project treats the Project as logical WBS level `0` and passes every top-level WBS root. One deterministic typed traversal returns Execution aggregate and coverage, Commitment aggregate and coverage, completed/total known Effort, percentage, and Task-without-Effort count. Timeline dates come only from complete Task pairs; completion requires complete Actual Date; missing Effort is never coerced to zero. Integer minutes remain the arithmetic source so half-hour precision and percentage calculation do not accumulate floating-point error.

The existing per-Project WBS tree contract already carries `children`, Effort, both timeline pairs, Actual Start, and Actual End, so US-4.3 adds no backend endpoint, persistence field, migration, or summary-specific production query. Edit Project reuses a fresh cached WBS tree or invokes the existing WBS read; its summary region owns local loading/error/Retry without blocking Project form draft or Save/Cancel. Existing Project `startDate`/`endDate` are not sufficient inputs for the two separate timeline summaries. The summary is not stored as an independent confirmed copy; it is derived again from the current confirmed roots/subtree. Existing versioned WBS cache invalidation prevents an older tree response from restoring stale summary values after a mutation. The WBS application gateway exposes an intentional confirmed-change subscription implemented by the shared projection clock, so open Group and Project summaries can request the same versioned tree again without presentation code importing infrastructure internals. Rollback-only Task schedule previews and unsaved drafts are deliberately excluded.

The Dependency feature stores one directed Finish-to-Start relation between
two executable WBS Tasks. Relations may cross active Projects, but Groups and
Tasks belonging to Closed Projects cannot become new endpoints. Completed
Tasks may be blockers; completed Tasks cannot become newly blocked, and an
existing relation whose blocked Task is completed is historical read-only.
Dependency editing is composed into the shared contextual Edit Task dialog
opened from Home. One endpoint pair is persisted once with `manual_owned` and
`automatic_owned` flags. The API projects the source as `manual`, `automatic`,
or `both`; deleting a shared relation removes manual ownership only, while an
automatic-only relation is read-only through generic delete and may be retained
as manual.

Dependency mutation serializes active-portfolio graph changes, validates the
complete directed graph, rejects direct and indirect cycles with a safe path,
and relies on the endpoint-pair unique constraint for concurrent duplicates.
When any affected Project uses Automatic Scheduling, create, manual unlink, and
keep-as-manual invoke concrete portfolio recalculation in the same transaction.
Automatic reconciliation may change only automatic ownership; manual ownership
and relation identity are preserved.

Task deletion explicitly hard-deletes incoming and outgoing relations in the
owning WBS transaction; foreign-key cascade is intentionally not used. When an
executable Task is converted to a Group, its executable data and dependency
endpoints move to the deterministic conversion child in the same transaction.
Any callback, persistence, or retarget failure rolls hierarchy and graph back
together.

Dependency APIs are:

```text
GET    /api/tasks/{taskId}/dependencies
GET    /api/dependency-candidates?taskId=...&direction=blockedBy|blocks&search=...&page=1&pageSize=5
POST   /api/dependencies
DELETE /api/dependencies/{dependencyId}
```

Dependency candidate search is backend-owned, case-insensitive substring search
against normalized Task Name and Project Name. Results are deterministically
ordered, cycle-filtered, and paginated.

Each candidate response includes a backend-generated `hierarchyPath`. The path
starts with the Project as WBS level 0, includes every ancestor Group in order,
and ends with the executable candidate Task. The frontend renders Task Name as
the primary label and the full hierarchy path as supporting context; it does not
reconstruct hierarchy from partial DTO fields.

Example:

```text
Task 2
NTB > Task 1 > Task 2
```

Leading-wildcard substring matching does not efficiently use the ordinary
B-tree prefix index. This is an accepted MVP usability trade-off for dependency
candidate search. If portfolio scale makes this query unacceptable, introduce a
dialect-appropriate search index, such as PostgreSQL trigram indexing, without
changing the domain or application contract. The relation table retains its
unique endpoint pair and covering indexes in both directions. Runtime access
uses GORM, not raw DML.

Task name and executable fields share one Edit Task dialog and one atomic
`PUT .../executable` operation. The repository locks and updates the Task by its
indexed primary key; sibling-name uniqueness remains protected by the existing
parent/name unique index. Group rename stays on the structural rename endpoint.
No additional query or index is required for the combined Task update.

For an Open unfinished Task with Automatic Scheduling enabled, Role, Assignee,
Effort, Lag, and Capacity Allocation Percentage blur events may request
`POST .../executable/preview`. This endpoint is calculation-only: it applies the
current draft inside the normal schedule-mutation serialization boundary, runs
the concrete portfolio scheduler using the same database transaction, reads the
generated Task projection plus the Task's dependency projection, and then
deliberately rolls back the transaction. A cleared Assignee is still a valid
preview input: the scheduler removes assignment-owned automatic dependencies,
recalculates downstream work, and returns the Task as unscheduled with the safe
missing-Assignee reason. Temporary Task fields, dependency ownership, daily
allocations, Project dates, and schedule versions therefore never become
confirmed state. The frontend renders preview dependency rows as
unconfirmed and read-only, cancels or resolution-orders superseded preview
requests, and does not invalidate confirmed WBS caches. The existing atomic
`PUT .../executable` remains the only operation that persists the draft.

Projects use the same feature-layer boundaries and shared list, search,
pagination, dialog, loading, and Toast primitives. Their list cache identity is
`search/page/pageSize`; mutations invalidate every Project list entry so an
older in-flight response cannot restore stale ordering or lifecycle state.

Project Name and settings share the same Add/Edit dialog and persist through one atomic create or update operation; there is no separate Settings action. Edit Project opts into the shared wide Dialog variant and composes the read-only US-4.3 whole-Project summary after Project fields; Add Project remains summary-free and may retain standard width. Summary loading/error is isolated from form mutation state and is excluded from the Project payload. Status
remains outside the form and changes only through lifecycle command buttons.
Open settings are editable, Locked settings remain visible but read-only while
Name keeps its existing edit contract, and the complete Closed form is
read-only. Closing discards pending changes. OFF→ON uses nested confirmation,
while Project Buffer stays stored and disabled whenever Automatic Scheduling is
off. The dedicated PATCH endpoint remains available for API compatibility.

## Frontend state and shared infrastructure

HTTP gateways use `src/shared/infrastructure/RequestCache.ts` to deduplicate
identical list loads and reuse confirmed results. Consumer cancellation does not
cancel shared work. Versioned invalidation prevents stale in-flight responses
from repopulating a cache after mutation.

Dependency detail caches are keyed by Task ID. Dependency mutations invalidate
both endpoints, while WBS conversion or Task deletion invalidates all cached
dependency details because endpoint identity may change. Candidate requests
include Task, direction, search, page, and page size in their identity, and
consumer cancellation prevents stale UI restoration.

The composition root invalidates the Member list after a successful Role
mutation because Member projections embed the Role name. Capacity Override
mutations invalidate filtered and unfiltered entries while preserving the active
effective-date filter and confirmed visible data during safe refresh.

Shared list search, pagination, skeleton, focus management, and calendar
mechanics live in `src/shared/presentation`. Shared primitives also own the
application's button, field, dialog, alert, empty-state, and list-surface
contracts. Feature code retains business selection and result-state decisions.

When Member presentation needs Role options, the Member feature consumes its
own minimal `RoleOptionsGateway` contract. `App.tsx` connects that contract to
the existing Role adapter and composes Capacity Overrides into the selected
Member workflow. Feature presentation modules therefore do not import one
another's internals.

## Project theme and design tokens

Tailwind theme configuration in `frontend/src/app/styles.css` is the authoritative
token-value source. It defines the current brand, semantic text and surfaces,
feedback colors, status indicators, typography, spacing, sizing, layout,
radii, elevations, motion, and named layer levels. Those exact values are
SchedMind theme choices; the semantic contracts follow the reusable
[design-system standard](../frontend-design-system.md).

The shared calendar owns month navigation, accessibility mechanics, and
viewport-aware placement. Capacity Override presentation supplies either its
single-date filtering rule or two-step date-range rule.

## Configuration and local topology

The root `.env.example` documents safe local configuration. Integration values
must remain environment-driven. The frontend consumes `VITE_API_BASE_URL` and
the Vite development server uses `VITE_BACKEND_PROXY_TARGET`.

Local PostgreSQL is started with:

```sh
docker compose up -d postgres
```

The stack can be started with:

```sh
./scripts/run-all.sh
```

Individual entry commands are `./scripts/run-backend.sh` and
`./scripts/run-frontend.sh`.

## Validation commands

Backend:

```sh
cd backend
go fmt ./...
go vet ./...
go test ./...
go test -race ./...
```

Frontend:

```sh
cd frontend
npm run format:check
npm run lint
npm run typecheck
npm test
npm run build
```

After backend source, configuration, dependency, or migration changes, restart
the backend and smoke-test an affected endpoint. Documentation-only changes do
not require an application restart.

## Sprint management

US-8.1 adds a standalone Sprint aggregate and frontend feature boundary. Sprint
is not nested under Project or Portfolio despite being placed under the visible
Project navigation group. Persist only Sprint fields, optimistic Version,
selected Member relations, and selected Task relations. Capacity, Task identity,
Project/WBS context, Effort, Execution/Commitment dates, status, and canonical
Execution allocation are live read projections; do not copy them as Sprint-owned
snapshots.

The backend keeps Sprint commands in `internal/sprints/application` and the
aggregate invariants in `internal/sprints/domain`. The GORM and HTTP adapters
compose existing capacity/scheduling tables without importing Sprint
into scheduler domain logic. Suggestion is deterministic read computation: for
each selected Member, include every eligible unfinished scheduled Task with
Execution End on or before Sprint End regardless of capacity, then add later
Tasks with positive in-period Execution allocation in Execution End, Project
Priority, WBS, and ID order until capacity is met/exceeded or candidates end.
The calculation must reuse final per-Date Member Execution capacity and canonical
Task Execution allocation; it must not derive allocation from Effort or write any
schedule data.

The persistence target is `sprints`, `sprint_members`, and `sprint_tasks` (or an
equivalent normalized model) with normalized Name uniqueness, relation
uniqueness, optimistic Version, and atomic create/edit/start/delete. Inclusive
range overlap for a shared selected Member must be coordinated transactionally
so concurrent conflicting writes cannot both commit. PostgreSQL mutations take
transaction-scoped advisory lock `76081` before overlap validation and write;
SQLite tests use a process-local serialization guard because SQLite lacks that
lock. Overlap errors retain the stable code and include the conflicting Sprint,
inclusive range, and shared Member IDs/names. Task hard-delete cleans up
Sprint Task relations in the owning Task transaction. A valid Member soft-delete
is not blocked solely by Sprint; Sprint Member relations are removed in the
Member transaction and retained Sprint Tasks surface live Needs Review warnings.

Sprint detail and suggestion reads must be bounded and set-based across selected
Members, candidate/selected Tasks, capacity inputs, holidays/overrides, Projects,
complete WBS trees for represented Projects, and allocation rows. The selected
Task row query also projects current Effort plus complete Execution and Commitment
date pairs; these fields require no separate lookup. Avoid per-Member, per-Task,
per-Date, or per-Project N+1 access. The read model exposes
one server-resolved flat daily-plan order using earliest positive allocation Date,
Project Priority, depth-first WBS rank, and Task ID, with readable completed Tasks
after unfinished Tasks and unreadable Tasks in deterministic Needs Review fallback
order. The projection token includes the Sprint, Member/capacity values, Task and
Project values, Task Effort and Execution/Commitment dates, WBS ancestor ordering
inputs, overrides, holidays, and allocation rows so changed planning metadata, path,
or order cannot be paired with a stale token.
Suggestion read or canonical-allocation integrity failures are normalized by the
Sprint application boundary to the stable recoverable
`SPRINT_SUGGESTION_UNAVAILABLE` contract. The HTTP adapter does not depend on the
Scheduling domain and a failed suggestion writes no Sprint aggregate or relation.

For every Sprint Date, Member Capacity, Selected Allocation, Remaining, and
Overcapacity are composed independently. Member period values sum their daily
values; Sprint daily and period values sum Member daily values. Do not recompute
Remaining or Overcapacity from aggregate Capacity minus aggregate Allocation,
because that would net one Member or Date against another. Readable Needs Review
allocation is exposed separately by Date and excluded from selected-member
utilization.

The frontend renders a flat Member-grouped daily grid without loading a separate
WBS tree. The set-based WBS context projection resolves each Task's immediate Parent
Name; root WBS nodes use the owning Project Name. The UI keeps that Parent Name and
owning Project status on every Task row while WBS path/rank remains an internal
ordering input. The grid contains exactly the inclusive Sprint
Start-to-End Date range in one horizontally scrollable grid with no Previous/Next
Date controls. Positive allocation outside the Sprint Period remains available
to canonical ordering/read-model logic but creates no visible Date column or Task
allocation cell. Each Member exposes `Capacity: {Sprint Usage Capacity} of
{Sprint Execution Capacity}` plus per-Date Capacity; usage is the uncapped sum of
positive selected-Task Execution allocation inside the Sprint range, excludes
Needs Review and outside-Sprint allocation, and is recalculated from the current
local review selection after Add, Remove, regeneration, and Task-edit refresh.
Task allocation remains on Task rows. A `0h` Member Daily Capacity marks the
complete Member Date column and adds one textual `No capacity` state in its
header; Remaining Capacity does not control the marker. Sprint, Remaining, and
Overcapacity summaries are intentionally not rendered.

Task Name is the edit activation inside Sprint Planning. It opens the same
`WBSPanel` direct-entry Task controller used by Home, loading the authoritative
Project and WBS tree without changing the active hash/page. Closing the shared
dialog leaves the local Sprint Planning selection untouched. A confirmed Task
mutation closes only the dialog and invokes the existing Sprint suggestion read
against the saved Sprint Dates/Members; the replacement remains local until
`Save Sprint Planning`. The regenerated projection immediately refreshes Member
usage and live Task metadata. Home keeps its existing composition-root overlay
wiring unchanged.

Migration `000025_manage_sprints` supplies deterministic list and reverse-relation
indexes. The set-based WBS context read selects node Name together with hierarchy
keys to resolve immediate Parent Name in memory and reuses
`wbs_nodes_project_tree_idx`; allocation composition and candidate readability
checks reuse the allocation primary key and
`task_schedule_allocations_member_date_idx`. Projecting existing Task Effort and
Commitment columns adds no join, persistence field, schema migration, or
speculative index; any additional index still requires measured PostgreSQL plan
evidence.

The PostgreSQL query-review gate was measured against temporary tables cloned
from the production schema and indexes with 10,000 Sprints, 200 Members, 500
Projects, 20,000 Tasks, 50,000 Sprint-Member relations, 100,000 Sprint-Task
relations, 200,000 allocation rows, 4,000 capacity overrides, and 1,000 holiday
dates. `EXPLAIN (ANALYZE, BUFFERS)` confirmed indexed normalized-Name lookup and
Sprint list ordering; primary-key/index access for detail relations; the
Member-to-Sprint reverse index for overlap validation and Member cleanup; the
Task-to-Sprint reverse index for Task cleanup; assignee-index restriction for
candidate Tasks; and indexed allocation, override, and holiday range reads.
The candidate leaf check builds one set-based anti-join input rather than
issuing per-Task queries. The overlap plan first restricts the 50,000 relation
rows through the shared-Member index; its date filtering used a sequential scan
of 10,000 Sprints at the measured broad selectivity, so no additional
speculative date index is justified. Representative execution times were below
2.2 ms for every reviewed shape on the local PostgreSQL 16 validation dataset.

The Sprint Create/Edit dialog owns only Details and Members. A saved Sprint is
planned on the main Sprint page, where Task membership is generated and managed.
Sprint Planning groups by selected Member but does not render Project roots,
Group rows, or a WBS hierarchy. It consumes server-provided WBS path/rank only for
canonical ordering, keeps immediate Parent Name plus owning Project status on each
flat Task row, uses Project Name for a root WBS node, and renders a
compact definition list for Effort, Execution, and Commitment. Normal Member rows
do not repeat Assignee; Needs Review rows show the current Assignee or
`Unassigned`. Complete ranges use same-Date, same-month, cross-month, or
cross-year compact formatting; missing Effort is `Not set`, incomplete date pairs
are `Not scheduled`, and full start/end dates remain available to assistive
technology. Date headers remain `DD MMM YYYY`, Task names wrap within `50ch`, and
order is preserved while the user horizontally scrolls one all-Date grid. Editing
Details or Members preserves existing Task IDs; suggestion replacement remains an
explicit page action.

Sprint endpoints are:

```text
GET    /api/sprints
POST   /api/sprints
POST   /api/sprints/suggestion
POST   /api/sprints/task-candidates
GET    /api/sprints/{sprintId}
PUT    /api/sprints/{sprintId}
POST   /api/sprints/{sprintId}/start
DELETE /api/sprints/{sprintId}?version={version}
GET    /api/sprints/{sprintId}/task-candidates
```

The collection candidate endpoint accepts unsaved date/member context and
explicitly reviewed Task IDs, enabling manual Add before aggregate creation
without persisting a temporary Sprint. The entity candidate endpoint derives
the same context from a saved Sprint. Both are read-only and paginated.

## Deliberate project constraints

- PostgreSQL is the current supported runtime dialect; MySQL compatibility is a
  query/index design consideration, not an enabled runtime.
- Raw runtime DML is prohibited by project decision, while version-controlled
  raw DDL is permitted.
- Default production-list page size `5` is a product/project UX decision, not a
  reusable universal constant.
- Current hash-based navigation and custom request cache are implementation
  choices, not requirements of the reusable frontend architecture.

## Task Capacity Allocation target contract

Executable WBS persists a non-null integer `capacity_allocation_percentage` with
valid range `1–100` and default `100`. Rollout must backfill every existing
Executable WBS to `100` before enforcing the invariant; otherwise numeric zero
could incorrectly make legacy Tasks consume no capacity. Legacy create omission
normalizes to `100`, update omission preserves the existing value, and explicit
`0` is rejected. Selecting, changing, or clearing Assignee preserves the
Task-level value. Update omission preserves the existing value even when
Assignee changes; only an explicit valid percentage replaces it.

Execution and Commitment Task Daily Limits are calculated after each final
timeline capacity has been independently rounded. The percentage is a maximum,
not a reservation or priority. Automatic rows are ordered by Project Priority
and visual WBS order and share residual same-Date capacity. Manual timeline rows
are fixed reservations and may persist planned overcapacity. The field moves
with executable data during WBS conversion; Grouping WBS never owns it.

## Assignee recommendation target contract

US-6.5 extends the existing WBS Task boundary with one batch recommendation
operation for a confirmed executable Task identity. The operation is a query-like
calculation command, not a new aggregate and not a Task mutation. It resolves all
active Role-matching candidates in one request and returns backend-ranked results
with calculation Date and schedule-version snapshot identity.

Automatic Scheduling `ON` applies each candidate draft inside the existing
schedule-mutation serialization and database transaction boundary, invokes the
same concrete scheduler used by confirmed Task mutation, reads the candidate
Execution result and ranking metrics, and deliberately rolls back. The edited
Task's own planned allocation/projection is removed before each candidate is
applied so current assignment is not counted twice. The scheduler algorithm,
Project anchor, horizon, priority, dependency, capacity, rounding, fixed/manual,
completed, and Locked rules remain owned by US-6.1/US-6.2/US-6.3; recommendation
must not maintain a duplicate implementation.

Automatic Scheduling `OFF` preserves confirmed/manual timelines and runs a
side-effect-free advisory Execution allocation using the same capacity,
dependency, Lag, priority, WBS-order, fixed-allocation, and horizon primitives.
Its start lower bound is draft Manual Execution Start when present; otherwise
`max(Project Scheduling Start Date, today)`; otherwise today. Backend resolves
today in `APP_TIMEZONE`. Manual Execution End is neither changed nor used as the
ranking completion result.

All candidates share one confirmed Recommendation Snapshot. The backend ranks:
feasible completion without incremental candidate-Member overcapacity, then
completion with added overcapacity, then no completion. Inside the first two
groups it orders by earliest simulated Execution End, largest Execution Capacity
remaining immediately after candidate allocation on that completion Date, then
normalized Member name and ID. Lower-priority work that consumes residual
capacity later is excluded from the remaining-capacity metric, and downstream
lower-priority schedule impact is not an optimization objective.

The frontend places Role and Assignee immediately after Lag and before timeline
and dependency controls, requests the latest batch before stale options become
selectable, freezes order
for the open option interaction, and ignores superseded responses. Failure
falls back to deterministic alphabetical selection without blocking normal Task
Save. Recommendation never reserves capacity, advances schedule version,
invalidates confirmed caches as a mutation, or makes its snapshot a Save token.

Production access must avoid one HTTP request or database candidate query per
Member. Candidate lookup and scheduler inputs are loaded in bounded sets and
reuse one snapshot; every new/changed query remains subject to the backend query
review gate.

## Actual Date and Reopen completed Task target contract

US-6.2 replaces Actual End-only completion with a complete Actual Date pair.
The WBS persistence target therefore stores nullable `actual_start` and
`actual_end` with a pair invariant: both null or both non-null, and
`actual_start <= actual_end`. Completion command writes both fields together.
Reopen command clears both fields together; generic Task update cannot bypass
either command.

Actual Allocation is a separate daily projection keyed by Task, Assignee, Date,
and projection kind. Execution, Commitment, and Actual allocation rows remain
separately queryable. Construction of a Task's Actual rows considers only other
completed Actual rows for the same Assignee/Date; unfinished automatic, manual,
and Locked planned rows are ignored. The algorithm works only inside Actual
Date, preserves `0.5`-hour Effort exactly, progressively rebalances remaining
shares, backfills spare BAU capacity, and equalizes unavoidable total
overcapacity. Existing completed Actual rows are immutable, so completion order
may affect attribution. For future unfinished recalculation, Actual Allocation
is the sole capacity-consuming reservation for a completed Task; Execution and
Commitment rows remain planning/baseline evidence and are not double-counted.
One canonical Actual row set supports both Task-centric and assignee-centric
queries.

Actual Date/Reopen commands run impact simulation before persistence. The
preview is version-bound and returns only timeline-impacted Project names grouped
by Open and Locked. Another Open Project requires confirmation only when one of
its Executable Tasks changes Execution Start/End or Commitment Start/End. An
ordinary mutation is rejected only when counterfactual simulation requires a
protected Task date in another Locked Project to change; allocation-only pressure
does not make the Locked Project impacted. Actual Date is the factual-data
exception: after grouped confirmation it persists while Locked Projects remain
immutable and the complete transitive Open recalculation scope may persist.
Reopen Task is not an exception and remains blocked by timeline impact to a
Locked Project.

Project Reopen calculates a transitive Required Locked Reopen Closure. Mutual
A/B or transitive A/B/C lock dependencies are presented as one `Reopen All`
operation. Closure discovery is simulation-driven rather than graph-driven:
shared Assignee/dependency connectivity defines only candidate recalculation
scope, while a remaining Locked Project joins the closure only when the
counterfactual scheduler would change one of its protected Execution/Commitment
Task dates. Project Priority remains authoritative, so reopening a lower-priority
Locked Project does not pull a higher-priority Locked Project into the closure
when that higher-priority baseline remains unchanged. The Open list shown in the
Reopen warning is likewise the timeline-impacted subset, not every Project that
was considered or recalculated; a missing-anchor Project with no existing
timeline is therefore omitted. Closure status changes, unfinished scheduling, dependency
reconciliation, allocation persistence, and version changes are atomic. Stale
preview, optimistic conflict, or persistence failure leaves all statuses and
projections unchanged.

### Query and index review target

- Completion/Reopen point mutation uses primary-key plus Project scope and a
  pair-state predicate; cardinality remains at most one Task.
- Actual Allocation lookup requires Task/date and assignee/date access paths so
  both read models avoid full scans.
- Impact preview/confirm uses the full schedule-relevant recalculation signature plus schedule/status versions rather than trusting a
  client-provided warning Project list.
- Allocation writes and affected schedule rows share one transaction boundary.
- Exact index design must follow measured PostgreSQL query shapes before
  implementation; this requirement does not authorize speculative indexes.

## Home Portfolio Gantt target contract

US-7.1 adds a Home feature boundary as the default frontend composition. Home is
a portfolio read workspace, not a new WBS or scheduler aggregate. The left grid
renders selected active Project/WBS rows, while the right timeline renders
read-only daily Execution or Commitment bars and effective dependency arrows.
The standalone Project Structure entry point is removed. Project-level Add Child,
non-Project Add Sibling/Add Child, Project edit, and Task/Group edit must reuse
existing application use cases and dialogs directly over Home so the canonical
workspace does not fork validation, impact preview, transaction, or rollback
behaviour.

The Home read path requires a dedicated portfolio projection or equivalent
bounded set-based composition. It must return stable Project/WBS identities,
ordered hierarchy, assignee display data, effort minutes, both approved planning
date pairs (or a version-safe projection-specific equivalent), unscheduled
state, effective dependency endpoint pairs, Project/schedule versions, and
Public Holidays for the rendered range. The implementation must avoid
per-Project, per-Task, per-dependency, and per-date N+1 requests. Exact endpoint
naming is local design; observable ordering, version safety, date-only mapping,
and query-plan review are mandatory.

WBS numbers in Home are derived presentation values. Project is level `0` but
its WBS cell is blank; descendant numbering restarts at `1` inside each Project.
Project and Group dates and known Effort use the same recursive confirmed-task
semantics as US-4.3 and are not persisted as new writable aggregate columns.
The daily timeline includes weekends and Public Holidays. Its first header row
counts working dates from the earliest scheduled Start of the selected
Execution/Commitment projection; pre-anchor and non-working cells remain blank.
When an effective Finish-to-Start chain shares one calendar date because the
successor uses remaining same-day capacity, Home derives equal visual sequence
slots from the visible dependency chain. This prevents overlapping bars and a
backward-pointing arrow without claiming that slot width represents allocated
minutes.

The frontend must use bounded row/date rendering. A naive permanent
`visible rows × visible dates` interactive DOM matrix is not acceptable for
large portfolios or multi-year daily ranges. Grid and timeline vertical scroll,
row heights, expansion, focus, and stale-response handling must remain
synchronized. Dependency geometry is computed only for visible renderable
endpoints. Home participates in the existing schedule projection clock or an
equivalent shared version boundary; it must not introduce an independent cache
version that can disagree with Project, WBS, or Dependency gateways.

Home owns a viewport-bounded shell: the document does not scroll vertically on
this route, while the synchronized Gantt body is the single vertical scroll
owner. The timeline header remains fixed inside the workspace and non-Home pages
retain ordinary document scrolling.

Saved filters are a small global backend aggregate because authentication and
user identity are absent. Persist ID, normalized case-insensitive unique Name,
selected Project IDs, timestamps, and Version. Empty selections are valid.
Save and Delete use simple optimistic concurrency. Opening a filter intersects
stored IDs with current Open/Locked Projects; Closed/deleted IDs are silently
omitted and are removed on Save or Save As. The UI lists names only and does not
need merge or ownership semantics.

### Home Gantt query and index review target

- Portfolio reads must preserve authoritative Project priority and recursive WBS
  order without issuing one query per selected Project or Task.
- Dependency and assignee projection must be fetched or joined in bounded sets.
- Holiday range access must use date-bounded query paths.
- Saved-filter normalized Name requires measured uniqueness/index support;
  update/delete use ID plus Version point predicates.
- PostgreSQL plans must be reviewed against realistic selected-Project, WBS,
  dependency, and date-range cardinality before implementation completion.
- Do not add speculative indexes without a production query shape and plan
  evidence.

### Home Gantt implemented query/index evidence

The MVP implementation uses a bounded set-based read path: one ordered active or
selected Project query, one WBS query joined to Team Member for all selected
Project IDs, one dependency query joining both endpoint Tasks for those Project
IDs, and one date-bounded Public Holiday query. Saved-filter reads and writes are
separate point/list operations. The HTTP contract rejects more than 100 selected
Projects and more than 730 inclusive calendar days, so the repository never
expands into an unbounded Project-by-date request. No query is issued per Project,
Task, dependency, or date.

The reviewed index mapping is:

- `projects_active_priority_idx` supports active Project ordering by priority;
  the primary key resolves selected ID sets.
- `wbs_nodes_project_tree_idx (project_id, parent_key, position, id)` supports
  the selected-Project hierarchy read and deterministic recursive composition.
- `task_dependencies_blocking_task_id_idx` and
  `task_dependencies_blocked_task_id_idx` support both joined endpoint filters.
- `public_holiday_dates_date_uidx (date)` supports the bounded holiday range.
- `portfolio_saved_filters_name_unique` enforces normalized Name uniqueness,
  while the primary key plus Version predicate protects update/delete and
  `portfolio_saved_filters_order_idx` supports deterministic listing.
- `task_schedule_allocations` primary key `(task_id, timeline, allocation_date)`
  supports Task-centric allocation reads, and
  `task_schedule_allocations_member_date_idx` supports assignee/timeline/date
  capacity composition.

No additional speculative index is introduced. PostgreSQL plan execution remains
part of local validation. Run representative `EXPLAIN (ANALYZE, BUFFERS)` for the
query shapes above after loading realistic Project/WBS/dependency/date-range
cardinality. The expected success criterion is indexed restriction/order where
applicable, no per-row nested application query, and no sequential scan caused by
an absent required index on realistically selective predicates. Status:
`AUTHORED — NOT RUN — LOCAL VALIDATION REQUIRED`.
