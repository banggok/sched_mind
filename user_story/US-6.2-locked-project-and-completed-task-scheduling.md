# US-6.2 — Actual Date, Locked Project, and Cross-Project Scheduling Impact

> **Authority and supersession:** This story is the authoritative requirement
> for Actual Date, completed-Task timeline and capacity treatment, Locked
> Project mutability, cross-project impact confirmation, Project Reopen,
> transitive recalculation scope, and Project Priority/capacity changes while
> Locked Projects exist. It supersedes contradictory wording in US-1.2,
> US-2.1, US-2.2, US-3.1, US-3.3, US-4.1, US-4.2, US-5.1, and US-6.1.
> US-7.1 remains authoritative for the Home presentation surface and direct
> dialog orchestration described by this story. US-6.3 remains authoritative
> for planned Task Capacity Allocation Percentage; this story is authoritative
> for the rule that planned percentage never caps Actual Allocation.

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
Allocation dihitung hanya di dalam Actual Date. Distribusinya berusaha memakai
BAU capacity yang belum dikonsumsi oleh Actual Allocation completed Task lain,
melakukan rebalance dan backfill sebelum menciptakan historical overcapacity,
lalu memengaruhi capacity yang tersedia bagi unfinished work pada Open Projects.
Planned Execution, Commitment, manual, dan Locked baseline allocation tidak
menjadi lawan head-to-head ketika Actual Allocation baru dibentuk.

SchedMind mendukung shared assignee capacity dan cross-project dependency.
Karena itu setiap scheduling-impacting mutation harus melakukan server-side
simulation terhadap scope yang dapat mempropagasi perubahan. **Recalculation
scope** dan **warning impact** adalah dua konsep berbeda: Project lain tetap
boleh dihitung ulang untuk menjaga correctness scheduler, tetapi hanya Project
yang Execution/Commitment timeline-nya benar-benar berubah yang ditampilkan
dalam cross-project warning. Perubahan allocation, capacity availability,
dependency readiness, atau unscheduled reason tanpa perubahan Start/End tidak
cukup untuk memunculkan warning. Locked Project dengan protected timeline yang
secara counterfactual harus berubah tetap mengikuti Locked blocking rule. Actual
Date adalah pengecualian karena fakta eksekusi harus tetap dapat dicatat; Locked
Project lain tidak berubah dan overlap historis tetap valid.

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
| Actual Allocation | Historical/analytical attribution of Task Effort to eligible Actual Dates, with an Actual End fallback when the range has no working date; bukan literal timestamp setiap jam kerja |
| Execution Allocation | Daily allocation used to derive Execution Start/End |
| Commitment Allocation | Daily allocation used to derive Commitment Start/End |
| Actual Allocation window | Eligible Dates inside inclusive `Actual Start`–`Actual End`; Execution/Commitment dates are not part of this window |
| Existing Actual load | Canonical Actual Allocation of other completed Tasks for the same Assignee and Date |
| Available Actual capacity | Non-negative BAU Capacity remaining after Existing Actual load, normalized defensively to a `0.5`-hour quantum |
| Historical overcapacity | Total Actual Allocation pada suatu Date melebihi Actual BAU Capacity pada Date tersebut |
| Completion-order immutability | Existing completed Actual rows are not recomputed when another Task completes; the later command allocates around them |
| Locked baseline | Persisted Execution/Commitment dates and allocations protected while Project Locked |
| Scheduling-impacting mutation | Mutation that can change dates, allocations, dependency readiness, unscheduled state, priority order, or capacity available to another Task/Project |
| Recalculation-affected Project | Project other than the mutation owner whose scheduling state must be recalculated or whose schedule-relevant projection changes while evaluating downstream effects; this may include allocation/readiness/capacity changes with unchanged dates |
| Timeline-impacted Project | Project other than the mutation owner where simulation changes at least one Executable Task's Execution Start, Execution End, Commitment Start, or Commitment End; `null ↔ date` also counts |
| Impacted Project | In the cross-project warning/confirmation contract, alias for Timeline-impacted Project |
| Transitive recalculation scope | Recalculation-affected scope expanded through schedule-relevant propagation A→B→C until fixed point, even when an intermediate Project has no timeline change |
| Warning impacted set | Timeline-impacted subset of the transitive recalculation scope; only this set is listed in the generic cross-project warning |
| Required Locked Reopen Closure | Every Locked Project that must become Open together so a requested Reopen can be calculated without mutating any remaining Locked Project |
| Locked Project Name exception | Project Name may be renamed while Locked; all scheduling-relevant Project settings and Task/WBS planning data remain immutable |

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
- Recalculation of unfinished work in the transitive recalculation scope.
- Generic cross-project recalculation simulation plus timeline-only impact warning, confirmation, stale-preview protection, and atomic save.
- Impact caused by Task, dependency, priority, Member capacity/buffer, Capacity Override, Public Holiday, Project Buffer, Project Reopen, and future scheduling-impacting settings.
- Locked Project blocking rules, Project Name-only rename exception, and Actual Date exception.
- Atomic bulk reopen for mutually/transitively related Locked Projects.
- Direct Home entry for Locked Project rename and Locked Task Actual Date without hidden Project/Project Structure navigation.
- Lock eligibility, rollback, concurrency, structured errors, accessibility, and Three-Level Confidence evidence.

### 4.2 Out of Scope

