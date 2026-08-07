# US-7.1 — Home Portfolio Gantt Workspace

> **Product decision update — US-4.4:** The shared Group dialog opened from Home now includes Group scheduling inheritance/override and eligible Lock/Reopen controls in addition to rename and the US-4.3 summary. Home remains the route/background; no separate Group settings page is introduced. Row mutation eligibility follows effective Group/Project lifecycle.

> **Authority:** This story owns the Home navigation entry, portfolio Gantt read
> model, Project selection and global saved-filter behaviour, WBS/timeline
> presentation, canonical Home orchestration of Project/WBS actions, and removal
> of the standalone Project Structure entry point. It does not replace the
> mutation rules owned by US-3.1, US-3.3, US-4.1, US-4.2, US-5.1, US-6.1, or
> US-6.2. US-8.1 separately owns the Sprints destination under the Project
> navigation group; Home remains the first/default route.
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

The product needs one primary workspace that combines multiple active Projects,
preserves their individual WBS hierarchies, visualizes Execution or Commitment
dates on one daily timeline, and exposes the established Project/WBS actions
without requiring a separate Project Structure page.

The Home Gantt is a **read-only schedule projection plus the canonical WBS
management workspace**. It is not a second scheduler, a second WBS aggregate,
or a second mutation implementation. Add Sibling, Add Child, Project edit,
Task/Group edit, Dependency, completion, Reopen, move, reorder, delete,
lifecycle, impact preview, validation, and rollback continue to use the same
application contracts and business invariants as their owning stories.

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
- Make Home the canonical presentation surface for active Project WBS
  management.
- Remove the standalone **Project Structure** entry point and intermediate page
  from the Projects workflow after Home reaches the required capability parity.
- Display Project, Group, and Task rows in a hierarchical left grid.
- Display WBS, Name, Role, Assignee, Effort, Start Date, and End Date columns.
- Remember the latest Project Grid column widths as a browser-local Home layout
  preference and restore them when Home is opened again.
- Switch the entire workspace between Execution and Commitment projections.
- Remember the last successfully applied projection as a browser-local Home
  preference and restore it when Home is opened again.
- Display Project and Group recursive summary ranges and recursive Effort.
- Display daily calendar columns including weekends and Public Holidays.
- Display a three-row timeline header with working-day sequence, month/year,
  and calendar date.
- Display read-only Task bars, Project/Group summary bars, and dependency arrows.
- Expand and collapse Project and Group rows.
- Provide one contextual **Collapse All / Expand All** action in the Name-column
  header and remember collapse state as a browser-local Home preference.
- Open shared Project, Group, and Task dialogs directly over Home by activating
  the row Name.
- Replace the single add icon with labelled Add Sibling/Add Child row actions
  while keeping the overflow trigger independently anchored at the right edge
  of the row.
- Provide one row overflow menu for eligible Project lifecycle, WBS reorder,
  Move to, and Delete actions.
- Provide Move Up/Down for adjacent siblings and Move to for parent changes,
  using the existing US-4.1 use cases.
- Provide a dedicated leading drag handle for eligible Task and Group rows so a
  user can reorder a complete WBS subtree before or after another sibling
  without opening the overflow menu.
- Provide Project lifecycle actions on both Projects and the active Project row
  in Home, using the same commands and confirmations.
- Select saved filter, projection, Projects, and Task Roles through the Configure
  Gantt modal workflow.
- Provide a system view named `All Active Projects`.
- Create, open, update, duplicate, and delete globally persisted named filters.
- Use an automatic visible range derived from the currently filtered Task set.
- Preserve scheduler, lifecycle, confirmation, transaction, concurrency, and
  stale-projection rules from the owning stories.
- Provide backend read and persistence contracts, loading/error/empty states,
  accessibility, responsive behaviour, and performance safeguards.

### 4.2 Out of Scope

- A second WBS mutation implementation dedicated to Home.
- Inline editing of grid cells or names.
- Dragging or resizing Project, Group, or Task bars.
- Dragging a row to change parent, insert as a child, or move across Projects;
  those operations remain owned by Move to and its existing constraints.
- Choosing an explicit sibling index during Move to.
- Creating, deleting, or retargeting dependency from the chart.
- Editing generated or manual dates directly from the chart.
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

| Term                    | Definition                                                                                                                                                                                                   |
| ----------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Home                    | Default application page containing the Portfolio Gantt Workspace.                                                                                                                                           |
| Portfolio Gantt         | Combined read-only view of selected active Projects, their WBS rows, and schedule bars.                                                                                                                      |
| Project Grid            | Hierarchical left panel containing Project, Group, and Task rows and fixed data columns.                                                                                                                     |
| Timeline                | Right panel containing daily calendar columns, bars, and dependency arrows.                                                                                                                                  |
| Active Project          | A Project with status Open or Locked.                                                                                                                                                                        |
| Selected Project        | An active Project checked in the current Filter draft.                                                                                                                                                       |
| Selected Role           | A Task Role checked in the current Filter draft.                                                                                                                                                             |
| Selected Projection     | Either Execution or Commitment.                                                                                                                                                                              |
| Working-Day Anchor      | Earliest complete scheduled Start among selected Projects for the selected projection.                                                                                                                       |
| Visible Range           | Calendar-date interval currently rendered in the timeline.                                                                                                                                                   |
| System View             | Non-persisted logical view `All Active Projects`.                                                                                                                                                            |
| Saved Filter            | Globally persisted name and selected active Project IDs.                                                                                                                                                     |
| Configuration Draft     | Current saved-filter, projection, Project, and Role selection inside the Configure Gantt modal. Project selection may differ from persisted saved-filter content until Save; Role selection is session-only. |
| Projection Preference   | Browser-local UI preference containing only the last successfully applied `Execution` or `Commitment` selection. It is independent of backend saved filters.                                  |
| Column Width Preference | Browser-local Home layout preference containing the latest bounded width of each Project Grid data column. It is independent of projection, filters, and backend business data.              |
| Hierarchy Preference    | Browser-local Home UI preference containing collapsed Project/Group row identities only. It is independent of filters, projection, column widths, and backend WBS data.                     |
| Row Actions             | Labelled quick-creation controls owned by the Name cell and rendered as a temporary floating second line over the Project Grid, plus an independent icon-only overflow trigger anchored at the right edge. An eligible row remains one line at rest and expands only while hovered or keyboard-focused; an ineligible row never expands for creation actions. |
| Add Sibling            | Creates a new Task under the target row's current parent and inserts it immediately after the target in authoritative persisted sibling order.                                                            |
| Add Child              | Existing create-child workflow. On Project it creates a root Task; on Group/Task it creates beneath the selected row and preserves existing executable-to-Grouping conversion rules.                    |
| Restrictive Role Filter | Applied Role selection that does not include every currently available Role option, including `No role`; it may hide siblings.                                                                               |
| Move Up / Move Down     | Adjacent-sibling reorder within the same parent; parent never changes.                                                                                                                                       |
| Drag Handle             | Dedicated leading pointer/touch target on an eligible Task or Group row. It starts sibling reorder; the rest of the row is not draggable.                                                                    |
| Drag Reorder            | Places a Task or complete Group subtree immediately before or after another authoritative sibling under the same parent; it never reparents.                                                                 |
| Move to                 | Existing WBS move operation that changes parent within the same Project and moves the complete subtree; user does not choose sibling position.                                                               |

---

## 6. Navigation and Page Ownership

### 6.1 Home as Root and Canonical WBS Surface

- Add `Home` as the first visible item in application navigation.
- Opening the frontend application at its root displays Home without requiring
  the user to choose another menu first.
- The implementation may preserve the existing hash-navigation architecture;
  empty/default route resolution must still render Home.
- Home is the default safe fallback when no recognized application page is
  selected.
- The **Projects** page remains available for Project listing, search,
  pagination, Add Project, Project configuration, and lifecycle management.
