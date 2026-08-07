# US-6.3 — Task Capacity Allocation Percentage

> **Product decision update — US-4.4:** Task Capacity Allocation Percentage editability and scheduling trigger resolve from the Task's effective lifecycle and Automatic Scheduling configuration. A Task in a Locked Group is planning read-only even when Project is Open, and a Group override may make a Task automatic/manual independently from the raw Project toggle.

> **Authority and supersession:** This story is the authoritative requirement
> for Task-level planned capacity percentage, concurrent same-assignee planned
> allocation, percentage rounding, manual fixed allocation, migration/default
> behaviour, and the resulting automatic-dependency reconciliation. It
> supersedes contradictory whole-Task, no-overlap, contiguous-gap, and
> non-preemption wording in US-6.1 Sections 10.3, 11.2, 11.5, 13.2–13.4, 14,
> 15, AC-18–AC-22, and their related test cases. US-6.1 remains authoritative
> for capacity resolution, buffers, dependency readiness, Lag, portfolio
> priority, scheduling transactions, impact coordination, and all rules not
> explicitly changed here. US-6.2 remains authoritative for Actual Allocation,
> including the Actual-Date-only, Actual-versus-Actual, balanced/rebalanced allocation revision recorded together with this story.
> US-6.5 supersedes this story's former Assignee reset contract: Capacity
> Allocation Percentage is preserved when Assignee is first selected, changed,
> or cleared so every recommendation and confirmed mutation uses the same visible
> Task-level percentage.

## 1. User Story

**Sebagai** Engineering Lead,
**Saya ingin** membatasi persentase kapasitas harian seorang Assignee untuk Task tertentu,
**Sehingga** satu Assignee dapat menjalankan beberapa Task pada hari yang sama, misalnya `20%` untuk fixing dan menggunakan sisa kapasitas untuk development, tanpa mengubah Project Priority dan visual WBS order sebagai urutan utama scheduler.

Story ini merupakan bagian dari **Epic 6: Scheduling Engine** dan memperluas
Task planning contract, Automatic Scheduling, Manual Scheduling, daily
allocation projection, dan automatic dependency by Assignee.

---

## 2. Business Context

US-6.1 sebelumnya memodelkan satu Assignee sebagai antrean serial whole-Task:
setelah satu Task mulai, Task tersebut menggunakan seluruh remaining capacity
pada setiap eligible Date sampai selesai dan Task lain tidak boleh berjalan
paralel. Model itu tidak mencerminkan pola kerja seperti fase testing, ketika
seorang engineer dapat membatasi fixing ke sebagian kapasitas hariannya sambil
menggunakan sisa capacity untuk development.

Task Capacity Allocation Percentage adalah **maximum planned capacity per
Date**, bukan guaranteed reservation dan bukan priority baru. Scheduler tetap
memproses Task berdasarkan dependency readiness, Project Priority, lalu visual
WBS order. Task yang lebih dahulu dapat memakai capacity lebih dahulu sampai
batas hariannya. Task berikutnya menggunakan remaining capacity. Bila Task
berpersentase tidak memperoleh capacity karena Task yang lebih prioritas sudah
menghabiskan hari tersebut, Task itu belum mulai dan dicoba pada eligible Date
berikutnya.

Nilai default `100%` mempertahankan behaviour Task existing: Task dapat memakai
seluruh remaining capacity pada Date tersebut. Nilai yang lebih kecil membuat
Task dapat overlap dengan Task berikutnya untuk Assignee yang sama tanpa
mengizinkan total automatic allocation melebihi final timeline capacity.

---

## 3. Terminology

| Term | Meaning |
| --- | --- |
| Task Capacity Allocation Percentage | Integer `1–100` yang membatasi planned capacity maksimum satu Executable WBS per Date |
| Default Allocation | Persisted value `100` ketika Task dibuat tanpa explicit percentage |
| Final Timeline Capacity | Execution Capacity atau Commitment Capacity setelah capacity precedence, buffer, dan rounding `0.5` jam |
| Task Daily Limit | Maximum planned allocation Task pada satu Date untuk timeline tertentu setelah percentage dan rounding diterapkan |
| Ordered Task | Dependency-ready unfinished Task yang ditempatkan menurut Project Priority lalu visual WBS order |
| Remaining Timeline Capacity | Final Timeline Capacity dikurangi fixed reservations dan planned allocations yang sudah ditempatkan lebih dahulu pada Date tersebut |
| Concurrent Planned Allocation | Dua atau lebih unfinished Task dengan Assignee sama menerima positive allocation pada Date yang sama |
| Fixed Manual Allocation | Daily allocation projection dari manual timeline yang tidak digeser scheduler otomatis; terhadap unfinished automatic Task, capacity precedence mengikuti Project Priority |
| Planned Overcapacity | Manual fixed allocation pada Date melebihi final timeline capacity; valid karena manual dates bersifat authoritative |
| Effective `100%` | Task boleh memakai seluruh remaining capacity, bukan reservation eksklusif atas seluruh daily capacity |

---

## 4. Scope

### 4.1 In Scope

