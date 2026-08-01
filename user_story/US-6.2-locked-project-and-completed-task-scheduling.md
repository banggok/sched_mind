# US-6.2 — Actual Date, Locked Project, and Cross-Project Scheduling Impact

> **Authority and supersession:** This story is the authoritative requirement
> for Actual Date, completed-Task timeline and capacity treatment, Locked
> Project mutability, cross-project impact confirmation, Project Reopen,
> transitive recalculation scope, and Project Priority/capacity changes while
> Locked Projects exist. It supersedes contradictory wording in US-1.2,
> US-2.1, US-2.2, US-3.1, US-3.3, US-4.1, US-4.2, US-5.1, and US-6.1.

## 1. User Story

**Sebagai** Engineering Lead,
**Saya ingin** Actual Date dan perubahan scheduling lintas Project diproses secara konsisten,
**Sehingga** fakta eksekusi dapat dicatat, capacity tetap dapat diverifikasi, Locked Project tetap terlindungi, dan hanya Project yang benar-benar terdampak yang dihitung ulang.

Story ini merupakan bagian dari **Epic 6: Scheduling Engine** dan melengkapi
Project lifecycle, WBS, Dependency, Member Capacity, Project Settings, dan
Automatic Scheduling contracts.

---

## 2. Business Context

Actual completion tidak lagi direpresentasikan hanya oleh `Actual End`. Task yang
selesai memiliki **Actual Date**, yaitu pasangan `Actual Start` dan `Actual End`
yang wajib diisi bersamaan. Actual Date adalah fakta historis. Planning
Dependency tetap mengatur Execution dan Commitment, tetapi tidak mengubah atau
menolak overlap yang benar-benar terjadi pada Actual Date setelah seluruh
predecessor sudah completed.

Pada Project Open, Actual Date mengaktualkan Execution dan Commitment timeline.
Pada Project Locked, Actual Date tetap dapat dicatat tetapi protected Execution
dan Commitment baseline tidak berubah. Dalam kedua status tersebut, Actual
Allocation tetap dihitung dan memengaruhi capacity yang tersedia bagi unfinished
work pada Open Projects.

SchedMind mendukung shared assignee capacity dan cross-project dependency.
Karena itu setiap scheduling-impacting mutation harus melakukan impact
simulation. Open Project lain yang terdampak memerlukan confirmation sebelum
save. Locked Project yang terdampak biasanya memblokir save. Actual Date adalah
pengecualian karena fakta eksekusi harus tetap dapat dicatat; Locked Project
lain tidak berubah dan overlap historis tetap valid.

Project Reopen mempunyai risiko mutual lock. Jika Reopen A membutuhkan B yang
masih Locked, dan B juga membutuhkan A, sistem harus menawarkan atomic bulk
reopen terhadap seluruh transitive Locked closure. User tidak diwajibkan
membuka Project satu per satu dalam urutan yang mustahil.

---

## 3. Terminology

| Term | Meaning |
| --- | --- |
| Actual Date | Required inclusive range `Actual Start`–`Actual End`, entered together after Task selesai |
| Completed Task | Executable WBS dengan complete Actual Date pair |
| Unfinished Task | Executable WBS tanpa complete Actual Date pair |
| Actual Allocation | Historical/analytical attribution of Task Effort to assignee working dates; bukan literal timestamp setiap jam kerja |
| Execution Allocation | Daily allocation used to derive Execution Start/End |
| Commitment Allocation | Daily allocation used to derive Commitment Start/End |
| Allocation baseline | Execution Start used as the planned baseline for Actual Allocation |
| Capacity-debt window | Working dates from Allocation Start before Actual Start that may receive only remaining positive capacity |
| Historical overcapacity | Actual Allocation pada suatu Date melebihi effective assignee capacity pada Date tersebut |
| Locked baseline | Persisted Execution/Commitment dates and allocations protected while Project Locked |
| Scheduling-impacting mutation | Mutation that can change dates, allocations, dependency readiness, unscheduled state, priority order, or capacity available to another Task/Project |
| Impacted Project | Project other than the mutation owner whose confirmed scheduling projection would change |
| Transitive impacted scope | Impacted set expanded A→B→C until no further Project changes |
| Required Locked Reopen Closure | Every Locked Project that must become Open together so a requested Reopen can be calculated without mutating any remaining Locked Project |

---

## 4. Scope

### 4.1 In Scope

- Replace `Actual End` input with required Actual Date range.
- Actual Date validation and completed lifecycle.
- Execution/Commitment actualization for Open Projects.
- Locked Project Actual Date exception without baseline mutation.
- Actual Allocation algorithm, working-date rules, overcapacity, and persistence/projection requirements.
- Task-centric allocation verification UI.
- Data contract capable of supporting future assignee-centric allocation analytics.
- Dependency validation when Actual Date is entered.
- Recalculation of unfinished work in transitive impacted scope.
- Generic cross-project impact preview, warning, confirmation, stale-preview protection, and atomic save.
- Impact caused by Task, dependency, priority, Member capacity/buffer, Capacity Override, Public Holiday, Project Buffer, Project Reopen, and future scheduling-impacting settings.
- Locked Project blocking rules and Actual Date exception.
- Atomic bulk reopen for mutually/transitively related Locked Projects.
- Lock eligibility, rollback, concurrency, structured errors, accessibility, and Three-Level Confidence evidence.

