# US-6.4 — Remove Auto Dependency by Assignee

> **Authority and supersession:** This story is the authoritative requirement
> for removing scheduler-owned dependency relations from SchedMind. It
> supersedes every Auto Dependency, automatic ownership, shared ownership,
> dependency-source badge, `Keep as Manual Dependency`, automatic dependency
> preview, reconciliation, and convergence rule in US-3.1, US-3.3, US-4.1,
> US-5.1, US-6.1, and US-6.3.
>
> Manual Finish-to-Start dependency, Task Lag, dependency readiness, Project
> Priority, visual WBS order, Task Capacity Allocation Percentage, remaining
> capacity allocation, Actual Allocation, Locked Project protection, impact
> confirmation, and all unaffected scheduling rules remain authoritative in
> their owning stories.

## 1. User Story

**Sebagai** Engineering Lead,
**Saya ingin** scheduler mengatur perebutan kapasitas langsung dari remaining
capacity tanpa membuat Auto Dependency by Assignee,
**Sehingga** dependency hanya merepresentasikan prerequisite yang sengaja
ditentukan user dan bukan artefak turunan dari hasil alokasi resource.

Story ini merupakan bagian dari **Epic 6: Scheduling Engine** dan merupakan
breaking simplification terhadap dependency, scheduling, persistence, API,
preview, UI, migration, serta automated evidence yang sudah ada.

---

## 2. Product Decision

Auto Dependency by Assignee dihapus sepenuhnya.

Scheduler sudah mempunyai seluruh informasi yang dibutuhkan untuk menentukan
kapan Task dapat memperoleh capacity:

1. manual dependency readiness;
2. Project Priority;
3. visual WBS order;
4. fixed completed, Locked, dan manual allocation;
5. Task Capacity Allocation Percentage;
6. remaining capacity Assignee per Date dan timeline.

Karena itu, resource contention tidak perlu disimpan kembali sebagai dependency.
Urutan atau keterlambatan akibat Assignee yang sama adalah **hasil allocation**,
bukan prerequisite bisnis.

Target invariant:

```text
Persisted Dependency = explicit manual Finish-to-Start prerequisite only
```

Scheduler dilarang membuat, menambah ownership, menghapus ownership, atau
mempertahankan dependency berdasarkan Assignee atau hasil capacity allocation.

---

## 3. Business Rationale

Auto Dependency mempunyai kelemahan berikut:

- menduplikasi informasi yang sudah diketahui allocator melalui remaining
  capacity;
- mengubah hasil scheduling menjadi input graph baru sehingga membutuhkan
  reconciliation/convergence loop;
- dapat membuat serial relation palsu ketika Task valid berjalan paralel melalui
  Task Capacity Allocation Percentage;
- membuat satu endpoint pair mempunyai state `manual`, `automatic`, atau `both`
  yang tidak diperlukan oleh user;
- membuat Assignee change dan schedule preview ikut memutasi atau memproyeksikan
  relation yang bukan business prerequisite;
- membuat Dependency UI mempunyai action dan badge tambahan tanpa memberi user
  kontrol yang bermakna;
- membuat Execution-derived resource ordering berpotensi membatasi Commitment
  melalui graph yang seharusnya tidak dimiliki salah satu timeline.

Setelah story ini, dependency menjawab satu pertanyaan saja:

> “Apakah user secara eksplisit menyatakan bahwa blocker harus selesai sebelum
> blocked Task boleh mulai?”

Resource contention menjawab pertanyaan berbeda dan tetap diselesaikan oleh
allocator:

> “Dengan priority, WBS order, fixed reservation, percentage, dan remaining
> capacity saat ini, kapan Task memperoleh positive allocation pertama?”

Kedua konsep tidak boleh dicampur kembali.

---

## 4. Terminology

| Term | Meaning |
| --- | --- |
| Manual Dependency | Persisted explicit Finish-to-Start relation yang dibuat user melalui `Blocks` atau `Blocked by` |
| Dependency Graph | Graph yang hanya terdiri dari Manual Dependency |
| Resource Contention | Dua atau lebih Task bersaing atas capacity Assignee yang sama |
| Allocation Ordering | Urutan allocator berdasarkan dependency readiness, Project Priority, lalu visual WBS order |
| Remaining Capacity | Final timeline capacity yang belum dipakai fixed reservation atau Task yang diproses lebih dahulu |
| Automatic-Only Legacy Relation | Existing row dengan `automatic_owned = true` dan `manual_owned = false` sebelum migration |
| Shared Legacy Relation | Existing row dengan `automatic_owned = true` dan `manual_owned = true` sebelum migration |
| Migration Recalculation | One-time full recalculation setelah legacy automatic ownership dibersihkan |

Istilah `Auto Dependency`, `Automatic Dependency`, `automatic ownership`,
`shared ownership`, dan `Manual + Automatic` tidak menjadi bagian dari target
product contract setelah migration selesai.

---

## 5. Scope

### 5.1 In Scope

