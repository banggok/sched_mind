# US-6.1 — Automatic Execution and Commitment Scheduling

> **Product decision update — US-7.1:** Home Portfolio Gantt is the canonical
> active-Project WBS surface and a read-only consumer of confirmed
> Execution/Commitment dates, unscheduled state, dependencies, and schedule
> versions. The standalone Project Structure entry point is removed. Projection
> switching or timeline range changes never invoke this scheduler; mutations
> opened directly from Home continue
> through the existing owning use cases and US-6.2 impact coordination.

> **Product decision update — US-6.2:** Actual Date range, Open timeline actualization, Actual Allocation, historical overcapacity, Locked Project immutability, generic cross-project impact warning/confirmation, capacity-setting impact, atomic bulk Project Reopen, transitive impacted-scope recalculation, and Priority validation around Locked Projects are owned by US-6.2 and supersede contradictory wording in this story.

> **Product decision update — US-6.3:** Task Capacity Allocation Percentage allows
> concurrent same-assignee planned allocation while preserving dependency
> readiness, Project Priority, and visual WBS order. US-6.3 supersedes this
> story's exclusive whole-Task/non-overlap rules in Sections 10.3, 11.2,
> 11.5, 13.2–13.4, 14, 15, AC-18–AC-22, and related test cases wherever they
> conflict. Capacity resolution, Lag, transactions, impact coordination, and
> unaffected rules in this story remain authoritative.

> **Product decision update — US-6.5:** Assignee recommendation with Automatic
> Scheduling `ON` must invoke this same concrete scheduler in rollback-only
> candidate simulation. US-6.5 does not define a second automatic algorithm and
> does not change confirmed scheduler ordering, capacity, dependency, horizon,
> transaction, or missing-anchor behaviour. Automatic Scheduling `OFF` advisory
> recommendation is owned entirely by US-6.5 and does not mutate manual dates.

## 1. User Story

**Sebagai** Engineering Lead,
**Saya ingin** sistem membentuk urutan kerja assignee, dependency otomatis, Lag, Execution Timeline, dan Commitment Timeline ketika Automatic Scheduling aktif,
**Sehingga** tanggal Task terisi secara konsisten berdasarkan prioritas, dependency, kapasitas harian, dan buffer tanpa menyusun timeline secara manual.

Story ini merupakan bagian dari **Epic 6: Scheduling Engine** dan memenuhi concrete scheduler contract yang sebelumnya hanya disiapkan oleh Project, WBS, dan Dependency stories.

---

## 2. Business Context

SchedMind menggunakan satu portfolio-level Scheduling Engine karena satu Team Member dapat mengerjakan Task dari beberapa Project dan dependency dapat melintasi Project.

Task bukan entity terpisah. **Task adalah Executable WBS**, yaitu WBS leaf yang tidak mempunyai child. Grouping WBS tidak dijadwalkan dan tidak mengonsumsi kapasitas.

Ketika Project menggunakan `Automatic Scheduling = ON`:

- Assignee, Effort, dan Task Capacity Allocation Percentage dari US-6.3 menentukan konsumsi planned capacity.
- Project Priority lalu urutan WBS menentukan urutan Task ketika beberapa Task sama-sama eligible.
- Manual dependency menentukan batas readiness yang tidak boleh dilanggar.
- Scheduler membentuk paling banyak satu safe auto-dependency by assignee hanya ketika completion satu Task benar-benar menjelaskan delay Start Task lain; valid same-day parallel allocation tidak boleh diserialkan.
- Lag pada Task dapat menunda earliest start yang telah ditentukan scheduler.
- Execution Start/End dihitung menggunakan Execution Capacity.
- Commitment Start/End dihitung ulang menggunakan Commitment Capacity.
- Execution dan Commitment dates bersifat date-only dan read-only di UI.

Ketika `Automatic Scheduling = OFF`, Execution dan Commitment tetap menggunakan manual timeline contract dari US-4.1. Story ini tidak mengubah manual dates secara otomatis.

---

## 3. Baseline and Superseded Rules

Story ini menggunakan baseline berikut:

- US-1.2 — Manage Team Members.
- US-2.1 — Manage Capacity Override.
- US-2.2 — Manage Public Holiday.
- US-3.1 — Create Project.
- US-3.3 — Configure Project Settings.
- US-4.1 — Manage WBS.
- US-4.2 — Reopen Completed Task.
- US-5.1 — Manage Dependency.

Story ini secara eksplisit **menggantikan** rule lama berikut untuk concrete Scheduling Engine:

1. Rule lama bahwa Execution Capacity sama dengan Daily Capacity dan tidak dipengaruhi buffer.
2. Rule lama bahwa Member Buffer hanya digunakan pada Commitment Timeline.
3. Rule lama bahwa hasil setelah Member Buffer disebut Commitment Capacity; pada model baru hasil tersebut menjadi rounded Execution Capacity, sedangkan Commitment Capacity dihitung terpisah dari Resolved Daily Capacity menggunakan Member Buffer dan Project Buffer.
4. Out-of-scope Lag dan Auto Dependency pada US-5.1.
5. No-op scheduler adapter yang hanya membuktikan port invocation.

Rule baru:

```text
Resolved Daily Capacity
= Public Holiday `0`
  / minimum active Capacity Override for the Member and Date
  / Daily Capacity when no override applies

Execution Capacity
= MROUND(
    Resolved Daily Capacity × (1 − Member Buffer Percentage),
    0.5
  )

Commitment Capacity
= MROUND(
    Resolved Daily Capacity
    × (1 − Member Buffer Percentage)
    × (1 − Project Buffer Percentage),
    0.5
  )
```

Contoh:

```text
Resolved Daily Capacity = 8 jam
Member Buffer           = 30%
Project Buffer          = 20%

Raw Execution Capacity
= 8 × 70%
= 5.6 jam/hari

Execution Capacity
= MROUND(5.6, 0.5)
= 5.5 jam/hari

Raw Commitment Capacity
= 8 × 70% × 80%
= 4.48 jam/hari

Commitment Capacity
= MROUND(4.48, 0.5)
= 4.5 jam/hari
```

Execution Capacity dan Commitment Capacity masing-masing dibulatkan sekali ke kelipatan `0.5` jam setelah semua buffer yang relevan untuk timeline tersebut diterapkan.

---

## 4. Terminology

| Product Term            | Meaning                                                                                                                                    |
| ----------------------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| Task                    | Executable WBS / WBS leaf                                                                                                                  |
| Group                   | Grouping WBS / WBS yang mempunyai child                                                                                                    |
| Manual Dependency       | Dependency yang dibuat atau dipertahankan secara eksplisit oleh user melalui `Blocks` / `Blocked by`                                       |
| Auto Dependency         | Dependency by assignee yang dibentuk oleh scheduler karena Task sebelumnya menghalangi earliest start                                      |
| Blocking Task           | Task yang harus selesai lebih dahulu                                                                                                       |
| Blocked Task            | Task yang menunggu blocker                                                                                                                 |
| Lag                     | Non-negative integer pada Task yang menunda earliest start dalam satuan hari kalender, kemudian tunduk pada availability harian            |
| Resolved Daily Capacity | Kapasitas sebelum buffer: weekend/Public Holiday `0`; otherwise minimum active Capacity Override per Member/Date; otherwise Daily Capacity |
| Execution Capacity      | Resolved Daily Capacity setelah Member Buffer                                                                                              |
| Commitment Capacity     | Resolved Daily Capacity setelah Member Buffer dan Project Buffer, lalu dibulatkan ke kelipatan `0.5` jam                                     |
| Remaining Capacity      | Kapasitas Task assignee yang belum terpakai pada satu Date dan timeline tertentu                                                           |
| Contiguous Allocation   | Task yang sudah mulai terus dialokasikan pada setiap Date eligible berikutnya sampai Effort selesai; Task tidak dihentikan untuk Task lain |
| Preemption              | Menghentikan Task yang sudah mulai, menjalankan Task lain, lalu melanjutkan Task pertama                                                   |
| Scheduling Anchor       | Project Scheduling Start Date                                                                                                              |
| Active Portfolio        | Seluruh Open dan Locked Project yang dapat menjadi scheduling input; only Open unfinished Tasks are mutable scheduler output, Locked Projects are immutable anchors, and Closed Projects are excluded |
| Impacted Scheduling Scope | Project awal dan Open Projects yang benar-benar terdampak secara transitif melalui shared capacity, dependency, priority competition, atau approved scheduling constraints |

---

## 5. Scope

### 5.1 In Scope

- Concrete portfolio-level Execution Scheduling Engine.
- Concrete portfolio-level Commitment Scheduling Engine.
- Task-level `Lag` field dengan default `0`.
- Auto Dependency by Assignee saat Automatic Scheduling aktif.
- Per-day capacity resolution dan allocation.
- Remaining-capacity carry-over pada tanggal yang sama untuk Task dengan assignee yang sama.
- Non-preemptive Task scheduling.
- Project Priority lalu WBS order sebagai scheduling priority.
- Manual and automatic dependency coexistence.
- Cross-project capacity allocation dan dependency.
- Recalculation setelah established scheduler triggers.
- Persistence Execution Start/End dan Commitment Start/End sebagai date-only values.
- Atomic mutation, auto-dependency update, scheduling, and rollback.
- Structured errors, cache invalidation, stale-response protection, concurrency, accessibility, and responsive behaviour.
- Three-Level Confidence Matrix untuk setiap Acceptance Criterion.

### 5.2 Out of Scope

