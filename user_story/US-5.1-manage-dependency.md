# US-5.1 — Manage Dependency

## 1. User Story

**Sebagai** Engineering Lead,  
**Saya ingin** mengelola hubungan `Blocks` dan `Blocked by` antar Task,  
**Sehingga** Scheduling Engine mengetahui Task mana yang harus selesai sebelum Task lain dapat dimulai.

Dependency Management merupakan bagian dari **Epic 5: Dependency & Lag Management**.

---

## 2. Business Context

Dependency membentuk graph pekerjaan yang akan digunakan oleh portfolio-level Scheduling Engine.

Dalam UI, hubungan dependency dijelaskan dengan istilah:

- **Blocks**: Task ini harus selesai sebelum Task lain dapat dimulai.
- **Blocked by**: Task ini menunggu Task lain selesai sebelum dapat dimulai.

Keduanya merupakan dua tampilan dari satu relation yang sama. Backend tidak menyimpan dua relation terpisah.

Dependency dapat melintasi Project karena satu Task pada Project tertentu dapat menjadi prasyarat bagi Task pada Project lain. Konsekuensinya, perubahan dependency dapat memengaruhi lebih dari satu Project dan harus tersedia sebagai input bagi portfolio scheduler pada Epic 6.

Story ini hanya membangun dan memvalidasi dependency graph. Story ini tidak menghitung Execution, Commitment, atau Forecast Timeline.

---

## 3. Scope

Engineering Lead dapat:

1. Melihat daftar Task yang memblokir Task saat ini melalui `Blocked by`.
2. Melihat daftar Task yang diblokir Task saat ini melalui `Blocks`.
3. Menambahkan dependency melalui `Blocked by`.
4. Menambahkan dependency melalui `Blocks`.
5. Menghapus dependency yang masih boleh diubah.
6. Memilih Task dari Project aktif yang sama maupun Project aktif lain.
7. Melihat Task Name, full WBS hierarchy path yang dimulai dari Project sebagai WBS level 0, completion state, dan expected start Task yang diblokir bila projection sudah tersedia.
8. Mendapatkan validation feedback untuk self-dependency, duplicate dependency, invalid Task type, completed-task restriction, Closed Project, dan cycle.
9. Melihat dependency yang sudah ada tetap konsisten dari kedua arah.
10. Menghapus Task melalui owning WBS story dan memastikan seluruh dependency yang melibatkan Task tersebut ikut dihapus secara atomik.

---

## 4. Out of Scope

Story ini tidak mencakup:

- Lag.
- Auto Dependency.
- Execution Scheduler.
- Commitment Scheduler.
- Forecast Scheduler.
- Perhitungan expected start.
- Daily capacity allocation.
- Drag-to-create dependency pada Gantt.
- Dependency type selain Finish-to-Start.
- Dependency terhadap Summary Task atau Group.
- Restore dependency yang sudah dihapus.
- Dependency terhadap Project sebagai entity.
- Bulk dependency creation.
- Dependency template.
- Cross-portfolio atau cross-tenant dependency.
- Permission model baru.

Expected Start pada bagian `Blocks` merupakan optional scheduler projection. Sebelum Epic 6 tersedia, UI menampilkan `Not scheduled` atau `—` dan tidak membuat kalkulasi sementara.

---

## 5. UI Placement and User Flow

Dependency pada Phase 1 dikelola dari contextual Edit Task form di Project
Structure yang sudah tersedia. Komponen dan application contract harus dapat
digunakan kembali oleh Integrated Gantt Workspace melalui Task Grid ketika
workspace tersebut tersedia; implementasi story ini tidak membuat Gantt
Workspace baru.

### Task Grid

Task menyediakan dua field atau detail sections:

- **Blocked by**
- **Blocks**

Keduanya editable pada Phase 1 melalui contextual Edit Task form. Future Task
Grid menggunakan behaviour yang sama. Gantt tetap read-only pada Phase 1.

### Blocked by

Engineering Lead memilih Task yang harus selesai sebelum Task saat ini dapat dimulai.

Contoh:

```text
Integration Test
Blocked by:
- Backend API
- Authentication
```

### Blocks

Engineering Lead memilih Task yang menunggu Task saat ini selesai.

Contoh:

```text
Backend API
Blocks:
- Integration Test
- Frontend Integration
```

Setiap Task pada `Blocks` menampilkan bila tersedia:

- Project Name.
- Task Name.
- Expected Start.
- Status scheduled atau `Not scheduled`.

Tujuannya agar Engineering Lead dan engineer menyadari downstream impact apabila blocker terlambat.

### Selector

Task selector:

- Mencari seluruh Task pada seluruh Project aktif.
- Menggunakan case-insensitive backend substring search terhadap Task Name dan Project Name, dengan bounded pagination.
- Menampilkan Task Name sebagai primary label.
- Menampilkan full WBS hierarchy path sebagai supporting context.
- Hierarchy path dimulai dari Project sebagai WBS level 0 dan berakhir pada candidate Task.
- Contoh:

  ```text
  Task 2
  NTB > Task 1 > Task 2
  ```

- Menandai Task yang sudah completed.
- Mengecualikan Summary Task.
- Mengecualikan Task dari Closed Project.
- Mengecualikan Task yang tidak valid berdasarkan arah relation.
- Mengecualikan Task saat ini.
- Tidak menampilkan candidate yang sudah membentuk dependency yang sama.

### Feedback

- Create dan delete menampilkan mutation state dan success feedback.
- Form atau selector tetap mempertahankan draft/search setelah failure.
- Duplicate submission dicegah.
- Raw backend errors tidak ditampilkan.
- Cycle error menampilkan dependency path yang menyebabkan cycle.

---

## 6. Domain Model

### Dependency

| Field            | Type                        | Required | Source       | Rules                                                |
| ---------------- | --------------------------- | -------: | ------------ | ---------------------------------------------------- |
| ID               | System-generated identifier |       Ya | System       | Immutable                                            |
| Blocking Task ID | Executable Task identifier  |       Ya | User/context | Task yang harus selesai lebih dahulu                 |
| Blocked Task ID  | Executable Task identifier  |       Ya | User/context | Task yang menunggu blocker selesai                   |
| Created At       | DateTime                    |       Ya | System       | Immutable                                            |
| Updated At       | DateTime                    |       Ya | System       | System-managed bila dibutuhkan repository convention |

Dependency mempunyai semantics **Finish-to-Start**:

```text
Blocking Task finishes
        ↓
Blocked Task may become ready to start
```

`Blocks` dan `Blocked by` bukan field terpisah. Keduanya merupakan projection dari relation `Blocking Task ID → Blocked Task ID`.

---

## 7. Business Rules

### 7.1 Task-only Dependency

- Dependency hanya boleh melibatkan Executable Task.
- Summary Task atau Group tidak boleh menjadi blocker maupun blocked task.
- Project root tidak boleh menjadi endpoint dependency.
- Task dapat berada pada Project yang sama atau Project berbeda.

### 7.2 Finish-to-Start Only

MVP hanya mendukung Finish-to-Start.

Task yang diblokir tidak dapat menjadi ready sebelum blocker selesai.

Dependency type lain berikut tidak tersedia:

- Start-to-Start.
- Finish-to-Finish.
- Start-to-Finish.

### 7.3 Cardinality

- Satu Task boleh memblokir banyak Task.
- Satu Task boleh diblokir banyak Task.
- Relation dengan Blocking Task dan Blocked Task yang sama hanya boleh tersedia satu kali.

### 7.4 Self Dependency

Task tidak boleh memblokir dirinya sendiri.

### 7.5 Cross-project Dependency

- Dependency boleh melintasi Project aktif.
- Project asal dan Project tujuan harus tersedia.
- Closed Project tidak boleh menyediakan Task untuk dependency baru.
- Existing dependency yang melibatkan Task completed tetap dipertahankan sebagai histori.
- Dependency cross-project menjadi input portfolio scheduler pada Epic 6.
- Mutation dependency harus menginvalidasi projection dari seluruh affected Project, bukan hanya Project tempat form dibuka.

### 7.6 Cycle Detection

Dependency graph harus selalu acyclic.

Create dependency ditolak jika menghasilkan direct maupun indirect cycle.

Contoh:

```text
A blocks B
B blocks C
Candidate: C blocks A
```

Candidate ditolak.

Error harus menyediakan path yang menjelaskan cycle:

```text
A → B → C → A
```

Cycle validation berlaku lintas Project.

### 7.7 Completed Task as Blocking Task

Task dengan Actual End dianggap completed.

Completed Task pada Project aktif **boleh dipilih sebagai Blocking Task baru**.

Tujuannya adalah memungkinkan Engineering Lead melengkapi dependency yang baru disadari setelah pekerjaan blocker selesai.

Ketika completed Task menjadi blocker:

- Dependency dianggap satisfied berdasarkan Actual End.
- Epic 6 menggunakan Actual End sebagai dependency-ready anchor.
- Execution End atau Commitment End tidak menggantikan Actual End.
- Data completed Task tidak berubah.

### 7.8 Completed Task as Blocked Task

Completed Task tidak boleh menjadi Blocked Task baru.

Contoh yang ditolak:

```text
Task A already completed
Candidate: Task C blocks Task A
```

Relation tersebut akan mengubah histori seolah Task A seharusnya menunggu Task C.

### 7.9 Dependency involving Completed Blocked Task

- Dependency baru yang menjadikan completed Task sebagai blocked task ditolak.
- Existing dependency yang sisi blocked task-nya sudah completed tidak dapat dihapus atau dimodifikasi.
- Completed Task tetap read-only sebagai work record.
- Completed blocker boleh ditambahkan tanpa mengubah blocker tersebut.

