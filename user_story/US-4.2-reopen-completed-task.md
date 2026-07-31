# US-4.2 — Reopen Completed Task

## 1. User Story

**Sebagai** Engineering Lead,  
**Saya ingin** membuka kembali Task yang sudah completed,  
**Sehingga** Actual End yang salah dapat dikoreksi dan Task yang sebenarnya belum selesai dapat kembali diperlakukan sebagai unfinished work tanpa membuat Task baru.

---

## 2. Business Context

SchedMind tidak mempunyai entity Task yang terpisah. **Task adalah Executable WBS**, yaitu WBS leaf yang tidak mempunyai child.

Completion Task ditentukan hanya oleh `Actual End`:

- `Actual End != null` berarti Task **completed**.
- `Actual End == null` berarti Task **unfinished**.

US-4.1 sebelumnya menetapkan completed Task sebagai immutable dan Actual End tidak dapat dikosongkan. Story ini memperkenalkan satu pengecualian domain yang sempit dan eksplisit: **Reopen Task**.

Reopen Task bukan pembuatan Task pengganti, bukan perubahan Project Status, dan bukan generic edit terhadap completed Task. Reopen Task mempertahankan Task ID serta seluruh data Task, lalu menghapus `Actual End` secara atomik.

---

## 3. Terminology

| Product Term | Domain Meaning |
| --- | --- |
| Project Structure | UI untuk hierarchy WBS |
| Task | Executable WBS / WBS leaf |
| Group | Grouping WBS / WBS yang mempunyai child |
| Completed Task | Executable WBS dengan `Actual End` |
| Unfinished Task | Executable WBS tanpa `Actual End` |
| Reopen Task | Dedicated command yang mengubah `Actual End` menjadi `null` |
| Reopen Project | Lifecycle command terpisah yang mengubah Closed Project menjadi Open |

`Reopen Task` dan `Reopen Project` tidak boleh menggunakan command, side effect, atau copy yang ambigu.

---

## 4. Scope

### 4.1 In Scope

- Menampilkan action `Reopen Task` untuk completed Task yang eligible.
- Menampilkan confirmation sebelum Reopen Task dijalankan.
- Menghapus `Actual End` melalui dedicated backend command.
- Mempertahankan Task ID, hierarchy, planning data, dependency, dan seluruh field lain.
- Mengubah derived completion state dari completed menjadi unfinished.
- Mendukung Reopen Task pada Open dan Locked Project.
- Menolak Reopen Task pada Closed Project.
- Mempertahankan Locked Project sebagai Locked.
- Mempertahankan Execution dan Commitment dates/baselines.
- Memanggil Forecast recalculation contract karena Actual End berubah.
- Menginvalidasi Forecast, Delivery Impact, Project Health, dependency candidate state, Task detail, Project Structure, dan available Gantt/workspace projections.
- Menangani failure, rollback, concurrency, duplicate submission, stale response, accessibility, dan responsive behaviour.
- Menyediakan automated evidence untuk setiap Acceptance Criterion melalui Three-Level Confidence Matrix.

### 4.2 Out of Scope

- Mengubah atau mengganti Actual End dengan tanggal lain dalam satu operation.
- Generic edit terhadap field lain pada completed Task.
- Reopen Grouping WBS.
- Reopen seluruh Task secara bulk.
- Reopen Closed Project secara otomatis.
- Mengubah Locked Project menjadi Open.
- Menghapus atau membuat ulang dependency.
- Mengubah Effort menjadi remaining effort.
- Menyimpan percentage complete atau remaining effort.
- Menghitung ulang Execution Timeline.
- Menghitung ulang Commitment Timeline.
- Mengimplementasikan Forecast scheduling algorithm.
- Audit-log feature baru di luar timestamp/convention yang sudah tersedia.
- Notification, approval workflow, atau permission model baru.

---

## 5. Locked Product Rules

### 5.1 Task and Completion Source of Truth

- Task adalah Executable WBS.
- Grouping WBS bukan Task dan tidak mempunyai Actual End.
- `Actual End` adalah satu-satunya source of truth untuk completion.
- UI tidak menyimpan boolean completion terpisah sebagai authoritative state.
- Reopen Task selesai hanya ketika persisted `Actual End` telah menjadi `null`.

### 5.2 Dedicated Reopen Command

- Reopen harus menggunakan dedicated command/use case.
- Generic Task update tidak boleh menerima `actualEnd: null` sebagai bypass terhadap completed-task read-only invariant.
- Semua mutation lain terhadap completed Task tetap ditolak dengan `COMPLETED_TASK_READ_ONLY`.
- Reopen command adalah satu-satunya exception yang boleh menghapus Actual End.
- Command harus mengidentifikasi Task berdasarkan immutable Task/WBS ID.
- Reopen tidak membuat Task baru dan tidak mengganti ID.

### 5.3 Eligible Project Status

#### Open Project

- Completed Task dapat direopen.
- Project tetap Open.
- Execution dan Commitment dates tidak diubah oleh Reopen Task.
- Forecast recalculation contract dipanggil.

#### Locked Project

- Completed Task dapat direopen melalui explicit `Reopen Task` exception.
- Project tetap Locked.
- Reopen Task tidak dianggap sebagai `Locked → Open` transition.
- Locked Execution dan Commitment baselines tetap immutable.
- Forecast tetap dynamic dan harus direcalculate melalui established Forecast contract.

#### Closed Project

- Reopen Task ditolak.
- UI tidak menyediakan enabled Reopen Task action.
- Backend tetap menolak direct API bypass dengan `PROJECT_CLOSED_READ_ONLY`.
- Engineering Lead harus menjalankan `Reopen Project` terlebih dahulu sehingga Project menjadi Open, kemudian menjalankan `Reopen Task` secara terpisah.

### 5.4 Atomic State Change

Successful Reopen Task harus mengubah tepat satu business field:

```text
Actual End: <existing date> → null
```

Mutation harus mempertahankan:

