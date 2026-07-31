# US-4.1 Manage WBS

> **Product decision update — US-6.1:** WBS mutations use the concrete
> portfolio scheduler when Automatic Scheduling is ON. Dependency remains owned
> by US-5.1; Task Lag and generated timeline algorithms are owned by US-6.1.

## User Story

**As an** Engineering Lead
**I want** to manage the Work Breakdown Structure (WBS) of a project
**So that** project work can be decomposed into executable work items for future scheduling.

---

## Business Context

A Project consists of a hierarchical WBS. A WBS without children is automatically an Executable WBS. A WBS with children is automatically a Grouping WBS. There is no separate Task entity.

The UI uses product-friendly derived labels without changing this model:
**Project Structure** for the feature, **Task** for an Executable WBS, and
**Group** for a Grouping WBS. These are presentation labels, not persisted
entity types.

---

## Scope

### In Scope

- Create, rename, move and delete WBS
- Unlimited hierarchy
- Automatic conversion between Grouping and Executable WBS
- Manage executable attributes owned by this story: Role, Assignee, Effort, Manual Execution Timeline, Manual Commitment Timeline, Actual End
- Integrate with Dependency from US-5.1 and Lag/generated dates from US-6.1 without duplicating their business rules
- Validation, Persistence, API and UI

### Out of Scope

- Execution and Commitment scheduling algorithms; owned by US-6.1
- Forecast Scheduler
- Dependency editor and graph rules; owned by US-5.1
- Lag validation and calculation; owned by US-6.1

---

## Business Rules

- Unlimited WBS hierarchy.
- Leaf node automatically becomes Executable WBS.
- Parent node automatically becomes Grouping WBS.
- Grouping WBS cannot own executable attributes.
- When adding the first child to an Executable WBS, display a warning and move executable attributes to the first child.
- Manual Execution/Commitment Timeline is editable only when Automatic Scheduling is OFF.
- Actual End is only available on Executable WBS.

---

## Move Rules

- A WBS node may be moved to any valid parent within the same Project.
- Moving a node also moves its entire descendant subtree.
- A node cannot be moved under itself or under one of its descendants.
- The move must preserve a valid acyclic tree.
- The moved node receives a deterministic position among the destination parent's children.
- When the source parent has no remaining children after the move, it automatically becomes an Executable WBS.
- When the destination parent receives its first child, it automatically becomes a Grouping WBS.
- If the destination parent was an Executable WBS with executable attributes, the normal Executable-to-Grouping conversion and confirmation rules apply.
- Cross-Project moves are not allowed in this story.
- When Project Automatic Scheduling is ON, a confirmed move triggers the concrete portfolio scheduler from US-6.1 for the affected active scope.
- When Project Automatic Scheduling is OFF, existing manual Execution and Commitment dates remain unchanged.
- US-4.1 owns the mutation trigger and atomic coordination; US-6.1 owns the scheduling algorithm. Integrated acceptance tests must verify observable generated-date changes, not only port invocation.

## Delete Rules

- Only a leaf WBS node may be deleted.
- A Grouping WBS that still has one or more children cannot be deleted.
- The user must first delete or move all children before deleting that parent.
- Delete requires confirmation.
- Delete is a hard delete for a leaf that has no descendants, subject to Project lifecycle restrictions.
- Deleting a leaf may cause its parent to have no remaining children; that parent then automatically becomes an Executable WBS.
- Deleting a WBS must not cascade-delete descendants because deletion of a node with descendants is prohibited.
- When Automatic Scheduling is ON, confirmed deletion triggers the concrete portfolio scheduler from US-6.1 for the affected active scope.
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
17. With Automatic Scheduling ON, a successful delete invokes concrete portfolio recalculation and refreshes affected generated dates.
18. With Automatic Scheduling OFF, successful deletion preserves remaining manual dates.
19. Executable WBS supports Role, Assignee, Effort, Manual Execution Timeline, Manual Commitment Timeline and Actual End.
20. Grouping WBS cannot edit executable fields.
21. Manual timeline is editable only when Automatic Scheduling is OFF.
22. Actual End is only available for Executable WBS.
23. With Automatic Scheduling ON, create, sibling reorder, Assignee change, Effort change, and structural conversion invoke concrete portfolio recalculation.
24. Rename and Role-only changes do not invoke scheduling when Assignee is unchanged.
25. WBS mutation and required scheduling are atomic; scheduling failure rolls back hierarchy, executable data, generated dates, and confirmed UI state.
26. On an Open unfinished automatic Task, blur previews generated dates and automatic dependency ownership without persisting the Task when Role, valid Effort, and valid Lag are present. Assignee may be selected or explicitly cleared: a selected Assignee previews its reconciled schedule, while a cleared Assignee still calls preview to remove stale automatic ownership and return a missing-Assignee unscheduled projection. Missing or invalid Role, Effort, or Lag makes no preview request.