- Forecast Scheduling Engine.
- Delivery Impact calculation.
- Project Health calculation.
- Percentage complete atau remaining-effort estimation.
- Actual Start.
- Partial task completion.
- Task-level manual priority field.
- Resource levelling selain berdasarkan Assignee.
- Multi-assignee Task.
- Parallel execution oleh satu Assignee pada waktu yang sama.
- Preemption atau split Task di sekitar Task lain.
- Overtime, working-hour clock, shift, atau intraday start/end timestamp.
- Lag per dependency relation.
- Dependency type selain Finish-to-Start.
- Gantt drag scheduling.
- What-if simulation.
- Bulk schedule override.
- Capacity Override/Public Holiday CRUD ownership dan impact-preview UI;
  owning stories dan US-6.2 menentukan mutation coordination, sedangkan story
  ini hanya memiliki capacity resolution dan allocation algorithm.
- Permission model baru.

---

## 6. Domain Model Changes

### 6.1 Task Lag

Executable WBS menambahkan field berikut:

| Field | Type                      | Required | Default | Rules                                                                      |
| ----- | ------------------------- | -------: | ------: | -------------------------------------------------------------------------- |
| Lag   | Non-negative integer days |       Ya |     `0` | Task-owned; bukan field Dependency; negative atau fractional value ditolak |

Rules:

- Lag hanya dimiliki Executable WBS.
- Grouping WBS tidak mempunyai Lag.
- Lag disimpan sebagai integer.
- Lag menggunakan hari kalender sebagai offset minimum, kemudian scheduler maju ke Date pertama yang mempunyai positive final capacity.
- Lag tidak mengubah Effort.
- Lag tidak membuat dependency relation baru dengan sendirinya.
- Lag tetap tersimpan ketika Automatic Scheduling OFF tetapi tidak mengubah manual timeline.
- Completed Task tetap read-only; Lag tidak dapat diubah melalui generic update setelah complete Actual Date tersedia.

### 6.2 Dependency Ownership

US-5.1 tetap menetapkan satu visible relation untuk endpoint pair yang sama:

```text
Blocking Task ID → Blocked Task ID
```

Concrete implementation harus membedakan ownership relation:

- `manual`
- `automatic`
- keduanya

Storage design boleh menggunakan source flags, ownership records, atau equivalent, selama invariant berikut dipenuhi:

1. Endpoint pair yang sama hanya tampil satu kali pada `Blocks` / `Blocked by`.
2. Recalculation hanya boleh menambah atau menghapus automatic ownership.
3. Recalculation tidak boleh menghapus manual ownership.
4. Manual create terhadap relation automatic yang sama menambahkan manual ownership tanpa duplicate visible relation.
5. Manual delete hanya menghapus manual ownership; relation tetap ada jika automatic ownership masih valid.
6. Relation automatic-only merupakan scheduler-owned projection dan tidak boleh silently dihapus oleh generic manual unlink.
7. Cycle validation tetap berlaku pada effective graph yang terlihat scheduler.

### 6.3 Daily Allocation Projection

Scheduler harus dapat merepresentasikan, secara persisted atau reconstructable deterministic projection:

- Task ID.
- Assignee ID.
- Date.
- Timeline type: `execution` atau `commitment`.
- Allocated Effort.
- Remaining Capacity after allocation.

Daily allocation merupakan scheduler detail. User-facing source of truth tetap date-only Start/End dan Task Effort.

Implementation tidak boleh menggunakan binary floating-point untuk business comparison. Fixed-point decimal, rational arithmetic, atau equivalent deterministic arithmetic harus digunakan.

---

## 7. Activation and Scheduling Triggers

### 7.1 Automatic Scheduling ON

Concrete scheduling dan Auto Dependency aktif hanya jika owning Project mempunyai:

```text
Automatic Scheduling = ON
```

Successful confirmed mutations berikut menjalankan portfolio recalculation sesuai affected scope:

- Create child yang mengonversi Executable WBS existing dan memindahkan executable data atau me-retarget dependency endpoint.
- Sibling reorder.
  - The business trigger remains an adjacent-sibling persisted-position change.
  - Home may disable the UI action while a restrictive Role filter hides
    siblings; this does not change scheduler semantics.
- Move WBS atau subtree.
- Delete Executable WBS.
- Structural conversion.
- Assignee set, change, atau clear.
- Effort change.
- Lag change.
- Manual dependency create.
- Manual dependency delete.
- Project Priority Move Up/Down.
- Automatic Scheduling `OFF → ON`.

Create root atau child yang hanya menyimpan Name, tidak mengonversi Executable WBS existing, dan tidak memindahkan dependency endpoint tidak menjalankan scheduler. Mutation tersebut menyimpan Task baru dengan generated Execution dan Commitment dates kosong. Scheduler baru berjalan ketika mutation berikutnya mengubah scheduling input yang telah menjadi trigger, seperti Assignee, Effort, atau Lag, atau ketika create sekaligus melakukan structural conversion yang memengaruhi scheduling. Keberadaan completed Task pada Project tidak mengubah rule ini.

Delete unfinished leaf juga menggunakan scheduling-impact boundary. Delete tidak menjalankan scheduler jika Task hanya memiliki Name atau Role, Lag tetap default `0`, tidak mempunyai generated Execution/Commitment state atau automatic unscheduled projection, dan tidak menjadi endpoint dependency. Delete tetap menjalankan affected-scope scheduling ketika Task memiliki Assignee, Effort, non-zero Lag, generated state, automatic unscheduled projection, atau dependency endpoint. Sibling position compaction setelah pure structural delete tidak mengubah relative order Task yang tersisa. Keberadaan completed Task lain pada Project tidak mengubah rule ini.

Rename dan Role-only change tetap tidak menjalankan scheduler bila tidak mengubah Assignee.

Complete Actual Date is a scheduling trigger owned by US-6.2. On Open Project it actualizes Execution/Commitment and persists Actual Allocation. On Locked Project it preserves protected baseline but persists Actual Allocation. In both cases, impacted Open Projects may be recalculated. Factual Actual Date remains saveable even when another Locked Project is impacted. Reopen Task remains governed by US-4.2 and is available only while the Project is Open.

### 7.2 Automatic Scheduling OFF

- Auto Dependency tidak dibuat atau diubah oleh Task mutation.
- Lag tidak mengubah manual Execution atau Commitment dates.
- Existing generated dates menjadi initial manual values sesuai US-3.3.
- Existing dependency graph tetap tersedia.
- Ketika project kembali `OFF → ON`, automatic ownership dan timeline dihitung ulang dari confirmed portfolio state.

### 7.3 Capacity and Holiday Mutation

Daily Capacity, Member Buffer, Public Holiday, Project Buffer, and other effective capacity-setting mutations follow US-6.2 generic impact simulation. Capacity Override create/update/delete follows the same guard only when the before/after minimum active override changes Resolved Daily Capacity on at least one Date. Open-only impact requires confirmation; any ordinary Locked impact blocks save; confirmed allowed mutation recalculates only transitive impacted Open Projects.

Pada impact preview dan confirmed recalculation, engine wajib menggunakan
latest confirmed:

- Public Holiday.
- seluruh active Capacity Override dan minimum per Member/Date;
- Daily Capacity;
- Member Buffer;
- Project Buffer.

Capacity Override mutation yang tidak mengubah resolved minimum pada Date mana
pun tidak memanggil scheduler sesuai US-2.1 dan US-6.2.

---

## 8. Scheduling Inputs and Eligibility

### 8.1 Schedulable Task

Task dapat menerima generated dates jika:

- berupa Executable WBS;
- unfinished (no complete Actual Date pair);
- mempunyai Assignee;
- mempunyai valid Effort;
- mempunyai valid Lag;
- owning Project active;
- Automatic Scheduling ON pada owning Project;
- Project Scheduling Start Date tersedia.

Jika satu input wajib tidak tersedia:

- Task tetap dapat disimpan bila owning story mengizinkan;
- generated Execution dan Commitment dates kosong;
- UI menampilkan safe unscheduled reason;
- scheduler tidak mengarang Assignee, Effort, atau Start Date.

Task eligibility dan mutation trigger adalah dua contract berbeda. Create root
atau child pada US-4.1 hanya menerima structural input dan Name, sehingga create
tersebut tidak menjalankan scheduler hanya untuk menghasilkan unscheduled
projection. Scheduler berjalan ketika confirmed Task mutation membuat Task
memenuhi scheduling input, atau ketika scheduling input pada Task yang sudah
memiliki generated state/automatic ownership diubah atau dikosongkan sehingga
stale dates dan ownership harus direconcile. Pada setiap run, scheduler
mengevaluasi complete confirmed Task state, termasuk Project Scheduling Start
Date.

### 8.2 Project Scheduling Start Date

- Project Scheduling Start Date adalah satu-satunya Project-level initial anchor.
- Task tanpa dependency tidak dapat mulai sebelum anchor Project-nya.
- Cross-project predecessor dapat mendorong Task setelah anchor.
- Anchor tidak boleh diganti oleh Task-level Earliest Start atau field baru lain.
- Automatic Scheduling ON tanpa anchor mempertahankan generated dates kosong dan warning baseline:

```text
Automatic Scheduling requires a Project Scheduling Start Date.
```

### 8.3 Project Status

#### Open

- Unfinished eligible Tasks may be recalculated.
- Actual Date actualizes completed Task timelines and Actual Allocation according to US-6.2.
- Recalculation is limited to the transitive impacted scheduling scope.

#### Locked

- Locked Project is never a mutable Execution/Commitment scheduler output.
- Persisted Execution/Commitment dates and allocations are immutable anchors.
- Actual Date may be entered without changing the protected baseline; its Actual Allocation may trigger recalculation of impacted Open Projects.
- Planning changes require explicit Project Reopen to Open.

#### Closed

- Closed Project is excluded from scheduling output and capacity allocation.
- Existing historical dependency may remain readable according to US-5.1.

### 8.4 Completed Task

