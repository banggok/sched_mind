# US-3.1 — Create Project

> Product decision update: Project includes a nullable `Scheduling Start Date`
> (`SQL DATE`, API `YYYY-MM-DD`). It is the only initial anchor for future
> Automatic Scheduling and is configured in the combined Add/Edit Project form.
> Automatic Scheduling may be saved without it, but generated timelines must
> remain empty and the UI must warn: `Automatic Scheduling requires a Project
> Scheduling Start Date.` No task-level Earliest Start or equivalent anchor is
> permitted. Scheduling calculation remains outside this story.

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

1. Membuka page **Projects** di bawah navigation group **Project**.
2. Melihat paginated Project List.
3. Mencari Project berdasarkan Name secara case-insensitive.
4. Membuka Project detail.
5. Membuat Project dengan data yang valid.
6. Melihat status Project.
7. Mengubah data Project yang berstatus Open.
8. Mengubah status sesuai transition yang diizinkan.
9. Mengunci Open Project dan melindungi Execution serta Commitment baseline.
10. Menutup Open atau Locked Project yang seluruh Executable Leaf-nya selesai.
11. Membuka kembali Closed Project menjadi Open.
12. Melihat Closed Project setelah seluruh active Project pada Project List.
13. Mengubah Priority melalui Move Up atau Move Down.
14. Menghapus Project yang belum mempunyai child WBS/Executable Leaf.

Kontrak berikut ditetapkan sekarang untuk consumer pada story berikutnya:

- Open dan Locked Project ikut scheduling serta terlihat di Gantt.
- Project harus mempunyai sedikitnya satu Executable Leaf sebelum dapat di-Lock.
- Closed Project tidak ikut scheduling atau Gantt, tetapi tetap ada di list.
- Completed Executable Leaf ditentukan oleh Actual End dan tidak dapat diedit.
- Close validation memeriksa seluruh descendant Executable Leaf.
- Locked Execution dan Commitment baseline tetap immutable selama status Locked.
- Forecast, Delivery Impact, dan Project Health tetap dinamis saat Locked.

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

- Navigation menambahkan group **Project** dengan menu **Projects**.
- Page menggunakan persistent application shell; top bar, sidebar, dan global
  chrome tidak diduplikasi atau diremount.
- Page title dan menu label adalah **Projects**.
- Primary action adalah **Add Project**.
- Project List mengikuti shared list, search, pagination, skeleton, empty state,
  no-results, Retry, background refresh, dan Toast patterns.
- Add form tidak meminta user memilih status; status Open dijelaskan sebagai
  default yang ditetapkan backend.
- Add form hanya meminta Project Name. Priority ditetapkan sistem pada posisi
  terendah; derived dates dan automatic configuration tidak perlu diinput.
- Draft dipertahankan setelah validation atau backend failure.
- Controls terkait dinonaktifkan selama mutation untuk mencegah duplicate
  submission tanpa memblokir seluruh page.
- Status ditampilkan sebagai text label, bukan hanya warna.
- Open Project menyediakan action **Lock Project** dan **Close Project**.
- Locked Project menyediakan action **Close Project**, bukan Reopen.
- Closed Project menyediakan action **Reopen Project**; mutation lain tidak
  tersedia dan alasan read-only dapat dipahami.
- Project tanpa child menyediakan confirmed **Delete Project**. Project yang
  mempunyai child tidak dapat dihapus dan diarahkan menggunakan Close setelah
  completion requirement terpenuhi.
- Lock confirmation menjelaskan bahwa Execution dan Commitment dates akan
  menjadi baseline yang dilindungi.
- Close confirmation menjelaskan bahwa semua task harus mempunyai Actual End.
- Reopen confirmation menjelaskan bahwa Project kembali aktif, editable,
  eligible untuk scheduling, dan terlihat di Gantt.