- Menghapus Auto Dependency by Assignee dari Scheduling Engine.
- Menghapus automatic dependency reconciliation dan convergence loop.
- Menjadikan persisted dependency selalu manual.
- Membersihkan existing automatic-only dan shared ownership melalui migration.
- Menghapus ownership columns dan ownership-specific constraints dari schema.
- Menghapus dependency source dan manual-removability projection dari API.
- Menghapus endpoint dan use case `Keep as Manual Dependency`.
- Menghapus automatic/shared badge dan action dari UI.
- Menghapus dependency ownership dari schedule preview.
- Memastikan scheduler dan preview tidak menulis dependency.
- Menjalankan one-time recalculation agar dates/allocations tidak mempertahankan
  pengaruh automatic graph lama.
- Mempertahankan shared-Assignee impacted-scope calculation.
- Mempertahankan manual dependency graph validation dan scheduling readiness.
- Memperbarui architecture, product context, tests, and Three-Level Confidence
  evidence.

### 5.2 Out of Scope

- Menghapus Manual Dependency.
- Mengubah dependency type dari Finish-to-Start.
- Menambahkan resource link, soft dependency, informational arrow, atau relation
  pengganti Auto Dependency.
- Menambahkan Task Priority baru; priority tetap Project Priority lalu visual
  WBS order.
- Mengubah Task Capacity Allocation Percentage.
- Mengubah capacity precedence, buffer, rounding, or Actual Allocation rules.
- Mengubah Lag ownership atau calculation.
- Mengubah manual scheduling rules ketika Automatic Scheduling OFF.
- Menambahkan optimization engine yang mencari allocation paling efisien.
- Menampilkan “resource blocker” sebagai persisted relation.
- Forecast redesign atau Locked Project Forecast behaviour.

---

## 6. Target Dependency Domain Model

### 6.1 One Relation, One Meaning

Satu persisted row merepresentasikan:

```text
Blocking Task ID → Blocked Task ID
```

Keberadaan row berarti relation dibuat atau dipertahankan secara eksplisit oleh
user. Tidak ada ownership discriminator.

Target domain entity tidak mempunyai:

- `manualOwned`;
- `automaticOwned`;
- `source`;
- `manualRemovable`;
- `KeepAsManual` behaviour;
- automatic-only delete restriction.

### 6.2 Preserved Manual Dependency Rules

Rules berikut tetap berlaku:

- endpoint harus dua Executable WBS Task yang berbeda;
- duplicate endpoint pair ditolak;
- direct dan indirect cycle ditolak;
- relation dapat cross-project sesuai active/lifecycle rules US-5.1;
- completed blocker dapat tetap menjadi prerequisite;
- completed blocked Task tidak dapat menerima relation baru;
- existing relation yang blocked Task-nya completed tetap historical read-only;
- Closed Project tidak dapat menjadi endpoint relation baru;
- Task deletion dan executable/group conversion tetap membersihkan atau
  memindahkan endpoint secara atomik;
- Lag tetap milik blocked Task, bukan relation.

### 6.3 Delete Semantics

Karena seluruh target relation adalah manual:

- generic dependency delete menghapus row, subject to existing lifecycle,
  completed-history, graph, impact, dan transaction rules;
- tidak ada lagi operation “remove manual ownership but retain automatic
  ownership”;
- tidak ada lagi automatic-only relation yang read-only karena ownership;
- relation yang read-only tetap read-only hanya karena lifecycle atau historical
  completed-task rule, bukan source.

---

## 7. Scheduler Contract After Removal

### 7.1 Effective Graph

Execution dan Commitment menggunakan graph yang sama:

```text
Effective Graph = persisted Manual Dependency only
```

Tidak ada edge yang dibuat dari:

- Assignee equality;
- prior Task End;
- capacity exhaustion;
- Project Priority;
- WBS order;
- allocation percentage;
- Execution allocation result;
- Commitment allocation result.

### 7.2 Eligibility and Ordering

Task menjadi allocation candidate ketika:

1. lifecycle dan required scheduling fields valid;
2. semua Manual Dependency readiness terpenuhi untuk timeline tersebut;
3. Lag minimum sudah terpenuhi;
4. Date mempunyai positive final capacity setelah fixed reservations.

Ketika beberapa Task eligible, allocator tetap memproses:

```text
Project Priority → visual WBS order
```

Task yang diproses lebih dahulu menggunakan capacity sampai Task Daily Limit
atau Remaining Capacity, mana yang lebih kecil. Task berikutnya menggunakan
remaining capacity pada Date yang sama.

### 7.3 Same-Assignee Behaviour

Task dengan Assignee sama:

- boleh mempunyai overlapping dates jika keduanya memperoleh positive allocation
  pada Date yang sama;
- boleh menjadi serial secara alami ketika Task lebih dahulu menghabiskan
  capacity;
- tidak mendapatkan dependency relation hanya karena serial atau parallel
  allocation tersebut;
- dapat berubah dari serial menjadi parallel, atau sebaliknya, ketika percentage,
  capacity, priority, WBS order, fixed allocation, atau manual dependency berubah;
- tidak memerlukan relation cleanup ketika allocation pattern berubah.

### 7.4 Resource Delay Is Not Dependency

Bila Task B baru mulai setelah Task A selesai karena A menghabiskan remaining
capacity:

- B memiliki Start Date yang lebih lambat;
- allocation rows menjelaskan penggunaan capacity;
- tidak ada `A → B` dependency kecuali user membuatnya secara manual;
- mengubah atau menghapus A dapat membuat B maju tanpa dependency mutation.

