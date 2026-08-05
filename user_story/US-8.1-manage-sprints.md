# US-8.1 — Manage Sprints

> **Authority:** This story owns the standalone Sprint aggregate, Sprint list and
> Details/Members form and page-level Sprint Planning workspace, Sprint Member and
> Sprint Task membership, live Execution capacity/allocation projection, daily
> working-plan ordering, per-Date capacity/allocation/variance presentation,
> deterministic Task suggestion, overlap protection, Planned/Started metadata,
> drift warnings, and Sprint-specific persistence/API contracts. It does not own
> or change Project, WBS, scheduler, dependency, Actual Date, Actual Allocation,
> or capacity mutation behaviour.
>
> A Sprint is placed under the visible **Project** navigation group for product
> discoverability only. It is not owned by a Project or Portfolio and must not be
> persisted as a child of either aggregate.

## 1. User Story

**As an** Engineering Lead

**I want** to group scheduled Tasks and selected Members into a Sprint and
present their canonical daily working plan

**So that** the team can execute Tasks in planned allocation-date order and
confirm daily capacity, remaining capacity, and overcapacity without changing
the authoritative schedule.

---

## 2. Business Context

SchedMind already owns authoritative Project/WBS planning, per-date Member
capacity, and canonical Execution allocation. Sprint management must compose
those existing facts instead of introducing another scheduler, reservation
engine, or Project hierarchy.

A Sprint is therefore a standalone planning group with explicit Member and Task
membership. It suggests Tasks from the current Execution schedule, allows manual
consolidation, and always displays current live schedule data. Creating, editing,
starting, or deleting a Sprint must never move a Task, reserve capacity, change a
Project, invoke schedule impact confirmation, or advance any schedule version.

The highest-priority suggestion rule is date coverage: every eligible unfinished
Task assigned to a selected Member whose Execution End is on or before the
Sprint End must be selected, even when that immediately produces overcapacity.
Remaining Sprint capacity is then filled on a best-effort basis from later
scheduled Tasks. Capacity is informative; it is not a Save constraint.

Sprint Planning is also an execution-facing daily working plan. Its row order must be
driven by the complete canonical Execution allocation projection, not by Project
or WBS grouping. The earliest Date with positive allocation determines which
Task is shown first. Project remains visible context; WBS remains an internal
deterministic same-Date tie-breaker. Sprint never changes either ordering source.

Daily capacity calculation must remain lossless. The read model preserves
capacity, selected allocation, remaining capacity, and overcapacity per selected
Member and Sprint Date without cross-Member or cross-Date netting. Sprint Planning
renders only Member capacity and individual Task allocation cells; the other
calculated summaries are intentionally hidden.

---

## 3. Scope

### 3.1 In Scope

- Add **Sprints** under the existing visible **Project** navigation group.
- Keep Home as the default/root application page.
- List standalone Sprints with Name, Start Date, End Date, Status, and Actions.
- Create and edit Sprint metadata through a two-step modal form:
  1. Sprint Details;
  2. Members.
- Manage Sprint Planning on the main Sprint page after the Sprint exists.
- Select one or more active Members.
- Calculate each selected Member's live Sprint Execution Capacity per Date.
- Suggest scheduled unfinished Tasks deterministically from canonical Execution
  dates and Execution allocation.
- Always include every eligible Task with `Execution End <= Sprint End`, even
  when Member or Sprint capacity is exceeded.
- Fill remaining in-Sprint capacity with later Tasks on a best-effort basis.
- Display selected Tasks grouped by current Assignee and ordered across Projects
  by their current canonical daily Execution allocation.
- Keep Project Name/status visible as Task-row context. WBS path/rank remains an
  internal deterministic ordering input and is not rendered in Sprint Planning.
- Display current canonical Execution allocation only for Dates inside the
  inclusive Sprint Period. Allocation before Sprint Start or after Sprint End
  does not create a Sprint Planning column and is not rendered in the grid.
- Display Member capacity only: one period capacity value and one per-Date
  capacity row for each selected Member. Task allocation remains visible in Task
  rows; Sprint daily summary, aggregate allocation, remaining-capacity, and
  overcapacity rows are not rendered.
- Render every Sprint Date in one horizontally scrollable grid without
  Previous/Next Date controls.
- Add or remove Tasks manually in the page-level Sprint Planning after Sprint
  creation and before or after saving a reviewed Task selection.
- Open the shared Home Edit Task dialog from a Sprint Planning Task Name without
  changing the active page. Closing the dialog performs no Sprint action;
  successful Task Save regenerates the local Sprint suggestion from the updated
  authoritative Task data.
- Persist Sprint Member and Sprint Task membership explicitly.
- Preserve Task membership when live Task data drifts and surface actionable
  warnings under `Needs Review`.
- Save new Sprints as `Planned`.
- Start a Planned Sprint through a one-way `Planned -> Started` action.
- Keep Planned and Started behaviour otherwise identical.
- Delete Planned or Started Sprints after confirmation.
- Reject inclusive date overlap when another Planned or Started Sprint shares at
  least one selected Member.
- Provide backend validation, atomic persistence, optimistic concurrency,
  deterministic ordering, loading/error/empty states, accessibility, and
  Three-Level Confidence evidence.

### 3.2 Out of Scope

- Owning a Sprint by Project, Portfolio, user, team, or organization.
- Moving, splitting, rescheduling, prioritizing, or otherwise mutating Tasks.
- Reserving Member capacity or reducing capacity available to another Sprint,
  Project, or scheduler operation.
- Creating a second Execution allocation projection.
- Commitment, Forecast, Actual Allocation, Remaining Effort, velocity, points,
  burndown, burnup, sprint goal, backlog rank, story points, or ceremonies.
- Completed/Closed/Cancelled Sprint statuses.
- Reverting Started to Planned.
- Different edit, delete, Task, capacity, or allocation behaviour by Sprint
  status.
- Adding unscheduled Tasks.
- Automatically adding a newly eligible Task to a saved Sprint.
- Automatically removing a Task because Assignee, dates, Project status, or
  allocation changed.
- Automatically adding a non-Sprint Member after Task reassignment.
- Treating overcapacity as a validation error, warning confirmation, scheduler
  debt, or future-date compensation.
- Snapshotting capacity or allocation at Save/Start time.
- Historical reconstruction of the Sprint as it appeared at Save or Start.
- Inferring an exact hour-by-hour or intra-Day execution sequence from daily
  allocation rows. Tasks sharing the same first positive allocation Date are
  same-Date work; their row tie-breaker is deterministic, not a new schedule.
- Reordering Project Priority, WBS order, dependencies, or scheduler priority to
  match Sprint presentation.
- Treating Task allocation outside current Sprint Task membership as selected
  Sprint allocation. This story confirms the reviewed Sprint plan, not every
  Task in the Member's portfolio schedule.
- Permissions, personal Sprint ownership, export, notifications, or reporting.

---

## 4. Terminology