- Editing Locked Project tidak pernah membuka status menjadi Open. Scheduling
  result hanya memperbarui Forecast; Execution dan Commitment tetap baseline.
- Dialog mengelola focus dan mengembalikannya ke trigger.
- Core workflow tetap usable pada supported viewport tanpa horizontal scrolling
  pada normal page content.

---

## 6. Domain Model

### Project

| Field | Type | Required | Source | Rules |
| --- | --- | ---: | --- | --- |
| ID | System-generated identifier | Ya | System | Dibuat otomatis dan immutable |
| Name | String | Ya | User | Trimmed; non-blank; maksimum 100 karakter; unique case-insensitive |
| Status | `open \| locked \| closed` | Ya | System/transition | Default `open`; hanya berubah melalui lifecycle command |
| Start Date | Nullable date-only | Tidak | System-derived | Min Start Date descendant leaf; `null` bila belum ada leaf/date |
| End Date | Nullable date-only | Tidak | System-derived | Max End Date descendant leaf; `null` bila belum ada leaf/date |
| Auto Calculate Date | Boolean | Ya | System | Selalu `true`; tidak editable |
| Auto Dependency by Assignee | Boolean | Ya | System | Selalu `true`; tidak editable |
| Project Priority | Positive integer position | Ya | System/User command | Unique; create menggunakan global `MAX(priority)+1`; Move Up/Down menukar position dengan active neighbour |
| Closed At | Nullable DateTime | Tidak | System | Diisi saat successful close; retained untuk audit dan Closed ordering |
| Created At | DateTime | Ya | System | Dibuat otomatis dan immutable |
| Updated At | DateTime | Ya | System | Diperbarui setelah confirmed mutation |

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
| Open | Active planning | Ya, subject to domain rules | Included | Visible | Execution, Commitment, dan Forecast dapat recalculated |
| Locked | Active dengan protected baseline | Unfinished planning data tetap editable | Included | Visible | Execution dan Commitment baseline tetap; Forecast dynamic |
| Closed | Completed historical Project | Tidak | Excluded | Hidden | Semua planning dan timeline read-only |

- Status saat create selalu `open`, ditegakkan backend.
- Tidak ada status lain.
- Status bukan free-form value.
- Status transition merupakan business command, bukan generic field update.

---

## 8. Status Transition Rules

| From | To | Allowed | Conditions and effects |
| --- | --- | ---: | --- |
| Open | Locked | Ya | Harus mempunyai sedikitnya satu Executable Leaf; capture/protect current Execution dan Commitment baseline |
| Open | Closed | Ya | Seluruh descendant Executable Leaf harus memiliki Actual End |
| Locked | Closed | Ya | Full close validation tetap wajib |
| Closed | Open | Ya | Project kembali editable, schedulable, dan visible in Gantt |
| Closed | Locked | Tidak | Reject tanpa partial mutation |

- Invalid status value ditolak.
- Transition selain tabel di atas ditolak kecuali idempotency semantics kemudian
  dikunci; story ini tidak menganggap same-status request berhasil.
- Existing status dan data dipertahankan bila transition gagal.
- Lock, close, Closed-to-Open reopen, dan Priority swap harus transactional.
- Concurrent transition tidak boleh menghasilkan state yang melanggar
  lifecycle; stale command harus ditolak atau diserialisasi secara deterministik.

---

## 9. Timeline Behaviour by Status

### Open

- Execution, Commitment, dan Forecast Timeline dapat diperbarui oleh Scheduling
  Engine.
- Relevant constraints mencakup Daily Capacity, Capacity Override, Public
  Holiday, Project Priority, Effort, Assignee, Dependency, Lag, Actual End, dan
  Project Freeze.
- Dalam scope requirement saat ini, hanya explicit Priority Move Up/Down yang
  langsung memicu Scheduling Engine menghitung ulang seluruh task dates.
  Perubahan constraint lain tidak menjadi trigger baru dalam US-3.1; trigger
  tersebut harus ditentukan oleh story pemilik fiturnya.
