# US-7.1 — Home Portfolio Gantt Workspace

> **Authority:** This story owns the Home navigation entry, portfolio Gantt read
> model, Project selection and global saved-filter behaviour, WBS/timeline
> presentation, and integration of the existing Project/WBS forms into the Home
> workspace. It does not replace the mutation rules owned by US-3.1, US-3.3,
> US-4.1, US-4.2, US-5.1, US-6.1, or US-6.2.
>
> **UX reference:** GanttPRO is a non-normative interaction reference for the
> split project-grid/timeline layout, hierarchical rows, summary bars, dependency
> arrows, portfolio composition, and visible non-working dates. Where GanttPRO
> behaviour conflicts with this story, this story wins. In particular, SchedMind
> Phase 1 keeps every Gantt cell and timeline bar read-only.

## 1. User Story

**As an** Engineering Lead

**I want** a portfolio Gantt workspace on the Home page

**So that** I can review active Projects, understand their WBS and schedules,
select useful Project combinations, and open the existing Project/Task editing
flows from one planning view.

---

## 2. Business Context

Project Structure currently exposes one Project's WBS through a hierarchical
list and contextual dialogs. The product also needs a portfolio-level planning
view that combines multiple active Projects, preserves their individual WBS
hierarchies, and visualizes Execution or Commitment dates on one daily timeline.

The Home Gantt is a **read-only schedule projection plus navigation surface**.
It is not a second scheduler, a second WBS aggregate, or a second mutation
implementation. Add Task, Add Child, Project edit, Task/Group edit, Dependency,
completion, Reopen, impact preview, validation, and rollback continue to use the
same application contracts and business invariants as their owning stories.

Project is WBS level `0`, but the Project row does not display a literal `0` in
the WBS column. Every Project starts its visible descendant numbering at `1`.
This keeps multi-Project rows distinguishable while avoiding a redundant WBS
value on the Project itself.

---

## 3. Product Reference

The approved interaction model is inspired by the following GanttPRO concepts:

- a project/task grid on the left and schedule bars on the right;
- portfolio Gantt composition across multiple Projects;
- hierarchical WBS numbering and expandable summary rows;
- standard columns such as WBS, assignee, dates, and effort/estimation;
- visible dependency arrows;
- visible days off in a daily timeline.

The reference does **not** authorize GanttPRO-specific behaviour such as inline
field editing, bar drag/resize, drag-to-create dependency, custom zoom levels,
permissions, user-scoped views, progress fields, milestones, or cost fields.
Those capabilities are out of scope unless approved separately.

Reference pages:

- `https://help.ganttpro.com/hc/en-us/articles/360020650677-GanttPRO-s-interface`
- `https://help.ganttpro.com/hc/en-us/articles/18253453898514-Project-portfolio-management`
- `https://help.ganttpro.com/hc/en-us/articles/5423187790353-Standard-fields`
- `https://help.ganttpro.com/hc/en-us/articles/16403302087314-Types-of-tasks-and-levels`
- `https://help.ganttpro.com/hc/en-us/articles/5486711724433-Customizing-Gantt-chart-view`

---

## 4. Scope

### 4.1 In Scope

- Add **Home** as the first application navigation menu and default frontend
  root page.
- Display a portfolio Gantt for selected Open and Locked Projects.
- Preserve the existing Project Structure menu as an alternative workflow.
- Display Project, Group, and Task rows in a hierarchical left grid.
- Display WBS, Name, Role, Assignee, Effort, Start Date, and End Date columns.
- Switch the entire workspace between Execution and Commitment projections.
- Display Project and Group recursive summary ranges and recursive Effort.
- Display daily calendar columns including weekends and Public Holidays.
- Display a two-row timeline header with working-day sequence and calendar date.
- Display read-only Task bars, Project/Group summary bars, and dependency arrows.
- Expand and collapse Project and Group rows.
- Open the existing Project or WBS form by clicking a row or bar.
- Provide Add Task on Project rows and Add Child on eligible WBS rows.
- Select saved filter, projection, Projects, and Task Roles through the Configure Gantt modal workflow.
- Provide a system view named `All Active Projects`.
- Create, open, update, duplicate, and delete globally persisted named filters.
- Use an automatic visible range derived from the currently filtered Task set.
- Preserve scheduler, lifecycle, confirmation, transaction, concurrency, and
  stale-projection rules from the owning stories.
- Provide backend read and persistence contracts, loading/error/empty states,
  accessibility, responsive behaviour, and performance safeguards.

### 4.2 Out of Scope

- Replacing or removing Project Structure.
- Inline editing of any grid cell.
- Dragging or resizing Project, Group, or Task bars.
- Dragging rows to reorder or move WBS nodes.
- Creating, deleting, or retargeting dependency from the chart.
- Editing generated or manual dates from the chart.
- Actual Date or Actual Allocation visualization.
- Forecast, Delivery Impact, Project Health, Critical Path, baseline comparison,
  milestones, progress, cost, workload, or resource-utilization visualization.
- Week, month, quarter, year, or custom zoom scales.
- User login, ownership, per-user filter visibility, or permissions.
- Saved filter rename as a dedicated action; `Save As` creates a new name.
- Saved filter merge or advanced conflict-resolution UI.
- Persisting timeline range, scroll position, selected projection,
  expand/collapse state, task selection, or search text in saved filters.
- Historical Gantt for Closed Projects.
- Export, printing, share link, or public portfolio access.
- A new scheduler or duplicated WBS/Dependency mutation service.

---

## 5. Terminology

| Term | Definition |
| --- | --- |
| Home | Default application page containing the Portfolio Gantt Workspace. |
| Portfolio Gantt | Combined read-only view of selected active Projects, their WBS rows, and schedule bars. |
| Project Grid | Hierarchical left panel containing Project, Group, and Task rows and fixed data columns. |
| Timeline | Right panel containing daily calendar columns, bars, and dependency arrows. |
| Active Project | A Project with status Open or Locked. |
| Selected Project | An active Project checked in the current Filter draft. |
| Selected Role | A Task Role checked in the current Filter draft. |
| Selected Projection | Either Execution or Commitment. |
| Working-Day Anchor | Earliest complete scheduled Start among selected Projects for the selected projection. |
| Visible Range | Calendar-date interval currently rendered in the timeline. |
| System View | Non-persisted logical view `All Active Projects`. |
| Saved Filter | Globally persisted name and selected active Project IDs. |
| Configuration Draft | Current saved-filter, projection, Project, and Role selection inside the Configure Gantt modal. Project selection may differ from persisted saved-filter content until Save; Role selection is session-only. |