- Task/WBS ID.
- Project ID.
- Parent ID dan hierarchy path.
- Sibling position.
- Name.
- Role.
- Assignee.
- Effort.
- Manual atau calculated Execution dates.
- Manual atau calculated Commitment dates.
- Incoming dependencies.
- Outgoing dependencies.
- Created At.
- Field lain yang tidak dimiliki oleh Reopen Task.

`Updated At` boleh berubah mengikuti repository convention.

Reopen, Forecast coordination, dan persistence harus mempunyai transaction/rollback behaviour yang konsisten. Jika required Forecast coordination gagal, Actual End tetap pada confirmed value sebelumnya dan Task tetap completed.

### 5.5 Forecast and Timeline Behaviour

Menghapus Actual End adalah Actual End change. Karena itu:

- Reopen hanya memanggil **Forecast recalculation contract** yang established oleh US-4.1/Epic 6.
- Reopen tidak memanggil full Execution/Commitment recalculation contract.
- Execution dan Commitment dates tidak berubah sebagai direct side effect Reopen.
- Pada Locked Project, protected Execution dan Commitment baselines tidak berubah.
- Pada Automatic Scheduling OFF, manual Execution dan Commitment dates tetap tidak berubah.
- Concrete Forecast algorithm tetap dimiliki Epic 6.
- Sebelum concrete Forecast Scheduler tersedia, no-op adapter hanya membuktikan port wiring, bukan timeline calculation.

Future Forecast Scheduler memperlakukan reopened Task sebagai unfinished work:

- Task tidak lagi dikunci ke Actual End.
- Task dijadwalkan ulang menggunakan original Effort, bukan estimated remaining effort.
- Dependency, capacity, holiday, override, freeze, Project Priority, dan same-assignee rules tetap berlaku.
- Completed Task lain tetap locked ke Actual End masing-masing.

### 5.6 Delivery Impact and Health Projection

- Reopen menginvalidasi Forecast End before/after projection yang digunakan Delivery Impact.
- Delivery Impact tidak dihitung secara lokal oleh frontend.
- Project Forecast End, Delivery Impact, dan Project Health harus direfresh atau diinvalidasi sesuai available projection contract.
- Stale projection tidak boleh ditampilkan sebagai confirmed result setelah successful Reopen.

### 5.7 Dependency Behaviour

Reopen Task tidak mengubah dependency graph.

- Seluruh incoming dependency dipertahankan.
- Seluruh outgoing dependency dipertahankan.
- Tidak ada dependency yang dihapus, dibuat, di-retarget, atau di-auto-reconnect.
- Dependency tetap mereferensikan immutable Task ID yang sama.
- Setelah `Actual End` menjadi `null`, Task tidak lagi completed untuk rule dependency berikutnya.
- Existing dependency dengan Task tersebut sebagai blocked Task tidak lagi read-only hanya karena completion history sebelumnya; rule historical read-only dievaluasi berdasarkan current persisted Actual End.
- Candidate list dan completed marker harus direfresh sehingga reopened Task tampil sebagai unfinished candidate sesuai US-5.1.
- Dependency graph tidak perlu divalidasi ulang karena endpoint dan relation tidak berubah.

### 5.8 UI Placement and Flow

Phase 1 menggunakan **Project Structure** dan contextual **Edit Task** form yang sudah tersedia pada US-4.1.

Completed Task form:

- Menampilkan existing Task details dan Actual End secara read-only.
- Tidak menyediakan Save untuk mutation biasa terhadap completed Task.
- Menyediakan explicit action `Reopen Task` bila Project Open atau Locked.
- Tidak menyediakan enabled `Reopen Task` bila Project Closed.

Reopen flow:

1. Engineering Lead membuka contextual Edit Task pada completed Task.
2. Engineering Lead memilih `Reopen Task`.
3. Confirmation menjelaskan bahwa Actual End akan dihapus dan Task kembali unfinished.
4. Confirmation menampilkan Task Name dan existing Actual End sebagai context.
5. `Cancel` menutup confirmation tanpa request dan mengembalikan focus ke trigger.
6. Confirm mengirim tepat satu Reopen command.
7. Selama request pending, confirm action disabled dan duplicate submission dicegah.
8. Success menutup confirmation, menampilkan recoverable success feedback, dan memperbarui confirmed Task state tanpa hard refresh.
9. Failure mempertahankan completed state dan Actual End, menampilkan safe error, dan memungkinkan retry.

Recommended confirmation copy:

```text
Reopen Task?

Actual End for “<Task Name>” will be removed. The Task will return to unfinished work and its Forecast may change.

Cancel | Reopen Task
```

### 5.9 Concurrency and Idempotency

- Dua concurrent Reopen command tidak boleh menghasilkan partial state atau duplicate side effect.
- Satu request dapat berhasil; request lain menerima deterministic conflict berdasarkan confirmed state.
- Reopen terhadap unfinished Task ditolak dengan `TASK_NOT_COMPLETED`.
- Repeated request tidak boleh memanggil Forecast coordination berulang setelah Task sudah unfinished.
- Persistence harus menggunakan transaction, conditional update, optimistic locking, row lock, atau equivalent repository strategy yang mencegah lost update.

### 5.10 Cache and Stale Response Protection

Successful Reopen harus memperbarui atau menginvalidasi minimal:

- Current Task detail.
- Project Structure/WBS tree.
- Contextual Edit Task state.
- Completed/unfinished badges or derived state.
- Dependency sections and candidate search state.
- Project detail.
- Forecast projection.
- Delivery Impact projection.
- Project Health projection.
- Integrated Gantt workspace projections bila tersedia.

Old request yang selesai setelah Reopen tidak boleh mengembalikan Actual End lama atau completed state ke UI.

---

## 6. Domain and Application Contract

### 6.1 Domain Preconditions

Reopen dapat dieksekusi hanya jika:

1. WBS tersedia.
2. WBS merupakan Executable WBS.
3. WBS mempunyai Actual End.
4. Owning Project berstatus Open atau Locked.

### 6.2 Domain Result

Successful Reopen menghasilkan:

- Task ID sama.
- Actual End `null`.
- Derived completion state unfinished.
- Seluruh data lain unchanged kecuali system-managed update metadata.
- Project status unchanged.