### 7.5 Independent Execution and Commitment

Execution dan Commitment tetap dihitung secara independen menggunakan final
capacity masing-masing.

Dilarang:

- menggunakan Execution-derived ordering sebagai dependency input Commitment;
- menyimpan edge dari salah satu timeline untuk digunakan timeline lain;
- menjalankan convergence loop untuk membuat graph dan allocation saling
  menstabilkan.

Kedua timeline boleh menghasilkan allocation shape atau Start/End berbeda tanpa
menghasilkan relation baru.

### 7.6 Scheduler Side Effects

Confirmed scheduler mutation boleh menulis:

- generated Execution dates;
- generated Commitment dates;
- unscheduled reason;
- daily planned allocations;
- affected Project date projection;
- schedule snapshots/version menurut existing contract.

Confirmed scheduler mutation **tidak boleh** menulis `task_dependencies`.

Scheduler failure rollback scope tidak lagi menyebut dependency ownership.
Manual dependency create/delete tetap dapat memanggil scheduler, tetapi graph
mutation tersebut dilakukan oleh Dependency use case sebelum scheduler membaca
confirmed transaction state.

### 7.7 Convergence Removal

Hapus seluruh contract berikut:

- automatic dependency candidate selection;
- stale automatic ownership removal;
- automatic ownership add/update;
- “at most one automatic blocker” rule;
- reconciliation iteration;
- maximum convergence iteration;
- `ErrNoConvergence` atau equivalent product error;
- second scheduling pass yang hanya diperlukan karena graph berubah dari hasil
  allocation.

Satu scheduler invocation menghitung graph manual yang sudah final dan kemudian
mengalokasikan timeline tanpa graph mutation.

---

## 8. Scheduling Triggers and Impacted Scope

### 8.1 Preserved Triggers

Automatic scheduling tetap dipicu oleh confirmed changes yang memengaruhi
readiness, order, atau capacity, termasuk:

- Assignee;
- Effort;
- Lag;
- Task Capacity Allocation Percentage;
- manual dependency create/delete;
- WBS order or hierarchy changes;
- Project Priority;
- scheduling start date;
- capacity, buffer, override, holiday;
- Actual Date/Allocation rules owned by US-6.2;
- Project lifecycle transitions.

Assignee change tetap merupakan scheduler trigger karena shared capacity berubah,
bukan karena dependency harus direkonsiliasi.

### 8.2 Shared-Assignee Closure Must Remain

Penghapusan Auto Dependency tidak boleh mempersempit recalculation hanya ke
manual graph.

Impacted scheduling scope tetap mencakup transitive Open Projects yang terdampak
melalui:

- shared Assignee capacity;
- Manual Dependency;
- Project Priority competition;
- fixed allocation;
- approved scheduling constraints from existing stories.

Contoh: Task pada Project A berubah Assignee dan Member tersebut juga dipakai
Project B. Project B tetap harus direcalculate walaupun A dan B tidak mempunyai
manual dependency.

### 8.3 Locked and Closed Projects

- Locked Project tetap immutable scheduling anchor.
- Locked dates dan allocations tidak berubah karena feature removal.
- Open Project mengalokasikan capacity di sekitar Locked fixed allocations.
- Manual dependency ke/dari Locked Task tetap berlaku sesuai existing rules.
- Closed Project tetap excluded dari active scheduler.
- Menghapus legacy automatic ownership bukan alasan untuk membuka atau memutasi
  Locked/Closed Project.

---

## 9. Preview Contract

### 9.1 Draft Schedule Preview

Untuk eligible Open unfinished automatic Task, preview tetap dapat menghitung:

- draft generated Execution Start/End;
- draft generated Commitment Start/End;
- draft unscheduled reason;
- draft allocation projection bila current API memang menampilkannya;
- cross-project impact information menurut existing preview contract.

Preview tidak menghitung atau mengembalikan dependency ownership.

### 9.2 Assignee Change and Clear

Ketika Assignee draft berubah atau dikosongkan:

- preview merecalculate capacity impact untuk Assignee lama dan baru;
- cleared Assignee mengembalikan missing-Assignee unscheduled projection;
- preview tidak mencari stale automatic relation;
- preview tidak menambah, mengubah, atau menghapus dependency;
- confirmed Manual Dependency list tetap ditampilkan oleh dependency read API,
  bukan sebagai bagian dari draft scheduling result.

### 9.3 Rollback-Only Invariant

Preview tetap calculation-only dan rollback-only. Rollback scope mencakup draft
Task fields, dates, allocations, Project projections, snapshots, dan schedule
versions. Dependency table tidak seharusnya berubah sama sekali selama preview;
rollback tidak boleh dipakai untuk membenarkan temporary automatic relation
writes.

---

## 10. API Contract Changes

### 10.1 Dependency Response

Dependency projection tetap menyertakan identity dan endpoint information yang
dibutuhkan UI, misalnya:

- dependency ID;
- blocking Task;
- blocked Task;
- Task/Project display information;
- completion state;
- expected start projection bila tersedia;
- created/updated metadata sesuai existing public contract.