### 4.2 Out of Scope

- Forecast calculation and Forecast allocation.
- Delivery Impact and Project Health changes.
- Remaining-effort estimation or in-progress Actual Start-only lifecycle.
- Overtime compensation or recovery-capacity debt.
- User-editable Actual Allocation rows.
- Assignee-centric analytics page design; only the underlying consistent allocation projection is required now.
- Reason/detail per impacted Project in warning UI; warning lists Project names only.
- Automatic repair of corrupted persisted data.

---

## 5. Actual Date Contract

### 5.1 Required Pair

Actual Date consists of:

```text
Actual Start
Actual End
```

Rules:

- Both values are required in one completion command.
- Actual Start-only and Actual End-only states are rejected.
- Actual Date is entered only after work is completed; no in-progress state is introduced.
- `Actual Start <= Actual End` is mandatory.
- Both dates may be working or non-working calendar dates.
- A successful command changes the Task from unfinished to completed.
- Existing `Actual End` terminology in older stories means Actual Date completion unless explicitly discussing the end field.

### 5.2 Reopen Completed Task

Reopen removes the complete Actual Date pair atomically:

```text
Actual Start = null
Actual End   = null
```

Partial removal is invalid. Reopen is available only while the owning Project
is Open and follows US-4.2 plus the impact rules in this story.

### 5.3 Dependency Validation at Completion

When Actual Date is entered for successor Task B:

- every effective predecessor of B must already be completed with a complete Actual Date pair;
- if at least one predecessor is unfinished, completion of B is rejected;
- once all predecessors are completed, Actual Date B may overlap predecessor Actual Dates;
- Actual Start B may be before, equal to, or after predecessor Actual End;
- same-assignee and different-assignee planned start rules do not constrain historical Actual Date;
- existing dependency endpoints and ownership are not mutated by completion.

Dependency continues to govern Execution and Commitment scheduling. Actual Date
overlap is historical fact, not a scheduling integrity conflict.

---

## 6. Open Project Completion

### 6.1 Timeline Actualization

When Actual Date is saved on an unfinished Task in an Open Project:

```text
Execution Start = Actual Start
  when existing Execution Start is null
  or Actual Start < existing Execution Start
otherwise existing Execution Start

Execution End = Actual End

Commitment Start = Actual Start
  when existing Commitment Start is null
  or Actual Start < existing Commitment Start
otherwise existing Commitment Start

Commitment End = Actual End
```

Equivalent formulas:

```text
Execution Start  = min(existing Execution Start, Actual Start)
Commitment Start = min(existing Commitment Start, Actual Start)
Execution End    = Actual End
Commitment End   = Actual End
```

A null planned Start is treated as later than Actual Start.

Rules:

- Execution and Commitment are actualized independently.
- Actual Start never moves an existing Start later.
- Actual End is always used as completed dependency readiness anchor.
- `Actual End < former Start` is normalized by the formulas above and must not produce `end before start`.
- The completed Task is not moved again by unfinished scheduling.
- Scheduler recalculates unfinished Tasks in the transitive impacted scope.

### 6.2 Planned Dependency Start Rules Remain

Before completion, normal Execution/Commitment scheduling remains authoritative:

- same-assignee predecessor/successor with zero Lag may share predecessor End Date when positive remaining capacity exists;
- different-assignee successor starts no earlier than the next working date after predecessor readiness;
- these rules determine planned Execution/Commitment Start and therefore may determine whether a capacity-debt window exists;
- these rules do not constrain Actual Date overlap after all predecessors are completed.

---

## 7. Actual Allocation

### 7.1 Single Source, Two Read Models

Execution Allocation, Commitment Allocation, and Actual Allocation are distinct
daily projections. Actual Allocation is stored or deterministically
reconstructable as one canonical set of rows.

The same Actual Allocation rows must support:

1. **Task-centric view:** for Task X, which assignee Date received how many hours.
2. **Assignee-centric view:** for assignee Y on Date D, which Tasks consumed capacity.

No separate calculation is allowed for the two points of view.

For a completed Task, Actual Allocation is the sole capacity-consuming
reservation used by subsequent Execution and Commitment recalculation.
Execution and Commitment allocation rows remain available as planning/baseline
projections for verification, but they are not subtracted again from capacity.
This prevents double counting.

### 7.2 Allocation Start

Actual Allocation uses Execution Start as its planning baseline because
Commitment Start cannot be earlier than Execution Start and Commitment includes
delivery buffer.

```text
Allocation Start = min(existing/persisted Execution Start, Actual Start)
```

If Execution Start is null:

```text
Allocation Start = Actual Start
```

For Open Project, Execution Start is actualized according to Section 6 but the
pre-completion Execution Start remains the baseline input required to construct
Actual Allocation deterministically. For Locked Project, protected Execution
Start remains unchanged and is used directly as the baseline.