| Term                       | Definition                                                                                                             |
| -------------------------- | ---------------------------------------------------------------------------------------------------------------------- |
| Sprint                     | Standalone grouping aggregate containing date boundaries, Status, selected Member IDs, and selected Task IDs.          |
| Sprint Member              | Active Member explicitly selected for one Sprint.                                                                      |
| Sprint Task                | Executable WBS explicitly associated with one Sprint.                                                                  |
| Sprint Period              | Inclusive calendar range from Sprint Start through Sprint End.                                                         |
| Sprint Execution Capacity  | Sum of one selected Member's resolved Execution Capacity for every Date in the Sprint Period.                          |
| In-Sprint Allocation       | Sum of current canonical Task Execution allocation whose allocation Date is inside the inclusive Sprint Period.        |
| Total Execution Allocation | Sum of all current canonical Execution allocation rows for a selected Task, including Dates outside the Sprint Period. |
| Outside-Sprint Allocation  | Total Execution Allocation minus In-Sprint Allocation.                                                                 |
| Mandatory Candidate        | Eligible Task whose Execution End is on or before Sprint End; it must be selected regardless of capacity.              |
| Fill Candidate             | Remaining eligible Task with positive In-Sprint Allocation considered while unused capacity remains.                   |
| Suggested Task             | Task selected by deterministic Sprint suggestion before manual review.                                                 |
| Needs Review               | Retained Sprint Task whose live state no longer matches selected-member or scheduling conditions.                      |
| Schedule Drift             | Live Task/Member/Project/allocation change after Sprint Task membership was persisted.                                 |
| Daily Plan Order Date      | Earliest Date with positive canonical Execution Allocation across a Task's complete readable allocation projection.     |
| Daily Selected Allocation  | Sum of current canonical Execution Allocation on one Sprint Date for selected Sprint Tasks currently grouped to one selected Member. |
| Daily Remaining Capacity   | Positive difference between one selected Member's capacity and selected allocation on one Sprint Date.                  |
| Daily Overcapacity         | Positive difference between one selected Member's selected allocation and capacity on one Sprint Date.                  |
| Planned                    | Initial Sprint status after successful create.                                                                         |
| Started                    | One-way metadata status reached through Start; behaviour otherwise equals Planned.                                     |

---

## 5. Aggregate Ownership and Non-Interference

### 5.1 Standalone Ownership

- Sprint has its own identity and lifecycle.
- Sprint has no `project_id`, `portfolio_id`, parent WBS ID, or owning Project.
- Sprint may contain Tasks from multiple Open or Locked Projects.
- Menu placement under **Project** is presentation grouping only.
- Project Priority and WBS order are read only as deterministic suggestion and
  same-Date daily-plan tie-breakers; Sprint does not modify either value.

### 5.2 Scheduler Isolation

No Sprint command may:

- invoke automatic scheduling or scheduling preview;
- create, delete, or retarget a dependency;
- write Execution/Commitment dates or allocations;
- modify Task Assignee, Effort, Lag, Capacity Allocation Percentage, Actual Date,
  Project status, Project Priority, capacity input, override, holiday, or buffer;
- acquire scheduling-impact confirmation;
- advance Project schedule version;
- create a fixed allocation, reservation, or capacity debt.

Sprint calculations consume current canonical read models. They are not inputs
to US-6.1, US-6.2, or US-6.3.

---

## 6. Sprint Data Contract

A Sprint persists at least:

| Field           | Type                 | Required | Ownership      | Rule                             |
| --------------- | -------------------- | -------- | -------------- | -------------------------------- |
| ID              | Stable identifier    | Yes      | System         | Immutable                        |
| Name            | String               | Yes      | User           | Trimmed; `1..200` characters     |
| Normalized Name | String               | Yes      | System         | Case-insensitive uniqueness      |
| Start Date      | Date-only            | Yes      | User           | Inclusive                        |
| End Date        | Date-only            | Yes      | User           | Inclusive; not before Start Date |
| Status          | `planned \| started` | Yes      | System/command | Create defaults to `planned`     |
| Version         | Positive integer     | Yes      | System         | Optimistic concurrency           |
| Created At      | Timestamp            | Yes      | System         | Audit                            |
| Updated At      | Timestamp            | Yes      | System         | Audit                            |
| Started At      | Nullable timestamp   | No       | System         | Set once on successful Start     |

Relations:

- `Sprint Member`: unique `(sprint_id, member_id)`.
- `Sprint Task`: unique `(sprint_id, task_id)`.
- Relation ordering must not be used as business priority.
- Allocation rows and capacity totals are not copied into Sprint persistence.
- Task Name, Member Name, Project Name, dates, status, and allocation are live
  projections and are not authoritative snapshots in Sprint storage.

### 6.1 Name Rules

- Trim leading and trailing whitespace before validation.
- Empty or whitespace-only Name is invalid.
- Name uniqueness is case-insensitive after repository-standard normalization.
- A deleted Sprint releases its Name for reuse.
- Concurrent creates/renames to the same normalized Name must yield one winner;
  the loser receives a stable conflict error.

### 6.2 Date Rules

- Start and End are date-only values and cannot shift through timezone mapping.
- `End Date < Start Date` is invalid.
- A one-day Sprint is valid.
- Weekend and Public Holiday may be Start or End boundaries.
- All overlap comparisons are inclusive.

### 6.3 Member Rules

- At least one active Member is required at Save.
- Duplicate Member IDs in one command are normalized or rejected before write;
  persistence remains unique.
- A Member with zero resolved Sprint capacity is valid.
- A Member with no eligible Task is valid and remains visible in Sprint Planning.
- A soft-deleted/inactive Member cannot be newly selected.
- Sprint membership alone does not block Member deletion under US-1.2. When a
  Member is deleted through an otherwise valid US-1.2 command, its Sprint Member
  relations are removed atomically; retained Sprint Tasks then follow the
  `Needs Review` rules.

---

## 7. Navigation and Sprint List

### 7.1 Navigation

The visible application navigation contains:

```text
Home
Project
├── Projects
└── Sprints
```

- Home remains the first visible item and default/root route.
- Sprints opens the Sprint List directly.
- The Sprint route is independent from Projects and does not require selecting a
  Project.

### 7.2 Sprint List Columns

Columns appear in this order:

1. Sprint Name
2. Start Date
3. End Date
4. Status
5. Actions

Rules:

- Date display uses the shared date-only format `D Mon YYYY`.
- Status labels are `Planned` and `Started`.
- The list uses the shared design-system surface, spacing, typography, action,
  and status semantics rather than browser-default table presentation.
- Status remains visible as text and does not rely on colour alone.
- On narrow viewports, the complete five-column list remains keyboard-scrollable
  inside a visibly focusable named region without moving the application shell.
- Sprint Name opens the Sprint form in edit/review mode.
- Planned row actions include `Start` and `Delete`.
- Started row actions include `Delete`.
- Edit remains available by opening the Name for both statuses.
- Start is not shown/enabled for Started.
- Delete requires shared destructive confirmation.

### 7.3 List Ordering and States

Default deterministic order:

1. Start Date descending;
2. End Date descending;
3. normalized Name ascending;
4. ID ascending.

The page provides:

- loading state;
- empty state with `Create New Sprint`;
- recoverable list error and retry;
- disabled duplicate submission while create/start/delete is pending;
- pagination or bounded list handling consistent with project conventions when
  volume requires it.