### 7.10 Closed Project

- Task dari Closed Project tidak tersedia pada selector dependency baru.
- Dependency existing yang melibatkan completed Task dari Closed Project tetap dipertahankan sebagai histori.
- Dependency existing tidak dihapus hanya karena Project berubah menjadi Closed.
- Project hanya dapat Closed setelah seluruh Task memiliki Actual End, sesuai Project lifecycle story.

### 7.11 Delete Dependency

Dependency dapat dihapus selama penghapusan tidak melanggar completed-task historical rule.

Delete:

- menggunakan hard delete untuk relation;
- memerlukan confirmation bila current UX pattern mengharuskannya;
- tidak menghapus Task;
- tidak membuat relation pengganti;
- tidak melakukan auto-reconnect.

### 7.12 Delete Task

Ketika Task dihapus melalui owning WBS workflow:

- seluruh dependency dengan Task tersebut sebagai blocker dihapus;
- seluruh dependency dengan Task tersebut sebagai blocked task dihapus;
- operation atomic;
- tidak ada auto-reconnect.

Contoh:

```text
A → B → C
Delete B
```

Hasil manual dependency:

```text
A    C
```

Tidak otomatis menjadi `A → C`.

Auto Dependency pada story terpisah dapat membentuk relation baru berdasarkan rule-nya sendiri setelah confirmed deletion.

### 7.13 Rename, Move, and Reorder

- Rename Task tidak mengubah dependency.
- Move Task atau subtree tidak menghapus dependency karena dependency tidak bergantung pada hierarchy parent.
- Reorder Task tidak otomatis mengubah manual dependency.
- Relation tetap valid selama kedua endpoint masih berupa Executable Task.
- Jika structural conversion membuat Task menjadi Summary Task, operation harus mengikuti owning WBS conversion contract dan tidak boleh meninggalkan dependency dengan Summary Task endpoint.
- Resolution untuk structural conversion yang melibatkan dependency harus transactional dan tidak boleh silently discard dependency.
- Ketika owning WBS conversion memindahkan seluruh executable data ke conversion
  child, seluruh incoming dan outgoing dependency milik Task lama di-retarget ke
  conversion child dalam transaksi yang sama.
- Task lama menjadi Group tanpa dependency endpoint. Tidak ada dependency yang
  dihapus, digabung, atau ditimpa. Kegagalan retarget me-roll back conversion,
  hierarchy, executable data, ordering, dan dependency graph.

### 7.14 Scheduler Integration Contract

Story ini tidak menghitung timeline.

Successful dependency mutation harus:

- menginvalidasi dependency graph projection;
- menginvalidasi affected Project timeline/Gantt projections bila tersedia;
- memanggil minimal scheduler integration port yang disetujui ketika Automatic Scheduling aktif;
- tidak menjalankan speculative scheduling algorithm.

Epic 6 bertanggung jawab menggunakan graph tersebut untuk portfolio scheduling.

---

## 8. List and Search Behaviour

Task selector menggunakan backend search dan pagination.

Rules:

- Search menggunakan case-insensitive substring matching terhadap Task Name.
- Contoh: pencarian `task` menemukan candidate bernama `Sub Task Karton`.
- Project Name juga menjadi searchable context dengan aturan substring yang sama.
- Default page size mengikuti shared lookup convention.
- Maximum page size mengikuti architecture.
- Search dijalankan sebelum ordering, limit, dan offset.
- Result ordering deterministik:
  1. Project Name ascending case-insensitive.
  2. Task hierarchy/order dalam Project.
  3. Task ID ascending sebagai tie-breaker.
- Search state dipertahankan setelah recoverable failure.
- Closed Project dan invalid candidates disaring backend, bukan frontend-only.
- Selector tidak memuat seluruh portfolio tanpa batas.

---

## 9. Acceptance Criteria

### AC-1 — Menampilkan Blocked by dan Blocks

**Given** Engineering Lead membuka contextual Edit Task form pada Project Structure  
**When** dependency data berhasil dimuat  
**Then** Task menampilkan `Blocked by` dan `Blocks`  
**And** keduanya merepresentasikan relation yang sama dari arah berbeda.

### AC-2 — Initial loading

**Given** dependency request belum selesai  
**Then** section menampilkan local loading state  
**And** tidak menampilkan empty state secara prematur  
**And** existing confirmed Task data tetap terlihat.

### AC-3 — Empty dependency state

**Given** Task belum memiliki dependency  
**When** data berhasil dimuat  
**Then** `Blocked by` dan `Blocks` menampilkan empty state yang sesuai  
**And** menyediakan action untuk menambahkan dependency.

### AC-4 — Dependency load failure dan Retry

