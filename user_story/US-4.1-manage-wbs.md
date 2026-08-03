# US-4.1 Manage WBS

> **Product decision update — US-6.4:** Task mutations and rollback-only schedule
> preview recalculate dates, allocations, and unscheduled state without creating,
> removing, or previewing automatic dependency ownership. Dependency remains
> explicit manual data owned by US-5.1. This supersedes every automatic
> dependency preview/reconciliation clause in this story.

> **Product decision update — US-7.1:** Home Portfolio Gantt is the canonical
> active-Project WBS presentation surface. The standalone Project Structure
> entry point is removed. Home reuses the same Project, Group, and Task forms and
> the same Add Task/Add Child, rename, reorder, move, and delete application
> use cases. The Gantt cells, bars, and dependency arrows remain read-only;
> US-4.1 remains the owner of WBS mutation and conversion rules.

> **Product decision updates — US-6.1 and US-4.3:** WBS mutations use the
> concrete portfolio scheduler when Automatic Scheduling is ON. Dependency
> remains owned by US-5.1; Task Lag and generated timeline algorithms are owned
> by US-6.1. View Group and Edit Project may display the recursive read-only summary owned by
> US-4.3 without making executable attributes belong to the Group or writable aggregate fields belong to Project.
> **Product decision update — US-6.2:** Completion uses required Actual Date
> (`Actual Start` and `Actual End`) entered together. Locked Project WBS and Task
> planning are immutable, but Actual Date may be recorded without changing the
> protected baseline; its Actual Allocation may recalculate impacted Open
> Projects. Task details expose read-only Execution, Commitment, and Actual daily
> allocation for verification. Any other WBS/Task mutation requires explicit
> Project Reopen to Open.

> **Product decision update — US-6.3:** Executable WBS owns persisted `Capacity
> Allocation (%)` with default `100` and valid integer range `1–100`. US-6.3
> owns percentage semantics, migration, planned allocation, and scheduler
> interaction. This story owns Task-form integration and structural conversion:
> percentage moves with executable data to a conversion child, while a newly
> executable WBS defaults to `100`.

## User Story

**As an** Engineering Lead
**I want** to manage the Work Breakdown Structure (WBS) of a project
**So that** project work can be decomposed into executable work items for future scheduling.

---

## Business Context

A Project consists of a hierarchical WBS. A WBS without children is automatically an Executable WBS. A WBS with children is automatically a Grouping WBS. There is no separate Task entity.

The UI uses product-friendly derived labels without changing this model:
the US-7.1 **Home Portfolio Gantt** is the canonical active-Project WBS
presentation surface, **Task** labels an Executable WBS, and **Group** labels a
Grouping WBS. These are presentation labels, not persisted entity types. Home
must invoke the existing application use cases and reusable dialogs rather than
duplicating mutation rules.

---

## Scope

### In Scope

- Create, rename, move and delete WBS
- Unlimited hierarchy
- Automatic conversion between Grouping and Executable WBS
- Manage executable attributes owned by this story: Role, Assignee, Effort, Capacity Allocation Percentage integration, Manual Execution Timeline, Manual Commitment Timeline, Actual Start, Actual End
- Integrate Task Capacity Allocation Percentage from US-6.3 without duplicating its scheduling or migration rules
- Integrate with Dependency from US-5.1 and Lag/generated dates from US-6.1 without duplicating their business rules
- Validation, Persistence, API and UI

### Out of Scope

- Execution and Commitment scheduling algorithms; owned by US-6.1
- Forecast Scheduler
- Dependency editor and graph rules; owned by US-5.1
- Lag validation and calculation; owned by US-6.1
- Task Capacity Allocation Percentage calculation and migration; owned by US-6.3
- Recursive Group and whole-Project timeline/effort-completion summary; owned by US-4.3

---

## Business Rules

- Unlimited WBS hierarchy.
- Leaf node automatically becomes Executable WBS.
- Parent node automatically becomes Grouping WBS.
- Grouping WBS cannot own executable attributes. A Group may display the
  read-only recursive descendant summary defined by US-4.3; the summary is not
  persisted on the Group.