- Forecast calculation and Forecast allocation.
- Delivery Impact and Project Health changes.
- Remaining-effort estimation or in-progress Actual Start-only lifecycle.
- Overtime compensation, recovery capacity, or carrying overcapacity as debt to another Date.
- User-editable Actual Allocation rows.
- Assignee-centric analytics page design; only the underlying consistent allocation projection is required now.
- Reason/detail per timeline-impacted Project in warning UI; warning lists Project names only.
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
- Scheduler recalculates unfinished Tasks in the transitive recalculation scope.

### 6.2 Planned Dependency Start Rules Remain

Before completion, normal Execution/Commitment scheduling remains authoritative:

- same-assignee predecessor/successor with zero Lag may share predecessor End Date when positive remaining capacity exists;
- different-assignee successor starts no earlier than the next working date after predecessor readiness;
- these rules determine planned Execution/Commitment only and never extend the Actual Allocation window outside Actual Date;
- these rules do not constrain Actual Date overlap after all predecessors are completed.

---

## 7. Actual Allocation

### 7.1 Canonical Projection and Head-to-Head Scope

Execution Allocation, Commitment Allocation, and Actual Allocation are distinct
daily projections. Actual Allocation is stored or deterministically
reconstructable as one canonical set of rows.

The same Actual Allocation rows must support:

1. **Task-centric view:** for Task X, which Assignee Date received how many hours.
2. **Assignee-centric view:** for Assignee Y on Date D, which completed Tasks consumed Actual capacity.

No separate calculation is allowed for the two points of view.

When constructing or reconstructing Actual Allocation for one completed Task,
only these rows reduce available capacity:

- canonical Actual Allocation of **other completed Tasks**;
- same Assignee;
- same allocation Date.

The following are ignored for this head-to-head calculation:

- unfinished automatic Execution/Commitment allocation;
- unfinished manual allocation;
- Locked baseline Execution/Commitment allocation;
- Commitment-only reservations;
- the Task's own old Actual rows while that same Task is being recalculated.

After completion succeeds, the new Actual rows become immutable existing load
for later completion commands. Existing completed rows are not rebalanced when
another Task completes. Therefore attribution may depend on confirmed completion
order. This is the approved **completion-order immutability** model; reporting
normalization or retrospective recomputation is deferred.

For unfinished scheduling after completion, Actual Allocation remains the sole
capacity-consuming reservation for that completed Task. Its Execution and
Commitment rows remain planning/baseline evidence and are not subtracted again.

### 7.2 Actual Allocation Window and BAU Capacity

Actual Allocation never uses Execution Start, Commitment Start, Project Start,
or any Date before Actual Start as an allocation baseline.

```text
Actual Allocation Window
= eligible Dates inside inclusive Actual Start–Actual End
```

Rules:

- Actual Start and Actual End remain factual calendar dates and may be weekend or Public Holiday.
- Normal Actual rows use working dates inside the Actual Date according to the effective Assignee calendar.
- Weekend and Public Holiday are excluded from normal distribution.
- Actual Allocation uses **BAU Capacity** before Member Buffer and Project Buffer:

```text
Actual BAU Capacity
= 0 on weekend/Public Holiday
= minimum active Capacity Override on the Date, when one or more exist
= otherwise Member Daily Capacity
```

- Member Buffer and Project Buffer do not alter Actual BAU Capacity.
- Planned Task Capacity Allocation Percentage from US-6.3 does not cap or scale Actual Allocation.
- If the Actual Date contains no working date, all Effort is allocated on Actual End as zero-capacity historical overcapacity. No row is created before Actual Start or after Actual End.

### 7.3 Precision and Conservation Invariants

Effort and Actual Allocation use a `0.5`-hour quantum.

Mandatory invariants:

```text
sum(Actual Allocation rows for Task) = Task Effort

Actual Allocation row >= 0
Actual Allocation row is a multiple of 0.5 hour
```

The normal domain flow already guarantees that Daily Capacity, Capacity
Override, Effort, and Actual Allocation use `0.5`-hour increments. As defensive
handling only, if legacy/corrupt data produces fractional Available Actual
Capacity, usable capacity is floored to the nearest lower `0.5` hour. This floor
is not a normal business scenario and does not authorize accepting invalid new
capacity data.

### 7.4 Balanced Forward Allocation with Progressive Recalculation

Let the eligible Actual Dates be ordered `D1 ... Dn`. For each Date, calculate:

```text
Existing Actual Load(D)
= sum Actual Allocation of other completed Tasks
  for the same Assignee and Date

Available Actual Capacity(D)
= max(0, Actual BAU Capacity(D) - Existing Actual Load(D))
```

The first pass distributes Effort chronologically while keeping the remaining
Effort as balanced as possible across the remaining Dates:

1. Express Remaining Effort in `0.5`-hour units.
2. Divide those units by the number of remaining eligible Dates.
3. The current Date receives the rounded-up share when a remainder exists; this gives normal distribution remainder to earlier Dates.
4. Allocate the smaller of that target and Available Actual Capacity on the current Date.
5. Subtract the confirmed allocation and repeat the same calculation from the next Date using the new Remaining Effort and number of remaining Dates.
6. A constrained Date therefore causes the remaining Effort to be divided again across every later eligible Date; shortfall is not merely added to the immediately following Date.

Base example without existing load:

```text
Effort: 10h
Actual Date: Date 1–3
Result: 3.5h / 3.5h / 3h
```

Rebalanced example:

```text
BAU Capacity: 8h/day
Existing Actual Load: 6h / 0h / 0h
Available: 2h / 8h / 8h
Effort: 10h
```

- Date 1 target is `3.5h`, but only `2h` is available.
- Remaining `8h` is recalculated over Date 2–3 as `4h / 4h`.

