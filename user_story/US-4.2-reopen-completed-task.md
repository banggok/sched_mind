# US-4.2 — Reopen Completed Task

> **Product decision update — US-6.2:** Completion uses a required Actual Date
> pair (`Actual Start` and `Actual End`). Reopen Task atomically clears both
> fields and removes Actual Allocation. Reopen is allowed only while the owning
> Project is Open and follows the US-6.2 transitive recalculation plus timeline-only
> cross-project impact preview/confirmation contract.

> **Product decision update — US-7.1:** Completed Task Name on Home opens the
> shared Edit Task dialog directly over Home. The action remains **Edit Task**,
> not View Task, because eligible Open completed Tasks may Reopen from inside the
> dialog. Locked Task also opens Edit Task for Actual Date, but Reopen remains
> unavailable until Project Reopen. No Project Structure route or background is
> used.

## 1. User Story

**Sebagai** Engineering Lead,
**Saya ingin** membuka kembali Task yang sudah completed,
**Sehingga** Actual Date yang salah dapat dibatalkan dan Task yang sebenarnya belum selesai dapat kembali diperlakukan sebagai unfinished work tanpa membuat Task baru.

---

## 2. Business Context

Task adalah Executable WBS. Completion ditentukan oleh complete Actual Date:

```text
Actual Start != null
Actual End   != null
```

Kedua field harus sama-sama terisi atau sama-sama kosong. Partial Actual Date
adalah invalid state.

Reopen Task adalah dedicated command. Command ini bukan generic Task edit,
bukan Project Reopen, dan bukan pembuatan Task pengganti. Reopen mempertahankan
Task identity, hierarchy, planning fields, dan dependency, lalu menghapus Actual
Date dan Actual Allocation atomically.

---

## 3. Terminology

| Product Term | Domain Meaning |
| --- | --- |
| Completed Task | Executable WBS dengan complete Actual Date pair |
| Unfinished Task | Executable WBS tanpa Actual Date |
| Reopen Task | Dedicated command yang menghapus Actual Start dan Actual End |
| Reopen Project | Lifecycle command terpisah yang membuka Locked/Closed Project |
| Actual Allocation | Daily historical capacity rows derived from Actual Date and Effort |

---

## 4. Scope

### 4.1 In Scope

- Contextual `Reopen Task` action inside the shared Edit Task dialog for an eligible completed Task.
- Open that dialog from completed Task Name on Home without route/background change.
- Confirmation before mutation.
- Clear Actual Start and Actual End together.
- Remove/rebuild Actual Allocation as required by US-6.2.
- Preserve Task ID, Project ID, hierarchy, Name, Role, Assignee, Effort, Lag,
  Execution/Commitment fields, and dependency endpoints.
- Change derived completion state to unfinished.
- Allow Reopen only when owning Project is Open.
- Reject Locked and Closed Project.
- Run US-6.2 transitive recalculation simulation before save.
- Require confirmation only when another Open Project has an Executable Task Execution/Commitment Start or End change.
- Block only when another Locked Project would counterfactually require a protected Task timeline date change.
- Server-side impact revalidation, atomic recalculation, rollback, concurrency,
  stale-response protection, accessibility, and Three-Level Confidence.

### 4.2 Out of Scope

- Editing Actual Date to another range inside Reopen command.
- Reopen Group.
- Bulk Task Reopen.
- Implicit Project status change.
- Remaining-effort or percentage-complete field.
- Dependency mutation.
- Forecast implementation.
- User-editable allocation rows.

---

## 5. Product Rules

### 5.1 Eligibility

Reopen Task is available only when:

- target is an Executable WBS;
- complete Actual Date pair exists;
- owning Project status is Open.

Locked Project must use Project Reopen first. Closed Project must use the
approved Project lifecycle command first. Backend rejects direct API bypass.

### 5.2 Home Edit Task Entry

- Completed Task row uses the same **Edit Task** primary action as unfinished
  Task.
- Reopen Task is located inside Edit Task, not as a permanent row icon.
- Completed Task row may expose structural Move Up/Down/Move to through Home
  overflow according to US-4.1, but it exposes no Add Child or Delete.
- Locked Task Name still opens Edit Task for US-6.2 Actual Date; Reopen is hidden
  or disabled with the Project Reopen requirement.
- Save/Close/Reopen success keeps Home as the route and visible background.

### 5.3 Dedicated Atomic Command

Successful Reopen changes exactly:

```text
Actual Start: <confirmed date> → null
Actual End:   <confirmed date> → null
Actual Allocation rows: delete/reconcile
```

It preserves:

- Task/WBS ID;
- Project and Parent IDs;
- hierarchy and sibling position;
- Name, Role, Assignee, Effort, and Lag;
- Execution and Commitment values until confirmed recalculation applies;
- incoming and outgoing dependency endpoints/ownership;
- Created At and every unrelated field.

`Updated At`, schedule version, and affected projection versions may change
according to repository conventions.

### 5.4 Scheduling Impact

Reopen removes historical capacity consumption and returns the Task to
unfinished scheduling. Therefore it is a scheduling-impacting mutation:

1. Server simulates the proposed Reopen and transitive recalculation scope.
2. Current Project is excluded from the warning list.
3. Another Open Project requires Project-name confirmation only when an Executable Task Execution/Commitment Start or End changes.
4. A Locked Project blocks Reopen only when counterfactual simulation requires a protected Task timeline date change; allocation/readiness-only pressure does not block.
5. Confirm performs server-side revalidation.
6. Reopen, Actual Allocation removal, dependency reconciliation, unfinished
   scheduling, and persistence commit or rollback together.

The reopened Task uses original Effort, not estimated remaining effort.

### 5.5 Dependency Preservation

Reopen never creates, deletes, retargets, or changes ownership of dependency.
The same immutable Task ID remains the endpoint. Because the Task is unfinished
again, normal dependency readiness applies on confirmed recalculation.

### 5.6 Failure and Concurrency

- Duplicate submission sends one command.
- Concurrent Reopen allows at most one completed→unfinished transition.
- Persistence/scheduler/concurrency failure preserves both Actual Date fields,
  Actual Allocation, completion state, and every unrelated projection.
- Old detail/tree/allocation responses cannot restore stale completed state after
  successful Reopen or stale unfinished state after failure.

---

## 6. API and Error Contract

The dedicated endpoint remains owned by implementation architecture. Minimum
stable error concepts:

| Error Code | HTTP | Condition |
| --- | ---: | --- |
| `TASK_NOT_COMPLETED` | 409 | Target lacks complete Actual Date |
| `PROJECT_LOCKED_READ_ONLY` | 409 | Owning Project Locked |
| `PROJECT_CLOSED_READ_ONLY` | 409 | Owning Project Closed |
| `SCHEDULING_IMPACT_CONFIRMATION_REQUIRED` | 409 | Other Open Projects impacted |
| `SCHEDULING_LOCKED_PROJECT_IMPACT` | 409 | Another Locked Project impacted |
| `SCHEDULING_IMPACT_STALE` | 409 | Preview/version changed before confirm |

Generic Task update may not clear either Actual field as a bypass.

---

## 7. Acceptance Criteria

### AC-1 — Eligibility uses complete Actual Date

**Given** Open Project and Executable Task with complete Actual Date
**Then** Reopen Task action is available.

### AC-2 — Invalid targets are rejected

**Given** unfinished Task, Group, Locked Project, or Closed Project
**Then** Reopen is unavailable/rejected
**And** no state changes.

### AC-3 — Confirmation is required

**When** user selects Reopen
**Then** confirmation identifies the Task and explains Actual Date removal
**And** Cancel sends no command and restores focus.

### AC-4 — Both Actual fields clear atomically

**When** Reopen succeeds
**Then** Actual Start and Actual End are both null
**And** Task becomes unfinished
**And** partial Actual Date is never persisted.

### AC-5 — Identity and unrelated fields are preserved

**Then** Task ID, hierarchy, planning data, and dependencies remain unchanged.

### AC-6 — Generic update cannot bypass Reopen

**When** generic edit submits null Actual field
**Then** request is rejected
**And** dedicated Reopen remains the only allowed command.

### AC-7 — Actual Allocation is removed/reconciled

**When** Reopen succeeds
**Then** prior Actual Allocation no longer consumes capacity
**And** unfinished scheduling recalculates according to US-6.2/US-6.1.

### AC-8 — Open timeline impact requires confirmation

**Given** Reopen causes another Open Project Executable Task Execution/Commitment Start or End to change
**Then** grouped warning lists that timeline-impacted Project name
**And** allocation/readiness-only recalculation Projects are not listed
**And** confirmed operation persists the complete recalculation scope atomically.

### AC-9 — Locked timeline impact blocks Reopen