- When adding the first child to an Executable WBS, display a warning and move executable attributes to the first child.
- Manual Execution/Commitment Timeline is editable only when Automatic Scheduling is OFF and the Project is Open.
- Actual Date is only available on Executable WBS. Actual Start and Actual End are entered together after completion. On Locked Project, baseline remains unchanged while Actual Allocation may recalculate impacted Open Projects.
- All other WBS/Task create, edit, delete, move, reorder, conversion, and planning mutations are rejected while the Project is Locked. Project Name rename is a Project-level exception owned by US-3.1 and does not make Group/Task structure editable.

---

## Home Presentation Contract

US-7.1 owns placement and interaction composition; this story owns the invoked
WBS commands.

- The standalone Project Structure entry point/page is removed.
- Active Project WBS create, rename, reorder, Move to, and Delete are initiated
  from Home.
- Activating Group Name opens one dialog that combines this story's Group rename
  with the read-only US-4.3 Group summary.
- Activating Task Name opens Edit Task for unfinished, completed, and Locked
  Tasks. Completed Task Reopen remains inside that dialog under US-4.2.
- Project Add Task and eligible Group/Task Add Child remain visually distinct
  icon-only quick actions.
- Move Up, Move Down, Move to, and Delete are secondary actions in the Home row
  overflow menu.
- Home invokes reusable dialogs/use cases directly and must not mount an
  intermediate Project Structure page.
- Acceptance-level evidence for active WBS management must exercise the Home
  workflow rather than the removed Project Structure workflow.

---

## Move Rules

- A WBS node may be moved to any valid parent within the same Open Project. Locked and Closed Projects reject Move.
- The user-facing **Move to** operation changes parent only. It does not accept a sibling position or perform Move Up/Down.
- Moving a node also moves its entire descendant subtree.
- A node cannot be moved under itself or under one of its descendants.
- The move must preserve a valid acyclic tree.
- The moved node receives the existing deterministic position among the destination parent's children; the user does not choose that position.
- When the source parent has no remaining children after the move, it automatically becomes an Executable WBS.
- When the destination parent receives its first child, it automatically becomes a Grouping WBS.
- If the destination parent was an Executable WBS with executable attributes, the normal Executable-to-Grouping conversion and confirmation rules apply.
- Cross-Project moves are not allowed in this story.
- Home Role filtering does not change Move validity. The destination picker must
  resolve the authoritative full Project tree and include every valid parent,
  including a parent hidden by the current Role filter.
- When Project Automatic Scheduling is ON, a confirmed move triggers the concrete portfolio scheduler from US-6.1 for the affected active scope.
- When Project Automatic Scheduling is OFF, existing manual Execution and Commitment dates remain unchanged.
- US-4.1 owns the mutation trigger and atomic coordination; US-6.1 owns the scheduling algorithm. Integrated acceptance tests must verify observable generated-date changes, not only port invocation.

## Delete Rules

- Only a leaf WBS node may be deleted.
- A Grouping WBS that still has one or more children cannot be deleted.
- The user must first delete or move all children before deleting that parent.
- Delete requires confirmation.
- Delete is a hard delete for a leaf that has no descendants and is allowed only while the Project is Open, subject to completed-task restrictions.
- Deleting a leaf may cause its parent to have no remaining children; that parent then automatically becomes an Executable WBS.
- Deleting a WBS must not cascade-delete descendants because deletion of a node with descendants is prohibited.
- When Automatic Scheduling is ON, confirmed deletion triggers the concrete portfolio scheduler from US-6.1 only when the deleted Task has scheduling impact: Assignee, Effort, non-zero Lag, generated Execution/Commitment state, an automatic unscheduled projection, or dependency endpoints. Deleting a Name-only or Role-only unfinished Task with no dependency does not invoke scheduling. The presence of completed Tasks elsewhere in the Project does not change this trigger rule.
- When Automatic Scheduling is OFF, remaining manual dates stay unchanged.

---

## Acceptance Criteria