- Field `Capacity Allocation (%)` pada Executable WBS.
- Persisted integer default `100` dan validation `1–100`.
- Migration/backfill seluruh Executable WBS existing menjadi `100`.
- Preserve allocation ketika Assignee pertama dipilih, berubah, atau dikosongkan.
- Independent Execution dan Commitment Task Daily Limit.
- Rounding Task Daily Limit ke kelipatan `0.5` jam.
- Minimum positive Task Daily Limit `0.5` jam.
- Project Priority dan visual WBS order tetap mendahului percentage.
- Concurrent same-assignee planned allocation menggunakan remaining capacity.
- Automatic allocation tidak boleh overcapacity.
- Manual timeline menghasilkan fixed daily allocation dan boleh planned overcapacity.
- Fixed manual allocation memengaruhi shared Assignee capacity untuk Project otomatis berdasarkan Project Priority.
- Reconciliation automatic dependency yang aman untuk parallel allocation.
- Preview, confirmation, transaction, schedule version, stale-response, Locked,
  completed, structural conversion, API, persistence, UI, accessibility, dan
  Three-Level Confidence evidence.
- Explicit confirmation bahwa planned percentage tidak membatasi Actual Allocation.

### 4.2 Out of Scope

- Percentage allocation per Role, Member, Project, Group, dependency, atau Date range.
- Percentage yang berubah per Date.
- Separate Execution percentage dan Commitment percentage.
- User-defined Task Priority; priority tetap Project Priority lalu visual WBS order.
- Guarantee/reservation bahwa Task berpersentase selalu memperoleh jatah hariannya.
- Automatic redistribution untuk mencapai average percentage mingguan/bulanan.
- Actual Allocation percentage atau timesheet entry.
- Remaining Effort atau in-progress lifecycle.
- Assignee analytics redesign.
- Forecast Allocation dan Locked Project Forecast behaviour.
- Optimizer yang mencari kombinasi percentage paling efisien.

---

## 5. Task Field Contract

### 5.1 Ownership and Applicability

- `Capacity Allocation (%)` adalah executable attribute milik Task/Executable WBS.
- Grouping WBS tidak boleh memiliki atau mengedit field ini.
- Field disimpan bersama Assignee, Effort, Lag, dan executable planning data.
- Satu Task memiliki satu percentage yang digunakan untuk Execution dan
  Commitment; timeline capacity masing-masing tetap dihitung secara independen.
- Percentage tidak dimiliki Assignee dan tidak mengubah Member Daily Capacity.

### 5.2 Valid Value

Persisted Task value wajib:

```text
1 <= Capacity Allocation Percentage <= 100
```

Rules:

- integer only;
- `0`, negative, fractional, malformed, non-numeric, `null`, dan value di atas
  `100` tidak valid untuk persisted Executable WBS;
- user tidak dapat memilih `0%`;
- minimum user-selectable value adalah `1%`;
- no custom value tetap direpresentasikan oleh persisted `100`, bukan `null` atau `0`;
- `100%` berarti maksimum seluruh Final Timeline Capacity, tetapi Task tetap
  hanya memperoleh Remaining Timeline Capacity setelah work yang lebih prioritas.

### 5.3 Default, Create, and Update Semantics

- Add Task menampilkan default `100%`.
- Create command yang tidak mengirim field karena legacy client dinormalisasi
  backend menjadi `100`.
- Update command yang tidak mengirim field mempertahankan persisted value existing;
  omission tidak boleh mengubah custom percentage menjadi `100` atau `0`.
- Explicit `0` selalu ditolak; backend tidak boleh memaknai `0` sebagai omitted.
- API must distinguish omitted field from explicit numeric zero at its boundary.
- Backend/domain validation adalah source of truth; frontend default bukan satu-satunya protection.

### 5.4 Assignee Selection, Change, and Clear

- Capacity Allocation Percentage adalah Task attribute dan tidak dimiliki oleh
  Assignee.
- Ketika Assignee pertama dipilih, Capacity Allocation draft tetap menggunakan
  value yang terlihat; custom value tidak di-reset menjadi `100`.
- Ketika Assignee berubah ke Member lain, draft dan persisted percentage tetap
  dipertahankan kecuali user mengirim explicit valid percentage baru.
- Update command yang mengganti Assignee dan tidak mengirim explicit percentage
  mempertahankan persisted value existing.
- Ketika Assignee dikosongkan, draft dan persisted percentage tetap dipertahankan.
- Task tanpa Assignee boleh menyimpan custom percentage dan tetap unscheduled
  menurut US-6.1.
- Role change yang membuat current Assignee tidak sesuai juga tidak mengubah
  percentage.
- US-6.5 menggunakan percentage yang sama untuk seluruh candidate simulation;
  backend/frontend tidak boleh memakai implicit `100%` untuk candidate lain.

### 5.5 Structural Conversion

- Executable-to-Grouping conversion memindahkan percentage bersama seluruh
  executable attributes ke conversion child yang menerima Assignee/Effort/Lag.
- Group yang dihasilkan tidak menyimpan percentage.
- Grouping-to-Executable conversion tidak mengarang custom percentage dan
  menggunakan default `100`.
- Move, conversion, atau retarget failure merollback hierarchy, percentage,
  executable data, dependency ownership, dates, dan allocations atomically.

---

## 6. Migration and Backward Compatibility

Fitur ini menambah persisted field baru. Tanpa migration yang benar, zero-value
persistence atau application mapping dapat membuat Task lama terbaca `0%` dan
tidak pernah memperoleh capacity. Behaviour tersebut dilarang.

Mandatory rollout order:

1. Add the new storage field using a migration-safe representation.
2. Backfill **every existing Executable WBS** to `100`.
3. Verify no existing Executable WBS remains `null`, missing, or `0`.
4. Apply database/domain invariant that Executable WBS value is `1–100`.
5. Set create/default behaviour to `100`.
6. Keep Grouping WBS representation consistent with the existing executable-
   attribute storage model; Group may remain non-applicable but must never be
   interpreted as a schedulable `0%` Task.
7. Deploy read/write mapping that treats legacy create omission as `100`, update
   omission as preserve-existing, and explicit `0` as invalid.

Additional rules:

- Migration is atomic and rerunnable/idempotent according to repository standards.
- Migration failure must not leave a partially backfilled production state.
- Existing Execution/Commitment dates and allocation rows are not rewritten by
  the data backfill alone; confirmed recalculation follows normal scheduling-
  impact coordination when the implementation is activated.
- Rollback strategy must not silently discard a custom value after users can edit it.
- No implementation may rely only on a Go/TypeScript/database numeric zero-value.

---

## 7. Timeline Capacity and Task Daily Limit

### 7.1 Final Timeline Capacity Remains Authoritative

US-6.1 capacity precedence and formulas remain unchanged:

```text
Execution Capacity
= MROUND(
    Resolved Daily Capacity
    × (1 − Member Buffer),
    0.5
  )

Commitment Capacity
= MROUND(
    Resolved Daily Capacity
    × (1 − Member Buffer)
    × (1 − Project Buffer),
    0.5
  )
```

Percentage is applied **after** the applicable Final Timeline Capacity has been
resolved and rounded. It does not override Daily Capacity, Capacity Override,
Public Holiday, Member Buffer, or Project Buffer.

### 7.2 Task Daily Limit Formula

For each Task and Date:

```text
Raw Task Daily Limit
= Final Timeline Capacity × Capacity Allocation Percentage / 100
```

If Final Timeline Capacity is `0`, Task Daily Limit is `0`.

If Final Timeline Capacity is positive:

```text
Rounded Task Daily Limit
= round Raw Task Daily Limit to nearest 0.5 hour
  using deterministic half-up behaviour

Task Daily Limit
= max(0.5 hour, Rounded Task Daily Limit)
```

The result is capped at Final Timeline Capacity so `100%` never exceeds the
applicable timeline capacity.

Examples:

```text
Final capacity 5.5h, percentage 50%
Raw limit       2.75h
Rounded limit   3.0h

Final capacity 5.5h, percentage 20%
Raw limit       1.10h
Rounded limit   1.0h

Final capacity 8h, percentage 1%
Raw limit       0.08h
Rounded limit   0h
Minimum limit   0.5h
```

### 7.3 Allocation is Limited by Remaining Capacity

Task Daily Limit is a maximum, not a guarantee:

```text
Allocated(Task, Date)
= min(
    Remaining Effort,
    Task Daily Limit,
    Remaining Timeline Capacity(Date)
  )
```

Therefore:

- Task may receive less than its percentage limit;
- Task may receive `0` when higher-priority/fixed work exhausted the Date;
- unused capacity is available to the next ordered Task;
- unused percentage is not carried to later Dates;
- a Task cannot claim tomorrow's capacity because it received less today;
- sum of configured percentages may exceed `100`; daily actual planned
  allocation remains bounded by Remaining Timeline Capacity in automatic mode.

---

## 8. Automatic Scheduling Rules

### 8.1 Priority Remains Primary

Dependency readiness is applied first. Among ready Task for one Assignee:

1. Project Priority ascending;
2. visual depth-first WBS order.

Percentage does not create a separate reservation group and does not move a
Task ahead of higher-priority work.

A lower-priority `20%` Task may receive no allocation on a Date when a prior
`100%` Task consumes the whole capacity. Its Start is the first later eligible
Date with positive allocation.

### 8.2 Ordered Allocation Across the Horizon

For each timeline independently:

1. Load immutable completed Actual Allocation and Locked baseline reservations as absolute reservations, then load fixed manual allocations as immutable rows whose capacity precedence follows Project Priority.
2. Resolve dependency/Lag readiness.
3. Select the next ready Task by Project Priority and visual WBS order.
4. From its earliest eligible Date, allocate its Effort across Dates using its
   Task Daily Limit and each Date's Remaining Timeline Capacity.
5. A Date with no remaining capacity gives the Task `0`; search continues.
6. The Task may coexist with later ordered Task on any Date where its own limit
   leaves remaining capacity.
7. Continue with the next ordered Task, which fills remaining capacity subject
   to its own limit and readiness.
8. Reconcile automatic dependency ownership using Section 10.
9. Persist both timelines and allocation projections atomically under US-6.1/
   US-6.2 transaction and impact rules.

### 8.3 Start and End

For Execution and Commitment independently:

- Start Date is the first Date with positive allocation.
- End Date is the last Date with positive allocation.
- Task may have positive allocation on the same Date as another Task with the
  same Assignee.
- Task with custom percentage may end after a later ordered default-`100%` Task.
- End order is not required to match Project Priority or WBS order.
- Generated dates never become priority input.

### 8.4 Revised Non-Preemption Meaning

The old rule “one running Task exclusively consumes the Assignee until complete”
is superseded.

The retained invariant is:

- lower-priority work may use only capacity left after higher-priority/fixed work;
- lower-priority work must not reduce a higher-priority Task below the allocation
  that the higher-priority Task would otherwise receive under its own daily limit;