### 6.3 Application Coordination

Application use case harus:

1. Load Task dan owning Project dalam consistency boundary yang sesuai.
2. Menegakkan Project status dan Task completion invariant di backend.
3. Menghapus Actual End secara atomik.
4. Menjalankan established Forecast recalculation/invalidation port untuk affected Project/portfolio context.
5. Roll back mutation bila required coordination gagal.
6. Mengembalikan confirmed Task projection.
7. Menyediakan invalidation metadata atau menjalankan invalidation yang diperlukan.

Domain tidak bergantung pada HTTP, ORM, database, React, atau scheduler implementation.

---

## 7. Preferred API Contract

Endpoint boleh disesuaikan dengan repository convention, tetapi behaviour harus setara dengan dedicated command berikut:

```http
POST /api/tasks/{taskId}/reopen
```

Request body kosong:

```json
{}
```

atau tidak mempunyai body.

Success:

```http
200 OK
```

```json
{
  "id": "task-123",
  "actualEnd": null,
  "completed": false,
  "updatedAt": "2026-07-29T06:30:00Z"
}
```

`completed` adalah response projection yang diturunkan dari `actualEnd`, bukan persisted source of truth terpisah.

Status expectations:

- `200 OK`: Reopen berhasil dan confirmed state dikembalikan.
- `400 Bad Request`: malformed request atau unknown field.
- `404 Not Found`: Task tidak tersedia.
- `409 Conflict`: Task bukan Executable WBS, Task belum completed, Project Closed, concurrent state conflict, atau business invariant lain.
- `500`/safe operation failure: unexpected infrastructure/coordination failure tanpa mengekspos internal detail.

Generic Task update endpoint tetap tidak boleh digunakan untuk clear Actual End pada completed Task.

---

## 8. Error Response Contract

Minimum stable error concepts:

| Error Code | HTTP | Condition |
| --- | ---: | --- |
| `TASK_NOT_FOUND` | 404 | Task/WBS ID tidak tersedia |
| `EXECUTABLE_TASK_REQUIRED` | 409 | Target adalah Grouping WBS/Group |
| `TASK_NOT_COMPLETED` | 409 | Actual End sudah `null` |
| `PROJECT_CLOSED_READ_ONLY` | 409 | Owning Project Closed |
| `COMPLETED_TASK_READ_ONLY` | 409 | Generic completed Task mutation selain Reopen |
| `TASK_REOPEN_CONFLICT` | 409 | Concurrent change membuat expected completed state tidak lagi valid |
| `TASK_REOPEN_FAILED` | 500 atau established safe operation status | Persistence/Forecast coordination gagal dan mutation di-roll back |
| `INVALID_REQUEST` | 400 | Malformed JSON, unknown field, atau unsupported payload |

Error response tidak boleh mengekspos SQL, stack trace, framework error, scheduler implementation detail, atau infrastructure identifier.

---

## 9. Acceptance Criteria

### AC-1 — Reopen action tersedia untuk completed Task aktif

**Given** Executable Task mempunyai Actual End  
**And** owning Project berstatus Open atau Locked  
**When** Engineering Lead membuka contextual Edit Task dari Project Structure  
**Then** Task ditampilkan sebagai completed dan read-only untuk normal editing  
**And** action `Reopen Task` tersedia.

### AC-2 — Reopen action tidak tersedia untuk target yang tidak eligible

**Given** target adalah unfinished Task, Grouping WBS, atau Task pada Closed Project  
**When** contextual form ditampilkan  
**Then** enabled `Reopen Task` action tidak tersedia  
**And** backend tetap menolak direct command bypass dengan structured business error.

### AC-3 — Confirmation dan cancel

**Given** eligible completed Task  
**When** Engineering Lead memilih `Reopen Task`  
**Then** confirmation menampilkan Task Name, existing Actual End, dan konsekuensi bahwa Task kembali unfinished  
**When** Engineering Lead memilih `Cancel`  
**Then** tidak ada request dikirim  
**And** Actual End serta completed state tetap  
**And** focus kembali ke Reopen trigger.

### AC-4 — Successful Reopen menghapus Actual End

**Given** eligible completed Task dengan Actual End  
**When** Engineering Lead mengonfirmasi Reopen Task  
**Then** dedicated backend command mengubah Actual End menjadi `null` secara atomik  
**And** response confirmed menunjukkan Task unfinished.

### AC-5 — Reopen mempertahankan identity dan seluruh data lain

**When** Reopen Task berhasil  
**Then** Task ID, Project ID, parent, hierarchy, sibling position, Name, Role, Assignee, Effort, Execution dates, Commitment dates, dan dependency tetap sama  
**And** tidak ada Task baru dibuat  
**And** hanya Actual End serta allowed system-managed update metadata yang berubah.

### AC-6 — Completed read-only invariant hanya mempunyai explicit Reopen exception

**Given** Task completed  
**When** generic update mencoba mengubah field Task atau mengirim `actualEnd: null`  
**Then** request ditolak dengan `COMPLETED_TASK_READ_ONLY`  
**But when** dedicated Reopen command dijalankan pada eligible Task  
**Then** Actual End dapat dihapus.

### AC-7 — Reopen pada Open Project memicu Forecast contract saja

**Given** completed Task berada pada Open Project  
**When** Reopen berhasil  
**Then** Project tetap Open  
**And** Forecast recalculation/invalidation contract dipanggil satu kali dengan affected context  
**And** Execution dan Commitment recalculation contract tidak dipanggil  
**And** stored Execution dan Commitment dates tidak berubah sebagai direct side effect.

### AC-8 — Reopen pada Locked Project mempertahankan status dan baselines

**Given** completed Task berada pada Locked Project  
**When** Reopen berhasil  
**Then** Project tetap Locked  
**And** operation tidak menjalankan atau menyimulasikan `Locked → Open`  
**And** locked Execution dan Commitment baselines tidak berubah  
**And** hanya Forecast contract yang dipanggil.

### AC-9 — Closed Project menolak Reopen Task

