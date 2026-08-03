# US-3.1 — Create Project

> **Product decision update — US-6.4:** Auto Dependency by Assignee is removed
> end-to-end. `Automatic Scheduling` remains the Project configuration for
> generated Execution/Commitment dates and capacity allocation only; it never
> creates dependency relations. This supersedes every Auto Dependency statement
> in this story.

> **Product decision update — US-6.1:** `Automatic Scheduling` is the only
> Project configuration that activates generated Execution/Commitment dates and
> Auto Dependency by Assignee. The former system field
> `Auto Dependency by Assignee` is removed.

> Product decision update: Project includes a nullable `Scheduling Start Date`
> (`SQL DATE`, API `YYYY-MM-DD`). It is the only initial anchor for future
> Automatic Scheduling and is configured in the combined Add/Edit Project form.
> Automatic Scheduling may be saved without it, but generated timelines must
> remain empty and the UI must warn: `Automatic Scheduling requires a Project
Scheduling Start Date.` No task-level Earliest Start or equivalent anchor is
> permitted. Scheduling calculation remains outside this story.

> **Product decision update — US-6.2:** Completion uses a required Actual Date
> pair (`Actual Start` and `Actual End`). Locked Project planning remains
> immutable, but Actual Date may be recorded without changing the protected
> Execution/Commitment baseline. Its Actual Allocation may still recalculate
> impacted Open Projects. `Locked → Open` uses atomic transitive Reopen closure
> when multiple Locked Projects must be opened together. Every scheduling-
> impacting Priority/capacity mutation uses grouped cross-project impact
> warning, server revalidation, and Locked-impact blocking, except factual
> Actual Date. US-6.2 supersedes contradictory lifecycle, priority, completion,
> and Locked-edit wording in earlier revisions of this story.

> **Product decision update — US-7.1:** Home is the canonical active-Project
> WBS surface. Project lifecycle commands remain available on Projects and are
> also exposed from eligible active Project rows on Home through the same
> application commands. The standalone Project Structure entry point is removed.
> Edit Project opened from Home renders directly over Home. A Locked Project may
> rename Project Name only; Settings and every scheduling-relevant field remain
> read-only.

## 1. User Story

**Sebagai** Engineering Lead,
**Saya ingin** membuat Project,
**Sehingga** Project dapat direncanakan.

Project merupakan bagian dari **Epic 3: Project Management**.

---

## 2. Business Context

Project adalah root planning entity yang menjadi batas kepemilikan WBS,
Executable Leaf, timeline, status lifecycle, prioritas, scheduling, dan Gantt.
Pembuatan Project menyediakan identitas dan konfigurasi awal agar workflow WBS
dan scheduling pada story berikutnya mempunyai parent yang jelas.

Status Project memengaruhi apakah planning dapat berubah, timeline mana yang
dinamis atau dilindungi, apakah Project ikut scheduling, dan apakah Project
ditampilkan pada Gantt. Karena itu lifecycle `Open`, `Locked`, dan `Closed`
merupakan bagian dari kontrak domain story ini walaupun implementasi WBS,
scheduler, timeline calculation, dan Gantt dimiliki story berikutnya.

---

## 3. Scope

Engineering Lead dapat:

1. Membuka page **Projects** di bawah navigation group **Project** dan membuka
   active Project dari Home.
2. Melihat paginated Project List.
3. Mencari Project berdasarkan Name secara case-insensitive.
4. Membuka Project detail.
5. Membuat Project dengan data yang valid.
6. Melihat status Project.
7. Mengubah seluruh data Project yang diizinkan ketika Open dan hanya Name
   ketika Locked.
8. Mengubah status sesuai transition yang diizinkan.
9. Mengunci Open Project dan melindungi Execution serta Commitment baseline.
10. Menutup Open atau Locked Project yang seluruh Executable Leaf-nya selesai.
11. Membuka kembali Locked atau Closed Project menjadi Open.
12. Melihat Closed Project setelah seluruh active Project pada Project List.
13. Mengubah Priority melalui Move Up atau Move Down.
14. Menghapus Project yang belum mempunyai child WBS/Executable Leaf.
15. Menjalankan lifecycle action melalui Projects atau eligible active Project
    row pada Home dengan behaviour yang sama.

Kontrak berikut ditetapkan sekarang untuk consumer pada story berikutnya:

- Open Project ikut scheduling dan terlihat di Gantt.
- Locked Project tetap terlihat di Gantt tetapi tidak menjalankan Execution/Commitment scheduler; persisted baseline dan allocation menjadi immutable anchors.
- Project harus mempunyai sedikitnya satu Executable Leaf dan seluruh unfinished Task harus scheduled sebelum dapat di-Lock.
- Closed Project tidak ikut scheduling atau Gantt, tetapi tetap ada di list.
- Completed Executable Leaf ditentukan oleh complete Actual Date pair dan tidak dapat diedit melalui normal planning mutation.
- Actual Date tetap dapat diisi pada unfinished Task milik Locked Project tanpa mengubah protected baseline. Actual Allocation dapat memicu recalculation pada impacted Open Projects, tetapi tidak boleh memutasi Locked Project.
- Close validation memeriksa seluruh descendant Executable Leaf.
- Locked Project harus direopen secara eksplisit ke Open sebelum Task, WBS, Settings, timeline, atau dependency dapat diubah. Project Name rename adalah satu-satunya Project-field mutation yang tetap diizinkan saat Locked dan tidak menjalankan scheduler.
- Forecast behaviour while Locked is deferred to its owning future requirement.

Story ini tidak mewajibkan implementasi WBS editor, task editor, Scheduling
Engine, timeline calculator, atau Gantt UI yang belum tersedia. Implementasi
Project harus menyediakan boundary dan kontrak agar invariant tersebut tidak
perlu didefinisikan ulang oleh story lanjutan.

---

## 4. Out of Scope

- Membuat atau mengubah WBS.
- Membuat atau mengubah task atau Executable Leaf.
- Implementasi dependency, lag, effort, assignee, Actual End, atau Project
  Freeze editor.
- Implementasi Scheduling Engine dan timeline calculation.
- Implementasi Gantt Chart atau historical Gantt.
- Menghapus Project yang sudah mempunyai child; Project tersebut hanya dapat
  dipertahankan sebagai active Project atau diubah menjadi Closed ketika eligible.
- Automatic closing.
- Completion Percentage atau task status terpisah.
- Archived status atau archived flag.
- `Show Closed Projects` toggle.
- Bulk status update.
- Approval workflow.
- Project template.
- Import atau external synchronization.
- Portfolio dashboard, reporting, budget, client, Project Manager, description,
  department, squad, atau team.
- Manual Forecast End, Manual Commitment End, atau Manual Execution End.
- Permission model baru; aplikasi belum mempunyai permission model.

Deferred functionality tidak boleh diimplementasikan sebagian hanya untuk
memenuhi story ini.

---

## 5. UI Placement and User Flow

### 5.1 Projects Surface

- Navigation tetap menyediakan group **Project** dengan menu **Projects**.
- Page menggunakan persistent application shell; top bar, sidebar, dan global
  chrome tidak diduplikasi atau diremount.