- Project ikut scheduling dan Gantt.
- Priority Move Up/Down langsung menjalankan Scheduling Engine untuk menghitung
  ulang task dates seluruh active Projects. Open Project dapat menerima updated
  Execution, Commitment, dan Forecast dates.

### Locked

- Current Execution Timeline menjadi locked Execution baseline pada transition.
- Current Commitment Timeline menjadi locked Commitment baseline.
- Execution dan Commitment dates tidak berubah selama Project tetap Locked.
- Forecast tetap mencerminkan kondisi terbaru.
- Delivery Impact dan Project Health tetap dinamis.
- Actual End tetap editable untuk unfinished leaf sesuai completed-task rules.
- Completed Task dapat dibuka kembali hanya melalui dedicated Reopen Task
  command dari US-4.2. Open dan Locked Project eligible; Reopen Task pada
  Locked Project mempertahankan status Locked dan seluruh locked baselines.
- Project tetap ikut scheduling berdasarkan Project Priority dan tetap terlihat
  di Gantt.
- External constraint changes tidak menggeser locked baselines.
- Priority change tetap menjalankan Scheduling Engine untuk seluruh active task,
  tetapi Locked Project hanya menerima updated Forecast dates.

### Closed

- Project tidak ikut scheduling atau Project Priority allocation.
- Project tidak tampil di Gantt.
- Seluruh planning, Actual End, dan timeline read-only.
- Project tetap tersedia sebagai historical record pada Project List.

---

## 10. Editing Rules by Status

### Open Project

- Project fields dan planning data dapat diubah subject to field-specific dan
  downstream domain rules.
- Dalam US-3.1, hanya Priority Move Up/Down yang memicu recalculation.

### Locked Project

- WBS structure, Effort, Assignee, Dependency, Lag, Project configuration,
  Actual End, dan established planning fields dapat diubah untuk unfinished
  work.
- Executable Leaf dengan Actual End adalah completed dan tidak dapat diedit
  melalui normal mutation. Dedicated Reopen Task dari US-4.2 adalah satu-satunya
  exception untuk menghapus Actual End.
- Confirmed planning change tetap mempertahankan status Locked.
- Scheduling Engine hanya memperbarui Forecast dates untuk Locked Project,
  termasuk bila Forecast bergerak lebih awal atau lebih lambat.
- Execution dan Commitment baselines tidak pernah berubah karena Locked edit.
- Locked Project tidak mempunyai transition atau action Reopen ke Open.

### Closed Project

- Project, WBS, task, Actual End, planning fields, dan timeline read-only.
- Backend menolak mutation walaupun frontend restriction dilewati.
- Project harus direopen ke Open sebelum perubahan lain, termasuk sebelum
  menjalankan Reopen Task. Closed Project tidak menerima Task-level exception.

---

## 11. Closed Validation

- Closing adalah hard validation, bukan warning.
- Project boleh Closed jika dan hanya jika setiap descendant Executable Leaf
  mempunyai Actual End.
- Actual End adalah satu-satunya source of truth untuk completion; task status
  terpisah tidak digunakan.
- Grouping WBS tidak dievaluasi langsung dan tidak membutuhkan Actual End.
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
- Untuk active Project dengan Priority sama, tie-breaker adalah Start Date ASC
  (`null` last), End Date ASC (`null` last), status dengan Locked sebelum Open,
  lalu Project ID ASC.
- Filter/search/pagination menjadi request-cache identity.
- Confirmed mutation memperbarui affected list/detail serta scheduling/Gantt
  projection tanpa hard refresh.
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

### AC-1 — Projects navigation

**Given** application shell tersedia
**When** Engineering Lead membuka group Project
**Then** menu Projects membuka page Projects
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

### AC-9 — Open to Locked