**Given** completed Task berada pada Closed Project  
**When** direct Reopen command dikirim  
**Then** request ditolak dengan `PROJECT_CLOSED_READ_ONLY`  
**And** Actual End, Project Status, timeline, dan dependency tetap unchanged  
**And** user harus menjalankan Reopen Project secara terpisah sebelum Reopen Task.

### AC-10 — Dependency graph dipertahankan

**Given** completed Task mempunyai incoming dan/atau outgoing dependency  
**When** Task direopen  
**Then** seluruh dependency tetap mereferensikan Task ID yang sama  
**And** tidak ada dependency yang dihapus, dibuat, di-retarget, atau di-auto-reconnect  
**And** completed marker/candidate eligibility direfresh berdasarkan Actual End yang sekarang `null`  
**And** completed-blocked historical read-only rule tidak lagi berlaku hanya karena Task sebelumnya completed.

### AC-11 — Forecast failure me-roll back Reopen

**Given** persistence dapat dilakukan tetapi required Forecast coordination gagal  
**When** Reopen dijalankan  
**Then** seluruh operation gagal secara atomik  
**And** Actual End tetap pada confirmed value sebelumnya  
**And** Task tetap completed  
**And** safe recoverable error ditampilkan tanpa internal detail.

### AC-12 — Duplicate submission dan concurrent Reopen aman

**Given** Reopen request masih pending  
**When** user mengaktifkan confirm kembali  
**Then** hanya satu request dikirim.

**Given** dua backend request concurrent mencoba mereopen Task yang sama  
**Then** graph akhir mempunyai satu Task dengan Actual End `null`  
**And** maksimal satu Forecast coordination dijalankan untuk successful state transition  
**And** request lain menerima deterministic conflict tanpa partial mutation.

### AC-13 — Reopen unfinished Task ditolak

**Given** Task tidak mempunyai Actual End  
**When** direct Reopen command dikirim  
**Then** request ditolak dengan `TASK_NOT_COMPLETED`  
**And** Forecast coordination tidak dipanggil  
**And** Task data tidak berubah.

### AC-14 — Success dan failure menjaga confirmed UI state

**Given** Reopen berhasil  
**Then** contextual form dan Project Structure berubah menjadi unfinished state tanpa hard refresh  
**And** success feedback ditampilkan.

**Given** Reopen gagal  
**Then** Actual End dan completed state sebelumnya tetap terlihat  
**And** raw backend error tidak ditampilkan  
**And** user dapat mencoba kembali.

### AC-15 — Cache dan stale response consistency

**Given** Reopen berhasil ketika old Task/detail/dependency/projection requests masih in-flight  
**Then** affected cache dan projection diinvalidasi atau diperbarui  
**And** old response tidak dapat mengembalikan Actual End lama, completed marker lama, atau Forecast lama sebagai confirmed state.

### AC-16 — Accessibility dan responsive behaviour

**Given** eligible Task dibuka dengan keyboard atau supported viewport  
**Then** Reopen trigger, confirmation, Cancel, dan Confirm dapat digunakan tanpa pointer-only interaction  
**And** dialog mempunyai accessible name/description dan focus management yang benar  
**And** pending, success, dan error state tidak dikomunikasikan hanya dengan warna  
**And** workflow tidak menyebabkan normal horizontal page scrolling di luar controlled workspace region.

---

## 10. Test Cases

| ID | Scenario | Expected |
| --- | --- | --- |
| TC-1 | Buka completed Task pada Open Project | Form read-only untuk normal edit; `Reopen Task` tersedia |
| TC-2 | Buka completed Task pada Locked Project | `Reopen Task` tersedia; Project tetap ditampilkan Locked |
| TC-3 | Buka unfinished Task | Reopen action tidak tersedia |
| TC-4 | Buka Grouping WBS | Reopen action tidak tersedia |
| TC-5 | Buka completed Task pada Closed Project | Reopen action disabled/absent; read-only explanation tersedia |
| TC-6 | Open confirmation | Task Name, Actual End, dan consequence copy tampil |
| TC-7 | Cancel confirmation | No request; state unchanged; focus kembali |
| TC-8 | Reopen completed Task valid | `200`; Actual End `null`; completed false |
| TC-9 | Verify immutable identity | Task ID dan hierarchy unchanged; no replacement Task |
| TC-10 | Verify all preserved fields | Name, Role, Assignee, Effort, dates, dependencies unchanged |
| TC-11 | Generic PATCH clear Actual End | `409 COMPLETED_TASK_READ_ONLY` |
| TC-12 | Direct command terhadap Group | `409 EXECUTABLE_TASK_REQUIRED` |
| TC-13 | Direct command terhadap unfinished Task | `409 TASK_NOT_COMPLETED`; no Forecast call |
| TC-14 | Reopen Open Project | Status Open; Forecast port once; no Execution/Commitment port |
| TC-15 | Reopen Locked Project | Status Locked; baselines unchanged; Forecast port once |
| TC-16 | Reopen Closed Project | `409 PROJECT_CLOSED_READ_ONLY`; no mutation |
| TC-17 | Automatic Scheduling OFF | Manual Execution/Commitment unchanged; Forecast contract tetap dipanggil |
| TC-18 | Incoming/outgoing dependencies | Semua relation dan endpoint ID unchanged |
| TC-19 | Completed blocked historical dependency | Setelah reopen, relation tetap; read-only completion restriction dievaluasi ulang |
| TC-20 | Forecast coordination failure | Actual End rollback; completed state retained; safe error |
| TC-21 | Repository persistence failure | No partial clear; no confirmed UI change |
| TC-22 | Double click confirm | Satu request |
| TC-23 | Concurrent Reopen commands | One success, one deterministic conflict; one transition |
| TC-24 | Old detail response returns after success | Actual End lama tidak dipulihkan |
| TC-25 | Old dependency candidate response returns after success | Completed marker lama tidak dipulihkan |
| TC-26 | Retry after recoverable failure | Initial state retained; next success clears Actual End |
| TC-27 | Keyboard-only flow | Open, cancel/confirm, focus return, feedback usable |
| TC-28 | Supported narrow viewport | Dialog dan form usable tanpa uncontrolled horizontal scroll |
| TC-29 | Unknown Task ID | `404 TASK_NOT_FOUND`; no Forecast call |
| TC-30 | Unknown request field/body | `400 INVALID_REQUEST`; no mutation |