Hapus fields berikut dari response DTO:

- `source`;
- `manualOwned`;
- `automaticOwned`;
- `manualRemovable`;
- field lain yang hanya menjelaskan ownership source.

Absence of these fields is intentional breaking API cleanup. Backend tidak
mengirim constant `manual` sebagai compatibility fiction.

### 10.2 Create

`POST /api/dependencies` selalu membuat Manual Dependency.

Request tidak menerima ownership/source field. Unknown ownership fields harus
mengikuti existing strict request-decoding rule dan ditolak bila strict decoding
sudah menjadi contract endpoint.

### 10.3 Delete

`DELETE /api/dependencies/{dependencyId}` menghapus relation bila lifecycle dan
historical rules mengizinkan. Response/error mapping tidak mempunyai automatic-
ownership-specific branch.

### 10.4 Removed Endpoint

Hapus endpoint:

```text
POST /api/dependencies/{dependencyId}/keep-manual
```

Route tersebut tidak mempunyai replacement. Setelah rollout, request ke route
lama harus diperlakukan sebagai route yang tidak tersedia; tidak boleh diam-diam
membuat relation baru atau mengubah existing row.

### 10.5 Preview DTO

Hapus automatic dependency projection dari schedule preview response. Confirmed
manual dependency tetap dimuat melalui existing dependency detail query.

### 10.6 Internal Ports

Hapus ownership-specific application/repository ports, termasuk equivalent of:

- `KeepAsManual`;
- reconcile automatic ownership;
- list automatic relations for allocator;
- preview dependency ownership.

Dependency read/write port tetap hanya menyediakan manual graph operations.

---

## 11. Persistence and Migration

### 11.1 Existing Ownership Matrix

Sebelum migration, setiap row diproses berdasarkan persisted ownership:

| `manual_owned` | `automatic_owned` | Migration result |
| ---: | ---: | --- |
| `true` | `false` | Keep row as Manual Dependency |
| `true` | `true` | Keep the same endpoint pair and relation identity as Manual Dependency |
| `false` | `true` | Delete row |
| `false` | `false` | Invalid/corrupt legacy row; fail migration before destructive schema change |

Shared relation dipertahankan karena user pernah memilih manual ownership atau
relation sudah mempunyai manual intent. Automatic-only relation dihapus karena
tidak mempunyai explicit user intent.

### 11.2 Lifecycle-Neutral Cleanup

Migration cleanup berlaku tanpa memandang endpoint saat ini:

- Open;
- Locked;
- Closed;
- unfinished;
- completed;
- historical read-only.

Alasannya, cleanup mengubah deprecated ownership metadata, bukan menjalankan
normal user dependency mutation.

Namun:

- mixed/shared row yang dipertahankan tetap tunduk pada lifecycle/historical
  rules setelah migration;
- deleting automatic-only row tidak mengubah Actual Date, Locked baseline, atau
  Closed Project state;
- migration tidak memerlukan per-row UI confirmation atau scheduling-impact
  confirmation.

### 11.3 Target Schema

Setelah data cleanup berhasil:

- drop `manual_owned`;
- drop `automatic_owned`;
- drop `task_dependencies_has_owner` atau equivalent ownership constraint;
- preserve primary key;
- preserve unique `(blocking_task_id, blocked_task_id)` constraint;
- preserve endpoint foreign keys and directional indexes;
- row existence menjadi satu-satunya ownership semantics.

New schema dilarang mempertahankan dormant automatic columns “untuk nanti”.

### 11.4 Migration Recalculation

Membersihkan relation saja tidak cukup karena persisted generated dates dan
allocations mungkin masih dipengaruhi automatic graph lama.

Setelah ownership cleanup dan sebelum rollout dianggap complete, sistem wajib
menjalankan **one-time full active-portfolio recalculation** dengan target rules
story ini:

- recalculate seluruh Open Project dengan Automatic Scheduling ON;
- use Manual Dependency graph only;
- use current Project Priority, WBS order, capacity, percentage, fixed manual,
  completed Actual Allocation, and Locked anchors;
- do not change Automatic Scheduling OFF manual timelines;
- do not change Locked generated timelines or allocations;
- exclude Closed Projects as mutable scheduler output;
- update schedule projection/version consistently with existing scheduler rules;
- remove stale dates/allocations that only existed because of legacy automatic
  readiness;
- allow Tasks to move earlier, later, or overlap when remaining-capacity result
  requires it.

A narrow impacted-scope calculation is not sufficient for this one-time rollout
because the full set of legacy automatic edges and their transitive historical
influence may not be reconstructible after cleanup.

### 11.5 Atomicity and Rollback

Rollout must run inside the repository's scheduling mutation serialization or a
maintenance boundary that prevents concurrent confirmed mutations.

The following must not be observable as a partially deployed state:

- schema still exposes ownership while application assumes manual-only;
- ownership rows cleaned but generated dates remain based on legacy graph;
- frontend removes source fields while backend still requires them;
- backend removes route while frontend still calls it.

If data cleanup, schema change, full recalculation, or persistence fails:

- rollout must fail closed;
- no partial target-state claim may be emitted;
- database/application compatibility must follow the repository's migration
  rollback strategy;
