# SchedMind Product and Domain Context

SchedMind — “Smarter Planning. Better Delivery.” — is a planning and scheduling
product intended to connect engineering structure, capacity, work, and delivery
expectations. This document is a compact orientation for coding agents, not a
replacement for approved product requirements.

## Authority

Approved files under [`user_story/`](../../user_story/) are authoritative for
implemented feature behavior, API contracts, acceptance criteria, and mandatory
tests. If this summary differs from an approved user story, the user story wins.
No separate EDD or PRD is currently present in this repository; future product
documents must state their authority relative to approved stories.

Do not infer scheduling behavior that is not yet approved. Concepts mentioned as
future context are not permission to implement them.

## Current implemented scope

> **Implemented dependency revision — 2026-08-03:** US-6.4 removes Auto
> Dependency by Assignee. Dependencies are explicit manual Finish-to-Start
> prerequisites; scheduling and preview never create, delete, or project
> ownership. Migration atomically removes legacy ownership and recalculates Open
> automatic schedules. This supersedes ownership wording retained below.

> **Implemented scheduling revision — 2026-08-03:** US-6.3 Task Capacity
> Allocation Percentage and the accompanying US-6.2 Actual Allocation revision
> are implemented across WBS persistence/API/UI, concurrent planned scheduling,
> manual fixed allocation, Actual-Date-only allocation, migration, and automated
> evidence. Detailed traceability is maintained in
> [`task-capacity-allocation-implementation-evidence.md`](task-capacity-allocation-implementation-evidence.md).

> **Approved Sprint target — 2026-08-03:** US-8.1 adds standalone Sprint
> grouping under the Project navigation group. Sprint persists selected Members
> and Tasks, reads live Execution capacity/allocation, and never changes the
> scheduler, Project, WBS, Task, or capacity source data.

> **Approved Assignee recommendation target — 2026-08-05:** US-6.5 ranks
> eligible Assignees for first assignment and Edit Task using one latest
> side-effect-free batch. Automatic Scheduling ON reuses the concrete scheduler
> in rollback-only mode; OFF uses the approved advisory anchor hierarchy with
> backend today resolved from `APP_TIMEZONE`. Ranking prefers no added overcapacity, earliest Execution End,
> then largest completion-Date remaining Execution Capacity. Recommendation does
> not reserve capacity or optimize lower-priority portfolio impact.

The current product manages:

- Roles used to classify Members and Tasks;
- Members with a Role, Daily Capacity, Buffer, and half-hour-rounded Base Execution Capacity preview;
- date-bounded Capacity Overrides scoped to one Member;
- Public Holidays used as global zero-capacity dates;
- Projects with priority, Open/Locked/Closed lifecycle, and Automatic Scheduling settings;
- WBS Groups and executable Tasks with Assignee, Effort, Lag, and generated timelines;
- manual, automatic, and shared dependency ownership;
- concrete portfolio-level Execution and Commitment scheduling;
- a technical backend health indicator.

Detailed rules are owned by:

- [US-1.1 Manage Roles](../../user_story/US-1.1-manage-roles.md)
- [US-1.2 Manage Team Members](../../user_story/US-1.2-manage-team-members.md)
- [US-2.1 Manage Capacity Override](../../user_story/US-2.1-manage-capacity-override.md)
- [US-2.2 Manage Public Holiday](../../user_story/US-2.2-manage-public-holiday.md)
- [US-3.1 Create Project](../../user_story/US-3.1-create-project.md)
- [US-3.3 Configure Project Settings](../../user_story/US-3.3-configure-project-settings.md)
- [US-4.1 Manage WBS](../../user_story/US-4.1-manage-wbs.md)
- [US-4.2 Reopen Completed Task](../../user_story/US-4.2-reopen-completed-task.md)
- [US-4.3 View Group and Project Summary](../../user_story/US-4.3-view-group-summary.md)
- [US-5.1 Manage Dependency](../../user_story/US-5.1-manage-dependency.md)
- [US-6.1 Automatic Scheduling](../../user_story/US-6.1-automatic-scheduling.md)
- [US-6.2 Locked Project and Completed Task Scheduling](../../user_story/US-6.2-locked-project-and-completed-task-scheduling.md)
- [US-6.3 Task Capacity Allocation Percentage](../../user_story/US-6.3-task-capacity-allocation.md)
- [US-6.5 Recommend Assignee by Simulated Completion](../../user_story/US-6.5-recommend-assignee.md)
- [US-7.1 Home Portfolio Gantt Workspace](../../user_story/US-7.1-home-portfolio-gantt.md)
- [US-8.1 Manage Sprints](../../user_story/US-8.1-manage-sprints.md)