---

## 11. Required Automated Tests by Layer

### 11.1 Domain Tests

- Executable completed Task can transition to unfinished through Reopen.
- Unfinished Task rejects Reopen.
- Grouping WBS rejects Reopen.
- Reopen changes only Actual End and allowed system metadata.
- Task identity and all planning fields remain unchanged.
- Completed-task generic mutation remains rejected.
- Reopen does not modify dependency identity or endpoints.

### 11.2 Application Tests

- Reopen Open Project Task.
- Reopen Locked Project Task while retaining Locked status and baselines.
- Reject Closed Project Task.
- Forecast port called exactly once on success.
- Execution/Commitment port not called.
- Forecast port not called on validation failure.
- Rollback when Forecast coordination fails.
- Task and Project not found mapping.
- Duplicate/concurrent command handling.
- Invalidation metadata/context contains affected Task, Project, dependency candidate, Forecast, Delivery Impact, Health, and workspace projections.

### 11.3 Repository Integration Tests

- Conditional clear only when `actual_end IS NOT NULL` or equivalent concurrency-safe strategy.
- Persisted Actual End becomes SQL `NULL`.
- All other columns remain unchanged.
- Task ID, parent, position, and dependency rows remain unchanged.
- Reopen Grouping WBS/unfinished Task is rejected through application invariant.
- Transaction rollback restores Actual End when coordination fails.
- Concurrent Reopen requests produce one successful state transition.
- SQLite automated contract plus PostgreSQL verification; preserve future MySQL-compatible behaviour.

### 11.4 API Integration Tests

- `POST /api/tasks/{taskId}/reopen` success.
- Empty body/approved body handling.
- Unknown field and malformed JSON rejection.
- Task not found.
- Group target.
- Task not completed.
- Closed Project.
- Generic PATCH cannot clear completed Actual End.
- Structured errors and safe messages.
- Forecast coordination failure mapping and rollback.
- Concurrent request behaviour.

### 11.5 Frontend Component/Integration Tests

- Contextual Edit Task displays Reopen for completed Task in Open Project.
- Reopen displayed for completed Task in Locked Project.
- Reopen absent/disabled for unfinished, Group, dan Closed Project.
- Confirmation content, cancel, focus return.
- Pending state and duplicate prevention.
- Success refresh without hard reload.
- Failure keeps Actual End and completed state.
- Retry succeeds.
- Dependency completed marker refreshes.
- Cache invalidation and stale-response protection.
- Accessible names, dialog semantics, keyboard flow, and responsive layout.
- Tests assert observable behaviour, not snapshot-only implementation details.

### 11.6 Acceptance-Level Tests

Acceptance-level tests must exercise the user workflow through the highest practical feature boundary already available in the repository, primarily Project Structure/contextual Edit Task. Mocking may replace external infrastructure, but the test must include the real presentation, application orchestration, validation mapping, and confirmed-state update for the behaviour being accepted.

Minimum acceptance-level scenarios:

1. Reopen completed Task on Open Project from contextual Edit Task.
2. Cancel Reopen without mutation.
3. Reopen completed Task on Locked Project while status/baselines remain unchanged.
4. Closed Project cannot Reopen Task.
5. Forecast coordination failure preserves completed state.
6. Dependency remains and completed marker changes after Reopen.
7. Concurrent/duplicate user interaction sends one command.
8. Stale response cannot restore completed state.
9. Keyboard-only Reopen flow.

A domain unit test or isolated button test alone is not acceptance-level evidence.

---

## 12. Three-Level Confidence Matrix

### 12.1 Mandatory Rule

Implementation of an Acceptance Criterion is **not complete** until all three evidence levels are present:

1. **Code Inspection** — concrete production-code path and invariant are inspected.
2. **Unit/Integration Test** — automated technical test proves domain/application/repository/API behaviour.
3. **Acceptance-Level Test** — automated feature-level test proves observable behaviour from the user-facing workflow or highest practical system boundary.

Rules:

- Passing tests without code inspection is insufficient.
- Code inspection without automated tests is insufficient.
- Unit/integration coverage without acceptance-level coverage is insufficient.
- A single test may support multiple ACs only when each AC assertion is explicit and traceable.
- Snapshot-only tests are not accepted as primary evidence.
- Manual verification may supplement but never replace automated acceptance-level evidence.
- Agent must stop completion reporting at the first AC lacking one evidence level.
- Final implementation report must list actual repository paths, test names, commands, and results. Generic statements such as `covered by tests` are invalid.

### 12.2 Required Evidence per AC