1. Engineering Lead can create root and child WBS.
2. Unlimited hierarchy is supported.
3. Leaf WBS automatically becomes Executable.
4. Parent WBS automatically becomes Grouping.
5. Executable attributes move to the first child after confirmation.
6. Engineering Lead can rename and reorder WBS.
7. A WBS node and its entire subtree can be moved to any valid parent in the same Project.
8. Moving a node under itself or its descendant is rejected.
9. Moving the last child away converts the source parent to Executable.
10. Moving a node into an Executable destination applies the confirmed Executable-to-Grouping conversion.
11. With Automatic Scheduling ON, a successful move invokes concrete portfolio recalculation and refreshes affected generated dates.
12. With Automatic Scheduling OFF, a successful move preserves existing manual dates.
13. Only a leaf WBS may be deleted.
14. A WBS with children cannot be deleted.
15. The delete conflict explains that children must be moved or deleted first.
16. Deleting the last child converts the parent to Executable.
17. With Automatic Scheduling ON, deleting an unfinished Name-only or Role-only leaf with no dependency succeeds without concrete portfolio recalculation and leaves all remaining Task state unchanged. Deleting a Task that has scheduling input, generated projection, or dependency endpoints still invokes affected-scope recalculation.
18. With Automatic Scheduling OFF, successful deletion preserves remaining manual dates.
19. Executable WBS supports Role, Assignee, Effort, Capacity Allocation Percentage, Manual Execution Timeline, Manual Commitment Timeline, Actual Start, and Actual End.
20. Grouping WBS cannot own or edit executable fields; View Group may display
    the read-only recursive descendant summary defined by US-4.3.
21. Manual timeline is editable only when Automatic Scheduling is OFF.
22. Complete Actual Date is only available for Executable WBS. On an Open Project it actualizes timelines and creates Actual Allocation through US-6.2; on a Locked Project it preserves baseline while Actual Allocation may recalculate impacted Open Projects.
23. Locked Project rejects every WBS/Task planning or structural mutation, including create, rename, edit, delete, move, reorder, conversion, and Task Reopen. Complete Actual Date entry is the only Task mutation exception.
24. With Automatic Scheduling ON, a root or child Task created only with Name and without structural conversion persists with empty generated dates and does not invoke concrete portfolio recalculation. Create that converts an existing Executable WBS and moves executable data or dependency endpoints, sibling reorder, Assignee change, Effort change, Lag change, and other established structural conversions still invoke concrete portfolio recalculation.
25. Rename and Role-only changes do not invoke scheduling when Assignee is unchanged.
26. WBS mutation and required scheduling are atomic; scheduling failure rolls back hierarchy, executable data, generated dates, and confirmed UI state.
27. On an Open unfinished automatic Task, blur previews generated dates and automatic dependency ownership without persisting the Task when Role, valid Effort, and valid Lag are present. Assignee may be selected or explicitly cleared: a selected Assignee previews its reconciled schedule, while a cleared Assignee still calls preview to remove stale automatic ownership and return a missing-Assignee unscheduled projection. Missing or invalid Role, Effort, or Lag makes no preview request.
28. Home is the canonical active WBS management surface and the standalone Project Structure entry point is removed.
29. Activating Group Name opens one summary/rename dialog; activating Task Name opens Edit Task for unfinished, completed, and Locked Tasks.
30. Add Task and Add Child are distinct icon-only quick actions; Move Up, Move Down, Move to, and Delete are placed in one row overflow menu according to eligibility.
31. Move Up/Down swap only adjacent siblings under the same parent and are disabled at the unavailable boundary.
32. Home disables Move Up/Down while a restrictive Role filter may hide siblings and explains that all Roles must be shown; Move to remains available.
33. Move to changes parent only, offers no sibling-position input, and resolves all valid destinations from the full authoritative Project tree.
34. A completed Task in an Open Project may be reordered or moved under existing BAU, but cannot Add Child or Delete; its executable data and Actual Date remain immutable.
35. Active WBS acceptance-level tests exercise `Home → row action/dialog → confirm → refreshed Home` and prove that no Projects/Project Structure background navigation occurs.

---

## Test Cases

### Happy Path

- Create root WBS with Name only and verify scheduler is not invoked and generated dates remain empty.
- Create nested WBS under an existing Group with Name only and verify scheduler is not invoked.
- Convert Executable to Grouping and verify concrete scheduling still runs when executable data or dependency endpoints move.
- Rename WBS.
- Move a leaf to another Grouping WBS.
- Move a Grouping WBS together with its full subtree.
- Move the last child away and verify the source parent becomes Executable.
- Move into an Executable destination and confirm conversion to Grouping.
- Move with Automatic Scheduling ON and verify generated Execution/Commitment dates are recalculated.
- Move with Automatic Scheduling OFF and verify manual dates remain unchanged.
- Delete a leaf.
- Delete the last child and verify the parent becomes Executable.
- Delete a Name-only unfinished leaf from a Project that also contains a completed Task and verify deletion succeeds without scheduler invocation or changes to remaining Task state.
- Delete a Task with scheduling input, generated projection, or dependency endpoints while Automatic Scheduling is ON and verify affected Execution/Commitment dates are recalculated.
- Delete with Automatic Scheduling OFF and verify remaining manual dates are unchanged.
- Edit executable attributes, including Capacity Allocation Percentage.
- Enter complete Actual Date and verify allocation.

