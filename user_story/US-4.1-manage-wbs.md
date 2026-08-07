# US-4.1 Manage WBS

> **Product decision update — US-4.4:** Grouping WBS may own Group-scoped scheduling configuration and local Lock/Reopen without becoming Executable. Task manual/generated editability, WBS mutation eligibility, and scheduler triggers resolve from effective Group/Project configuration and lifecycle. Structural move/conversion across Group boundaries follows US-4.4, including the guard against silently discarding a Group scheduling override on last-child conversion.

> **Product decision update — US-6.4:** Task mutations and rollback-only schedule
> preview recalculate dates, allocations, and unscheduled state without creating,
> removing, or previewing automatic dependency ownership. Dependency remains
> explicit manual data owned by US-5.1. This supersedes every automatic
> dependency preview/reconciliation clause in this story.

> **Product decision update — US-7.1:** Home Portfolio Gantt is the canonical
> active-Project WBS presentation surface. The standalone Project Structure
> entry point is removed. Home reuses the same Project, Group, and Task forms and
> the same Add Sibling/Add Child, rename, reorder, move, and delete application
> use cases. Home may invoke sibling reorder through Move Up/Down or a dedicated
> drag handle; both paths use the same authoritative reorder command. Add
> Sibling introduces an authoritative insert-after anchor but
> does not create a second WBS aggregate. The Gantt cells, bars, and dependency
> arrows remain read-only; US-4.1 remains the owner of WBS mutation and
> conversion rules.

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

> **Product decision update — US-6.5:** Editable unfinished Task Details ranks
> Assignee candidates through one latest batch simulation. Role and Assignee
> appear immediately after Lag, in that order and before timeline/dependency
> controls; current
> Role-mismatched Assignee remains visible until replaced/cleared, and Capacity
> Allocation Percentage is preserved on first selection, reassignment, and clear.
> US-6.5 owns ranking and simulation; this story owns form composition and atomic
> Task mutation integration.

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

- Create root, child, and explicitly positioned sibling WBS; rename, sibling
  reorder through command or Home drag handle, move, and delete WBS
- Unlimited hierarchy
- Automatic conversion between Grouping and Executable WBS
- Manage executable attributes owned by this story: Role, Assignee, Effort, Capacity Allocation Percentage integration, Manual Execution Timeline, Manual Commitment Timeline, Actual Start, Actual End
- Integrate Task Capacity Allocation Percentage from US-6.3 without duplicating its scheduling or migration rules
- Integrate Assignee recommendation from US-6.5 without duplicating ranking or simulation rules
- Integrate with Dependency from US-5.1 and Lag/generated dates from US-6.1 without duplicating their business rules
- Validation, Persistence, API and UI

### Out of Scope

- Execution and Commitment scheduling algorithms; owned by US-6.1
- Forecast Scheduler
- Dependency editor and graph rules; owned by US-5.1
- Lag validation and calculation; owned by US-6.1
- Task Capacity Allocation Percentage calculation and migration; owned by US-6.3
- Assignee recommendation ranking, simulation, and candidate metrics; owned by US-6.5
- Recursive Group and whole-Project timeline/effort-completion summary; owned by US-4.3

---

## Business Rules

- Unlimited WBS hierarchy.
- Leaf node automatically becomes Executable WBS.
- Parent node automatically becomes Grouping WBS.
- Grouping WBS cannot own executable attributes. A Group may display the
  read-only recursive descendant summary defined by US-4.3; the summary is not
  persisted on the Group.
- Add Child retains the existing workflow. On Project it creates a root Task;
  on WBS it creates beneath the selected row, and adding the first child to an
  Executable WBS displays a warning and moves executable attributes to the
  first child.
- Add Sibling creates a new Executable WBS under the selected non-Project
  row's same parent and immediately after that row in full persisted sibling
  order.
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
- Home replaces the single add icon with labelled creation actions: Project
  shows `Add Child`; eligible Group/unfinished Task shows
  `Add Sibling | Add Child`; completed Task shows `Add Sibling` only.
- The `...` overflow trigger remains independently anchored at the right edge of
  the row and is not moved beside the creation labels.
- Move Up, Move Down, Move to, and Delete remain secondary actions in the Home
  row overflow menu.