---

## 8. Create and Edit Stepper

The form is a wide dialog or dedicated page with three ordered steps. Valid
input is preserved when navigation or a recoverable backend error occurs.

### 8.1 Step 1 — Sprint Details

Fields:

- Sprint Name;
- Start Date;
- End Date.

The user cannot continue until client-side required/date validation passes.
Backend validation remains authoritative.

### 8.2 Step 2 — Members

- Show active Members with unambiguous Name and Role context.
- Support selecting and deselecting multiple Members.
- At least one Member is required before Sprint Planning or Save.
- Display overlap validation when the proposed date range shares one or more
  selected Members with another Sprint.
- A stale concurrent overlap is still rejected by the backend at Save.

### 8.3 Main-page Sprint Planning

After Create saves Details and Members, the main Sprint page opens the saved
Sprint's Sprint Planning workspace. Task management is not rendered inside the
Create/Edit modal. The user may generate one deterministic suggestion from the
saved Sprint details/members.

Sprint Planning provides:

- Member groups;
- within each Member, flat daily working-plan Task rows ordered by Section 12.3
  across Project boundaries;
- Project Name/status on every Task row; WBS ordering remains internal and is not
  displayed;
- one per-Date Capacity row and one period Capacity value for each Member;
- Task allocation rows, with no aggregate allocation, remaining, overcapacity, or
  Sprint-level daily summary rows;
- `Add Task`;
- `Remove` per Task;
- clickable Task Name that opens the same Edit Task dialog used by Home;
- `Regenerate Suggestion` when the user explicitly wants to replace the current
  reviewed membership with a new deterministic suggestion;
- `Save Sprint Planning` and `Close Sprint Planning`.

Task edit rules:

- opening Edit Task keeps the user on the Sprint page and leaves Sprint Planning
  mounted behind the shared dialog;
- closing Edit Task without Save performs no Sprint action and returns to the
  unchanged Sprint Planning state;
- successful Task Save closes only the Task dialog and regenerates the local
  Sprint suggestion from the saved Sprint Dates/Members and latest Task/schedule
  projection;
- the regenerated selection remains local until `Save Sprint Planning`;
- the Task editor must reuse the same WBS Task component and gateways as Home;
  Sprint must not fork or duplicate Task-edit behavior.

Regenerate rules:

- it uses the current saved Sprint Details and Members;
- it replaces the current Task selection, including manual additions/removals;
- it requires confirmation once the user has manually changed Task membership;
- it does not persist until Save;
- it does not invoke the scheduler.

Changing Details or Members preserves existing Task relations and returns the
user to the page-level Sprint Planning, where the user may explicitly regenerate.
The UI must not silently replace manually consolidated Tasks.

### 8.4 Edit Behaviour

- Planned and Started use the same editable fields and Task actions.
- Existing Sprint Member and Task memberships load first.
- Opening an existing Sprint does not automatically regenerate membership.
- Current live capacity, dates, assignee, Project status, and allocation are
  projected immediately.
- Saving Details, Members, or Task membership increments Sprint Version only;
  schedule versions remain unchanged.

---

## 9. Sprint Execution Capacity

### 9.1 Per-Date Resolution

For each selected Member and each Date in the inclusive Sprint Period, reuse the
Execution-capacity rules owned by US-1.2, US-2.1, US-2.2, and US-6.1:

1. Saturday, Sunday, or Public Holiday resolves to `0`.
2. Otherwise choose the minimum active Capacity Override for that Member/Date.
3. If no override applies, use Member Daily Capacity.
4. Apply Member Buffer.
5. Round once to the nearest `0.5` hour using the existing deterministic rule.

Project Buffer is not applied because Sprint uses Execution capacity.

```text
Member Sprint Execution Capacity
= sum of resolved Member Execution Capacity for every Sprint Date
```

### 9.2 Capacity Calculation and Display

For every selected Member and every Date inside the Sprint Period, calculate:

- Daily Sprint Execution Capacity;
- Daily Selected Allocation from readable selected Tasks currently assigned to
  that Member;
- Daily Remaining Capacity;
- Daily Overcapacity.

```text
Daily Remaining Capacity
= max(0, Member Daily Capacity - Member Daily Selected Allocation)

Daily Overcapacity
= max(0, Member Daily Selected Allocation - Member Daily Capacity)
```

The read model may retain these values for suggestion, validation, and coherent
projection purposes. Sprint Planning deliberately renders only:

- Member Sprint Execution Capacity for the period;
- one Daily Capacity row per selected Member;
- individual Task allocation cells.

Sprint Planning does not render Sprint daily summaries, aggregate selected
allocation, Remaining Capacity, or Overcapacity rows. A Date whose resolved
Member Daily Capacity is `0h` must mark the complete Date column for that Member,
including the Date header, Capacity cell, and every Task allocation cell. The
column carries one visible and textual `No capacity` state in its header so a
holiday, leave, or other zero-capacity Date is distinguishable without relying
on color alone. This state is derived only from Member Daily Capacity and never
from Remaining Capacity.

Calculation rules remain:

- Never net one Member's unused capacity against another Member's overcapacity.
- Never net one Date's unused capacity against another Date's overcapacity.
- A Date with `0h` capacity and positive allocation has full calculated Daily
  Overcapacity, even though Sprint Planning shows the Task allocation cell and the
  `No capacity` marker rather than an Overcapacity summary row.
- Daily Selected Allocation includes only current Sprint Task membership.
- Needs Review allocation whose current Assignee is not a selected Member is
  excluded from selected-member utilization.
- Capacity/variance and Sprint Planning columns are required for Sprint Dates only.
  Outside-Sprint allocation remains available to canonical ordering/read-model
  logic but is ignored by the visible daily grid.
- Capacity is computed live whenever the Sprint is read.
- Capacity mutation outside Sprint may change these values without changing
  Sprint Version or membership.
- Zero capacity and overcapacity do not block Save and require no confirmation.

---

## 10. Task Eligibility

### 10.1 Eligible for Automatic Suggestion

A Task is eligible for automatic suggestion for one selected Member only when:

- it is an Executable WBS/leaf Task;
- it is unfinished under the current completed-Task contract;
- its current Assignee is that selected Member;
- its Project status is Open or Locked;
- it has a complete current Execution Start/End pair;
- its current canonical Execution allocation projection is readable and
  internally consistent.

A Task is not eligible when:

- it is a Group;
- it is completed;
- it is unassigned;
- it is unscheduled or has a partial Execution date pair;
- its Project is Closed;
- it was deleted;
- its Execution projection cannot be resolved consistently.

Locked Project Tasks remain eligible because Sprint is read-only and consumes
the protected Execution baseline.

### 10.2 Eligible for Manual Add

Manual Add uses the same rules except the Task does not need positive allocation
inside the Sprint Period. Therefore a scheduled Task entirely before or after
the Sprint may be added and its complete current Execution allocation is shown
as-is.

Unscheduled Tasks are excluded from the picker and cannot be added through API
bypass.

### 10.3 Add Task Picker

- Search/list only eligible Tasks whose current Assignee is a selected Sprint
  Member.