- Page title dan menu label adalah **Projects**.
- Primary action adalah **Add Project**.
- Project List mengikuti shared list, search, pagination, skeleton, empty state,
  no-results, Retry, background refresh, dan Toast patterns.
- The standalone **Project Structure** action/page is removed.
- Projects remains the surface for Add Project, list/search/pagination, Edit
  Project, and lifecycle actions for Open, Locked, and Closed Projects.

### 5.2 Shared Add/Edit Project Dialog

- Add form tidak meminta user memilih status; status Open dijelaskan sebagai
  default yang ditetapkan backend.
- Add/Edit Project menggunakan combined form dari US-3.3: Project Name,
  Automatic Scheduling, Scheduling Start Date, dan Project Buffer. Status dan
  Priority tidak dipilih user; Priority ditetapkan sistem pada posisi terendah.
- Karena Project adalah WBS level `0`, Edit Project juga composes the read-only
  whole-Project summary owned by US-4.3. Add Project does not display it.
- Edit Project uses the existing shared wide Dialog variant so settings and the
  summary are not constrained to the standard dialog width; Add Project may
  retain the standard width.
- The same Edit Project dialog is opened from Projects and from Project Name on
  Home. Home invocation renders directly over Home without a route/background
  switch.
- Open Project exposes all fields allowed by US-3.1 and US-3.3.
- Locked Project exposes Project Name as the only editable field. Automatic
  Scheduling, Scheduling Start Date, Project Buffer, and other Settings remain
  visible/read-only.
- Closed Project remains fully read-only until Reopen.
- Locked Project Name rename is a non-scheduling mutation: no impact preview,
  scheduler invocation, baseline mutation, allocation mutation, or status
  transition occurs.
- Draft dipertahankan setelah validation atau backend failure.
- Controls terkait dinonaktifkan selama mutation untuk mencegah duplicate
  submission tanpa memblokir seluruh page.

### 5.3 Lifecycle Actions on Projects and Home

- Status ditampilkan sebagai text label, bukan hanya warna.
- Projects exposes the established lifecycle actions for every status.
- Home active Project row overflow exposes the same eligible commands:
  - Open: Lock Project, Close Project, and Delete Project when childless.
  - Locked: Reopen Project and Close Project.
  - Closed: absent from Home; Reopen Project remains available on Projects.
- Both surfaces invoke the same command, eligibility validation, impact preview,
  confirmation, transaction, error mapping, cache invalidation, and rollback.
- No second Home-specific lifecycle state machine is permitted.
- Project tanpa child menyediakan confirmed **Delete Project**. Project yang
  mempunyai child tidak dapat dihapus dan diarahkan menggunakan Close setelah
  completion requirement terpenuhi.
- Lock confirmation menjelaskan bahwa Execution dan Commitment dates menjadi
  protected baseline dan Lock ditolak bila ada unfinished Task yang unscheduled.
- Close confirmation menjelaskan bahwa semua Task harus mempunyai complete
  Actual Date pair.
- Reopen confirmation membedakan `Locked → Open` dan `Closed → Open`; keduanya
  menjelaskan bahwa Project kembali editable dan eligible untuk scheduling.
  Locked Reopen juga menjelaskan bahwa unfinished work dalam transitive impacted
  scope akan dihitung ulang.
- Locked Project tidak berubah menjadi Open secara implicit. User harus memilih
  Reopen Project sebelum planning mutation selain Project Name rename.
- Dialog dan overflow menu mengelola focus dan mengembalikannya ke trigger.
- Core workflow tetap usable pada supported viewport tanpa horizontal scrolling
  pada normal page content.

---

## 6. Domain Model

### Project

| Field                 | Type                        | Required | Source              | Rules                                                                                                      |
| --------------------- | --------------------------- | -------: | ------------------- | ---------------------------------------------------------------------------------------------------------- |
| ID                    | System-generated identifier |       Ya | System              | Dibuat otomatis dan immutable                                                                              |
| Name                  | String                      |       Ya | User                | Trimmed; non-blank; maksimum 100 karakter; unique case-insensitive                                         |
| Status                | `open \| locked \| closed`  |       Ya | System/transition   | Default `open`; hanya berubah melalui lifecycle command                                                    |
| Start Date            | Nullable date-only          |    Tidak | System-derived      | Min Start Date descendant leaf; `null` bila belum ada leaf/date                                            |
| End Date              | Nullable date-only          |    Tidak | System-derived      | Max End Date descendant leaf; `null` bila belum ada leaf/date                                              |
| Automatic Scheduling  | Boolean                     |       Ya | User/default        | Default `true`; configured through US-3.3 combined Add/Edit Project form                                   |
| Scheduling Start Date | Nullable date-only          |    Tidak | User                | Project-level scheduler anchor; SQL `DATE`, API `YYYY-MM-DD`                                               |
| Project Buffer        | Decimal percentage          |       Ya | User/default        | Default `20`; configured through US-3.3                                                                    |
| Project Priority      | Positive integer position   |       Ya | System/User command | Unique; create menggunakan global `MAX(priority)+1`; Move Up/Down menukar position dengan active neighbour |
| Closed At             | Nullable DateTime           |    Tidak | System              | Diisi saat successful close; retained untuk audit dan Closed ordering                                      |
| Created At            | DateTime                    |       Ya | System              | Dibuat otomatis dan immutable                                                                              |
| Updated At            | DateTime                    |       Ya | System              | Diperbarui setelah confirmed mutation                                                                      |

Project Priority dicantumkan karena active ordering membutuhkannya. Closed At
adalah additional system field untuk audit dan deterministic Closed ordering.
Project adalah WBS level `0`; creation tidak membuat root WBS record terpisah.

Locked Project membutuhkan persisted locked Execution dan Commitment timeline
values atau immutable schedule snapshot yang ekuivalen. Nilai tersebut adalah
state pendukung lifecycle, bukan manual date input. Representasi database dan
granularity snapshot ditentukan saat contract timeline/WBS tersedia.

---

## 7. Project Status Model

| Status | Meaning | Editable | Scheduling | Gantt | Timeline behaviour |
| --- | --- | --- | --- | --- | --- |
| Open | Active planning | Project/Task/WBS/dependency subject to domain rules | Included | Visible | Execution and Commitment may recalculate |
| Locked | Protected planning baseline with ongoing actual completion capture | Project Name and Actual Date on unfinished Task only; planning/settings read-only | Locked scheduling state immutable; Actual Date may recalculate impacted Open Projects | Visible | Execution/Commitment baseline immutable; Actual Allocation historical; Forecast deferred |
| Closed | Completed historical Project | No | Excluded | Hidden | All planning, actual, and timeline data read-only |

- Status saat create selalu `open`, ditegakkan backend.
- Tidak ada status lain.
- Status bukan free-form value.
- Status transition merupakan business command, bukan generic field update.

---

## 8. Status Transition Rules

| From | To | Allowed | Conditions and effects |
| --- | --- | ---: | --- |
| Open | Locked | Ya | At least one Executable Task; every unfinished Task fully scheduled; protect current Execution/Commitment timeline and allocation; no scheduler run |
| Locked | Open | Ya | Explicit Reopen; calculate Required Locked Reopen Closure, offer atomic Reopen All when needed, then recalculate transitive impacted Open scope |
| Open | Closed | Ya | Every descendant Executable Task has complete Actual Date |
| Locked | Closed | Ya | Full close validation; locked baselines remain unchanged |
| Closed | Open | Ya | Reactivate Project and return it to editable/scheduling eligibility |
| Closed | Locked | Tidak | Reject without partial mutation |