---

## 6. Navigation and Page Ownership

### 6.1 Home as Root Page

- Add `Home` as the first visible item in application navigation.
- Opening the frontend application at its root displays Home without requiring
  the user to choose another menu first.
- The implementation may preserve the existing hash-navigation architecture;
  empty/default route resolution must still render Home.
- Home is the default safe fallback when no recognized application page is
  selected.
- Existing navigation destinations remain available.
- Project Structure remains accessible through the existing Project workflow.

### 6.2 Page Regions

Home contains:

1. a compact Gantt header;
2. a fixed hierarchical Project Grid on the left;
3. a horizontally scrollable Timeline on the right.

The Home page does not display a breadcrumb or a persistent configuration
toolbar. The compact Gantt header contains:

- the active saved-filter name as the page title;
- `All Active Projects` when the system view is active;
- no generic `Portfolio Gantt` title and no visible-range date subtitle;
- one icon-only **Configure Gantt** action with an accessible name and tooltip;
- an optional non-blocking loading indicator.

The Configure Gantt action opens one modal form containing all Home display
configuration:

- saved-filter selection and Save, Save As, and Delete actions;
- Execution/Commitment projection selection;
- Project selection and search;
- Role selection.

The configuration modal must not change page height or push the Gantt
vertically while it is opened or closed. Draft changes affect the Gantt only
when Apply is confirmed, except an explicitly completed Save, Save As, or Delete
operation on persisted saved-filter data.

Grid and Timeline requirements:

- the Project Grid and Timeline are separate sibling panels and must never
  overlap or paint over each other;
- the Timeline takes the remaining viewport width after the Project Grid;
- vertical scrolling is synchronized so one logical row never separates from
  its bar;
- grid column headers and both timeline header rows remain visible while
  vertically scrolling;
- horizontal timeline scrolling does not move the left grid;
- Project Grid columns are individually resizable; changing their total width
  reduces or expands the remaining Timeline width;
- row heights remain identical across both panels;
- Project and Group indentation remains readable at supported viewport sizes;
- the page does not create nested uncontrolled vertical scroll regions that
  break keyboard or pointer navigation.

---

## 7. Project Eligibility, Ordering, and Rows

### 7.1 Status Eligibility

- Open Projects are visible and editable according to their owning stories.
- Locked Projects are visible and act as immutable planning anchors; only
  operations explicitly allowed by US-6.2 remain available through opened
  forms.
- Closed Projects are inactive and excluded from the Home checklist, Project
  Grid, Timeline, and saved-filter active selection.
- Deleted Projects are excluded.
- A Closed or deleted Project ID may remain historically stored in a saved
  filter until the next Save or Save As, but it never appears in the current
  selection or chart.

### 7.2 Project Order

- Selected Projects use the authoritative Project ordering from US-3.1.
- Active Projects are ordered by Project Priority and the established
  deterministic tie behaviour.
- Opening a saved filter does not create a custom Project order.
- A Project Priority change visible to Home refreshes the ordering without a
  hard page reload.

### 7.3 Project Row

Each selected Project is displayed as a top-level row:

- WBS column is blank; literal `0` is not shown.
- Name displays Project Name.
- Assignee is blank.
- Start and End use recursive Project aggregate for the selected projection.
- Effort uses recursive known descendant Task Effort.
- The row can expand or collapse its WBS descendants.
- The row has an Add Task action when Project lifecycle rules allow it.
- Clicking the row or its summary bar opens the existing Edit Project form.
- Locked status is communicated with text/icon and not by colour alone.

### 7.4 Group Row

Each Group row:

- displays derived hierarchical WBS number;
- displays Group Name;
- displays blank Assignee;
- displays recursive Start, End, and Effort values;
- can expand or collapse descendants;
- has Add Child when lifecycle and WBS rules allow it;
- opens the existing Group form when row or summary bar is clicked;
- does not become an executable entity because it is shown on Gantt.

### 7.5 Task Row

Each executable Task row:

- displays derived hierarchical WBS number;
- displays Task Name;
- displays Task Assignee or blank when unassigned;
- displays selected-projection Start and End when the pair is complete;
- displays direct Effort or blank when Effort is absent;
- may expose Add Child according to US-4.1 conversion rules;
- opens the existing Task form when row or bar is clicked;
- does not allow direct cell or bar mutation.

---

## 8. WBS Numbering and Hierarchy

- Project is logical WBS level `0`, but its WBS cell is blank.
- Numbering restarts independently inside every Project.
- Top-level WBS nodes are `1`, `2`, `3`, and so on.
- Child nodes are `1.1`, `1.2`, `2.1`, and so on.
- Deeper descendants continue as `1.1.1`, `1.1.2`, and so on without a product
  depth limit.
- Numbering derives from current sibling order and hierarchy; it is not a
  persisted business identifier.
- Move or reorder completed through the existing Project Structure flow changes
  derived numbers on the next Home projection.
- IDs, not WBS numbers or names, identify rows, dependencies, and saved filter
  Projects.
- A malformed or stale projection must not use duplicate WBS numbers as row
  identity.

---

## 9. Grid Columns

Columns appear in this order:

1. WBS
2. Task / Group Name
3. Role
4. Assignee
5. Effort
6. Start Date
7. End Date

### 9.1 Column Semantics

| Column | Project | Group | Task |
| --- | --- | --- | --- |
| WBS | Blank | Derived hierarchy number | Derived hierarchy number |
| Task / Group Name | Project Name | Group Name | Task Name |
| Role | Distinct descendant Task Roles | Distinct descendant Task Roles | Task Role or blank |
| Assignee | Blank | Blank | Assignee Name or blank |
| Effort | Recursive known Task Effort sum | Recursive known Task Effort sum | Direct Task Effort |
| Start Date | Earliest recursive complete selected-projection Start | Earliest recursive complete selected-projection Start | Selected-projection Start |
| End Date | Latest recursive complete selected-projection End | Latest recursive complete selected-projection End | Selected-projection End |

### 9.2 Date-Only Display

- Dates use date-only values and shared date formatting.
- Browser timezone conversion must not shift a date.
- The visual format must remain unambiguous across month and year boundaries.
- Accessible names expose a complete calendar date even when the visual cell is
  compact.