- Complete Actual Date is the authoritative completion state; Actual End is the dependency-ready Date.
- On Open Project, completed timeline actualization and Actual Allocation follow US-6.2.
- Historical overcapacity is valid, remaining capacity is floored at zero, and excess is not carried to later Dates.
- Completed Task is never rescheduled as unfinished work.
- On Locked Project, Actual Date changes no protected Execution/Commitment dates or baseline allocations; Actual Allocation is recorded separately and affects impacted Open capacity.
- `end before start` caused by Actual Date preceding a former Start must be actualized under US-6.2, not treated as data-integrity conflict.

## 9. Capacity Resolution

Untuk satu Team Member dan satu Date:

### 9.1 Resolved Daily Capacity

Precedence:

1. Weekend default → `0`.
2. Public Holiday → `0`.
3. Jika satu atau lebih Capacity Override aktif untuk Member dan Date tersebut →
   gunakan Capacity terkecil dari seluruh active overrides.
4. Jika tidak ada active override → gunakan Team Member Daily Capacity.

Public Holiday dan weekend tidak dapat dibuat available oleh Capacity Override.
Minimum dipilih hanya antar active overrides; Daily Capacity tidak menjadi cap
ketika override berlaku. Hasil ini merupakan base sebelum Member Buffer dan
Project Buffer.

```text
ResolvedDailyCapacity(member, date)
```

### 9.2 Execution Capacity

```text
ExecutionCapacity(member, date)
= MROUND(
    ResolvedDailyCapacity(member, date)
    × (1 − MemberBufferPercentage / 100),
    0.5
  )
```

Member Buffer diterapkan pada Execution Timeline.

### 9.3 Commitment Capacity

```text
CommitmentCapacity(member, project, date)
= MROUND(
    ResolvedDailyCapacity(member, date)
    × (1 − MemberBufferPercentage / 100)
    × (1 − ProjectBufferPercentage / 100),
    0.5
  )
```

Project Buffer diterapkan setelah Member Buffer.

### 9.4 Precision

- Scheduler menghitung menggunakan deterministic decimal precision.
- Execution Capacity dibulatkan ke kelipatan `0.5` jam setelah Member Buffer.
- Commitment Capacity dibulatkan ke kelipatan `0.5` jam setelah Member Buffer dan Project Buffer diterapkan ke Resolved Daily Capacity.
- Capacity tidak dibulatkan menjadi whole day sebelum allocation.
- Execution dan Commitment output yang dipersist hanya Start Date dan End Date.
- Date ditentukan dari hari pertama dan terakhir yang menerima positive allocation.

### 9.5 Zero Capacity

Date dengan final capacity `<= 0` dilewati.

Jika semua future Dates yang dapat diperiksa tidak mempunyai positive capacity atau Project Buffer menghasilkan Commitment Capacity `0`:

- timeline terkait tidak boleh menghasilkan fabricated End Date;
- result menjadi unscheduled dengan safe reason;
- Execution dapat tetap tersedia walaupun Commitment tidak dapat dihitung.

---

## 10. Priority and Ordering

### 10.1 Dependency before Priority

Manual dan effective dependency selalu lebih kuat daripada priority.

Scheduler tidak boleh menjalankan Task sebelum seluruh blocking conditions satisfied hanya karena Task mempunyai Project Priority lebih tinggi.

### 10.2 Ready-task Priority

Di antara Task yang sudah dependency-ready dan bersaing untuk kapasitas assignee yang sama, urutan adalah:

1. Project Priority ascending; angka lebih kecil berarti lebih tinggi.
2. Visual WBS order dalam Project.

Visual WBS order berarti depth-first order Executable WBS berdasarkan persisted sibling position sebagaimana tampil pada Home. Generated Execution/Commitment Start atau End tidak pernah menjadi sumber row order atau scheduler priority.

Current Project baseline menggunakan unique active Project Priority sehingga cross-project priority tie tidak tersedia. WBS sibling position juga deterministik. Tidak ada additional business tie-breaker seperti Created At, Task ID, atau alphabetical Name.

Jika invalid persisted state menghasilkan duplicate Project Priority atau duplicate WBS position, scheduler harus gagal dengan safe data-integrity error dan tidak mengarang tie-breaker.

### 10.3 Priority-Preserving Concurrent Allocation

US-6.3 supersedes the former exclusive whole-Task/non-preemptive queue.

- Dependency-ready Tasks are still processed by Project Priority then visual WBS order.
- A Task receives at most its Task Daily Limit and the Date's Remaining Timeline Capacity.
- Later ordered Tasks may use the same Date when positive capacity remains.
- Lower-priority work never reduces the allocation a higher-priority Task would receive under its own daily limit.
- Recalculation may displace mutable future lower-priority rows. Completed Actual Allocation, Locked baseline, and fixed manual allocation remain immutable; US-6.3 defines that lower-priority manual allocation may overlap without reducing higher-priority automatic capacity.
- Valid same-assignee overlap caused by Task Capacity Allocation Percentage is not preemption and is not an integrity conflict.

---

## 11. Daily Allocation Rules

### 11.1 Inclusive Date Boundaries

Start dan End bersifat inklusif.

Task yang selesai menggunakan satu hari memiliki:

```text
Start Date = End Date
```

### 11.2 Allocation until Completion

Untuk setiap timeline, US-6.3 adds a per-Task daily ceiling after the final timeline capacity has been independently resolved and rounded:

```text
Remaining Effort = Task Effort

For each eligible Date from calculated readiness:
  Task Daily Limit = US-6.3 percentage limit for this timeline/Date
  Allocated = min(
    Remaining Effort,
    Task Daily Limit,
    Remaining Timeline Capacity on Date
  )
  Remaining Effort -= Allocated
  Remaining Timeline Capacity -= Allocated

Stop when Remaining Effort = 0
```

A Task may receive zero on a Date and continue scanning. Later ordered Tasks may
use capacity left by the current Task's daily limit. Start Date is the first Date
with positive allocation; End Date is the last Date with positive allocation.

### 11.3 Same-assignee Remaining Capacity

Jika predecessor yang menghalangi dan successor mempunyai Assignee yang sama, `Lag = 0`, dan predecessor selesai dengan Remaining Capacity pada End Date:

- successor boleh mulai pada End Date yang sama;
- successor menggunakan Remaining Capacity tersebut;
- dependency tetap Finish-to-Start pada daily capacity boundary, bukan timestamp boundary.

Contoh final Execution Capacity `5` jam per hari dan setiap Task Effort `8` jam:

```text
Task A
Day 1: 5 jam
Day 2: 3 jam
Start Day 1, End Day 2
Remaining Capacity Day 2: 2 jam

Task B
Day 2: 2 jam
Day 3: 5 jam
Day 4: 1 jam
Start Day 2, End Day 4
Remaining Capacity Day 4: 4 jam

Task C
Day 4: 4 jam
Day 5: 4 jam
Start Day 4, End Day 5
Remaining Capacity Day 5: 1 jam
```

### 11.4 Different-assignee Predecessor

Jika predecessor yang menghalangi dan blocked Task mempunyai Assignee berbeda serta `Lag = 0`:

- capacity predecessor tidak dapat dipindahkan ke blocked Task;
- blocked Task paling cepat mulai pada Date setelah predecessor End Date yang mempunyai positive final capacity untuk blocked Task assignee.

### 11.5 Multiple Blockers

- Scheduler mengambil blocker dengan readiness paling akhir sebagai controlling dependency-ready anchor.
- Completed blocker menggunakan Actual End.
- Unfinished blocker menggunakan End Date timeline terkait.
- Jika beberapa blocker mempunyai anchor sama, strictest start rule berlaku: same-day remaining capacity hanya dapat digunakan bila seluruh controlling blockers yang relevan tidak memerlukan next-day boundary dan Task assignee mempunyai remaining capacity.
- Auto-dependency tie rule tidak diperlukan karena satu assignee tidak dialokasikan ke beberapa running Task pada waktu yang sama.

---

## 12. Lag Rules

### 12.1 Task-owned Lag

Lag dimiliki blocked Task, bukan dependency relation.

Satu Lag berlaku satu kali terhadap calculated readiness Task, walaupun Task mempunyai beberapa blockers.

### 12.2 Lag Calculation

For a Task with one or more blockers:

1. Tentukan controlling predecessor planned End atau completed Actual End paling akhir.
2. Tentukan earliest candidate berdasarkan Lag:

```text
candidateDate = controllingEndDate + Lag calendar days
```

3. Apply assignee boundary:
   - `Lag = 0`, same assignee, dan remaining capacity tersedia → candidate dapat tetap pada controlling End Date.
   - `Lag = 0`, different assignee atau tidak ada remaining capacity → mulai mencari setelah controlling End Date.
   - `Lag > 0` → tidak menggunakan same-day capacity pada controlling End Date.
4. Maju ke Date pertama pada atau setelah candidate yang mempunyai positive final capacity dan tidak melanggar assignment queue.

For a Task without blockers:

```text
candidateDate = Project Scheduling Start Date + Lag calendar days
```

Kemudian scheduler mencari Date pertama dengan positive final capacity.

### 12.3 Weekend Example

```text
Predecessor End: Friday
Lag: 1
Initial candidate: Saturday
Saturday capacity: 0
Sunday capacity: 0
Successor earliest Start: Monday
```

Lag tidak dihitung sebagai jumlah full working days yang harus dikosongkan. Lag adalah calendar-day offset minimum, kemudian zero-capacity Dates dilewati.

---

## 13. Auto Dependency by Assignee

### 13.1 Activation

Auto Dependency dijalankan ketika:

- owning Project Automatic Scheduling ON; dan
- Task mempunyai Assignee; dan
- Task unfinished serta schedulable.

Assignee set/change/clear memicu recalculation automatic ownership untuk affected assignee schedules.

### 13.2 Capacity Delay under Parallel Allocation

US-6.3 supersedes the assumption that all same-assignee Tasks form one serial
queue. A Task that starts on its earliest eligible Date by sharing remaining
capacity has no capacity-based auto blocker.

