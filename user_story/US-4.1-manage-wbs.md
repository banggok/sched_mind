# US-4.1 Manage WBS

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
- Manage executable attributes: Role, Assignee, Effort, Dependency, Lag, Manual Execution Timeline, Manual Commitment Timeline, Actual End
- Validation, Persistence, API and UI

### Out of Scope
- Execution Scheduler
- Commitment Scheduler
- Forecast Scheduler
- Timeline calculation

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
- When Project Automatic Scheduling is ON, a confirmed move triggers the scheduler integration contract to recalculate the affected Project.
- When Project Automatic Scheduling is OFF, existing manual Execution and Commitment dates remain unchanged.
- The real scheduling algorithm remains deferred to Epic 6; this story must invoke the established scheduling port and verify the invocation through automated tests.

## Delete Rules

- Only a leaf WBS node may be deleted.
- A Grouping WBS that still has one or more children cannot be deleted.
- The user must first delete or move all children before deleting that parent.
- Delete requires confirmation.
- Delete is a hard delete for a leaf that has no descendants, subject to Project lifecycle restrictions.
- Deleting a leaf may cause its parent to have no remaining children; that parent then automatically becomes an Executable WBS.
- Deleting a WBS must not cascade-delete descendants because deletion of a node with descendants is prohibited.
- When Automatic Scheduling is ON, confirmed deletion triggers the scheduler integration contract for the affected Project.
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
11. With Automatic Scheduling ON, a successful move invokes Project recalculation through the scheduler port.
12. With Automatic Scheduling OFF, a successful move preserves existing manual dates.
13. Only a leaf WBS may be deleted.
14. A WBS with children cannot be deleted.
15. The delete conflict explains that children must be moved or deleted first.
16. Deleting the last child converts the parent to Executable.
17. With Automatic Scheduling ON, a successful delete invokes Project recalculation through the scheduler port.
18. With Automatic Scheduling OFF, successful deletion preserves remaining manual dates.
19. Executable WBS supports Role, Assignee, Effort, Manual Execution Timeline, Manual Commitment Timeline and Actual End.
20. Grouping WBS cannot edit executable fields.
21. Manual timeline is editable only when Automatic Scheduling is OFF.
22. Actual End is only available for Executable WBS.

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
- Move with Automatic Scheduling ON and verify the scheduler port is invoked once.
- Move with Automatic Scheduling OFF and verify manual dates remain unchanged.
- Delete a leaf.
- Delete the last child and verify the parent becomes Executable.
- Delete with Automatic Scheduling ON and verify the scheduler port is invoked.
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
- Scheduler-port invocation
- Manual-date preservation
- Concurrency and rollback
- Validation

---

## Deferred Implementation

This story prepares the complete WBS aggregate only. Automatic scheduling is deferred to Epic 6 (Scheduling Engine).

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

Dependency and Lag are removed from US-4.1 and deferred completely to Epic 5.
US-4.1 does not persist or present them and does not calculate a dependency
graph.

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
- Epic 6 will use the anchor as the earliest working date for the first
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
Project scheduling port. Rename and Role-only changes do not. Cancelled, failed,
and manual-mode mutations do not invoke it. Actual End invokes a separate
Forecast recalculation contract. US-4.1 implements only port invocation and
contract tests; algorithms remain deferred to Epic 6.

### Actual End and Completion

- Actual End is date-only and only valid on Executable WBS.
- Setting it completes the WBS permanently for MVP. It cannot be edited or
  cleared, and every later planning or structural mutation affecting that
  completed WBS is rejected. Correction/reopen is out of scope.

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

A completed leaf or subtree containing completed descendants may be moved.
Structural movement may change only parent/path and sibling position; completed
executable fields and Actual End remain immutable.

### Capacity Allocation

US-4.1 stores only Effort minutes and date-only manual boundaries. Per-working-
date allocation projections and all capacity consumption belong to Epic 6 and
must not be introduced by this story.

## Unresolved Questions

None.