```text
New Actual Allocation: 2h / 4h / 4h
```

Repeated constraint example:

```text
Existing Actual Load: 6h / 7h / 0h
Available: 2h / 1h / 8h
Effort: 10h
New Actual Allocation: 2h / 1h / 7h
```

### 7.5 Backfill Before Historical Overcapacity

If Remaining Effort is still positive after the forward pass, the system must
not create new overcapacity while unused BAU capacity still exists anywhere in
the Actual Allocation Window.

Backfill rules:

1. Recompute spare capacity on every eligible Actual Date after the forward pass.
2. Fill spare capacity chronologically from the earliest eligible Date.
3. Continue until Remaining Effort becomes zero or no spare capacity remains.
4. Backfill never changes Existing Actual Load and never moves allocation outside Actual Date.

Example:

```text
BAU Capacity: 8h/day
Existing Actual Load: 0h / 8h / 8h
Effort: 10h
```

The forward pass initially gives `3.5h / 0h / 0h`, leaving `6.5h`. Date 1 still
has `4.5h` spare, so backfill increases Date 1 to `8h`. Only the remaining `2h`
is unavoidable overcapacity.

### 7.6 Unavoidable Overcapacity Equalization

When Remaining Effort is still positive after backfill, it is unavoidable
historical overcapacity. Allocate it in `0.5`-hour units to make **total
historical overcapacity across Actual Dates as even as possible**.

For each `0.5`-hour unit:

1. Calculate current total overcapacity per Date:

```text
Current Overcapacity(D)
= max(
    0,
    Existing Actual Load(D)
      + New Actual Allocation(D)
      - Actual BAU Capacity(D)
  )
```

2. Choose the Date with the smallest Current Overcapacity.
3. When several Dates tie, choose the latest Date.
4. Add `0.5h` to New Actual Allocation on that Date.
5. Repeat until Remaining Effort is zero.

Example after the Section 7.5 backfill:

```text
Existing Actual Load: 0h / 8h / 8h
Within-capacity New Allocation: 8h / 0h / 0h
Unavoidable remainder: 2h
```

Equalized result:

```text
New Actual Allocation: 8.5h / 0.5h / 1h
Final total load:       8.5h / 8.5h / 9h
Final overcapacity:     0.5h / 0.5h / 1h
```

The extra `0.5h` goes to the latest tied Date. The Task total remains exactly
`10h`.

Second example:

```text
BAU Capacity: 8h/day
Existing Actual Load: 6h / 7h / 8h
Effort: 10h
```

Within-capacity allocation is `2h / 1h / 0h`; unavoidable remainder is `7h`.
Equalization produces additional overcapacity `2h / 2.5h / 2.5h`, therefore:

```text
New Actual Allocation: 4h / 3.5h / 2.5h
```

Existing historical overcapacity is also considered. With Existing Actual Load
`10h / 8h / 8h` and new Effort `3h`, the preferred new allocation is
`0h / 1.5h / 1.5h`, producing final overcapacity `2h / 1.5h / 1.5h` instead of
making the already-most-overcapacity Date worse.

### 7.7 Capacity Effect and No Carry-Over

For unfinished scheduling on Date D and a specific timeline:

```text
Remaining Capacity(D)
= max(0, Final Timeline Capacity(D) - Actual Allocation(D))
```

Rules:

- Actual Allocation may exceed Actual BAU, Execution, or Commitment capacity.
- Historical overcapacity is valid and must not produce `SCHEDULING_DATA_INTEGRITY_CONFLICT` merely because it exceeds capacity.
- Overcapacity does not become negative-capacity debt on later Dates.
- No recovery capacity or overtime compensation exists.
- The next working Date uses its own normal capacity minus Actual Allocation on that Date.
- Saving Actual Date may recalculate unfinished Open work after Actual rows become fixed anchors.

### 7.8 Allocation Verification UI

View/Edit Task provides a read-only expandable **Capacity Allocation** section
with three groups:

- Execution Allocation;
- Commitment Allocation;
- Actual Allocation.

Each Actual row shows at minimum:

- Date;
- allocated hours;
- Actual BAU Capacity;
- Existing Actual Load considered when this allocation was created or the resulting total load;
- remaining BAU capacity or historical overcapacity.

Execution/Commitment rows use their own timeline capacity. Actual Allocation is
displayed only when Actual Date is complete. The UI must not imply that
analytical allocation dates are literal timestamps of work or that planned Task
Capacity Allocation Percentage was applied.

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

### 8.2 Planning Immutability and Project Name Exception

While Project Locked, the following are prohibited:

- create, rename, edit, delete, move, or reorder Task/WBS;
- add child or Executable-to-Group conversion;
- change Task/WBS Role, Assignee, Effort, Lag, Name, or manual/generated timeline;
- create, edit, delete, retarget, or change dependency ownership;
- Reopen completed Task;
- Project Settings changes that alter scheduling behavior.

The only Project-level edit permitted while Locked is **Project Name**. Rules:

- Edit Project remains available from both Projects and the Home Project Name;
- only the Project Name control is enabled; status and every scheduling-related
  setting remain read-only;
- the update payload must contain only the normalized Name and concurrency data
  required by the established Project contract;
- mixed payloads that also attempt to change Settings, status, Priority, or any
  other protected field are rejected atomically;
- successful rename does not run impact simulation, scheduler recalculation,
  baseline mutation, status transition, or Project-version change beyond the
  normal metadata concurrency contract defined by US-3.1;