**Given** Project mempunyai sedikitnya satu Executable Leaf
**When** Engineering Lead mengonfirmasi Lock Project
**Then** status menjadi Locked
**And** current Execution dan Commitment Timeline menjadi protected baselines
secara atomic.

**Given** Project tidak mempunyai Executable Leaf
**When** Lock Project diminta
**Then** transition ditolak dengan `PROJECT_CANNOT_LOCK_WITHOUT_TASKS`
**And** status tetap Open
**And** user dapat menambahkan task atau menghapus Project.

### AC-10 — Locked baseline dan dynamic Forecast

**Given** Project Locked
**When** capacity constraint atau Actual End berubah
**Then** locked Execution dan Commitment dates tetap
**And** Forecast, Delivery Impact, dan Project Health boleh berubah
**And** Project tetap priority-ordered, schedulable, dan visible in Gantt.

### AC-11 — Locked unfinished and completed leaf editing

**Given** Executable Leaf di Locked Project belum memiliki Actual End
**Then** established planning change dapat dievaluasi dan disimpan
**But given** leaf mempunyai Actual End
**Then** mutation ditolak dengan `COMPLETED_TASK_READ_ONLY`.

### AC-12 — Locked edit changes Forecast only

**When** planning data pada Locked Project diubah dan scheduling dijalankan
**Then** change dapat disimpan dan status tetap Locked
**And** status tetap Locked serta protected baselines tidak berubah
**And** Forecast mengikuti kondisi terbaru serta boleh bergerak lebih awal atau
lebih lambat.

### AC-13 — Priority change triggers scheduling

**When** Priority Project dipindahkan Up atau Down
**Then** position ditukar secara atomic dengan active neighbour
**And** Scheduling Engine menghitung ulang task dates seluruh active Projects
**And** Open Projects menerima updated Execution, Commitment, dan Forecast
**And** Locked Projects hanya menerima updated Forecast.

### AC-14 — Locked cannot reopen to Open

**Given** Project berstatus Locked
**When** target status Open diminta
**Then** transition ditolak dengan `PROJECT_STATUS_TRANSITION_NOT_ALLOWED`
**And** Reopen hanya tersedia bagi Closed Project.

### AC-15 — Close Open or Locked Project

**Given** seluruh descendant Executable Leaf mempunyai Actual End
**When** Close Project dikonfirmasi dari Open atau Locked
**Then** status menjadi Closed secara atomic
**And** Grouping WBS tidak memerlukan Actual End.

### AC-16 — Reject incomplete close