### 7.3 Working Dates

- Actual Date may include non-working dates.
- Allocation rows may exist only on working dates according to the effective assignee calendar.
- Weekend and Public Holiday remain non-working regardless of Capacity Override.
- Actual Allocation uses **BAU Capacity**, defined as Resolved Daily Capacity before Member Buffer and Project Buffer:

```text
Actual BAU Capacity
= 0 on weekend/Public Holiday
= Capacity Override when available
= otherwise Member Daily Capacity
```

- Member Buffer affects Execution Capacity only.
- Member Buffer plus Project Buffer affect Commitment Capacity only.
- Neither buffer reduces or increases Actual BAU Capacity.
- Non-working Actual Start or Actual End remains stored as factual Actual Date.

### 7.4 Capacity-Debt Window

For working dates from Allocation Start up to but excluding Actual Start:

- allocate earliest-first;
- use only positive remaining capacity after immutable reservations: other completed Actual Allocations and Locked baseline reservations;
- Open unfinished allocations are movable simulation outputs and do not prevent Actual Allocation from becoming the fixed anchor; they are recalculated afterward;
- do not create overcapacity in the capacity-debt window against immutable reservations;
- stop when remaining Effort becomes zero or Actual Start is reached.

This allows an earlier planned Execution Start to retain its unused capacity
contribution even when actual work started later.

Example:

```text
Task A blocks Task B
Task A and B assignee: Harry
Daily capacity: 8h
Task A: 4h on Date 1
Task B Effort: 18h
Task B Execution Start: Date 1
Task B Actual Date: Date 2–3
```

Actual Allocation B:

```text
Date 1: 4h  (remaining capacity after Task A)
Date 2: 8h
Date 3: 6h
```

### 7.5 Allocation Within Actual Date

After the capacity-debt window, allocate remaining Effort over working dates
inside Actual Start–Actual End:

1. Resolve remaining capacity on every working date in the Actual Date.
2. Fill available capacity earliest-first while Effort remains.
3. If Effort exceeds the total available capacity in the Actual Date, calculate the excess.
4. Distribute excess evenly across all working dates in the Actual Date.
5. Perform arithmetic in integer minutes.
6. If even distribution leaves a minute remainder, assign one additional minute from the earliest working date forward until exhausted.
7. Resulting allocation may exceed daily capacity and becomes historical overcapacity.

Example with a different predecessor assignee:

```text
Task A assignee: Tirta
Task B assignee: Harry
Task B Execution Start: Date 2
Task B Actual Date: Date 2–3
Task B Effort: 18h
Harry capacity: 8h/day
```

Actual Allocation B:

```text
Date 2: 9h
Date 3: 9h
```

### 7.6 No Working Date in Actual Date

If Actual Start–Actual End contains no working date:

- first retain any valid capacity-debt-window allocation before Actual Start;
- allocate all remaining Effort to the first working date after Actual End;
- that allocation may exceed effective capacity;
- Actual End remains the completion and dependency-ready date even though analytical allocation occurs later;
- the later allocation does not mean the Task was unfinished after Actual End.

### 7.7 Capacity Effect and No Carry-Over

For unfinished scheduling on Date D:

```text
Remaining Capacity(D)
  = max(0, Effective Capacity(D) - Actual Allocation(D))
```

Rules:

- historical overcapacity is valid;
- overcapacity does not become negative-capacity debt on later dates;
- no recovery capacity or overtime compensation exists;
- the next working date uses its own normal effective capacity minus allocations on that date;
- completed Actual Allocation must not produce `SCHEDULING_DATA_INTEGRITY_CONFLICT` merely because it exceeds capacity.

### 7.8 Allocation Verification UI

View/Edit Task provides a read-only expandable **Capacity Allocation** section
with three groups:

- Execution Allocation;
- Commitment Allocation;
- Actual Allocation.

Each row shows at minimum:

- Date;
- allocated hours;
- effective BAU capacity;
- remaining BAU capacity or overcapacity for Actual rows; Execution/Commitment rows use their own timeline capacity.

Actual Allocation is displayed only when Actual Date is complete. The UI must
not imply that analytical allocation dates are literal timestamps of work.

The assignee-centric analytics page is deferred, but the data/projection must be
queryable without recomputing different numbers.

---

## 8. Locked Project Rules

### 8.1 Lock Eligibility

`Open → Locked` is allowed only when:

- Project has at least one Executable Task;
- every unfinished Task has complete Execution Start/End;
- every unfinished Task has complete Commitment Start/End;
- no unfinished Task has Execution Unscheduled Reason;
- no unfinished Task has Commitment Unscheduled Reason.

Lock validates current confirmed state and does not run scheduler to repair an
unscheduled Task. Failure leaves Project Open and unchanged.

### 8.2 Planning Immutability

While Project Locked, the following are prohibited:

- create, rename, edit, delete, move, or reorder Task/WBS;
- add child or Executable-to-Group conversion;
- change Role, Assignee, Effort, Lag, Name, or manual/generated timeline;
- create, edit, delete, retarget, or change dependency ownership;
- Reopen completed Task;
- Project Settings changes that alter scheduling behavior.

Frontend disables/hides planning actions with a clear explanation. Backend
rejects direct API bypass. There is no implicit Reopen.

### 8.3 Actual Date Exception

Actual Date remains writable on an unfinished Task in a Locked Project.

When saved:

- complete Actual Date is persisted;
- Task becomes completed;
- locked Execution/Commitment dates and allocations remain unchanged;
- the Locked Project itself is not rescheduled;
- Actual Allocation is calculated using the protected Execution Start baseline and Sections 7.3–7.7;
- new Actual Allocation changes capacity available to Open Projects;
- scheduler recalculates transitive impacted Open Projects;
- Locked Project dependencies and WBS order remain unchanged;
- Forecast effect remains deferred.

Therefore, “Locked Project does not run scheduler” means the scheduler may not
mutate that Locked Project. It does not prevent Actual Date from triggering
recalculation of impacted Open Projects.

### 8.4 Actual Date Impact on Other Locked Projects

Actual Date is factual data and is an exception to ordinary Locked-impact
blocking:

- Actual Date remains saveable even when its Actual Allocation or readiness affects another Locked Project;
- when any other Project is impacted, UI shows the same grouped Locked/Open Project-name warning and requires confirmation;
- Locked impact is informational for Actual Date and does not block confirmation;
- server revalidates impact on Confirm;
- no Locked Project timeline, allocation baseline, dependency, or status is mutated;
- overlap between Actual Allocation and locked allocations is accepted as historical overcapacity;
- overlap between actual dependent Task ranges is accepted after predecessor-completion validation;
- impacted Open Projects are recalculated atomically with Actual Date save.

### 8.5 Close

Locked Project may be Closed after every Executable Task has a complete Actual
Date pair. Close does not rewrite the locked baseline.

---

## 9. Generic Cross-Project Impact Guard

### 9.1 Covered Mutations

The same impact contract applies to every confirmed mutation that can change
scheduling state or available capacity, including:

- create/edit/delete/move/reorder scheduling-relevant Task/WBS;
- complete or Reopen Task;
- dependency create/edit/delete/retarget/ownership reconciliation;
- Project Priority;
- Member Daily Capacity;
- Member Buffer;
- Capacity Override;
- Public Holiday;
- Project Buffer;
- Project Scheduling settings;
- Project Reopen;
- future scheduling-impacting capacity or planning settings.

A mutation that has no scheduling impact follows its owning story and need not
show a cross-project warning.

### 9.2 Impact Definition

Another Project is impacted when server-side simulation predicts a change to at
least one confirmed scheduling projection, including:

- Execution dates or allocation;
- Commitment dates or allocation;
- Actual Allocation rows or Actual-capacity availability consumed by unfinished work;
- scheduled/unscheduled state or unscheduled reason;
- dependency readiness;
- aggregate Project dates derived from affected Tasks.

The current Project being edited is excluded from the warning list.

### 9.3 Transitive Scope

Impact is expanded transitively:

```text
A changes B
B changes C
=> impacted Projects: B and C
```

Independent Project D is not recalculated, version-updated, or allowed to block
the mutation because of unrelated state.

### 9.4 Warning Behaviour

#### No Other Project Impact

Save proceeds without cross-project warning.

#### Open Projects Only

Before save, UI shows a confirmation warning listing impacted Project names:

```text
Open Projects
- Project B
- Project C
```

No per-Project reason is required. User may Confirm or Cancel.

#### Locked and Open Projects

For ordinary planning/capacity mutation, UI groups names:

```text
Locked Projects
- Project B
- Project C

Open Projects
- Project D
```

If at least one Locked Project is impacted:

- save is blocked;
- no partial mutation or Open-Project recalculation is persisted;
- user must Reopen the required Locked Projects first.

Actual Date follows the exception in Section 8.4 and remains saveable.

### 9.5 Server Revalidation and Atomic Save

Impact preview is advisory and version-bound. On Confirm, server must:

1. reacquire required transaction/scheduling locks;
2. recalculate the proposed mutation and transitive impacted set;
3. validate current Project statuses and schedule versions;
4. compare with the preview token/version;
5. persist the mutation and all allowed Open-Project recalculation atomically.

If the impacted set or relevant version changed:

- do not save;
- return a stale-impact response with the new grouped Project names;
- UI replaces the old warning and requires a new confirmation.

No stale confirmation may mutate a newly Locked Project.

---

## 10. Project Reopen and Mutual Locked Impact

### 10.1 Explicit Reopen

Planning changes to a Locked Project require explicit:

```text
Locked → Open
```

Task-level Reopen is not a substitute for Project Reopen.

### 10.2 Required Locked Reopen Closure

When Reopen Project A is requested, server simulates the recalculation needed
once A becomes Open.

If remaining Locked Project B would need to change, B is added to the Required
Locked Reopen Closure. Simulation repeats as though A and B are Open. If that
requires Locked C to change, C is added. Expansion continues until fixed point.