- Exclude Tasks already associated with the Sprint.
- Render loading, empty, failure, and successful-add feedback inline beside the
  Sprint Planning actions; opening the picker must not navigate or open another
  modal.
- Display Project, Task Name, Assignee, Execution Start, and Execution End.
- Format dates as `DD MMM YYYY`, wrap Task Name within `50ch`, and do not display
  WBS or aggregate In-Sprint/Total allocation metadata in the picker.
- Deterministic picker ordering uses Execution End, Project Priority, WBS order,
  and ID.
- Adding a Task never adds its Assignee automatically; the Assignee must already
  be a selected Sprint Member.

---

## 11. Deterministic Task Suggestion

Suggestion is calculated independently for every selected Member, then composed
into one Sprint draft.

### 11.1 Mandatory Selection

Select **every** eligible Task satisfying:

```text
Task Execution End <= Sprint End
```

Rules:

- Sprint Start is not a lower bound for mandatory selection.
- An overdue unfinished Task whose complete allocation is before Sprint Start is
  still mandatory.
- A Task is selected even when its In-Sprint Allocation is `0`.
- A Task is selected even when mandatory Tasks already exceed Member or Sprint
  capacity.
- No mandatory Task may be dropped, truncated, or replaced by an optimizer.

Mandatory candidates are returned deterministically using:

1. Execution End ascending;
2. higher Project Priority first using the authoritative US-3.1/US-6.1 comparator;
3. persisted depth-first WBS order;
4. Task ID ascending.

This ordering never changes the must-include rule and does not control Sprint Planning
row order. The daily working-plan presentation order is owned by Section 12.3.

### 11.2 Fill Selection

After mandatory selection for one Member:

```text
Selected In-Sprint Allocation
= sum of selected Task Execution allocations dated inside Sprint Period
```

If this value is below Member Sprint Execution Capacity, consider remaining
eligible Tasks that:

- have `Execution End > Sprint End`; and
- have positive In-Sprint Allocation.

Order Fill Candidates by:

1. Execution End ascending;
2. higher Project Priority first using the authoritative US-3.1/US-6.1 comparator;
3. persisted depth-first WBS order;
4. Task ID ascending.

Add each candidate as a complete Task until:

- selected In-Sprint Allocation is equal to or above Member Sprint Execution
  Capacity; or
- no Fill Candidate remains.

Rules:

- A Task is never partially associated with a Sprint.
- The last selected Task may exceed remaining capacity.
- Overcapacity is allowed.
- Do not add a zero-In-Sprint-allocation Task merely to claim capacity was
  filled; such a Task remains available for manual Add.
- Under-capacity is valid when no positive Fill Candidate remains.
- Selection is deterministic for identical authoritative input.

### 11.3 Suggestion Failure

- A recoverable read failure must not persist a partial Sprint.
- The UI identifies the failed Member/section and supports retry.
- If canonical schedule/allocation integrity is inconsistent, affected Tasks are
  excluded from a new suggestion and the user receives a recoverable error;
  the system must not invent allocation.
- Save is disabled until the initial create suggestion has completed or the user
  explicitly retries/cancels.

---

## 12. Allocation Projection and Presentation

### 12.1 Source of Truth

Sprint reads canonical **Execution Allocation** only.

- Do not derive daily allocation by evenly spreading Effort across dates.
- Do not recalculate Task Capacity Allocation Percentage.
- Do not use Commitment or Actual Allocation.
- Do not persist copied allocation rows in the Sprint aggregate.

### 12.2 Per-Task Values

For each selected Task show:

- Project Name and status;
- Task Name, wrapped within a maximum `50ch` text width;
- current Assignee;
- Execution Start and End formatted as `DD MMM YYYY`;
- current daily positive Execution allocations;
- Completed/Needs Review indicators when applicable.

Do not display WBS number/path, aggregate In-Sprint/Outside/Total allocation text,
or the internal Daily Plan Order Date. WBS and Daily Plan Order Date remain
available to deterministic ordering logic only.

Only positive current Execution allocation on Sprint Dates is rendered in Task
Review. Positive allocation outside the Sprint Period remains a live scheduling
fact and may remain an ordering/read-model input, but it must not create a Date
column or allocation cell.

### 12.3 Daily Working-Plan Order and Grid

Date columns are chronological and formatted as `DD MMM YYYY`. Sprint boundary
columns remain visually identifiable. The grid renders exactly the inclusive
Sprint Start-to-End Date range in one horizontally scrollable grid; Sprint Planning
has no Previous/Next Date controls. Positive allocation outside the Sprint
Period is ignored for column discovery and is not rendered.

Within each selected Member group, Task rows are ordered using the complete live
canonical Execution allocation projection:

1. Unfinished Tasks with at least one readable positive allocation row come
   first.
2. `Daily Plan Order Date` ascending.
3. On the same Daily Plan Order Date, higher Project Priority first using the
   authoritative US-3.1/US-6.1 comparator.
4. Persisted depth-first WBS order.
5. Task ID ascending.
6. Retained completed Tasks that remain otherwise readable appear after all
   unfinished daily-plan rows, using Daily Plan Order Date, Project Priority,
   WBS order, and Task ID.
7. Retained Tasks without a readable positive allocation row appear last under
   `Needs Review`, using Project Priority, WBS order, and Task ID for
   deterministic fallback ordering.

Rules:

- The order uses the Task's complete positive allocation projection, not only
  the currently visible or Sprint-period columns; horizontal scrolling must not
  reorder rows.
- An unfinished overdue Task whose only positive allocation is before Sprint
  Start sorts before Tasks first allocated inside the Sprint.
- Ordering is not relative to `today` or application time; canonical allocation
  Dates remain the only date-order source.
- A manually added Task whose first positive allocation is after Sprint End
  sorts after Tasks with earlier allocation Dates.
- For non-contiguous allocation, only the earliest positive allocation Date is
  the primary row key; later positive allocation is visible only when its Date
  is inside the Sprint Period.
- Tasks sharing the same Daily Plan Order Date are planned on the same Date.
  Their tie-breaker is deterministic display order and must not be presented as
  an exact intra-Day start sequence.
- Live schedule/allocation drift may reorder rows on the next coherent read
  without changing Sprint membership, Status, or Version.
- Project context remains visible on every Task row. WBS path/rank remains an
  internal tie-breaker and is not displayed. Project roots, Group rows, and WBS
  hierarchy do not partition or override the daily plan order.

For every Sprint Date, each Member group exposes one Capacity row only. Task
allocation is represented by the Task rows themselves. There is no Sprint-level
daily summary and no aggregate allocation, remaining, or overcapacity row.

Additional grid rules:

- The first visible Date is Sprint Start and the last visible Date is Sprint End.
- Positive allocation before Sprint Start or after Sprint End does not create a
  column and is not displayed in Sprint Planning.
- Ignoring outside-Sprint allocation is a presentation rule only; it does not
  mutate canonical allocation or Sprint Task membership.
- A Sprint Date with `0h` resolved Member Daily Capacity marks the complete
  Member Date column and shows one textual `No capacity` marker in its header.
  Remaining Capacity does not control this marker.
- Daily values use hours in the repository-standard precision.
- Task names wrap at a maximum `50ch` width, including long unbroken names.