**Given** dependency gagal dimuat  
**Then** sistem menampilkan recoverable error tanpa detail internal  
**And** menyediakan Retry  
**When** Retry berhasil  
**Then** dependency ditampilkan.

### AC-5 — Create melalui Blocked by

**Given** Task B unfinished dan Task A merupakan candidate valid  
**When** Engineering Lead menambahkan Task A pada `Blocked by` Task B  
**Then** satu relation `A blocks B` dibuat  
**And** Task A menampilkan Task B pada `Blocks`  
**And** kedua view diperbarui tanpa hard refresh.

### AC-6 — Create melalui Blocks

**Given** Task A dan Task B merupakan candidate valid  
**When** Engineering Lead menambahkan Task B pada `Blocks` Task A  
**Then** satu relation `A blocks B` dibuat  
**And** Task B menampilkan Task A pada `Blocked by`.

### AC-7 — Cross-project dependency

**Given** Task A berada pada active Project Alpha  
**And** Task B berada pada active Project Beta  
**When** Engineering Lead membuat `A blocks B`  
**Then** dependency diterima  
**And** kedua Project tercatat sebagai affected scheduling projections.

### AC-8 — Multiple blockers

**Given** Task C unfinished  
**When** Task A dan Task B ditambahkan pada `Blocked by` Task C  
**Then** kedua relation disimpan  
**And** Task C menampilkan kedua blocker.

### AC-9 — Multiple blocked tasks

**Given** Task A valid  
**When** Task B dan Task C ditambahkan pada `Blocks` Task A  
**Then** kedua relation disimpan  
**And** Task A menampilkan kedua downstream Task.

### AC-10 — Self dependency ditolak

**Given** Task A dibuka  
**When** request mencoba membuat `A blocks A`  
**Then** request ditolak  
**And** tidak ada relation dibuat.

### AC-11 — Duplicate dependency ditolak

**Given** `A blocks B` sudah tersedia  
**When** relation yang sama dibuat kembali melalui `Blocks` atau `Blocked by`  
**Then** request ditolak sebagai duplicate  
**And** hanya satu relation tetap tersedia.

### AC-12 — Summary Task ditolak

**Given** candidate merupakan Summary Task  
**When** dependency dibuat dengan candidate tersebut sebagai blocker atau blocked task  
**Then** request ditolak  
**And** graph tidak berubah.

### AC-13 — Direct cycle ditolak

**Given** `A blocks B` tersedia  
**When** Engineering Lead mencoba membuat `B blocks A`  
**Then** request ditolak  
**And** error menunjukkan path `A → B → A`.

### AC-14 — Indirect cycle lintas Project ditolak

**Given** `A blocks B` dan `B blocks C` tersedia, termasuk bila berada pada Project berbeda  
**When** Engineering Lead mencoba membuat `C blocks A`  
**Then** request ditolak  
**And** error menunjukkan seluruh cycle path  
**And** tidak ada partial relation tersimpan.

### AC-15 — Completed Task boleh menjadi blocker

**Given** Task A completed dengan Actual End  
**And** Project Task A masih active  
**And** Task B unfinished  
**When** Engineering Lead membuat `A blocks B`  
**Then** dependency diterima  
**And** Task A tidak berubah  
**And** dependency-ready anchor untuk future scheduler adalah Actual End Task A.

### AC-16 — Completed Task tidak boleh menjadi blocked task

**Given** Task A completed  
**When** Engineering Lead mencoba membuat `Task C blocks Task A`  
**Then** request ditolak  
**And** histori Task A tidak berubah.

### AC-17 — Closed Project Task tidak tersedia

**Given** Project Alpha berstatus Closed  
**When** Engineering Lead mencari blocker atau blocked task  
**Then** Task dari Project Alpha tidak tersedia sebagai candidate baru.

### AC-18 — Selector seluruh active Project

**Given** beberapa active Project memiliki Task  
**When** Engineering Lead membuka dependency selector  
**Then** candidate berasal dari seluruh active Project  
**And** setiap result menampilkan Task Name sebagai primary label  
**And** setiap result menampilkan full hierarchy path dari Project sebagai WBS level 0 sampai candidate Task  
**And** completed Task ditandai secara jelas.

### AC-18A — Duplicate Task Name dapat dibedakan melalui hierarchy

**Given** dua candidate Task mempunyai nama yang sama dalam Project yang sama  
**And** kedua Task berada pada parent hierarchy yang berbeda  
**When** Engineering Lead membuka dependency selector  
**Then** kedua candidate tetap ditampilkan  
**And** setiap candidate menampilkan full WBS hierarchy path  
**And** Engineering Lead dapat membedakan candidate berdasarkan hierarchy tersebut.

Contoh:

```text
Task 2
NTB > Task 1 > Task 2

Task 2
NTB > Task 2 > Task 2
```