This resolves mutual lock:

```text
A requires B
B requires A
=> required closure: A and B
```

and transitive lock:

```text
A requires B
B requires C
=> required closure: A, B, and C
```

### 10.3 Reopen Warning

UI shows names only:

```text
Locked Projects that must be reopened together
- Project A
- Project B

Open Projects that will be recalculated
- Project C
- Project D
```

Actions:

```text
Reopen All
Cancel
```

User cannot remove a required Locked Project or request partial Reopen.

### 10.4 Atomic Bulk Reopen

On **Reopen All**, server revalidates the closure and versions. Then atomically:

- changes every Project in the required Locked closure to Open;
- keeps completed Tasks as historical Actual Allocation anchors;
- recalculates unfinished Tasks across the transitive impacted Open scope;
- reconciles dependencies and allocations;
- persists status, schedule, allocation, and version changes together.

Valid unscheduled results do not fail Reopen. Reopened Projects remain Open and
cannot Lock again until lock eligibility is restored.

If corruption, concurrency conflict, persistence failure, or internal scheduler
defect occurs, the whole operation rolls back and all Projects retain their
previous statuses and projections.

---

## 11. Project Priority and Capacity Changes with Locked Projects

Project Priority, Member Daily Capacity, Member Buffer, Public Holiday,
Project Buffer, and other capacity-setting changes use the generic impact guard
in Section 9. Capacity Override create/update/delete uses that guard only when
before/after Resolved Daily Capacity changes on at least one Date; a record-only
change that leaves the per-Date minimum unchanged is persisted without warning,
scheduler invocation, or schedule-version change.

- Locked Projects are immutable anchors during simulation.
- Open Projects may be recalculated.
- A change succeeds only when no Locked Project would require mutation.
- If a Locked Project is impacted, the ordinary change is blocked atomically.
- The response/UI lists names grouped by Locked/Open without detailed reasons.
- Project Priority may include changing the Priority of a Locked Project only if simulation proves all Locked Projects remain unchanged and dependency-valid.

Actual Date remains the factual-data exception defined in Section 8.4.

---

## 12. Scheduler Failure Contract

For valid domain state, inability to place unfinished work is a valid
`Unscheduled + Reason` result, not scheduler failure.

Scheduler failure is limited to:

- corrupted persisted state violating an invariant;
- concurrency/version conflict;
- database, transaction, or persistence failure;
- internal scheduler defect.

Historical overcapacity, Actual Date overlap, missing Assignee/Effort, no
positive capacity, unmet readiness, or no schedulable date are not data-integrity
failures when represented according to owning requirements.

Corrupted data is repaired outside normal scheduling flow. Scheduler must not
silently invent or repair business data.

---

## 13. API and Error Contract

Existing owning endpoints remain authoritative. Minimum stable concepts:

| Error Code | HTTP | Condition |
| --- | ---: | --- |
| `ACTUAL_DATE_INCOMPLETE` | 400/422 | Only Actual Start or Actual End supplied |
| `ACTUAL_DATE_INVALID_RANGE` | 400/422 | Actual Start is after Actual End |
| `ACTUAL_DATE_PREDECESSOR_UNFINISHED` | 409 | At least one effective predecessor lacks a complete Actual Date |
| `PROJECT_CANNOT_LOCK_WITH_UNSCHEDULED_TASKS` | 409 | At least one unfinished Task is not fully scheduled |
| `PROJECT_LOCKED_READ_ONLY` | 409 | Prohibited planning mutation requested on Locked Project |
| `SCHEDULING_IMPACT_CONFIRMATION_REQUIRED` | 409 | Mutation affects Open Projects and requires user confirmation |
| `SCHEDULING_LOCKED_PROJECT_IMPACT` | 409 | Ordinary mutation affects at least one Locked Project and is blocked |
| `SCHEDULING_IMPACT_STALE` | 409 | Impact set/version changed after preview |
| `PROJECT_BULK_REOPEN_REQUIRED` | 409 | Reopen requires a transitive Locked Project closure |
| `PROJECT_REOPEN_SCHEDULING_FAILED` | 409/500 according to established mapping | Atomic Reopen failed and rolled back |

Impact responses expose only safe Project identity/name grouped by lifecycle
status plus a version/confirmation token. They do not expose SQL, stack traces,
internal lock IDs, or per-Project scheduler internals.

---

## 14. Acceptance Criteria

### AC-1 — Actual Date requires a complete valid pair

**When** completion is submitted
**Then** Actual Start and Actual End must both be present
**And** Actual Start must not be after Actual End
**And** partial/invalid input is rejected without mutation.

### AC-2 — Actual Date may use non-working dates

**Given** Actual Start or Actual End is weekend/holiday
**When** complete Actual Date is saved
**Then** factual dates are retained
**And** allocation rows are created only on eligible working dates.

### AC-3 — Open completion actualizes Execution and Commitment

**Given** unfinished Task on Open Project
**When** Actual Date is saved
**Then** each Start becomes the earlier of existing Start and Actual Start
**And** each End becomes Actual End.