- The **Project** navigation group may also contain **Sprints** as owned by
  US-8.1. This does not change Home as the first/default route or make Sprint a
  Project child.
- Project lifecycle actions are also available from eligible active Project rows
  on Home; both surfaces invoke the same application commands.
- Remove the standalone **Project Structure** action, page, or panel from the
  Projects workflow.
- Active WBS creation, edit, move, reorder, and delete are initiated from Home.
- Opening any Project/Group/Task dialog from Home must keep the application route
  and visible background on Home. The implementation must not navigate to
  Projects or mount an intermediate Project Structure page and then forward
  back.

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
- eligible row actions remain logically owned by the Name cell and do not create
  a separate grid column; while revealed, the creation-label layer may float
  horizontally over following Project Grid cells but must remain clipped at the
  Project Grid/Timeline boundary;
- vertical scrolling is synchronized so one logical row never separates from
  its bar;
- grid column headers and all three timeline header rows remain visible while
  vertically scrolling;
- horizontal timeline scrolling does not move the left grid;
- every Project Grid column is individually resizable;
- the latest valid width of every Project Grid column is stored browser-locally
  and restored after refresh or navigating away from Home and returning;
- changing total grid width reduces or expands the remaining Timeline width;
- row heights remain identical across both panels;
- Project and Group indentation remains readable at supported viewport sizes;
- the page does not create nested uncontrolled vertical scroll regions that
  break keyboard or pointer navigation.

---

## 7. Project Eligibility, Ordering, and Rows

### 7.1 Status Eligibility

- Open Projects are visible and editable according to their owning stories.
- Locked Projects are visible and remain protected planning anchors.
- Locked Project Name may be renamed through the shared Edit Project dialog
  without scheduler invocation; Project Settings remain read-only.
- Locked unfinished Task Actual Date remains editable through the shared Edit
  Task dialog according to US-6.2.
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
- Role is blank.
- Assignee is blank.
- Start and End use recursive Project aggregate for the selected projection.
- Effort uses recursive known descendant Task Effort.
- The row can expand or collapse its WBS descendants.
- Activating the Project Name opens the shared Edit Project dialog directly over
  Home.
- Open Project exposes the full editable Project fields allowed by US-3.1 and
  US-3.3.
- Locked Project exposes Project Name as the only editable Project field.
- An eligible Open Project row exposes the labelled `Add Child` action only;
  Project has no `Add Sibling` action because it is logical WBS level `0`.
- Project has no drag handle and cannot participate as a reorder source or
  sibling drop target.
- The overflow menu exposes lifecycle actions valid for the current status:
  Open may expose Lock, Close, and eligible Delete; Locked may expose Reopen and
  Close.
- Closed Reopen remains available on Projects because Closed Projects are absent
  from Home.
- Locked status is communicated with text/icon and not by colour alone.

### 7.4 Group Row

Each Group row:

- displays derived hierarchical WBS number;
- displays Group Name;
- displays blank Role;
- displays blank Assignee;
- displays recursive Start, End, and Effort values;
- can expand or collapse descendants;
- activating the Group Name opens one shared Group dialog containing the
  US-4.3 summary and the US-4.1 rename capability;
- Open Project allows rename in that dialog;
- Locked Project opens the same summary dialog read-only;
- In an Open Project, the row exposes labelled `Add Sibling | Add Child`
  actions; when expanded, a newly created sibling renders after the Group's
  complete visible subtree;
- In an Open Project, the row exposes a dedicated leading drag handle for
  same-parent sibling reorder; dragging the Group moves its complete subtree.
- Open Project overflow exposes Move Up, Move Down, and Move to;
- Group does not expose Delete while it has children and does not become
  executable merely because it is shown on Gantt.

### 7.5 Task Row

Each executable Task row:

- displays derived hierarchical WBS number;
- displays Task Name;
- displays Task Role or blank when no Role is assigned;
- displays Task Assignee or blank when unassigned;
- displays selected-projection Start and End when the pair is complete;
- displays direct Effort or blank when Effort is absent;
- activating Task Name opens the shared Edit Task dialog for unfinished,
  completed, and Locked Tasks;
- an unfinished Task in an Open Project exposes labelled
  `Add Sibling | Add Child` actions; Add Child remains subject to US-4.1
  conversion rules;
- an unfinished Task in an Open Project exposes a dedicated leading drag
  handle plus eligible Move Up, Move Down, Move to, and Delete through overflow;
- a completed Task in an Open Project also exposes the drag handle and labelled
  `Add Sibling` plus
  Move Up, Move Down, and Move to through overflow, but no Add Child or Delete;
  Reopen remains inside Edit Task;
- a Task in a Locked Project has no structural quick/overflow action, but Edit
  Task remains available for Actual Date according to US-6.2;
- no Task allows direct cell or bar mutation.

---

## 8. WBS Numbering, Ordering, Reorder, and Move

### 8.1 Numbering and Stable Display Order

- Project is logical WBS level `0`, but its WBS cell is blank.
- Numbering restarts independently inside every Project.
- Top-level WBS nodes are `1`, `2`, `3`, and so on.
- Child nodes are `1.1`, `1.2`, `2.1`, and so on.
- Deeper descendants continue as `1.1.1`, `1.1.2`, and so on without a product
  depth limit.
- Numbering derives from current persisted sibling position and hierarchy; it is
  not a persisted business identifier.
- Home renders siblings by persisted WBS position. Execution/Commitment Start or
  End may move bars and date values but must never reorder rows.
- IDs, not WBS numbers or names, identify rows, dependencies, and saved filter
  Projects.
- A malformed or stale projection must not use duplicate WBS numbers as row
  identity.

### 8.2 Move Up and Move Down

- Move Up/Down use the existing US-4.1 sibling reorder operation.
- The operation swaps only with the adjacent sibling under the same parent.
- Parent and subtree membership do not change.
- The unavailable boundary direction is disabled.
- Both directions are disabled while a Restrictive Role Filter is applied,
  because hidden siblings make an adjacent move ambiguous.
- Disabled reorder controls remain discoverable in overflow and explain
  `Show all roles to reorder WBS items.` through tooltip and accessible
  description.
- Project filtering does not disable reorder because a selected Project includes
  its complete visible hierarchy before Role filtering.
- Successful reorder updates persisted sibling positions, derived WBS numbers,
  scheduler order where applicable, and the refreshed Home projection.

### 8.3 Drag Reorder

- Drag reorder is available only through the dedicated leading handle on Task
  and Group rows in an Open Project. Project, Locked, and Closed rows have no
  handle. Completed Tasks remain eligible because reorder changes structure, not
  completed executable data.
- The rest of the row is not draggable. Starting pointer movement from Name,
  expand/collapse, Add Sibling/Add Child, data cells, timeline, or `...` must not
  initiate reorder.
- A valid drop places the source immediately before or after a sibling under the
  same authoritative parent. Drop indicators appear only at valid boundaries
  between sibling blocks.
- Drag reorder never changes parent, never inserts as a child, and never crosses
  Projects. Move to remains the only parent-change workflow.
- A Group moves with its entire subtree. If expanded, the visible Group and all
  rendered descendants move as one block; no drop boundary may appear between
  that Group and its descendants.
- Moving an item across an expanded sibling Group treats that sibling's complete
  subtree as one block. `After Group` means after its last descendant.
- The client submits source identity, target sibling identity, and before/after
  intent. It must not derive or submit an arbitrary position from virtualized or
  filtered row indexes.
- A Restrictive Role Filter disables drag handles and uses the same accessible
  explanation as Move Up/Down: `Show all roles to reorder WBS items.`
- Project filtering and expand/collapse state do not disable otherwise-valid
  drag reorder.
- Dragging near the vertical edge of the Project Grid auto-scrolls that grid so
  long sibling lists can be reordered beyond the current viewport. Horizontal
  timeline scrolling must not be captured by row drag.
- Escape, pointer cancellation, drop outside a valid boundary, or drop into the
  source's current position performs no mutation.