- recalculation with newly introduced higher-priority work may displace mutable
  future allocation of lower-priority work;
- completed Actual Allocation, Locked baseline, and fixed manual allocation are
  never displaced, but a lower-priority fixed manual allocation does not reduce
  capacity available to a higher-priority automatic Task;
- no automatic allocation may exceed Final Timeline Capacity.

Task overlap caused by percentage is valid and is not considered preemption.

### 8.5 Percentage and Dependency

- Manual/effective dependency still overrides priority and percentage.
- A Task receives no allocation before dependency/Lag readiness.
- Same-assignee `Lag = 0` successor may use remaining capacity on predecessor End
  Date only after predecessor readiness is satisfied.
- Different-assignee boundary rules remain owned by US-6.1.
- Percentage cannot be used to bypass a dependency.

---

## 9. Worked Examples

### 9.1 Original 5.5-Hour 50/50 Example

```text
Assignee final capacity: 5.5h
Task I percentage:       50%
Task II percentage:      50%
Order: I before II
```

Each Task Daily Limit is `3.0h` after rounding.

```text
Task I receives 3.0h
Remaining capacity is 2.5h
Task II receives 2.5h, not 3.0h
Total is 5.5h
```

The remaining-capacity constraint prevents overcapacity. Priority/WBS order
resolves which Task receives the rounded half-hour when both raw limits are
`2.75h`.

### 9.2 A/B/C Share Capacity from Day 1

```text
Project Start: Date 1
Assignee capacity: 8h/day
Order: A → B → C
A Effort 4h, 100%
B Effort 4h, 20%  → daily limit 1.5h
C Effort 4h, 100%
```

| Date | A | B | C | Remaining |
| --- | ---: | ---: | ---: | ---: |
| 1 | 4h | 1.5h | 2.5h | 0h |
| 2 | 0h | 1.5h | 1.5h | 5h |
| 3 | 0h | 1h | 0h | 7h |

Result:

| Task | Start | End |
| --- | --- | --- |
| A | Date 1 | Date 1 |
| B | Date 1 | Date 3 |
| C | Date 1 | Date 2 |

C may finish before B because B is intentionally capped. No auto dependency is
created among Tasks that can all start on the earliest eligible Date solely by
sharing capacity.

### 9.3 Higher-Priority Task Exhausts Day 1

```text
Project Start: Date 1
Assignee capacity: 8h/day
Order: A → B → C
A Effort 8h, 100%
B Effort 4h, 20%  → daily limit 1.5h
C Effort 4h, 100%
```

| Date | A | B | C | Remaining |
| --- | ---: | ---: | ---: | ---: |
| 1 | 8h | 0h | 0h | 0h |
| 2 | 0h | 1.5h | 4h | 2.5h |
| 3 | 0h | 1.5h | 0h | 6.5h |
| 4 | 0h | 1h | 0h | 7h |

B and C start on Date 2. B's configured `20%` does not reserve Date 1 ahead of A.

### 9.4 Configured Percentages Above 100%

```text
Capacity: 8h
Task A: 70% → limit 5.5h
Task B: 70% → limit 5.5h
Order: A before B
```

On a Date with no other reservation:

```text
A receives 5.5h
B receives 2.5h
Total remains 8h
```

The configuration is valid; scheduler uses actual remaining capacity.

### 9.5 Manual versus Automatic Project Priority

Common input:

```text
Assignee Daily Capacity: 8h
Task A: Automatic Project, Effort 16h
Task B: Manual Project, Effort 8h, fixed Date 1–2
Manual projection for B: 4h on Date 1 and 4h on Date 2
```

When Project A has higher priority:

| Date | Task A automatic | Task B manual | Aggregate |
| --- | ---: | ---: | ---: |
| 1 | 8h | 4h | 12h |
| 2 | 8h | 4h | 12h |

Task A remains Date 1–2. Task B remains Date 1–2. The overlap is accepted and
produces no overcapacity warning because Task B's own `4h` row does not exceed
its `8h` Task Daily Limit; the lower-priority manual row does not consume Task
A's higher-priority capacity.

When Project B has higher priority:

| Date | Task B manual | Task A automatic | Remaining |
| --- | ---: | ---: | ---: |
| 1 | 4h | 4h | 0h |
| 2 | 4h | 4h | 0h |
| 3 | 0h | 8h | 0h |

Task B remains Date 1–2. Task A uses shared remaining capacity and finishes on
Date 3.

---

## 10. Automatic Dependency by Assignee

Parallel planned allocation invalidates the former assumption that every
same-assignee Task forms a serial queue. Automatic dependency must represent a
real Finish-to-Start readiness boundary and must not be invented merely because
allocations overlap.

### 10.1 When No Auto Blocker Exists

Do not create capacity-based automatic ownership when:

- Task starts on its earliest dependency/Lag/Project-anchor eligible Date; or
- Task shares that Date with prior ordered Task using remaining capacity; or
- capacity becomes available because of calendar/capacity change rather than a
  prior Task completion; or
- the only candidate prior Task continues after blocked Task Start, because a
  Finish-to-Start relation would incorrectly move the blocked Task later.

Manual dependency ownership remains unchanged.

### 10.2 Safe Auto Blocker

A capacity-based auto blocker may be created only when all conditions hold:

1. blocked Task received no positive allocation on one or more earlier eligible
   Dates because prior ordered same-assignee mutable/fixed allocation exhausted
   capacity;
2. a specific prior Task completion releases positive capacity that allows the
   blocked Task's first allocation;
3. selected blocker Execution End is not after blocked Task Execution Start;
4. adding the Finish-to-Start relation reproduces, rather than changes, the
   confirmed Execution Start;
5. effective graph remains acyclic.

When several Tasks complete on the controlling Date, choose deterministically
using reverse scheduling order among the completion allocations whose removal
first creates at least `0.5h` positive capacity for the blocked Task.

### 10.3 Maximum Ownership

- At most one automatic blocker per Task remains the persisted contract.
- Automatic ownership uses confirmed Execution allocation only.
- Commitment uses the same effective graph but calculates independent capacity.
- Reconciliation must remove an old serial auto dependency that is no longer
  required after percentage allows same-day overlap.
- Reconciliation must not retarget or delete manual ownership.
- If no safe blocker exists, Task may be capacity-delayed without an automatic
  dependency; the allocation projection remains authoritative for that delay.

### 10.4 Stability

Automatic relation reconciliation must be deterministic and bounded:

- adding a relation must not push Start beyond the allocation it was intended to explain;
- relation that materially changes readiness requires recalculation;
- oscillation between parallel allocation and serial relation is a scheduler failure;
- failure rolls back percentage, dates, allocations, ownership, and schedule version.

---

## 11. Manual Scheduling and Fixed Allocation

### 11.1 Manual Dates Remain Authoritative

When Automatic Scheduling is OFF:

- Execution and Commitment Start/End remain user-entered fixed date pairs under US-4.1;
- Capacity Allocation Percentage remains editable on Open unfinished Task;
- Save is not rejected merely because Effort cannot fit within Final Timeline
  Capacity or Task Daily Limit inside the manual range;
- planned overcapacity is expected and valid in manual mode;
- manual dates are never expanded automatically to make Effort fit.

### 11.2 Fixed Daily Allocation Projection

For each complete manual timeline pair independently:

1. Determine eligible working Dates inside the inclusive manual range.
2. If at least one working Date exists, distribute Task Effort evenly in `0.5`-hour units across all eligible Dates.
3. When the units do not divide evenly, assign the extra `0.5h` units from the earliest Date forward.
4. Compare each row with Task Daily Limit and Final Timeline Capacity.
5. Persist the result as fixed manual allocation even when a row exceeds either limit; the excess is planned overcapacity.
6. If the manual range has no working Date, allocate all Effort on manual Start as a fixed zero-capacity overcapacity row so the reservation remains visible and does not silently disappear.

The percentage remains useful as the expected per-Date ceiling and overcapacity
reference, but manual dates override the ceiling when both cannot be satisfied.

### 11.3 Interaction with Automatic Projects

- Completed Actual Allocation and Locked baseline remain absolute reservations
  that reduce automatic capacity regardless of Project Priority.
- Fixed manual allocation is immutable, but competes with unfinished automatic
  allocation by Project Priority on the same Assignee/timeline/Date.
- A higher-priority manual Project reduces capacity available to a lower-priority
  automatic Project. The automatic Task uses the remaining capacity and may move
  to later eligible Dates.
- A lower-priority manual Project does not reduce capacity available to a
  higher-priority automatic Project. Both allocations remain on the Date even
  when their cumulative total exceeds Final Timeline Capacity.
- This priority-authorized overlap is accepted without validation error,
  confirmation, or overcapacity warning. It does not create capacity debt on
  later Dates.
- Manual daily allocation that independently exceeds its own Task Daily Limit or
  Final Timeline Capacity retains the existing manual planned-overcapacity
  behaviour from Sections 11.1–11.2.
- Current active Project Priority is unique; no equal-priority manual-versus-
  automatic tie exists.
- Automatic allocation uses:

```text
Priority-Available Capacity
= max(
    0,
    Final Timeline Capacity
    − completed Actual reservations
    − Locked baseline reservations
    − higher-priority fixed manual allocations
    − higher-priority mutable automatic allocations
  )
```
- Cross-project dependency readiness uses the manual Task's persisted End Date.
- A Project automatic scheduler must not move or rewrite another Project's
  manual dates or fixed manual allocation.
- Mutation of manual dates, Effort, Assignee, or percentage follows US-6.2
  scheduling-impact preview/confirmation and Locked-impact rules.

---

## 12. Actual Allocation Interaction

Task Capacity Allocation Percentage is a planning constraint only.

- It applies to Execution Allocation and Commitment Allocation.
- It does not cap, scale, reserve, or redistribute Actual Allocation.
- Actual Allocation is limited to eligible Dates inside Actual Start–Actual End and never uses pre-completion Execution/Commitment dates as its window.
- When Actual Allocation is built, it competes only with canonical Actual Allocation of other completed Tasks for the same Assignee/Date.
- Unfinished automatic allocation, manual fixed allocation, and Locked baseline allocation are ignored for Actual head-to-head distribution.
- Actual Allocation uses `0.5h` balanced shares, progressive remaining-date recalculation, spare-capacity backfill, and total-overcapacity equalization from revised US-6.2.
- Existing completed Actual rows remain immutable; a later completion allocates around them.
- Actual Allocation may be below or above the planned percentage.
- Historical overcapacity remains valid and has no carry-over debt.
- Completing a Task replaces its unfinished capacity consumption with canonical Actual Allocation according to US-6.2; planned percentage remains persisted for display but is not applied to Actual rows.
- Reopen removes Actual Date/Actual Allocation and returns the Task to planned scheduling using its persisted percentage, subject to US-4.2 and US-6.2.