- Invalid status value ditolak.
- Same-status atau transition selain tabel ditolak unless separately approved.
- Existing status dan data dipertahankan bila transition gagal.
- Lock, close, Locked/Closed Reopen, dan Priority change harus transactional.
- Locked Reopen may produce valid unscheduled results; this does not fail the transition, but the Project cannot be Locked again until every unfinished Task is scheduled.
- Corruption, concurrency conflict, persistence failure, or internal scheduler defect during Locked Reopen rolls back the entire transition and leaves the Project Locked.

---

## 9. Timeline Behaviour by Status

### Open

- Execution and Commitment may be updated by approved scheduling triggers.
- Entering Actual Date actualizes completed Task timelines and creates Actual Allocation according to US-6.2.
- Recalculation is limited to the transitive impacted scheduling scope, not unrelated active Projects.
- Project ikut scheduling dan Gantt.

### Locked

- Current Execution/Commitment timelines and daily allocations are protected baselines.
- Execution/Commitment scheduler is not run for Locked Project mutations.
- Actual Date may be recorded for unfinished Task. Protected Execution/Commitment baseline, order, and dependencies remain unchanged; Actual Allocation may recalculate impacted Open Projects.
- All planning, WBS, Task, Settings, and dependency changes require explicit `Locked → Open` Reopen first.
- Project Name may be renamed while Locked because it is identity metadata, not a scheduling input; the rename changes no protected baseline, allocation, dependency, Priority, or status.
- Completed Task cannot be reopened while Locked.
- Project remains visible in Gantt and may be read as an immutable cross-project scheduling anchor.
- Forecast, Delivery Impact, and Health behaviour while Locked is deferred.

### Closed

- Project tidak ikut scheduling atau Project Priority allocation.
- Project tidak tampil di Gantt.
- Seluruh planning, Actual Date, dan timeline read-only.
- Project tetap tersedia sebagai historical record pada Project List.

---

## 10. Editing Rules by Status

### Open Project

- Project fields and planning data may change subject to field-specific rules.
- Completed Task remains read-only except dedicated Task Reopen while the Project is Open.

### Locked Project

- Project Name is editable and remains subject to the same trim, length, and
  case-insensitive uniqueness validation.
- Project Name rename is persisted without scheduler invocation or generic
  scheduling-impact preview because it cannot change timeline, allocation,
  dependency validity, Priority, or capacity.
- Project Settings, WBS structure/order, Task planning fields,
  Execution/Commitment dates, and dependency ownership/endpoints are read-only.
- Complete Actual Date is the only Task mutation allowed. It does not change the
  locked baseline, but its Actual Allocation may trigger recalculation of
  impacted Open Projects.
- Reopen completed Task is rejected until the Project is explicitly reopened to
  Open.
- Priority may change only when internal simulation proves no Locked Project
  timeline, allocation, or dependency-validity impact.
- No planning mutation implicitly changes status to Open.

### Closed Project

- Project, WBS, Task, Actual Date, planning fields, and timeline are read-only.
- Backend rejects mutation even when frontend restrictions are bypassed.
- Project must be reopened to Open before any mutation.

---

## 11. Lock and Closed Validation

- Locking adalah hard validation, bukan scheduler trigger.
- Project may Lock only when it has at least one Executable Task and every unfinished Task has complete Execution and Commitment pairs with no unscheduled reason.
- One unscheduled unfinished Task rejects the entire Lock operation with `PROJECT_CANNOT_LOCK_WITH_UNSCHEDULED_TASKS`; status remains Open.
- Closing adalah hard validation, bukan warning.
- Project boleh Closed jika dan hanya jika setiap descendant Executable Leaf
  mempunyai complete Actual Date pair.
- Complete Actual Date pair adalah source of truth untuk completion; task status
  terpisah tidak digunakan.
- Grouping WBS tidak dievaluasi langsung dan tidak membutuhkan Actual Date.
- Traversal mencakup nested grouping sedalam apa pun.
- Satu unfinished descendant menyebabkan seluruh transition ditolak.
- Existing status, baselines, dan planning data tetap utuh pada failure.
- Project dengan zero Executable Leaves tidak boleh diubah menjadi Closed dan
  tidak boleh diubah menjadi Locked; Project tersebut dapat dihapus.

---

## 12. Project List Behaviour

- Backend pagination wajib; default page `1`, default page size `5`, maximum
  page size `100`.
- Response menyediakan `page`, `pageSize`, dan `total` matching records.
- Search berdasarkan Name bersifat case-insensitive dan menggunakan indexed
  prefix semantics yang sudah dipakai repository (`normalized-name%`).
- Search dijalankan backend sebelum count, ordering, limit, dan offset.
- Search berubah atau di-clear mereset page ke `1`.
- Normal empty state berbeda dari no-results dan load-error state.
- Project Priority merupakan primary ordering criterion untuk active Projects.
- Priority disimpan sebagai unique positive integer. Create memakai global
  `MAX(priority)+1`; Move Up/Down menukar position secara atomic dengan active
  neighbour. Closed Project tidak menjadi neighbour karena diabaikan active
  allocation.
- Open dan Locked Project digabung dalam active ordering; status tidak
  memisahkan keduanya sebelum Priority.
- Semua active Project ditampilkan sebelum Closed Project.
- Closed Project tidak termasuk active priority allocation.
- Closed Project diurutkan `closed_at DESC, id ASC`.
- Duplicate active Project Priority adalah invalid persisted state. List dan
  Scheduling Engine tidak mengarang tie-breaker dari Start Date, End Date, status,
  Name, Created At, atau Project ID.
- Filter/search/pagination menjadi request-cache identity.
- Priority Move Up/Down uses an internal scheduling simulation. When Locked Projects exist, Save succeeds only if no Locked timeline, allocation, or cross-project dependency validity changes.
- Priority scheduling/recalculation is limited to the transitive impacted scope; unrelated Projects are not recalculated or version-updated.
- Confirmed mutation memperbarui affected list/detail serta scheduling/Gantt projection tanpa hard refresh.
- Versioned invalidation mencegah older in-flight response mengembalikan stale
  data.
- Current search dan page dipertahankan setelah mutation bila masih valid.
- Confirmed delete pada childless Project mengikuti current-page correction
  pattern bila item terakhir membuat page tidak valid.

---

## 13. Gantt Visibility

- Open Project terlihat di Gantt.
- Locked Project terlihat di Gantt.
- Closed Project tidak terlihat di Gantt.
- Closed Project tidak mengonsumsi scheduling capacity.
- Closed Project tidak memengaruhi active Project ordering.
- Reopened Closed Project langsung kembali eligible untuk Gantt dan scheduling,
  bahkan sebelum recalculation berikutnya selesai.
- Tidak ada `Show Closed Projects` toggle.
- Historical Gantt untuk Closed Project out of scope.

---

## 14. Acceptance Criteria