- On valid drop, the source remains visually pending until the authoritative
  reorder succeeds. Failure or stale conflict restores the confirmed original
  order and refreshes Home.
- Move Up/Down remains available in overflow as the keyboard-accessible and
  non-drag fallback and must produce the same persisted ordering and business
  effects as an equivalent drag drop.

### 8.4 Move to

- Move to uses the existing US-4.1 parent-change operation.
- Source may be a Task or Group; a Group moves with its complete subtree.
- Destination is Project root or a valid Group in the same Project.
- A node cannot move under itself or any descendant.
- The user selects only the destination parent and never selects sibling
  position.
- Destination children receive the moved item at the deterministic position
  owned by US-4.1.
- Move to remains available while a Role filter is active.
- The destination picker resolves all valid parents from the authoritative full
  Project tree, including parents hidden by the applied Role filter.
- Successful Move to preserves the applied Project/Role filter and refreshes
  hierarchy, numbering, summaries, dependencies, and affected schedules.

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

| Column            | Project                                               | Group                                                 | Task                      |
| ----------------- | ----------------------------------------------------- | ----------------------------------------------------- | ------------------------- |
| WBS               | Blank                                                 | Derived hierarchy number                              | Derived hierarchy number  |
| Task / Group Name | Project Name                                          | Group Name                                            | Task Name                 |
| Role              | Blank                                                 | Blank                                                 | Task Role or blank        |
| Assignee          | Blank                                                 | Blank                                                 | Assignee Name or blank    |
| Effort            | Recursive known Task Effort sum                       | Recursive known Task Effort sum                       | Direct Task Effort        |
| Start Date        | Earliest recursive complete selected-projection Start | Earliest recursive complete selected-projection Start | Selected-projection Start |
| End Date          | Latest recursive complete selected-projection End     | Latest recursive complete selected-projection End     | Selected-projection End   |

### 9.2 Row Actions in Name

- There is no Actions column.
- Eligible creation actions are rendered as compact text controls on a
  temporary floating second line below the row Name, logically owned by the Name
  cell but allowed to paint over following Project Grid separators.
- At rest, the row keeps its existing single-line height and renders only the
  primary line: drag handle when eligible, collapse/expand control, row Name,
  status indicators, and the independently right-anchored `...` trigger.
- Only a row with at least one eligible Add Sibling or Add Child action may
  expand. While that eligible row is hovered or contains keyboard focus, it
  temporarily expands and reveals the creation-label line below the primary
  line. A row with no eligible creation action remains at the single-line height
  even when hovered or focused for Name, lifecycle, or overflow actions.
- The creation group uses one leading plus marker followed by separately
  clickable labels and a non-interactive separator:
  - Project: `⊕ Add Child`;
  - eligible Group or unfinished Task: `⊕ Add Sibling | Add Child`;
  - completed Task in an Open Project: `⊕ Add Sibling`.
- The icon-only `...` overflow trigger is not moved into the creation-label
  line. It remains independently anchored at the far right edge of the primary
  Name line.
- Creation labels are revealed only for an eligible row when the row is hovered
  or a row control receives keyboard focus. Mouse-click focus on expand/collapse,
  Name, or overflow must not pin the expanded height after the pointer leaves the
  row. Overflow follows its existing hover/focus visibility independently and
  does not make an ineligible row expand.
- Controls remain keyboard reachable and available on non-hover/touch input;
  focus-driven expansion is reserved for keyboard navigation and non-hover touch
  interaction rather than ordinary mouse-click focus.
- Keyboard order follows the visual action order: Add Sibling when present, Add
  Child when present, then overflow.
- Revealing controls must not change the grid column count, wrap or clip the
  action labels, or move the overflow trigger. The creation-label line is a
  content-width floating layer above Project Grid cell separators, may overlay
  following Project Grid columns, and is clipped before the Timeline. It may
  temporarily increase only the eligible hovered/focused row height. The
  matching timeline row, virtualization spacers, dependency geometry, and
  vertical scroll bounds must use the same temporary height so the split grid
  remains aligned.
- The creation-label line must never overlay or consume horizontal space from
  the primary Name line. Long row Names remain readable/editable through normal
  truncation within the primary line, independently of the creation controls
  below.
- Each labelled action and the overflow trigger has an explicit accessible name
  containing the target row Name. The visual separator is not focusable and is
  ignored by assistive technology.
- Destructive Delete is separated at the bottom of overflow and uses the shared
  destructive treatment.

### 9.3 Date-Only Display

- Dates use date-only values and shared date formatting.
- Start Date and End Date cells use `D Mon YYYY`, for example `1 Aug 2026`.
- Browser timezone conversion must not shift a date.
- The visual format must remain unambiguous across month and year boundaries.
- Accessible names expose a complete calendar date even when the visual cell is
  compact.
- A partial Start/End pair is invalid source data and must not be presented as a
  valid scheduled range.

### 9.4 Task Without Effort

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

When Home is mounted, the initially applied projection follows the Projection
Preference defined in Section 10.4. If no valid preference exists, Home starts
in Execution view.

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

### 10.4 Browser-Local Projection Preference

The last **successfully applied** Execution/Commitment selection is remembered
as a browser-local Home preference. This is UI preference persistence, not a
business-data mutation and not part of the backend saved-filter model.

Persistence and restore rules:

- persist only the projection enum: `Execution` or `Commitment`;
- update the preference only after Apply has committed the configuration to the
  visible Home workspace; selecting a draft option before Apply does not update
  it;
- a successful Save As that also applies the current configuration updates the
  preference to the applied projection; Save alone does not update it when the
  projection draft is not applied;
- Cancel preserves both the currently applied projection and its stored
  preference;
- restore the preference after a browser refresh, after navigating away from
  Home and returning, and after reopening the frontend in the same browser
  profile;
- restoration is independent of the active system/saved filter and does not
  select, modify, or save a filter;
- this preference must not persist the active saved-filter identity, Project
  checklist, Role selection, visible range, expansion state, scroll position,
  task selection, search text, or any other Home configuration;
- a missing, unknown, malformed, or unreadable preference falls back safely to
  Execution;
- inability to write browser storage must not block the current projection
  change. The current session may continue with the applied projection even if
  it cannot be restored later; no backend call or blocking error is required.

### 10.5 Browser-Local Column Width Preference

The latest Project Grid column widths are remembered as a separate browser-local
Home layout preference. This is not part of a saved filter, projection
preference, or backend business model.

Persistence and restore rules:

- persist one bounded numeric width for each known Project Grid column key;
- update the preference whenever a mouse or keyboard resize changes a width;
- restore the widths after browser refresh, navigation away from Home and back,
  or reopening the frontend in the same browser profile;
- clamp restored values to each column's current minimum and maximum so an old
  preference cannot break the current layout;
- ignore unknown column keys and use the current default for missing, malformed,
  non-finite, or unreadable values;
- storage failure does not block resizing in the current session and requires no
  backend call or blocking error;
- this preference stores no projection, saved-filter identity/content, Project
  selection, Role selection, range, expansion state, scroll position, row
  selection, or business data.

### 10.6 Browser-Local Hierarchy Preference

Home remembers collapsed Project and Group identities as a separate
browser-local hierarchy preference. The preference changes presentation only;
it is not part of a saved filter, schedule, WBS persistence, or backend business
model.

Persistence and restore rules:

- with no valid stored preference, every Project and Group is expanded by
  default;
- persist only unique non-empty row identities that are currently marked
  collapsed;
- restore those identities after browser refresh, navigation away from Home and
  back, or reopening the frontend in the same browser profile;
- ignore stored identities that are absent, filtered out, or no longer
  expandable in the current projection; they must not hide unrelated rows;
- an individual expand/collapse action updates the in-memory and stored
  preference;
- automatic ancestor expansion required to reveal a newly created, edited, or
  focused row also removes those ancestor identities from the stored preference;