---

## 13. Lifecycle, Editing, and Triggers

### 13.1 Editable State

- Open unfinished Task: field editable.
- Open completed Task: read-only until Reopen succeeds.
- Locked Task: read-only; Actual Date remains the only Task mutation exception.
- Closed Project: read-only.
- Group: field absent/read-only non-applicable.

### 13.2 Scheduling Trigger

Changing Capacity Allocation Percentage is scheduling-impacting because it can
change:

- Execution/Commitment allocation and dates;
- capacity available to later Tasks/Projects;
- automatic dependency ownership;
- Project summary/Gantt range;
- transitive recalculation scope.

With Automatic Scheduling ON:

- blur may request rollback-only preview under existing Task preview contract;
- Save performs latest-state impact simulation;
- required confirmation, stale-preview token, atomic recalculation, rollback,
  schedule version, and stale-response protection follow US-6.1/US-6.2.

With Automatic Scheduling OFF:

- percentage change recalculates fixed manual allocations for complete manual
  date pairs;
- manual dates remain unchanged;
- cross-project impact is still simulated because shared capacity may change.

### 13.3 Locked Impact

- Ordinary percentage change is prohibited on Locked Project.
- Change on Open/manual/automatic Project is blocked when it would mutate another
  Locked Project's protected baseline under US-6.2.
- Actual Date exception does not make percentage editable while Locked.

---

## 14. UI and Accessibility

### 14.1 Task Form

- Place `Capacity Allocation (%)` in the Task scheduling section before the
  final Role and Assignee fields defined by US-6.5.
- Control is integer numeric input or equivalent accessible spin control.
- Default visible value is `100`.
- Display `%` as unit without making it part of the numeric value.
- Helper text communicates: `Maximum planned capacity per day. Remaining capacity may be used by other tasks.`
- The field is disabled/non-editable when Task state is not editable.
- Selecting, changing, or clearing Assignee preserves the visible draft value.
- Add Task still starts at `100` only as the Task default, not as an Assignee-change side effect.
- Validation does not wait for scheduler preview.

### 14.2 Allocation Verification

Existing read-only Capacity Allocation section continues to show Execution,
Commitment, and Actual rows. For planned rows, expose enough information to
verify percentage behaviour:

- Date;
- allocated hours;
- Final Timeline Capacity;
- configured percentage;
- rounded Task Daily Limit;
- remaining timeline capacity or planned overcapacity for manual rows.

Actual rows must not imply that percentage was applied.

### 14.3 Interaction States

- Preserve valid draft on backend/scheduler failure.
- Prevent duplicate Save.
- Expose preview/loading/error/success states through existing shared patterns.
- Validation message is associated with the input and keyboard/screen-reader accessible.
- Percentage changes and allocation details must not rely on colour alone.

---

## 15. API and Persistence Contract

Naming may follow repository conventions, but all boundaries must represent:

```text
capacityAllocationPercentage: integer
```

Mandatory semantics:

- create omission → `100`;
- update omission → preserve existing;
- explicit `0`/invalid → `INVALID_TASK_CAPACITY_ALLOCATION`;
- Assignee first-select/change/clear with percentage omission → preserve existing;
- Role change or Role/Assignee mismatch → preserve existing;
- accepted value included in confirmed Task response;
- preview and confirmed schedule response expose allocation rows consistent with value;
- Group payload cannot smuggle executable percentage;
- persistence constraint and domain validation agree;
- generated OpenAPI/types/mappers, if present, must preserve omitted-versus-zero semantics;
- no partial write when scheduling or allocation persistence fails.

Recommended structured error:

| Code | HTTP | Meaning |
| --- | ---: | --- |
| `INVALID_TASK_CAPACITY_ALLOCATION` | 422 | Percentage is omitted where required after normalization, non-integer, below `1`, or above `100` |

Existing error codes remain authoritative for Locked, completed, stale preview,
concurrency, integrity, dependency cycle, and scheduling persistence failures.

---

## 16. Acceptance Criteria

### AC-1 — Default and persisted range

**When** a new Executable WBS is created without explicit percentage
**Then** confirmed value is `100`.

**When** user saves integer `1–100`
**Then** value persists and is returned after reload.

### AC-2 — Invalid percentage rejected

**When** value is `0`, negative, fractional, malformed, empty after required
normalization, or above `100`
**Then** frontend and backend reject it with `INVALID_TASK_CAPACITY_ALLOCATION`
**And** confirmed Task and schedule remain unchanged.

### AC-3 — Migration backfills existing Tasks

**Given** Executable WBS created before the field exists
**When** migration completes
**Then** every existing Executable WBS has confirmed value `100`
**And** no existing Task is interpreted as `0%` or unschedulable because of
language/database zero-value.

### AC-4 — Legacy omission semantics

**Given** legacy create request omits percentage
**Then** backend stores `100`.

**Given** legacy update request omits percentage
**Then** backend preserves current value.

**Given** request explicitly sends `0`
**Then** backend rejects it rather than treating it as omitted.