### AC-1 — Projects and Home navigation

**Given** application shell tersedia
**When** Engineering Lead membuka group Project
**Then** menu Projects membuka page Projects
**And** Home remains the canonical active-WBS surface
**And** the standalone Project Structure entry point is absent
**And** global chrome tetap stabil.

### AC-2 — Initial list, pagination, dan ordering

**Given** Projects tersedia
**When** list dimuat
**Then** maksimal lima records dan pagination metadata ditampilkan
**And** active Projects muncul sebelum Closed Projects
**And** Open dan Locked Project diurutkan berdasarkan Project Priority lebih
dulu, bukan dikelompokkan berdasarkan status.

### AC-3 — Search

**When** user mencari Name dengan perbedaan casing
**Then** indexed prefix search dijalankan backend sebelum pagination
**And** perubahan atau Clear search mereset page ke 1.

### AC-4 — Loading, empty, no-results, dan failure

**Then** initial request menampilkan local shape-preserving skeleton
**And** empty, no-results, dan load-error merupakan state berbeda
**And** load failure menyediakan Retry.

### AC-5 — Add Project form

**When** Add Project dipilih
**Then** labeled fields yang telah dikunci ditampilkan
**And** user tidak diwajibkan memilih Open
**And** Cancel tidak mengirim mutation.

### AC-6 — Valid creation dan backend default

**When** valid creation data disimpan
**Then** backend membuat ID, `status: open`, Created At, dan Updated At
**And** tidak membuat task atau dependency secara otomatis
**And** Project menjadi WBS level 0 tanpa membuat root WBS record terpisah
**And** Project tampil pada active list tanpa hard refresh.

### AC-7 — Validation, failure recovery, dan duplicate submission

**When** frontend validation dilewati atau dependency failure terjadi
**Then** backend tetap memvalidasi request dan mengembalikan structured error
**And** draft dipertahankan serta recovery action tersedia
**And** repeated submit selama request berjalan tidak mengirim mutation kedua.

### AC-8 — Open behaviour

**Given** Project berstatus Open
**Then** established planning data editable
**And** Project ikut scheduling serta terlihat di Gantt
**And** ketika Scheduling Engine dijalankan oleh Priority Move Up/Down,
Execution, Commitment, dan Forecast Timeline boleh diperbarui.

### AC-9 — Open to Locked eligibility

**Given** Project mempunyai sedikitnya satu Executable Task dan seluruh unfinished Task fully scheduled
**When** Engineering Lead mengonfirmasi Lock Project
**Then** status menjadi Locked
**And** current Execution/Commitment timelines and allocations become protected baselines
**And** scheduler is not run.

**Given** Project tidak mempunyai Executable Task
**Then** Lock ditolak dengan `PROJECT_CANNOT_LOCK_WITHOUT_TASKS`.

**Given** satu unfinished Task mempunyai incomplete timeline atau unscheduled reason
**Then** Lock ditolak dengan `PROJECT_CANNOT_LOCK_WITH_UNSCHEDULED_TASKS`
**And** status tetap Open.

### AC-10 — Locked baseline and Actual Date exception

**Given** Project Locked
**When** complete Actual Date disimpan pada unfinished Task
**Then** Task menjadi completed
**And** protected Execution/Commitment baseline remains unchanged
**And** Actual Allocation is persisted
**And** grouped impact warning/confirmation applies when other Projects are impacted
**And** impacted Open Projects may be recalculated while every Locked Project remains unchanged.

### AC-11 — Locked planning is read-only with Name exception

**Given** Project Locked
**When** Project Name is changed validly
**Then** rename succeeds
**And** status, Priority, Settings, WBS, timelines, allocations, dependencies,
and schedule version remain unchanged
**And** no scheduler or impact preview is invoked.

**When** planning, WBS, Task, Settings, dependency, timeline, atau Task Reopen
mutation selain Project Name diminta
**Then** request ditolak dengan `PROJECT_LOCKED_READ_ONLY`
**And** confirmed state remains unchanged.

### AC-12 — Locked to Open Reopen

**Given** Project Locked
**When** Engineering Lead requests Reopen Project
**Then** server calculates the Required Locked Reopen Closure
**And** UI lists every Locked Project that must be reopened together plus impacted Open Projects
**And** Reopen All changes the complete closure to Open atomically
**And** all unfinished Tasks in the transitive impacted scheduling scope are recalculated
**And** valid unscheduled results do not fail Reopen.

### AC-13 — Priority change respects Locked Projects

**When** Priority Project dipindahkan Up atau Down
**Then** proposed priority is simulated atomically
**And** only the transitive impacted scope may be recalculated
**And** Save succeeds only when every Locked Project timeline, allocation, and dependency validity remains unchanged.

**Given** proposed priority impacts a Locked Project
**Then** grouped Locked/Open Project names are returned
**And** the entire operation is rejected with `SCHEDULING_LOCKED_PROJECT_IMPACT`
**And** priorities and schedules remain unchanged.

### AC-14 — Locked Reopen failure rollback

**Given** Locked→Open recalculation fails due to corrupted state, concurrency, persistence, or internal scheduler defect
**Then** status remains Locked
**And** no partial timeline, allocation, dependency, or cache state is persisted.

### AC-15 — Close Open or Locked Project

**Given** seluruh descendant Executable Leaf mempunyai complete Actual Date
**When** Close Project dikonfirmasi dari Open atau Locked
**Then** status menjadi Closed secara atomic
**And** Grouping WBS tidak memerlukan Actual Date.

### AC-16 — Reject incomplete close

**Given** satu direct atau nested descendant Executable Leaf tidak mempunyai
complete Actual Date
**When** close diminta
**Then** transition ditolak dengan
`PROJECT_CANNOT_CLOSE_WITH_ACTIVE_TASKS`
**And** existing status dan data tetap utuh.

**Given** Project tidak mempunyai Executable Leaf
**When** close diminta
**Then** transition ditolak dengan `PROJECT_CANNOT_CLOSE_WITHOUT_TASKS`
**And** user dapat memilih Delete Project sebagai gantinya.

### AC-17 — Closed behaviour

**Given** Project Closed
**Then** Project dan semua planning data read-only
**And** backend menolak mutation dengan `PROJECT_CLOSED_READ_ONLY`
**And** Project tidak ikut scheduling atau Gantt
**And** Project tetap tampil setelah seluruh active Project pada list.

### AC-18 — Reopen Closed

**When** Engineering Lead mengonfirmasi Reopen Project
**Then** Closed Project berubah langsung ke Open
**And** previous locked baselines retained
**And** langsung kembali editable, eligible untuk scheduling, dan visible in
Gantt sebelum recalculation berikutnya.

### AC-19 — Invalid status and transition

**When** status value tidak dikenal atau Closed Project diminta langsung menjadi
Locked
**Then** request ditolak dengan structured business error
**And** confirmed state tidak berubah.

### AC-20 — Not found

**When** get, update, atau status command menggunakan Project ID yang tidak ada
**Then** API mengembalikan `404 PROJECT_NOT_FOUND`.

### AC-21 — Cache and downstream consistency