- missing, malformed, unreadable, or unwritable storage falls back safely to an
  expanded hierarchy or preserves current-session interaction without blocking
  Home;
- this preference stores no projection, column width, saved-filter
  identity/content, Project selection, Role selection, range, scroll position,
  row selection, or business data.

Bulk hierarchy action:

- the Name-column header exposes one compact icon-only action with tooltip and
  accessible name;
- when at least one currently visible expandable row is expanded, the action is
  **Collapse all rows** and collapses every expandable row in the currently
  loaded, Role-adjusted hierarchy;
- when no currently visible expandable row is expanded, the action is
  **Expand all rows** and expands every expandable row in that same hierarchy;
- the control is absent when the current hierarchy has no expandable row;
- bulk actions preserve stored collapse identities belonging to Projects that
  are not currently loaded or selected, and perform no backend request,
  scheduler invocation, WBS mutation, filter mutation, or projection reload.

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

### 12.2 Three Header Rows

Timeline header contains:

1. working-day sequence;
2. month and year;
3. actual calendar date.

The month/year row groups contiguous date columns belonging to the same month
and displays `Mon YYYY`, for example `Aug 2026`.

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
- When an effective Finish-to-Start chain reuses remaining capacity on the same
  calendar date, the shared date cell is divided into ordered, non-overlapping
  visual slots for the visible chain. The slots communicate sequence only and
  are not proportional to allocated minutes.
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
- The arrow connects the rendered predecessor bar end to the rendered successor
  bar start. It must never point backward merely because both endpoints share a
  calendar date.

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

## 16. Row Interaction and Shared Forms

### 16.1 Primary Action on Name

- The Name cell is the primary Edit/View trigger.
- Activating Project Name opens Edit Project.
- Activating Group Name opens the combined Group summary/rename dialog.
- Activating Task Name opens Edit Task, including completed and Locked Tasks.
- The full row and timeline bar do not become implicit edit triggers; they remain
  available for expand/collapse, scrolling, inspection, and future read-only
  interactions without accidental mutation dialogs.
- The interactive Name has visible hover/focus treatment and does not rely on a
  pencil or eye icon.

### 16.2 Quick Creation Labels

- The existing single add icon is replaced by the labelled creation group
  defined in Section 9.2.
- Project exposes `Add Child` only. It opens the existing Create Task dialog and
  creates a root Task under that Project using the existing root-Task create behaviour.
- Eligible Group and unfinished Task rows expose both `Add Sibling` and
  `Add Child` as separate buttons divided by a visual `|` separator.
- Completed Task in an Open Project exposes `Add Sibling` only; completion does
  not block creation of another Task beside it, but Add Child remains forbidden.
- Locked Group/Task rows and every Closed Project context expose no creation
  action.
- The labelled creation group appears beside the row content. The `...` overflow
  trigger remains at its current fixed trailing position on the right edge of
  the row and is not visually grouped beside the add labels.
- Labels retain explicit accessible names and invoke the US-4.1 create,
  insertion, conversion, scheduling, and rollback contracts.

### 16.3 Overflow Menu

- Secondary actions are consolidated under one icon-only overflow trigger.
- The trigger is shown only when at least one secondary action is meaningful.
- Project overflow exposes lifecycle actions owned by US-3.1.
- Group overflow may expose Move Up, Move Down, and Move to.
- Unfinished Task overflow may expose Move Up, Move Down, Move to, and eligible
  Delete.
- Completed Task overflow may expose Move Up, Move Down, and Move to.
- Locked Group/Task rows expose no overflow when no structural action is
  permitted.
- Disabled Move Up/Down remains visible with its reason; unavailable actions
  that are not contextually useful may be omitted.
- Delete is separated from non-destructive actions and always requires the
  existing confirmation.

### 16.4 Project Lifecycle on Two Surfaces

Project lifecycle remains available from both:

```text
Projects page
Home → active Project row → overflow
```

- Both surfaces invoke the same Lock, Reopen, Close, and Delete commands,
  eligibility checks, impact preview, confirmation, transaction, error mapping,
  cache invalidation, and rollback.
- Open Home row may expose Lock, Close, and eligible Delete.
- Locked Home row may expose Reopen and Close.
- Closed Reopen remains on Projects because Closed Projects do not render on
  Home.
- Home does not implement a second lifecycle state machine.

### 16.5 Reorder and Move

- Move Up/Down, drag reorder, and Move to follow Section 8 and US-4.1.
- The drag handle appears at the leading edge of an eligible Task/Group row on
  hover or row keyboard focus. It is visually separate from Name, creation
  labels, and the right-anchored `...` overflow.
- Move Up/Down remains the keyboard-accessible fallback; adding drag must not
  remove or weaken either command.
- Reorder works only against full authoritative sibling order and is disabled
  under a Restrictive Role Filter. Drag reorder may cross multiple siblings in
  one operation, while Move Up/Down remains adjacent-only.
- Move to changes only parent and remains available under Role filtering.
- The destination picker uses the authoritative full Project tree.
- Completed Task/subtree structural eligibility remains exactly as defined by
  US-4.1; Home does not weaken completed-data immutability.

### 16.6 Shared Dialog Ownership

- Project, Group, Task, Add Sibling, and Add Child use extracted reusable
  dialog/form components that can be composed by Home and Projects where
  applicable. Both creation actions open the same Create Task form; only the
  structural insertion intent differs.
- Home invokes those components directly; it must not open Projects or an
  intermediate Project Structure page in the background.
- Project edit, Group rename, Task edit, dependency, Actual Date, Reopen,
  manual timeline, confirmation, and validation remain owned by their existing
  application use cases.
- Open Group dialog combines rename with the US-4.3 summary.
- Locked Project Edit permits Project Name only; Locked Task Edit permits
  Actual Date according to US-6.2; Locked Group summary is read-only.

### 16.7 Add Sibling Insertion

When Add Sibling is activated on a Group or Task row:

- open the same Create Task dialog used by Add Child;
- submit the selected row identity as the insertion anchor rather than deriving
  a sibling index from the currently rendered rows;
- create the new Task under the anchor's current authoritative parent;
- insert the new Task immediately after the anchor in the full persisted sibling
  order, including siblings hidden by the applied Role filter;
- shift later sibling positions atomically while preserving their relative
  order;
- never convert or mutate the anchor merely because it is used as the insertion
  reference;
- when the anchor is an expanded Group, the new sibling appears after the
  Group's complete subtree, not between the Group and its children;
- keep the applied Role filter unchanged. The new Task may disappear from the
  refreshed projection when its Role does not match that filter;
- if the created Task is filtered out, retain focus on the nearest still-visible
  ancestor or anchor row rather than clearing focus or changing the filter;
- if the anchor is deleted, its Project is no longer Open, or authoritative
  validation rejects the insertion before commit, create nothing and surface the
  recoverable conflict;
- creation, sibling-position changes, required scheduling, projection refresh,
  and rollback follow the atomic US-4.1 contract.

### 16.8 Add Child Conversion

When Add Child targets an executable Task:

- US-4.1 executable-to-Group conversion rules apply;
- required warning/confirmation is shown;
- executable fields and dependencies move according to existing rules;
- cancellation leaves hierarchy and Home projection unchanged;
- scheduling and impact coordination remain atomic.

### 16.9 Projection Refresh

After a successful mutation:

- refresh every selected Project whose visible projection is affected;
- include cross-Project transitive impacts from US-6.2;
- preserve the applied saved filter, Project selection, Role selection, and
  unaffected expanded/collapsed rows where IDs still exist;
- after Add Sibling or Add Child, use the confirmed created identity to reveal
  its ancestor path, scroll the refreshed row into view, and move keyboard
  focus to that row's Name action when it remains visible; when Role filtering
  hides it, preserve the filter and focus the nearest visible anchor/ancestor;
- after a successful Project/Group/Task Edit, Move Up, Move Down, drag
  reorder, or Move to that retains the entity, wait for the post-mutation
  projection, then reveal,
  scroll to, and focus the retained ID's Name action;