- A partial Start/End pair is invalid source data and must not be presented as a
  valid scheduled range.

### 9.3 Task Without Effort

- A Task with null Effort displays blank, not `0`.
- Project/Group Effort sums only known descendant Task Effort using integer
  minutes before formatting.
- A Project/Group with no descendant Task having known Effort displays blank,
  not invented `0h`.
- A Project/Group with one or more descendant Tasks without Effort exposes a
  visible/accessibly labelled incomplete-effort indicator using the same
  disclosure semantics as US-4.3.
- Completed and unfinished Tasks with known Effort both contribute to total
  Effort; this column is not remaining effort or completed effort.

---

## 10. Execution and Commitment View

### 10.1 Selector

Home provides exactly two schedule-view options in this story:

- Execution
- Commitment

Actual and Forecast are absent.

### 10.2 Atomic Projection Change

Changing the selected projection updates one coherent workspace state:

- Task Start and End values;
- Project and Group recursive Start and End values;
- Task bars;
- Project and Group summary bars;
- working-day anchor and header sequence;
- default visible range when the current range has not been manually changed;
- scheduled/unscheduled and incomplete-schedule indicators;
- dependency geometry.

The UI must not show Execution grid dates with Commitment bars or the reverse.
A loading transition may be used if the read contract does not already contain
both projections, but stale responses cannot restore the old projection.

### 10.3 No Business Mutation

Changing Execution/Commitment:

- does not persist Project, WBS, Task, Dependency, or filter data;
- does not invoke the scheduler;
- does not advance schedule version;
- does not alter Actual Date or allocation;
- does not change the active saved filter.

---

## 11. Recursive Date and Effort Aggregation

### 11.1 Reuse of Summary Semantics

Project and Group grid values reuse the recursive confirmed-descendant principles
from US-4.3. Home must not create writable aggregate fields or a competing
aggregation algorithm.

### 11.2 Selected-Projection Date Range

For a Project or Group:

```text
Aggregate Start = earliest selected-projection Start among descendant Tasks
                  with a complete Start/End pair
Aggregate End   = latest selected-projection End among descendant Tasks
                  with a complete Start/End pair
```

Rules:

- traversal is recursive through every descendant Group;
- only executable Tasks contribute dates;
- Start and End may originate from different Tasks;
- a Task with an incomplete or missing pair does not contribute dates;
- a Project/Group with no scheduled descendant Task displays blank Start/End and
  no summary bar;
- when only part of the descendant scope is scheduled, aggregate dates still
  use the scheduled subset and an incomplete-schedule indicator is shown;
- Actual Date does not replace the chosen projection in Home;
- unconfirmed preview or unsaved Task draft never contributes.

### 11.3 Recursive Effort

```text
Aggregate Effort = sum of non-null Effort for every descendant Task
```

Rules:

- use integer minutes as calculation source;
- nested Groups contribute only through descendant Tasks;
- Task order does not change the sum;
- null Effort is excluded and disclosed;
- no persisted aggregate column is introduced solely for Home.

---

## 12. Timeline Header and Working-Day Sequence

### 12.1 Daily Calendar Columns

- Daily is the only timeline scale.
- Every calendar date inside Visible Range has one timeline column.
- Saturday, Sunday, and Public Holiday columns remain visible.
- Non-working dates use a distinct background/stripe that remains perceivable
  behind bars and satisfies contrast requirements.
- Public Holiday description is available through tooltip and keyboard-
  accessible text.
- A member Capacity Override does not mark the global date as a holiday because
  it is member-specific.

### 12.2 Two Header Rows

Timeline header contains:

1. working-day sequence;
2. actual calendar date.

The first row displays only integers:

```text
1, 2, 3, 4, ...
```

It does not display `D1`, `Day 1`, or manday terminology.

### 12.3 Working-Day Anchor

The Working-Day Anchor is the earliest Start among all selected Tasks with a
complete Start/End pair for the selected projection.

- Execution view uses earliest Execution Start.
- Commitment view uses earliest Commitment Start.
- The first working date on or after the anchor displays `1`.
- Each subsequent working date increments the value by one.
- Saturday, Sunday, and Public Holiday display no number and do not increment
  the sequence.
- The default Visible Range begins on the earliest selected Task Start, so
  calendar dates before the Working-Day Anchor are not added merely as leading
  padding.
- The sequence is global across all selected Projects; it does not restart per
  Project.
- Changing Visible Range does not change the anchor or renumber the schedule.
- Changing selected projection may move the anchor and renumber the header.
- If no selected Task has a complete selected-projection range, every first-row
  cell is blank.

Example when the anchor is Friday:

```text
Calendar: Monday | Tuesday | Wednesday | Thursday | Friday | Saturday | Sunday | Monday
Sequence:        |         |           |          | 1      |          |        | 2
```

### 12.4 Public Holiday Precedence

- Public Holiday remains non-working even when it falls on a weekday.
- A Public Holiday on Saturday or Sunday remains one calendar column and one
  non-working date; it does not consume or duplicate a sequence number.
- Holiday updates reflected in Home recompute the visible sequence without
  rewriting persisted Task dates by the Home feature itself. Any schedule
  mutation remains owned by the capacity/scheduler stories.

---

## 13. Visible Timeline Range

### 13.1 Default Auto-Range

When at least one selected Task has a complete range in the selected projection:

```text
Default From = earliest selected-projection Start
Default To   = latest selected-projection End + 7 calendar days
```

Only the trailing padding is retained. It uses calendar days, not working days.
No leading padding is added before the earliest Start.

### 13.2 No Scheduled Range

- If selected Projects exist but every Task is unscheduled for the selected
  projection, display the current calendar month.
- If no Project is selected, display the current calendar month with an empty
  Project state.
- If there are no active Projects, display the current calendar month with an
  active-Project empty state.

### 13.3 Auto-Range Only

- Home does not expose manual From/To controls.
- Project or Role filter changes recalculate the visible range from the Tasks
  that remain visible after filtering.
- Execution/Commitment changes recalculate the range from that projection.
- Automatic range recalculation does not run scheduling or persist business
  data.
- Daily columns must use bounded rendering so a long derived range does not
  require one permanent DOM element per Task/date combination.

---

## 14. Timeline Bars

### 14.1 Task Bar

- A Task with a complete selected-projection pair displays one bar from inclusive
  Start through inclusive End.