### AC-19 — Blocks menampilkan expected start projection

**Given** Task A memblok Task B  
**When** scheduler projection Task B tersedia  
**Then** `Blocks` Task A menampilkan Expected Start Task B.

**Given** scheduler projection belum tersedia  
**Then** UI menampilkan `Not scheduled` atau `—`  
**And** tidak menghitung expected start secara lokal.

### AC-20 — Delete dependency

**Given** dependency editable `A blocks B` tersedia  
**When** Engineering Lead mengonfirmasi delete  
**Then** relation dihapus  
**And** Task A tidak lagi menampilkan B pada `Blocks`  
**And** Task B tidak lagi menampilkan A pada `Blocked by`.

### AC-21 — Delete dependency dibatalkan

**Given** delete confirmation terbuka  
**When** Engineering Lead memilih Cancel  
**Then** tidak ada delete request  
**And** relation tetap tersedia.

### AC-22 — Historical dependency completed blocked task tidak dapat dihapus

**Given** dependency existing melibatkan Task yang sekarang completed sebagai blocked task  
**When** Engineering Lead mencoba menghapus atau memodifikasi dependency tersebut  
**Then** request ditolak  
**And** histori graph tetap tersedia.

### AC-23 — Delete Task memutus seluruh dependency

**Given** Task B mempunyai incoming dan outgoing dependency  
**When** Task B berhasil dihapus melalui WBS workflow  
**Then** seluruh dependency yang melibatkan Task B ikut dihapus atomik  
**And** tidak ada relation pengganti dibuat.

### AC-24 — Rename dan Move mempertahankan dependency

**Given** Task mempunyai dependency  
**When** Task di-rename atau dipindahkan dalam hierarchy  
**Then** dependency tetap mereferensikan Task ID yang sama  
**And** relation tidak hilang.

### AC-25 — Backend validation

**Given** frontend validation dilewati  
**When** invalid dependency dikirim langsung ke API  
**Then** backend tetap menolak sesuai invariant  
**And** database tidak menyimpan graph invalid.

### AC-26 — Duplicate submission prevention

**Given** create atau delete masih diproses  
**When** user memicu action yang sama kembali  
**Then** hanya satu mutation dikirim.

### AC-27 — Failure recovery

**Given** create atau delete gagal  
**Then** confirmed graph sebelumnya tetap terlihat  
**And** selection/search draft dipertahankan bila relevan  
**And** user dapat mencoba kembali.

### AC-28 — Cache consistency

**Given** dependency mutation berhasil  
**Then** affected Task detail, Blocks, Blocked by, Project workspace, dan available timeline projection diinvalidasi atau direfresh  
**And** stale in-flight response tidak dapat mengembalikan graph lama.

### AC-29 — Scheduler contract

**Given** dependency mutation berhasil dan Automatic Scheduling aktif  
**Then** scheduler integration contract menerima affected portfolio context  
**And** US-5.1 tidak menghitung timeline sendiri.

### AC-30 — Accessibility dan responsive behaviour

**Given** user menggunakan keyboard atau supported viewport  
**Then** dependency dapat dilihat, ditambah, dan dihapus tanpa pointer-only interaction  
**And** labels, status, errors, dan focus management tetap accessible  
**And** normal workspace tidak membutuhkan horizontal page scrolling di luar controlled grid/timeline region.

---

## 10. Preferred API Contract

Endpoint dapat disesuaikan dengan repository convention, tetapi behaviour harus setara dengan berikut.

### List Dependency Task

```http
GET /api/tasks/{taskId}/dependencies
```

Response menyediakan:

- `blockedBy`
- `blocks`

### Search Candidate

```http
GET /api/dependency-candidates?taskId={taskId}&direction=blockedBy&search=api&page=1&pageSize=5
```

Candidate response includes a backend-generated `hierarchyPath` display projection:

```json
{
  "id": "task-2-under-task-1",
  "name": "Task 2",
  "projectId": "ntb",
  "projectName": "NTB",
  "hierarchyPath": "NTB > Task 1 > Task 2",
  "completed": false
}
```

`hierarchyPath` is generated from the authoritative WBS hierarchy. The frontend must not reconstruct it from partial DTO fields.

Direction:

- `blockedBy`
- `blocks`

### Create Dependency

```http
POST /api/dependencies
Content-Type: application/json
```

```json
{
  "blockingTaskId": "task-a",
  "blockedTaskId": "task-b"
}
```

### Delete Dependency

```http
DELETE /api/dependencies/{dependencyId}
```

Expected status direction:

- List success: `200 OK`.
- Search success: `200 OK`.
- Create success: `201 Created`.
- Delete success: `204 No Content`.
- Invalid input: `400 Bad Request`.
- Not found: `404 Not Found`.
- Business conflict: `409 Conflict`.

Use existing structured error format.

---

## 11. Error Response Contract