- when the preserved Role selection intentionally excludes that new row, move
  focus to its nearest visible ancestor and resolve the focus request so a
  later filter change cannot unexpectedly steal focus;
- keep Home as the route and visible background;
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
- A successfully applied projection updates the browser-local Projection
  Preference defined in Section 10.4.
- Cancel discards the modal draft, preserves the currently applied chart, and
  does not change the Projection Preference.
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
- `Team A`, `team a`, and `TEAM A` conflict.
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
- invoke Add Sibling/Add Child where available;
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
2. Home displays no breadcrumb and no persistent configuration toolbar.
3. The standalone Project Structure entry point/page is removed from Projects.
4. Projects remains available for Project list/configuration/lifecycle, while
   active WBS management is initiated from Home.
5. Opening Project, Group, Task, Add Sibling, or Add Child from Home keeps Home as
   the route and visible background; no intermediate Projects/Project Structure
   page is mounted or forwarded.
6. Home shows a synchronized left Project Grid and right daily Timeline without
   overlap; row actions are contained inside Name cells.
7. Header and row alignment remain synchronized through vertical/horizontal
   scrolling and column resize.

### Project and WBS Rows

8. Every selected Open/Locked Project has a visible Project row; Closed Projects
   are absent.
9. Project WBS cell is blank while descendants start at `1` for each Project.
10. Siblings render by persisted WBS position; date recalculation never changes
    vertical row order.
11. WBS numbering reflects recursive hierarchy and current persisted sibling
    order and restarts per Project.
12. Project and Group rows expand/collapse without changing persisted hierarchy.
12A. The Name-column header exposes one contextual `Collapse all rows` or
     `Expand all rows` action whenever the current hierarchy has at least one
     expandable row.
12B. Collapse All collapses every expandable row in the currently loaded,
     Role-adjusted hierarchy; Expand All expands those rows without changing
     WBS data, filters, projection, or scheduling.
12C. Individual and bulk collapse state is restored after refresh or returning
     to Home from a separate browser-local preference. Missing, malformed,
     unreadable, or unwritable storage does not block Home and defaults safely
     to expanded rows.
12D. Stored identities that are absent, filtered out, or no longer expandable
     do not hide unrelated rows. Bulk actions preserve identities belonging to
     Projects outside the currently loaded hierarchy.
13. Project/Group Role and Assignee are blank; Task Role and Assignee show their
    direct values or blank.
14. Project/Group Effort equals recursive known Task Effort; Task shows direct
    Effort.
15. Null Task Effort is blank and incomplete-effort state is disclosed without
    invented values.

### Row Actions and Dialogs

16. Activating Project Name opens Edit Project, Group Name opens one combined
    summary/rename dialog, and Task Name opens Edit Task.
17. Timeline bars and non-Name row space remain read-only and do not implicitly
    open mutation dialogs.
18. At rest, each row remains at the existing single-line height and shows
    only its primary line. Hover or keyboard focus temporarily expands only a
    row that has at least one eligible creation action. A mouse click on a primary
    row control does not retain that expanded height after hover ends. A row with
    no Add Sibling or Add Child action remains single-line. The revealed second
    line is a content-width floating layer over Project Grid separators and does
    not add an Actions column or clip its labels at the Name-column width.
19. Project shows `Add Child`; eligible Group/unfinished Task shows
    `Add Sibling | Add Child`; completed Open Task shows `Add Sibling` only.
19A. A single leading plus marker and non-interactive `|` separator reproduce the
     approved compact label treatment while each label remains a separate
     keyboard-accessible button.
19B. The `...` overflow trigger remains independently anchored at the far right
     edge of the primary Name line and does not move into the creation-label
     line.
19C. Add Sibling opens the shared Create Task dialog, uses the selected row as an
     authoritative insertion anchor, and creates a Task immediately after it in
     full persisted sibling order under the same parent.
19D. Add Sibling on an expanded Group renders the new Task after the complete
     Group subtree; it never inserts between the Group and its children.
19E. Role filtering does not change sibling placement. A newly created Task may
     be filtered out after refresh; the applied filter remains unchanged and
     focus returns to the nearest visible anchor/ancestor.
20. Open Project row exposes eligible Add Child and lifecycle actions; Locked
    Project row exposes Reopen/Close and Edit Project.
21. Project lifecycle commands on Home and Projects use the same eligibility,
    confirmation, impact, transaction, error, and rollback contracts.
22. Locked Edit Project allows only Project Name mutation; Project Settings and
    schedule inputs remain read-only.
23. Open Group dialog allows Rename and displays US-4.3 summary in the same form;
    Locked Group dialog displays the same summary read-only.
24. Unfinished Open Task exposes Add Sibling, Add Child, and eligible Move Up,
    Move Down, Move to, and Delete.
25. Completed Open Task still opens Edit Task and exposes Add Sibling, Move Up,
    Move Down, and Move to, but not Add Child or Delete; Reopen remains inside
    Edit Task.
26. Locked Task still opens Edit Task for US-6.2 Actual Date and exposes no
    structural action.
27. Destructive Delete is separated in overflow and uses existing confirmation
    and backend eligibility.

### Reorder and Move

28. Move Up/Down swaps only the adjacent sibling under the same parent.
29. First sibling has disabled Move Up and last sibling has disabled Move Down.
30. A Restrictive Role Filter disables both reorder actions and explains
    `Show all roles to reorder WBS items.` accessibly.
31. Selecting every Role option, including `No role`, restores reorder
    eligibility.
32. Project filtering alone does not disable reorder.
32A. Eligible Task and Group rows in an Open Project expose a dedicated leading
     drag handle; Project, Locked, and Closed rows do not.
32B. The row itself is not draggable. Name, expand/collapse, creation labels,
     data cells, timeline, and overflow preserve their existing interactions.
32C. Dragging places the source immediately before or after an authoritative
     sibling under the same parent; no valid drop reparents or crosses Projects.
32D. Dragging a Group moves the complete subtree while preserving descendant
     parentage and internal order. Expanded Groups and their descendants are one
     visual drop block.
32E. Restrictive Role Filter disables drag reorder with the same accessible
     reason as Move Up/Down; Project filtering does not.
32F. Cancel, invalid drop, and same-position drop are no-ops. A valid drop is
     atomic with required scheduling, and failure restores confirmed order.
32G. Move Up/Down remains the keyboard-accessible fallback and yields the same
     authoritative order as equivalent drag reorder.
33. Move to changes only parent, moves the complete subtree, and never asks for
    sibling position.
34. Move to remains available under Role filtering.
35. Move destination options come from the authoritative full Project tree,
    including valid parents hidden by Role filtering.
36. Move to rejects self/descendant/cross-Project destinations and uses the
    existing deterministic destination position.
37. Successful reorder or Move to refreshes WBS numbering, summaries,
    dependencies, and affected schedules while preserving applied filters.

### Projection and Aggregation

38. Execution view shows only Execution dates/bars and Commitment view shows only
    Commitment dates/bars.
39. Projection switch updates grid values, bars, summaries, sequence, and
    dependency geometry coherently.
40. Projection switch performs no business mutation or scheduler invocation.
40A. After a projection is successfully applied, Home stores only that
     `Execution`/`Commitment` value as a browser-local preference and restores it
     after refresh, navigation away and back, or reopening the frontend in the
     same browser profile.
40B. Projection draft selection and Cancel do not update the stored preference;
     only the successfully applied projection does.
40C. Missing, invalid, unreadable, or unwritable browser storage falls back to
     Execution or preserves the current-session switch without blocking Home.
40D. Projection preference persistence does not persist or alter saved-filter
     identity/content, Project selection, Role selection, range, expansion,
     scroll, task selection, search, or business schedule data.
40E. Home stores the latest bounded Project Grid column widths as a separate
     browser-local preference and restores them after refresh or navigation away
     and back. Malformed or unavailable storage falls back safely without
     blocking current-session resize.