### 13.3 Safe Blocker Selection

A capacity-based auto blocker may exist only when a specific prior same-assignee
Task completion releases the first positive capacity for the delayed Task and a
Finish-to-Start relation reproduces, rather than changes, the confirmed Start.
A prior Task that continues after the delayed Task starts is not a valid blocker.
If several completions qualify, US-6.3 owns the deterministic reverse-order
selection rule.

### 13.4 Priority Displacement with Percentage

Higher-priority ready work may displace mutable lower-priority future allocation.
It does not need to move the lower Task as one contiguous block. Lower-priority
work may remain on Dates where capacity is still available after all higher-
priority daily limits and fixed reservations are applied.

### 13.5 Automatic Relation Reconciliation

Pada recalculation:

1. Load manual graph dan validate acyclic state.
2. Calculate deterministic Execution allocation.
3. Tentukan auto blocker per affected Task.
4. Add/remove automatic ownership agar sesuai allocation.
5. Preserve seluruh manual ownership.
6. Validate effective graph tetap acyclic.
7. Recalculate allocation jika automatic relation materially changes readiness.
8. Commit hanya setelah graph dan kedua timelines konsisten.

Scheduler tidak boleh oscillate. Implementation harus mempunyai deterministic convergence rule dan bounded failure bila consistent graph tidak dapat dibentuk.

### 13.6 Manual Dependency Interaction

- Manual dependency tidak pernah ditimpa oleh Auto Dependency.
- Auto Dependency tidak menggantikan seluruh `Blocked by` list.
- Jika manual blocker sudah mendorong Task lebih lambat daripada assignee queue, tidak perlu membuat auto blocker yang tidak lagi menghalangi earliest start.
- Jika endpoint pair yang sama sudah manual, scheduler dapat menambahkan automatic ownership tanpa duplicate relation.
- User tetap dapat melihat relation dari kedua arah melalui US-5.1 contract.
- Automatic-only relation ditandai `Automatic` pada UI.

### 13.7 Assignee Clear or Change

When Assignee changes:

- automatic ownership yang berasal dari old assignee schedule direconcile;
- manual dependency tetap;
- old dan new assignee schedules menjadi affected;
- affected Open Project dates dihitung ulang;
- stale relation atau stale dates tidak boleh tertinggal.

When Assignee becomes empty:

- automatic ownership untuk Task tersebut dihapus bila tidak mempunyai manual ownership;
- generated dates Task menjadi empty/unscheduled;
- downstream tasks direcalculate berdasarkan remaining effective dependencies.

---

## 14. Execution Scheduling Algorithm

Execution scheduler harus:

1. Load seluruh Active Portfolio input yang diperlukan.
2. Treat completed Actual Allocation and Locked allocations as absolute reservations; treat fixed manual allocation as immutable with priority-aware capacity precedence from US-6.3.
3. Exclude Closed Projects from mutable planned scheduling.
4. Exclude completed Tasks from unfinished recalculation while retaining Actual End readiness anchors and Actual Allocation capacity consumption under revised US-6.2.
5. Resolve manual dependency graph and reject cycle.
6. Build ready set using Project anchors, dependency readiness, and Lag.
7. For each Assignee, order ready Tasks by Project Priority then visual WBS order.
8. Resolve each Task Daily Limit from rounded Execution Capacity and US-6.3 percentage.
9. Allocate each ordered Task across eligible Dates up to Remaining Effort, Task Daily Limit, and Remaining Execution Capacity; allow later ordered Tasks to use leftover capacity on the same Date.
10. Reconcile only safe automatic dependency ownership under US-6.3.
11. Repeat deterministically until graph/allocation stable.
12. Persist Execution Start/End and daily allocation projection atomically.

Execution scheduler must not:

- schedule Grouping WBS;
- schedule Task before Project anchor or dependency/Lag readiness;
- exceed final Execution Capacity with mutable automatic rows;
- let lower-priority work reduce a higher-priority Task below its own daily limit;
- apply percentage before final timeline capacity rounding;
- round Task Effort into whole days;
- shift completed Tasks, Locked baselines, or fixed manual allocation; lower-priority manual rows may overlap higher-priority automatic rows under US-6.3;
- invent a serial auto dependency for valid parallel allocation.

---

## 15. Commitment Scheduling Algorithm

Commitment scheduler uses:

- the same Task set;
- the same effective dependency graph;
- the same Project Priority and WBS order;
- the same priority-preserving concurrent allocation and Task Daily Limit rules from US-6.3;
- Commitment Capacity, calculated and rounded independently from the raw resolved capacity.

Commitment is recalculated independently. It is not calculated by adding a fixed number of days to Execution End.

For each Task:

```text
Commitment Capacity
= Resolved Daily Capacity
  × (1 − Member Buffer)
  × (1 − Project Buffer)
```

Commitment Start/End are determined from the first and last positive Commitment allocation Date.

Expected invariant:

- Commitment timeline normally ends on or after Execution timeline because Commitment Capacity is never greater than Execution Capacity for valid non-negative Project Buffer.
- Implementation must not hard-code this by copying or clamping dates; it must result from allocation.

Locked Commitment baseline remains immutable and acts as fixed reservation where required by portfolio capacity planning.

---

## 16. Transaction and Failure Behaviour

### 16.1 Atomic Confirmed Mutation

US-6.2 adds these mandatory distinctions:

- valid inability to schedule is persisted as `Unscheduled + Reason`, not returned as scheduler failure;
- historical completed overcapacity is valid and is not repaired or rejected;
- Locked→Open Reopen commits status plus transitive impacted-scope recalculation atomically;
- technical/integrity/concurrency failure rolls Reopen back to Locked;
- unrelated Projects outside the impacted scope are neither recalculated nor version-updated;
- priority mutation is rejected atomically when any Locked Project would be affected.


For a mutation that requires scheduling, confirmed operation must be atomic across:

- owning Task/WBS change;
- manual dependency mutation when applicable;
- automatic dependency ownership reconciliation;
- Execution Timeline update;
- Commitment Timeline update;
- affected allocation projection;
- required version/invalidation metadata.

If scheduling or persistence fails:

- initiating mutation is rolled back;
- previous dependency graph remains;
- previous generated dates remain;
- no partial allocation remains;
- UI retains draft where applicable;
- safe retry is available.

### 16.2 Concurrency

Concurrent mutations affecting shared assignee, dependency graph, Project Priority, or overlapping portfolio schedule must not produce:

- duplicate automatic relations;
- two overlapping running Tasks for one assignee;
- stale dates committed over newer dates;
- partial graph/timeline state;
- duplicate WBS/project positions.

Implementation may serialize, use optimistic version checks, or equivalent. Conflict must be deterministic and safe.

### 16.3 Stale Response Protection

Successful scheduling invalidates or version-updates:

- affected Home WBS hierarchy and action eligibility.
- Task detail.
- Blocks / Blocked by.
- candidate lookup.
- Project list derived dates.
- Execution and Commitment projections.
- US-7.1 Home Portfolio Gantt confirmed read projection and invalidation.

Older in-flight response must not restore previous Assignee, dependency, dates, or allocation as confirmed state.

---

## 17. UI Behaviour

### 17.1 Task Form

Contextual Edit Task form adds labeled field:

```text
Lag (days)
```

- Default Add value: `0`.
- Accepts whole number `0` or greater.
- Fraction, negative value, comma, or non-numeric input is rejected.
- Draft is retained after recoverable failure.
- Automatic Scheduling ON: Execution and Commitment dates are read-only outputs.
- Automatic Scheduling OFF: manual timeline remains editable under baseline rules; Lag is stored but inactive.
- For an Open unfinished Task with Automatic Scheduling ON, leaving Role,
  Assignee, Effort, or Lag triggers draft schedule validation.
- When Role, valid Effort, and valid Lag are present, the UI requests a
  non-persistent schedule preview using the current draft before Save.
- Assignee may be selected or explicitly cleared. A selected Assignee previews
  the reconciled old/new assignee queues and generated dates. A cleared
  Assignee still requests preview so stale automatic ownership is removed,
  downstream work is recalculated, and the Task returns a missing-Assignee
  unscheduled projection.
- The preview uses the concrete portfolio scheduler, including capacity,
  dependency, Priority, WBS order, and Lag rules. It returns both generated
  Task dates and the Task's resulting manual/automatic/shared dependency
  projection, then rolls back every temporary Task, dependency-ownership,
  allocation, Project-date, and schedule-version write.
- Previewed dependencies appear in the existing `Blocked by` / `Blocks`
  sections as unconfirmed, read-only projection data. Save remains required
  before automatic dependency ownership becomes persisted or actionable.
- Missing or invalid Role, Effort, or Lag does not call the preview API and
  displays which required scheduling inputs must be completed. An explicitly
  cleared Assignee is not treated as an incomplete preview draft.
- Name changes do not trigger schedule preview because Name does not affect
  scheduling.
- Save remains the only action that confirms Task-field changes. A preview must
  never silently auto-save the draft.
- A newer draft or confirmed mutation invalidates an older in-flight preview;
  stale preview responses cannot replace the latest visible draft schedule.

### 17.2 Auto Dependency Display

- Automatic-only relation appears in existing `Blocked by` / `Blocks` sections.
- Relation displays `Automatic` text/badge; status must not rely only on colour.
- Manual+automatic endpoint pair appears once and remains understandable.
- Automatic-only relation does not expose a misleading normal Delete action while scheduler still requires it.
- Manual relation controls remain as defined in US-5.1.

### 17.3 Mutation State

When Task Save or dependency mutation triggers scheduling:

- related Save/action is pending and duplicate submission disabled;
- unrelated page navigation is not globally blocked unless unsafe;
- success updates dependency and dates without hard refresh;
- failure shows safe error and preserves confirmed previous timeline plus user draft;
- raw scheduler, SQL, stack, or infrastructure detail is never displayed.