**When** Project mutation berhasil
**Then** affected Project list/detail dan available scheduling/Gantt projections
diinvalidasi atau diperbarui
**And** stale in-flight response tidak dapat merepopulasi invalid data.

### AC-22 — Accessibility and responsive behaviour

**Then** status tidak dikomunikasikan hanya dengan warna
**And** forms, warning, confirmation, pagination, dan navigation keyboard
accessible dengan focus management yang benar
**And** workflow usable pada supported viewport tanpa normal horizontal scroll.

### AC-23 — Concurrency and rollback

**When** concurrent status commands atau dependency failure terjadi
**Then** lifecycle invariants tetap terjaga
**And** lock, close, Closed reopen, serta Priority swap tidak meninggalkan partial
state.

### AC-24 — Conditional Project deletion

**Given** Project tidak mempunyai child WBS atau Executable Leaf
**When** Engineering Lead mengonfirmasi Delete Project
**Then** Project dihapus dan list diperbarui tanpa hard refresh.

**Given** Project mempunyai sedikitnya satu child
**When** delete diminta melalui API
**Then** request ditolak dengan `PROJECT_HAS_CHILDREN`
**And** Project tetap tersedia; lifecycle penyelesaian menggunakan Closed.

### AC-25 — Edit Project whole-Project summary and width

**Given** Project adalah logical WBS level `0`
**When** Engineering Lead membuka Edit Project
**Then** form composes the read-only whole-Project summary owned by US-4.3 from every confirmed Task in the Project
**And** summary is absent from Add Project
**And** Edit Project uses the existing shared wide Dialog variant while preserving responsive viewport behaviour
**And** summary loading/failure does not change Project lifecycle, validation, or mutation rules.

### AC-26 — Shared Edit Project from Home

**Given** an active Project row is visible on Home
**When** Engineering Lead activates Project Name
**Then** the shared Edit Project dialog opens directly over Home
**And** Save or Close leaves Home as the route and visible background
**And** Locked Project permits only Project Name mutation.

### AC-27 — Lifecycle parity on two surfaces

**Given** a lifecycle command is eligible for an active Project
**When** it is invoked from Home row overflow or Projects
**Then** both surfaces use the same confirmation, application command, impact
handling, transaction, error mapping, and refresh contract.

---

## 15. API Contract

Existing repository menggunakan plural REST resources, structured JSON errors,
`page/pageSize/total`, dan explicit DTO mapping. Status transition dipilih
sebagai command endpoint tunggal karena status mempunyai invariant dan side
effects yang tidak boleh diperlakukan sebagai generic update field.

### List Projects

```http
GET /api/projects?search=alpha&page=1&pageSize=5
```

```json
{
  "data": [
    {
      "id": "project-id",
      "name": "Alpha",
      "status": "open",
      "startDate": "2026-08-01",
      "endDate": "2026-09-30",
      "automaticScheduling": true,
      "schedulingStartDate": "2026-08-01",
      "projectBuffer": 20,
      "projectPriority": 1,
      "closedAt": null,
      "createdAt": "2026-07-26T10:00:00Z",
      "updatedAt": "2026-07-26T10:00:00Z"
    }
  ],
  "page": 1,
  "pageSize": 5,
  "total": 1
}
```

`projectPriority` adalah positive integer position. Nilai yang lebih kecil
berarti prioritas lebih tinggi.

### Get Project

```http
GET /api/projects/{projectId}
```

Success: `200 OK` with `{ "data": Project }`.

### Create Project

```http
POST /api/projects
Content-Type: application/json
```

```json
{
  "name": "Alpha",
  "automaticScheduling": true,
  "schedulingStartDate": "2026-08-01",
  "projectBuffer": 20
}
```

Backend assigns `status: open`, lowest Project Priority, `startDate: null`,
`endDate: null`, default Project Settings from US-3.3 when omitted, ID, and timestamps. Request must
not accept system-derived fields or manually supplied locked baselines.

Success: `201 Created` with `{ "data": Project }`.

### Update Project

- Open Project may update fields allowed by US-3.3.
- Locked Project update accepts Project Name only; any Settings/scheduling field
  mutation returns `PROJECT_LOCKED_READ_ONLY`.
- Locked Name-only update uses the normal Project name validation and does not
  invoke scheduling impact preview.

```http
PUT /api/projects/{projectId}
Content-Type: application/json
```

The current Project-level writable DTO contains only `name`. Derived dates,
Priority, status, Closed At, timestamps, and locked baselines are not accepted.
`automaticScheduling`, `schedulingStartDate`, and `projectBuffer` follow the
combined Project form contract from US-3.3. Status changes use the command endpoint. Closed Project
updates return `409 PROJECT_CLOSED_READ_ONLY`.

### Change Project Status

```http
POST /api/projects/{projectId}/status
Content-Type: application/json
```

```json
{
  "status": "locked"
}
```

- `locked`, `open`, dan `closed` are the only accepted targets.
- Success: `200 OK` with the confirmed Project.
- Open to Locked validates that every unfinished Task is scheduled, captures/protects baselines, and does not run scheduler.
- Open to Locked rejects zero-leaf Project with `409 PROJECT_CANNOT_LOCK_WITHOUT_TASKS`.
- Open to Locked rejects any unscheduled unfinished Task with `409 PROJECT_CANNOT_LOCK_WITH_UNSCHEDULED_TASKS`.
- Locked to Open is explicit Reopen. Required mutually/transitively related Locked Projects are offered as one atomic Reopen All closure, then the transitive impacted Open scope is recalculated.
- Open/Locked to Closed runs descendant completion validation.
- Closed to Open reactivates the Project.
- Closed to Locked returns `409 PROJECT_STATUS_TRANSITION_NOT_ALLOWED`.

### Change Project Priority

```http
POST /api/projects/{projectId}/priority
Content-Type: application/json
```

```json
{
  "direction": "up"
}
```

- Command hanya tersedia untuk Open atau Locked Project dan `direction` hanya
  menerima `up` atau `down`.
- Success menukar Priority Project dengan active neighbour secara atomic dan
  mengembalikan confirmed Project.
- Move yang tidak mempunyai active neighbour pada arah tersebut ditolak dengan
  `409 PROJECT_PRIORITY_MOVE_NOT_ALLOWED`.
- Successful move simulates only the transitive impacted scheduling scope and requires grouped Project-name confirmation when other Open Projects are affected.
- Locked Projects are immutable anchors and receive no timeline/allocation/dependency mutation.
- If proposed priority would impact any Locked Project, grouped Locked/Open names are returned and the command returns `409 SCHEDULING_LOCKED_PROJECT_IMPACT`.
- Priority swap and affected Open scheduling must be atomic; failure leaves all priorities and schedules unchanged.

### Pagination Validation

- `page` default `1` and must be greater than `0`.
- `pageSize` default `5` and must be `1..100`.
- Invalid values are rejected before repository query.

### Delete Childless Project

```http
DELETE /api/projects/{projectId}
```

- Success for a Project with zero children: `204 No Content`.
- A Project with any child returns `409 PROJECT_HAS_CHILDREN`.
- Delete is hard delete because no planning history exists below the Project.

---

## 16. Error Response Contract

```json
{
  "code": "PROJECT_STATUS_TRANSITION_NOT_ALLOWED",
  "message": "Project status transition is not allowed",
  "field": "status"
}
```