40F. Collapse state uses a third independent browser-local preference containing
     only collapsed row identities; it is not stored in projection, column-width,
     or backend saved-filter data.
41. Project/Group Start is recursive earliest complete selected-projection Start.
42. Project/Group End is recursive latest complete selected-projection End.
43. Partially scheduled scopes show aggregates from scheduled Tasks plus
    incomplete-schedule disclosure.
44. No scheduled descendants produce blank dates and no summary bar.
45. Actual data is not displayed as a Home projection.

### Timeline Header and Range

46. Timeline renders one column for every calendar date in the automatically
    derived Visible Range.
47. Weekend and Public Holiday columns remain visible and are marked non-working.
48. Header row one displays `1, 2, 3...` only on working dates from the
    Working-Day Anchor; row two displays grouped `Mon YYYY`; row three displays
    calendar date.
49. Dates before the anchor and all non-working dates display no sequence number,
    while Start/End grid dates use `D Mon YYYY` without timezone shift.
50. Public Holiday does not increment the sequence, including when it overlaps a
    weekend.
51. Default range begins on the earliest Start and ends seven calendar days after
    the latest End; no leading seven-day padding is added.
52. No schedule or no selection defaults to current calendar month.
53. Home exposes no manual From/To controls.
54. Project, Role, or projection changes recalculate the automatic range without
    changing persisted schedule data.

### Bars and Dependencies

55. Complete Task ranges display inclusive read-only bars; one-day Task occupies
    one date.
56. Task bar spans intervening non-working dates without claiming daily
    allocation.
    56A. Same-date Finish-to-Start chains render ordered non-overlapping slots in
    the shared date cell; the slots show sequence rather than exact hours.
57. Project/Group use visually and semantically distinct read-only summary bars.
58. No bar can be dragged, resized, or edited regardless of Automatic Scheduling
    state.
59. Effective dependencies display as one read-only arrow per endpoint pair.
    59A. A same-date dependency arrow connects forward from predecessor end to
    successor start and does not visually imply that the successor starts
    before the predecessor finishes.
60. Manual/automatic/shared source is accessible and does not create duplicate
    arrows.
61. Hidden, collapsed, filtered, unscheduled, or out-of-range endpoints produce
    no dangling arrow.
62. Dependency cannot be created, deleted, or retargeted directly from Home;
    mutation remains inside Edit Task.

### Mutation Refresh and Consistency

63. Successful Home mutation refreshes all affected selected Projects without
    hard reload and leaves Home visible.
64. Applied saved filter, Project selection, Role selection, and unaffected
    expansion state are preserved where IDs still exist.
65. Failed mutation/rollback does not leave Home showing unconfirmed hierarchy
    or schedule.
66. Stale Home response cannot overwrite a newer confirmed mutation.
67. Add Sibling insertion, Add Child conversion, Move, reorder, Delete, Actual Date, Reopen, and
    lifecycle commands retain owning-story atomicity and impact handling.

### Project Selection

68. `All Active Projects` is the default system view and selects all Open/Locked
    Projects.
69. System view automatically includes newly active Projects and excludes Closed
    Projects.
70. The icon-only Configure Gantt action opens one modal form and does not push
    the Gantt vertically.
71. Saved-filter, projection, Project, and Role draft changes are applied
    atomically through Apply; Cancel preserves the applied chart.
72. Role filtering retains ancestors, recomputes their aggregates from matching
    Tasks, and removes dangling dependency arrows.

### Saved Filters

73. The saved-filter selector is available only inside Configure Gantt and shows
    system/saved filter names without Project detail metadata.
74. Save As from system or saved view creates a globally visible named filter
    from current active selection.
75. Saved-filter Name is trimmed, required, case-insensitively unique, and
    cannot use reserved system-view Name.
76. Opening a saved filter checks only currently active stored Projects and
    silently ignores Closed/deleted IDs.
77. If all stored Projects are inactive, opening the filter shows empty
    selection without warning.
78. Save remains available on an active saved filter even when draft is
    unchanged.
79. Save replaces selected IDs with the current active checklist and removes
    inactive IDs.
80. Save As leaves the source filter unchanged and activates the newly created
    filter.
81. Empty saved filters are valid for Save and Save As.
82. Delete confirmation removes a saved filter; deleting active filter returns
    to `All Active Projects`.
83. System view cannot be saved over or deleted.
84. Stale Save/Delete is rejected by Version and does not silently overwrite
    concurrent changes.
85. Saved-filter failure preserves current draft and does not corrupt the
    selector state.

### Data, Performance, and Accessibility

86. Home uses a bounded portfolio read model and does not issue
    per-Project/per-Task N+1 requests.
87. Move destination resolution obtains one authoritative Project tree per
    opened Move dialog and does not derive valid parents from filtered rows.
88. Read and saved-filter queries have deterministic ordering and completed
    query/index review evidence.
89. Date-only values do not shift because of browser timezone.
90. Rapid Project filter, Role filter, projection, or auto-range changes cannot
    restore stale data.
91. Large row/date ranges use bounded rendering/virtualization rather than
    permanent full cell matrices.
92. Names, hierarchy, statuses, schedules, holidays, dependencies, quick
    actions, overflow menus, disabled reasons, confirmations, and errors are
    keyboard and assistive-technology accessible.
93. Loading, empty, retry, conflict, and failure states preserve valid user
    context.
94. Backend and frontend validation suites required by project architecture
    pass.

---

## 25. Mandatory Test Scenarios

### 25.1 Hierarchy and Multi-Project

- Two selected Projects both contain WBS `1` and `1.1`; rows remain uniquely
  associated with their Project.
- Project with no WBS.
- Deep Group hierarchy.
- Reorder/Move to completed from Home updates WBS numbering and keeps Home visible.
- Collapse Project and nested Group, then refresh affected projection.
- Collapse an individual Project and nested Group, refresh or navigate away and
  back, and verify both remain collapsed.
- Use Collapse All from a mixed hierarchy, verify every expandable row is
  collapsed and the same header control becomes Expand All; then expand all and
  verify the complete hierarchy returns.
- Switch Project/Role selection, use a bulk hierarchy action, and verify stored
  collapse identities for Projects outside the loaded hierarchy remain intact.
- Corrupt or block hierarchy-preference browser storage and verify Home defaults
  safely to expanded rows while current-session expand/collapse remains usable.
- Long names and narrow viewport preserve row/bar alignment.

### 25.2 Projection and Aggregation

- Execution and Commitment ranges differ for the same Task.
- Task scheduled in Execution but unscheduled in Commitment.
- Group with scheduled and unscheduled descendants.
- Project with multiple top-level WBS roots.
- Task without Effort mixed with Tasks having fractional-hour Effort.
- Completed Task still contributes total Effort.
- Unconfirmed schedule preview does not change Home summary.
- Apply Commitment, refresh the browser, and verify Commitment remains applied.
- Apply Commitment, navigate to another application page, return to Home, and
  verify Commitment remains applied.
- Select another projection in the configuration draft, Cancel, and verify both
  the applied projection and stored preference remain unchanged.
- Remove or corrupt the stored projection preference and verify Home falls back
  to Execution without blocking rendering.
- Make browser storage unavailable for writes and verify the projection still
  changes for the current session without a backend mutation.
- Verify restoring the projection does not restore or modify active filter,
  Project/Role selection, range, expansion, scroll, task selection, or search.
- Resize multiple Project Grid columns, navigate away from Home and return, and
  verify the latest bounded widths are restored independently of projection and
  saved filters.
- Corrupt or block column-width browser storage and verify current-session
  resizing remains usable with safe per-column defaults.

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
- Same-assignee Finish-to-Start chain whose predecessor End equals successor
  Start renders non-overlapping shared-date slots and a forward-pointing arrow.
- Dependency endpoint has no selected-projection bar.
- Attempts to drag/resize/create dependency produce no mutation.

### 25.5 Forms, Actions, and Lifecycle