| AC | Code Inspection — required production evidence | Unit/Integration Test — required automated evidence | Acceptance-Level Test — required observable evidence |
| --- | --- | --- | --- |
| AC-1 | Contextual Edit Task eligibility derives completed state from Actual End and active Project status | Presentation/application eligibility tests for Open and Locked completed Task | Open completed Task from Project Structure and observe read-only form plus `Reopen Task`; repeat for Locked |
| AC-2 | UI eligibility plus backend guards for unfinished, Group, and Closed | Domain/API tests for all three invalid targets | Open each invalid target through real workflow and verify no enabled action; direct API bypass remains rejected where covered by acceptance harness |
| AC-3 | Dialog wiring, copy source, cancellation, focus restoration | Frontend integration test asserts no gateway call and focus return | User opens confirmation, reviews context, cancels, and sees unchanged Actual End/completion |
| AC-4 | Dedicated use case/command clears Actual End atomically | Domain + repository + API success tests | User confirms Reopen and Project Structure/contextual form shows unfinished Task without hard refresh |
| AC-5 | Mutation whitelist/aggregate copy preserves identity and fields | Repository integration compares before/after record and dependencies | Acceptance fixture with populated fields verifies visible data and hierarchy remain unchanged after Reopen |
| AC-6 | Generic update path retains completed read-only guard; only Reopen bypass is explicit | API/application tests reject generic clear and allow dedicated command | Completed Task normal Save/edit remains unavailable while Reopen succeeds through separate action |
| AC-7 | Open status branch invokes Forecast port only and does not call full schedule port | Application contract test verifies exact calls and stored dates | Reopen Open Task and verify status plus visible Execution/Commitment remain unchanged while Forecast refresh state is triggered |
| AC-8 | Locked branch preserves status and protected baselines | Application/repository test compares locked baselines and port calls | Reopen Locked Task through UI and verify Locked label/baselines remain unchanged |
| AC-9 | Closed guard precedes mutation/port coordination | Application/API test asserts conflict, no write, no port call | Closed Project workflow exposes no enabled Reopen; bypass rejection leaves UI/history unchanged |
| AC-10 | Reopen use case does not mutate dependency repository; invalidation includes dependency projections | Repository/application tests compare incoming/outgoing rows and current completion-based editability | Reopen Task with displayed dependencies; relations remain and completed marker/read-only behaviour refreshes |
| AC-11 | Transaction boundary encloses Actual End clear and required Forecast coordination | Application/repository rollback test with failing Forecast adapter | User confirms, receives safe error, and still sees prior Actual End/completed state; retry remains possible |
| AC-12 | Pending guard plus concurrency-safe persistence strategy | Frontend duplicate test and repository/application concurrent test | Double activation sends one request; acceptance harness confirms one completed-to-unfinished transition |
| AC-13 | Precondition checks Actual End before mutation and port | Domain/application/API tests assert `TASK_NOT_COMPLETED` and zero port calls | Unfinished Task has no action; direct command rejection does not alter visible state |
| AC-14 | Confirmed-state reducer/cache update distinguishes success and failure | Frontend integration tests for success, failure, and retry | End-to-end feature flow observes immediate unfinished state on success and retained completed state on failure |
| AC-15 | Versioned invalidation/request identity prevents stale overwrite | Cache/application/frontend race tests resolve old requests after mutation | Acceptance race scenario proves old detail/dependency/projection response cannot restore completed state |
| AC-16 | Semantic button/dialog markup, focus management, responsive container rules | Accessibility-focused frontend tests and supported viewport test | Keyboard-only Reopen and cancellation/confirmation workflow succeeds with perceivable pending/error/success feedback |

### 12.3 Implementation Evidence Table

Agents must fill this table with concrete evidence after implementation. Do not mark an AC complete using planned test names only.

| AC | Code Inspection path + invariant | Unit/Integration test path + test name + result | Acceptance-Level test path + test name + result | Status |
| --- | --- | --- | --- | --- |
| AC-1 | `frontend/src/features/wbs/presentation/WBSDetailDialog.tsx`: `completed` is derived from `ExecutableFields.actualEnd`; normal fields remain read-only and `canReopen` is true only for completed Tasks in `open` or `locked` Projects. `frontend/src/features/wbs/presentation/WBSPanel.tsx` opens the contextual Task workflow and replaces the visible Task only from the confirmed Reopen response. | Backend test code exists at `backend/internal/wbs/domain/wbs_test.go` (`TestReopenCompletedExecutablePreservesPlanningAndIdentity_AC1_AC2_AC4_AC5`) and `backend/internal/wbs/application/service_test.go` (`TestReopenUsesDedicatedStoreAndForecastExactlyOnce_AC1_AC7_AC8`, `TestReopenAllowsConfirmedOpenAndLockedStoreTransitions_AC7_AC8`). Diagnostic-only command with a temporary Go 1.23 compatibility directive: `GOTOOLCHAIN=local GOPROXY=off go test ./internal/wbs/domain ./internal/wbs/application ./internal/wbs/transport/http -count=1`; result PASS. Authoritative command `GOTOOLCHAIN=auto go test -count=1 -v ./internal/wbs/domain` did not execute tests because Go 1.24 toolchain download failed with DNS connection refusal. Frontend infrastructure typecheck command passed, but it is supplementary and not a presentation test. | `frontend/src/features/wbs/presentation/WBSPanel.reopen.test.tsx` — `AC-1 AC-4 AC-5 AC-6 AC-7 AC-14 reopens a completed Open Task from Project Structure without hard reload` and `AC-1 AC-8 reopens a completed Locked Task and keeps Locked editing semantics`; NOT RUN. Command `npm test -- src/features/wbs/infrastructure/httpWBSGateway.test.ts src/features/dependencies/infrastructure/httpDependenciesGateway.test.ts src/features/wbs/presentation/WBSPanel.reopen.test.tsx` exited 127 before discovery: `vitest: not found`. | PARTIAL — TEST GAP |
| AC-2 | Audit stopped at AC-1, the first AC without all three passing evidence levels. | Not adjudicated after AC-1. | Not adjudicated after AC-1. | NOT VERIFIED |
| AC-3 | Audit stopped at AC-1. | Not adjudicated after AC-1. | Not adjudicated after AC-1. | NOT VERIFIED |
| AC-4 | Audit stopped at AC-1. | Not adjudicated after AC-1. | Not adjudicated after AC-1. | NOT VERIFIED |
| AC-5 | Audit stopped at AC-1. | Not adjudicated after AC-1. | Not adjudicated after AC-1. | NOT VERIFIED |
| AC-6 | Audit stopped at AC-1. | Not adjudicated after AC-1. | Not adjudicated after AC-1. | NOT VERIFIED |
| AC-7 | Audit stopped at AC-1. | Not adjudicated after AC-1. | Not adjudicated after AC-1. | NOT VERIFIED |
| AC-8 | Audit stopped at AC-1. | Not adjudicated after AC-1. | Not adjudicated after AC-1. | NOT VERIFIED |
| AC-9 | Audit stopped at AC-1. | Not adjudicated after AC-1. | Not adjudicated after AC-1. | NOT VERIFIED |
| AC-10 | Audit stopped at AC-1. | Not adjudicated after AC-1. | Not adjudicated after AC-1. | NOT VERIFIED |
| AC-11 | Audit stopped at AC-1. | Not adjudicated after AC-1. | Not adjudicated after AC-1. | NOT VERIFIED |
| AC-12 | Audit stopped at AC-1. | Not adjudicated after AC-1. | Not adjudicated after AC-1. | NOT VERIFIED |
| AC-13 | Audit stopped at AC-1. | Not adjudicated after AC-1. | Not adjudicated after AC-1. | NOT VERIFIED |
| AC-14 | Audit stopped at AC-1. | Not adjudicated after AC-1. | Not adjudicated after AC-1. | NOT VERIFIED |
| AC-15 | Audit stopped at AC-1. | Not adjudicated after AC-1. | Not adjudicated after AC-1. | NOT VERIFIED |
| AC-16 | Audit stopped at AC-1. | Not adjudicated after AC-1. | Not adjudicated after AC-1. | NOT VERIFIED |