### 12.4 Member Group Capacity

Tasks are grouped by current Assignee when that Assignee is a selected Sprint
Member. Tasks are not secondarily grouped by Project because that would break the
daily working-plan order. Each group shows only:

- Member Sprint Execution Capacity;
- per-Date Member Capacity;
- individual Task allocation rows.

A selected Member with no Task remains as an empty group with period and per-Date
capacity.

### 12.5 Read-model Totals

The backend read model may continue to distinguish, per Sprint Date and for the
whole Sprint Period:

- selected-member Sprint Capacity;
- selected-member In-Sprint Allocation;
- selected-member Remaining Capacity;
- selected-member Overcapacity;
- Needs Review In-Sprint Allocation;
- all Sprint Task In-Sprint Allocation;
- all Sprint Task Total Execution Allocation.

Sprint-level Remaining Capacity and Overcapacity still sum per-Member, per-Date
values and must not be recomputed only from aggregate Capacity minus aggregate
Allocation. Needs Review allocation must not be disguised as selected-member
utilization when its current Assignee is not a selected Member. These totals are
not rendered in Sprint Planning.

---

## 13. Manual Consolidation

### 13.1 Remove Task

- Remove detaches only the Sprint Task relation.
- It does not delete or mutate the Task.
- A mandatory suggested Task may be removed manually during review; suggestion
  is advisory after generation.
- Regenerate may add it again because it still satisfies mandatory rules.
- Remove is available in Planned and Started.

### 13.2 Add Task

- Add creates one unique Sprint Task relation.
- Duplicate Add is idempotent or returns a stable duplicate error without
  creating two rows.
- Add does not alter capacity, dates, Assignee, schedule, or Project.
- Add is available in Planned and Started.

### 13.3 Save

Create Save persists atomically:

- Sprint fields;
- selected Member relations;
- reviewed Task relations;
- initial Status `planned`;
- Version/timestamps.

Edit Save atomically replaces the submitted Member/Task sets and updates fields
under optimistic concurrency. Failure preserves the existing Sprint unchanged.

---

## 14. Live Data and Schedule Drift

### 14.1 Persist Membership, Project Live Facts

Sprint Member IDs and Sprint Task IDs are explicit persisted decisions. All
capacity, Task, Project, date, status, and allocation facts are read live.

A change outside Sprint may alter the displayed capacity/allocation without:

- changing Sprint membership;
- changing Sprint Version;
- changing Sprint Status;
- invoking automatic regeneration.

### 14.2 No Silent Removal

A persisted Sprint Task must not be automatically removed because:

- its Assignee changed;
- its Assignee was cleared;
- its Execution dates/allocation changed;
- it moved entirely outside the Sprint Period;
- it became unscheduled;
- its Project became Closed;
- it became completed.

Only explicit Remove, Sprint deletion, or physical Task deletion removes the
relation.

### 14.3 Normal Regrouping

When a Task is reassigned to another selected Sprint Member:

- retain membership;
- move it to the new Member group;
- recalculate live group totals;
- no Needs Review warning is required solely for the reassignment.

### 14.4 Needs Review Conditions

Retain the Task under `Needs Review` when any of these current conditions apply:

- current Assignee is not a selected Sprint Member;
- current Assignee is empty;
- Task is unscheduled or has incomplete Execution dates;
- canonical Execution allocation is unavailable/inconsistent;
- Project is Closed;
- another live condition makes the Task ineligible for current Add/suggestion.

Warnings identify the exact condition and do not silently repair it.

Examples:

- `Assignee is not included in this Sprint.`
- `Task no longer has an Assignee.`
- `Task is no longer scheduled.`
- `Project is Closed.`
- `Execution allocation could not be resolved.`

### 14.5 Allocation Contribution During Drift

- Unscheduled/incomplete Task contributes zero live allocation until canonical
  allocation exists again.
- Task assigned to a non-Sprint Member is excluded from selected-member capacity
  utilization.
- Its readable allocation remains included in Needs Review and overall Sprint
  Task totals.
- Task entirely outside the Sprint remains a persisted Sprint Planning row, but its
  outside-Sprint daily allocation is not rendered because the grid contains
  Sprint Dates only. This condition alone may be informational rather than Needs
  Review when it remains otherwise eligible.
- A completed Task remains grouped when its Assignee is selected and shows a
  `Completed` badge; completion alone is not a Needs Review error.
- A Closed Project Task remains retained and receives a warning.

### 14.6 Master and Entity Deletion

- Hard deletion of a Task removes every Sprint Task relation for that Task in the
  same transaction as the Task deletion.
- This relation cleanup does not weaken US-4.1 delete eligibility or scheduler
  impact rules.
- Sprint membership alone does not block Member soft delete. Sprint Member
  relations are removed in the Member deletion transaction and affected retained
  Sprint Tasks become `Needs Review` on the next read.
- Deleted Sprint removes its Sprint Member and Sprint Task relations atomically;
  it never deletes Member, Task, Project, allocation, or schedule data.

---

## 15. Sprint Overlap

### 15.1 Conflict Rule

A create or edit is rejected when another non-deleted Sprint satisfies both:

```text
existing.start_date <= draft.end_date
AND draft.start_date <= existing.end_date
```

and the two Sprints share at least one selected Member.

Rules:

- Planned and Started participate identically.
- Boundary contact on the same Date is overlap.
- Different Members may have overlapping Sprint periods.
- The edited Sprint excludes itself from conflict comparison.
- Task membership does not determine overlap; selected Sprint Members do.
- Schedule drift or a Task reassignment does not silently mutate Sprint Members
  and therefore does not create/remove overlap by itself.

### 15.2 Error Detail

Conflict response identifies at least:

- conflicting Sprint ID and Name;
- inclusive conflicting range;
- conflicting Member IDs/display names safe for UI.

The UI keeps the draft and lets the user change dates or Members.

### 15.3 Concurrency

Two concurrent creates/edits that would create a same-Member overlap must not
both commit. The backend must coordinate validation and write in one transaction
using a measured persistence strategy appropriate to PostgreSQL. Frontend
pre-check is advisory only.

---

## 16. Sprint Status and Actions

### 16.1 Create

- Successful create always sets Status `planned`.
- Client cannot create directly as Started.

### 16.2 Start

`Start` performs only:

```text
planned -> started
```

- Set `started_at` once.
- Increment Sprint Version.
- Do not regenerate Tasks or snapshot capacity/allocation.
- Do not invoke scheduler or mutate related entities.
- Repeated Start on the current Started version is idempotent or returns a stable
  already-started conflict; it never changes other data.
- Stale Version is rejected.

### 16.3 Planned and Started Parity

Both statuses allow:

- open/view;
- edit Details;
- edit Members;
- Add Task;
- Remove Task;
- Regenerate Suggestion;
- Save;
- Delete.

There is no Stop, Reopen, Complete, or Close action in this story.

### 16.4 Delete

- Delete requires confirmation containing Sprint Name.
- Delete uses Version/optimistic concurrency.
- Planned and Started are both deletable.
- Successful delete atomically removes Sprint and its relation rows only.
- Cancel sends no delete request.

---