| Error Code                               | HTTP | Field       | Condition                                                                |
| ---------------------------------------- | ---: | ----------- | ------------------------------------------------------------------------ |
| `INVALID_REQUEST`                        |  400 | —           | Malformed JSON, unknown field, atau multiple payloads                    |
| `PROJECT_NAME_REQUIRED`                  |  400 | `name`      | Trimmed Name kosong                                                      |
| `PROJECT_NAME_TOO_LONG`                  |  400 | `name`      | Trimmed Name lebih dari 100 karakter                                     |
| `PROJECT_NAME_ALREADY_EXISTS`            |  409 | `name`      | Name digunakan Project lain secara case-insensitive                      |
| `PROJECT_NOT_FOUND`                      |  404 | —           | Project ID tidak tersedia                                                |
| `PROJECT_STATUS_INVALID`                 |  400 | `status`    | Target bukan open, locked, atau closed                                   |
| `PROJECT_STATUS_TRANSITION_NOT_ALLOWED`  |  409 | `status`    | Transition tidak diizinkan, termasuk Closed ke Locked                    |
| `PROJECT_CANNOT_LOCK_WITHOUT_TASKS`      |  409 | `status`    | Lock diminta untuk Project tanpa Executable Leaf                         |
| `PROJECT_CANNOT_LOCK_WITH_UNSCHEDULED_TASKS` | 409 | `status` | Sedikitnya satu unfinished Task belum fully scheduled                    |
| `PROJECT_LOCKED_READ_ONLY`               |  409 | —           | Mutation selain Project Name, allowed Actual Date, atau eligible Priority command pada Locked Project |
| `SCHEDULING_LOCKED_PROJECT_IMPACT`         |  409 | `direction` | Proposed priority would affect a Locked Project timeline, allocation, or dependency validity |
| `SCHEDULING_IMPACT_CONFIRMATION_REQUIRED` | 409 | — | Priority affects other Open Projects and requires confirmation |
| `SCHEDULING_IMPACT_STALE` | 409 | — | Impact set/version changed after preview |
| `PROJECT_BULK_REOPEN_REQUIRED` | 409 | `status` | Project Reopen requires multiple Locked Projects to reopen together |
| `PROJECT_CANNOT_CLOSE_WITH_ACTIVE_TASKS` |  409 | `status`    | Sedikitnya satu descendant Executable Leaf tidak mempunyai complete Actual Date |
| `PROJECT_CANNOT_CLOSE_WITHOUT_TASKS`     |  409 | `status`    | Close diminta untuk Project tanpa Executable Leaf                        |
| `PROJECT_CLOSED_READ_ONLY`               |  409 | —           | Mutation planning pada Closed Project                                    |
| `COMPLETED_TASK_READ_ONLY`               |  409 | —           | Mutation pada Executable Leaf yang mempunyai complete Actual Date        |
| `PROJECT_PRIORITY_DIRECTION_INVALID`     |  400 | `direction` | Direction bukan up atau down                                             |
| `PROJECT_PRIORITY_MOVE_NOT_ALLOWED`      |  409 | `direction` | Tidak ada active neighbour pada arah yang diminta                        |
| `PROJECT_HAS_CHILDREN`                   |  409 | —           | Delete diminta untuk Project yang mempunyai child                        |
| `INVALID_PAGE`                           |  400 | `page`      | Page bukan positive integer                                              |
| `INVALID_PAGE_SIZE`                      |  400 | `pageSize`  | Page size di luar 1..100                                                 |

Field-specific date/configuration errors tidak diperlukan karena values tersebut
system-derived. Internal error tidak
mengekspos stack trace, SQL, database, atau infrastructure detail.

---

## 17. Test Cases

| ID    | Scenario                                       | Expected                                                                                       |
| ----- | ---------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| TC-1  | Buka Project > Projects                        | Page berada di shared shell; menu dan breadcrumb aktif                                         |
| TC-2  | Initial list pending                           | Local skeleton; empty/no-results belum tampil                                                  |
| TC-3  | Empty successful list                          | Empty state dan Add Project action                                                             |
| TC-4  | List failure lalu Retry                        | Actionable error, kemudian list pulih                                                          |
| TC-5  | Search Name beda casing                        | Backend prefix search match                                                                    |
| TC-6  | Search tanpa match lalu Clear                  | No-results berbeda; Clear kembali ke page 1                                                    |
| TC-7  | Lebih dari lima Projects                       | Backend pagination dan metadata benar                                                          |
| TC-8  | Active ordering                                | Open/Locked mengikuti Priority tanpa status grouping                                           |
| TC-9  | Closed ordering group                          | Semua Closed berada setelah seluruh active Projects                                            |
| TC-10 | Buka/Cancel Add form                           | Labeled approved fields; no mutation on Cancel                                                 |
| TC-11 | Create valid Project                           | `201`; generated ID/timestamps; backend status Open                                            |
| TC-12 | Create success                                 | Project terlihat tanpa hard refresh; relevant caches consistent                                |
| TC-13 | Create invalid via direct API                  | Structured validation; repository not mutated                                                  |
| TC-14 | Create dependency failure                      | Draft preserved; retry available; no partial data                                              |
| TC-15 | Repeated create submit                         | Satu request                                                                                   |
| TC-16 | Open Project contract                          | Editable, scheduled, dan visible in Gantt                                                      |
| TC-17 | Lock fully scheduled Project                  | Status Locked; baselines/allocations protected; no scheduler run                               |
| TC-18 | Lock with one unscheduled unfinished Task       | `409 PROJECT_CANNOT_LOCK_WITH_UNSCHEDULED_TASKS`; status Open                                   |
| TC-19 | Actual Date on unfinished Locked Task           | Actual Date and Actual Allocation saved; baseline unchanged; impacted Open scope recalculated    |
| TC-20 | Edit/create/delete/move Task while Locked       | `409 PROJECT_LOCKED_READ_ONLY`; state unchanged                                                |
| TC-20A | Rename Locked Project Name                       | Success; no scheduler/impact; baseline and settings unchanged                                  |
| TC-20B | Update Locked Project Settings with Name payload | Entire request rejected; no partial Name or Settings change                                    |
| TC-21 | Reopen completed Task while Locked              | `409 PROJECT_LOCKED_READ_ONLY`; Project Reopen required                                        |
| TC-22 | Locked to Open                                  | Status Open; transitive impacted scope recalculated                                             |
| TC-23 | Locked Reopen produces unscheduled Task         | Reopen succeeds; Project Open; later Lock rejected until scheduled                              |
| TC-24 | Locked Reopen technical failure                | Entire operation rollback; Project remains Locked                                               |
| TC-25 | Priority change with no Locked impact           | Atomic swap; affected Open scope recalculated; Locked state unchanged                           |
| TC-26 | Priority change impacts Locked Project          | `409 SCHEDULING_LOCKED_PROJECT_IMPACT`; priorities and schedules unchanged                        |
| TC-27 | Close Open; all leaves have Actual Date        | Closed succeeds                                                                                |
| TC-28 | Close Locked; all leaves have Actual Date      | Closed succeeds                                                                                |
| TC-29 | One leaf lacks complete Actual Date           | Close rejected; state preserved                                                                |
| TC-30 | Nested unfinished descendant                   | Close rejected after descendant traversal                                                      |
| TC-31 | Grouping WBS dan completed descendant leaf     | Group tidak butuh Actual Date; close succeeds                                                  |
| TC-32 | Closed Project mutation                        | Backend rejects read-only violation                                                            |
| TC-33 | Closed scheduling/Gantt                        | Excluded from both but retained in Project List                                                |
| TC-34 | Closed to Open                                 | Transition succeeds and Project becomes active/editable                                        |
| TC-35 | Closed to Locked                               | Structured invalid-transition conflict                                                         |
| TC-36 | Unknown status                                 | `400 PROJECT_STATUS_INVALID`                                                                   |
| TC-37 | Unknown Project ID                             | `404 PROJECT_NOT_FOUND`                                                                        |
| TC-38 | Lock dependency failure                        | Transaction rollback                                                                           |
| TC-39 | Close dependency failure                       | Transaction rollback                                                                           |
| TC-40 | Reopen dependency failure                      | Transaction rollback                                                                           |
| TC-41 | Priority swap or scheduling dependency failure | Priority and timeline transaction/coordination rollback                                        |
| TC-42 | Concurrent incompatible transitions            | At most one valid confirmed state; invariants hold                                             |
| TC-43 | Mutation with active search/page               | Query state preserved and affected caches invalidated                                          |
| TC-44 | Stale response completes after mutation        | Stale data cannot repopulate cache                                                             |
| TC-45 | Keyboard and assistive technology              | Controls, warnings, status, and focus accessible                                               |
| TC-46 | Supported viewport                             | Workflow usable without normal horizontal scrolling                                            |
| TC-47 | Delete Project tanpa child                     | Confirmed hard delete; list/page corrected                                                     |
| TC-48 | Delete Project dengan child                    | `409 PROJECT_HAS_CHILDREN`; data tetap utuh                                                    |
| TC-49 | Close Project tanpa Executable Leaf            | `409 PROJECT_CANNOT_CLOSE_WITHOUT_TASKS`; Delete tersedia                                      |
| TC-50 | Lock Project tanpa Executable Leaf             | `409 PROJECT_CANNOT_LOCK_WITHOUT_TASKS`; status tetap Open                                     |
| TC-51 | Open Edit Project from Home                       | Shared dialog over Home; no Projects/Project Structure route                                   |
| TC-52 | Lifecycle command from Home and Projects          | Same command/confirmation/result and consistent refresh                                        |