- A one-day Task occupies one date column.
- Calendar columns between Start and End remain part of the bar span even when
  they are weekends or Public Holidays.
- The bar represents scheduled range, not per-day allocated hours.
- A Task without a complete range has no bar and exposes its unscheduled reason
  where available.

### 14.2 Project and Group Summary Bars

- Project and Group use a distinct summary-bar form, inspired by conventional
  Gantt summary tasks.
- Summary bar spans recursive aggregate Start through End.
- It does not imply that the Project or Group consumes capacity.
- It remains visually distinguishable from an executable Task bar without
  relying only on colour.
- A Project/Group without scheduled descendants has no summary bar.

### 14.3 Read-Only Contract

Every bar is read-only regardless of Automatic Scheduling or Project status:

- no drag;
- no resize;
- no inline date editor;
- no dependency handle;
- no mutation on pointer movement;
- clicking opens the owning existing form when the actor is allowed to view it.

Automatic Scheduling OFF does not make Home bars editable. Manual date editing
continues through the existing Task form and US-4.1/US-6.1 rules.

---

## 15. Dependency Visualization

### 15.1 Included Relations

- Display effective Finish-to-Start dependency arrows from US-5.1.
- Manual-only, automatic-only, and shared ownership refer to the same visible
  endpoint pair and must not produce duplicate arrows.
- Dependency source is available through tooltip/accessibility text and is not
  communicated only by colour.
- Cross-Project dependency may be shown when both endpoint Projects and Tasks are
  currently rendered.

### 15.2 Visibility Rules

An arrow is rendered only when:

- both endpoint Projects are selected and active;
- both endpoint Task rows are expanded/visible;
- both endpoint bars exist for the selected projection;
- required geometry can be represented inside the current rendered timeline.

When a parent is collapsed, a Project is filtered out, or an endpoint is outside
the rendered horizontal range:

- do not draw a dangling arrow;
- do not redirect it to a Project/Group summary bar;
- restore it when both endpoints become renderable.

### 15.3 Read-Only Contract

- Arrow cannot be created, moved, or deleted from Home.
- Clicking an arrow may expose read-only relation details but cannot mutate it.
- Dependency editing remains inside the existing Task form owned by US-5.1.
- Home does not add new dependency type semantics.

---

## 16. Row Interaction and Existing Forms

### 16.1 Direct Row Actions

The only Home row-level creation actions are:

- an icon-only **Add Task** action on an eligible Project row;
- an icon-only **Add Child** action on an eligible Group or Task row.

The two actions use visually distinct icons, retain explicit accessible names
and tooltips, and invoke the same application use cases and confirmations as
Project Structure.

### 16.2 Open Existing Forms

- Clicking a Project row/summary bar opens the existing Edit Project form.
- Clicking a Group row/summary bar opens the existing Group form.
- Clicking a Task row/bar opens the existing Task form.
- Home must reuse the form contract rather than fork field validation or
  mutation orchestration.
- Available form actions continue to follow the owning stories, including
  dependency management, Actual Date, Reopen, Delete, manual timeline, and
  Locked restrictions.

### 16.3 Add Child Conversion

When Add Child targets an executable Task:

- US-4.1 executable-to-Group conversion rules apply;
- required warning/confirmation is shown;
- executable fields and dependencies move according to existing rules;
- cancellation leaves hierarchy and Home projection unchanged;
- scheduling and impact coordination remain atomic.

### 16.4 Projection Refresh

After a successful form mutation:

- refresh every selected Project whose visible projection is affected;
- include cross-Project transitive impacts from US-6.2;
- preserve unaffected expanded/collapsed rows where IDs still exist;
- do not hard reload the application;
- do not let an older read response overwrite the confirmed mutation.

---

## 17. Home Configuration

### 17.1 Configure Gantt Modal

The compact Gantt-header action is an icon-only **Configure Gantt** control with
an accessible name and tooltip. It opens one modal form containing saved-filter,
projection, Project, and Role sections. Applying the modal updates the Gantt
atomically; Cancel preserves the currently applied configuration.

The Project checklist contains all current active Projects:

- Open and Locked Projects are available;
- Closed and deleted Projects are absent;
- checklist entries display Project Name and a non-colour-only Locked indicator;
- search by Project Name is available;
- Select All selects every active Project;
- Clear All clears every Project;
- empty selection is valid and displays an empty chart state;
- Project order follows the authoritative active Project order.

### 17.2 Default Selection

On first Home load, use the system view:

```text
All Active Projects
```

It selects every current Open and Locked Project.

- New active Projects automatically appear selected when this system view is
  reopened or refreshed.
- The system view is not a persisted row and cannot become stale from stored
  Project IDs.
- It cannot be deleted.
- It cannot be updated with Save.
- A modified checklist based on it may be persisted through Save As.

### 17.3 Configuration Draft Behaviour

- Opening Configure Gantt initializes the draft from the currently applied saved
  filter, projection, Project selection, and Role selection.
- Selecting another system or saved filter inside the modal replaces the Project
  checklist draft but does not update the chart before Apply.
- Projection, Project, and Role draft changes update the chart atomically only
  after Apply.
- Cancel discards the modal draft and preserves the currently applied chart.
- Save updates the selected persisted filter's Project IDs without implicitly
  applying the remaining projection or Role draft.
- Save As persists the Project checklist under a new Name and applies the current
  configuration after the create succeeds.
- Selecting another filter inside the modal may discard unsaved checklist draft
  without confirmation because it is a low-impact local draft.
- Discarding a draft never changes backend data.


### 17.4 Role Filter

- Role options derive from Task Role IDs in the selected active Projects.
- A Task without Role is represented by a distinct `No role` option.
- Selecting one or more Roles displays only matching Tasks plus the Project and
  Group ancestors required to preserve hierarchy.
- Project and Group Start, End, Effort, Role, incomplete-effort, and
  incomplete-schedule values are recomputed from the matching descendant Tasks.
- Dependencies render only when both endpoint Tasks remain visible.
- Role selection is current-session UI state and is not persisted in the
  existing saved-filter model.
- Select All and Clear All are available independently for Projects and Roles.

---

## 18. Global Saved Filters

### 18.1 Persistence and Visibility

There is no login or user identity in current scope. Saved filters are therefore
backend-persisted and global:

- every application user sees the same names;
- every user can open, Save, Save As, and delete them;
- no Created By or Owner field is required;
- permissions and per-user isolation are deferred.