**Given** satu direct atau nested descendant Executable Leaf tidak mempunyai
Actual End
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
      "autoCalculateDate": true,
      "autoDependencyByAssignee": true,
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
  "name": "Alpha"
}
```

Backend assigns `status: open`, lowest Project Priority, `startDate: null`,
`endDate: null`, both automatic flags `true`, ID, and timestamps. Request must
not accept system-derived fields or manually supplied locked baselines.

Success: `201 Created` with `{ "data": Project }`.

### Update Project

```http
PUT /api/projects/{projectId}
Content-Type: application/json
```

The current Project-level writable DTO contains only `name`. Derived dates,
automatic flags, Priority, status, Closed At, timestamps, and locked baselines
are not accepted. Status changes use the command endpoint. Closed Project
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
- Open to Locked captures/protects baselines.
- Open to Locked rejects a zero-leaf Project with
  `409 PROJECT_CANNOT_LOCK_WITHOUT_TASKS`.
- Open/Locked to Closed runs descendant completion validation.
- Closed to Open reactivates the Project.
- Reopen retains previous locked baselines and immediately restores scheduling
  and Gantt eligibility.
- Locked to Open returns `409 PROJECT_STATUS_TRANSITION_NOT_ALLOWED`; Reopen
  hanya berlaku untuk Closed Project.
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
- Successful move memicu Scheduling Engine menghitung ulang task dates seluruh
  active Projects.
- Open Projects menerima hasil Execution, Commitment, dan Forecast; Locked
  Projects hanya menerima Forecast.
- Priority swap dan hasil scheduling harus konsisten. Failure tidak boleh
  meninggalkan partial priority atau timeline state.

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

| Error Code | HTTP | Field | Condition |
| --- | ---: | --- | --- |
| `INVALID_REQUEST` | 400 | — | Malformed JSON, unknown field, atau multiple payloads |
| `PROJECT_NAME_REQUIRED` | 400 | `name` | Trimmed Name kosong |
| `PROJECT_NAME_TOO_LONG` | 400 | `name` | Trimmed Name lebih dari 100 karakter |
| `PROJECT_NAME_ALREADY_EXISTS` | 409 | `name` | Name digunakan Project lain secara case-insensitive |
| `PROJECT_NOT_FOUND` | 404 | — | Project ID tidak tersedia |
| `PROJECT_STATUS_INVALID` | 400 | `status` | Target bukan open, locked, atau closed |
| `PROJECT_STATUS_TRANSITION_NOT_ALLOWED` | 409 | `status` | Transition tidak diizinkan, termasuk Locked ke Open dan Closed ke Locked |
| `PROJECT_CANNOT_LOCK_WITHOUT_TASKS` | 409 | `status` | Lock diminta untuk Project tanpa Executable Leaf |
| `PROJECT_CANNOT_CLOSE_WITH_ACTIVE_TASKS` | 409 | `status` | Sedikitnya satu descendant Executable Leaf tidak mempunyai Actual End |
| `PROJECT_CANNOT_CLOSE_WITHOUT_TASKS` | 409 | `status` | Close diminta untuk Project tanpa Executable Leaf |
| `PROJECT_CLOSED_READ_ONLY` | 409 | — | Mutation planning pada Closed Project |
| `COMPLETED_TASK_READ_ONLY` | 409 | — | Mutation pada Executable Leaf yang mempunyai Actual End |
| `PROJECT_PRIORITY_DIRECTION_INVALID` | 400 | `direction` | Direction bukan up atau down |
| `PROJECT_PRIORITY_MOVE_NOT_ALLOWED` | 409 | `direction` | Tidak ada active neighbour pada arah yang diminta |
| `PROJECT_HAS_CHILDREN` | 409 | — | Delete diminta untuk Project yang mempunyai child |
| `INVALID_PAGE` | 400 | `page` | Page bukan positive integer |
| `INVALID_PAGE_SIZE` | 400 | `pageSize` | Page size di luar 1..100 |

Field-specific date/configuration errors tidak diperlukan karena values tersebut
system-derived. Internal error tidak
mengekspos stack trace, SQL, database, atau infrastructure detail.

---

## 17. Test Cases

| ID | Scenario | Expected |
| --- | --- | --- |
| TC-1 | Buka Project > Projects | Page berada di shared shell; menu dan breadcrumb aktif |
| TC-2 | Initial list pending | Local skeleton; empty/no-results belum tampil |
| TC-3 | Empty successful list | Empty state dan Add Project action |
| TC-4 | List failure lalu Retry | Actionable error, kemudian list pulih |
| TC-5 | Search Name beda casing | Backend prefix search match |
| TC-6 | Search tanpa match lalu Clear | No-results berbeda; Clear kembali ke page 1 |
| TC-7 | Lebih dari lima Projects | Backend pagination dan metadata benar |
| TC-8 | Active ordering | Open/Locked mengikuti Priority tanpa status grouping |
| TC-9 | Closed ordering group | Semua Closed berada setelah seluruh active Projects |
| TC-10 | Buka/Cancel Add form | Labeled approved fields; no mutation on Cancel |
| TC-11 | Create valid Project | `201`; generated ID/timestamps; backend status Open |
| TC-12 | Create success | Project terlihat tanpa hard refresh; relevant caches consistent |
| TC-13 | Create invalid via direct API | Structured validation; repository not mutated |
| TC-14 | Create dependency failure | Draft preserved; retry available; no partial data |
| TC-15 | Repeated create submit | Satu request |
| TC-16 | Open Project contract | Editable, scheduled, dan visible in Gantt |
| TC-17 | Lock Project dengan Executable Leaf | Status Locked dan both baselines captured atomically |
| TC-18 | External capacity change while Locked | Execution/Commitment baseline unchanged |
| TC-19 | Actual End/constraint change while Locked | Forecast dapat berubah |
| TC-20 | Edit unfinished Locked leaf | Evaluated and eligible to save |
| TC-21 | Edit completed leaf | `409 COMPLETED_TASK_READ_ONLY`; data unchanged |
| TC-22 | Locked change produces same timeline | Saved, remains Locked, baselines unchanged |
| TC-23 | Locked change produces earlier timeline | Saved, remains Locked; only Forecast may move earlier |
| TC-24 | Locked change produces later timeline | Saved, remains Locked; only Forecast may move later |
| TC-25 | Move Priority Up/Down | Atomic active-neighbour swap and Scheduling Engine recalculates every active Project task date |
| TC-26 | Locked to Open | `409 PROJECT_STATUS_TRANSITION_NOT_ALLOWED`; state unchanged |
| TC-27 | Close Open; all leaves have Actual End | Closed succeeds |
| TC-28 | Close Locked; all leaves have Actual End | Closed succeeds |
| TC-29 | One leaf lacks Actual End | Close rejected; state preserved |
| TC-30 | Nested unfinished descendant | Close rejected after descendant traversal |
| TC-31 | Grouping WBS dan completed descendant leaf | Group tidak butuh Actual End; close succeeds |
| TC-32 | Closed Project mutation | Backend rejects read-only violation |
| TC-33 | Closed scheduling/Gantt | Excluded from both but retained in Project List |
| TC-34 | Closed to Open | Transition succeeds and Project becomes active/editable |
| TC-35 | Closed to Locked | Structured invalid-transition conflict |
| TC-36 | Unknown status | `400 PROJECT_STATUS_INVALID` |
| TC-37 | Unknown Project ID | `404 PROJECT_NOT_FOUND` |
| TC-38 | Lock dependency failure | Transaction rollback |
| TC-39 | Close dependency failure | Transaction rollback |
| TC-40 | Reopen dependency failure | Transaction rollback |
| TC-41 | Priority swap or scheduling dependency failure | Priority and timeline transaction/coordination rollback |
| TC-42 | Concurrent incompatible transitions | At most one valid confirmed state; invariants hold |
| TC-43 | Mutation with active search/page | Query state preserved and affected caches invalidated |
| TC-44 | Stale response completes after mutation | Stale data cannot repopulate cache |
| TC-45 | Keyboard and assistive technology | Controls, warnings, status, and focus accessible |
| TC-46 | Supported viewport | Workflow usable without normal horizontal scrolling |
| TC-47 | Delete Project tanpa child | Confirmed hard delete; list/page corrected |
| TC-48 | Delete Project dengan child | `409 PROJECT_HAS_CHILDREN`; data tetap utuh |
| TC-49 | Close Project tanpa Executable Leaf | `409 PROJECT_CANNOT_CLOSE_WITHOUT_TASKS`; Delete tersedia |
| TC-50 | Lock Project tanpa Executable Leaf | `409 PROJECT_CANNOT_LOCK_WITHOUT_TASKS`; status tetap Open |

### Acceptance Criteria Traceability

| Acceptance Criteria | Test Case(s) |
| --- | --- |
| AC-1 | TC-1 |
| AC-2 | TC-7—TC-9 |
| AC-3 | TC-5—TC-6, TC-43 |
| AC-4 | TC-2—TC-4 |
| AC-5—AC-7 | TC-10—TC-15 |
| AC-8 | TC-16 |
| AC-9—AC-10 | TC-17—TC-19, TC-50 |
| AC-11 | TC-20—TC-21 |
| AC-12—AC-14 | TC-22—TC-26 |
| AC-15—AC-16 | TC-27—TC-31, TC-49 |
| AC-17 | TC-32—TC-33 |
| AC-18—AC-20 | TC-34—TC-37 |
| AC-21 | TC-43—TC-44 |
| AC-22 | TC-45—TC-46 |
| AC-23 | TC-38—TC-42 |
| AC-24 | TC-47—TC-48 |

---

## 18. Required Automated Tests

### Domain Tests

- Project creation and backend-enforced default Open status.
- Valid status values, allowed transitions, and invalid transitions.
- Identity/Created At preservation and Updated At changes.
- Closed read-only invariant.
- Completed Executable Leaf read-only invariant using Actual End.
- Close eligibility across all descendant Executable Leaves.
- Grouping WBS exclusion from completion validation.
- Zero-leaf close rejection and childless delete eligibility.
- Zero-leaf lock rejection.
- Locked baseline invariant.
- Locked edit preserves status and baselines while Forecast may move earlier or
  later.
- Locked-to-Open transition rejection.
- Positive integer Priority and valid/invalid move rules.

### Application Tests

- Paginated list, Name search, create, get, dan Open update.
- Lock, Open-to-Closed, Locked-to-Closed, dan Closed-to-Open.
- Reject Open-to-Locked when the Project has zero Executable Leaves.
- Reject Closed-to-Locked, Closed mutation, completed-leaf mutation, dan close
  with unfinished leaves.
- Delete childless Project and reject delete when any child exists.
- Reject Locked-to-Open and preserve Locked baselines on every planning edit.
- Atomic Priority Move Up/Down and Scheduling Engine recalculation contract for
  all active Project task dates.
- Dependency failure rollback and concurrent transition behaviour.
- Scheduling/Gantt visibility and downstream invalidation contracts.

### Repository Integration Tests

- Persist/retrieve Project, default status, and transitions.
- Persisted baseline snapshot or equivalent protection.
- Descendant leaf completion query excluding Grouping WBS.
- Indexed Name prefix search before count/pagination.
- Pagination metadata.
- Active-before-Closed grouping.
- Active Project Priority ordering with Start/End/status/ID tie-breakers.
- Atomic active-neighbour Priority swap and rollback.
- Closed ordering by Closed At DESC then ID ASC.
- Transaction rollback and concurrent transition protection.
- Query plans and indexes for search, status, Priority, active/Closed ordering,
  and completion validation.
- SQLite isolation for automated tests, PostgreSQL verification, dan future
  MySQL-compatible contract.

### API Integration Tests

- List, search, pagination metadata, get, create, dan update.
- Lock, Closed reopen, close, invalid status, invalid transition, not-found.
- Zero-leaf Lock conflict mapping.
- Close rejection, Closed read-only, completed-task read-only.
- Locked-to-Open rejection and Priority Move Up/Down contracts.
- Structured error mapping, persistence, rollback, and unknown-field rejection.
- Conditional Project delete success and `PROJECT_HAS_CHILDREN` conflict.

### Frontend Tests

- Projects navigation, stable shell, initial loading, empty, error/Retry,
  no-results/Clear, and pagination.
- Add form, valid create, backend default Open display, failure draft
  preservation, and duplicate prevention.
- Conditional delete confirmation, success, rejection, and page correction.
- Status labels and available actions for Open, Locked, dan Closed.
- Lock/close/reopen confirmations and close rejection.
- Zero-leaf Lock rejection and actionable recovery message.
- Closed read-only presentation.
- Locked edit retains status and baselines while displaying updated Forecast.
- Priority Move Up/Down availability, success, scheduling feedback, and failure
  recovery.
- Completed-leaf edit prevention where task UI exists.
- Cache invalidation and stale-response protection.
- Keyboard interaction, dialog focus, status accessibility, dan responsive
  layout.

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
7. Forecast remains mutable while Locked.
8. Backend enforces Closed Project and completed-leaf read-only invariants.
9. Close eligibility traverses all descendant Executable Leaves and ignores
   Grouping WBS; Actual End is the only completion source of truth.
10. Project without leaves cannot lock or close and may be hard-deleted; any child makes
    delete unavailable and backend-enforced `PROJECT_HAS_CHILDREN` applies.
11. Lock, close, Closed reopen, and Priority swap are transactional.
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

No existing documentation other than this new story is changed by authoring
this requirement.

---

## 21. Locked Product Decisions

- Project is the root planning entity in Epic 3.
- Visible navigation and page label is Projects under Project group.
- Fields include ID, Name, Status, Start Date, End Date, Auto Calculate Date,
  Auto Dependency by Assignee, Created At, and Updated At; Project Priority is
  required by active ordering and Closed At supports audit/ordering.
- Status values are exactly Open, Locked, and Closed; create defaults to Open
  and backend enforces it.
- Open is active, editable, scheduled, visible in Gantt, and all timelines may
  be recalculated.
- Locked remains active and Priority-ordered; Execution and Commitment baseline
  remain immutable while Forecast, Delivery Impact, and Project Health remain
  dynamic.
- Completed Executable Leaf means it has Actual End and is read-only.
- Locked planning changes retain Locked status. Scheduling may update Forecast
  earlier or later, but never changes Execution or Commitment baselines.
- Closed is manually selected, historical, read-only, excluded from scheduling
  and Gantt, and retained after active Projects in the list.
- Close is permitted only when every descendant Executable Leaf has Actual End;
  Grouping WBS is not evaluated directly.
- Lock is permitted only when the Project has at least one Executable Leaf.
- Allowed transitions: Open→Locked, Open→Closed, Locked→Closed, dan
  Closed→Open. Locked→Open dan Closed→Locked are forbidden; Reopen means only
  Closed→Open.
- Active list ordering uses Project Priority before status; Open and Locked are
  not separated into status groups.
- Project list has backend Name prefix search and pagination default 5/max 100.
- Project creation does not automatically create tasks or dependencies, lock,
  close, or run Scheduling Engine.
- Project is WBS level 0; creation does not create a separate root WBS record.
- Project without children can be hard-deleted; Project with children cannot be
  deleted and must use Closed for historical retention.
- Initial creation accepts Name together with the Project Settings defined by
  US-3.3; Name follows trimmed, required, maximum 100, case-insensitive unique
  BAU rules. Status is not accepted and remains system/lifecycle-command owned.
  System assigns lowest Priority.
- Priority is a unique positive integer position; a smaller value is higher.
  Create assigns global `MAX(priority)+1`; Move Up/Down atomically swaps with
  the active neighbour.
- Only an explicit Priority Move Up/Down triggers the Scheduling Engine in this
  story and recalculates all task dates across active Projects. Open Projects
  may receive Execution, Commitment, and Forecast updates; Locked Projects
  receive Forecast updates only.
- Project Start/End are derived from descendant leaf dates and remain null when
  no relevant leaf date exists. Both automatic flags are always enabled.
- Equal-Priority active ordering uses Start Date, End Date, Locked-before-Open,
  then ID. Closed ordering uses Closed At descending then ID ascending.
- Reopening retains previous locked baselines and immediately restores Gantt and
  scheduling eligibility. A Locked edit may move Forecast earlier or later
  without changing status or protected baselines.
- Project with zero Executable Leaves cannot lock or close and may instead be
  deleted.
- Closed At is persisted for audit and ordering.

---

## 22. Unresolved Questions

Tidak ada unresolved question untuk scope US-3.1.