### Validation

- Attempt executable fields on Grouping WBS.
- Verify View Group summary remains derived/read-only and does not create Group
  executable state.
- Add child to Executable containing data without confirming conversion.
- Edit manual timeline while Automatic Scheduling is ON.
- Move a node under itself.
- Move a node under one of its descendants.
- Move a node to a different Project.
- Delete a Grouping WBS that still has children.
- Bypass frontend and call delete API for a parent with children.
- Project lifecycle restriction rejects mutation for Locked or Closed Project where required by approved Project rules.
- Concurrent move/delete operations preserve one valid tree.

### Regression

- Existing WBS data is preserved after unrelated moves.
- Child ordering remains deterministic.
- Executable-to-Grouping conversion preserves executable data in the first child.
- Grouping-to-Executable conversion does not invent executable field values.
- Actual Start and Actual End persist atomically.
- Failed move or delete leaves hierarchy and dates unchanged.
- Older in-flight list/tree responses cannot restore stale hierarchy after mutation.

---

## Required Automated Tests

- Domain
- Application
- Repository
- API
- Frontend
- WBS conversion
- Move subtree and cycle prevention
- Leaf-only deletion
- Concrete scheduler trigger and observable date update
- Manual-date preservation
- Concurrency and rollback
- Validation

---

## Scheduling Ownership

US-4.1 owns WBS mutation, validation, persistence, and trigger coordination.
US-6.1 owns concrete Execution/Commitment scheduling, capacity resolution, Lag,
and shared scheduler orchestration. US-6.3 owns Task Capacity Allocation
Percentage, concurrent planned allocation, manual fixed allocation, and revised
safe Auto Dependency behaviour. US-4.1 must call the concrete scheduler when
available and may not claim completion from a no-op adapter.

---

## Locked Product Decisions

- WBS is the only work-item aggregate.
- Task is represented by Executable WBS.
- Unlimited hierarchy.
- Automatic Grouping/Executable conversion.
- Group and whole-Project summary is a recursive read-only projection from descendant Tasks
  owned by US-4.3; it is neither a Group executable attribute nor writable Project state.
- A node and its subtree may move to any valid parent within the same Project.
- Tree cycles and cross-Project moves are prohibited.
- Only leaf WBS nodes may be deleted; parents must have all children moved or deleted first.
- Move invokes scheduler recalculation only when Automatic Scheduling is ON. Delete invokes recalculation only when Automatic Scheduling is ON and the deleted Task has scheduling input, generated projection, or dependency endpoints; Name-only and Role-only unfinished leaves without dependencies are pure structural deletes.
- Automatic Scheduling OFF preserves existing manual dates after structural changes.
- Manual timeline only when Automatic Scheduling is OFF and Project Open.
- Locked Project rejects every WBS/Task planning and structural mutation. Actual Date entry is the only Task mutation exception; protected timeline stays unchanged while Actual Allocation may recalculate impacted Open Projects.
- For an Open unfinished Task with Automatic Scheduling ON, leaving Role,
  Assignee, Effort, Lag, or Capacity Allocation Percentage requests the rollback-only schedule preview defined
  by US-6.1/US-6.3 when Role, valid Effort, valid Lag, and valid percentage are present. Assignee may be
  selected or explicitly cleared; clearing it still previews automatic-
  dependency reconciliation and the missing-Assignee unscheduled result. Save
  remains the only confirmed Task mutation.

---

## Approved Detailed Decisions

### Effort

- API accepts decimal hours in `0.5` hour increments, minimum `0.5` hour.
- The backend persists integer minutes as the source of truth; there is no MVP
  product maximum.
- The Task form accepts digits and one decimal point, preserves the draft while
  typing, rounds to the nearest `0.5` hour on blur, and repeats normalization
  and validation on Save. A non-empty Effort below `0.5` is rejected.

### Dependency and Lag