- Eligible Task and Group rows also expose a dedicated leading drag handle for
  same-parent sibling reorder. The Project row has no drag handle, and the rest
  of the row is not a drag surface.
- Home invokes reusable dialogs/use cases directly and must not mount an
  intermediate Project Structure page.
- Acceptance-level evidence for active WBS management must exercise the Home
  workflow rather than the removed Project Structure workflow.

---

## Create Position Rules

- Project is logical WBS level `0` and has no Add Sibling action. Project Add
  Child opens the standard Create Task dialog and creates a root WBS using the
  existing root-create position behaviour.
- Add Child on Group or unfinished Task keeps the existing child-create and
  Executable-to-Grouping conversion behaviour. Completed Task cannot Add Child.
- Add Sibling is available for Group, unfinished Task, and completed Task while
  the owning Project is Open. Locked and Closed Projects reject it.
- Add Sibling opens the same Create Task dialog as Add Child and always creates
  a new Executable WBS; the anchor's Group/Task/completed state is not copied to
  the new Task.
- The create command expresses exactly one structural intent: existing
  root/child creation or sibling insertion. `insert_after_wbs_id` is mutually
  exclusive with a client-selected parent/position; when it is present, the
  backend derives both parent and position from the authoritative anchor.
- The command carries the selected WBS identity as an insert-after anchor. The
  backend resolves the anchor from the authoritative tree; the client must not
  calculate or submit a position based only on filtered/rendered rows.
- The new Task receives the anchor's authoritative parent and is inserted
  immediately after the anchor in full persisted sibling order. Every later
  sibling shifts atomically while retaining its relative order.
- An expanded Group's descendants remain children. Its new sibling renders only
  after the complete subtree; insertion never occurs between the Group and its
  first child.
- Role filtering does not change the authoritative parent or position. A created
  Task may be absent from the refreshed filtered Home projection.
- If the anchor no longer exists, the Project is no longer Open, or the anchor
  cannot be resolved consistently when the transaction executes, the command
  fails without creating a Task or changing any sibling position.
- Concurrent create/reorder operations must preserve one gap-free deterministic
  sibling order. The persistence boundary may serialize them or return a
  recoverable conflict, but may not lose a Task, duplicate a position, or create
  the Task under a different parent than the resolved anchor.
- Task creation, sibling-position updates, executable conversion when applicable,
  required scheduling, and confirmed projection state commit or roll back as one
  unit.
- Scheduler invocation follows the created Task's established scheduling impact.
  A Name-only Add Sibling does not trigger scheduling merely because later
  numeric positions shift: existing siblings keep the same relative priority.
  A created Task with scheduling inputs or any other established concrete
  scheduling trigger follows US-6.1.
- Existing sibling-name uniqueness, validation, impact confirmation, and error
  mapping remain authoritative.

---

## Sibling Reorder Rules

- Reorder is valid only for a Task or Group in an Open Project. Project level
  `0`, Locked Projects, and Closed Projects cannot be reordered. A completed
  Task in an Open Project remains structurally reorderable.
- Move Up/Down swaps one adjacent sibling. Home drag reorder may place the source
  immediately before or after any sibling under the same authoritative parent.
- Drag reorder is sibling reorder only. It never changes parent, never moves
  across Projects, and never replaces the existing Move to workflow.
- A Group is reordered as one subtree. Every descendant keeps its parent and
  internal sibling order while the complete subtree changes position together.
- The reorder command carries source WBS identity, target sibling identity, and
  `before` or `after` placement. The backend resolves the authoritative shared
  parent and resulting positions; the client must not submit an arbitrary index
  derived from rendered rows.
- Source and target must still exist under the same parent in the same Open
  Project when the transaction executes. Stale, cross-parent, cross-Project,
  self/descendant, or otherwise invalid placement fails without changing order.
- Dropping into the source's existing position is a no-op and must not invoke
  persistence, scheduling, or impact confirmation.
- A Restrictive Role Filter disables drag reorder and Move Up/Down because hidden
  siblings make placement ambiguous. Project filtering alone does not disable
  reorder.
- Successful reorder updates one gap-free persisted sibling order. Derived WBS
  numbers, subtree priority, summaries, dependencies, and affected scheduling
  refresh through the existing reorder contract.