- Activate Project, Group, and Task Name and verify shared dialogs open directly
  over Home without route/background change.
- Hover/focus each row type and verify Project shows `Add Child`, eligible
  Group/unfinished Task shows `Add Sibling | Add Child`, completed Task shows
  `Add Sibling`, and Locked WBS shows no creation action.
- Verify a row without creation eligibility remains single-line when hovered and
  when its Name or overflow receives keyboard focus.
- Resize Name to its minimum, hover an eligible row, and verify the floating
  creation labels remain complete over following Project Grid separators and do
  not cross into the Timeline.
- Verify the `...` overflow trigger stays anchored at the right edge while add
  labels appear, including with a long truncated row Name.
- While an eligible row is hovered, click expand/collapse and `...`; after the
  pointer leaves, verify the row and matching timeline row return to single-line
  height even though the clicked control or overflow menu still owns focus.
- Add Child from Open Project and verify the existing root-Task behaviour.
- Add Sibling after a top-level Task and after a nested Task.
- Add Sibling after an expanded Group and verify the created Task renders after
  the Group's complete subtree.
- Add Sibling beside a completed Task and verify the completed Task is unchanged.
- Add Sibling under a Restrictive Role Filter and verify authoritative full-order
  insertion; when the new Task does not match the filter, preserve the filter and
  focus the nearest visible anchor/ancestor.
- Add Child to Group.
- Add Child to executable Task with confirmed conversion.
- Cancel conversion.
- Open Group summary and rename the Group in the same dialog.
- Open completed Task, verify Edit Task and Reopen remain available, and verify
  Add Child/Delete are absent.
- Open Locked Project and rename only Project Name.
- Open Locked Task and enter Actual Date while planning fields remain read-only.
- Move Up/Down swaps one adjacent sibling and updates persisted order.
- Drag a Task before and after non-adjacent siblings and verify the exact
  persisted order, WBS numbering, focus restoration, and schedule refresh.
- Drag an expanded Group across siblings and verify its complete visible subtree
  follows while descendant hierarchy and internal order remain unchanged.
- Verify no drop boundary appears between a Group and its descendants, and that
  dropping after an expanded Group means after its complete subtree.
- Cancel with Escape, drop outside, and drop into the original position; verify
  no persistence or scheduling request.
- Attempt cross-parent/cross-Project drop and verify no valid indicator or
  mutation. Simulate stale target and persistence/scheduling failure and verify
  rollback to confirmed order.
- Verify drag starts only from the handle, supports vertical grid auto-scroll,
  and does not capture Name, expand/collapse, creation labels, timeline scroll,
  or right-anchored overflow interactions.
- Move Up/Down and drag reorder are disabled under a Restrictive Role Filter and
  restored after all Role options are selected.
- Move to changes parent while a Role filter is active and includes a valid
  destination hidden by that filter.
- Delete an eligible unfinished leaf Task through overflow and verify
  confirmation.
- Run Lock, Reopen, Close, and eligible Delete from Home Project overflow and
  verify the same behaviour through Projects.
- Edit causes impacted Locked Project and follows US-6.2 blocking.
- Project becomes Locked while a form or overflow menu is open.
- Mutation failure rolls back and refreshes confirmed data without leaving Home.

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
- Keyboard expands Project/Group, activates Name, opens overflow, and invokes
  eligible creation labels.
- Screen reader receives row type, WBS, status, dates, Effort, and
  scheduled/unscheduled state.
- Public Holiday and dependency source are understandable without colour.
- Focus survives projection refresh when the focused entity still exists.
- Successful visible Add Sibling/Add Child focuses the newly created row after
  that row is available in the refreshed projection, including outside the
  current virtual row window.
- Successful Edit, Move Up/Down, drag reorder, and Move to focus the retained
  row Name only after the confirmed post-mutation projection is rendered; the
  stale row or disabled overflow trigger is not accepted as focus evidence.
- A Role filter that excludes the new row preserves its selection, focuses the
  nearest visible ancestor, and leaves no pending focus request.

---

## 26. Three-Level Confidence Requirements

Every Acceptance Criterion must meet the repository's Three-Level Confidence
Standard.

### 26.1 Code Inspection

Inspect at minimum:

- Home routing/navigation, fallback, and removal of Project Structure entry;
- reusable dialog boundaries with no Projects/Project Structure background route;
- row action eligibility, overflow composition, and lifecycle command reuse;
- persisted WBS ordering, adjacent and anchor-based drag reorder, Group
  subtree movement, Role-filter disablement, and Move to destination resolution;
- recursive row composition and date/Effort aggregation;
- date-only and working-day sequence calculation;
- active Project and saved-filter domain invariants;
- normalized unique Name and optimistic concurrency;
- portfolio read query and N+1 prevention;
- cache/request identity and stale-response handling;
- browser-local hierarchy restore, contextual bulk action, and safe storage
  fallback;
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
- projection-preference parsing, apply-only persistence, safe fallback, and
  storage-failure behaviour;
- hierarchy-preference parsing, deterministic identity persistence, individual
  and bulk restore, unknown-identity handling, and storage-failure behaviour;
- row virtualization and visible dependency selection;
- component interaction and form integration;
- action matrix by Project status, WBS type, and completion state;
- reorder boundary/Role-filter behavior, authoritative before/after drag
  placement, subtree invariants, rollback, and full-tree Move to destinations.

### 26.3 Acceptance-Level Tests

Provide public-boundary evidence for:

- root Home navigation;
- active Project selection and portfolio rendering;
- Execution/Commitment switching and browser-local restoration after refresh
  or Home remount;
- individual and bulk hierarchy collapse/expand with browser-local restoration,
  unknown-identity preservation, and no WBS/projection mutation;
- daily header/holiday behaviour;
- direct Home Project/Group/Task dialog workflows with no background route
  transition;
- Add Sibling/Add Child labelled-action workflows, exact sibling insertion,
  expanded-Group placement, Role-filter preservation, and right-anchored overflow;
- Group summary plus rename in one dialog;
- sibling Move Up/Down, handle-only drag reorder with Group subtree movement,
  Role-filter disabled reason, Move to, and Delete;
- Project lifecycle actions from both Home and Projects;
- completed Task Edit/Reopen and Locked Task Actual Date;
- read-only bars/dependencies;
- saved-filter Save/Save As/Delete and concurrency errors;
- Closed exclusion and Locked lifecycle behaviour;
- stale-response prevention after mutation.

Trace each AC to production files, unit/integration tests, and highest practical
acceptance workflow in the completion evidence.

---

## 27. Technical and Documentation Impact

Implementation must assess and update:

- frontend navigation and root-route composition;
- removal of the standalone Project Structure entry point/panel;
- Home/Gantt as the canonical active-WBS presentation boundary;
- extraction of reusable Project/WBS dialogs and action orchestration without
  circular feature dependencies or hidden page routing;
- replacement/migration of Project Structure acceptance workflows to Home;
- portfolio read application/query service and transport contract;
- saved-filter domain/application/persistence/transport/UI;
- frontend browser-local projection-preference ownership, storage key/versioning,
  parsing, fallback, and failure handling with no backend schema/API change;
- frontend browser-local hierarchy-preference ownership, storage key/versioning,
  bulk-action scope, parsing, fallback, and failure handling with no backend
  schema/API change;
- database migration for saved filters and normalized uniqueness/version;
- Project/WBS/Member/Dependency/Public Holiday read composition;
- WBS create-command support for an authoritative insert-after anchor, atomic
  sibling-position shifting, concurrent insertion safety, and rollback;
- cache and schedule projection clock integration;
- frontend design-system needs for treegrid, split grid/timeline, hover/focus
  labelled creation actions on a second line below Name, independently
  right-anchored overflow,
  disabled-action explanation, bars, holiday cells,
  tooltip, empty/loading/error states, and virtualization;
- project architecture and product context;
- US-3.1, US-3.3, US-4.1, US-4.2, US-4.3, US-5.1, US-6.1, and
  US-6.2 cross-story ownership wording;