Allowed Status:

- `NOT VERIFIED`
- `PARTIAL — CODE ONLY`
- `PARTIAL — TEST GAP`
- `VERIFIED`

`VERIFIED` is allowed only when all three evidence columns contain concrete, passing evidence.

---

### 12.4 MVF-01 Implementation Readiness Addendum

`MVF-01 — Reopen Task confirmation dialog is clipped or incorrectly positioned`
is local manual-validation feedback against the existing Reopen Task contract.
It introduces no new business rule. The affected criteria are AC-3, AC-12,
AC-14, and AC-16 because the repair changes shared nested-dialog composition
while preserving cancellation, pending, success, and failure behaviour.

| AC | Code Inspection | Unit/Integration | Acceptance-Level | Overall |
| --- | --- | --- | --- | --- |
| AC-3 | `IMPLEMENTED BY CODE INSPECTION` — `frontend/src/shared/presentation/Dialog.tsx` uses `DialogStackContext` to portal each nested level to `document.body`, assign deterministic depth, expose only the deepest dialog as active, and restore focus through `useDialogFocus`. `frontend/src/features/wbs/presentation/WBSDetailDialog.tsx` keeps the Task detail mounted behind the `alertdialog`; Cancel only clears `reopenOpen`. | `AUTHORED — NOT RUN — LOCAL VALIDATION REQUIRED` — `frontend/src/shared/presentation/Dialog.test.tsx`: `MVF-01 assigns a higher semantic layer to a dialog nested inside another nested dialog`; `frontend/src/features/wbs/presentation/WBSPanel.reopen.test.tsx`: `AC-3 cancel and Escape send no request, preserve state, and return focus`.<br>`cd frontend && npm test -- src/shared/presentation/Dialog.test.tsx src/features/wbs/presentation/WBSPanel.reopen.test.tsx -t 'MVF-01 assigns a higher semantic layer|AC-3 cancel and Escape'` | `AUTHORED — NOT RUN — LOCAL VALIDATION REQUIRED` — `frontend/src/features/wbs/presentation/WBSPanel.reopen.test.tsx`: `MVF-01 AC-3 AC-16 portals the active confirmation above Task detail with contained narrow-viewport content` proves distinct dialogs coexist, portal ownership is outside Task detail, and the confirmation is active. Browser geometry requires the manual viewport checks in the local handoff.<br>`cd frontend && npm test -- src/features/wbs/presentation/WBSPanel.reopen.test.tsx -t 'MVF-01 AC-3 AC-16 portals the active confirmation'` | `IMPLEMENTED — LOCAL VALIDATION REQUIRED` |
| AC-12 | `IMPLEMENTED BY CODE INSPECTION` — `WBSDetailDialog.reopen` uses `reopenLock` plus `reopenBusy`; confirmation backdrop, Escape, Cancel, and parent close are guarded while pending. | `AUTHORED — NOT RUN — LOCAL VALIDATION REQUIRED` — `frontend/src/features/wbs/presentation/WBSPanel.reopen.test.tsx`: `AC-12 duplicate activation sends one command and pending cannot close`.<br>`cd frontend && npm test -- src/features/wbs/presentation/WBSPanel.reopen.test.tsx -t 'AC-12 duplicate activation sends one command'` | `AUTHORED — NOT RUN — LOCAL VALIDATION REQUIRED` — the WBSPanel workflow submits the real presentation command twice before resolution and observes one gateway request while the confirmation remains open.<br>`cd frontend && npm test -- src/features/wbs/presentation/WBSPanel.reopen.test.tsx -t 'AC-12 duplicate activation sends one command'` | `IMPLEMENTED — LOCAL VALIDATION REQUIRED` |
| AC-14 | `IMPLEMENTED BY CODE INSPECTION` — `WBSDetailDialog.reopen` closes only after a confirmed response and retains `reopenOpen`, completed state, and recoverable error on failure; `WBSPanel.onReopened` replaces the confirmed node without hard reload. | `AUTHORED — NOT RUN — LOCAL VALIDATION REQUIRED` — `frontend/src/features/wbs/presentation/WBSPanel.reopen.test.tsx`: `AC-11 AC-14 preserves completed state after Forecast failure and permits retry` and `AC-1 AC-4 AC-5 AC-6 AC-7 AC-14 reopens a completed Open Task from Project Structure without hard reload`.<br>`cd frontend && npm test -- src/features/wbs/presentation/WBSPanel.reopen.test.tsx -t 'preserves completed state after Forecast failure|reopens a completed Open Task'` | `AUTHORED — NOT RUN — LOCAL VALIDATION REQUIRED` — highest practical Project Structure workflow observes retained Actual End after failure and immediate unfinished confirmed state after success.<br>`cd frontend && npm test -- src/features/wbs/presentation/WBSPanel.reopen.test.tsx -t 'preserves completed state after Forecast failure|reopens a completed Open Task'` | `IMPLEMENTED — LOCAL VALIDATION REQUIRED` |
| AC-16 | `IMPLEMENTED BY CODE INSPECTION` — `DialogStackContext` derives nesting depth through React composition; `.dialog-overlay` remains viewport-fixed with safe centring; `.dialog-overlay-nested` uses the shared layer token plus semantic depth offset; inactive ancestors are `aria-hidden`; Reopen copy and errors use contained wrapping. | `AUTHORED — NOT RUN — LOCAL VALIDATION REQUIRED` — `frontend/src/shared/presentation/Dialog.test.tsx`: `portals nested dialogs outside the scrollable parent overlay`, `MVF-01 assigns a higher semantic layer to a dialog nested inside another nested dialog`, and `manages initial focus, traps tab navigation, and restores focus`.<br>`cd frontend && npm test -- src/shared/presentation/Dialog.test.tsx -t 'portals nested dialogs|MVF-01 assigns a higher semantic layer|manages initial focus'` | `AUTHORED — NOT RUN — LOCAL VALIDATION REQUIRED` — `frontend/src/features/wbs/presentation/WBSPanel.reopen.test.tsx`: keyboard-only flow and `MVF-01 AC-3 AC-16 portals the active confirmation above Task detail with contained narrow-viewport content`. Local browser validation must confirm full visibility, viewport centring, overlay coverage, no clipping, no uncontrolled horizontal scrolling, and long-content containment at normal and minimum supported widths.<br>`cd frontend && npm test -- src/features/wbs/presentation/WBSPanel.reopen.test.tsx -t 'completes the Reopen workflow using keyboard activation only|MVF-01 AC-3 AC-16 portals the active confirmation'` | `IMPLEMENTED — LOCAL VALIDATION REQUIRED` |