- Position updates, required scheduling, confirmed projection refresh, and error
  handling are atomic. Failure or concurrency conflict restores the confirmed
  original order and cannot detach or partially move descendants.

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
- Hard-deleting an eligible leaf Task removes every Sprint Task relation for that Task in the same transaction. This relation cleanup does not delete or mutate any Sprint and does not change scheduler-impact eligibility.
- When Automatic Scheduling is ON, confirmed deletion triggers the concrete portfolio scheduler from US-6.1 only when the deleted Task has scheduling impact: Assignee, Effort, non-zero Lag, generated Execution/Commitment state, an automatic unscheduled projection, or dependency endpoints. Deleting a Name-only or Role-only unfinished Task with no dependency does not invoke scheduling. The presence of completed Tasks elsewhere in the Project does not change this trigger rule.
- When Automatic Scheduling is OFF, remaining manual dates stay unchanged.

---

## Acceptance Criteria

1. Engineering Lead can create root, child, and sibling WBS.
1A. Add Sibling accepts an existing non-Project WBS as an authoritative
    insert-after anchor, creates a Task under the same parent, and places it
    immediately after that anchor in full persisted sibling order.
1B. Add Sibling on an expanded Group places the Task after the complete
    subtree, not between the Group and its children.
1C. Completed Task in an Open Project may Add Sibling but not Add Child;
    Locked and Closed Project contexts reject both structural mutations.
1D. Add Child preserves existing behaviour, including Project root creation
    and executable-to-Grouping conversion.
1E. Create and position shifting are atomic; Cancel or failed validation,
    scheduling, persistence, or stale-anchor resolution creates nothing and
    leaves order unchanged.
1F. Sibling insertion accepts an authoritative anchor identity, not a
    client-computed sibling index or independently editable parent/position.
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
22. Complete Actual Date is only available for Executable WBS. On an Open Project it actualizes timelines and creates Actual Allocation through US-6.2; on a Locked Project it preserves baseline while Actual Allocation may recalculate the transitive Open scope; only resulting Executable Task Execution/Commitment date changes warn.
23. Locked Project rejects every WBS/Task planning or structural mutation, including create, rename, edit, delete, move, reorder, conversion, and Task Reopen. Complete Actual Date entry is the only Task mutation exception.
24. With Automatic Scheduling ON, a root or child Task created only with Name and without structural conversion persists with empty generated dates and does not invoke concrete portfolio recalculation. Create that converts an existing Executable WBS and moves executable data or dependency endpoints, sibling reorder, Assignee change, Effort change, Lag change, and other established structural conversions still invoke concrete portfolio recalculation.
25. Rename and Role-only changes do not invoke scheduling when Assignee is unchanged.
26. WBS mutation and required scheduling are atomic; scheduling failure rolls back hierarchy, executable data, generated dates, and confirmed UI state.
27. On an Open unfinished automatic Task, blur previews generated dates and automatic dependency ownership without persisting the Task when Role, valid Effort, and valid Lag are present. Assignee may be selected or explicitly cleared: a selected Assignee previews its reconciled schedule, while a cleared Assignee still calls preview to remove stale automatic ownership and return a missing-Assignee unscheduled projection. Missing or invalid Role, Effort, or Lag makes no preview request.
28. Home is the canonical active WBS management surface and the standalone Project Structure entry point is removed.
29. Activating Group Name opens one summary/rename dialog; activating Task Name opens Edit Task for unfinished, completed, and Locked Tasks.
30. Home uses labelled Add Sibling/Add Child controls while the `...` overflow
    remains anchored at the right edge; Move Up, Move Down, Move to, and Delete
    remain in overflow according to eligibility.
31. Move Up/Down swap only adjacent siblings under the same parent and are disabled at the unavailable boundary.
31A. Eligible Open-Project Task and Group rows support same-parent drag reorder
     through a dedicated handle; Project and Locked/Closed rows do not.
31B. Drag reorder places the complete source Task/Group subtree immediately
     before or after an authoritative sibling and never changes parent.
31C. A Group's descendants follow the Group as one subtree while retaining their
     hierarchy and internal order.
32. Home disables Move Up/Down and drag reorder while a restrictive Role filter
    may hide siblings and explains that all Roles must be shown; Move to remains
    available.
33. Move to changes parent only, offers no sibling-position input, and resolves all valid destinations from the full authoritative Project tree.
34. A completed Task in an Open Project may Add Sibling, reorder, or move under
    existing BAU, but cannot Add Child or Delete; its executable data and Actual
    Date remain immutable.