### AC-5 — Assignee changes preserve percentage

**Given** Task Capacity Allocation is a valid custom value
**When** Assignee is first selected, changed, or cleared
**Then** UI preserves the visible percentage draft
**And** backend preserves the confirmed percentage when update omission is used
**And** only an explicit valid percentage changes the value.

### AC-6 — Percentage applied after final capacity

**Given** final Execution Capacity `5.5h` and Task percentage `50%`
**Then** raw limit is `2.75h`
**And** Task Daily Limit is `3.0h`.

**Given** separate Commitment Capacity
**Then** Commitment limit is calculated independently from that capacity.

### AC-7 — Minimum positive half-hour

**Given** positive Final Timeline Capacity and percentage whose rounded result is `0`
**Then** Task Daily Limit is `0.5h`.

**Given** Final Timeline Capacity is `0`
**Then** Task Daily Limit remains `0`.

### AC-8 — Priority precedes percentage

**Given** Task A `100%` is ordered before Task B `20%`
**And** A consumes the whole Date
**Then** B receives `0` on that Date
**And** B Start is a later Date with positive allocation.

### AC-9 — Later Task uses remaining capacity

**Given** prior Task allocation is below Final Timeline Capacity
**When** next ready Task is processed
**Then** next Task may use remaining capacity up to its own Task Daily Limit.

### AC-10 — Configured sum may exceed 100%

**Given** two ready Tasks each configured `70%`
**Then** configuration is valid
**And** ordered allocations are capped by actual Remaining Timeline Capacity
**And** automatic daily total does not exceed final capacity.

### AC-11 — Concurrent same-assignee allocation

**Given** A `4h/100%`, B `4h/20%`, C `4h/100%`, capacity `8h`, and order A→B→C
**When** scheduler runs from Date 1
**Then** Date 1 allocations are A `4h`, B `1.5h`, C `2.5h`
**And** B and C both start Date 1
**And** C may finish before B.

### AC-12 — No unused-percentage carry-over

**Given** Task receives less than its daily limit because Effort finishes or
remaining capacity is smaller
**Then** unused limit is available to later ordered Task on that Date
**And** it is not added to Task limit on a later Date.

### AC-13 — Higher-priority recalculation displacement

**Given** mutable lower-priority future allocations exist
**When** higher-priority ready work is introduced or reordered ahead
**Then** scheduler recalculates lower work around the higher-priority Task Daily Limit
**And** fixed/completed/Locked allocations are unchanged
**And** automatic total never exceeds capacity.

### AC-14 — Dependency still blocks percentage allocation

**Given** Task has unsatisfied effective dependency
**Then** it receives no allocation even when remaining capacity exists.

### AC-15 — No auto dependency for valid parallel start

**Given** multiple same-assignee Tasks receive positive allocation on their
common earliest eligible Date
**Then** scheduler does not create serial automatic dependency solely because
one appears earlier in priority/WBS order.

### AC-16 — Safe automatic blocker only

**Given** Task start is delayed by exhausted capacity
**When** a prior Task completion is the deterministic event that releases first
positive capacity
**And** its End is not after blocked Start
**Then** it may become the single automatic blocker.

**When** candidate prior Task continues after blocked Start
**Then** it must not become a Finish-to-Start auto blocker.

### AC-17 — Stale serial ownership removed

**Given** existing automatic dependency was required by old serial allocation
**When** custom percentage now permits valid same-day overlap
**Then** automatic ownership is removed if no longer required
**And** manual ownership on the same endpoint is preserved.

### AC-18 — Manual dates accept overcapacity

**Given** Automatic Scheduling OFF and fixed date range cannot fit Effort within
Task Daily Limit or Final Timeline Capacity
**When** user saves valid manual dates and percentage
**Then** Save succeeds
**And** dates remain unchanged
**And** fixed manual allocation records planned overcapacity.

### AC-19 — Manual allocation obeys Project Priority

**Given** fixed manual allocation and an unfinished automatic Task share one
Assignee/timeline/Date
**When** the manual Project has higher Project Priority
**Then** its fixed allocation is subtracted before the automatic Task
**And** the automatic Task uses shared remaining capacity and may finish later
**And** manual dates/allocation are not moved.

**Given** the automatic Project has higher Project Priority
**When** the scheduler recalculates
**Then** the lower-priority fixed manual allocation does not reduce the automatic
Task's available capacity
**And** the automatic Task keeps the schedule it would receive without that
lower-priority manual allocation
**And** the manual dates/allocation remain unchanged
**And** cumulative overcapacity is accepted without warning or later capacity debt.

### AC-20 — Manual range with no working Date

**Given** complete manual range contains no working Date
**When** manual Task is saved
**Then** dates remain authoritative
**And** all Effort is represented on manual Start as fixed zero-capacity
planned overcapacity
**And** allocation does not silently disappear.

### AC-21 — Actual Allocation ignores planned percentage

**Given** Task planned percentage is `20%`
**When** complete Actual Date is saved
**Then** Actual Allocation follows revised US-6.2 Actual-Date-only and Actual-versus-Actual rules
**And** uses `0.5h` balanced/rebalanced allocation without any planned percentage cap
**And** may be below or above `20%`.

### AC-22 — Completion and Reopen

**When** Task completes
**Then** canonical Actual Allocation becomes its capacity-consuming historical anchor.