The primary product actor in these stories is the Engineering Lead.

## Home portfolio workspace

US-7.1 defines Home as the root frontend page and canonical active-WBS surface.
It composes selected Open/Locked Projects in a read-only daily Gantt, invokes
the shared Project, Group, Task, Add Task, Add Child, and lifecycle dialogs
directly over Home, and persists globally named Project-selection filters. The
standalone Project Structure action is not a current navigation surface.

## Sprint planning

US-8.1 defines Sprint as a standalone grouping aggregate. Its location under the
visible Project navigation group does not make it a Project or Portfolio child.
A Sprint stores date boundaries, Planned/Started metadata, selected Member IDs,
and selected Task IDs. Capacity and Task Execution allocation remain live
projections from their existing authoritative sources. Sprint suggestion must
include every eligible unfinished scheduled Task for a selected Member whose
Execution End is on or before Sprint End, even when overcapacity results, then
fill remaining in-period capacity from later Tasks deterministically. Sprint
never reserves capacity, invokes scheduling, or changes Project/WBS/Task data.
Saved Task membership survives schedule drift; invalid current conditions are
shown under Needs Review instead of being removed silently. Readable Needs Review
allocation is excluded from selected-member utilization. Sprint Planning uses the
complete canonical Execution allocation to order rows, but the visible daily
grid contains only the inclusive Sprint Date range. Positive allocation before
Sprint Start or after Sprint End creates no visible Date column or allocation
cell. The page shows Member capacity only; Task allocation remains on Task rows,
while Sprint summaries and aggregate allocation/remaining/overcapacity rows are
intentionally absent. Same-Member Sprint date ranges cannot overlap inclusively.
Exact behaviour belongs to US-8.1.

## Terminology and current invariants

### Role

A Role is uniquely named master data used to classify Members and future tasks.
Uniqueness is case-insensitive after normalization. Renaming preserves identity
and must propagate to Member projections without requiring a hard refresh. A
referenced Role cannot be deleted. See US-1.1 for exact validation and API rules.

### Member

A Member represents an engineer available to future scheduling. It references a
Role and owns base Daily Capacity and Buffer. Names are not unique. Member
updates preserve identity so future assignments remain valid. Deletion is
restricted by active executable WBS assignments in Open or Locked Projects;
Closed-only assignment history does not block deletion. See US-1.2.

### Daily, execution, and commitment capacity

Daily Capacity is a Member input in 0.5-hour increments. The scheduler resolves
the date-specific capacity by applying weekend/Public Holiday zero capacity,
then the minimum Capacity among all active overrides for that Member/Date, then
Member Daily Capacity only when no override applies. The selected override is
not final capacity: Member Buffer produces raw Execution Capacity, which is
rounded to the nearest `0.5` hour. Commitment
Capacity is calculated independently from Resolved Daily Capacity after both
Member Buffer and owning Project Buffer, then rounded to the nearest `0.5` hour. Scheduler arithmetic remains deterministic and does not round capacity
to whole days. Formula ownership belongs to US-1.2, US-2.2, US-3.3, and US-6.1.

### Capacity Override

A Capacity Override temporarily replaces one Member's Daily Capacity input for
an inclusive date range. It may represent reduced availability, sickness,
leave, or overtime/support. One Member may have multiple overlapping overrides;
the minimum active Capacity is resolved independently for each Date. Exact
duplicates with the same Member, Start Date, End Date, and Capacity are rejected
even when Description differs. Effective-date filtering is inclusive. Mutation
uses cross-project impact coordination only when the resolved per-Date minimum
changes. Exact creation, editing, deletion, concurrency, filtering, and capacity
resolution rules belong to US-2.1.

Public Holiday has precedence over a Capacity Override and resolves daily
capacity to zero. Daily Capacity, Member Buffer, Capacity Override, Public
Holiday, and Project Buffer mutations use US-6.2 cross-project impact preview.
Open-only impact requires confirmation; Locked impact blocks; confirmed allowed
changes and impacted Open schedules persist atomically.

### Project

