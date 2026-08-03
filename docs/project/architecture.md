# Current Project Architecture

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

Member deletion soft-deletes the Member and all owned Capacity Overrides in one
transaction. Direct Capacity Override deletion remains a hard delete. Default
GORM scopes exclude deleted records; historical readers must opt in explicitly.
Active Member name search uses a PostgreSQL partial prefix index; names remain
non-unique and can also be reused after deletion. `executable_leaves` remains a
provisional Member-assignment projection and has no Project/WBS ownership
columns. US-3.1 introduces the Project root lifecycle without retrofitting that
provisional table because the WBS model is explicitly out of scope. The future
WBS implementation must replace or extend the projection so Member assignment
checks can distinguish active Project work from Closed history.

## Backend API conventions

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
Only Open Projects may
change settings; Locked and Closed settings are read-only. Re-enabling
Automatic Scheduling invokes the project-scoped `RecalculateProjectSchedule`
application port inside the settings transaction only when Scheduling Start
Date exists, so dependency failure rolls back the settings update. Without the
anchor, the setting is retained but scheduling is not invoked. Disabling
preserves existing timeline data.

Scheduling Start Date is the only Project scheduling anchor. Automatic
Scheduling may be configured without it, but the scheduler never invents Task
dates; it clears generated dates and stores an explicit unscheduled reason.
Executable WBS contains no Earliest Start, Start Constraint, or Task Anchor.
Lag belongs to the Task and is applied after the Project/dependency readiness
anchor while zero-capacity dates are skipped.

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

The scheduler loads the transitive impacted scheduling scope, validates unique Priority and WBS ordering, resolves manual and retained automatic dependency edges, and allocates Execution and Commitment independently. Open unfinished Tasks are mutable outputs; Locked Projects are immutable anchors; unrelated Projects are not recalculated or version-updated. Capacity arithmetic uses
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
allocation never exceeds final timeline capacity. Fixed manual allocation may
represent planned overcapacity and is honoured cross-Project.

US-6.2 defines Actual Date, Actual Allocation, Locked Project, and generic cross-project impact coordination. Completion persists required Actual Start/Actual End together. Open completion moves each planned Start earlier only when Actual Start is earlier and always sets each End to Actual End. Actual Allocation ignores planned Task percentage and uses only eligible Dates inside Actual Start–Actual End. It competes only with other completed Actual Allocation for the same Assignee/Date, distributes Effort in `0.5`-hour balanced shares, recalculates remaining shares after constrained Dates, backfills spare BAU capacity, and then levels unavoidable total historical overcapacity with a latest-Date tie-breaker. Existing completed Actual rows remain immutable; planned automatic/manual/Locked rows are ignored for Actual head-to-head construction. Historical overcapacity never carries debt to later Dates. Locked Projects remain immutable scheduler outputs, but Actual Date may be saved and its Actual Allocation may recalculate transitively impacted Open Projects. Ordinary Task/dependency/priority/capacity mutations use grouped impact preview, confirmation, server revalidation, and Locked-impact blocking. Project Reopen expands an atomic transitive Locked closure to avoid mutual-lock dead ends.

Execution, Commitment, and Actual daily allocations are separate projections. Actual Allocation rows are canonical and support both Task-centric and assignee-centric queries; current UI exposes the Task-centric read-only verification section while assignee analytics remains deferred. Impact simulation is version-bound: preview returns grouped Project names and a confirmation token, confirm re-simulates under scheduling locks, and stale impact never persists.

`task_schedule_allocations` stores exact decimal allocation and remaining
capacity by Task, timeline, assignee, and date. Its primary key prevents duplicate
Task/date projection. `projects.schedule_version` is updated optimistically.
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
workflow rather than as standalone navigation.

The WBS domain remains the only work-item model. Its frontend presents an
Executable WBS as **Task** and a Grouping WBS as **Group**; these labels are
derived from child existence and never create or persist a separate type field.
Home is the canonical active-WBS surface. `App.tsx` keeps Home mounted while it
opens the shared Project controller or a direct-entry WBS dialog for Group,
Task, Add Task, Add Child, and Move to. The standalone Project Structure panel
has been removed; the WBS controller accepts only a direct Home action and
renders the corresponding shared dialog. Direct-entry dialogs fetch the
authoritative Project tree, so Move to destinations are not narrowed by the
current Home Role projection.

The Home portfolio is rendered as a synchronized split grid/timeline. WBS,
Name, Role, Assignee, Effort, Start, and End remain individually resizable data
columns. Eligible quick and overflow actions are positioned inside Name and are
revealed on hover or keyboard focus; no separate Actions column exists. Name is
the only row edit activation. Project and Group Role cells are blank, while
Task Role remains direct data. Start and End display timezone-stable
`D Mon YYYY` values. The timeline header uses three rows for working-day
sequence, grouped month/year, and calendar date. Timeline bars and other cells
are read-only.
Structural actions reuse the WBS gateway commands, and Project lifecycle
actions reuse the same `ProjectsPage` controller used by Projects. Confirmed
mutations advance the shared schedule projection clock or explicitly reload the
portfolio projection while retaining selected filters and expansion IDs.

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
`0` is rejected. Changing or clearing Assignee resets the value to `100` unless
the same new-Assignee command explicitly supplies another valid value.

Execution and Commitment Task Daily Limits are calculated after each final
timeline capacity has been independently rounded. The percentage is a maximum,
not a reservation or priority. Automatic rows are ordered by Project Priority
and visual WBS order and share residual same-Date capacity. Manual timeline rows
are fixed reservations and may persist planned overcapacity. The field moves
with executable data during WBS conversion; Grouping WBS never owns it.

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
preview is version-bound and returns affected Project names grouped by Open and
Locked. Open-only impact requires confirmation. Ordinary mutation with Locked
impact is rejected. Actual Date is the factual-data exception: after grouped
confirmation it persists while Locked Projects remain immutable and impacted
Open Projects recalculate. Reopen Task is not an exception and remains blocked
by impacted Locked Projects.

Project Reopen calculates a transitive Required Locked Reopen Closure. Mutual
A/B or transitive A/B/C lock dependencies are presented as one `Reopen All`
operation. Closure status changes, unfinished scheduling, dependency
reconciliation, allocation persistence, and version changes are atomic. Stale
preview, optimistic conflict, or persistence failure leaves all statuses and
projections unchanged.

### Query and index review target

- Completion/Reopen point mutation uses primary-key plus Project scope and a
  pair-state predicate; cardinality remains at most one Task.
- Actual Allocation lookup requires Task/date and assignee/date access paths so
  both read models avoid full scans.
- Impact preview/confirm uses schedule/status versions rather than trusting a
  client-provided Project list.
- Allocation writes and affected schedule rows share one transaction boundary.
- Exact index design must follow measured PostgreSQL query shapes before
  implementation; this requirement does not authorize speculative indexes.

## Home Portfolio Gantt target contract

US-7.1 adds a Home feature boundary as the default frontend composition. Home is
a portfolio read workspace, not a new WBS or scheduler aggregate. The left grid
renders selected active Project/WBS rows, while the right timeline renders
read-only daily Execution or Commitment bars and effective dependency arrows.
The standalone Project Structure entry point is removed. Add Task, Add Child,
Project edit, and Task/Group edit must reuse existing application use cases and
dialogs directly over Home so the canonical workspace does not fork validation,
impact preview, transaction, or rollback behaviour.

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