- implementation evidence must record the failure path and recovery procedure.

### 11.6 Idempotence

Migration and rollout guard must be safe against restart:

- already-kept manual rows must not duplicate;
- deleted automatic-only rows must not reappear;
- recalculation must not create dependency rows;
- repeated completion checks must detect target schema/state rather than rerun
  destructive ownership assumptions.

---

## 12. UI and UX Changes

### 12.1 Dependency List

`Blocks` dan `Blocked by` tetap ditampilkan pada Edit Task.

Hapus:

- `Automatic` badge;
- `Manual + Automatic` badge;
- source label/tooltips;
- automatic-only read-only explanation;
- `Keep as Manual Dependency` action;
- ownership-specific helper text;
- unconfirmed automatic dependency preview rows.

Setiap visible dependency adalah manual. UI tidak perlu menampilkan badge
`Manual` karena tidak ada source alternatif.

### 12.2 Mutation Actions

- Add dependency tetap tersedia sesuai lifecycle rules.
- Remove dependency tersedia sesuai lifecycle/historical rules.
- Disabled state menjelaskan lifecycle restriction, bukan automatic ownership.
- Existing duplicate-action protection, Retry, loading, stale response, focus,
  keyboard, and accessibility rules tetap berlaku.

### 12.3 Gantt

Dependency arrow pada Home/Gantt hanya menggambar persisted Manual Dependency.

Resource contention tidak digambar sebagai arrow, connector, dashed relation,
atau hidden dependency.

Task yang serial karena capacity dapat terlihat dari dates/allocations tanpa
mengklaim prerequisite bisnis.

### 12.4 Schedule Preview

Preview UI hanya menandai generated dates/allocations sebagai unconfirmed.
Dependency list tetap confirmed manual data dan tidak berubah sampai user
secara eksplisit menambah/menghapus dependency.

### 12.5 User-Facing Terminology

Hapus product copy yang menyebut:

- Auto Dependency by Assignee;
- automatic blocker;
- dependency source;
- shared ownership;
- keep automatic dependency as manual;
- reconciliation.

Automatic Scheduling copy harus menjelaskan generated dates/capacity allocation,
bukan automatic dependency creation.

---

## 13. Detailed Behaviour and Edge Cases

### 13.1 Capacity Exhaustion Without Dependency

```text
Daily Execution Capacity = 8h
Task A = 8h, 100%, order first
Task B = 8h, 100%, order second
No Manual Dependency
```

Expected:

- A allocated Date 1;
- B allocated Date 2;
- no dependency row exists;
- deleting/changing A may move B earlier through recalculation only.

### 13.2 Parallel Allocation Without Dependency

```text
Daily Execution Capacity = 8h
Task A = 4h, 100%, order first
Task B = 8h, 50%, order second
No Manual Dependency
```

Expected Date 1:

- A receives 4h;
- B may receive up to its rounded daily limit from the remaining 4h;
- both may start Date 1;
- no dependency row exists.

### 13.3 Three-Task Percentage Example

```text
Capacity = 8h
A = 4h / 100%
B = 4h / 20%
C = 4h / 100%
Order = A → B → C
```

Expected allocation remains governed by US-6.3, for example Date 1 A `4h`, B
`1.5h`, C `2.5h` when its rounding rules produce those limits. B and C may both
start Date 1 and C may finish first. No auto relation is created among A, B, C.

### 13.4 Manual Dependency Still Blocks

```text
A manually Blocks B
Capacity is otherwise available for B
```

B receives no allocation before manual Finish-to-Start readiness plus B Lag is
satisfied. Remaining capacity does not override explicit dependency.

### 13.5 Removing Manual Dependency

When user removes `A → B` and Save succeeds:

- scheduler recalculates affected scope;
- B may move earlier or overlap based on remaining capacity;
- no automatic relation recreates `A → B`;
- undo requires user to create the manual relation again.

### 13.6 Higher-Priority Displacement

When higher-priority Task C becomes ready and shares Assignee with lower-priority
A/B:

- mutable future allocations may move according to Project Priority/WBS rules;
- fixed completed, Locked, and manual allocations remain fixed;
- no dependency is created to explain the displacement.

### 13.7 Assignee Change

Changing Task B from Member X to Member Y:

- recalculates projects sharing X or Y as required;
- leaves all Manual Dependency rows unchanged;
- does not remove a Manual Dependency merely because blocker has different
  Assignee;
- does not create a relation to a Task assigned to Y;
- stale preview response cannot restore old Assignee/dates/allocations.

### 13.8 Cleared Assignee

Clearing Assignee:

- leaves Manual Dependency intact;
- makes Task unscheduled under missing-Assignee rule;
- releases future mutable planned capacity previously used by the Task;
- recalculates impacted shared-capacity scope;
- performs no dependency cleanup.

### 13.9 Execution/Commitment Divergence

If buffer differences make Execution Task B start Date 1 but Commitment start
Date 2:

- both projections are valid;
- no relation is created from either result;
- manual graph remains identical for both timelines.

### 13.10 Completed and Actual Allocation

Completed Task remains historical fixed Actual Allocation anchor according to
US-6.2.

A later unfinished Task may be delayed by that fixed capacity without receiving
a dependency. Actual chronology does not create dependency automatically.