---

## Test Cases

### Happy Path

- Create root WBS.
- Create nested WBS.
- Convert Executable to Grouping.
- Rename WBS.
- Move a leaf to another Grouping WBS.
- Move a Grouping WBS together with its full subtree.
- Move the last child away and verify the source parent becomes Executable.
- Move into an Executable destination and confirm conversion to Grouping.
- Move with Automatic Scheduling ON and verify generated Execution/Commitment dates are recalculated.
- Move with Automatic Scheduling OFF and verify manual dates remain unchanged.
- Delete a leaf.
- Delete the last child and verify the parent becomes Executable.
- Delete with Automatic Scheduling ON and verify generated Execution/Commitment dates are recalculated.
- Delete with Automatic Scheduling OFF and verify remaining manual dates are unchanged.
- Edit executable attributes.
- Enter Actual End.

### Validation

- Attempt executable fields on Grouping WBS.
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
- Actual End persists.
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
US-6.1 owns concrete Execution/Commitment scheduling, daily allocation, Lag, and
Auto Dependency. US-4.1 must call the concrete scheduler when available and may
not claim completion from a no-op adapter after US-6.1 is implemented.

---

## Locked Product Decisions

- WBS is the only work-item aggregate.
- Task is represented by Executable WBS.
- Unlimited hierarchy.
- Automatic Grouping/Executable conversion.
- A node and its subtree may move to any valid parent within the same Project.
- Tree cycles and cross-Project moves are prohibited.
- Only leaf WBS nodes may be deleted; parents must have all children moved or deleted first.
- Move/delete invokes scheduler recalculation only when Automatic Scheduling is ON.
- Automatic Scheduling OFF preserves existing manual dates after structural changes.
- Manual timeline only when Automatic Scheduling is OFF.
- For an Open unfinished Task with Automatic Scheduling ON, leaving Role,
  Assignee, Effort, or Lag requests the rollback-only schedule preview defined
  by US-6.1 when Role, valid Effort, and valid Lag are present. Assignee may be
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
validation, UI field, and calculation are owned by US-6.1. US-4.1 must preserve
both fields during structural conversion and atomic Task mutation but must not
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
- The Task form uses shared calendar behaviour. Execution and Commitment each
  use one date-range picker; Actual End uses one single-date picker. These
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

- Name is trimmed, required, not whitespace-only, and at most 100 characters.
- Sibling names are case-insensitively unique; equal names under different
  parents are allowed. Rename excludes the current node from the uniqueness
  check.
- Executable Task name and executable details are edited in one form and saved
  by one atomic backend request. The Project Structure row exposes only one
  `Edit Task` action. Group rename remains a structural Group action.

### Scheduling Contracts

With Automatic Scheduling ON, create root, create child, sibling reorder, move,
delete, Assignee change, Effort change, and structural conversion invoke the
concrete portfolio scheduler from US-6.1. Rename and Role-only changes do not
when Assignee is unchanged. Cancelled, failed, and manual-mode mutations do not
invoke it. Actual End invokes a separate Forecast recalculation contract. WBS
mutation and required scheduling must commit or roll back atomically.

### Actual End and Completion

- Actual End is date-only and only valid on Executable WBS.
- Setting Actual End completes the Executable WBS. A completed Task remains
  read-only for normal planning, executable-field, and structural mutation.
- `US-4.2 — Reopen Completed Task` is the only explicit exception that may
  clear Actual End. Reopen uses a dedicated command; generic Task update may
  not clear Actual End or bypass the completed-task read-only invariant.

### Reorder

- Move Up and Move Down atomically swap adjacent siblings and are the
  keyboard-accessible reorder mechanism.
- The unavailable boundary direction is disabled.
- Move is a separate operation that changes parent and moves the full subtree.

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
executable fields and Actual End remain immutable.

### Capacity Allocation

US-4.1 stores only Effort minutes and date-only manual boundaries. Per-working-
date allocation projections and all capacity consumption belong to US-6.1 and
must not be reimplemented inside the WBS feature.

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