Minimum stable error concepts:

- `TASK_NOT_FOUND`
- `DEPENDENCY_NOT_FOUND`
- `DEPENDENCY_SELF_REFERENCE`
- `DEPENDENCY_ALREADY_EXISTS`
- `DEPENDENCY_EXECUTABLE_TASK_REQUIRED`
- `DEPENDENCY_CYCLE_DETECTED`
- `DEPENDENCY_CLOSED_PROJECT_TASK_NOT_ALLOWED`
- `DEPENDENCY_COMPLETED_TASK_CANNOT_BE_BLOCKED`
- `DEPENDENCY_COMPLETED_HISTORY_READ_ONLY`
- `INVALID_DEPENDENCY_DIRECTION`
- `INVALID_PAGE`
- `INVALID_PAGE_SIZE`
- `INVALID_REQUEST`

Cycle error response harus menyediakan safe metadata untuk menampilkan path, misalnya:

```json
{
  "code": "DEPENDENCY_CYCLE_DETECTED",
  "message": "Dependency cannot be added because it creates a cycle.",
  "details": {
    "path": [
      { "taskId": "task-a", "taskName": "Task A", "projectName": "Alpha" },
      { "taskId": "task-b", "taskName": "Task B", "projectName": "Beta" },
      { "taskId": "task-c", "taskName": "Task C", "projectName": "Gamma" },
      { "taskId": "task-a", "taskName": "Task A", "projectName": "Alpha" }
    ]
  }
}
```

Tidak boleh mengekspos SQL, stack trace, atau infrastructure details.

---

## 12. Test Cases

### TC-1 — Load dependency kosong

**Precondition:** Task tidak mempunyai dependency.  
**Action:** Buka dependency section.  
**Expected:** `Blocks` dan `Blocked by` empty state tampil.

### TC-2 — Create dari Blocked by

**Precondition:** A dan B valid serta unfinished.  
**Action:** Tambahkan A pada `Blocked by` B.  
**Expected:** Relation A → B dibuat dan terlihat dari kedua Task.

### TC-3 — Create dari Blocks

**Action:** Tambahkan B pada `Blocks` A.  
**Expected:** Relation yang sama dibuat.

### TC-4 — Create cross-project

**Precondition:** A pada Project Alpha, B pada Project Beta, keduanya active.  
**Action:** Buat A → B.  
**Expected:** Success dan kedua Project menjadi affected.

### TC-5 — Multiple blockers

**Action:** Buat A → C dan B → C.  
**Expected:** C menampilkan A dan B pada `Blocked by`.

### TC-6 — Multiple blocked tasks

**Action:** Buat A → B dan A → C.  
**Expected:** A menampilkan B dan C pada `Blocks`.

### TC-7 — Self dependency

**Action:** Buat A → A.  
**Expected:** `409 DEPENDENCY_SELF_REFERENCE`.

### TC-8 — Duplicate dari arah berbeda

**Precondition:** A → B tersedia.  
**Action:** Tambahkan A melalui `Blocked by` B atau B melalui `Blocks` A lagi.  
**Expected:** Duplicate ditolak.

### TC-9 — Summary sebagai blocker

**Action:** Pilih Summary Task sebagai blocker.  
**Expected:** Ditolak.

### TC-10 — Summary sebagai blocked task

**Action:** Pilih Summary Task sebagai blocked task.  
**Expected:** Ditolak.

### TC-11 — Direct cycle

**Precondition:** A → B.  
**Action:** Buat B → A.  
**Expected:** Ditolak dengan path A → B → A.

### TC-12 — Indirect cycle satu Project

**Precondition:** A → B → C.  
**Action:** Buat C → A.  
**Expected:** Ditolak dengan path lengkap.

### TC-13 — Indirect cycle lintas Project

**Precondition:** Chain berada pada tiga active Project.  
**Action:** Tutup cycle.  
**Expected:** Ditolak secara atomik.

### TC-14 — Completed blocker

**Precondition:** A Actual End terisi, Project A belum Closed; B unfinished.  
**Action:** Buat A → B.  
**Expected:** Success dan Actual End menjadi future ready anchor.

### TC-15 — Completed blocked task

**Precondition:** B completed.  
**Action:** Buat A → B.  
**Expected:** Ditolak.

### TC-16 — Closed Project candidate

**Precondition:** Project A Closed.  
**Action:** Search Task A.  
**Expected:** Tidak muncul.

### TC-17 — Completed active Project candidate

**Precondition:** Task A completed, Project A active.  
**Action:** Search blocker candidate.  
**Expected:** A muncul dan ditandai completed.

### TC-18 — Search berdasarkan substring Task Name

**Action:** Cari `task` ketika candidate bernama `Sub Task Karton`.  
**Expected:** Candidate muncul pada backend paginated result secara case-insensitive.

### TC-18A — Duplicate Task Name pada hierarchy berbeda

