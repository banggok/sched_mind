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

The primary product actor in these stories is the Engineering Lead.

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
restricted by references described in US-1.2.

### Daily, execution, and commitment capacity

Daily Capacity is a Member input in 0.5-hour increments. The scheduler resolves
the date-specific capacity by applying weekend/Public Holiday zero capacity,
then Capacity Override, then Member Daily Capacity. Member Buffer produces raw
Execution Capacity, which is rounded to the nearest `0.5` hour. Commitment
Capacity is calculated independently from Resolved Daily Capacity after both
Member Buffer and owning Project Buffer, then rounded to the nearest `0.5` hour. Scheduler arithmetic remains deterministic and does not round capacity
to whole days. Formula ownership belongs to US-1.2, US-2.2, US-3.3, and US-6.1.

### Capacity Override

A Capacity Override temporarily replaces one Member's Daily Capacity for an
inclusive date range. It may represent reduced availability or overtime. One
Member cannot have overlapping overrides, while adjacent ranges and equivalent
ranges for different Members are allowed. Effective-date filtering is inclusive.
Exact creation, editing, deletion, concurrency, filtering, and capacity
resolution rules belong to US-2.1.

Public Holiday has precedence over a Capacity Override and resolves daily
capacity to zero. Public Holiday and Capacity Override mutations do not trigger
immediate recalculation; the concrete scheduler consumes their latest confirmed
values on the next scheduling trigger.

### Project

A Project is the root planning entity and WBS level `0`. It has a unique
case-insensitive Name, system-assigned Priority, and an Open, Locked, or Closed
lifecycle. Locked protects Execution and Commitment baselines while Forecast
remains dynamic. Closed Projects are historical, read-only, and excluded from
scheduling and Gantt. Exact transitions, ordering, deletion, and downstream
contracts belong to US-3.1.

Project Settings use `automaticScheduling` as the sole activation toggle and
optionally define the Project-level Scheduling Start Date. This date is the
initial anchor; it is not a Task field. Automatic Scheduling without an anchor
persists the setting but leaves generated dates empty with a safe unscheduled
reason. Project Buffer affects only Commitment Capacity. Exact rules belong to
US-3.3 and US-6.1.

### WBS, dependency ownership, and scheduling

Executable Tasks own Assignee, Effort, and non-negative integer Lag. Execution
and Commitment dates are generated independently when Automatic Scheduling is
ON and are treated as retained manual values when it is OFF. Dependency endpoint
pairs are stored once and may be manual-owned, automatic-owned, or both. Removing
manual ownership never removes scheduler-required automatic ownership.

The concrete scheduler operates across the active portfolio, serializes relevant
mutations, and persists daily allocation projections plus monotonic Project
schedule versions. Dependency readiness is applied before Project Priority and
depth-first WBS order. Allocation is whole-Task, contiguous, and non-preemptive.
Locked allocations are fixed reservations; Closed Projects are excluded;
completed Tasks retain generated dates and use Actual End as successor readiness.
For an Open unfinished Task, scheduling-field blur can run the same concrete
scheduler as a rollback-only draft preview so generated dates are visible before
Save; changing Assignee recalculates dates and dependency ownership, while
clearing Assignee returns an unconfirmed missing-Assignee schedule and removes
stale automatic ownership. The preview does not advance persisted schedule state.

US-4.3 defines the approved View Group and Edit Project summary behaviour. Project is the logical WBS level `0`, so Edit Project recursively summarizes every confirmed Task across all top-level WBS roots; View Group summarizes only the selected subtree. Both contexts use one read-only calculation contract. Execution and Commitment ranges each use the earliest Start and latest End among Tasks with a complete pair, with separate scheduled-Task coverage. Effort Completion uses Actual End as the only completion source and compares completed known Effort with total known Effort; Tasks without Effort are excluded from the arithmetic and disclosed. The summary is not persisted on Group or Project and never consumes unconfirmed Task preview data.

Edit Project composes this summary after Project fields using the existing shared wide Dialog variant. Add Project remains summary-free. If the Project WBS tree is not already fresh in cache, only the summary region loads or retries; Project form draft and Save/Cancel remain independent. Existing Project `startDate`/`endDate` are not substitutes for the separate recursive timeline summaries.

Forecast coordination remains separate. Actual End and Reopen Task keep their
existing Forecast callback and do not become full Execution/Commitment triggers.
Freeze-date behavior, Delivery Impact calculation, Project Health, Gantt, and
reporting remain deferred unless an authoritative story states otherwise.

## UX terminology

The application navigation groups Role and Member workflows under the concise
visible context “Team”. Within that context, user-facing labels prefer “Roles”
and “Members” rather than redundant “Team Roles” or “Team Members”. Domain and
API names remain explicit when shortening would reduce meaning.

Capacity Overrides are accessed from a Member workflow and do not have a
standalone sidebar destination. Product-specific control behavior, including the
shared calendar usage, is authoritative in US-2.1.