### AC-4 — Actual End is readiness anchor

**When** completed Task blocks unfinished work
**Then** scheduler uses Actual End as readiness anchor
**And** former planned End does not replace Actual End.

### AC-5 — Completion requires completed predecessors

**Given** Task B has at least one unfinished effective predecessor
**When** Actual Date B is submitted
**Then** completion is rejected with `ACTUAL_DATE_PREDECESSOR_UNFINISHED`.

### AC-6 — Completed actual ranges may overlap

**Given** all predecessors are completed
**When** Task B Actual Date overlaps predecessor Actual Date
**Then** completion succeeds regardless of same/different assignee
**And** dependency relation remains unchanged.

### AC-7 — Actual Allocation uses Execution baseline

**Then** Allocation Start is the earlier of pre-completion/persisted Execution Start and Actual Start
**And** null Execution Start falls back to Actual Start
**And** Commitment Start is not used as Actual Allocation baseline.

### AC-8 — Capacity-debt window uses remaining capacity only

**Given** Allocation Start is before Actual Start
**Then** working dates before Actual Start receive earliest positive remaining capacity
**And** no overcapacity is created in that pre-Actual window.

### AC-9 — Actual-Date capacity fills earliest-first

**Given** remaining Effort fits within available capacity of working dates in Actual Date
**Then** capacity is consumed earliest-first
**And** later working dates receive only the remainder.

### AC-10 — Excess is distributed evenly

**Given** remaining Effort exceeds total available capacity in Actual Date
**Then** excess is distributed evenly across all working dates in Actual Date
**And** minute remainder is assigned from earliest date forward
**And** historical overcapacity is valid.

### AC-11 — No working date uses next working date

**Given** Actual Date contains no working date
**Then** remaining Effort is allocated to the first working date after Actual End
**And** Actual End remains the completion/readiness date.

### AC-12 — No overcapacity carry-over

**Given** Actual Allocation exceeds capacity on Date D
**Then** remaining capacity on D is minimum zero
**And** excess is not carried as debt to later dates.

### AC-13 — Locked completion preserves baseline

**Given** unfinished Task on Locked Project
**When** Actual Date is saved
**Then** protected Execution/Commitment dates and allocations remain unchanged
**And** Actual Allocation is persisted
**And** owning Locked Project is not rescheduled.

### AC-14 — Locked completion recalculates impacted Open Projects

**Given** Actual Allocation on Locked Project changes shared capacity/readiness
**When** Actual Date is saved
**Then** transitively impacted Open Projects are recalculated
**And** independent Projects are untouched.

### AC-15 — Actual Date is not blocked by Locked impact

**Given** completion affects another Locked Project
**When** grouped impact warning is confirmed and server revalidation succeeds
**Then** factual completion succeeds
**And** every Locked Project remains unchanged
**And** impacted Open Projects are recalculated
**And** overlap is represented as historical state rather than integrity failure.

### AC-16 — Locked planning mutation remains rejected

**Given** Project Locked
**When** non-Actual planning, WBS, dependency, Settings, or Task Reopen mutation is requested
**Then** request is rejected with `PROJECT_LOCKED_READ_ONLY`.

### AC-17 — Lock requires every unfinished Task scheduled

**Given** an unfinished Task has incomplete dates or unscheduled reason
**When** Lock is requested
**Then** Lock is rejected and Project remains Open.

### AC-18 — Task-centric allocation verification

**When** Task details are viewed
**Then** Execution, Commitment, and Actual daily allocation groups are available read-only
**And** each row exposes Date, allocated amount, capacity, and remaining/overcapacity.

### AC-19 — Canonical allocation supports assignee POV

**Then** the same persisted/reconstructable Actual Allocation rows can answer Task-centric and assignee-centric queries
**And** no separate arithmetic produces divergent totals.

### AC-20 — Open-only cross-project impact requires confirmation

**Given** mutation affects one or more other Open Projects and no Locked Project
**Then** warning lists impacted Project names
**And** save occurs only after confirmation and server revalidation.

### AC-21 — Ordinary Locked impact blocks save

**Given** non-Actual mutation affects a Locked Project
**Then** warning groups Locked/Open Project names
**And** complete operation is blocked with no partial save.

### AC-22 — Impact is transitively bounded

**Given** A changes B and B changes C
**Then** B and C appear as impacted
**And** independent D is not recalculated or version-updated.

### AC-23 — Stale impact confirmation cannot save

**Given** impact/status/version changes after preview
**When** user confirms old warning
**Then** save is rejected with `SCHEDULING_IMPACT_STALE`
**And** UI receives the current grouped Project list.

### AC-24 — Capacity sources use the same guard

**When** Daily Capacity, Member Buffer, Public Holiday, or Project Buffer changes
**Then** the same transitive impact warning/blocking/revalidation contract applies.

**When** Capacity Override create/update/delete changes the resolved per-Date
minimum for at least one Date
**Then** the same guard applies.

**When** only the override record changes and the resolved per-Date minimum does
not change
**Then** the mutation persists without impact warning, scheduler invocation, or
schedule-version change.