### 13.11 Locked Anchor

Locked Task allocation may delay Open Task sharing the same Assignee. The Open
Task is placed around Locked capacity; no dependency arrow is created and Locked
baseline is not changed.

### 13.12 Existing Shared Legacy Relation

For row `manual_owned = true`, `automatic_owned = true`:

- keep same dependency ID and endpoint pair;
- remove ownership metadata;
- relation remains manual;
- user can later delete it only under normal lifecycle rules;
- migration must not create a duplicate replacement row.

### 13.13 Existing Automatic-Only Legacy Relation

For row `manual_owned = false`, `automatic_owned = true`:

- delete relation during migration;
- do not convert it to manual;
- do not expose a migration prompt;
- recalculate Open automatic schedules without it;
- preserve Actual Date and Locked/Closed baseline data.

### 13.14 Completed Historical Automatic-Only Relation

Even when the blocked Task is completed and normal UI deletion would be
historically read-only, automatic-only relation is removed by migration because
it never represented explicit user intent. Actual history remains unchanged.

### 13.15 Manual Scheduling OFF

When Automatic Scheduling OFF:

- user-provided Execution/Commitment dates remain authoritative;
- manual dependency remains visible and validated;
- removing auto-dependency feature does not recalculate or rewrite manual dates;
- fixed manual allocation continues reducing capacity for automatic Projects.

### 13.16 Cross-Project Manual Dependency

Cross-project Manual Dependency continues to affect readiness and transitive
impact. Removing automatic ownership must not accidentally restrict dependencies
to same Project or same Assignee.

### 13.17 No Replacement Relation

Implementation must not introduce renamed equivalents such as:

- resource dependency;
- capacity dependency;
- inferred dependency;
- scheduler link;
- hidden dependency;
- blocker metadata persisted solely from allocator output.

Resource explanation belongs to allocation/timeline projection, not dependency
persistence.

---

## 14. Error Handling

Remove ownership-specific errors and branches, including equivalent of:

- automatic dependency cannot be deleted;
- dependency must be kept as manual first;
- automatic reconciliation did not converge;
- unsupported automatic ownership state.

Preserve errors for:

- dependency not found;
- duplicate relation;
- self-dependency;
- invalid endpoint type;
- cycle with safe path;
- completed blocked Task restriction;
- Closed/Locked lifecycle restriction;
- stale scheduling impact confirmation;
- scheduler/persistence failure;
- concurrent version conflict.

Errors must remain stable, safe, and must not expose SQL/internal graph details.

---

## 15. Security, Integrity, and Transaction Rules

- Client cannot submit hidden ownership/source fields to recreate automatic
  state.
- Backend is source of truth for dependency graph and lifecycle validation.
- Unique endpoint pair remains database-enforced.
- Manual graph cycle validation remains portfolio-wide and transaction-safe.
- Dependency create/delete and scheduling recalculation remain atomic.
- Scheduler cannot bypass Dependency application/domain invariants by writing
  relation rows directly.
- Preview cannot persist dependency rows.
- Migration validates row counts and ownership matrix before dropping columns.
- No automatic-only relation may survive target migration.
- No target row may have zero owners because ownership concept no longer exists.

---

## 16. Performance and Complexity

Removing reconciliation should reduce scheduler complexity:

```text
Old: load graph → allocate → infer edges → mutate graph → repeat until convergence
New: load manual graph → allocate once per timeline
```

Requirements:

- no repeated scheduling pass solely for dependency reconciliation;
- no ownership query in normal allocator path;
- shared-Assignee impacted-scope query remains indexed and bounded by existing
  portfolio rules;
- dependency detail and candidate indexes remain;
- migration full recalculation may be maintenance-time work and is not evidence
  that every normal mutation may recalculate the entire portfolio;
- normal mutation must continue using approved impacted scope, not globally
  recalculate as a shortcut.

---

## 17. Acceptance Criteria

### AC-1 — Dependency becomes manual-only

**Given** target migration is complete
**Then** every persisted dependency represents explicit manual Finish-to-Start
intent
**And** no ownership/source field exists in domain, persistence, or public API.

### AC-2 — Scheduler never creates dependency

**When** confirmed automatic scheduling runs for any trigger
**Then** dependency row count and endpoint pairs are unchanged except for a
manual dependency mutation in the same owning use case
**And** allocator output alone cannot create a relation.

### AC-3 — Scheduler never deletes manual dependency

**Given** a Manual Dependency exists
**When** Assignee, percentage, capacity, priority, WBS order, dates, or allocation
shape changes
**Then** relation remains unchanged unless user explicitly deletes it through
Dependency use case.

### AC-4 — Capacity exhaustion remains schedulable

**Given** same-Assignee Tasks have no Manual Dependency and earlier ordered work
uses all capacity
**When** scheduler runs
**Then** later Task starts on the next Date with capacity
**And** no dependency is created.

### AC-5 — Parallel capacity remains schedulable

**Given** same-Assignee Tasks have no Manual Dependency and percentage leaves
capacity
**When** scheduler runs
**Then** multiple Tasks may receive positive allocation on the same Date
**And** no dependency is created.

### AC-6 — Manual dependency remains authoritative