A Project is the root planning entity and WBS level `0`. It has a unique
case-insensitive Name, system-assigned Priority, and an Open, Locked, or Closed
lifecycle. Locked protects Execution/Commitment baseline as an immutable anchor.
Project Name is the only Project field that may be changed while Locked; the
rename path does not change settings, priority, snapshots, schedule version, or
invoke scheduling. Planning, WBS structure, Task planning fields, Settings, and
dependency changes otherwise require explicit `Locked → Open` Reopen. Complete
Actual Date remains the only Task mutation allowed while Locked. It changes no
protected baseline, but Actual Allocation may recalculate impacted Open
Projects. Mutual/transitive Locked impact is resolved through atomic Reopen All
closure. Forecast behavior while Locked is deferred. Closed Projects are
historical, read-only, and excluded from scheduling and Gantt. Exact
transitions, ordering, deletion, and downstream contracts belong to US-3.1 and
US-6.2.

Project Settings use `automaticScheduling` as the sole activation toggle and
optionally define the Project-level Scheduling Start Date. This date is the
initial anchor; it is not a Task field. Automatic Scheduling without an anchor
persists the setting but leaves generated dates empty with a safe unscheduled
reason. Project Buffer affects only Commitment Capacity. Exact rules belong to
US-3.3 and US-6.1.

### WBS, dependency ownership, and scheduling

Executable Tasks own Assignee, Effort, non-negative integer Lag, and Task
Capacity Allocation Percentage. The percentage is a persisted integer `1–100`
with default `100`; it is a maximum planned allocation per Date, not a guaranteed
reservation or priority. Existing Executable WBS records must be migrated to
`100`, explicit `0` is invalid, legacy create omission becomes `100`, and update
omission preserves the existing value. Selecting, changing, or clearing
Assignee preserves the Task-level percentage according to revised US-6.3 and
US-6.5. Execution and Commitment dates are generated
independently when Automatic Scheduling is ON and are treated as retained manual
values when it is OFF. Dependency endpoint pairs are stored once and may be
manual-owned, automatic-owned, or both. Removing manual ownership never removes
scheduler-required automatic ownership.

The concrete scheduler supports shared capacity and cross-project dependency,
but each mutation recalculates only its transitive impacted scheduling scope.
Open unfinished Tasks may change; Locked Projects are immutable outputs and
unrelated Projects are not recalculated or version-updated. Dependency readiness
is applied before Project Priority and depth-first WBS order. A Task's percentage
is applied only after final Execution/Commitment capacity is independently
resolved and rounded; later ordered Tasks may use remaining capacity on the same
Date, so same-assignee planned allocations may overlap. Automatic allocation remains priority-safe. Fixed manual allocation is
immutable, but it reduces an unfinished automatic Task only when the manual
Project has higher Project Priority. A lower-priority manual allocation may
overlap a higher-priority automatic allocation without warning or later capacity
debt. Completed Actual Allocation and Locked baselines remain absolute
reservations regardless of priority.
Completed Tasks use complete Actual Date, with Actual End as readiness anchor.
Actual Allocation ignores planned percentage and uses only eligible Dates inside
Actual Start–Actual End. It competes only with other completed Actual Allocation
for the same Assignee/Date, distributes in `0.5h` balanced shares, recalculates
remaining shares after constrained Dates, backfills spare BAU capacity, then
levels unavoidable total overcapacity with a latest-Date tie-breaker. Existing
completed Actual rows remain immutable, and overcapacity never becomes debt on
later Dates.
For an Open unfinished Task, scheduling-field blur can run the same concrete
scheduler as a rollback-only draft preview so generated dates are visible before
Save; changing Assignee recalculates dates and dependency ownership, while
clearing Assignee returns an unconfirmed missing-Assignee schedule and removes
stale automatic ownership. The preview does not advance persisted schedule state.

US-6.5 adds Assignee recommendation to the same Task workflow. Role and
Assignee are the final two planning inputs so Effort, Capacity Allocation
Percentage, Lag, and manual timeline data are available first. One batch removes
the edited Task's own allocation from a shared confirmed baseline and evaluates
all Role-matching Members. Automatic ON invokes the concrete scheduler without
inventing a missing Project anchor. Automatic OFF preserves manual dates and
uses Manual Execution Start, otherwise `max(Project Scheduling Start Date,
today)`, otherwise backend today in `APP_TIMEZONE` as an advisory anchor.
Feasible candidates precede candidates that add incremental overcapacity;
within each group, earliest simulated Execution End, largest completion-Date
remaining Execution Capacity, normalized name, and ID determine order. The
recommendation is advisory, side-effect free, and does not reserve capacity or
penalize downstream lower-priority impact.