### Acceptance Criteria Traceability

| Acceptance Criteria | Test Case(s)       |
| ------------------- | ------------------ |
| AC-1                | TC-1               |
| AC-2                | TC-7—TC-9          |
| AC-3                | TC-5—TC-6, TC-43   |
| AC-4                | TC-2—TC-4          |
| AC-5—AC-7           | TC-10—TC-15        |
| AC-8                | TC-16              |
| AC-9—AC-10          | TC-17—TC-19, TC-50 |
| AC-11               | TC-20—TC-21, TC-20A—TC-20B |
| AC-12—AC-14         | TC-22—TC-26        |
| AC-15—AC-16         | TC-27—TC-31, TC-49 |
| AC-17               | TC-32—TC-33        |
| AC-18—AC-20         | TC-34—TC-37        |
| AC-21               | TC-43—TC-44        |
| AC-22               | TC-45—TC-46        |
| AC-23               | TC-38—TC-42        |
| AC-24               | TC-47—TC-48        |
| AC-25               | Project summary acceptance workflow |
| AC-26               | TC-51              |
| AC-27               | TC-52              |

---

## 18. Required Automated Tests

### Domain Tests

- Project creation and backend-enforced default Open status.
- Valid status values, allowed transitions, and invalid transitions.
- Identity/Created At preservation and Updated At changes.
- Closed read-only invariant.
- Completed Executable Leaf read-only invariant using complete Actual Date.
- Close eligibility across all descendant Executable Leaves.
- Grouping WBS exclusion from completion validation.
- Zero-leaf close rejection and childless delete eligibility.
- Zero-leaf lock rejection.
- Locked baseline invariant and Actual Date/Actual Allocation exception.
- Lock rejection when any unfinished Task is unscheduled.
- Locked Project Name rename success without scheduling mutation.
- Locked planning mutation and Task Reopen rejection.
- Locked-to-Open transition, unscheduled-result success, and technical-failure rollback.
- Positive integer Priority, valid/invalid move rules, and Locked-impact validation.

### Application Tests

- Paginated list, Name search, create, get, Open update, and Locked Name-only update.
- Lock, Locked-to-Open, Open-to-Closed, Locked-to-Closed, and Closed-to-Open.
- Reject Open-to-Locked when the Project has zero Executable Leaves or any unscheduled unfinished Task.
- Reject Closed-to-Locked, Closed mutation, completed-leaf normal mutation,
  Locked mutation other than Name/Actual Date/eligible Priority, and close with
  unfinished leaves.
- Delete childless Project and reject delete when any child exists.
- Preserve Locked baselines on Actual Date entry, persist Actual Allocation, recalculate impacted Open Projects, and reject Task Reopen while Locked.
- Atomic Priority Move Up/Down with transitive impacted-scope recalculation and Locked-impact rejection.
- Dependency failure rollback and concurrent transition behaviour.
- Scheduling/Gantt visibility and downstream invalidation contracts.

### Repository Integration Tests

- Persist/retrieve Project, default status, and transitions.
- Persisted baseline snapshot or equivalent protection.
- Descendant leaf completion query excluding Grouping WBS.
- Indexed Name prefix search before count/pagination.
- Pagination metadata.
- Active-before-Closed grouping.
- Active Project Priority ordering and duplicate-priority data-integrity rejection.
- Atomic active-neighbour Priority swap and rollback.
- Closed ordering by Closed At DESC then ID ASC.
- Transaction rollback and concurrent transition protection.
- Query plans and indexes for search, status, Priority, active/Closed ordering,
  and completion validation.
- SQLite isolation for automated tests, PostgreSQL verification, dan future
  MySQL-compatible contract.

### API Integration Tests

- List, search, pagination metadata, get, create, dan update.
- Lock, Locked/Closed reopen, close, invalid status, invalid transition, and not-found.
- Zero-leaf and unscheduled-Task Lock conflict mapping.
- Close rejection, Closed read-only, completed-task read-only, and Locked planning read-only.
- Locked Actual Date exception, generic impact guard, bulk Reopen closure, and Priority/capacity Locked-impact contracts.
- Structured error mapping, persistence, rollback, and unknown-field rejection.
- Conditional Project delete success and `PROJECT_HAS_CHILDREN` conflict.

### Frontend Tests

- Projects navigation, Home Project-row entry, removed Project Structure
  entry, stable shell, initial loading, empty, error/Retry, no-results/Clear,
  and pagination.
- Add form, valid create, backend default Open display, failure draft
  preservation, and duplicate prevention.
- Conditional delete confirmation, success, rejection, and page correction.
- Status labels and available actions for Open, Locked, dan Closed on Projects,
  plus eligible active lifecycle actions on Home.