- rename uses the same shared Edit Project dialog and command from both entry
  points; Home must not route through or render Projects in the background.

Frontend disables/hides prohibited planning actions with a clear explanation.
Backend rejects direct API bypass. There is no implicit Reopen.

### 8.3 Actual Date Exception

Actual Date remains writable on an unfinished Task in a Locked Project. The
Task Name on Home opens the shared Edit Task dialog directly; Home must not
change route, mount Projects, or open Project Structure in the background. All
planning fields stay read-only while the eligible Actual Date controls remain
writable.

When saved:

- complete Actual Date is persisted;
- Task becomes completed;
- locked Execution/Commitment dates and allocations remain unchanged;
- the Locked Project itself is not rescheduled;
- Actual Allocation is calculated using the protected Execution Start baseline and Sections 7.3–7.7;
- new Actual Allocation changes capacity available to Open Projects;
- scheduler recalculates the transitive recalculation scope of Open Projects;
- Locked Project dependencies and WBS order remain unchanged;
- Forecast effect remains deferred.

Therefore, “Locked Project does not run scheduler” means the scheduler may not
mutate that Locked Project. It does not prevent Actual Date from triggering
recalculation of recalculation-affected Open Projects.

### 8.4 Actual Date Impact on Other Locked Projects

Actual Date is factual data and is an exception to ordinary Locked-impact
blocking. The timeline-only warning definition in Section 9 still applies:

- Actual Date remains saveable even when its Actual Allocation or readiness affects another Locked Project;
- another Locked Project is warning-impacted only when simulation shows that its protected Execution/Commitment Start or End would need to change under current scheduling rules;
- allocation-only overlap, capacity-pressure, readiness change, or unscheduled-reason change with unchanged protected dates does not put that Locked Project in the warning;
- when at least one other Project is timeline-impacted, UI shows the same grouped Locked/Open Project-name warning and requires confirmation;
- Locked timeline impact is informational for Actual Date and does not block confirmation;
- server revalidates the full recalculation state and current timeline-impacted set on Confirm;
- no Locked Project timeline, allocation baseline, dependency, or status is mutated;
- overlap between Actual Allocation and locked allocations is accepted as historical overcapacity;
- overlap between actual dependent Task ranges is accepted after predecessor-completion validation;
- recalculation-affected Open Projects are recalculated atomically with Actual Date save, including intermediate Projects that are not warning-impacted.

### 8.5 Close

Locked Project may be Closed after every Executable Task has a complete Actual
Date pair. Close does not rewrite the locked baseline.

---

## 9. Generic Cross-Project Impact Guard

### 9.1 Covered Mutations

The same simulation contract applies to every confirmed mutation that can change
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
run cross-project simulation. Project Name-only rename is explicitly outside
this guard.

### 9.2 Timeline-Only Warning Impact Definition

A Project other than the mutation owner is **timeline-impacted** only when the
server-side before/after simulation changes at least one Executable Task's
confirmed planning date:

- Execution Start;
- Execution End;
- Commitment Start; or
- Commitment End.

Comparison is value-based. `null → date` and `date → null` are timeline changes.
A Task date change counts even when the Project-level aggregate Start/End remains
identical because another Task still defines the same aggregate boundary.

The following changes **do not by themselves** make another Project warning-
impacted when all four date fields above remain unchanged for every Executable
Task in that Project:

- Execution or Commitment daily-allocation shape/amount;
- available or remaining capacity;
- Actual Allocation or Actual-capacity availability;
- dependency readiness or dependency-derived internal state;
- scheduled/unscheduled reason when the date fields remain unchanged;
- priority/order metadata;
- Project/group aggregate values that merely recompute to the same task-derived
  timeline.

The mutation-owner Project is excluded from the cross-project warning. For
non-Project-scoped mutations such as Member capacity or Public Holiday changes,
there is no owner Project to exclude.

### 9.3 Recalculation Scope Is Broader Than Warning Scope

Scheduler traversal must not use the warning impacted set as its propagation
boundary. It expands the **transitive recalculation scope** through any schedule-
relevant change that can affect downstream work, including dates, allocations,
capacity availability, readiness, or scheduled/unscheduled state.

Example:

```text
Mutation in Project B
=> Project A allocation changes, but A dates stay identical
=> A capacity change shifts Project C dates

Recalculation-affected: A and C
Warning-impacted: C only
```

Project A must still be recalculated/persisted as required even though it is not
listed in the warning. Independent Project D is not recalculated, persisted, or
version-updated merely because it shares no effective propagation path.

If recalculation changes an Open Project's non-date scheduling projection while
its dates stay unchanged, that projection may be persisted atomically without
adding the Project to the warning. The resulting version/concurrency state must
still participate in server revalidation.

### 9.4 Locked Project Evaluation

Locked Project Execution/Commitment dates and allocations remain immutable
anchors. Ordinary simulation must schedule Open work around those protected
reservations rather than treating allocation-only pressure as permission to
rewrite a Locked baseline.

A Locked Project is warning-impacted when the proposed mutation makes its
protected Execution/Commitment timeline invalid under current scheduling rules
or would require at least one protected Start/End to change if the Project were
recalculated. This counterfactual timeline delta is sufficient even though the
actual Locked baseline is never mutated.

Allocation-only pressure against a Locked reservation does not create a warning
when its protected Start/End remain valid. If an ordinary mutation requires a
Locked timeline change, existing Locked blocking applies. Actual Date keeps the
factual exception in Section 8.4.

### 9.5 Warning Behaviour

#### No Other Project Timeline Impact