### AC-25 — Mutual Locked Reopen produces closure

**Given** Reopen A requires B and Reopen B requires A
**When** Reopen A is requested
**Then** required Locked closure contains A and B
**And** user is offered Reopen All rather than an impossible single-Project sequence.

### AC-26 — Reopen closure expands transitively

**Given** A requires B and B requires Locked C
**Then** required closure contains A, B, and C.

### AC-27 — Bulk Reopen is atomic

**When** Reopen All is confirmed
**Then** every required Locked Project becomes Open together
**And** impacted Open scope is recalculated
**And** partial Reopen is impossible.

### AC-28 — Bulk Reopen failure rolls back

**Given** corruption, concurrency, persistence, or internal scheduler failure
**When** bulk Reopen fails
**Then** all Project statuses, dates, allocations, dependencies, and versions remain at confirmed pre-operation state.

### AC-29 — Unscheduled output does not fail Reopen

**Given** valid recalculation returns Unscheduled + Reason
**Then** Reopen succeeds
**And** Project remains Open but cannot Lock until eligibility is restored.

### AC-30 — Accessibility and recoverability

**Then** allocation sections, impact warning, grouped Project lists, Reopen All confirmation, pending state, stale-impact feedback, and retry/reload actions are keyboard accessible
**And** status is not communicated by color alone
**And** recoverable errors do not erase unrelated draft or confirmed state.

---

## 15. Test Cases

| ID | Scenario | Expected |
| --- | --- | --- |
| TC-1 | Submit Actual Start only | Rejected; no partial completion |
| TC-2 | Actual Start after Actual End | Rejected |
| TC-3 | Open Task Actual Start earlier than both planned Starts | Both Starts move earlier; Ends become Actual End |
| TC-4 | Open Task Actual Start later than planned Starts | Planned Starts retained; Ends become Actual End |
| TC-5 | Same-assignee example: A 4h Date 1, B 18h, Actual 2–3 | Actual Allocation B = 4h, 8h, 6h |
| TC-6 | Different-assignee example: B 18h, Execution/Actual 2–3 | Actual Allocation B = 9h, 9h |
| TC-7 | Actual Date includes weekend but has one working date | Allocation uses working date only |
| TC-8 | Actual Date contains no working date | Remaining Effort allocated on next working date |
| TC-9 | Actual excess uneven in minutes | Excess distributed evenly; remainder earliest-first |
| TC-10 | Historical overcapacity followed by next working day | No debt carry-over |
| TC-10A | Member/Project Buffer differs from Daily Capacity | Actual uses BAU capacity; Execution/Commitment use buffered capacities |
| TC-11 | Complete successor while predecessor unfinished | Rejected |
| TC-12 | Completed predecessor and overlapping successor Actual Date | Accepted |
| TC-13 | Locked Task receives Actual Date | Baseline unchanged; Actual Allocation stored |
| TC-14 | Locked Actual Date impacts Open B and independent C | B recalculated; C untouched |
| TC-15 | Locked Actual Date overlaps another Locked allocation | Grouped warning/confirm; save succeeds; both locked baselines unchanged |
| TC-16 | View completed Task allocation | Execution/Commitment/Actual groups show daily rows and totals |
| TC-17 | Query Task POV and assignee POV | Same allocation row totals |
| TC-18 | Edit Task impacts Open B/C | Warning names B/C; confirm required |
| TC-19 | Edit Task impacts Locked B and Open C | Blocked; grouped names displayed |
| TC-20 | Actual Date impacts Locked B | Grouped warning/confirm; factual save succeeds; B unchanged |
| TC-21 | Member Daily Capacity impacts Open Projects | Same warning/confirmation contract |
| TC-22 | Member Buffer impacts Locked Project | Save blocked |
| TC-23 | Capacity Override changes resolved minimum | Same generic guard |
| TC-23A | Capacity Override record changes but minimum is unchanged | Persist record; no warning/scheduler/version change |
| TC-23B | Public Holiday or Project Buffer impact | Same generic guard |
| TC-24 | Impact changes between preview and confirm | Stale confirmation rejected; updated list returned |
| TC-25 | A impacts B impacts C | B/C recalculated; independent D untouched |
| TC-26 | Mutual A/B Locked Reopen | Required closure A/B; Reopen All offered |
| TC-27 | A/B closure requires Locked C | Closure expands to A/B/C |
| TC-28 | User attempts partial closure | Not allowed |
| TC-29 | Bulk Reopen succeeds | All closure statuses Open; impacted schedules atomic |
| TC-30 | Bulk Reopen persistence failure | Everything rolls back |
| TC-31 | Bulk Reopen yields unscheduled Task | Reopen succeeds; later Lock rejected |
| TC-32 | Old response arrives after completion/reopen/impact save | Stale state cannot replace confirmed state |

---

## 16. Required Automated Tests by Layer

### 16.1 Domain Tests