**Given** Reopen would counterfactually require another Locked Project protected Executable Task Execution/Commitment Start or End to change
**Then** operation is blocked
**And** Actual Date and allocation remain confirmed.

### AC-10 — Dependency graph is preserved

**Then** incoming/outgoing endpoint pairs and ownership remain unchanged.

### AC-11 — Failure rolls back

**Given** scheduler, persistence, or concurrency failure
**Then** Actual Date, Actual Allocation, completion state, and projections remain unchanged.

### AC-12 — Duplicate/concurrent request is safe

**Then** at most one transition succeeds and no partial state exists.

### AC-13 — Stale response protection

**Then** older Task/tree/allocation responses cannot overwrite newer confirmed state.

### AC-14 — Accessibility

**Then** action, confirmation, impact warning, pending/error state, and focus
management are keyboard and assistive-technology accessible.

### AC-15 — Completed Task remains editable from Home

**Given** a completed Task in an Open Project is visible on Home
**When** Engineering Lead activates Task Name
**Then** the shared Edit Task dialog opens directly over Home
**And** Reopen Task is available inside the dialog
**And** Add Child and Delete are absent from the row.

### AC-16 — Locked Task edit does not imply Reopen

**Given** a completed or unfinished Task belongs to a Locked Project
**When** Engineering Lead activates Task Name on Home
**Then** Edit Task opens for the factual Actual Date capability allowed by
US-6.2
**And** Reopen Task is unavailable
**And** no Projects/Project Structure background route is opened.

---

## 8. Test Cases

| ID | Scenario | Expected |
| --- | --- | --- |
| TC-1 | Reopen Open completed Task | Both Actual fields clear; Task unfinished |
| TC-2 | Reopen partial/unfinished Task | Rejected |
| TC-3 | Reopen Group | Rejected |
| TC-4 | Reopen Locked Task | `PROJECT_LOCKED_READ_ONLY` |
| TC-5 | Reopen Closed Task | `PROJECT_CLOSED_READ_ONLY` |
| TC-6 | Cancel confirmation | No request; focus restored |
| TC-7 | Populated Task Reopen | ID, hierarchy, fields, dependencies preserved |
| TC-8 | Generic clear Actual Start/End | Rejected |
| TC-9 | Reopen removes Actual Allocation | Capacity released and impacted scope recalculated |
| TC-10 | Reopen impacts Open B/C | Warning names B/C; confirm required |
| TC-11 | Reopen impacts Locked B | Blocked; no partial save |
| TC-12 | Impact changes before confirm | Stale confirmation rejected |
| TC-13 | Scheduler/persistence failure | Full rollback |
| TC-14 | Double activation | One command |
| TC-15 | Concurrent Reopen | At most one success |
| TC-16 | Old response after success | Cannot restore completed state |
| TC-17 | Open completed Task from Home | Edit Task over Home; Reopen inside; no Add Child/Delete |
| TC-18 | Open Locked Task from Home | Edit Task factual fields only; Reopen unavailable |

---

## 9. Required Automated Tests

- Domain: complete pair and eligibility.
- Application: impact preview/confirm, Locked block, atomic coordination.
- Repository: both fields, allocation removal, identity/dependency preservation,
  rollback, concurrency.
- API: dedicated command and structured errors.
- Frontend/acceptance: Home Task Name opens shared Edit Task directly, Reopen
  remains inside completed Task dialog, Locked Task retains Actual Date without
  Reopen, confirmation, grouped impact warning, success/failure/stale state,
  keyboard/focus behavior, and no Project Structure routing.

Every affected AC requires Code Inspection, Unit/Integration, and
Acceptance-Level evidence.

---

## 10. Locked Product Decisions

- Completion uses complete Actual Date, not Actual End alone.
- Reopen clears Actual Start and Actual End together.
- Reopen is available only on Open Project.
- Reopen is scheduling-impacting because Actual Allocation is removed.
- Open cross-project confirmation is limited to Executable Task Execution/Commitment date changes.
- Locked Project blocking is limited to counterfactual protected Task timeline date changes; allocation/readiness-only pressure does not block Reopen.
- Confirmation is server-revalidated and atomic.
- Task identity, planning data, and dependency are preserved.
- Original Effort is used when Task returns to unfinished scheduling.
- Forecast remains deferred.
- Completed Task remains an Edit Task workflow on Home; Reopen stays inside the
  dialog.
- Locked Task Edit is retained for Actual Date but does not expose Reopen.

## 11. Unresolved Questions

None.