US-4.3 defines the approved View Group and Edit Project summary behaviour. Project is the logical WBS level `0`, so Edit Project recursively summarizes every confirmed Task across all top-level WBS roots; View Group summarizes only the selected subtree. Both contexts use one read-only calculation contract. Execution and Commitment ranges each use the earliest Start and latest End among Tasks with a complete pair, with separate scheduled-Task coverage. Effort Completion uses complete Actual Date as the completion source and compares completed known Effort with total known Effort; Tasks without Effort are excluded from the arithmetic and disclosed. The summary is not persisted on Group or Project and never consumes unconfirmed Task preview data.

Edit Project composes this summary after Project fields using the existing shared wide Dialog variant. Add Project remains summary-free. If the Project WBS tree is not already fresh in cache, only the summary region loads or retries; Project form draft and Save/Cancel remain independent. Existing Project `startDate`/`endDate` are not substitutes for the separate recursive timeline summaries.


Actual Allocation is a canonical daily attribution projection with two read
directions: Task-centric (which Dates/capacity an assignee spent for one Task)
and assignee-centric (which Tasks consumed one assignee's capacity on a Date).
Current scope exposes Task-centric Execution/Commitment/Actual allocation for
verification; assignee analytics UI is deferred. Historical Actual ranges may
overlap dependency ranges after every predecessor is already completed. Planned
Task percentage is displayed for verification but never caps Actual Allocation.

Forecast coordination remains separate and Locked Project Forecast behavior is deferred. Actual Date on Open Project actualizes planned dates and creates Actual Allocation; Actual Date on Locked Project preserves protected baseline while its allocation may recalculate impacted Open Projects. Reopen Task clears both Actual fields and is allowed only while Project Open. US-7.1 now owns the approved Home Portfolio Gantt target. Freeze-date behavior, Delivery Impact calculation, Project Health, historical Gantt, and reporting remain deferred unless an authoritative story states otherwise.

## UX terminology

The application navigation groups Role and Member workflows under the concise
visible context “Team”. Within that context, user-facing labels prefer “Roles”
and “Members” rather than redundant “Team Roles” or “Team Members”. Domain and
API names remain explicit when shortening would reduce meaning.

Capacity Overrides are accessed from a Member workflow and do not have a
standalone sidebar destination. Product-specific control behavior, including the
shared calendar usage, is authoritative in US-2.1.

Home is the default application page and canonical active-WBS workspace. Its
left Project Grid uses the existing Project/Task/Group terminology, keeps Name
as the primary edit action, and reveals creation, lifecycle, reorder, Move to,
and Delete icons inside the Name cell on hover or keyboard focus. There is no
Actions column. Project and Group Role cells are blank; only Task rows display
their direct Role. Start and End use `D Mon YYYY`. The right Gantt header has
working-day, grouped month/year, and calendar-date rows, while timeline bars and
all non-Name row space remain read-only. Shared dialogs open directly over Home
without routing through Projects or a Project Structure background. Saved
Project filters are global backend data and are presented by name because the
product has no login or user identity in current scope. Exact Home behaviour
belongs to US-7.1.

Sprints are accessed under the visible Project navigation group beside Projects.
This menu grouping is navigational only. Sprint List, the two-step modal
Details/Members form, and the main-page Sprint Planning workspace belong to US-8.1;
Home remains the root page. Sprint Planning groups by selected Member and renders
flat Task rows across Projects in canonical daily allocation order. Project
Name/status remains visible on every row; WBS path/rank remains an internal
ordering input and is not displayed. Each Member shows period and per-Date
capacity only. A Date with `0h` Member Daily Capacity marks that Member's whole
Date column and carries one textual `No capacity` state in the header; Remaining
Capacity is not used for this marker. Task allocation stays visible per Task for
Sprint Dates only, every Sprint Date shares one horizontal grid, and
Sprint/remaining/overcapacity summaries are not rendered.
Task Name opens the same shared Edit Task controller used by Home while the
active page remains Sprints. Closing the Task dialog leaves Sprint Planning
unchanged; saving the Task regenerates the local suggestion and returns to the
same Sprint Planning workspace. The regenerated Task membership is not persisted
until Save Sprint Planning.
Suggestion projection failure is recoverable through the stable
`SPRINT_SUGGESTION_UNAVAILABLE` API contract and must not persist partial Sprint
state.