**Precondition:** Dua executable Task bernama `Task 2` berada di bawah Group yang berbeda pada Project NTB.  
**Action:** Buka dependency selector dan cari `Task 2`.  
**Expected:** Kedua result muncul dengan full path yang berbeda sehingga dapat dibedakan.

### TC-19 — Expected Start tersedia

**Precondition:** Scheduler projection tersedia.  
**Expected:** Blocks menampilkan expected start.

### TC-20 — Expected Start belum tersedia

**Precondition:** Epic 6 belum menghasilkan projection.  
**Expected:** Tampil `Not scheduled`, bukan tanggal buatan frontend.

### TC-21 — Delete dependency editable

**Action:** Delete A → B.  
**Expected:** Relation hilang dari dua sisi.

### TC-22 — Cancel delete

**Action:** Cancel confirmation.  
**Expected:** Tidak ada request dan relation tetap.

### TC-23 — Delete historical completed dependency

**Precondition:** Blocked task dependency sudah completed.  
**Action:** Delete.  
**Expected:** Ditolak read-only.

### TC-24 — Delete Task dengan incoming dependency

**Action:** Delete Task B melalui WBS flow.  
**Expected:** Incoming relation ikut terhapus.

### TC-25 — Delete Task dengan outgoing dependency

**Expected:** Outgoing relation ikut terhapus.

### TC-26 — Delete Task di tengah chain

**Precondition:** A → B → C.  
**Action:** Delete B.  
**Expected:** Kedua relation B terhapus; A → C tidak dibuat.

### TC-27 — Rename Task

**Action:** Rename blocker.  
**Expected:** Relation tetap dan label baru terlihat.

### TC-28 — Move Task

**Action:** Move Task/subtree.  
**Expected:** Relation tetap berdasarkan immutable ID.

### TC-29 — Concurrent duplicate create

**Action:** Dua request simultan membuat A → B.  
**Expected:** Satu berhasil, satu conflict, satu relation tersimpan.

### TC-30 — Concurrent cycle candidates

**Action:** Concurrent mutations yang bila keduanya commit akan membuat cycle.  
**Expected:** Transaction/locking memastikan graph akhir acyclic.

### TC-31 — Rollback create failure

**Action:** Persistence atau scheduler-port failure.  
**Expected:** Tidak ada partial relation.

### TC-32 — Rollback delete failure

**Expected:** Relation tetap utuh.

### TC-33 — Stale response protection

**Action:** Mutation sukses saat old list request masih in-flight.  
**Expected:** Old response tidak mengembalikan graph lama.

### TC-34 — Keyboard selector

**Action:** Cari, navigasi, pilih, dan hapus dependency via keyboard.  
**Expected:** Workflow selesai tanpa pointer.

### TC-35 — Responsive workspace

**Expected:** Dependency field tetap usable di supported viewport.

---

## 13. Required Automated Tests

### Domain Tests

- Valid dependency.
- Self-reference rejection.
- Executable Task requirement.
- Duplicate relation invariant.
- Completed blocker allowed.
- Completed blocked task rejected.
- Historical completed dependency read-only.
- Identity and endpoint invariants.

### Application Tests

- List `Blocks` dan `Blocked by`.
- Create dari kedua direction.
- Delete dependency.
- Cross-project validation.
- Closed Project candidate rejection.
- Cycle detection direct dan indirect.
- Cycle path mapping.
- Completed-task workflows.
- Task-delete dependency cleanup.
- Scheduler integration port invocation.
- Rollback.
- Concurrency.

### Repository Integration Tests

- Persist dan retrieve relation.
- Unique `(blocking_task_id, blocked_task_id)` constraint.
- Incoming/outgoing query.
- Candidate search seluruh active Project.
- Closed Project exclusion.
- Completed candidate filtering per direction.
- Cycle traversal support.
- Cross-project graph traversal.
- Cascade cleanup melalui application transaction, bukan unsafe database cascade yang mengabaikan domain flow.
- Index strategy.
- PostgreSQL query plans.
- Future MySQL contract compatibility.

### API Integration Tests

- List Task dependency.
- Candidate search dan pagination.
- Create dependency.
- Delete dependency.
- Malformed/unknown fields.
- Not found.
- Self dependency.
- Duplicate.
- Summary Task.
- Cycle response dengan path.
- Cross-project.
- Completed blocker.
- Completed blocked task.
- Closed Project.
- Structured errors.
- Transaction rollback.

### Frontend Tests

- Render `Blocks` dan `Blocked by`.
- Loading, empty, error, Retry.
- Search candidate.
- Project/Task label.
- Completed marker.
- Create dari kedua arah.
- Multiple selection if approved interaction supports it.
- Cycle error path.
- Delete confirmation/cancel/success.
- Read-only historical dependency.
- Expected Start versus `Not scheduled`.
- Duplicate submission prevention.
- Draft/search preservation.
- Cache invalidation.
- Stale-response protection.
- Keyboard and focus management.
- Responsive behaviour.