- Actual Date pair/range invariants.
- Open timeline actualization.
- Allocation Start and capacity-debt window.
- Working-date-only allocation and no-working-date fallback.
- Even excess distribution with deterministic minute remainder.
- Historical overcapacity and no carry-over.
- Completed predecessor validation with allowed Actual overlap.
- Lock eligibility and Locked mutation policy.
- Impact classification and Locked exception for Actual Date.
- Required Locked Reopen closure/fixed-point expansion.

### 16.2 Application Tests

- Completion coordination for Open and Locked Projects.
- Generic impact preview and confirmation token.
- Stale preview rejection.
- Actual Date save with Locked impact exception.
- Ordinary mutation blocked by Locked impact.
- Member/Project capacity-setting impact coordination.
- Atomic bulk Reopen and valid unscheduled result.
- Failure rollback.

### 16.3 Repository Integration Tests

- Persist/reload complete Actual Date pair.
- Persist Execution/Commitment/Actual allocation rows independently.
- Task-centric and assignee-centric queries produce equal totals.
- Cross-project capacity consumption from Locked completed Task.
- Transitive impacted-scope traversal.
- Locked rows unchanged during Open recalculation.
- Bulk Reopen closure, optimistic concurrency, and rollback.

### 16.4 API Integration Tests

- Actual Date validation errors.
- Completion predecessor conflict.
- Open-only confirmation-required response.
- Locked-impact blocking response.
- Actual Date factual exception.
- Stale-impact response.
- Bulk-Reopen-required response and Reopen All command.
- Locked read-only bypass protection.

### 16.5 Frontend and Acceptance-Level Tests

- Actual Date pair input and validation.
- Read-only allocation verification section.
- Non-working factual dates with working-date allocation display.
- Grouped Open/Locked impact warning names only.
- Confirmation, stale warning refresh, cancel, and no partial save.
- Mutual Locked Reopen All workflow.
- Accessibility, focus management, pending state, retry/reload, and stale-response protection.

Every affected Acceptance Criterion requires Code Inspection, Unit/Integration,
and Acceptance-Level evidence under the repository Three-Level Confidence rule.

---

## 17. Documentation Impact

This story requires reconciliation of:

- US-1.2 Member Daily Capacity and Buffer impact behavior;
- US-2.1 Capacity Override impact behavior;
- US-2.2 Public Holiday impact behavior;
- US-3.1 Project Lock/Reopen/Priority lifecycle;
- US-3.3 Project Buffer and Settings impact behavior;
- US-4.1 Task completion, Actual Date fields, and allocation verification;
- US-4.2 clearing both Actual Start and Actual End;
- US-5.1 completed predecessor validation and Actual overlap semantics;
- US-6.1 trigger, completed allocation, transitive impact, and Locked anchors;
- architecture and product/domain context.

Forecast remains deferred and must not be inferred from Actual Allocation.

---

## 18. Locked Product Decisions

- Actual completion is a required Actual Start–Actual End pair entered together after completion.
- Actual Date may use non-working calendar dates; allocation rows use working dates only.
- Open completion moves each Start earlier only when Actual Start is earlier and always sets each End to Actual End.
- Actual End is always the completed dependency-ready anchor.
- Planned same/different-assignee dependency rules apply to Execution/Commitment, not to historical Actual overlap.
- Successor completion requires every predecessor already completed, but Actual ranges may overlap afterward.
- Actual Allocation uses Execution Start as baseline, not Commitment Start.
- Actual Allocation capacity is BAU Resolved Daily Capacity; Member and Project buffers apply only to Execution/Commitment.
- Capacity before Actual Start uses remaining positive capacity only.
- Capacity inside Actual Date fills earliest-first; excess is distributed evenly across actual working dates.
- If Actual Date has no working date, remaining Effort is allocated on the next working date.
- Historical overcapacity is valid and never carried as debt.
- One canonical allocation projection supports Task-centric and future assignee-centric views and is the sole capacity-consuming reservation for a completed Task.
- Current UI displays Execution, Commitment, and Actual allocation in Task details for verification.
- Locked Project baseline remains immutable, but Actual Date and Actual Allocation may be recorded.
- Locked completion may trigger scheduler for impacted Open Projects while never mutating Locked Projects.
- Actual Date remains saveable after grouped warning/confirmation even when another Locked Project is impacted.
- Ordinary scheduling-impacting mutation is blocked when any Locked Project is impacted.
- Open-only impact requires warning, confirmation, and server-side revalidation.
- Impact warning lists Project names only, grouped by Locked/Open, excluding the current Project.
- Impact scope is transitive and excludes independent Projects.
- Daily Capacity, Member Buffer, Public Holiday, Project Buffer, Priority, and other effective capacity changes use the same impact guard.
- Capacity Override overlap is allowed; the minimum active override is resolved per Member/Date before buffers.
- Capacity Override mutation uses the guard only when that resolved minimum changes on at least one Date.
- Mutual/transitive Locked Reopen uses an atomic required closure with Reopen All; partial Reopen is not allowed.
- Valid unscheduled output does not fail Reopen.
- Forecast remains deferred.

---

## 19. Unresolved Questions

None.