When a Task draft schedule preview is pending:

- generated dates and dependency ownership in the preview are identified as unconfirmed;
- the user may continue editing, and the superseded request is cancelled or
  ignored by resolution order;
- Save aborts an in-flight preview and executes the normal confirmed mutation;
- preview failure preserves both the current draft and the last visible
  confirmed or successfully previewed schedule.

### 17.4 Unscheduled State

UI must distinguish at least:

- missing Project Scheduling Start Date;
- missing Assignee;
- missing/invalid Effort;
- no positive Execution Capacity;
- no positive Commitment Capacity;
- dependency/cycle conflict;
- scheduling operation failure.

`Not scheduled` must not be presented as a confirmed valid date.

---

## 18. API and Error Contract

Implementation must extend established Task and Dependency APIs rather than create duplicate aggregate ownership.

Draft schedule preview uses:

```text
POST /api/projects/{projectId}/wbs/{wbsId}/executable/preview
```

The request contains only scheduling-affecting draft fields: `roleId`,
`assigneeId`, `effortHours`, and `lag`. `assigneeId` may be omitted or `null`
when previewing an Assignee clear; the scheduler must still reconcile automatic
ownership and return the Task's missing-Assignee unscheduled projection. The
response wraps the established WBS Task projection together with the Task's
resulting dependency projection:
`{ task, dependencies: { blockedBy, blocks } }`. Preview dependency rows use
the same source and Task projection contract as the confirmed dependency API.
The endpoint must not persist the draft or advance confirmed cache/version state.

Minimum Task response fields where relevant:

```json
{
  "lag": 0,
  "executionStart": "2026-08-03",
  "executionEnd": "2026-08-05",
  "commitmentStart": "2026-08-03",
  "commitmentEnd": "2026-08-06"
}
```

Dependency response must expose enough projection data for UI to distinguish automatic ownership without duplicating endpoint relation.

Minimum stable errors:

| Error Code                           |                                     HTTP | Condition                                                     |
| ------------------------------------ | ---------------------------------------: | ------------------------------------------------------------- |
| `INVALID_LAG`                        |                                      400 | Lag negative, fractional, malformed, or unsupported           |
| `SCHEDULING_INPUT_INCOMPLETE`        |                                      400 | Required scheduling input missing                             |
| `SCHEDULING_CYCLE`                   |                                      409 | Effective manual/automatic graph cyclic; path returned safely |
| `SCHEDULING_NO_EXECUTION_CAPACITY`   |            409 or unscheduled projection | No positive Execution Capacity                                |
| `SCHEDULING_NO_COMMITMENT_CAPACITY`  |            409 or unscheduled projection | No positive Commitment Capacity                               |
| `SCHEDULING_DATA_INTEGRITY_CONFLICT` |                                      409 | Duplicate priority/order or invalid fixed reservation state   |
| `SCHEDULING_CONFLICT`                |                                      409 | Concurrent confirmed state changed                            |
| `SCHEDULING_FAILED`                  | 500 or established safe operation status | Atomic scheduling/persistence failed                          |
| `PROJECT_CLOSED_READ_ONLY`           |                                      409 | Mutation targets Closed Project                               |
| `COMPLETED_TASK_READ_ONLY`           |                                      409 | Generic mutation targets completed Task                       |
| `SCHEDULE_PREVIEW_UNAVAILABLE`       |                                      409 | Preview targets non-Open, completed, or Automatic Scheduling OFF state |
| `INVALID_REQUEST`                    |                                      400 | Malformed JSON or unknown fields                              |

Error response must not expose implementation internals.

---

## 19. Acceptance Criteria

### AC-1 — Automatic Scheduling activation

**Given** Project Automatic Scheduling ON
**When** an established scheduling trigger succeeds
**Then** concrete Execution and Commitment scheduling runs.

**Given** Automatic Scheduling OFF
**Then** Task mutation does not regenerate automatic dependency or overwrite manual timelines.

### AC-2 — Lag field and default

**When** Add Task form opens
**Then** `Lag (days)` defaults to `0`.

**When** valid integer Lag is saved
**Then** value persists on Executable WBS.

### AC-3 — Lag validation

**When** Lag is negative, fractional, malformed, or non-numeric
**Then** frontend and backend reject it with `INVALID_LAG`
**And** confirmed Task and dates remain unchanged.

### AC-4 — Scheduling anchor required

**Given** Automatic Scheduling ON without Project Scheduling Start Date
**When** scheduler is triggered
**Then** generated Execution and Commitment dates remain empty
**And** approved anchor warning is shown
**And** no Task-level anchor is invented.

### AC-5 — Capacity precedence

**Given** a Date is weekend or Public Holiday
**Then** Resolved Daily Capacity is `0` despite Capacity Override.

**Given** a working Date is not Public Holiday
**And** one or more Capacity Overrides are active for the Member
**Then** the smallest override Capacity is used as Resolved Daily Capacity
**And** Daily Capacity is not compared as a cap.

**Given** no Capacity Override is active
**Then** Team Member Daily Capacity is used.

### AC-6 — Execution Capacity formula

**Given** Resolved Daily Capacity and Member Buffer
**When** Execution allocation is calculated
**Then** `Raw Execution Capacity = Resolved Daily Capacity × (1 − Member Buffer)`
**And** `Execution Capacity = MROUND(Raw Execution Capacity, 0.5)`
**And** Member Buffer is not deferred only to Commitment.

### AC-7 — Commitment Capacity formula

**Given** Resolved Daily Capacity, Member Buffer, and Project Buffer
**When** Commitment allocation is calculated
**Then** `Raw Commitment Capacity = Resolved Daily Capacity × (1 − Member Buffer) × (1 − Project Buffer)`
**And** `Commitment Capacity = MROUND(Raw Commitment Capacity, 0.5)`.

For `8`, `30%`, and `20%`, Raw Commitment is `4.48` and rounded Commitment is `4.5` hours/day.

### AC-8 — Half-hour Execution rounding before allocation

**Then** scheduler rounds Execution Capacity to the nearest `0.5` hour after Member Buffer
**And** independently rounds Commitment Capacity to the nearest `0.5` hour after Member Buffer and Project Buffer
**And** does not round capacity to whole days
**And** uses deterministic decimal arithmetic
**And** persists date-only Start/End outputs.

### AC-9 — Inclusive one-day Task

**Given** Effort fits within available capacity on one Date
**When** Task is scheduled
**Then** Start Date equals End Date.

### AC-10 — Multi-day daily allocation

**Given** Effort exceeds capacity on the first Date
**When** Task is scheduled
**Then** scheduler consumes per-Date capacity until Effort reaches zero
**And** End Date is the final allocation Date.

### AC-11 — Same-assignee same-day continuation

**Given** predecessor and successor have the same Assignee, Lag `0`, and remaining capacity exists on predecessor End Date
**When** successor is scheduled
**Then** successor may start on that same Date using remaining capacity.

### AC-12 — Different-assignee next available Date

**Given** predecessor and successor have different Assignees and Lag `0`
**When** predecessor ends
**Then** successor starts no earlier than the next Date with positive capacity for successor Assignee.

### AC-13 — Lag skips zero-capacity Dates

**Given** predecessor ends Friday and blocked Task Lag is `1`
**When** Saturday and Sunday have zero capacity
**Then** earliest Start is Monday.

### AC-14 — Lag without dependency

**Given** Task has no blocker
**When** Lag is greater than `0`
**Then** Lag offsets Project Scheduling Start Date before capacity availability is applied.

### AC-15 — Dependency overrides priority

**Given** higher-priority Task is blocked by lower-priority Task
**When** scheduler runs
**Then** higher-priority Task does not start before dependency is satisfied.

### AC-16 — Ready-task ordering

**Given** multiple ready Tasks compete for one Assignee
**When** scheduler selects next Task
**Then** lower Project Priority number is selected first
**And** within one Project, visual WBS order is used.

### AC-17 — No invented tie-breaker

**Given** valid baseline data
**Then** unique Project Priority and deterministic WBS position provide total order.

**Given** duplicate invalid ordering data
**Then** scheduling fails safely instead of using Task ID, Name, or Created At as an invented tie-breaker.

### AC-18 — Concurrent Task Daily Limits

**Given** multiple ready Tasks share one Assignee
**Then** each Task is capped by its own US-6.3 Task Daily Limit
**And** later ordered Tasks may use positive remaining capacity on the same Date.

### AC-19 — Higher-priority displacement

**Given** higher-priority ready Task C is introduced before mutable lower-priority Task B
**When** schedule is recalculated
**Then** C receives capacity first up to its daily limit
**And** B is moved only where required
**And** fixed/completed/Locked allocations remain unchanged.

### AC-20 — Lower-priority Task uses residual capacity only

**Given** higher-priority Task B has positive allocation on a Date
**Then** lower-priority Task C may use only the remaining capacity after B and fixed reservations
**And** C cannot reduce B below B's own daily-limit allocation.

### AC-21 — Safe automatic blocker

**Given** a Task's Start is delayed by exhausted same-assignee capacity
**When** one prior Task completion releases the first positive capacity
**And** Finish-to-Start ownership reproduces the confirmed Start
**Then** that Task may become the single automatic blocker.

**When** candidate blocker continues after the delayed Task starts
**Then** it is not selected.

### AC-22 — At most one auto dependency

**When** scheduler reconciles an unfinished assigned Task
**Then** Task has at most one automatic blocker
**And** valid same-day parallel allocation creates no serial ownership
**And** manual ownership remains preserved.

### AC-23 — Manual dependency preservation

**Given** Task has manual dependency
**When** Assignee, priority, WBS order, or schedule changes
**Then** recalculation does not delete or retarget manual ownership.

### AC-24 — Shared endpoint ownership