## 17. API Contract

Exact route naming may follow repository conventions. Observable behaviour must
support at least:

```text
GET    /api/sprints
POST   /api/sprints
GET    /api/sprints/{sprintId}
PUT    /api/sprints/{sprintId}
POST   /api/sprints/{sprintId}/start
DELETE /api/sprints/{sprintId}
POST   /api/sprints/suggestion
GET    /api/sprints/{sprintId}/task-candidates
```

An equivalent command/query split is allowed when behaviour remains identical.

### 17.1 Create/Update Command

Command contains:

- Name;
- Start Date;
- End Date;
- selected Member IDs;
- selected Task IDs;
- Version for update.

Backend validates independently of the suggestion endpoint:

- field validation;
- Member existence/active state;
- Task existence and manual-add eligibility at command time;
- uniqueness of relation sets;
- no same-Member overlap;
- optimistic Version;
- referential integrity.

Because schedule data is live, a Task that drifted after review but before Save
may be retained only when it was already part of an edited Sprint; a newly added
Task must still satisfy manual-add eligibility at commit time. The response may
return stable drift warnings instead of deleting existing membership.

### 17.2 Read Contract

Sprint detail read returns one version-coherent composition containing:

- Sprint fields/status/version;
- selected Members and live capacity per Sprint Date;
- per-Member and Sprint per-Date selected allocation, Remaining Capacity, and
  Overcapacity under the no-netting rules;
- selected Tasks with Project/WBS/Assignee/status;
- complete current positive Execution allocation dates;
- server-resolved daily working-plan row order or an explicit nullable Daily Plan
  Order Date plus stable display rank that yields the exact Section 12.3 order;
- In-Sprint/Outside/Total allocation;
- Member and Sprint period totals;
- Needs Review daily allocation and warnings;
- enough version/projection data to prevent stale UI overwrite.

Avoid per-Member, per-Task, and per-Date N+1 requests.

### 17.3 Suggestion Contract

Suggestion is rollback-free read computation and returns:

- normalized draft details;
- selected Member capacities per Sprint Date;
- mandatory and fill-selected Task IDs;
- per-Task reason `mandatory` or `capacity_fill`;
- complete daily allocation and the exact daily working-plan row order;
- per-Member and Sprint per-Date Remaining Capacity and Overcapacity without
  cross-Member or cross-Date netting;
- period totals;
- projection/version token sufficient to identify stale display.

Suggestion does not reserve data and does not guarantee later Save input remains
unchanged.

### 17.4 Error Contract

Use stable machine-readable codes, including equivalents of:

| Code                            |    HTTP | Meaning                                                 |
| ------------------------------- | ------: | ------------------------------------------------------- |
| `SPRINT_NAME_REQUIRED`          |     400 | Missing/blank Name                                      |
| `SPRINT_NAME_TOO_LONG`          |     400 | Name exceeds limit                                      |
| `SPRINT_NAME_CONFLICT`          |     409 | Normalized Name already exists                          |
| `SPRINT_DATE_RANGE_INVALID`     |     400 | End before Start or malformed date                      |
| `SPRINT_MEMBER_REQUIRED`        |     400 | No selected Member                                      |
| `SPRINT_MEMBER_NOT_FOUND`       | 404/409 | Submitted Member unavailable                            |
| `SPRINT_MEMBER_INACTIVE`        |     409 | Submitted Member is inactive/deleted                    |
| `SPRINT_MEMBER_OVERLAP`         |     409 | Another Sprint overlaps for shared Member               |
| `SPRINT_TASK_NOT_FOUND`         | 404/409 | Submitted Task unavailable                              |
| `SPRINT_TASK_NOT_ELIGIBLE`      |     409 | Newly added Task fails manual-add eligibility           |
| `SPRINT_TASK_UNSCHEDULED`       |     409 | Newly added Task has no complete Execution schedule     |
| `SPRINT_ALREADY_STARTED`        |     409 | Start requested for Started Sprint                      |
| `SPRINT_VERSION_CONFLICT`       |     409 | Stale update/start/delete                               |
| `SPRINT_NOT_FOUND`              |     404 | Sprint unavailable                                      |
| `SPRINT_SUGGESTION_UNAVAILABLE` | 503/409 | Canonical capacity/allocation cannot be composed safely |

Infrastructure details must not leak to users.

---

## 18. Persistence, Transaction, and Query Requirements

### 18.1 Persistence

Recommended relational shape:

- `sprints`;
- `sprint_members`;
- `sprint_tasks`.

Required invariants:

- normalized Sprint Name unique among active rows;
- Start/End valid at domain and persistence boundary where supported;
- Status constrained to Planned/Started values;
- unique Sprint Member pair;
- unique Sprint Task pair;
- foreign-key/repository integrity;
- cascade only for relation cleanup when Sprint or Task is deleted, and for
  Sprint Member cleanup under valid Member deletion.

### 18.2 Atomicity

The following are atomic:

- create Sprint plus all Member/Task relations;
- edit Sprint plus complete submitted Member/Task set;
- Start status/version/timestamp;
- Delete Sprint plus relations;
- Task delete plus Sprint Task cleanup;
- Member delete plus Sprint Member cleanup and existing US-1.2 relations.

Any validation, overlap, version, or persistence failure leaves prior state
unchanged.

### 18.3 Query and Index Review Gate

Implementation must review measured PostgreSQL plans for:

- normalized Name lookup/uniqueness;
- list ordering;
- detail relation fetch;
- shared-Member inclusive overlap lookup;
- eligible/suggested Tasks by selected Assignee, Project status, completion, and
  Execution End;
- bounded allocation fetch for selected/candidate Task IDs;
- daily-plan ordering across selected Tasks from multiple Projects/WBS branches;
- per-Member, per-Date selected-allocation and variance aggregation;
- capacity inputs for selected Member IDs and Sprint range.

Do not issue one query per Member, Task, allocation Date, Project, holiday, or
override. Do not add speculative indexes without the production query shape and
plan evidence required by project architecture.

### 18.4 Optimistic Concurrency

- Update, Start, and Delete use Sprint Version.
- A stale response cannot overwrite a newer Sprint draft/detail in the frontend.
- Schedule/allocation drift does not increment Sprint Version; detail response
  still carries a live projection token/version boundary so rapid reads cannot
  restore stale capacity/allocation.

---

## 19. UX, Accessibility, and Responsive Behaviour

- Use shared Dialog, Stepper, Table/Grid, Button, Badge, Alert, Tooltip, date
  controls, and semantic tokens.
- Every input has an associated label and inline validation.
- Stepper state and errors are keyboard accessible.
- Member groups and `Needs Review` use semantic headings.
- Daily Capacity cells expose Member, full formatted Date, and Capacity to
  assistive technology; `0h` capacity includes textual `No capacity` state.
- Daily Task allocation cells expose Member, Task, Project context, full formatted
  Sprint Date, and allocation to assistive technology.
- Zero-capacity state is not communicated by color alone.
- Warning badges include text and accessible description.
- Horizontal scrolling does not trap keyboard focus.
- Sticky identity columns may be used, but Task Name/Member context must remain
  available while reviewing dates.