35. Active WBS acceptance-level tests exercise `Home → row action/dialog → confirm → refreshed Home` and prove that no Projects/Project Structure background navigation occurs.
36. Successful eligible Task deletion atomically removes every Sprint Task relation for that Task; any delete failure preserves both the Task and its Sprint relations.
37. First assignment and Edit Task expose the US-6.5 ranked Assignee list when Role, Effort, Capacity Allocation Percentage, and Lag are valid.
38. Role and Assignee appear immediately after Lag, in that order and before timeline/dependency controls; accessible tab order matches the visible order.
39. Selecting, changing, or clearing Assignee preserves Task Capacity Allocation Percentage.
40. Role change keeps a mismatched current Assignee visible until user replaces or clears it; backend mismatch validation remains authoritative.

---

## Test Cases

### Happy Path

- Create root WBS with Name only and verify scheduler is not invoked and generated dates remain empty.
- Create nested WBS under an existing Group with Name only and verify scheduler is not invoked.
- Add Sibling after a top-level Task and a nested Task and verify same-parent
  insertion immediately after the anchor.
- Add Sibling after an expanded Group and verify the new Task appears after
  the complete subtree.
- Add Sibling beside a completed Task and verify completed data and Actual Date
  remain unchanged.
- Cancel Add Sibling and verify no Task or sibling-position change is persisted.
- Add Sibling while Role filtering hides other siblings and verify position is
  calculated against the full persisted order.
- Create a Name-only sibling and verify later position numbers shift without
  invoking scheduling or changing existing relative scheduler priority.
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
- Delete a Task associated with Planned and Started Sprints and verify all Sprint Task relations are removed in the same transaction without deleting either Sprint.
- Edit executable attributes, including Capacity Allocation Percentage.
- Complete Effort, Capacity Allocation Percentage, Lag, and Role, then verify the Assignee control shows one latest US-6.5 batch ranking.
- Select/change/clear Assignee and verify Capacity Allocation Percentage is preserved.
- Change Role to conflict with current Assignee and verify the current value remains visible until replaced/cleared.
- Enter complete Actual Date and verify allocation.

### Validation

- Attempt executable fields on Grouping WBS.
- Leave Role, Effort, Capacity Allocation Percentage, or Lag invalid and verify recommendation is not requested while alphabetic Assignee selection remains usable.
- Verify completed, Locked, Closed, and Group contexts do not request recommendation.
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
- Delete or close the anchor/Project before Add Sibling commit and verify the
  command fails without creation or position change.
- Concurrent Add Sibling/reorder operations preserve one gap-free sibling
  order without lost Tasks or duplicate positions.
- Concurrent move/delete operations preserve one valid tree.

### Regression

- Existing WBS data is preserved after unrelated moves.
- Child ordering remains deterministic.
- Executable-to-Grouping conversion preserves executable data in the first child.
- Grouping-to-Executable conversion does not invent executable field values.
- Actual Start and Actual End persist atomically.
- Failed create, move, or delete leaves hierarchy, positions, dates, and Sprint
  Task relations unchanged.
- Older in-flight list/tree responses cannot restore stale hierarchy after mutation.

---

## Required Automated Tests

- Domain
- Application
- Repository
- API
- Frontend
- WBS conversion
- Same-parent insert-after creation and sibling position shifting
- Expanded-Group sibling placement and filtered full-order insertion
- Concurrent insertion/reorder conflict safety and create rollback
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
- Add Sibling creates a new Task immediately after an existing non-Project
  anchor under the same authoritative parent. Project offers Add Child only.
- Completed Task in an Open Project may Add Sibling but may not Add Child.
- Add Sibling position is resolved from the full persisted order, never the
  filtered visible subset, and create/position/scheduling commit atomically.
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
- For first assignment and Edit Task, Assignee recommendation follows US-6.5:
  one latest side-effect-free batch, current Task allocation removed from the
  baseline, and deterministic ranked metadata. Automatic ON reuses the concrete
  scheduler; Automatic OFF uses advisory simulation without changing manual dates.
- Capacity Allocation Percentage remains unchanged when Assignee is selected,
  changed, or cleared.

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
- The Task form must not clear a conflicting current Assignee automatically; it
  remains visible with the US-6.5 mismatch warning until user replaces or clears it.