**Given** A manually Blocks B
**When** B otherwise has capacity
**Then** B remains blocked until Finish-to-Start readiness plus Lag is satisfied.

### AC-7 — Manual delete is not recreated

**When** user deletes a Manual Dependency and confirmed recalculation succeeds
**Then** relation is absent
**And** scheduler does not recreate it based on Assignee or allocation order.

### AC-8 — Priority displacement uses capacity only

**When** higher-priority work displaces lower-priority mutable allocation
**Then** dates/allocations change according to approved ordering
**And** no dependency is added between displaced Tasks.

### AC-9 — Execution and Commitment use manual graph only

**When** Execution and Commitment allocations differ
**Then** each uses the same Manual Dependency graph
**And** neither timeline persists an edge for the other.

### AC-10 — No convergence loop

**When** scheduler runs
**Then** it does not iterate to reconcile automatic ownership
**And** no `automatic dependency reconciliation did not converge` error can be
returned.

### AC-11 — Shared-Assignee impact retained

**Given** Project A mutation changes capacity usage of Member X
**And** Project B also uses Member X without Manual Dependency
**When** mutation is confirmed
**Then** Project B is included when actually impacted under existing scope rules.

### AC-12 — Assignee change does not mutate graph

**When** Task Assignee changes or is cleared
**Then** affected schedules/allocations recalculate
**And** every Manual Dependency remains unchanged
**And** no automatic relation is created or removed.

### AC-13 — Preview has no ownership projection

**When** Task draft preview runs
**Then** it returns generated scheduling projection without dependency ownership
**And** preview does not write dependency rows even temporarily.

### AC-14 — Dependency API source fields removed

**When** dependency detail is returned
**Then** response contains no `source`, `manualOwned`, `automaticOwned`, or
`manualRemovable` field
**And** frontend does not require them.

### AC-15 — Keep-as-manual endpoint removed

**When** client calls the former keep-manual route after rollout
**Then** route is unavailable
**And** no dependency mutation occurs.

### AC-16 — Dependency UI simplified

**When** Engineering Lead opens Edit Task
**Then** `Blocks` and `Blocked by` show manual relations without source badge
**And** no `Keep as Manual Dependency` action or automatic preview row exists.

### AC-17 — Gantt arrows are manual-only

**When** Home/Gantt renders dependencies
**Then** every arrow maps to persisted Manual Dependency
**And** resource contention alone produces no arrow.

### AC-18 — Automatic-only migration rows deleted

**Given** legacy row has `manual_owned = false` and `automatic_owned = true`
**When** migration runs
**Then** row is deleted
**And** it is not converted to manual.

### AC-19 — Shared migration rows retained

**Given** legacy row has both ownership flags true
**When** migration runs
**Then** same relation ID and endpoint pair remain as Manual Dependency
**And** no duplicate row is created.

### AC-20 — Manual-only migration rows retained

**Given** legacy row has only manual ownership
**When** migration runs
**Then** relation remains with same identity and endpoints.

### AC-21 — Ownership schema removed

**When** migration completes
**Then** ownership columns and ownership constraint are absent
**And** endpoint uniqueness, foreign keys, and indexes remain valid.

### AC-22 — Full migration recalculation

**Given** generated dates may have been constrained by legacy automatic graph
**When** rollout migration succeeds
**Then** all Open automatic Projects are recalculated using Manual Dependency
and current capacity rules
**And** stale graph-derived dates/allocations do not remain.

### AC-23 — Locked baseline preserved during migration

**Given** legacy automatic relations involve Locked Task
**When** cleanup/recalculation runs
**Then** deprecated automatic-only rows are removed as required
**And** Locked dates/allocations remain unchanged
**And** Open work allocates around Locked anchors.

### AC-24 — Manual timelines preserved during migration

**Given** Project has Automatic Scheduling OFF
**When** migration runs
**Then** manual Execution/Commitment dates remain unchanged
**And** fixed manual allocation remains a capacity reservation.

### AC-25 — Completed history preserved

**Given** automatic-only relation involves completed Task
**When** migration removes relation
**Then** Actual Date and Actual Allocation remain unchanged
**And** no false manual history is created.

### AC-26 — Atomic rollout

**When** ownership cleanup, schema transition, or mandatory recalculation fails
**Then** rollout fails closed under approved migration strategy
**And** system is not declared migrated with mixed old/new contracts.

### AC-27 — Mutation rollback simplified

**When** normal confirmed scheduler persistence fails
**Then** Task, dates, allocations, Project projection, snapshot, and version
rollback atomically
**And** dependency ownership is not part of scheduler rollback because scheduler
never mutates it.

### AC-28 — No replacement inferred relation

**Then** code, schema, API, and UI contain no renamed scheduler-generated
resource dependency or hidden edge that recreates removed behaviour.

### AC-29 — Documentation consistency

**Then** architecture, project context, affected user stories, API docs, and
implementation evidence consistently describe manual-only dependency
**And** old Auto Dependency clauses are explicitly superseded rather than used
as acceptance authority.

### AC-30 — Three-Level Confidence

**Then** every acceptance criterion has traceable Code Inspection,
Unit/Integration Test, and Acceptance-Level Test evidence according to
`AGENTS.md`
**And** pre-removal tests that expect automatic ownership are deleted or rewritten,
not counted as passing evidence.