- Lock/close/reopen confirmations and close rejection.
- Zero-leaf Lock rejection and actionable recovery message.
- Closed read-only presentation.
- Locked Project Name is editable while Settings/planning remain read-only and
  complete Actual Date remains available through Task.
- Locked Project Reopen, unscheduled-result feedback, and rollback recovery.
- Priority Move Up/Down availability, no-impact success, Locked-impact rejection, scheduling feedback, and failure recovery.
- Completed-leaf edit prevention where task UI exists.
- Cache invalidation and stale-response protection.
- Keyboard interaction, dialog focus, status accessibility, dan responsive
  layout.
- Edit Project uses the shared wide Dialog variant and composes the US-4.3
  whole-Project summary, including empty/loading/failure/Retry states without
  form draft, Save/Cancel, focus, or lifecycle regression.
- Home opens the shared Edit Project dialog directly without Projects/Project
  Structure background navigation.
- Lifecycle parity tests invoke the same Open/Locked commands from Home and
  Projects.

Tests verify observable behaviour and do not rely only on snapshots.

---

## 19. Technical Completion Criteria

1. Dedicated Project backend boundary uses domain, application,
   infrastructure, and transport packages consistent with current repository.
2. Domain remains independent of HTTP, database, GORM, framework, and
   infrastructure.
3. Frontend uses feature-oriented domain/application/infrastructure/
   presentation boundaries without speculative abstraction or TypeScript `any`.
4. Versioned migration exists; status uses safe constrained persistence.
5. Backend, not only frontend, assigns default Open status.
6. Locked Execution and Commitment baselines are persisted or protected by an
   equivalent deterministic immutable snapshot.
7. Forecast behavior while Locked is deferred; no mutable Forecast contract is asserted by this story.
8. Backend enforces Closed Project and completed-leaf read-only invariants.
9. Close eligibility traverses all descendant Executable Leaves and ignores
   Grouping WBS; complete Actual Date is the completion source of truth.
10. Project without leaves cannot lock or close and may be hard-deleted; any child makes
    delete unavailable and backend-enforced `PROJECT_HAS_CHILDREN` applies.
11. Lock, close, Locked/Closed reopen, and Priority change are transactional.
12. Concurrent status commands cannot violate lifecycle invariants.
13. Active ordering uses Project Priority before status; Closed records follow
    all active records.
14. Scheduling queries and Gantt consumers exclude Closed Projects.
15. Backend list pagination defaults to 5, maximum 100, and returns page,
    pageSize, serta total.
16. Name search uses the established indexed, escaped, case-insensitive prefix
    semantics.
17. Every new production query is reviewed with indexes for search, status,
    Priority, active/Closed ordering, joins, and completion traversal.
18. Non-trivial PostgreSQL query plans are verified with `EXPLAIN`; design
    preserves future MySQL contracts via dialect-specific migrations when
    required.
19. Runtime persistence uses GORM without raw SELECT/INSERT/UPDATE/DELETE; raw
    migration SQL is limited to permitted DDL.
20. API DTO, domain entity, and database models remain separate where their
    responsibilities differ.
21. Timeline and lifecycle rules are not implemented only in React,
    handler, or repository code.
22. Confirmed mutation invalidates affected Project list/detail and available
    scheduling/Gantt caches or projections.
23. Versioned invalidation prevents stale in-flight responses from restoring
    invalid state.
24. Loading, empty, no-results, error, warning, confirmation, and success states
    remain distinct and reuse shared design-system components.
25. Forms preserve draft after failure and prevent duplicate submission.
26. Status and warnings are accessible without relying only on colour;
    responsive behaviour is tested.
27. Mandatory domain, application, repository, API, and frontend tests pass.
28. Backend formatting, vet/static analysis, migration verification, tests, and
    race tests pass.
29. Frontend formatting, lint, typecheck, tests, and build pass.
30. Backend restarts after backend/config/migration changes; replacement process
    is confirmed listening and affected endpoints are smoke-tested.
31. API and project architecture documentation are updated.
32. Completion report includes documentation-impact assessment, deviations,
    remaining risks, and technical debt.

Implementation must stop for product clarification if unresolved answers are
required to define a mandatory request, invariant, ordering, or persistence
contract.

---

## 20. Documentation Impact

Future implementation must assess and update:

- `docs/project/architecture.md` for Project feature boundary, lifecycle,
  baseline persistence, completion traversal, active/Closed ordering,
  scheduling/Gantt exclusion, pagination/search/cache contracts, and indexes.
- API documentation for create, update, status command, Priority command,
  response DTO, and structured errors.
- `README.md` only if setup or usage changes.
- `.env.example` only if new configuration is introduced.
- Related future WBS, Scheduling, Gantt, Forecast, and Project Priority stories
  when their observable contracts are introduced or changed.
- This story when unresolved product decisions are approved or implementation
  reveals a contradiction.
- `AGENTS.md` only if a durable cross-project rule is introduced.

This requirement is synchronized with US-3.3, US-4.1, US-4.2, US-4.3, and project architecture/context documentation. US-4.3 remains authoritative for summary calculations and loading/copy rules.

---

## 21. Locked Product Decisions

- Home is the canonical active-WBS surface; standalone Project Structure is
  removed.
- Project lifecycle commands remain available on Projects and eligible active
  Home Project rows through the same use cases.
- Locked Project may rename Project Name only; Settings and planning remain
  read-only.
- Locked Name rename never invokes scheduler or scheduling-impact preview.

- Project is the root planning entity and logical WBS level `0`; no root WBS record is created.
- Status values are exactly Open, Locked, and Closed; create defaults to Open.
- Open is editable and participates in scheduling.
- Locked protects Execution/Commitment timelines and allocations as immutable anchors.
- Locked planning, WBS, Task, Settings, and dependency data are read-only.
- Actual Date entry is the only Task mutation allowed while Locked; it does not change protected baseline, but Actual Allocation may recalculate impacted Open Projects.
- Reopen completed Task requires the Project to be Open.
- Project may Lock only when it has at least one Executable Task and every unfinished Task is fully scheduled. Lock validates current state and does not run scheduler.
- Allowed transitions: Open→Locked, Locked→Open, Open→Closed, Locked→Closed, and Closed→Open. Closed→Locked is forbidden.
- Locked→Open calculates an atomic Required Locked Reopen Closure, then recalculates all unfinished Tasks in the transitive impacted scheduling scope.
- Unscheduled result does not fail Locked Reopen; technical/integrity/concurrency failure rolls the entire transition back.
- Priority may change while Locked Projects exist only when simulation proves no Locked impact; Open-only impact requires grouped warning/confirmation and server revalidation.
- Priority recalculation is transitively bounded; unrelated Projects are not recalculated or version-updated.
- Closed is historical, read-only, excluded from scheduling and Gantt, and ordered after active Projects.
- Close is permitted only when every descendant Executable Task has complete Actual Date.
- Forecast, Delivery Impact, and Project Health behavior while Locked is deferred.
- Project list, summary, deletion, naming, pagination, search, active ordering, and US-4.3 composition rules remain unchanged unless explicitly superseded above.

---

## 22. Unresolved Questions

Tidak ada unresolved question untuk scope US-3.1.

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