The selector displays saved filters by **name only**. It does not expose Project
IDs, counts, timestamps, or version in the normal list UI.

### 18.2 Persisted Model

A saved filter stores at minimum:

```text
ID
Name
Selected Project IDs
Created At
Updated At
Version
```

Rules:

- Project IDs are authoritative; names and WBS numbers are not persisted as
  references.
- Empty Selected Project IDs are valid.
- Stored IDs may temporarily include Projects that later become Closed or
  deleted until the next Save or Save As.
- `Version` supports simple optimistic concurrency; no merge UI is required.

### 18.3 Naming

- Trim leading and trailing whitespace.
- Reject empty normalized Name.
- Name is unique case-insensitively.
- `Team A`, `team a`, and ` TEAM A ` conflict.
- Reserved system-view name `All Active Projects` and its case-insensitive
  normalized equivalent are rejected.
- A practical maximum length must be enforced consistently by domain, database,
  API, and UI; use the existing project naming convention unless a shared
  smaller limit is already authoritative.

### 18.4 Open Saved Filter

When a saved filter is selected:

1. load its stored Project IDs;
2. intersect them with current active Projects;
3. automatically check the surviving active Projects;
4. omit Closed/deleted Projects without warning;
5. show empty state when none survive;
6. keep Save and Save As available even when the draft has not changed.

Opening does not automatically rewrite the saved filter.

### 18.5 Save

Save updates the currently active persisted saved filter:

- Name remains unchanged.
- Persist only Project IDs that are still active at confirmation time.
- Remove Closed/deleted IDs automatically.
- Empty result is valid.
- Save is available even when the draft appears unchanged.
- Save on `All Active Projects` is unavailable.
- Successful Save advances Version and Updated At.
- Stale Version is rejected with a safe conflict; UI reloads/reopens the latest
  filter rather than silently overwriting it.
- Failure preserves the current draft and active filter selection.

### 18.6 Save As

Save As creates a new saved filter:

- prompt for a new valid unique Name;
- use the current checklist draft;
- persist only Projects still active at confirmation time;
- allow empty result;
- do not change the source saved filter;
- after success, the new filter becomes active;
- duplicate Name or persistence failure preserves the current draft and source
  active filter.

Save As is available from both `All Active Projects` and a saved filter.

### 18.7 Delete

- A saved filter can be deleted after confirmation.
- `All Active Projects` cannot be deleted.
- Delete uses current Version to avoid deleting a filter changed concurrently.
- After deleting the active filter, Home returns to `All Active Projects` and
  selects all current active Projects.
- Deleting a non-active filter leaves current checklist and chart unchanged.
- Delete failure leaves the filter present and current state intact.

### 18.8 Ordering

- `All Active Projects` appears first.
- Persisted saved-filter names use deterministic case-insensitive alphabetical
  order unless the repository already has a shared deterministic list-order
  convention that must be reused.
- Created/Updated timestamps are not used as hidden user-visible priority.

---

## 19. Empty, Loading, and Error States

Home distinguishes at least these states:

### 19.1 No Active Projects

- Show an empty state explaining that no Open or Locked Project is available.
- Project checklist is empty.
- Timeline displays current month.
- Saved filters remain manageable; they may resolve to empty.

### 19.2 No Selected Projects

- Show an empty state explaining that no Project is selected.
- Keep checklist and saved-filter actions available.
- Timeline displays current month.

### 19.3 Selected Projects Without Scheduled Tasks

- Show Project/WBS rows.
- Start/End and bars are blank where no complete selected-projection pair exists.
- Working-day sequence is blank.
- Timeline displays the current month when no filtered Task has a complete scheduled range.
- Expose unscheduled reasons per Task where available.

### 19.4 Loading

Project list/filter metadata, saved filters, and Gantt projection may load
independently, but the UI must not combine incompatible snapshots. Use localized
loading states that preserve already confirmed controls and avoid whole-page
flicker where possible.

### 19.5 Recoverable Error

- A failed Gantt read shows retry without discarding active filter draft.
- A failed saved-filter operation preserves user input/draft.
- A failed form mutation follows the owning form's error/rollback contract.
- Infrastructure details are not exposed.
- Stale or concurrent conflict uses actionable safe copy.

---

## 20. Read Model and API Contract

### 20.1 Dedicated Portfolio Projection

Home requires a dedicated read composition or equivalent bounded set-based API.
It must not implement:

```text
list Projects
→ request WBS once per Project
→ request assignee once per Task
→ request dependency once per Task
→ request holiday once per date
```

The read contract returns one coherent projection for selected active Projects,
including enough data for:

- Project identity, Name, status, priority/order, and version;
- ordered recursive WBS nodes and stable IDs;
- Group/Task distinction;
- Assignee display identity/name;
- Effort minutes;
- Execution Start/End;
- Commitment Start/End;
- unscheduled reason/projection state;
- effective dependency endpoint pairs and ownership source;
- Public Holidays required by the current rendered range;
- schedule/projection versions required to reject stale responses.

The exact endpoint name is an implementation decision, but observable fields,
ordering, version safety, and bounded query behaviour are mandatory.

### 20.2 Active Project and Saved Filter APIs

Provide backend operations for:

- listing active Project filter options;
- listing saved filters;
- creating saved filter;
- updating selected Project IDs with Version;
- deleting saved filter with Version.

Backend revalidates Project status during Save and Save As. The client cannot
persist a Closed/deleted Project merely by sending a stale ID.

### 20.3 Data Integrity

- Saved filter Name has normalized case-insensitive uniqueness.
- Project ID references do not require hard foreign-key deletion blocking if the
  chosen storage model intentionally tolerates stale IDs, but query and save
  semantics must remain deterministic.
- Update/Delete use optimistic concurrency or an equivalent safe mechanism.
- Create/Update/Delete are atomic.
- Partial selected-ID persistence is not allowed.
- Safe structured errors distinguish validation, duplicate Name, not found,
  conflict, and unexpected failure.

### 20.4 Query and Index Review

Before implementation completion:

- inspect read-query shapes for selected multi-Project WBS traversal;
- prevent N+1 Project, WBS, Member, Dependency, and Holiday access;
- verify deterministic ordering at the database/application boundary;
- review saved-filter normalized-name uniqueness and update/delete point lookup;
- measure PostgreSQL plans for realistic selected-Project and WBS cardinality;
- add only indexes justified by the measured production query shapes;
- document query-review evidence under the repository standard.