---

## 18. Mandatory Test Scenarios

### 18.1 Domain/Application

- dependency entity has no ownership flags or KeepAsManual transition;
- create always creates manual relation;
- delete removes relation subject to lifecycle rules;
- duplicate, self, cycle, completed blocked, Closed endpoint rules preserved;
- manual relation survives Assignee/percentage/capacity changes;
- scheduler ports expose no automatic reconciliation operation;
- removed convergence error is unreachable/absent;
- impacted scope includes shared Assignee without dependency.

### 18.2 Scheduler Unit/Integration

- capacity exhaustion serializes dates without relation creation;
- percentage permits parallel allocation without relation creation;
- higher priority displacement without relation creation;
- manual dependency blocks even when capacity exists;
- deleting manual dependency allows earlier allocation and is not recreated;
- Execution/Commitment divergence uses same manual graph;
- Assignee change/clear leaves graph unchanged;
- Locked and completed fixed allocations delay Open work without edges;
- preview performs zero inserts/updates/deletes on dependency table;
- confirmed scheduler performs zero dependency DML unless owning operation is a
  manual dependency create/delete;
- no reconciliation loop or convergence failure.

### 18.3 Migration/Repository

- ownership matrix row counts before/after;
- shared/manual row IDs preserved;
- automatic-only rows deleted including completed/Locked/Closed endpoints;
- invalid no-owner rows fail migration before ownership columns are dropped;
- columns/constraint dropped;
- unique endpoint pair, foreign keys, indexes retained;
- migration idempotence/restart safety;
- full active-portfolio recalculation updates Open automatic projections;
- manual timelines, Locked baselines, Actual history unchanged;
- injected cleanup/schema/recalculation failure follows rollback/recovery contract;
- concurrent writes are prevented during rollout boundary.

### 18.4 HTTP

- dependency response omits ownership fields;
- create ignores no ownership because unknown source fields are rejected under
  strict DTO contract;
- delete has no ownership-specific branch;
- keep-manual route absent and causes no mutation;
- preview response omits automatic dependency projection;
- existing stable graph/lifecycle error mappings remain.

### 18.5 Frontend Component

- no Automatic/Manual+Automatic badge;
- no Keep as Manual action;
- manual add/remove still works;
- lifecycle read-only state remains;
- dependency detail DTO compiles without source fields;
- preview renders dates/allocations only;
- Assignee clear preview does not expect dependency reconciliation;
- Gantt renders only confirmed manual arrows;
- loading, Retry, duplicate action protection, stale response, accessibility,
  keyboard, and focus behaviour remain.

### 18.6 Acceptance-Level

1. Create two same-Assignee 100% Tasks without dependency; verify later Task
   follows remaining capacity and no arrow/relation appears.
2. Set first Task below 100%; verify parallel allocation and no relation appears.
3. Add manual `A Blocks B`; verify B waits despite spare capacity.
4. Remove `A Blocks B`; verify B may move earlier and relation is not recreated.
5. Change Assignee; verify cross-project capacity recalculation occurs while
   manual dependency list is unchanged.
6. Compare Execution and Commitment; verify different dates do not create edges.
7. Upgrade database containing automatic-only, shared, and manual-only rows;
   verify migration matrix, stable IDs, and removed columns.
8. Verify full migration recalculation changes eligible Open schedules while
   preserving manual, Locked, Closed, and completed historical data.
9. Open Edit Task and Home Gantt; verify no source badge, Keep as Manual action,
   automatic preview row, or resource-derived arrow remains.
10. Force migration and normal scheduler failures; verify no mixed contract or
    partial confirmed state.

---

## 19. Required Documentation and Evidence Updates

Implementation of this story must update:

- US-3.1 project configuration wording;
- US-3.3 Automatic Scheduling setting wording;
- US-4.1 Task preview and mutation wording;
- US-5.1 dependency ownership, API, UI, AC, and tests;
- US-6.1 scheduler algorithm, triggers, preview, AC, and tests;
- US-6.3 percentage/parallel allocation sections and AC that mention automatic
  blockers;
- `docs/project/architecture.md`;
- `docs/project/schedmind-context.md`;
- OpenAPI/API documentation if present;
- automatic scheduling and capacity-allocation implementation evidence;
- migration operational notes and rollback procedure.

Historical text may remain only when it is clearly marked superseded and cannot
be mistaken for current target behaviour.

---

## 20. Definition of Done

- Auto Dependency by Assignee is removed end-to-end, not merely hidden in UI.
- Dependency persistence and API are manual-only.
- Legacy ownership is migrated using the approved matrix.
- Mandatory full active-portfolio recalculation is completed safely.
- Scheduler and preview perform no automatic dependency DML.
- Shared-Assignee impacted-scope recalculation remains correct.
- Manual dependency, Lag, priority, WBS order, percentage, fixed allocation,
  Actual Allocation, and Locked rules remain intact.
- No ownership badge, Keep as Manual action, automatic preview row, or resource
  arrow remains.
- No renamed inferred relation recreates the removed feature.
- Validation passes and every AC has Three-Level Confidence evidence.