Save proceeds without cross-project warning, even when another Open Project was
recalculated and only its allocation/readiness/capacity projection changed.

#### Open Timeline-Impacted Projects Only

Before save, UI shows a confirmation warning listing only timeline-impacted Open
Project names:

```text
Open Projects
- Project B
- Project C
```

No per-Project reason is required. User may Confirm or Cancel.

#### Locked and Open Timeline Impact

For ordinary planning/capacity mutation, UI groups timeline-impacted names:

```text
Locked Projects
- Project B
- Project C

Open Projects
- Project D
```

If at least one Locked Project is timeline-impacted:

- save is blocked;
- no partial mutation or Open-Project recalculation is persisted;
- user must Reopen the required Locked Projects first.

Actual Date follows the exception in Section 8.4 and remains saveable.

### 9.6 Server Revalidation and Atomic Save

Impact preview is advisory and version-bound. On Confirm, server must:

1. reacquire required transaction/scheduling locks;
2. recalculate the proposed mutation and full transitive recalculation scope;
3. recompute the current timeline-impacted warning set from before/after Task dates;
4. validate current Project statuses, scheduling state, and schedule versions;
5. compare the relevant state with the preview token/version;
6. persist the mutation and all allowed Open-Project recalculation atomically.

The confirmation token/revalidation boundary must cover the full schedule-
relevant recalculation state, not only Project names shown in the warning. A
hidden allocation/readiness/capacity change may alter downstream results even
when the visible warning list is unchanged.

If the timeline-impacted set or relevant scheduling/version state changed:

- do not save;
- return a stale-impact response with the new grouped timeline-impacted Project names;
- UI replaces the old warning and requires a new confirmation when confirmation is still required.

No stale confirmation may mutate a newly Locked Project. When the latest
simulation has no cross-project timeline impact, the server may proceed under
the normal latest-state atomic save path without presenting an obsolete warning.

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

If remaining Locked Project B would require a protected Execution/Commitment
Start or End change, B is added to the Required Locked Reopen Closure. Allocation-
only pressure with unchanged protected dates does not require B to be reopened;
its Locked baseline remains an immutable anchor. Simulation repeats as though A
and every required timeline-impacted Locked Project are Open. If that requires a
protected timeline change in Locked C, C is added. Expansion continues until
fixed point.

Connectivity alone is never sufficient to add a Project to the closure. Shared
Assignee or dependency links only make a Project part of the recalculation
candidate scope. Priority remains authoritative during simulation. Therefore,
when higher-priority Locked Project A keeps the same protected dates while
lower-priority Locked Project B is reopened, A remains Locked and is not added to
the required closure.

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