**Given** one endpoint pair is both manual and automatically required
**Then** UI displays one relation
**And** manual ownership can be removed without deleting still-required automatic ownership
**And** automatic reconciliation can be removed without deleting manual ownership.

### AC-25 — Assignee change triggers auto dependency and dates

**Given** Automatic Scheduling ON
**When** Assignee is set or changed and Task Save succeeds
**Then** old and new assignee schedules are reconciled
**And** automatic dependency is updated
**And** Execution and Commitment dates are updated without hard refresh.

### AC-26 — Assignee clear

**When** Assignee is cleared
**Then** automatic ownership specific to assignment is removed when not manually owned
**And** Task generated dates become unscheduled
**And** downstream affected schedules are recalculated.

### AC-27 — Dependency mutation triggers scheduling

**Given** Automatic Scheduling ON
**When** manual dependency is created or deleted successfully
**Then** effective graph and transitive impacted-scope dates are recalculated atomically.

### AC-28 — Lag and Effort mutation trigger scheduling

**Given** Automatic Scheduling ON
**When** Lag or Effort changes successfully
**Then** affected automatic dependency and both timelines are recalculated.

### AC-29 — WBS create and structural triggers

**Given** Automatic Scheduling ON
**When** root atau child Task dibuat hanya dengan Name tanpa structural conversion
**Then** Task berhasil dibuat
**And** concrete portfolio scheduler tidak dipanggil
**And** generated Execution dan Commitment dates Task baru tetap kosong
**And** existing Task, dependency, allocation, dan timeline tidak berubah.

**When** create child mengonversi Executable WBS existing dan memindahkan executable data atau dependency endpoint, atau established reorder/move/conversion atau Project Priority Move berhasil
**Then** concrete portfolio scheduling tetap berjalan sesuai affected scope.

**When** unfinished Name-only atau Role-only leaf tanpa dependency dihapus
**Then** delete berhasil tanpa concrete portfolio scheduler
**And** remaining Task, dependency, allocation, dan timeline state tidak berubah.

**When** deleted Task mempunyai Assignee, Effort, non-zero Lag, generated projection, automatic unscheduled projection, atau dependency endpoint
**Then** concrete portfolio scheduling tetap berjalan sesuai affected scope
**And** scheduler failure me-rollback delete secara atomik.

### AC-30 — Completed Task behaviour

**Given** complete Actual Date is entered on an Open unfinished Task
**Then** both timeline Ends become Actual End
**And** each Start becomes the earlier of existing Start and Actual Start
**And** completed Effort remains historical capacity consumption
**And** overcapacity is valid and not carried to the next Date.

### AC-31 — Locked Project behaviour

**Given** Project Locked
**Then** Execution/Commitment dates and allocations are immutable
**And** scheduler never mutates the Locked Project for planning or Actual Date mutation
**And** Actual Date may be entered without changing baseline
**And** Actual Allocation may recalculate impacted Open Projects
**And** every other planning/dependency/Task mutation is rejected until explicit Project Reopen.

### AC-32 — Transitive impacted-scope recalculation

**Given** Project A mutation/reopen affects B and B affects C
**When** recalculation runs
**Then** A, B, and C are recalculated where Open
**And** unrelated Project D is not recalculated or version-updated
**And** Locked Projects in the scope remain immutable anchors.

### AC-32A — Priority change with Locked Projects

**When** Project Priority changes while Locked Projects exist
**Then** simulation proves no Locked timeline, allocation, or dependency-validity impact before Save
**And** any Locked impact rejects the entire operation atomically.

### AC-33 — Independent Commitment schedule

**When** Commitment Timeline is calculated
**Then** it uses the same graph/order with Commitment Capacity
**And** is not produced by adding fixed days to Execution End.

### AC-34 — Zero-capacity unscheduled result

**Given** no positive Execution or Commitment Capacity is available
**When** scheduler runs
**Then** it does not fabricate dates
**And** UI exposes the relevant unscheduled reason.

### AC-35 — Atomic rollback

**Given** Task/dependency mutation requires scheduling
**When** graph reconciliation, allocation, or persistence fails
**Then** initiating mutation, automatic ownership, and both timelines roll back
**And** previous confirmed state remains.

### AC-36 — Concurrent consistency

**When** concurrent changes affect the same assignee, graph, or priority order
**Then** at most one consistent confirmed schedule version wins
**And** no overlap, duplicate auto relation, or stale overwrite is persisted.

### AC-37 — Cache and stale-response protection

**When** recalculation succeeds while older requests are in flight
**Then** old Task, dependency, or timeline response cannot restore stale confirmed state.

### AC-38 — UI feedback and duplicate prevention

**When** scheduling mutation is pending
**Then** duplicate action is prevented
**And** success/failure feedback is safe and actionable
**And** draft is retained after recoverable failure.

**When** Role, Assignee, Effort, or Lag changes on an Open unfinished Task with Automatic Scheduling ON
**And** the edited field loses focus
**Then** the form checks whether Role, valid Effort, and valid Lag are present
**And** a draft with a selected Assignee requests a concrete non-persistent schedule preview before Save and reconciles the old/new assignee queues
**And** an explicitly cleared Assignee still requests preview, removes stale automatic ownership, recalculates affected downstream work, and returns a missing-Assignee unscheduled projection
**And** missing or invalid Role, Effort, or Lag sends no preview request and exposes the missing-input state
**And** previewed generated dates and automatic dependency ownership are labelled as unconfirmed
**And** Save remains the only action that persists the Task draft
**And** stale preview responses cannot overwrite a newer draft.

### AC-39 — Accessibility and responsive behaviour

**Then** Lag, Automatic badges, read-only dates, pending/error/unscheduled states, and actions are keyboard accessible
**And** status is not communicated only by colour
**And** workflow remains usable on supported viewports without uncontrolled normal-page horizontal scrolling.

---

## 20. Test Cases

| ID    | Scenario                                          | Expected Result                                                   |
| ----- | ------------------------------------------------- | ----------------------------------------------------------------- |
| TC-1  | Automatic Scheduling ON and Name-only Task create | Task persists; scheduler is not invoked; generated dates stay empty |
| TC-1A | Open completion with Actual Start earlier/later than planned | Starts use earlier date; Ends use Actual End; no integrity conflict |
| TC-1B | Actual Effort exceeds working-date capacity | Excess distributed per US-6.2; next Date has no debt carry-over |
| TC-1C | Actual Date on Locked Task | Actual Date/Actual Allocation saved; baseline unchanged; impacted Open scope recalculated |
| TC-1D | Locked→Open A impacts B impacts C | A/B/C recalculated; independent D untouched |
| TC-1E | Priority would invalidate Locked successor | Entire priority change rejected |
| TC-2  | Automatic Scheduling OFF and Task edit            | Manual dates unchanged; no auto dependency regeneration           |
| TC-3  | Add Task form                                     | Lag defaults `0`                                                  |
| TC-4  | Valid Lag save                                    | Integer persists                                                  |
| TC-5  | Negative/fractional/malformed Lag                 | Frontend/backend reject; no mutation                              |
| TC-6  | Missing Scheduling Start Date                     | Dates empty; approved warning                                     |
| TC-7  | Weekend with override                             | Capacity remains `0`                                              |
| TC-8  | Public Holiday with override                      | Capacity remains `0`                                              |
| TC-9  | Working Date with multiple overrides `12`, `6`, `0` | Minimum `0` used as pre-buffer base                           |
| TC-9A | Working Date with override `12`, Daily Capacity `8` | Override `12` used; Daily Capacity is not a cap                 |
| TC-10 | Working Date without override                     | Daily Capacity used                                               |
| TC-11 | Capacity `8`, Member Buffer `30%`                 | Raw `5.6`; rounded Execution Capacity `5.5`                        |
| TC-12 | Capacity `8`, Member Buffer `30%`, Project Buffer `20%` | Raw Commitment `4.48`; rounded Commitment Capacity `4.5`      |
| TC-13 | Fractional raw Execution capacity                  | Rounded once to nearest `0.5`; no whole-day rounding               |
| TC-14 | Effort fits one Date                              | Start = End                                                       |
| TC-15 | Effort spans Dates                                | First/last positive allocation define dates                       |
| TC-16 | A/B/C with percentage limits                     | Same-assignee Tasks may overlap; first/last positive rows define each timeline |
| TC-17 | Same assignee, Lag 0, remaining capacity          | Successor starts predecessor End Date                             |
| TC-18 | Different assignee, Lag 0                         | Successor starts next positive-capacity Date                      |
| TC-19 | Friday End, Lag 1                                 | Monday Start when weekend zero                                    |
| TC-20 | No blocker, Lag 1                                 | Anchor shifted one calendar day then availability applied         |
| TC-21 | High-priority Task blocked by lower-priority Task | Dependency respected                                              |
| TC-22 | Two ready Projects                                | Lower Project Priority number first                               |
| TC-23 | Ready Tasks in same Project                       | Visual WBS order first                                            |
| TC-24 | Duplicate ordering fixture                        | Safe data-integrity failure; no invented tie-break                |
| TC-25 | Higher Task leaves capacity under its percentage    | Later ordered Task uses same-Date residual capacity                |
| TC-26 | Higher-priority C before B                        | C receives capacity first; mutable B shifts only where required     |
| TC-27 | Lower-priority C shares residual capacity          | C uses only residual capacity and may overlap B                    |
| TC-28 | C starts on earliest eligible Date using residual | No capacity-based auto blocker                                     |
| TC-29 | C is delayed until B completion releases capacity | B is selected only when safe blocker conditions hold               |
| TC-30 | First Task for assignee                           | No auto blocker if only anchor constrains                         |
| TC-31 | Manual blocker determines later date              | No irrelevant auto blocker added                                  |
| TC-32 | Manual relation survives reassignment             | Manual ownership retained                                         |
| TC-33 | Relation both manual and automatic                | One visible relation; source ownership preserved                  |
| TC-34 | Remove manual ownership from dual relation        | Relation remains automatic                                        |
| TC-35 | Auto blocker no longer needed                     | Automatic ownership removed; manual ownership retained if present |
| TC-36 | Assignee changes A→B                              | Both queues and dates reconciled                                  |
| TC-37 | Assignee cleared                                  | Auto ownership removed; Task unscheduled                          |
| TC-38 | Dependency create                                 | Graph and dates updated atomically                                |
| TC-39 | Dependency delete                                 | Graph and dates updated atomically                                |
| TC-40 | Lag change                                        | Dates and auto blocker recalculated                               |
| TC-41 | Effort change                                     | Daily allocation and downstream dates recalculated                |
| TC-42 | WBS reorder                                       | Priority and affected schedule updated                            |
| TC-43 | Project Priority move                             | Whole affected active portfolio recalculated                      |
| TC-44 | Completed blocker                                 | Actual End anchors planned successor; historical Actual ranges may overlap |
| TC-45 | Completed Task scheduler run                      | Completed dates unchanged; no future capacity                     |
| TC-46 | Locked Project                                    | Baselines unchanged and reserved                                  |
| TC-47 | Closed Project                                    | Excluded from allocation                                          |
| TC-48 | Cross-project dependency                          | Both affected Projects updated                                    |
| TC-49 | Shared assignee across Projects                   | Mutable automatic total remains within capacity; percentage overlap is valid |
| TC-50 | Commitment schedule                               | Independent allocation with lower capacity                        |
| TC-51 | Project Buffer 100%                               | Execution may exist; Commitment unscheduled, no fabricated date   |
| TC-52 | Scheduler failure after Task mutation starts      | Full rollback                                                     |
| TC-53 | Graph reconciliation failure                      | No partial auto relation or dates                                 |
| TC-54 | Concurrent assignee updates                       | One consistent version; deterministic conflict                    |
| TC-55 | Concurrent dependency/priority updates            | No cycle, overlap, or stale overwrite                             |
| TC-56 | Older response after success                      | Stale state cannot return                                         |
| TC-57 | Double Save/Delete                                | One mutation                                                      |
| TC-58 | Recoverable frontend failure                      | Draft retained; retry works                                       |
| TC-59 | Keyboard workflow                                 | Lag, dependencies, Save, feedback accessible                      |
| TC-60 | Supported narrow viewport                         | Workflow usable without uncontrolled page scroll                  |
| TC-61 | Effort edit followed by blur                      | Generated dates and automatic dependency projection appear before Save |
| TC-62 | Role, Effort, or Lag cleared/invalid then blurred | No preview API call; missing-input state replaces stale draft dates |
| TC-63 | Two preview requests resolve out of order         | Only the newest draft preview remains visible                      |
| TC-64 | Save while preview is in flight                   | Preview is aborted; one confirmed mutation persists and schedules  |
| TC-65 | Assignee explicitly cleared then blurred          | Preview API runs; stale automatic ownership is removed; Task shows missing-Assignee unscheduled projection |
| TC-66 | Create child converts executable parent or retargets dependency | Concrete portfolio scheduler runs atomically; failure rolls back conversion |
| TC-67 | Delete Name-only unfinished Task while Project also has completed Task | Task is deleted; scheduler is not invoked; completed and remaining Task state is unchanged |
| TC-68 | Delete Task with scheduling state | Affected portfolio scheduler runs atomically; failure rolls back deletion |