---

## 13. Technical Completion Criteria

1. Reopen is implemented as a dedicated Task/WBS application command.
2. No separate Task aggregate is introduced.
3. Domain remains independent of transport, persistence, scheduler implementation, and frontend.
4. Backend derives completion from Actual End.
5. Generic completed Task mutation remains read-only.
6. Only dedicated Reopen can clear Actual End.
7. Open and Locked completed Tasks are eligible; Closed Tasks are not.
8. Locked Project remains Locked; Locked→Open is never introduced.
9. Execution and Commitment values/baselines are unchanged by Reopen.
10. Forecast port is invoked exactly once only after valid state transition.
11. Forecast port failure rolls back Actual End clearing.
12. No Task replacement or new Task ID is created.
13. All non-Actual-End business data is preserved.
14. Dependencies remain unchanged and current completion-derived restrictions refresh.
15. API exposes a dedicated command and structured safe errors.
16. Conditional persistence/concurrency strategy prevents duplicate state transition.
17. Frontend prevents duplicate submission and preserves confirmed state on failure.
18. Cache invalidation includes Task, WBS, dependency, Forecast, Delivery Impact, Health, and workspace projections.
19. Stale responses cannot restore old completion/projection state.
20. Accessibility and responsive behaviour satisfy shared design-system rules.
21. Backend formatting, lint/static analysis, migrations if needed, unit/integration tests, and required race tests pass.
22. Frontend formatting, lint, type check, tests, and production build pass.
23. Each AC has concrete Three-Level Confidence evidence.
24. Agent stops at first unmet AC and does not claim story completion.
25. Completion report includes changed files, business-rule mapping, test commands/results, documentation impact, and deferred scope.

---

## 14. Documentation and Requirement Impact

Implementation must update or reconcile:

### US-4.1 Manage WBS

Replace the obsolete rule:

```text
Setting Actual End completes the WBS permanently for MVP. It cannot be edited or cleared, and correction/reopen is out of scope.
```

with:

```text
A completed Task remains read-only for normal mutation. US-4.2 provides the only explicit exception: Reopen Task may clear Actual End on an Open or Locked Project.
```

### US-3.1 Create Project

Clarify completed-leaf rule:

- Completed Task remains read-only for normal mutation.
- `Reopen Task` is an explicit command exception on Open and Locked Project.
- Reopen Task on Locked Project does not perform Locked→Open.
- Closed Project remains fully read-only.

### US-5.1 Manage Dependency

Clarify that:

- Reopen preserves dependency relations.
- Completed marker and candidate eligibility use current Actual End.
- Historical dependency read-only restriction for a completed blocked Task is reevaluated after Actual End is removed.

### US-3.3 Configure Project Settings

US-3.1 lifecycle rules are authoritative. Any wording that suggests a Locked Project can or must be changed back to Open is obsolete because `Locked → Open` is not an allowed Project transition. This story must not implement such a transition.

### Architecture/API Documentation

Document:

- Dedicated Reopen Task command.
- Completed read-only exception boundary.
- Forecast-only coordination.
- Atomic rollback strategy.
- Dependency preservation.
- Cache invalidation and stale-response strategy.
- Concurrency/conditional update approach.

AGENTS.md should be changed only if the repository does not yet contain the durable project-wide Three-Level Confidence rule. Do not duplicate the rule unnecessarily.

---

## 15. Deferred Implementation

This story wires and verifies the established Forecast recalculation contract but does not implement the Forecast scheduling algorithm.

The concrete Forecast Scheduler in Epic 6 is responsible for rescheduling the reopened Task and affected unfinished work using original Effort and all approved constraints.

A no-op adapter is acceptable only as temporary production composition before Epic 6, provided:

- the application port is invoked correctly;
- rollback/error contracts are testable through a failing adapter;
- implementation documentation clearly states that no-op success is not evidence of timeline recalculation.

---

## 16. Agent Execution Rules

Before implementation, the agent must inspect:

- `AGENTS.md`.
- `docs/architecture.md`.
- Project-specific architecture documentation.
- `docs/frontend-design-system.md`.
- US-3.1, US-4.1, and US-5.1.
- Existing WBS domain/application/repository/API/frontend implementation.
- Existing scheduler/Forecast ports and cache invalidation conventions.
- Existing acceptance-level test patterns, especially Project Structure/contextual Edit Task.

The agent must not:

- invent a second Task entity;
- implement Locked→Open;
- use generic PATCH to bypass completed read-only rules;
- implement speculative Forecast algorithm;
- delete/recreate Task or dependency;
- claim an AC complete without all three confidence levels;
- weaken an AC because existing architecture makes it inconvenient.

When implementation architecture conflicts with this story, the agent must identify the conflict, preserve locked product decisions, and update the appropriate project documentation rather than silently changing behaviour.

---

## 17. Unresolved Questions

None.