**When** Task is reopened successfully
**Then** Actual Allocation is removed
**And** unfinished scheduling resumes using the persisted Task percentage.

### AC-23 — Locked/completed restrictions

**Given** Task completed, Project Locked, or Project Closed
**When** percentage mutation is attempted outside allowed lifecycle
**Then** request is rejected with existing lifecycle error
**And** no schedule or allocation changes.

### AC-24 — Structural conversion preserves value

**When** Executable WBS converts to Grouping
**Then** percentage moves to the conversion child with other executable data
**And** Group owns no percentage.

**When** Grouping converts to Executable without prior executable value
**Then** value is `100`.

### AC-25 — Impact preview and atomicity

**When** percentage or manual fixed allocation changes another Project's schedule
**Then** US-6.2 impact preview/confirmation applies.

**When** preview is stale, Locked impact blocks, scheduler fails, or persistence fails
**Then** percentage, Task mutation, dependencies, dates, allocations, snapshots,
and schedule version rollback atomically.

### AC-26 — Stale response protection

**Given** two percentage/Assignee previews or saves overlap
**When** older response arrives last
**Then** it cannot restore stale percentage, Assignee, dates, allocation, or
automatic ownership.

### AC-27 — Allocation verification UI

**When** planned allocation is displayed
**Then** user can verify Date, allocated hours, timeline capacity, percentage,
and rounded Task Daily Limit
**And** manual overcapacity is distinguished from automatic allocation
**And** Actual rows do not imply percentage enforcement.

### AC-28 — Three-Level Confidence

**Then** every AC has traceable Code Inspection, Unit/Integration Test, and
Acceptance-Level Test evidence under `AGENTS.md`
**And** existing pre-US-6.3 serial-scheduler evidence is not reused as proof of
this story without new coverage.

---

## 17. Mandatory Test Scenarios

### Domain/Application

- valid `1`, `20`, `50`, `100`;
- invalid `0`, negative, fractional, malformed, above `100`;
- create omission versus update omission versus explicit zero;
- Assignee first-select/change/clear preservation semantics;
- half-up `0.5` rounding and minimum positive `0.5`;
- independent Execution/Commitment limits;
- A/B/C examples from Section 9;
- configured percentages above `100` total;
- dependency-ready versus blocked Tasks;
- parallel start without auto ownership;
- safe blocker selection and unsafe continuing blocker rejection;
- stale serial ownership removal;
- manual even allocation and planned overcapacity;
- no-working-date manual range;
- Actual Allocation unaffected by planned percentage;
- conversion preserve/default rules.

### Repository/Integration

- migration backfills every existing Executable WBS to `100`;
- conditional executable/group persistence invariant;
- explicit zero rejected at persistence/domain boundary;
- allocation row uniqueness and half-hour deterministic arithmetic;
- completed/Locked reservations subtract absolutely; fixed manual reservations subtract from automatic rows only when the manual Project has higher priority;
- automatic rows never exceed final capacity;
- manual rows may exceed capacity and persist as fixed;
- percentage mutation rollback with injected scheduler/allocation failure;
- relation reconciliation atomic with dates and allocations;
- concurrency and schedule-version conflict;
- legacy update omission preserves custom value;
- structural conversion transaction preserves percentage.

### Frontend Component

- default `100`, unit, helper text, label, keyboard and screen-reader semantics;
- integer validation and preserved draft;
- Assignee first-select/change/clear preserves the draft;
- lifecycle read-only states;
- preview loading/failure/stale response;
- allocation verification fields;
- manual overcapacity indication without colour-only meaning.

### Acceptance-Level

1. Set `20%` fixing Task and verify later development Task uses remaining capacity each day.
2. Put a `100%` higher-priority Task first and verify the `20%` Task starts later when no capacity remains.
3. Verify `5.5h`, `50%`, `50%` produces `3.0h` then `2.5h` by order.
4. Verify A/B/C `4h` example produces overlapping dates and C completes before B.
5. Select/change/clear Assignee and verify custom percentage is preserved without stale schedule.
6. Edit percentage and confirm impacted Open Projects; verify Locked impact blocks atomically.
7. Higher-priority Manual Project consumes shared capacity and moves a lower-priority automatic Task; lower-priority Manual Project overlaps without moving the higher-priority automatic Task or showing a warning.
8. Complete a `20%` Task and verify Actual Allocation follows US-6.2 rather than `20%`.
9. Reopen and verify planned percentage is reused.
10. Upgrade a pre-feature database and prove no existing Task becomes `0%` or loses dates merely from backfill.

---

## 18. Definition of Done

- Approved user story and supersession links are consistent across US-4.1,
  US-6.1, US-6.2, US-6.5, project context, and implementation evidence.
- Migration/backward-compatibility behaviour is implemented and tested before
  enabling percentage scheduling.
- Automatic same-assignee parallel allocation is deterministic and capacity-safe.
- Manual fixed allocation is deterministic, visible, immutable, and honoured cross-project according to Project Priority.
- Actual Allocation uses revised US-6.2 Actual-Date-only, Actual-versus-Actual, balanced/rebalanced rules and ignores planned percentage.
- Automatic dependency never serializes a valid parallel schedule incorrectly.
- All mutations preserve transaction, Locked protection, impact confirmation,
  concurrency, stale-response, accessibility, and Three-Level Confidence rules.
- No code may claim conformance using only the old serial-scheduler test suite.