Tests harus berfokus pada observable behaviour dan tidak hanya snapshot.

---

## 14. Technical Completion Criteria

1. Dependency mempunyai dedicated feature boundary.
2. Domain tidak bergantung pada HTTP, database, ORM, atau frontend.
3. Dependency persistence model terpisah dari domain entity.
4. Database migration dibuat dan version-controlled.
5. Composite unique constraint menjaga satu relation per endpoint pair.
6. Foreign-key/index strategy mendukung incoming, outgoing, candidate lookup, dan graph traversal.
7. Cross-project dependency tidak dibatasi oleh Project ID yang sama.
8. Closed Project candidate exclusion ditegakkan backend.
9. Cycle detection mencakup seluruh active portfolio graph yang relevan.
10. Cycle validation dan create mutation berada dalam safe transaction/concurrency strategy.
11. Summary Task tidak dapat menjadi endpoint.
12. Completed blocker rule dan completed blocked restriction ditegakkan backend.
13. Task deletion membersihkan seluruh related dependency secara atomik.
14. Tidak ada auto-reconnect.
15. Lag tidak ditambahkan.
16. Auto Dependency tidak ditambahkan.
17. Scheduling algorithm tidak ditambahkan.
18. Scheduler integration hanya berupa port/contract yang sesuai scope.
19. API menggunakan structured safe errors.
20. Candidate search bounded dan paginated.
21. Request-cache identity mencakup Task, direction, search, page, dan page size.
22. Mutations menginvalidasi incoming/outgoing relation, candidate state, workspace, dan affected projection.
23. Versioned invalidation atau equivalent mencegah stale restoration.
24. UI menggunakan `Blocks` dan `Blocked by`, bukan predecessor/successor sebagai primary copy.
25. Gantt tetap read-only pada Phase 1.
26. Accessibility dan responsive requirements terpenuhi.
27. Backend format, lint, static analysis, migrations, tests, dan required race tests lulus.
28. Frontend format, lint, type check, tests, dan production build lulus.
29. Query-review gate dan plan verification diselesaikan.
30. Backend direstart dan affected endpoints di-smoke-test.
31. Project architecture dan API documentation diperbarui.
32. AGENTS.md hanya diubah jika ada durable project-wide rule baru.
33. Completion report mencantumkan query/index review, documentation impact, test result, dan deferred scope.

---

## 15. Documentation Impact

Implementasi harus menilai dan memperbarui:

- Project-specific architecture:
  - Dependency aggregate/relation.
  - `Blocks` / `Blocked by` mapping.
  - Cross-project graph.
  - Cycle detection.
  - Completed-task rules.
  - Task-delete cleanup.
  - Portfolio scheduler integration contract.
  - Cache invalidation.
  - Concurrency and transaction strategy.
  - Index and query strategy.
- API documentation.
- Project Structure documentation untuk contextual dependency editor dan future
  reuse contract bagi Integrated Gantt Workspace Task Grid.
- Related Scheduling Engine story agar menggunakan dependency graph ini sebagai input.
- Related Auto Dependency story agar tidak menimpa manual dependency semantics.

README dan environment documentation hanya diubah bila setup berubah.

---

## 16. Locked Product Decisions

- Dependency hanya antar Executable Task.
- Summary Task tidak boleh menjadi endpoint.
- Dependency boleh cross-project.
- Scheduler bersifat portfolio-level.
- UI menggunakan `Blocks` dan `Blocked by`.
- Keduanya editable dan merepresentasikan satu relation.
- Finish-to-Start adalah satu-satunya type MVP.
- Multiple blockers dan multiple blocked tasks diperbolehkan.
- Cycle hard reject dan error menunjukkan path.
- Lag bukan bagian dependency dan tidak masuk story ini.
- Auto Dependency tidak masuk story ini.
- Completed Task pada active Project boleh menjadi blocker baru.
- Actual End completed blocker menjadi dependency-ready anchor untuk Epic 6.
- Completed Task tidak boleh menjadi blocked task baru.
- Task dari Closed Project tidak tersedia untuk dependency baru.
- Existing historical dependency tetap disimpan.
- Delete Task memutus seluruh incoming dan outgoing dependency.
- Tidak ada auto-reconnect manual dependency.
- Expected Start pada `Blocks` berasal dari Epic 6; sebelum tersedia tampil `Not scheduled`.
- Phase 1 menggunakan contextual Edit Task form pada Project Structure; future
  Integrated Gantt Task Grid menggunakan contract yang sama.
- Structural Task-to-Group conversion me-retarget dependency ke conversion child
  secara atomik.
- Story ini tidak menghitung timeline.

---

## 17. Unresolved Questions

None.