- Loading one capacity/allocation section does not erase valid form input.
- Failed Save/Start/Delete keeps the dialog/page recoverable.
- Duplicate action submission is disabled.
- Destructive Delete requires explicit confirmation; Remove Task does not delete
  the Task and must be labelled accordingly.
- Responsive layouts may switch the daily grid to a date-row disclosure, but all
  required Sprint-period values remain accessible. Outside-Sprint allocation is
  intentionally absent from Sprint Planning presentation.

---

## 20. Acceptance Criteria

### Navigation and List

1. `Sprints` appears under the visible Project navigation group while Home
   remains the default/root page.
2. Opening Sprints does not require or infer a selected Project.
3. Sprint List displays Name, Start Date, End Date, Planned/Started Status, and
   valid Actions in deterministic order.
4. Loading, empty, error/retry, and duplicate-submission states are present.
5. Sprint Name opens the same editable detail for Planned and Started.

### Details and Validation

6. Create uses modal Details and Members steps in order, then opens Sprint Planning
   on the main Sprint page.
7. Name is trimmed, required, limited to 200 characters, and unique
   case-insensitively.
8. Start/End are date-only inclusive boundaries; End before Start is rejected.
9. One-day, weekend-boundary, and Public-Holiday-boundary Sprints are valid.
10. At least one active Member is required.
11. Member with zero capacity or no eligible Task remains selectable/visible.
12. Backend rejects invalid/stale input even when frontend validation is bypassed.

### Standalone and Non-Interference

13. Sprint persists no Project/Portfolio owner.
14. One Sprint can contain Tasks from multiple Open/Locked Projects.
15. Sprint create/edit/start/delete/suggestion performs no scheduler, dependency,
    Project, WBS, Task, capacity, date, or allocation mutation.
16. Sprint commands never advance Project schedule version or require scheduling
    impact confirmation.

### Capacity

17. Per-Date capacity reuses weekend/Public Holiday, minimum active override,
    Daily Capacity, Member Buffer, and `0.5h` Execution rounding rules.
18. Project Buffer is not applied.
19. Member and Sprint views distinguish per-Date and period capacity,
    In-Sprint Allocation, Remaining Capacity, and Overcapacity; summaries sum
    per-Member, per-Date variance without cross-Member or cross-Date netting.
20. Zero capacity and overcapacity never block Save or require confirmation.
21. Live capacity changes update read results without changing Sprint membership,
    Status, or Version.

### Suggestion

22. Suggestion considers only unfinished scheduled executable Tasks assigned to
    selected Members in Open/Locked Projects.
23. Completed, unassigned, unscheduled, partial-date, Group, Closed-Project, and
    deleted Tasks are excluded from a new suggestion.
24. Every eligible Task with `Execution End <= Sprint End` is selected.
25. Mandatory selection includes overdue Tasks ending before Sprint Start and
    Tasks with zero In-Sprint Allocation.
26. Mandatory selection is not reduced by Member or Sprint overcapacity.
27. Remaining capacity uses only canonical In-Sprint Execution Allocation.
28. Fill considers later Tasks with positive In-Sprint Allocation in deterministic
    Execution End, Project Priority, WBS, ID order.
29. Fill adds complete Tasks until capacity is met/exceeded or candidates end.
30. The final fill Task may create overcapacity; no Task is partially selected.
31. Zero-In-Sprint later Tasks are not auto-added solely to fill capacity but are
    available for manual Add.
32. Identical authoritative input produces identical suggestion.
33. Suggestion failure persists no partial Sprint and does not invent allocation.

### Sprint Planning and Allocation

34. Sprint Planning groups normal Tasks by current selected Assignee and orders
    Task rows across Projects by earliest positive canonical Execution allocation
    Date, then Project Priority, WBS order, and Task ID. Project context remains
    visible; WBS context is not displayed and hierarchy never overrides order.
35. Selected Members with no Tasks remain visible with period and per-Date
    Capacity.
36. Every Task displays Project, Name, Assignee, formatted Execution dates, and
    daily Execution allocation. Task names wrap within `50ch`; WBS,
    In-Sprint/Outside/Total allocation text, and Daily Plan Order Date are hidden.
37. Sprint Planning renders exactly the inclusive Sprint Date range. Positive
    allocation outside the Sprint Period does not create a Date column or visible
    allocation cell.
38. Sprint Planning shows only Member Capacity summaries; allocation is represented
    by Task cells, while aggregate allocation, Remaining, and Overcapacity are
    not displayed.
39. Sprint Planning has no Previous/Next Date controls; all Sprint Dates are
    available through horizontal scrolling.
40. Add Task lists scheduled eligible Tasks for selected Members, including Tasks
    with zero In-Sprint Allocation.
41. Unscheduled Task is absent from the picker and backend Add rejects it.
42. Remove detaches Sprint membership only and never deletes/mutates the Task.
43. Manual Add/Remove works for both Planned and Started.
44. Regenerate is explicit, replaces reviewed membership, confirms after manual
    consolidation, and never persists before Save.

### Persistence and Drift

45. Create atomically persists Sprint, selected Members, selected Tasks, Planned
    Status, Version, and timestamps.
46. Edit atomically replaces submitted relation sets under Version control.
47. Sprint stores membership but reads capacity, Task, Project, dates, and
    allocation live.
48. Opening/editing a saved Sprint does not regenerate membership automatically.
49. Reassignment to another selected Member regroups the Task and recalculates
    totals without removing membership.
50. Reassignment to non-Sprint Member retains Task under Needs Review and excludes
    it from selected-member utilization.
51. Cleared Assignee retains Task under Needs Review.
52. A previously selected Task that becomes unscheduled remains under Needs
    Review and contributes zero allocation until live allocation returns.
53. A Task shifted entirely outside Sprint remains associated and visible as a
    Task row with zero visible Sprint-period allocation; outside allocation is
    not rendered in Sprint Planning.
54. Completed Task remains associated and displays Completed without automatic
    removal.
55. Closed-Project Task remains associated with an explicit warning and is not
    newly suggested/added.
56. Physical Task deletion removes Sprint Task relations atomically without
    weakening existing delete eligibility/scheduler rules.
57. Member deletion is not blocked solely by Sprint; Sprint Member relations are
    removed atomically and affected retained Tasks become Needs Review.
58. No drift condition silently adds/removes a Sprint Member or Sprint Task.

### Overlap

59. Inclusive overlapping Sprint periods are rejected when at least one selected
    Member is shared.
60. Boundary contact on one Date is rejected for the shared Member.
61. Overlap is allowed when no Member is shared.
62. Planned and Started participate identically in overlap validation.
63. Edit excludes itself but validates newly added Members/date changes.
64. Concurrent conflicting writes cannot both commit.
65. Conflict keeps draft input and identifies Sprint/range/conflicting Members.

### Status and Delete

66. Successful create always yields Planned.
67. Start performs only one-way Planned-to-Started metadata transition, sets
    Started At, and increments Sprint Version.
68. Started retains all Planned edit, Add, Remove, Regenerate, Save, and Delete
    behaviour.
69. Repeated/stale Start cannot mutate other data.
70. Planned and Started delete require confirmation and remove only Sprint plus
    relation rows atomically.
