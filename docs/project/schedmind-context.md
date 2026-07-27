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

- Roles used to classify Members and future work;
- Members with a Role, Daily Capacity, Buffer, and derived Commitment Capacity;
- date-bounded Capacity Overrides scoped to one Member;
- Public Holidays used as global zero-capacity dates;
- Projects with priority and Open, Locked, or Closed lifecycle;
- Project-specific Automatic Scheduling and Project Buffer settings;
- a technical backend health indicator.

Detailed rules are owned by:

- [US-1.1 Manage Roles](../../user_story/US-1.1-manage-roles.md)
- [US-1.2 Manage Team Members](../../user_story/US-1.2-manage-team-members.md)
- [US-2.1 Manage Capacity Override](../../user_story/US-2.1-manage-capacity-override.md)
- [US-2.2 Manage Public Holiday](../../user_story/US-2.2-manage-public-holiday.md)
- [US-3.1 Create Project](../../user_story/US-3.1-create-project.md)
- [US-3.3 Configure Project Settings](../../user_story/US-3.3-configure-project-settings.md)

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

Daily Capacity is base availability in hours. Commitment Capacity is a derived
planning value after Buffer and project-defined half-hour rounding. These are
not interchangeable. Exact bounds, defaults, increment rules, and calculation
belong to US-1.2.

### Capacity Override

A Capacity Override temporarily replaces one Member's Daily Capacity for an
inclusive date range. It may represent reduced availability or overtime. One
Member cannot have overlapping overrides, while adjacent ranges and equivalent
ranges for different Members are allowed. Effective-date filtering is inclusive.
Exact creation, editing, deletion, concurrency, filtering, and capacity
resolution rules belong to US-2.1.

Public Holiday has precedence over a Capacity Override and resolves daily
capacity to zero. Public Holiday management is implemented by US-2.2. The
scheduling engine consumes resolved capacity; its broader algorithm remains
future scope.

### Project

A Project is the root planning entity and WBS level `0`. It has a unique
case-insensitive Name, system-assigned Priority, and an Open, Locked, or Closed
lifecycle. Locked protects Execution and Commitment baselines while Forecast
remains dynamic. Closed Projects are historical, read-only, and excluded from
scheduling and Gantt. Exact transitions, ordering, deletion, and downstream
contracts belong to US-3.1.

Project Settings control whether future Execution and Commitment scheduling is
automatic, retain a Project Buffer percentage, and optionally define the
Project-level Scheduling Start Date. This date is the sole initial anchor for
future automatic schedules; it is not a task field. Automatic Scheduling
without an anchor must not invent timeline dates and must warn the user. Only Open Projects are
editable. The integration contract exists, but concrete timeline recalculation
remains deferred to Epic 6; a successful settings update must not be interpreted
as proof that timelines were recalculated. Exact rules belong to US-3.3.

## Future scheduling context

Product discussions anticipate WBS tasks, dependencies, assignees, effort,
buffers, and forecast, execution, commitment, and actual dates.
They may ultimately drive Delivery Impact and health status. The repository does
not yet contain approved authoritative rules for those concepts.

In particular, do not invent:

- WBS hierarchy or executable-leaf semantics beyond explicit references in an
  approved story;
- same-assignee scheduling algorithms beyond the Project Priority trigger
  contract approved in US-3.1;
- freeze-date behavior;
- forecast, execution, commitment, or actual-date calculations;
- project-buffer allocation;
- delivery-impact or product health formulas.

Add those rules only through approved product documentation and update this
context summary without duplicating its full acceptance criteria.

## UX terminology

The application navigation groups Role and Member workflows under the concise
visible context “Team”. Within that context, user-facing labels prefer “Roles”
and “Members” rather than redundant “Team Roles” or “Team Members”. Domain and
API names remain explicit when shortening would reduce meaning.

Capacity Overrides are accessed from a Member workflow and do not have a
standalone sidebar destination. Product-specific control behavior, including the
shared calendar usage, is authoritative in US-2.1.