- API documentation and query/index review evidence.

`AGENTS.md` changes only if implementation discovers a genuinely reusable,
project-wide agent rule; this story alone does not require one.

---

## 28. Locked Product Decisions

1. Home is the first menu and default root page, and it displays no breadcrumb.
2. Home is the canonical active-Project WBS management surface; the standalone
   Project Structure entry point/page is removed.
3. Projects remains available for Project list/configuration/lifecycle.
4. Project lifecycle actions are available on both Projects and eligible active
   Project rows in Home and use the same application commands.
5. Project, Group, Task, Add Sibling, and Add Child dialogs invoked from Home render
   directly over Home without routing through Projects or Project Structure.
6. Home displays a portfolio Gantt inspired by GanttPRO but governed by SchedMind
   rules.
7. Project row remains visible; only its WBS `0` label is omitted.
8. Descendant WBS numbering starts at `1`, restarts per Project, and follows
   persisted sibling position.
9. Execution/Commitment dates never determine vertical row order.
10. Grid columns are WBS, Name, Role, Assignee, Effort, Start, and End.
11. Labelled creation actions are rendered on a temporary second line below
    the row Name, inside the Name cell. Rows remain one line at rest and only the
    hovered or keyboard-focused row expands; there is no Actions column.
12. Activating row Name is the primary Edit/View action; bars and non-Name row
    space remain read-only.
13. Project shows `⊕ Add Child`; eligible Group/unfinished Task shows
    `⊕ Add Sibling | Add Child`; completed Open Task shows `⊕ Add Sibling`.
14. Add Sibling and Add Child are separate text buttons. The separator is visual
    only, the labels never overlay the primary Name line, and the icon-only
    `...` overflow remains independently anchored at the far right edge of the
    primary Name line rather than inside the add-label line.
15. Open Project overflow may expose Lock, Close, and eligible Delete; Locked
    Project overflow may expose Reopen and Close.
16. Locked Project Edit allows Project Name only; Project Settings remain
    read-only.
17. Group summary and Group rename are combined in one dialog for Open Project;
    Locked Group summary is read-only.
18. Completed Task still opens Edit Task and may be reopened from that dialog.
19. Locked Task still opens Edit Task for Actual Date according to US-6.2.
20. Move Up/Down swaps adjacent siblings only and never changes parent.
21. Move Up/Down is disabled under a Restrictive Role Filter and explains why.
22. Move to changes parent only, moves the full subtree, offers no sibling
    position, and remains usable under Role filtering.
23. Move to destinations come from the authoritative full Project tree, not the
    filtered visible rows.
24. Project/Group Role and Assignee are blank; only Task rows display direct
    Role and Assignee values.
25. Project/Group dates and Effort aggregate recursively across all matching
    descendant Tasks.
26. Execution/Commitment is the only projection selector; Actual is absent.
27. Gantt cells, bars, and dependency arrows are always read-only.
28. Home has no inline editing.
29. Dependency arrows are read-only and dependency editing remains in Edit Task.
30. Every calendar date is shown at daily scale, including weekends and Public
    Holidays.
31. Timeline header has three rows: working-day number, grouped `Mon YYYY`, and
    calendar date.
32. Header row one displays bare working-day numbers `1, 2, 3...`; Start/End
    grid dates use `D Mon YYYY`.
33. Working-day numbering anchors to earliest scheduled Start for the selected
    projection.
34. Dates before anchor and non-working dates have blank sequence cells.
35. Default range begins at earliest Start and ends seven calendar days after
    latest End, with no leading padding.
36. No schedule or no selection uses current calendar month.
37. Visible range is automatic only and follows the currently filtered Task set.
38. Closed Projects are inactive and absent; Open and Locked are eligible.
39. Default `All Active Projects` selects every Open and Locked Project and is
    displayed as the Home title.
40. Home has no persistent filter toolbar; saved filter, projection, Project,
    and Role configuration are consolidated in one icon-triggered modal form.
41. Saved filters are backend-persisted and global because login is absent.
42. Saved-filter selector displays names only.
43. Saved filters support Save, Save As, and Delete.
44. Save updates selected Project IDs under the same Name.
45. Save As creates a new Name and leaves the source unchanged.
46. Empty saved filters are valid.
47. Filter Name is trimmed, required, case-insensitively unique, and cannot use
    the system-view Name.
48. Closed/deleted Project IDs are silently omitted when a filter opens and
    removed on Save/Save As.
49. If every stored Project is inactive, show empty state without informational
    warning.
50. Switching filters discards unsaved checklist draft without confirmation.
51. Daily scale is the only zoom level in this story.
52. Saved filters do not store Role selection, projection, range, expansion,
    scroll, task selection, or search state.
53. Global Save/Delete uses simple optimistic concurrency; no merge UI is
    required.
54. Project Grid data columns are individually resizable and Timeline consumes
    the remaining width without overlap.
54A. The latest bounded width of every Project Grid column is stored as a
     browser-local layout preference and restored after refresh or returning to
     Home. Unknown/malformed values fall back per column, and storage failure does
     not block current-session resizing.
54B. The Name-column header uses one contextual icon-only Collapse All/Expand All
     action. It is hidden when the current hierarchy has no expandable row.
55. The last successfully applied Execution/Commitment projection is stored as
    a browser-local preference and restored when Home is opened again.
56. Projection preference persistence stores only the projection enum and is
    independent of backend saved filters and all other Home configuration.
57. No valid stored preference defaults to Execution; browser-storage read/write
    failure must not block Home or the current-session projection switch.
58. Draft projection changes are not remembered until Apply succeeds; Cancel
    leaves the prior preference unchanged.
59. Add Sibling is available on eligible Group, unfinished Task, and completed
    Task rows in an Open Project; it is unavailable on Project, Locked, and
    Closed contexts.
60. Add Sibling always creates a Task under the anchor's same authoritative
    parent immediately after the anchor in full persisted sibling order.
61. An expanded Group's sibling renders after its entire subtree; Add Sibling
    never creates a child or inserts inside the subtree.
62. Role filters do not influence insertion order and are not changed after
    create, even when the new Task becomes hidden.
63. Add Child retains the existing behaviour: Project creates a root Task;
    Group/Task creates beneath the row, including established conversion.
64. Create plus sibling-position shifting and any required scheduling are one
    atomic operation; failure creates no Task and leaves ordering unchanged.
65. Eligible Task and Group rows in an Open Project expose a dedicated leading
    drag handle; Project, Locked, and Closed rows do not.
66. Only the handle initiates row reorder. Name, expand/collapse, Add controls,
    data cells, timeline, and the right-anchored overflow do not.
67. Drag reorder places the source before or after a sibling under the same
    authoritative parent and never reparents or crosses Projects.
68. Dragging a Group moves its entire subtree while preserving every descendant's
    hierarchy and internal order.
69. Expanded Groups are indivisible visual blocks for drop targeting; no item may
    be dropped between a Group and its descendants.
70. A Restrictive Role Filter disables drag reorder with
    `Show all roles to reorder WBS items.`; Project filtering alone does not.
71. Move Up/Down remains available as the keyboard-accessible fallback and is
    behaviourally equivalent to adjacent drag reorder.
72. Escape, pointer cancellation, invalid boundary, and same-position drop are
    no-ops without scheduling.
73. Valid drag reorder, required scheduling, persistence, and confirmed refresh
    are atomic; stale or failed mutation restores the prior confirmed order.
74. Vertical auto-scroll supports drag across long sibling lists without
    capturing horizontal Timeline scrolling.
75. Collapsed Project/Group identities are stored in a separate browser-local
    hierarchy preference and restored after refresh or returning to Home; no
    valid preference defaults every row to expanded.
76. Collapse All/Expand All affects only expandable rows in the currently loaded,
    Role-adjusted hierarchy, preserves collapse identities for unloaded Projects,
    and performs no backend or scheduler mutation.