71. Cancel Delete sends no delete request.

### Concurrency, Errors, and Accessibility

72. Normalized Name conflicts and Version conflicts have stable errors and no
    partial write.
73. Relation uniqueness prevents duplicate Sprint Member or Sprint Task rows.
74. Rapid list/detail/suggestion changes cannot restore stale capacity,
    allocation, Member, Task projection, or daily working-plan order.
75. Backend read paths are set-based and avoid per-entity/per-Date N+1 queries.
76. All inputs, actions, Daily Task allocation values, Member Capacity values,
    whole-column zero-Daily-Capacity markers, and warnings are keyboard and
    screen-reader accessible.
77. Recoverable failures preserve valid modal input and page-level reviewed
    membership.
78. Overdue outside-only Tasks, in-Sprint Tasks, after-Sprint manual Tasks,
    retained completed Tasks, and no-allocation Needs Review Tasks appear in the
    deterministic order defined by Section 12.3.
79. The backend projection preserves per-Member/per-Date no-netting values even
    though Sprint Planning does not render Remaining, Overcapacity, or Sprint summary
    rows.
80. Task Name opens the shared Home Edit Task dialog without navigating away from
    Sprints. Close performs no Sprint action; successful Task Save returns to the
    same Sprint Planning workspace and regenerates the local suggestion without
    persisting Sprint membership until `Save Sprint Planning`.
81. Every AC is supported by Code Inspection, Unit/Integration Test, and
    Acceptance-Level Test evidence per `AGENTS.md`.

---

## 21. Mandatory Test Coverage

### 21.1 Domain/Application Tests

- Name normalization, blank/length/duplicate validation.
- Date-only valid/invalid and one-day boundaries.
- At-least-one-Member validation.
- Inclusive overlap truth table, including shared/no-shared Member and self-edit.
- Capacity per Date for weekday, weekend, Public Holiday, multiple overrides,
  buffer, zero capacity, and `0.5h` rounding.
- Daily Remaining/Overcapacity with no cross-Date or cross-Member netting,
  including positive allocation on a zero-capacity Date.
- Daily working-plan comparator for overdue, in-Sprint, after-Sprint,
  non-contiguous, same-Date tie, completed-after-unfinished, and
  unreadable-allocation fallback cases.
- Mandatory selection before Sprint Start, inside Sprint, exactly on End, and
  overcapacity.
- Fill ordering, final-Task overcapacity, no positive candidate, and deterministic
  tie-breakers.
- Manual Add eligibility for zero In-Sprint allocation and rejection of
  unscheduled/Closed/completed/unselected-Assignee Task.
- Planned-to-Started only; Planned/Started parity.
- Drift classification and allocation contribution rules.

### 21.2 Repository/Integration Tests

- Create/edit relation atomicity and rollback.
- Case-insensitive Name concurrency.
- Same-Member overlap concurrency so only one transaction commits.
- Unique Sprint Member/Task constraints.
- Optimistic update/start/delete conflict.
- Task hard-delete Sprint relation cleanup in the Task transaction.
- Member soft-delete Sprint relation cleanup in the existing Member transaction.
- Sprint delete relation cleanup without related entity deletion.
- Set-based detail/suggestion reads and reviewed PostgreSQL plans.

### 21.3 HTTP/Component Tests

- List columns/order/states/actions.
- Three-step create validation and input preservation.
- Member selection, zero-capacity/no-Task group.
- Initial suggestion, explicit regenerate, consolidation confirmation.
- Mandatory overcapacity visual and no Save block.
- Member period/per-Date Capacity only, with textual zero-capacity state.
- Absence of Sprint summary, Previous/Next Date controls, aggregate allocation,
  Remaining, and Overcapacity rows.
- Daily working-plan order across Projects/WBS paths while WBS remains hidden.
- Daily allocation for Sprint Dates only in one horizontally scrollable grid;
  outside-Sprint allocation creates no column.
- `DD MMM YYYY` date formatting and `50ch` Task-name wrapping.
- Add picker and Remove semantics.
- Task Name opens the shared Home Edit Task dialog without page navigation;
  Close preserves local Sprint Planning, while Save regenerates suggestion and
  does not persist Sprint membership.
- Needs Review variants and normal reassignment regrouping.
- Started edit parity and no Start action.
- Delete confirmation/cancel/failure.
- Stable errors for overlap, stale Version, ineligible Task, and suggestion read
  failure.
- Rapid response/stale projection tests.
- Keyboard, focus, accessible labels, warning text, and non-color-only state.

### 21.4 Acceptance-Level Scenarios

At minimum:

1. Create one Sprint across Tasks from multiple Projects and save as Planned.
2. Mandatory Tasks exceed capacity but all are retained and Save succeeds.
3. Mandatory overdue Task ends before Sprint Start, remains in deterministic row
   order, and shows no outside-Sprint allocation column.
4. Remaining capacity selects later Tasks until the last Task exceeds capacity.
5. Manually add a scheduled Task entirely after Sprint; retain its Task row while
   rendering only the Sprint Date columns.
6. Reject unscheduled Task through UI and direct API.
7. Reassign a selected Task to another Sprint Member and verify regrouping.
8. Reassign to non-Sprint Member and verify Needs Review plus separate totals.
9. Make a saved Task unscheduled and verify it remains with zero allocation.
10. Close its Project and verify retained warning/no automatic deletion.
11. Reject overlapping date range for one shared Member; allow same dates for
    disjoint Members.
12. Concurrent overlap attempts yield one committed Sprint.
13. Start Sprint and verify no behavioural difference except metadata/action.
14. Delete Started Sprint and verify Projects/Tasks/schedules remain unchanged.
15. Delete Task and Member through their owning flows and verify atomic Sprint
    relation cleanup.
16. Show one Member's Tasks from multiple Projects/WBS branches in earliest
    positive allocation-Date order without changing Project Priority or WBS.
17. Show an overdue outside-only Task before in-Sprint Tasks, a manually added
    after-Sprint Task after earlier planned work, and a retained completed Task
    below unfinished daily-plan rows.
18. The backend detail projection preserves Member A overcapacity and Member B
    remaining capacity independently without exposing those summaries in Sprint
    Planning.
19. Sprint Planning renders Member groups directly, with Capacity only and without a
    Sprint daily summary or Date-window navigation controls.
20. A Sprint Date whose Member Daily Capacity is `0h` marks that Member's whole
    Date column and shows `No capacity` in the column header; any positive
    canonical Task allocation remains visible in its marked Task cell and Save
    is still allowed. Remaining Capacity does not control the marker.
21. Click a Sprint Planning Task Name, edit it through the shared Home Task
    dialog, and verify Close returns unchanged while Save stays on Sprints,
    regenerates the local suggestion, and does not persist Sprint membership
    before `Save Sprint Planning`.

---

## 22. Documentation and Evidence

Implementation must update:

- project API documentation;
- migration/schema documentation;
- `docs/project/architecture.md` with actual aggregate/query decisions;
- this story if observable behaviour changes;
- implementation evidence mapping every AC to all three confidence levels.

Do not mark this story complete while mandatory validation or any confidence
level is missing.