Dependency business rules and editor are owned by US-5.1. Lag persistence,
validation, UI field, and calculation are owned by US-6.1. Task Capacity
Allocation Percentage is owned by US-6.3. US-4.1 must preserve Lag and
percentage during structural conversion and atomic Task mutation but must not
duplicate their domain rules.

### Manual Timeline

- Fields are Execution Start/End and Commitment Start/End.
- Values are date-only (`YYYY-MM-DD` API and SQL `DATE`).
- Each pair is either completely empty or complete; partial pairs are invalid.
- End must be on or after Start. Execution and Commitment pairs are independent.
- Their relative relationship may produce a UI warning in manual mode but is
  not a hard validation.
- Manual dates are editable only for an Open Project with Automatic Scheduling
  OFF.
- US-6.3 derives fixed manual allocation from these authoritative dates and permits planned overcapacity; cross-Project automatic scheduling must honour that fixed allocation without changing the manual range.
- The Task form uses shared calendar behaviour. Execution and Commitment each
  use one date-range picker; Actual Date uses one date-range picker requiring both Actual Start and Actual End. These
  calendars share holiday/weekend marking, viewport-aware placement, scroll
  fallback, Escape and outside-click dismissal, and read-only behaviour with
  the rest of the product.
- Executable WBS has no Earliest Start, Start Constraint, Task Anchor, or other
  task-level scheduling anchor. Execution and Commitment dates remain scheduler
  outputs while Automatic Scheduling is ON.

### Project Scheduling Anchor

- Automatic Scheduling has exactly one nullable Project-level Scheduling Start
  Date, stored as SQL `DATE` and exposed as `YYYY-MM-DD`.
- US-4.1 prepares and reuses this Project data only; it does not calculate task
  dates or introduce a scheduling algorithm.
- When Automatic Scheduling is ON without this Project anchor, generated
  timelines remain empty and the UI exposes
  `Automatic Scheduling requires a Project Scheduling Start Date.`
- US-6.1 uses the anchor as the earliest working date for the first
  executable tasks without predecessors. Dependency-ready dates continue to
  govern downstream tasks.

### Role and Assignee

- Role and Assignee are optional; Effort does not require Assignee.
- Selecting an Assignee makes its Role the WBS Role.
- When Role is selected, available Assignees are Members with that Role.
- A Role change that conflicts with the selected Assignee requires the user to
  replace or clear Assignee. Backend rejects mismatched combinations and never
  clears Assignee implicitly.

### WBS Name

- Name is trimmed, required, not whitespace-only, and at most 200 characters.
- Sibling names are case-insensitively unique; equal names under different
  parents are allowed. Rename excludes the current node from the uniqueness
  check.
- Executable Task name and executable details are edited in one form and saved
  by one atomic backend request. The Home Task Name exposes one `Edit Task`
  action. Group rename is composed into the Home Group summary dialog.

### Scheduling Contracts

With Automatic Scheduling ON, creating a root or child using only Name does
not invoke the concrete portfolio scheduler and leaves generated dates empty.
Create invokes the scheduler only when it also converts an existing Executable
WBS and moves executable data or dependency endpoints. Sibling reorder, move,
Assignee change, Effort change, Lag change, Capacity Allocation Percentage change, and other established structural
conversions continue to invoke the concrete portfolio scheduler from US-6.1.
Delete invokes scheduling only when the removed Task has scheduling input,
generated projection, or dependency endpoints. Deleting a Name-only or
Role-only unfinished Task without dependencies is a pure structural mutation:
it hard-deletes the leaf, compacts sibling positions, and leaves existing Task,
dependency, allocation, and timeline state unchanged. Rename and Role-only
changes do not invoke scheduling when Assignee is unchanged. Cancelled, failed,
and manual-mode mutations do not invoke it. Actual Date and Reopen Task follow
the US-6.2 cross-project impact/recalculation contract. WBS mutation and required scheduling must
commit or roll back atomically.

### Actual Date, Completion, and Allocation

- Actual Start and Actual End are date-only, valid only on Executable WBS, and required together.
- Setting complete Actual Date completes the Executable WBS. A completed Task remains read-only for normal planning and executable-field mutation. While the Project is Open, established sibling reorder and Move to may change only its structural parent/position; Add Child and Delete remain unavailable.
- On an Open Project, Actual Date actualizes Execution/Commitment dates, creates Actual Allocation, uses Actual End for readiness, and recalculates impacted scope according to US-6.2.
- On a Locked Project, Actual Date may be entered without changing protected dates/dependencies/order; Actual Allocation is persisted and may recalculate impacted Open Projects.
- `US-4.2 — Reopen Completed Task` clears both Actual Start and Actual End only while the owning Project is Open. Locked Project must be reopened first.