---

## 21. State, Cache, and Concurrency

### 21.1 Request Identity

Gantt cache/request identity includes all inputs that can change the response,
including at minimum:

- selected Project IDs as an order-independent normalized set;
- relevant active Project/schedule versions;
- selected projection if the backend response is projection-specific;
- valid Visible Range when Holiday/bar clipping is server-composed.

### 21.2 Stale Response Protection

- Changing filter, selected projection, or range invalidates or supersedes the
  previous request.
- An older response cannot replace a newer filter/projection/range state.
- Successful mutations advance/invalidate shared scheduling projection state as
  required by US-6.1/US-6.2.
- Home must participate in the existing cross-feature projection clock or an
  equivalent single authoritative stale-response mechanism.
- Do not create a separate Home-only version model that can disagree with
  Project/WBS/Dependency gateways.

### 21.3 Concurrent Saved Filter Changes

- Global filter Save/Delete includes Version.
- Stale mutation is rejected, not silently last-write-wins.
- UI need only show a simple conflict and reopen/reload the latest filter.
- No field-level merge is required.
- A stale saved-filter response cannot overwrite a newly selected filter.

### 21.4 Form Mutation Coordination

- Disable duplicate submission while an opened form command is in progress.
- Scheduler/impact preview and confirmation remain owned by existing forms.
- Successful cross-Project mutation refreshes all affected selected rows.
- If an affected Project is not selected, its schedule version still invalidates
  stale cached data for future selection.
- Transaction failure preserves pre-command Home projection after refresh.

---

## 22. Performance and Rendering

### 22.1 Virtualization

A daily multi-Project Gantt can produce `rows × dates` cells. The implementation
must virtualize or equivalently bound both dimensions where necessary:

- horizontal date columns;
- vertical Project/WBS rows;
- dependency geometry recomputation.

Do not permanently render one interactive DOM cell for every row/date pair over
large ranges.

### 22.2 Stable Rendering

- Use stable entity IDs as keys.
- Expanding/collapsing one subtree must not remount unrelated Project rows.
- Horizontal scroll must not trigger business-data refetch on every pixel.
- Dependency lines should be recomputed from visible endpoints only.
- Long Task names and deep hierarchy must truncate/wrap predictably without
  changing row/timeline alignment.

### 22.3 Practical Scale Tests

Implementation evidence must include representative tests or measured review for:

- multiple Projects with duplicate visible WBS numbers;
- deep hierarchy;
- thousands of visible WBS rows or the agreed realistic upper bound;
- multi-year daily range through virtualization;
- cross-Project dependencies;
- rapid filter/projection changes.

A browser test does not need to create production-maximum data if a lower
fixture plus explicit virtualization assertions proves bounded rendering.

---

## 23. Accessibility and Responsive Behaviour

### 23.1 Semantic Structure

- Project Grid uses table/treegrid semantics appropriate to hierarchical rows.
- Expand/collapse state is programmatically exposed.
- Project, Group, and Task type is available to assistive technology.
- WBS number, Name, Role, Assignee, dates, Effort, status, and scheduling state have
  understandable labels.
- Timeline and dependency visualization have a non-visual equivalent sufficient
  to understand each Task's Start/End and predecessor relationships.

### 23.2 Keyboard

Keyboard users can:

- reach the Configure Gantt icon in logical order;
- open the configuration modal and its saved-filter, projection, Project, and
  Role controls;
- select/deselect Projects;
- expand/collapse Project and Group rows;
- focus a row and open its form;
- invoke Add Task/Add Child where available;
- scroll the timeline without losing focused row context;
- operate Save, Save As, and Delete dialogs.

### 23.3 Visual Communication

- Weekend/Public Holiday, Locked status, row type, unscheduled state,
  incomplete schedule, incomplete Effort, and dependency source do not rely
  only on colour.
- Focus remains visible over both grid and timeline.
- Text and bars meet contrast requirements.
- Tooltips also have keyboard-accessible disclosure.

### 23.4 Responsive

- Desktop is the primary planning layout.
- On narrow viewports, the product may prioritize the grid and require explicit
  horizontal timeline scrolling; it must not hide required actions or make
  dialogs unusable.
- Grid/timeline synchronization remains correct after viewport resize.
- Existing shared Dialog responsive behaviour is reused.

---

## 24. Acceptance Criteria

### Navigation and Composition

1. Home is the first navigation menu and opening the frontend root renders Home.
2. Project Structure remains available and functional as an alternative page.
3. Home shows a synchronized left Project Grid and right daily Timeline.
4. Header and row alignment remain synchronized through vertical/horizontal scrolling.

### Project and WBS Rows

5. Every selected Open/Locked Project has a visible Project row; Closed Projects are absent.
6. Project WBS cell is blank while descendants start at `1` for each Project.
7. WBS numbering reflects recursive hierarchy and restarts per Project.
8. Project and Group rows expand/collapse without changing persisted hierarchy.
9. Project/Group Assignee is blank; Task Assignee shows current member or blank.
10. Project/Group Effort equals recursive known Task Effort; Task shows direct Effort.
11. Null Task Effort is blank and incomplete-effort state is disclosed without invented values.

### Projection and Aggregation

12. Execution view shows only Execution dates/bars and Commitment view shows only Commitment dates/bars.
13. Projection switch updates grid values, bars, summaries, sequence, and dependency geometry coherently.
14. Projection switch performs no business mutation or scheduler invocation.
15. Project/Group Start is recursive earliest complete selected-projection Start.
16. Project/Group End is recursive latest complete selected-projection End.
17. Partially scheduled scopes show aggregates from scheduled Tasks plus incomplete-schedule disclosure.
18. No scheduled descendants produce blank dates and no summary bar.
19. Actual data is not displayed as a Home projection.

### Timeline Header and Range

20. Timeline renders one column for every calendar date in the automatically derived Visible Range.
21. Weekend and Public Holiday columns remain visible and are marked non-working.
22. Header first row displays `1, 2, 3...` only on working dates from the Working-Day Anchor.
23. Dates before the anchor and all non-working dates display no sequence number.
24. Public Holiday does not increment the sequence, including when it overlaps a weekend.
25. Default range begins on the earliest Start and ends seven calendar days after the latest End; no leading seven-day padding is added.
26. No schedule or no selection defaults to current calendar month.
27. Home exposes no manual From/To controls.
28. Project, Role, or projection changes recalculate the automatic range without changing persisted schedule data.