---

## 21. Required Automated Tests by Layer

### 21.1 Domain Tests

- Lag default and validation.
- Capacity formulas.
- Date-only inclusive boundary.
- Deterministic decimal allocation.
- Same-assignee remaining-capacity rule.
- Different-assignee next-date rule.
- Lag calendar offset and zero-capacity skip.
- Priority comparator.
- Non-preemption invariant.
- Fit-based slot selection.
- Auto blocker selection.
- Dependency ownership merge/remove semantics.
- Completed and Project-status invariants.

### 21.2 Application Tests

- Name-only root/child create skips scheduler and preserves empty generated dates.
- Structural-conversion create remains an established scheduler trigger and rolls back on failure.
- Every other established scheduler trigger.
- Automatic Scheduling ON/OFF branching.
- Portfolio affected-scope calculation.
- Assignee old/new queue reconciliation.
- Manual dependency preservation.
- Auto dependency convergence.
- Execution then Commitment orchestration.
- Locked reservation handling.
- Atomic rollback on each coordination failure.
- Invalidation/version metadata.
- Concurrency conflict mapping.

### 21.3 Repository Integration Tests

- Lag persistence.
- Automatic/manual ownership persistence without duplicate visible endpoints.
- Daily allocation persistence or deterministic reconstruction.
- Date-only persistence.
- Decimal precision preservation.
- Query/index support for portfolio tasks, assignee queue, dependency graph, capacity ranges, Project Priority, and WBS order.
- Transaction rollback.
- Concurrent automatic relation uniqueness.
- Concurrent schedule-version protection.
- PostgreSQL query-plan review for portfolio scheduling inputs.

### 21.4 API Integration Tests

- Task create/update with Lag.
- Invalid Lag and unknown fields.
- Generated dates in response.
- Dependency origin/source projection.
- Assignee, Effort, Lag, WBS, Project Priority, and dependency triggers.
- ON/OFF behaviour.
- Missing anchor and unscheduled reasons.
- Structured cycle/capacity/concurrency/failure errors.
- Atomic confirmed response.
- Completed, Locked, and Closed guards.

### 21.5 Frontend Integration Tests

- Lag field default, validation, and draft retention.
- Dates read-only in automatic mode.
- Manual dates retained in manual mode.
- Automatic badge.
- Manual+automatic relation displayed once.
- Save pending and duplicate prevention.
- Success refresh without hard reload.
- Failure retains previous confirmed dates and draft.
- Unscheduled reason states.
- Cache invalidation and stale-response protection.
- Assignee change and explicit clear both request preview; clear returns a
  missing-Assignee projection instead of being blocked as incomplete input.
- Keyboard, focus, accessible names, non-colour status, and responsive behaviour.

### 21.6 Acceptance-Level Tests

Acceptance-level tests must exercise the highest practical user-facing feature boundary available, primarily Home → Task Name → shared Edit Task plus real application orchestration. Reorder/move acceptance starts from Home row overflow, and no acceptance workflow may depend on the removed Project Structure page.

Minimum scenarios:

1. Select Assignee and save; one relevant auto dependency plus dates appear.
2. Fit Task between A and B; A is auto blocker.
3. Task does not fit; B is auto blocker.
4. Insert higher-priority Project/WBS Task and observe lower Task shift without preemption.
5. Same-assignee predecessor shares End Date remaining capacity.
6. Different-assignee predecessor begins next available Date.
7. Change Lag from `0` to `1` across a weekend.
8. Verify Raw Execution `5.6`, rounded Execution `5.5`, Raw Commitment `4.48`, and rounded Commitment `4.5` through resulting dates/allocation fixture.
9. Preserve manual dependency when Assignee changes.
10. Automatic Scheduling OFF preserves manual dates and does not regenerate auto relation.
11. Scheduler failure rolls back Task, relation, and dates.
12. Shared assignee across automatic Projects stays capacity-safe by Project Priority; manual-versus-automatic overlap follows US-6.3 priority-aware immutable allocation rules.
13. Locked baseline does not move.
14. Stale response cannot restore old dependency/dates.
15. Clear Assignee and blur; preview removes stale automatic ownership, shows
    missing-Assignee unscheduled state, and persists nothing before Save.
16. Keyboard-only Task save and dependency review.

An isolated comparator, domain allocation test, gateway mock, or snapshot alone is not acceptance-level evidence.

---

## 22. Three-Level Confidence Matrix

### 22.1 Mandatory Rule

An Acceptance Criterion is incomplete until all three evidence levels are present:

1. **Code Inspection** — concrete production path and invariant inspected.
2. **Unit/Integration Test** — automated technical evidence.
3. **Acceptance-Level Test** — automated observable workflow evidence at the highest practical boundary.

Rules:

- Code inspection alone is insufficient.
- Technical tests alone are insufficient.
- Acceptance test without inspecting the real production path is insufficient.
- Snapshot-only evidence is insufficient.
- Manual smoke may supplement but never replace automation.
- One test may support several ACs only when assertions are explicit and traceable.
- Agent must stop completion reporting at the first AC missing one level.
- Completion report must name repository paths, symbols, test names, commands, and exact results.

### 22.2 Required Evidence per AC

The concrete implementation paths, exact authored test names, local commands,
and non-executed readiness statuses for every AC are recorded in
`docs/project/automatic-scheduling-implementation-evidence.md`. The table below
remains the required evidence shape; the linked evidence document is the
implementation-specific record.