### Reorder

- Move Up and Move Down atomically swap adjacent siblings under the same parent
  and are the keyboard-accessible reorder mechanism.
- The unavailable boundary direction is disabled.
- Home places both actions in row overflow.
- When the applied Role filter does not include every available Role option,
  including `No role`, Home disables both directions and explains
  `Show all roles to reorder WBS items.`
- Project filtering alone does not disable reorder.
- Reorder changes persisted sibling position and may affect scheduler ordering;
  generated Start/End dates never become the source of WBS row order.
- Move to is a separate operation that changes parent and moves the full subtree
  without accepting a sibling position.

### Move Into an Executable Destination

When an Executable destination contains data, Move atomically creates a first
child that receives all of the destination's executable data, places the moved
node/subtree as the next sibling, clears the destination's executable fields,
and thereby converts it to Grouping. Data must never be merged, overwritten, or
lost. The conversion child uses the destination name when sibling uniqueness
allows it; otherwise it uses the repository's deterministic `"<name>
(converted N)"` convention. Any failure rolls back hierarchy, positions,
executable data, dates, and confirmed cache state.

After US-5.1 introduces Task dependencies, this transaction also retargets all
incoming and outgoing dependency endpoints to the conversion child that
receives the executable data. The Group retains no dependency endpoint, and a
retarget failure rolls back both hierarchy and graph.

A completed leaf or subtree containing completed descendants may be moved.
Structural movement may change only parent/path and sibling position; completed
executable fields and Actual Date remain immutable.


### Group and Project Summary Ownership

US-4.3 owns the observable View Group and Edit Project summary. It recursively aggregates confirmed descendant Task Execution/Commitment timelines, separate schedule coverage, and Actual-End-based completed known Effort. Project is treated as logical WBS level `0`; all top-level WBS roots and descendants are included. US-4.1 continues to own the WBS tree and Group/Task conversion invariants.

The summary does not weaken the rule that Grouping WBS cannot own executable attributes. No aggregate date, Effort, percentage, or completion value is persisted on Group or added as writable Project state. The existing complete WBS tree read contract is reused; a new backend endpoint or summary-specific production query requires separate approval.

### Capacity Allocation

US-4.1 persists the executable Task field and integrates its form, lifecycle,
conversion, and atomic mutation behaviour. US-6.3 owns default `100`, integer
`1–100`, Assignee reset, migration, percentage rounding, concurrent planned
allocation, manual fixed allocation, and Actual Allocation non-interaction.
Per-Date projections and capacity consumption must not be reimplemented inside
the WBS feature.

### Home WBS Presentation

- Home row Name opens the reusable Project/Group/Task dialog directly over Home.
- Group summary and rename are proven in one Open-Project dialog; Locked Group
  remains read-only.
- Add Task and Add Child use distinct icon-only controls.
- Overflow eligibility is proven for Open Group, unfinished Task, completed Task,
  and Locked rows.
- Move Up/Down boundary behavior and restrictive-Role-filter disabled reason are
  proven against full persisted sibling order.
- Move to remains available under Role filtering and lists a valid hidden
  destination from the authoritative full tree.
- Completed Task may reorder/move but cannot Add Child/Delete.
- Delete remains confirmed and backend-authoritative.
- No acceptance workflow navigates through or renders the removed Project
  Structure page.

## Unresolved Questions

None.

---

## Implementation Evidence for the US-6.1 Requirement Delta

Concrete production paths, exact test names, per-AC local commands, and the
Three-Level Confidence readiness mapping are maintained in
`docs/project/automatic-scheduling-implementation-evidence.md`.

- Code Inspection: `IMPLEMENTED BY CODE INSPECTION`
- Unit/Integration: `AUTHORED — NOT RUN — LOCAL VALIDATION REQUIRED`
- Acceptance-Level: `AUTHORED — NOT RUN — LOCAL VALIDATION REQUIRED`
- Overall affected ACs: `IMPLEMENTED — LOCAL VALIDATION REQUIRED`

No automated validation result is recorded in this story.