### Bars and Dependencies

29. Complete Task ranges display inclusive read-only bars; one-day Task occupies one date.
30. Task bar spans intervening non-working dates without claiming daily allocation.
31. Project/Group use visually and semantically distinct read-only summary bars.
32. No bar can be dragged, resized, or edited regardless of Automatic Scheduling state.
33. Effective dependencies display as one read-only arrow per endpoint pair.
34. Manual/automatic/shared source is accessible and does not create duplicate arrows.
35. Hidden, collapsed, filtered, unscheduled, or out-of-range endpoints produce no dangling arrow.
36. Dependency cannot be created, deleted, or retargeted from Home.

### Forms and Mutations

37. Project row/bar opens existing Edit Project form and returns to Home after Save or Close when opened from Home.
38. Group/Task row/bar opens the existing WBS form and returns directly to Home after Save or Close when opened from Home; the intermediate Project Structure view is not exposed.
39. Eligible Project row provides a distinct icon-only Add Task control with an accessible name and tooltip, uses the existing WBS mutation flow, and returns to the refreshed Home after success.
40. Eligible Group/Task row provides a distinct icon-only Add Child control with an accessible name and tooltip, uses existing conversion and confirmation rules, and returns to the refreshed Home after success.
41. Locked Project hides/disables forbidden planning actions while preserving US-6.2 factual exceptions in the opened form.
42. Successful mutation refreshes all affected selected Projects without hard reload.
43. Failed mutation/rollback does not leave Home showing unconfirmed hierarchy or schedule.
44. Stale Home response cannot overwrite a newer confirmed mutation.

### Project Selection

45. `All Active Projects` is the default system view and selects all Open/Locked Projects.
46. System view automatically includes newly active Projects and excludes Closed Projects.
47. The icon-only Configure Gantt action opens one modal form and does not push the Gantt vertically.
48. Saved-filter, projection, Project, and Role draft changes are applied atomically through Apply; Cancel preserves the applied chart.
49. Role filtering retains ancestors, recomputes their aggregates from matching Tasks, and removes dangling dependency arrows.

### Saved Filters

50. The saved-filter selector is available only inside Configure Gantt and shows system/saved filter names without Project detail metadata.
51. Save As from system or saved view creates a globally visible named filter from current active selection.
52. Saved-filter Name is trimmed, required, case-insensitively unique, and cannot use reserved system-view Name.
53. Opening a saved filter checks only currently active stored Projects and silently ignores Closed/deleted IDs.
54. If all stored Projects are inactive, opening the filter shows empty selection without warning.
55. Save remains available on an active saved filter even when draft is unchanged.
56. Save replaces selected IDs with the current active checklist and removes inactive IDs.
57. Save As leaves the source filter unchanged and activates the newly created filter.
58. Empty saved filters are valid for Save and Save As.
59. Delete confirmation removes a saved filter; deleting active filter returns to `All Active Projects`.
60. System view cannot be saved over or deleted.
61. Stale Save/Delete is rejected by Version and does not silently overwrite concurrent changes.
62. Saved-filter failure preserves current draft and does not corrupt the selector state.

### Data, Performance, and Accessibility

63. Home uses a bounded portfolio read model and does not issue per-Project/per-Task N+1 requests.
64. Read and saved-filter queries have deterministic ordering and completed query/index review evidence.
65. Date-only values do not shift because of browser timezone.
66. Rapid Project filter, Role filter, projection, or auto-range changes cannot restore stale data.
67. Large row/date ranges use bounded rendering/virtualization rather than permanent full cell matrices.
68. Hierarchical rows, status, schedules, holidays, dependencies, controls, and errors are keyboard and assistive-technology accessible.
69. Loading, empty, retry, conflict, and failure states preserve valid user context.
70. Backend and frontend validation suites required by project architecture pass.

---

## 25. Mandatory Test Scenarios

### 25.1 Hierarchy and Multi-Project

- Two selected Projects both contain WBS `1` and `1.1`; rows remain uniquely
  associated with their Project.
- Project with no WBS.
- Deep Group hierarchy.
- Reorder/move completed in Project Structure updates Home WBS numbering.
- Collapse Project and nested Group, then refresh affected projection.
- Long names and narrow viewport preserve row/bar alignment.

### 25.2 Projection and Aggregation

- Execution and Commitment ranges differ for the same Task.
- Task scheduled in Execution but unscheduled in Commitment.
- Group with scheduled and unscheduled descendants.
- Project with multiple top-level WBS roots.
- Task without Effort mixed with Tasks having fractional-hour Effort.
- Completed Task still contributes total Effort.
- Unconfirmed schedule preview does not change Home summary.

### 25.3 Calendar and Working-Day Sequence

- Anchor begins Friday; Saturday/Sunday blank; Monday is `2`.
- Visible range begins before anchor.
- Visible range begins on weekend.
- Weekday Public Holiday between numbered days.
- Multi-day Public Holiday.
- Holiday overlaps weekend.
- Projection switch moves anchor.
- All selected Tasks unscheduled.
- Calendar range crosses month/year and timezone boundaries.

### 25.4 Bars and Dependencies

- One-day Task.
- Task range crosses weekend/holiday.
- Cross-Project dependency with both Projects visible.
- One endpoint Project filtered out.
- One endpoint hidden by collapsed Group.
- Automatic + manual shared relation renders one arrow.
- Dependency endpoint has no selected-projection bar.
- Attempts to drag/resize/create dependency produce no mutation.

### 25.5 Forms and Lifecycle

- Add root Task from Open Project row.
- Add Child to Group.
- Add Child to executable Task with confirmed conversion.
- Cancel conversion.
- Open and edit Task from row/bar; scheduler impact moves another selected Project.
- Edit causes impacted Locked Project and follows US-6.2 blocking.
- Locked Project row opens read-only planning form with allowed Actual Date path.
- Project becomes Locked while form is open.
- Mutation failure rolls back and refreshes confirmed data.

### 25.6 Saved Filters

- Initial system view selects every Open/Locked Project.
- Save As `Team A`, then open and verify checklist.
- Duplicate `team a` is rejected.
- Modify `Team A`, Save, reopen, verify replacement.
- Modify `Team A`, Save As `Team B`, verify source unchanged.
- Save unchanged filter succeeds.
- Save and Save As empty selection.
- One stored Project becomes Closed before opening.
- Project becomes Closed after opening but before Save; backend removes it.
- All stored Projects become Closed; empty state.
- Project is renamed; filter remains valid by ID.
- Project is deleted.
- New Project appears in system view but not existing saved filters.
- Two clients Save same version; one succeeds and one receives conflict.
- Active filter deleted; system view restored.
- Switching filters discards unsaved draft without prompt.