| AC       | Code Inspection                                                        | Unit/Integration Test                              | Acceptance-Level Test                                                             |
| -------- | ---------------------------------------------------------------------- | -------------------------------------------------- | --------------------------------------------------------------------------------- |
| AC-1     | Automatic-mode orchestration branch and concrete scheduler composition | ON/OFF application contract tests                  | Save Task in ON and OFF projects; observe generated versus preserved manual dates |
| AC-2–3   | Task Lag domain field, DTO, validation, form                           | Domain/API/frontend validation tests               | Add/Edit Task with default, valid, and invalid Lag                                |
| AC-4     | Project anchor guard before allocation                                 | Application/API missing-anchor test                | Trigger schedule without anchor and observe warning/empty dates                   |
| AC-5     | Capacity resolver precedence                                           | Resolver integration with weekend/holiday/override | Schedule fixture across weekend, holiday, and override                            |
| AC-6–8   | Execution and Commitment formulas, independent final half-hour rounding, and deterministic arithmetic | Formula/rounding unit tests | Capacity fixture produces expected independently rounded Execution and Commitment allocations |
| AC-9–10  | Daily allocation loop and date derivation                              | One-day/multi-day allocation tests                 | Task dates visible from real Task workflow                                        |
| AC-11–12 | Same/different-assignee readiness branch                               | Allocation integration tests                       | Successor starts same Date versus next available Date                             |
| AC-13–14 | Lag candidate-date logic                                               | Weekend and no-blocker Lag tests                   | Change Lag and observe dates through Task form                                    |
| AC-15–17 | Dependency-ready set and priority comparator                           | Graph/order/data-integrity tests                   | Competing tasks follow dependency, Project Priority, and WBS order                |
| AC-18–20 | Contiguous slot allocator and displacement rules                       | Preemption/gap/displacement tests                  | Insert higher/lower-priority Task and observe whole-task shifts                   |
| AC-21–22 | Auto-blocker selection and reconciliation                              | Fit/no-fit/first-task tests                        | Dependency UI shows only actual blocker                                           |
| AC-23–24 | Ownership model and mutation semantics                                 | Repository/application dual-ownership tests        | Manual relation survives automatic recalculation and appears once                 |
| AC-25–26 | Assignee mutation affected-scope orchestration                         | Old/new/clear assignee tests                       | Change/clear Assignee; dependency and dates update without reload                 |
| AC-27–29 | Trigger wiring to concrete portfolio scheduler                         | Per-trigger integration tests                      | Dependency, Lag, Effort, WBS, Priority workflow updates dates                     |
| AC-30    | Completed exclusion and Actual End anchor                              | Completed blocker/task tests                       | Completed blocker drives successor while its dates stay unchanged                 |
| AC-31    | Locked reservation and Closed exclusion                                | Project-status application/allocation tests        | Locked dates stay fixed; Closed absent from scheduling effects                    |
| AC-32    | Portfolio affected graph/assignee query                                | Cross-project integration tests                    | Shared assignee/dependency updates multiple Project views                         |
| AC-33    | Independent Commitment allocator                                       | Execution-versus-Commitment tests                  | UI shows separately calculated Commitment dates                                   |
| AC-34    | Unscheduled result model                                               | Zero-capacity tests                                | UI shows safe reason and no fabricated date                                       |
| AC-35    | Transaction boundary                                                   | Failure injection rollback tests                   | User sees failure and unchanged Task/dependency/dates                             |
| AC-36    | Version/locking strategy                                               | Concurrent repository/application tests            | Acceptance harness proves no duplicate/overlap after racing action                |
| AC-37    | Cache version/invalidation path                                        | Frontend stale-response tests                      | Older response cannot restore old dates/dependency                                |
| AC-38    | Pending/duplicate guards plus rollback-only Task schedule preview on scheduling-field blur, including explicit Assignee clear | Backend preview rollback/API tests and frontend selected/cleared-Assignee, incomplete, and stale-resolution tests | Edit Task blur shows unconfirmed dates/dependencies; Assignee clear shows missing-Assignee projection; closing without Save persists nothing |
| AC-39    | Accessible components and responsive layout                            | Frontend a11y/viewport tests                       | Keyboard-only and narrow-viewport end-to-end feature flow                         |

---

## 23. Technical Completion Criteria

1. Concrete scheduler replaces no-op adapter in production composition for covered Execution and Commitment paths.
2. Domain and application logic do not depend on HTTP, ORM, or frontend framework.
3. Lag migration and persistence exist.
4. Dependency ownership model preserves manual semantics.
5. Automatic endpoint pair uniqueness is database-safe under concurrency.
6. Portfolio input query is bounded by active scope and uses reviewed indexes.
7. Cross-project graph and shared-assignee allocation are supported.
8. Decimal business arithmetic is deterministic and not binary-float dependent.
9. Mutable automatic daily allocation never exceeds final timeline capacity; valid same-assignee Task rows may overlap under US-6.3.
10. Priority-preserving Task Daily Limits, residual-capacity sharing, and safe auto-blocker reconciliation are enforced by production code and tests.
11. Locked timelines and allocations cannot be overwritten; Locked Project has no mutable scheduler output.
12. Completed records are historical anchors with Actual-End normalization, valid overcapacity, and no next-day debt; Closed Projects are excluded.
13. Triggered mutation plus transitive impacted-scope scheduling is atomic.
14. Schedule version/concurrency strategy prevents stale overwrite.
15. Structured errors are stable and frontend-safe.
16. Cache invalidation covers Task, dependency, Project, timeline, and workspace projections.
17. All ACs have complete Three-Level Confidence evidence.
18. Backend formatting, static analysis, migrations, tests, and required race tests pass.
19. Frontend formatting, lint, type check, tests, and production build pass.
20. Query/index review includes realistic portfolio cardinality and PostgreSQL plan evidence.
21. Backend is restarted after backend/config/migration changes and affected endpoints are smoke-tested.
22. Completion report lists exact commands/results and explicit remaining blockers.
23. Agent stops at first AC that lacks any evidence level rather than claiming partial story completion.

---

## 24. Documentation Impact

Implementation must assess and update:

- `docs/project/architecture.md`
  - portfolio scheduler boundary;
  - daily allocation model;
  - capacity formulas;
  - fixed-point precision;
  - priority and WBS ordering;
  - Task Capacity Allocation Percentage and concurrent same-assignee allocation;
  - auto/manual dependency ownership;
  - transaction and concurrency strategy;
  - Locked immutable-anchor and Actual End exception treatment;
  - completed historical overcapacity and no carry-over;
  - transitive impacted-scope traversal;
  - Priority simulation and Locked-impact rejection;
  - cache/version strategy;
  - query/index strategy.
- API documentation for Lag, generated dates, dependency source projection, unscheduled state, and structured errors.
- Scheduling Engine documentation for Execution and Commitment algorithms.
- US-1.2 documentation because scheduler Execution Capacity now applies Member Buffer.
- US-3.3 documentation because old buffer formula is superseded.
- US-5.1 documentation because Lag and Auto Dependency are now implemented by this story.
- US-4.1 documentation where no-op scheduler wording becomes concrete implementation.
- `AGENTS.md` only if durable project-wide engineering rules are introduced.
- README or environment files only if setup changes.

---

## 25. Locked Product Decisions

- Automatic Scheduling ON is the only activation config for concrete auto scheduling and auto dependency.
- Assignee change triggers Auto Dependency reconciliation.
- Dependency mutation triggers schedule recalculation.
- Effort, Lag, scheduling-relevant WBS mutation, and Project Priority mutation retain their established scheduler triggers. Name-only Task create is explicitly not a scheduling-relevant mutation.
- Scheduler fills Execution Start/End.
- Commitment Start/End are independently recalculated using lower Commitment Capacity.
- Public Holiday/weekend → Capacity `0`; otherwise minimum active Capacity Override per Member/Date → Daily Capacity when no override applies.
- Execution Capacity applies Member Buffer.
- Commitment Capacity applies Project Buffer after Member Buffer.
- Example `8`, `30%`, `20%` produces Raw Execution `5.6`, rounded Execution `5.5`, Raw Commitment `4.48`, and rounded Commitment `4.5` hours/day.
- Execution and Commitment Capacity are each rounded once to `0.5` hours after all buffers relevant to that timeline; neither timeline is rounded to whole days.
- Output dates are date-only and inclusive.
- Same-assignee successor may use remaining capacity on predecessor End Date when Lag `0`.
- Different-assignee successor with Lag `0` begins on the next positive-capacity Date.
- Lag belongs to Task, defaults `0`, and is a non-negative integer calendar-day offset followed by capacity availability.
- Friday plus Lag `1` starts Monday when weekend capacity is `0`.
- Task Capacity Allocation Percentage is owned by US-6.3, defaults to `100`, and caps planned allocation only after final timeline capacity rounding.
- Same-assignee Tasks may overlap on a Date; later ordered Tasks use only residual capacity.
- Higher-priority ready work can displace mutable lower-priority future allocation without moving it as one indivisible block.
- Task priority is Project Priority followed by visual WBS order.
- No additional business tie-breaker is used.
- Auto Dependency selects only the Task that actually prevents the earliest Execution start, not all previous Tasks and not always the latest End Date.
- At most one auto blocker exists per Task; manual blockers may remain multiple.
- Manual dependency semantics cannot be overwritten or deleted by Auto Dependency.
- Auto Dependency uses Execution allocation; Commitment uses the resulting effective graph.
- Scheduler supports portfolio relationships but recalculates only the transitive impacted scheduling scope; unrelated Projects are not touched.
- Completed Tasks use Actual End as dependency anchor and follow US-6.2 Actual Date/Actual Allocation rules.
- Historical overcapacity is valid, floors remaining capacity at zero, and creates no next-day debt.
- Locked Projects do not run Execution/Commitment scheduling; their baselines and allocations are immutable anchors.
- Locked→Open is an explicit Project transition that recalculates affected unfinished work.
- Every scheduling-impacting Task/capacity/priority mutation follows US-6.2 grouped impact guard; ordinary Locked impact blocks save, while factual Actual Date is the explicit exception.
- Closed Projects are excluded.
- Forecast remains outside this story.

---

## 26. Unresolved Questions

None.

---

## 27. Implementation Readiness Handoff

See `docs/project/automatic-scheduling-implementation-evidence.md` for the
complete AC-1 through AC-39 mapping.

- Code Inspection: `IMPLEMENTED BY CODE INSPECTION`
- Unit/Integration: `AUTHORED — NOT RUN — LOCAL VALIDATION REQUIRED`
- Acceptance-Level: `AUTHORED — NOT RUN — LOCAL VALIDATION REQUIRED`
- Overall: `IMPLEMENTED — LOCAL VALIDATION REQUIRED`

No automated validation was executed or recorded in the sandbox.