Open Projects whose timeline will change
- Project C
- Project D
```

The Open list is the timeline-impacted subset, not the complete recalculation
scope. An Automatic Scheduling Project that has no Scheduling Start Date and no
existing Execution/Commitment timeline may still be considered during
recalculation, but it is not listed as Reopen impact merely because it remains
unscheduled or its non-date scheduling metadata is refreshed.

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
- recalculates unfinished Tasks across the transitive recalculation Open scope;
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
- Open Projects may be recalculated even when their dates remain unchanged.
- Allocation-only pressure must schedule around Locked reservations and must not rewrite their protected allocation baseline.
- A change succeeds only when no Locked Project would require a protected Execution/Commitment timeline change.
- If a Locked Project is timeline-impacted, the ordinary change is blocked atomically.
- The response/UI lists only timeline-impacted names grouped by Locked/Open without detailed reasons.
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
| `PROJECT_LOCKED_READ_ONLY` | 409 | Prohibited mutation requested on Locked Project; Project Name-only rename and eligible Actual Date mutation are excluded |
| `SCHEDULING_IMPACT_CONFIRMATION_REQUIRED` | 409 | Mutation changes another Open Project's Execution/Commitment timeline and requires user confirmation |
| `SCHEDULING_LOCKED_PROJECT_IMPACT` | 409 | Ordinary mutation would require at least one Locked Project's protected Execution/Commitment timeline to change and is blocked |
| `SCHEDULING_IMPACT_STALE` | 409 | Timeline-impact set or relevant recalculation/version state changed after preview |
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
**And** normal allocation rows use eligible working dates
**And** the no-working-date Actual End fallback in AC-7 remains valid.

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

### AC-7 — Actual Allocation uses Actual Date only

**Then** allocation Dates are limited to eligible Dates inside Actual Start–Actual End
**And** Execution Start, Commitment Start, and earlier planned Dates are ignored
**And** if no working Date exists, all Effort is represented on Actual End as zero-capacity historical overcapacity.

### AC-8 — Existing load is Actual-versus-Actual only

**When** Actual Allocation is created or reconstructed
**Then** available capacity is reduced only by canonical Actual Allocation of other completed Tasks for the same Assignee/Date
**And** unfinished automatic, manual, Locked baseline, and Commitment allocations are ignored
**And** the Task's own previous Actual rows are excluded during its recalculation.

### AC-9 — Balanced distribution and progressive recalculation

**Given** Effort `10h` over three unconstrained working Dates
**Then** allocation is `3.5h / 3.5h / 3h`.

**Given** BAU `8h/day` and Existing Actual Load `6h / 0h / 0h`
**Then** allocation is `2h / 4h / 4h`
**And** the remaining `8h` is divided again across Date 2–3 rather than carried only into Date 2.

### AC-10 — Spare capacity is backfilled before overcapacity

**Given** unused BAU capacity remains on any Date in Actual Date after the forward pass
**Then** system backfills that spare capacity before creating new historical overcapacity
**And** no allocation is moved before Actual Start or after Actual End.

### AC-11 — Unavoidable overcapacity is equalized

**Given** Remaining Effort still exists after backfill
**Then** each `0.5h` unit is placed on the Date with the smallest current total historical overcapacity
**And** a tie is resolved in favour of the latest Date
**And** total Task allocation remains exactly equal to Effort.

### AC-12 — No overcapacity carry-over

**Given** Actual Allocation exceeds capacity on Date D
**Then** remaining capacity on D is minimum zero
**And** excess is not carried as debt to later Dates
**And** Existing Actual rows remain immutable when a later Task completes.

### AC-13 — Locked completion preserves baseline

**Given** unfinished Task on Locked Project
**When** Actual Date is saved
**Then** protected Execution/Commitment dates and allocations remain unchanged
**And** Actual Allocation is persisted
**And** owning Locked Project is not rescheduled.

### AC-14 — Locked completion recalculates affected Open Projects

**Given** Actual Allocation on Locked Project changes shared capacity/readiness
**When** Actual Date is saved
**Then** transitively recalculation-affected Open Projects are recalculated
**And** independent Projects are untouched.

### AC-15 — Actual Date is not blocked by Locked impact

**Given** completion affects another Locked Project
**When** grouped impact warning is confirmed and server revalidation succeeds
**Then** factual completion succeeds
**And** every Locked Project remains unchanged
**And** recalculation-affected Open Projects are recalculated
**And** overlap is represented as historical state rather than integrity failure.

### AC-16 — Locked planning mutation remains rejected

**Given** Project Locked
**When** non-Actual Task/WBS planning, dependency, Settings, lifecycle, or Task Reopen mutation is requested
**Then** request is rejected with `PROJECT_LOCKED_READ_ONLY`
**And** the Project Name-only exception cannot be used to smuggle another protected change.

### AC-16A — Locked Project may be renamed from Projects or Home

**Given** Project Locked
**When** user opens Edit Project from Projects or clicks the Project Name on Home
**Then** Project Name is editable
**And** status and every scheduling-related setting are read-only
**And** saving a valid Name-only change uses the same Project command from both entry points
**And** no scheduler, impact preview, baseline mutation, or status transition runs.

### AC-16B — Locked Task Actual Date opens directly from Home

**Given** unfinished Task belongs to a Locked Project
**When** user clicks the Task Name on Home
**Then** the shared Edit Task dialog opens directly over Home
**And** planning controls remain read-only
**And** eligible Actual Date controls remain writable
**And** Projects and Project Structure are not opened or rendered in the background.

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

### AC-20 — Candidate-only or non-date cross-project change does not warn

**Given** Project B mutation causes Project A to enter cross-project simulation,
including because B and A share an Assignee
**When** every Executable Task in A keeps the same Execution Start/End and
Commitment Start/End values
**Then** Project A is not listed in a cross-project warning
**And** no confirmation is required solely because of Project A
**And** this remains true whether A's allocation/readiness/capacity projection
also stays identical or changes without changing those dates.

### AC-21 — Any other-Project timeline change requires warning

**Given** a mutation changes at least one Executable Task's Execution Start,
Execution End, Commitment Start, or Commitment End in another Open Project
**Then** that Project is listed in the warning
**And** save occurs only after confirmation and server revalidation
**And** the warning still applies when the Project aggregate Start/End remains
unchanged.

### AC-22 — Locked timeline impact blocks ordinary save

**Given** a non-Actual mutation would require at least one protected Execution or
Commitment Start/End in a Locked Project to change
**Then** warning groups timeline-impacted Locked/Open Project names
**And** the complete operation is blocked with no partial save.

**Given** only allocation pressure against a Locked baseline changes while the
protected dates remain valid
**Then** the Locked Project is not warning-impacted
**And** its dates and allocations remain unchanged anchors.

### AC-23 — Recalculation propagates through non-warning intermediate Projects

**Given** mutation in B changes A's schedule-relevant allocation/capacity state
without changing A's dates
**And** that change causes C's Execution or Commitment date to change
**Then** A remains absent from the warning
**And** C appears in the warning
**And** A and C are both included in the required transitive recalculation scope
**And** independent D is not recalculated or version-updated.

### AC-24 — Stale impact confirmation cannot save

**Given** timeline impact, recalculation state, status, or relevant version changes after preview
**When** user confirms the old warning
**Then** save is rejected with `SCHEDULING_IMPACT_STALE`
**And** UI receives the current grouped timeline-impacted Project list
**And** unchanged warning names do not make a stale hidden allocation/readiness/capacity state valid.

### AC-24A — Capacity sources use the same timeline-only warning guard

**When** Daily Capacity, Member Buffer, Public Holiday, or Project Buffer changes
**Then** scheduler traverses the required transitive recalculation scope
**And** warning/blocking is based only on resulting Execution/Commitment timeline changes in other Projects.

**When** Capacity Override create/update/delete changes the resolved per-Date
minimum for at least one Date
**Then** the same rule applies.

**When** only the override record changes and the resolved per-Date minimum does
not change
**Then** the mutation persists without impact warning, scheduler invocation, or
schedule-version change.

### AC-25 — Mutual Locked Reopen produces closure

**Given** Reopen A requires B and Reopen B requires A
**When** Reopen A is requested
**Then** required Locked closure contains A and B
**And** user is offered Reopen All rather than an impossible single-Project sequence.

**Given** Locked Project A has higher Priority than Locked Project B
**And** reopening B leaves every protected Execution/Commitment Start/End in A unchanged
**When** B is reopened
**Then** A is not added to the Required Locked Reopen Closure
**And** B may reopen without requiring A solely because both Projects share an Assignee or dependency-connected recalculation scope.

**Given** Open Automatic Scheduling Project C has no Scheduling Start Date and no existing Execution/Commitment timeline
**When** C is encountered while previewing another Project Reopen
**Then** C is not listed in the Reopen warning unless at least one existing Task timeline date in C actually changes.

### AC-26 — Reopen closure expands transitively

**Given** A requires B and B requires Locked C
**Then** required closure contains A, B, and C.

### AC-27 — Bulk Reopen is atomic

**When** Reopen All is confirmed
**Then** every required Locked Project becomes Open together
**And** transitive recalculation Open scope is recalculated
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
| TC-5 | Execution Start Date 1, Effort 14h, Actual Date 2–3 | Actual ignores Date 1 and allocates `7h / 7h` on Date 2–3 |
| TC-6 | Effort 10h, Actual Date 1–3, no existing Actual load | Actual Allocation = `3.5h / 3.5h / 3h` |
| TC-7 | Actual Date includes weekend but has one working date | Normal allocation uses the working date only |
| TC-8 | Actual Date contains no working date | All Effort is represented on Actual End as zero-capacity historical overcapacity |
| TC-9 | Existing Actual load `6 / 0 / 0`, BAU `8`, Effort `10` | Progressive recalculation produces `2 / 4 / 4` |
| TC-9A | Existing Actual load `6 / 7 / 0`, BAU `8`, Effort `10` | Repeated recalculation produces `2 / 1 / 7` |
| TC-9B | Existing Actual load `0 / 8 / 8`, BAU `8`, Effort `10` | Backfill then equalization produces `8.5 / 0.5 / 1` |
| TC-9C | Existing Actual load `6 / 7 / 8`, BAU `8`, Effort `10` | Final new allocation is `4 / 3.5 / 2.5` |
| TC-9D | Existing Actual load `10 / 8 / 8`, BAU `8`, Effort `3` | New allocation is `0 / 1.5 / 1.5` to level total overcapacity |
| TC-10 | Historical overcapacity followed by next working day | No debt carry-over |
| TC-10A | Member/Project Buffer differs from Daily Capacity | Actual head-to-head uses BAU; unfinished timelines use their independently buffered capacities |
| TC-10B | Task A completes before overlapping Task B | A rows remain immutable; B allocates around A; reversing completion order may reverse attribution |
| TC-11 | Complete successor while predecessor unfinished | Rejected |
| TC-12 | Completed predecessor and overlapping successor Actual Date | Accepted |
| TC-13 | Locked Task receives Actual Date | Baseline unchanged; Actual Allocation stored |
| TC-14 | Locked Actual Date changes Open B allocation but not B dates; independent C | B recalculated without warning; C untouched |
| TC-15 | Locked Actual Date creates allocation-only overlap with another Locked Project and no protected date would change | No warning for that Locked Project; save succeeds; both locked baselines unchanged |
| TC-15A | Rename Locked Project from Projects | Name changes; protected fields/baseline/status unchanged; no scheduler or impact preview |
| TC-15B | Rename Locked Project from Home | Same shared command/result as Projects; route stays Home |
| TC-15C | Submit Locked Project Name plus protected Settings change | Whole request rejected with `PROJECT_LOCKED_READ_ONLY` |
| TC-15D | Open Locked Task from Home and save Actual Date | Direct shared Task dialog; planning fields read-only; factual save follows Locked Actual Date rules |
| TC-16 | View completed Task allocation | Execution/Commitment/Actual groups show daily rows and totals |
| TC-17 | Query Task POV and assignee POV | Same allocation row totals |
| TC-18 | Edit Project B causes shared-Assignee Project A to enter simulation, but all A Execution/Commitment dates stay unchanged | No warning for A and no confirmation solely because A was considered/recalculated |
| TC-18A | Project B mutation changes A allocation with unchanged A dates, and that hidden change shifts C timeline | A recalculated but omitted from warning; warning names C; confirm required |
| TC-19 | Edit Task would change Locked B protected timeline and Open C timeline | Blocked; grouped B/C names displayed |
| TC-20 | Actual Date would require Locked B protected timeline change | Grouped warning/confirm; factual save succeeds; B unchanged |
| TC-21 | Member Daily Capacity impacts Open Projects | Same warning/confirmation contract |
| TC-22 | Member Buffer impacts Locked Project | Save blocked |
| TC-23 | Capacity Override changes resolved minimum | Same generic guard |
| TC-23A | Capacity Override record changes but minimum is unchanged | Persist record; no warning/scheduler/version change |
| TC-23B | Public Holiday or Project Buffer impact | Same generic guard |
| TC-24 | Hidden recalculation state or timeline-impact set changes between preview and confirm | Stale confirmation rejected; current warning list returned |
| TC-25 | A changes B allocation with unchanged B dates, which shifts C timeline | B/C recalculated; warning contains C only; independent D untouched |
| TC-26 | Mutual A/B Locked Reopen | Required closure A/B; Reopen All offered |
| TC-26A | A/B Locked share an Assignee; A has higher Priority; reopening lower-priority B leaves A dates unchanged | B reopens without requiring A |
| TC-26B | Connected Open auto Project C has no Scheduling Start Date and no existing timeline | C may remain in recalculation scope but is absent from Reopen warning |
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
- Actual-Date-only allocation window and no-working-date Actual End fallback.
- Actual-versus-Actual existing-load scope and self-row exclusion on reconstruction.
- `0.5`-hour balanced target, progressive remaining-date recalculation, and conservation of Effort.
- Spare-capacity backfill before overcapacity.
- Total-overcapacity equalization with latest-Date tie-breaker.
- Completion-order immutability, historical overcapacity, and no carry-over.
- Completed predecessor validation with allowed Actual overlap.
- Lock eligibility, Locked mutation policy, and Project Name-only exception.
- Separation of transitive recalculation scope from timeline-only warning impact, including Locked and Actual Date exceptions.
- Required Locked Reopen closure/fixed-point expansion.

### 16.2 Application Tests

- Completion coordination for Open and Locked Projects.
- Locked Project Name-only rename bypasses scheduling impact while mixed protected payloads fail atomically.
- Generic full-scope scheduling preview with timeline-only warning classification and confirmation token.
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
- Transitive recalculation-scope traversal through non-warning intermediate Projects.
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
- Locked read-only bypass protection, including mixed Name + protected-field payloads.

### 16.5 Frontend and Acceptance-Level Tests

- Actual Date pair input and validation.
- Read-only allocation verification section.
- Non-working factual dates with working-date allocation display.
- Grouped Open/Locked warning contains timeline-impacted Project names only.
- Confirmation, stale warning refresh, cancel, and no partial save.
- Mutual Locked Reopen All workflow.
- Locked Project rename from Projects and Home uses one shared command and leaves scheduling state untouched.
- Locked Task opens directly from Home for Actual Date without hidden route/page orchestration.
- Accessibility, focus management, pending state, retry/reload, and stale-response protection.

Every affected Acceptance Criterion requires Code Inspection, Unit/Integration,
and Acceptance-Level evidence under the repository Three-Level Confidence rule.

---

## 17. Documentation Impact

This story requires reconciliation of:

- US-1.2 Member Daily Capacity and Buffer impact behavior;
- US-2.1 Capacity Override impact behavior;
- US-2.2 Public Holiday impact behavior;
- US-3.1 Project Lock/Reopen/Priority lifecycle and Locked Project Name-only edit;
- US-3.3 Project Buffer and Settings impact behavior;
- US-4.1 Task completion, Actual Date fields, and allocation verification;
- US-4.2 clearing both Actual Start and Actual End;
- US-5.1 completed predecessor validation and Actual overlap semantics;
- US-6.1 trigger, completed allocation, transitive impact, and Locked anchors;
- US-6.3 planned Task Capacity Allocation Percentage and its explicit non-application to Actual Allocation;
- US-7.1 canonical Home WBS surface, direct shared dialogs, and removal of Project Structure;
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
- Actual Allocation uses only eligible Dates inside Actual Start–Actual End; Execution/Commitment dates never extend its window.
- Actual Allocation capacity is BAU Resolved Daily Capacity; Member and Project buffers apply only to Execution/Commitment.
- Only other completed Actual Allocation for the same Assignee/Date reduces capacity when new Actual rows are constructed; unfinished automatic/manual/Locked planned rows are ignored.
- Normal allocation uses `0.5`-hour balanced shares, recalculates Remaining Effort over remaining Dates after each constrained Date, and backfills any spare capacity before overcapacity.
- Unavoidable historical overcapacity is levelled by current total overcapacity; equal ties allocate the next `0.5h` to the latest Date.
- Existing completed Actual rows are immutable, so attribution may depend on completion order.
- If Actual Date has no working date, all Effort is represented on Actual End as zero-capacity historical overcapacity.
- Historical overcapacity is valid and never carried as debt.
- One canonical allocation projection supports Task-centric and future assignee-centric views and is the sole capacity-consuming reservation for a completed Task.
- Current UI displays Execution, Commitment, and Actual allocation in Task details for verification.
- Locked Project baseline remains immutable, but Project Name may be renamed and Actual Date/Actual Allocation may be recorded.
- Locked Project Name-only rename is metadata-only: it uses the same Project command from Projects and Home and never runs scheduler or impact preview.
- A Locked Project rename payload that also changes any protected field is rejected atomically.
- Locked Task Edit opens directly over Home for eligible Actual Date mutation; no hidden Projects or Project Structure navigation is permitted.
- Locked completion may trigger scheduler for recalculation-affected Open Projects while never mutating Locked Projects.
- Actual Date remains saveable after grouped warning/confirmation even when another Locked Project has a counterfactual protected timeline impact.
- Ordinary scheduling-impacting mutation is blocked when any Locked Project would require a protected Execution/Commitment timeline change.
- Open-only cross-project warning is required only for Execution/Commitment timeline changes; allocation/readiness/capacity-only changes do not warn.
- Impact warning lists timeline-impacted Project names only, grouped by Locked/Open, excluding the current Project when one exists.
- Recalculation scope is transitive through schedule-relevant changes; the warning set is its timeline-impacted subset and excludes independent Projects.
- Daily Capacity, Member Buffer, Public Holiday, Project Buffer, Priority, and other effective capacity changes use the same impact guard.
- Capacity Override overlap is allowed; the minimum active override is resolved per Member/Date before buffers.
- Capacity Override mutation uses the guard only when that resolved minimum changes on at least one Date.
- Mutual/transitive Locked Reopen uses an atomic required closure with Reopen All; partial Reopen is not allowed.
- Valid unscheduled output does not fail Reopen.
- Forecast remains deferred.

---

## 19. Unresolved Questions

None.