### 25.7 State and Performance

- Rapid A→B filter switch with A response arriving last.
- Rapid Execution→Commitment→Execution switch.
- Range request returns after filter changed.
- Cross-feature WBS/dependency mutation invalidates cached Home projection.
- Large fixture verifies virtualized/bounded row/date rendering.
- Dependency geometry recomputes only for visible rows.

### 25.8 Accessibility

- Keyboard opens filter, checks Projects, and Save As.
- Keyboard expands Project/Group and opens Task form.
- Screen reader receives row type, WBS, status, dates, Effort, and
  scheduled/unscheduled state.
- Public Holiday and dependency source are understandable without colour.
- Focus survives projection refresh when the focused entity still exists.

---

## 26. Three-Level Confidence Requirements

Every Acceptance Criterion must meet the repository's Three-Level Confidence
Standard.

### 26.1 Code Inspection

Inspect at minimum:

- Home routing/navigation and fallback;
- feature boundaries and reuse of existing forms/use cases;
- recursive row composition and date/Effort aggregation;
- date-only and working-day sequence calculation;
- active Project and saved-filter domain invariants;
- normalized unique Name and optimistic concurrency;
- portfolio read query and N+1 prevention;
- cache/request identity and stale-response handling;
- virtualization and dependency geometry boundaries;
- lifecycle, Closed exclusion, and Locked restrictions;
- accessibility and responsive states.

### 26.2 Unit or Integration Tests

Use the lowest meaningful boundaries for:

- pure WBS numbering, recursive aggregation, range, and working-day sequence;
- saved-filter domain validation and conflict handling;
- repository uniqueness, atomic update/delete, and stale Project IDs;
- set-based portfolio projection mapping and ordering;
- frontend request/cache race behaviour;
- row virtualization and visible dependency selection;
- component interaction and form integration.

### 26.3 Acceptance-Level Tests

Provide public-boundary evidence for:

- root Home navigation;
- active Project selection and portfolio rendering;
- Execution/Commitment switching;
- daily header/holiday behaviour;
- Add Task/Add Child/open form workflows;
- read-only bars/dependencies;
- saved-filter Save/Save As/Delete and concurrency errors;
- Closed/Locked lifecycle behaviour;
- stale-response prevention after mutation.

Trace each AC to production files, unit/integration tests, and highest practical
acceptance workflow in the completion evidence.

---

## 27. Technical and Documentation Impact

Implementation must assess and update:

- frontend navigation and root-route composition;
- new Home/Gantt frontend feature boundary;
- shared Project/WBS form composition without circular feature dependencies;
- portfolio read application/query service and transport contract;
- saved-filter domain/application/persistence/transport/UI;
- database migration for saved filters and normalized uniqueness/version;
- Project/WBS/Member/Dependency/Public Holiday read composition;
- cache and schedule projection clock integration;
- frontend design-system needs for treegrid, split grid/timeline, bars, holiday
  cells, tooltip, empty/loading/error states, and virtualization;
- project architecture and product context;
- US-4.1, US-4.3, US-5.1, and US-6.1 cross-story ownership wording;
- API documentation and query/index review evidence.

`AGENTS.md` changes only if implementation discovers a genuinely reusable,
project-wide agent rule; this story alone does not require one.

---

## 28. Locked Product Decisions

1. Home is the first menu and default root page, and it displays no breadcrumb.
2. Home displays a portfolio Gantt inspired by GanttPRO but governed by SchedMind rules.
3. Project row remains visible; only its WBS `0` label is omitted.
4. Descendant WBS numbering starts at `1` and restarts per Project.
5. Grid columns are WBS, Name, Role, Assignee, Effort, Start, and End.
6. Project/Group Role is the distinct set of matching descendant Task Roles; Project/Group Assignee is blank.
7. Project/Group dates and Effort aggregate recursively across all descendant Tasks.
8. Execution/Commitment is the only projection selector; Actual is absent.
9. Gantt cells, bars, and dependency arrows are always read-only.
10. Home has no inline editing.
11. Direct row creation actions are distinct icon-only Add Task and Add Child controls with accessible names and tooltips; other mutations use existing forms.
12. Dependency arrows are read-only and dependency editing remains in Task form.
13. Every calendar date is shown at daily scale, including weekends and Public Holidays.
14. Header row one displays bare working-day numbers `1, 2, 3...`.
15. Working-day numbering anchors to earliest scheduled Start for the selected projection.
16. Dates before anchor and non-working dates have blank sequence cells.
17. Default range begins at earliest Start and ends seven calendar days after latest End, with no leading padding.
18. No schedule or no selection uses current calendar month.
19. Visible range is automatic only and follows the currently filtered Task set.
20. Closed Projects are inactive and absent; Open and Locked are eligible.
21. Default `All Active Projects` selects every Open and Locked Project and is displayed as the Home title.
22. Home has no persistent filter toolbar; saved filter, projection, Project, and Role configuration are consolidated in one icon-triggered modal form.
23. Saved filters are backend-persisted and global because login is absent.
24. Saved-filter selector displays names only.
25. Saved filters support Save, Save As, and Delete.
26. Save updates selected Project IDs under the same Name.
27. Save As creates a new Name and leaves the source unchanged.
28. Empty saved filters are valid.
29. Filter Name is trimmed, required, case-insensitively unique, and cannot use the system-view Name.
30. Closed/deleted Project IDs are silently omitted when a filter opens and removed on Save/Save As.
31. If every stored Project is inactive, show empty state without informational warning.
32. Switching filters discards unsaved checklist draft without confirmation.
33. Project Structure remains available; removal requires a separate requirement.
34. Daily scale is the only zoom level in this story.
35. Saved filters do not store Role selection, projection, range, expansion, scroll, task selection, or search state.
36. Global Save/Delete uses simple optimistic concurrency; no merge UI is required.
37. Filter is modal, Project Grid columns are individually resizable, and the Timeline always consumes the remaining width without overlap.
38. Project, Task, Group, Add Task, and Add Child flows opened from Home return directly to the refreshed Home workspace.