- For editable unfinished Task planning, visible and accessible order is Effort,
  Capacity Allocation Percentage, Lag, Role, then Assignee. Applicable manual or
  generated timelines and confirmed dependency controls appear after Assignee.
  Actual Date remains a separate lifecycle action and may remain after the primary
  Save action.
- The Assignee control consumes the batch recommendation owned by US-6.5 when
  its required draft inputs are valid; incomplete or failed recommendation keeps
  deterministic alphabetic selection available.

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
dependency, allocation, and timeline state unchanged, while any Sprint Task
relations for the deleted Task are removed atomically. Rename and Role-only
changes do not invoke scheduling when Assignee is unchanged. Cancelled, failed,
and manual-mode mutations do not invoke it. Actual Date and Reopen Task follow
the US-6.2 cross-project impact/recalculation contract. WBS mutation and required scheduling must
commit or roll back atomically.

### Actual Date, Completion, and Allocation

- Actual Start and Actual End are date-only, valid only on Executable WBS, and required together.
- Setting complete Actual Date completes the Executable WBS. A completed Task remains read-only for normal planning and executable-field mutation. While the Project is Open, Add Sibling may create a separate Task beside it and established sibling reorder/Move to may change only its structural parent/position; Add Child and Delete remain unavailable.
- On an Open Project, Actual Date actualizes Execution/Commitment dates, creates Actual Allocation, uses Actual End for readiness, and recalculates impacted scope according to US-6.2.
- On a Locked Project, Actual Date may be entered without changing protected dates/dependencies/order; Actual Allocation is persisted and may recalculate impacted Open Projects.
- `US-4.2 — Reopen Completed Task` clears both Actual Start and Actual End only while the owning Project is Open. Locked Project must be reopened first.

### Reorder

- Move Up and Move Down atomically swap adjacent siblings under the same parent
  and remain the keyboard-accessible reorder mechanism.
- Home additionally exposes a dedicated drag handle on eligible Task and Group
  rows. The rest of the row, Name, expand/collapse control, creation labels,
  timeline, and `...` overflow are not drag surfaces.
- Drag reorder accepts an authoritative sibling anchor plus before/after
  placement and can cross multiple sibling positions in one operation. It cannot
  change parent or Project.
- A Group and its full descendant subtree move as one structural block.
  Descendant parent IDs and relative internal order remain unchanged.
- The unavailable Move Up/Down boundary direction is disabled. A same-position
  drag drop is a no-op.
- Home places Move Up/Down in row overflow as the keyboard and non-drag fallback.
- When the applied Role filter does not include every available Role option,
  including `No role`, Home disables Move Up/Down and drag reorder and explains
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
`1–100`, Assignee-selection preservation, migration, percentage rounding, concurrent planned
allocation, manual fixed allocation, and Actual Allocation non-interaction.
Per-Date projections and capacity consumption must not be reimplemented inside
the WBS feature.

### Home WBS Presentation

- Home row Name opens the reusable Project/Group/Task dialog directly over Home.
- Group summary and rename are proven in one Open-Project dialog; Locked Group
  remains read-only.
- Project shows labelled Add Child; eligible Group/unfinished Task shows
  `Add Sibling | Add Child`; completed Task shows Add Sibling only.
- The creation labels are separate controls and the `...` overflow remains
  independently anchored at the right edge of the row.
- Add Sibling exact same-parent insertion is proven against full authoritative
  sibling order, including an expanded Group and a Restrictive Role Filter.
- Overflow eligibility is proven for Open Group, unfinished Task, completed Task,
  and Locked rows.
- Move Up/Down boundary behavior and restrictive-Role-filter disabled reason are
  proven against full persisted sibling order.
- Drag reorder is proven for Task and expanded Group subtree movement, valid
  before/after sibling placement, same-position/cancel no-op, invalid
  cross-parent rejection, and atomic rollback.
- Move to remains available under Role filtering and lists a valid hidden
  destination from the authoritative full tree.
- Completed Task may Add Sibling/reorder/move but cannot Add Child/Delete.
- Delete remains confirmed and backend-authoritative.
- Successful eligible Task deletion atomically removes Sprint Task relations owned by US-8.1; rollback preserves both Task and relations.
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
