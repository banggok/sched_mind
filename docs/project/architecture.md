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

Capacity Override writes lock the parent Member row before checking overlap and
persisting within a transaction. Parent-scoped list and inclusive effective-date
queries use the `(team_member_id, start_date, end_date, id)` index. The filter is
applied before count, limit, and offset.

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
has precedence over Capacity Override and Daily Capacity, but holiday mutations
do not trigger schedule recalculation.

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

The scheduler loads all Open and Locked Projects, validates unique Priority and
WBS ordering, resolves manual and retained automatic dependency edges, and
allocates Execution and Commitment independently. Capacity arithmetic uses
`math/big.Rat`: weekend/Public Holiday resolves to zero, otherwise Capacity
Override replaces Member Daily Capacity, Member Buffer produces raw Execution
Capacity and rounds it to the nearest `0.5` hour. Commitment Capacity is
calculated independently after both Member Buffer and Project Buffer, then
rounded to the nearest `0.5` hour. Allocation is
inclusive-date, whole-Task, contiguous, and non-preemptive. Same-assignee
successors may consume remaining capacity on the predecessor End Date;
different-assignee successors begin on the next positive-capacity date.

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

The WBS domain remains the only work-item model. Its frontend presents the
feature as **Project Structure**, an Executable WBS as **Task**, and a Grouping
WBS as **Group**. These labels are derived from child existence and never create
or persist a separate type field.

The Dependency feature stores one directed Finish-to-Start relation between
two executable WBS Tasks. Relations may cross active Projects, but Groups and
Tasks belonging to Closed Projects cannot become new endpoints. Completed
Tasks may be blockers; completed Tasks cannot become newly blocked, and an
existing relation whose blocked Task is completed is historical read-only.
Dependency editing is composed into the contextual Edit Task dialog in Project
Structure. One endpoint pair is persisted once with `manual_owned` and
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
Effort, and Lag blur events may request
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

Project Name and settings share the same Add/Edit dialog and persist through one
atomic create or update operation; there is no separate Settings action. Status
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

## Reopen completed Task

`US-4.2` keeps Task as an Executable WBS and adds one explicit exception to the
normal completed-task read-only invariant. The dedicated command is
`POST /api/projects/{projectId}/wbs/{wbsId}/reopen`; generic executable update
continues to reject completed Tasks and cannot clear Actual End.

The WBS repository executes Reopen in one GORM transaction. It reads the scoped
Task snapshot, locks the owning Project, rejects Closed Projects and non-leaf or
unfinished WBS state, then conditionally updates by `id`, `project_id`, and
`actual_end IS NOT NULL`. A request that initially sees unfinished state returns
`TASK_NOT_COMPLETED`; a request that saw completed state but loses the
conditional transition returns `TASK_REOPEN_CONFLICT`. Exactly one successful
completed-to-unfinished transition invokes `RecalculateProjectForecast` inside
the transaction. Forecast failure rolls back Actual End and Updated At. Reopen
never invokes full schedule recalculation and never updates Project status,
Execution or Commitment timelines, or locked baselines.

Dependency rows are not written or revalidated because Task identity and graph
endpoints do not change. Dependency projections continue to derive completed
state from the current persisted Actual End. The frontend invalidates the
project WBS tree and all dependency-detail caches after confirmed success;
versioned request caches prevent older in-flight WBS or dependency responses
from becoming cached confirmed state. Current Task detail is replaced directly
from the confirmed response, so Project Structure updates without a browser hard
reload. Concrete Forecast, Delivery Impact, Health, and Gantt frontend stores do
not yet exist; their concrete invalidation remains deferred while the established
Forecast coordination contract is preserved.

### Reopen query review

The transition query shape is a primary-key lookup scoped by Project followed by
a conditional update with `id = ? AND project_id = ? AND actual_end IS NOT
NULL`. PostgreSQL serialises competing commands through the owning Project row
lock and the conditional Task update. SQLite automated contract tests rely on
the same conditional predicate; SQLite lock/busy errors during the competing
transition are mapped to the deterministic conflict error. The primary-key
predicate limits cardinality to at most one Task, and the Project lookup also
uses its primary key. No migration or new index is justified: an additional
Actual End index would add write cost without improving a primary-key point
mutation.
