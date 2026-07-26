# US-2.2 — Manage Public Holiday

## 1. User Story

**Sebagai** Engineering Lead,
**Saya ingin** mengelola hari libur,
**Sehingga** scheduler tidak menjadwalkan pekerjaan pada hari tersebut.

Public Holiday merupakan bagian dari **Epic 2: Schedule Management**.

---

## 2. Business Context

Public Holiday adalah constraint penjadwalan global. Berbeda dari Capacity
Override yang dimiliki satu Team Member, satu Public Holiday berlaku bagi semua
Team Member dan semua project pada tanggal tersebut.

Scheduler memerlukan data Public Holiday terbaru yang sudah dikonfirmasi untuk
menentukan kapasitas efektif. Pada tanggal libur, Execution Capacity dan
Commitment Capacity setiap Team Member harus bernilai `0`, termasuk ketika
terdapat Capacity Override. Pengelolaan Public Holiday tidak mengubah data
kapasitas dasar, buffer, atau Capacity Override yang tersimpan.

---

## 3. Scope

Engineering Lead dapat:

1. Membuka halaman **Public Holidays** dari group navigasi **Team Configuration**.
2. Melihat daftar Public Holiday yang paginated.
3. Memfilter daftar menggunakan satu **Holiday Date**.
4. Membuka detail Public Holiday.
5. Menambahkan Public Holiday.
6. Mengubah inclusive Date Range dan Description sebagai satu kesatuan.
7. Menghapus Public Holiday secara permanen setelah konfirmasi.
8. Mencoba kembali setelah list gagal dimuat.
9. Memperbaiki atau mencoba kembali form yang gagal tanpa kehilangan draft.

---

## 4. Out of Scope

User story ini tidak mencakup:

* Country, Region, Holiday Type, Recurrence, Half Day, Start Time, End Time,
  Capacity, Team Member, Project, Role, atau Approval Status.
* Public Holiday per Team Member, project, negara, atau region.
* Date-range filter dan text search. Tidak adanya text search merupakan exception
  eksplisit terhadap default search pada `AGENTS.md`; kebutuhan operasional MVP
  adalah lookup exact date.
* Import atau sinkronisasi hari libur dari layanan eksternal.
* Menjalankan Scheduling Engine secara otomatis setelah mutation.
* Menyimpan derived Execution Capacity atau Commitment Capacity.
* Restore Public Holiday yang sudah dihapus.
* Permission model baru. Aplikasi saat ini belum memiliki permission model.

---

## 5. UI Placement and User Flow

* **Public Holidays** tersedia sebagai page tersendiri di bawah group navigasi
  **Team Configuration**, bersama Roles dan Members.
* Page menggunakan application shell yang persisten; global top bar, sidebar,
  dan chrome tidak diduplikasi atau diremount ketika route berubah.
* Page title adalah **Public Holidays**.
* Primary action adalah **Add Public Holiday**.
* Page mengikuti pola list Roles dan Members untuk loading, list, pagination,
  form, feedback, dan destructive confirmation jika pola tersebut relevan.
* Add dan Edit menyediakan field berlabel **Date Range** dan **Description**. Label
  tidak boleh hanya berupa placeholder.
* Date Range menggunakan shared two-click range calendar pattern dengan
  month navigation, click-outside dismissal, keyboard operation, dan
  viewport-aware placement.
* Delete confirmation menampilkan Holiday Date dan Description serta action
  Cancel dan Delete.
* Cancel pada form atau dialog tidak mengirim mutation.
* Modal/dialog mengelola focus dan mengembalikannya ke trigger saat ditutup.
* Seluruh required action tetap tersedia pada supported screen sizes tanpa
  horizontal scrolling pada normal page content.

---

## 6. Domain Model

### Public Holiday

| Field       | Type                        | Required | Source | Rules |
| ----------- | --------------------------- | -------: | ------ | ----- |
| ID          | System-generated identifier | Ya       | System | Dibuat otomatis dan immutable |
| Start Date  | Date-only                   | Ya       | User   | ISO `YYYY-MM-DD`; awal range inklusif |
| End Date    | Date-only                   | Ya       | User   | ISO `YYYY-MM-DD`; akhir range inklusif dan tidak sebelum Start Date |
| Description | String                      | Ya       | User   | Trimmed, tidak blank, maksimum 100 karakter |
| Created At  | DateTime                    | Ya       | System | Dibuat otomatis dan immutable saat update |
| Updated At  | DateTime                    | Ya       | System | Dibuat dan diperbarui otomatis |

Public Holiday menyimpan kumpulan derived weekday dates sebagai bagian aggregate
untuk lookup scheduler yang efisien. Public Holiday tidak menyimpan Team Member ID, Project ID, Role ID, Capacity,
Execution Capacity, atau Commitment Capacity.

Batas Description `100` karakter mengikuti konvensi master-data repository yang
sudah digunakan oleh Roles dan Members. Panjang divalidasi setelah trim.

---

## 7. Business Rules

### Date Range Rules

* Start Date dan End Date wajib diisi, inklusif, dan merupakan date-only values.
* End Date harus sama dengan atau setelah Start Date.
* Pemilihan kedua yang lebih awal mengganti Start Date; tanggal yang sama menghasilkan range satu hari.
* Sabtu dan Minggu adalah holiday default dan dilewati saat membuat derived dates.
* Range yang tidak memiliki satu pun weekday ditolak dengan `NO_WORKING_DATES`.
* API menggunakan ISO `YYYY-MM-DD` yang merepresentasikan tanggal kalender valid.
* Date harus tetap sama pada frontend, backend, database, dan lintas timezone;
  serialisasi tidak boleh menggeser ke hari sebelumnya atau berikutnya.
* Tanggal lampau, hari ini, dan tanggal masa depan boleh dibuat, ditampilkan,
  diubah, dan dihapus.
* Tidak ada pembatasan future date.

### Description Rules

* Description wajib diisi dan harus human-readable.
* Leading dan trailing whitespace di-trim sebelum persistence.
* Nilai kosong atau hanya whitespace ditolak.
* Panjang maksimum setelah trim adalah `100` karakter.
* Internal casing, punctuation, dan wording pengguna dipertahankan.

### Identity and Timestamp Rules

* ID dibuat sistem dan tidak dapat diubah.
* Update hanya dapat mengubah Date dan Description.
* Created At tidak berubah setelah entity dibuat.
* Updated At berubah setelah update berhasil.
* Timestamp dikelola sistem dan tidak diterima dari request create/update.

### Mutation and Delete Rules

* Create, update, dan delete harus atomic; failure tidak boleh meninggalkan
  partial confirmed state.
* Delete Public Holiday menggunakan hard delete untuk MVP.
* Delete memerlukan confirmation yang menyebut Date dan Description.
* Cancel tidak mengirim delete request.
* Setelah delete berhasil, Public Holiday tidak dapat diambil kembali.
* Duplicate submission dicegah selama mutation yang sama masih berjalan.
* Failure mempertahankan form draft dan memberikan recovery action yang jelas.
* Mutation berhasil memperbarui list yang terlihat tanpa hard refresh.
* Semua cache list Public Holiday yang affected, filtered maupun unfiltered,
  diinvalidasi atau direfresh.
* Versioned invalidation atau established equivalent mencegah response lama yang
  masih in-flight mengembalikan stale data setelah mutation.

### Duplicate-Date Rule

* Satu weekday Date hanya boleh dimiliki satu Public Holiday aggregate.
* Jika satu saja weekday dalam range sudah digunakan, seluruh create/update ditolak secara atomic dengan `409 Conflict` dan code
  `PUBLIC_HOLIDAY_DATE_ALREADY_EXISTS`.
* Update ke Date milik Public Holiday lain ditolak dengan conflict yang sama.
* Update mengganti seluruh range dan derived dates dalam satu transaction.
* Concurrent writes harus tetap menjaga uniqueness setiap weekday Date.

---

## 8. Capacity Resolution Impact

Untuk satu Team Member pada satu tanggal, urutan resolution adalah:

1. Jika Date adalah Public Holiday, resolved capacity adalah `0`.
2. Jika bukan Public Holiday dan Capacity Override berlaku, gunakan override.
3. Jika tidak, gunakan Team Member Daily Capacity.
4. Team Member Buffer diterapkan terhadap resolved capacity untuk menghasilkan
   Commitment Capacity.

Konsekuensi:

* Public Holiday memiliki precedence lebih tinggi daripada Capacity Override.
* Capacity Override tidak dapat membuat Team Member available pada Public Holiday.
* Execution Capacity dan Commitment Capacity adalah `0` bagi seluruh Team Member
  dan seluruh project pada Public Holiday.
* Public Holiday hanya memengaruhi capacity resolution pada Date terkait.
* Public Holiday tidak mengubah atau menghapus Daily Capacity, Buffer, maupun
  Capacity Override.
* Create, update, dan delete tidak otomatis menjalankan Scheduling Engine.
* Data Public Holiday terbaru yang sudah dikonfirmasi tersedia bagi Scheduling
  Engine pada perhitungan berikutnya.

---

## 9. List Behaviour

* List menggunakan backend pagination dengan page default `1`, page size default
  `5`, dan maksimum page size `100`.
* Response menyediakan `page`, `pageSize`, dan `total`.
* UI menampilkan current page, previous/next availability, visible result range,
  dan total.
* Active/upcoming (`End Date >= today`) ditampilkan sebelum expired. Di dalam
  masing-masing group urutan deterministik adalah Start Date ASC, End Date ASC,
  lalu ID ASC. `today` mengikuti `APP_TIMEZONE` dan tidak ada batas masa depan.
* Optional filter **Holiday Date** menggunakan exact-date match dan dijalankan
  backend sebelum count, limit, serta offset.
* Tanpa Holiday Date, normal paginated list ditampilkan.
* Perubahan atau Clear Holiday Date mereset page ke `1`.
* Holiday Date dipertahankan selama pagination dan setelah mutation ketika masih
  applicable.
* Holiday Date menjadi bagian dari request-cache identity.
* Explicit Clear action tersedia ketika filter aktif.
* Text search dan date-range filter tidak tersedia pada story ini.
* Current page dipertahankan setelah mutation bila masih valid.
* Jika delete item terakhir membuat current page invalid, UI berpindah ke last
  valid page; jika dataset kosong, page menjadi `1`.
* Initial load menampilkan shape-preserving local skeleton atau loader.
* Safe background refresh mempertahankan data yang masih berguna.
* Normal empty state menjelaskan bahwa scheduler belum memiliki configured global
  holidays dan menyediakan Add Public Holiday.
* Filtered no-results state menyebut Holiday Date secara human-readable dan
  menyediakan Clear. State ini tidak ditampilkan sebelum response selesai.
* Load-error state berbeda dari empty/no-results, tidak mengekspos detail teknis,
  dan menyediakan Retry.

---

## 10. Acceptance Criteria

### AC-1 — Navigasi Public Holidays

**Given** application shell tersedia
**When** Engineering Lead membuka group Team Configuration
**Then** tersedia menu Public Holidays yang membuka page Public Holidays
**And** global chrome tetap stabil dan tidak diduplikasi.

### AC-2 — Initial paginated list dan ordering

**Given** Public Holiday tersedia
**When** page pertama dimuat tanpa pagination parameter
**Then** API mengembalikan maksimal lima item dan metadata pagination
**And** active/upcoming tampil sebelum expired; tiap group diurutkan Start Date,
End Date, lalu ID ascending
**And** UI menampilkan current page, navigation, visible range, dan total.

### AC-3 — Initial loading

**Given** initial list request belum selesai
**Then** page menampilkan local shape-preserving skeleton
**And** tidak menampilkan empty atau no-results secara prematur.

### AC-4 — Empty state

**Given** belum ada Public Holiday
**When** list berhasil dimuat
**Then** empty state menjelaskan bahwa belum ada global holiday terkonfigurasi
**And** menyediakan Add Public Holiday.

### AC-5 — List failure dan Retry

**Given** list gagal dimuat
**Then** error state yang aman dan actionable ditampilkan dengan Retry
**When** Retry berikutnya berhasil
**Then** list ditampilkan.

### AC-6 — Membuka Add form

**When** Add Public Holiday dipilih
**Then** form menampilkan field berlabel Date dan Description
**And** Date menggunakan shared date picker.

### AC-7 — Create valid

**When** Date valid dan Description valid disimpan
**Then** Public Holiday dibuat dengan ID dan timestamps dari sistem
**And** response `201 Created` diterima
**And** visible list diperbarui tanpa hard refresh.

### AC-8 — Date required dan valid

**When** Date kosong atau bukan tanggal kalender ISO `YYYY-MM-DD` yang valid
**Then** create/update ditolak
**And** error ditampilkan di dekat Date
**And** tidak ada mutation.

### AC-9 — Description required

**When** Description kosong atau hanya whitespace
**Then** create/update ditolak
**And** error ditampilkan di dekat Description.

### AC-10 — Description trim dan preservation

**When** Description memiliki leading/trailing whitespace
**Then** whitespace tersebut di-trim sebelum persistence
**And** internal casing, punctuation, serta wording dipertahankan.

### AC-11 — Maximum Description

**Given** batas Description adalah 100 karakter setelah trim
**Then** 100 karakter diterima dan lebih dari 100 karakter ditolak tanpa mutation.

### AC-12 — Date-only dan rentang waktu

**When** holiday lampau, hari ini, atau masa depan disimpan
**Then** semuanya diterima
**And** Date yang dibaca kembali tidak berubah pada timezone berbeda.

### AC-13 — Edit success dan identity

**Given** Public Holiday tersedia
**When** Date atau Description diubah dengan nilai valid
**Then** update berhasil dengan `200 OK`
**And** ID serta Created At tetap
**And** Updated At berubah
**And** list diperbarui tanpa hard refresh.

### AC-14 — Delete confirmation

**When** Delete dipilih
**Then** belum ada delete request
**And** dialog menampilkan Date, Description, Cancel, dan Delete.

### AC-15 — Delete cancellation

**When** Cancel dipilih pada confirmation
**Then** dialog ditutup, focus dikembalikan, dan tidak ada delete request.

### AC-16 — Delete success

**When** Delete dikonfirmasi dan berhasil
**Then** API mengembalikan `204 No Content`
**And** record di-hard-delete
**And** list diperbarui tanpa hard refresh.

### AC-17 — Not found

**When** get, update, atau delete menggunakan ID yang tidak tersedia
**Then** API mengembalikan `404 PUBLIC_HOLIDAY_NOT_FOUND`
**And** data lain tidak berubah.

### AC-18 — Backend validation

**Given** frontend validation dilewati
**When** invalid request dikirim langsung
**Then** backend tetap menolak request dengan structured error
**And** data invalid tidak disimpan.

### AC-19 — Duplicate submission prevention

**When** Save atau Delete ditekan kembali selama request pertama berjalan
**Then** hanya satu mutation dikirim
**And** hanya controls terkait yang disabled.

### AC-20 — Submission failure dan rollback

**Given** form memiliki draft
**When** validation, persistence, atau dependency failure terjadi
**Then** draft tetap tersedia, error actionable ditampilkan, dan retry tetap
memungkinkan
**And** tidak ada partial mutation yang tersimpan.

### AC-21 — Pagination validation dan navigation

**Given** terdapat lebih dari lima holiday
**Then** user dapat berpindah page melalui backend pagination
**And** page harus lebih dari `0` serta pageSize harus `1` sampai `100`
**And** invalid value ditolak sebelum list query dijalankan.

### AC-22 — Page preservation dan correction

**When** mutation berhasil dan current page masih valid
**Then** current page dipertahankan.
**When** delete item terakhir membuat page tidak valid
**Then** UI berpindah ke last valid page.

### AC-23 — Exact Holiday Date filter

**Given** Holiday Date dipilih
**When** list dimuat
**Then** backend hanya mengembalikan records dengan Date yang sama persis
**And** filtering diterapkan sebelum count dan pagination.

### AC-24 — Filter reset dan Clear

**When** Holiday Date berubah atau dihapus
**Then** page kembali ke `1`.
**When** Clear dipilih
**Then** normal unfiltered paginated list dimuat.

### AC-25 — Filtered no-results

**Given** filtered response sudah selesai tanpa match
**Then** UI menyebut Holiday Date secara human-readable dan menyediakan Clear
**And** state berbeda dari empty dan load-error.

### AC-26 — Filter preservation

**Given** Holiday Date aktif
**When** user berpindah page atau mutation berhasil
**Then** Holiday Date tetap terpilih dan digunakan pada refresh.

### AC-27 — Filtered cache consistency

**When** create, update, atau delete berhasil
**Then** affected filtered dan unfiltered caches diinvalidasi atau direfresh
**And** older in-flight response tidak dapat merepopulasi stale data.

### AC-28 — Invalid Holiday Date filter

**When** `holidayDate` bukan date-only `YYYY-MM-DD` yang valid
**Then** API mengembalikan `400 INVALID_HOLIDAY_DATE` pada field `holidayDate`
**And** repository list tidak dijalankan.

### AC-29 — Public Holiday precedence

**Given** Date adalah Public Holiday dan Capacity Override juga berlaku
**When** capacity di-resolve
**Then** Execution Capacity dan Commitment Capacity semua Team Member adalah `0`
**And** Capacity Override tidak menggantikan Public Holiday.

### AC-30 — Non-holiday resolution

**Given** Date bukan Public Holiday
**When** capacity di-resolve
**Then** Capacity Override digunakan bila berlaku; selain itu Daily Capacity
digunakan; Buffer kemudian menghasilkan Commitment Capacity.

### AC-31 — Scheduler consistency

**When** Public Holiday berhasil dibuat, diubah, atau dihapus
**Then** mutation tidak otomatis menjalankan Scheduling Engine
**And** latest confirmed data tersedia bagi perhitungan berikutnya.

### AC-32 — Accessibility

**When** workflow digunakan dengan keyboard atau assistive technology
**Then** navigation, filter, pagination, form, feedback, dan dialog dapat
dipahami serta dioperasikan
**And** focus dialog dikelola dan dikembalikan dengan benar
**And** status tidak dikomunikasikan hanya dengan warna.

### AC-33 — Responsive behaviour

**When** workflow digunakan pada supported screen sizes
**Then** list, filter, form, pagination, dan required actions tetap usable
**And** normal page content tidak mengalami horizontal scrolling.

### AC-34 — Date harus unik

**Given** Public Holiday sudah tersedia pada suatu Date
**When** create lain atau update record berbeda menggunakan Date tersebut
**Then** mutation ditolak dengan `409 PUBLIC_HOLIDAY_DATE_ALREADY_EXISTS`
**And** confirmed data sebelumnya tetap utuh.

**When** Public Holiday disimpan ulang tanpa mengubah Date miliknya sendiri
**Then** update tidak conflict dengan dirinya sendiri.

**When** dua create untuk Date yang sama diproses concurrent
**Then** maksimal satu record tersimpan.

Permission behaviour tidak menjadi Acceptance Criterion karena aplikasi belum
memiliki permission model dan story ini tidak memperkenalkannya.

---

## 11. API Contract

### List Public Holidays

```http
GET /api/public-holidays?holidayDate=2026-08-17&page=1&pageSize=5
```

```json
{
  "data": [
    {
      "id": "public-holiday-id",
      "startDate": "2026-08-17",
      "endDate": "2026-08-18",
      "description": "Independence Day",
      "createdAt": "2026-07-26T10:00:00Z",
      "updatedAt": "2026-07-26T10:00:00Z"
    }
  ],
  "page": 1,
  "pageSize": 5,
  "total": 1
}
```

Status: `200 OK`.

`page` default `1`; `pageSize` default `5` dan valid pada range `1..100`.
`holidayDate` optional dan harus berupa date-only ISO yang valid. Filtering
dijalankan sebelum count dan pagination.

### Get Public Holiday

```http
GET /api/public-holidays/{publicHolidayId}
```

Status: `200 OK`; response menggunakan satu object dengan field yang sama seperti
item list.

### Create Public Holiday

```http
POST /api/public-holidays
Content-Type: application/json
```

```json
{
  "startDate": "2026-08-17",
  "endDate": "2026-08-18",
  "description": "Independence Day"
}
```

Status: `201 Created`; response memuat entity yang dibuat.

### Update Public Holiday

```http
PUT /api/public-holidays/{publicHolidayId}
Content-Type: application/json
```

```json
{
  "startDate": "2026-08-17",
  "endDate": "2026-08-18",
  "description": "Independence Day"
}
```

Status: `200 OK`; request tidak menerima ID atau timestamps.

### Delete Public Holiday

```http
DELETE /api/public-holidays/{publicHolidayId}
```

Status: `204 No Content`.

### Failure Status

| Condition | Status |
| --------- | -----: |
| Invalid request, date, description, atau pagination | 400 Bad Request |
| Public Holiday tidak ditemukan | 404 Not Found |
| Date sudah digunakan Public Holiday lain | 409 Conflict |
| Persistence atau dependency failure | 500 Internal Server Error |

---

## 12. Error Response Contract

Semua error menggunakan format repository:

```json
{
  "code": "PUBLIC_HOLIDAY_DESCRIPTION_REQUIRED",
  "message": "Description is required",
  "field": "description"
}
```

| Error Code | Field | Condition |
| ---------- | ----- | --------- |
| `PUBLIC_HOLIDAY_START_DATE_REQUIRED` | `startDate` | Start Date tidak diisi |
| `PUBLIC_HOLIDAY_END_DATE_REQUIRED` | `endDate` | End Date tidak diisi |
| `INVALID_HOLIDAY_DATE` | `startDate`, `endDate`, atau `holidayDate` | Nilai bukan date-only ISO valid |
| `PUBLIC_HOLIDAY_INVALID_DATE_RANGE` | `endDate` | End Date sebelum Start Date |
| `PUBLIC_HOLIDAY_NO_WORKING_DATES` | `startDate` | Range hanya berisi Sabtu/Minggu |
| `PUBLIC_HOLIDAY_DESCRIPTION_REQUIRED` | `description` | Description kosong setelah trim |
| `PUBLIC_HOLIDAY_DESCRIPTION_TOO_LONG` | `description` | Description lebih dari 100 karakter setelah trim |
| `PUBLIC_HOLIDAY_NOT_FOUND` | — | ID tidak tersedia |
| `PUBLIC_HOLIDAY_DATE_ALREADY_EXISTS` | `startDate` | Sedikitnya satu weekday sudah digunakan aggregate lain; seluruh mutation ditolak |
| `INVALID_PAGE` | `page` | Page tidak lebih besar dari 0 |
| `INVALID_PAGE_SIZE` | `pageSize` | Page size di luar `1..100` |

Internal error tidak mengekspos stack trace, query, database, atau infrastructure.
Frontend memetakan error ke pesan yang dapat dipahami.

---

## 13. Test Cases

| ID | Scenario | Expected |
| -- | -------- | -------- |
| TC-1 | Buka Team Configuration > Public Holidays | Page terbuka dalam shared shell; menu aktif |
| TC-2 | Tunda initial list response | Local skeleton tampil; empty/no-results belum tampil |
| TC-3 | Muat active/upcoming dan expired ranges | Page size 5; group dan range ordering benar; metadata benar |
| TC-4 | Dataset kosong | Global-holiday empty state dan Add action tampil |
| TC-5 | List gagal lalu Retry berhasil | Safe error tampil, kemudian list pulih |
| TC-6 | Buka Add | Date Range dan Description berlabel serta dapat dioperasikan |
| TC-7 | Create range berisi weekday | `201`; weekend dilewati; aggregate dan visible list terbarui |
| TC-8 | Start/End Date missing | structured required error; no mutation |
| TC-9 | Date invalid, reversed, atau weekend-only | structured validation error; no mutation |
| TC-10 | Description missing/whitespace-only | Required error; no mutation |
| TC-11 | Description dengan outer whitespace/casing/punctuation | Outer whitespace di-trim; isi internal dipertahankan |
| TC-12 | Description 100 dan 101 karakter | 100 diterima; 101 ditolak `PUBLIC_HOLIDAY_DESCRIPTION_TOO_LONG` |
| TC-13 | Simpan/baca Date pada timezone berbeda | Date tetap identik |
| TC-14 | Create holiday lampau, hari ini, masa depan | Semua diterima |
| TC-15 | Edit Date Range | `200`; seluruh derived dates diganti secara atomic dan list terbarui |
| TC-16 | Edit Description | ID/Created At tetap; Updated At berubah |
| TC-17 | Update ID tidak ada | `404 PUBLIC_HOLIDAY_NOT_FOUND`; no mutation |
| TC-18 | Buka Delete | Dialog memuat Date/Description; belum ada request |
| TC-19 | Cancel Delete | Dialog tutup, focus kembali, no request |
| TC-20 | Confirm Delete | `204`; hard delete; list terbarui |
| TC-21 | Delete ID tidak ada | `404 PUBLIC_HOLIDAY_NOT_FOUND`; data lain utuh |
| TC-22 | Klik Save/Delete berulang | Hanya satu mutation dikirim |
| TC-23 | Backend form failure | Draft dan recovery action dipertahankan |
| TC-24 | Dependency failure saat mutation | Transaction rollback; tidak ada partial state |
| TC-25 | Lebih dari lima holidays dan invalid params | Pagination/navigation benar; invalid params ditolak sebelum query |
| TC-26 | Mutation saat current page masih valid | Page dipertahankan |
| TC-27 | Delete only item pada last page | Berpindah ke last valid page |
| TC-28 | Filter exact Holiday Date | Hanya exact Date; total/offset filtered benar |
| TC-29 | Ubah filter dari page selain 1 | Filtered request menggunakan page 1 |
| TC-30 | Filter tanpa match | Human-readable no-results dan Clear tampil setelah response |
| TC-31 | Clear filter | Unfiltered page 1 kembali |
| TC-32 | Paginate dengan filter aktif | Holiday Date tetap dikirim |
| TC-33 | Create/update/delete saat filter aktif | Filter tetap; filtered/unfiltered cache konsisten |
| TC-34 | Response lama selesai setelah mutation | Stale response tidak merepopulasi cache |
| TC-35 | Invalid `holidayDate` query | `400 INVALID_HOLIDAY_DATE`; repository tidak dipanggil |
| TC-36 | Holiday dan override pada Date sama | Execution/Commitment Capacity semua Member adalah 0 |
| TC-37 | Non-holiday dengan/tanpa override | Override lalu Daily Capacity digunakan sesuai precedence |
| TC-38 | Mutation holiday lalu perhitungan berikutnya | Latest confirmed data terbaca; scheduler tidak auto-run |
| TC-39 | Operasikan dengan keyboard/assistive tech | Controls, feedback, dan dialog focus accessible |
| TC-40 | Uji supported viewport | Workflow usable tanpa normal horizontal scroll |
| TC-41 | Create/update menggunakan Date yang sudah dipakai | `409 PUBLIC_HOLIDAY_DATE_ALREADY_EXISTS`; existing data utuh |
| TC-42 | Update tanpa mengubah Date sendiri | Update berhasil tanpa self-conflict |
| TC-43 | Dua concurrent create pada Date sama | Maksimal satu tersimpan; lainnya conflict |

### Acceptance Criteria Traceability

| Acceptance Criteria | Test Case(s) |
| ------------------- | ------------ |
| AC-1 | TC-1 |
| AC-2 | TC-3, TC-25 |
| AC-3—AC-6 | TC-2, TC-4—TC-6 |
| AC-7—AC-12 | TC-7—TC-14 |
| AC-13 | TC-15, TC-16 |
| AC-14—AC-17 | TC-18—TC-21 |
| AC-18 | TC-8—TC-12, TC-35 |
| AC-19—AC-20 | TC-22—TC-24 |
| AC-21—AC-22 | TC-25—TC-27 |
| AC-23—AC-28 | TC-28—TC-35 |
| AC-29—AC-31 | TC-36—TC-38 |
| AC-32—AC-33 | TC-39—TC-40 |
| AC-34 | TC-41—TC-43 |

---

## 14. Required Automated Tests

### Domain Tests

* Public Holiday creation and update.
* Required and valid Start/End Date, inclusive range, reversed range, same-day,
  weekday generation, weekend skipping, dan weekend-only rejection.
* Required, trimmed, non-blank Description.
* Description boundaries 100/101 after trim.
* Update preserves ID and Created At and changes Updated At.
* Date-only semantics across timezone boundaries.
* Capacity precedence contract when resolution component is introduced or
  extended by this story.

### Application Tests

* Paginated list and optional Holiday Date forwarding.
* Get, create, update, and hard delete.
* Not-found behavior.
* Transaction boundary and rollback.
* Latest confirmed holiday data available to capacity-resolution consumers.
* Mutation does not invoke Scheduling Engine.

### Repository Integration Tests

* Persist and retrieve entity.
* Date-only aggregate dan derived-date persistence.
* Active/upcoming-before-expired grouping dan Start/End/ID ordering.
* Exact Holiday Date filtering before count, limit, and offset.
* Pagination metadata.
* Timestamp behavior.
* Hard delete.
* Transaction rollback bila salah satu derived date conflict.
* Query plan and index support for exact Date lookup plus grouped range ordering.
* PostgreSQL and future MySQL compatibility.
* Unique child Date constraint, whole-range update, dan concurrent writes.

Automated repository tests use SQLite isolation where appropriate and must not
delete or mutate local PostgreSQL data. Dialect-specific behavior still requires
the repository's supported database verification strategy.

### API Integration Tests

* List, pagination, valid/invalid Holiday Date filter.
* Get, create, update, and delete mappings and success statuses.
* Start/End Date-only serialization dan bounded calendar-date endpoint.
* Required/length validation and all documented structured errors.
* Not-found behavior.
* Database persistence and rollback.
* Duplicate Date conflict response.

### Frontend Tests

* Team Configuration navigation and stable application shell.
* Initial loading, populated, empty, no-results, load-error, Retry, and safe
  background refresh states.
* Paginated list, page preservation, and page correction.
* Add/Edit form validation, success, and failure with preserved draft.
* Delete confirmation, cancellation, and success.
* Duplicate-submission prevention and mutation feedback.
* Holiday Date exact filter, reset, Clear, preservation, and cache identity.
* Mutation cache invalidation and stale-response protection.
* Shared range calendar interaction, weekend/public-holiday markings, dan click-outside behavior.
* Keyboard accessibility, dialog focus management, and responsive behavior.

Tests berfokus pada observable behavior dan tidak hanya mengandalkan snapshots.

---

## 15. Technical Completion Criteria

1. Dedicated backend Public Holiday feature boundary mengikuti struktur domain,
   application, infrastructure, dan transport yang ada.
2. Domain tidak bergantung pada HTTP, database, GORM, framework, atau
   infrastructure.
3. Frontend mengikuti feature-oriented domain/application/infrastructure/
   presentation boundaries tanpa speculative abstraction atau `any`.
4. Versioned database migration tersedia.
5. Start Date, End Date, dan derived weekday dates menggunakan date-only-compatible database type.
6. Description menggunakan maximum length `100` yang konsisten.
7. Unique child-date index menjaga invariant dan exact Date filtering; aggregate
   indexes mendukung status grouping serta Start/End/ID ordering.
8. Query/index strategy direview untuk PostgreSQL dan future MySQL, termasuk
   `EXPLAIN` untuk non-trivial supported query.
9. Runtime access menggunakan GORM tanpa raw DML; raw SQL hanya untuk permitted
   migration DDL/test setup.
10. Backend pagination menggunakan shared listing contract, default `5`, max
    `100`, serta metadata `page`, `pageSize`, dan `total`.
11. API DTO, domain entity, dan persistence model dipisahkan ketika tanggung
    jawab berbeda.
12. Business validation tidak hanya berada di React, handler, atau repository.
13. Frontend presentation tidak memanggil HTTP langsung ketika use case tersedia.
14. Successful mutations menginvalidasi affected filtered/unfiltered caches.
15. Established versioned invalidation mencegah stale in-flight response.
16. Capacity-resolution consumer dapat membaca latest confirmed holiday data.
17. Public Holiday memiliki precedence lebih tinggi daripada Capacity Override.
18. Story tidak memperkenalkan automatic schedule recalculation.
19. Loading, empty, filtered no-results, load-error, validation-error, mutation,
    dan success states berbeda dan konsisten.
20. Form mempertahankan draft setelah failure dan mencegah duplicate submission.
21. Delete menggunakan accessible confirmation dan hard delete.
22. Accessibility dan responsive behavior diimplementasikan serta diuji.
23. Mandatory domain, application, repository, API, dan frontend tests lulus.
24. Backend format, vet, static analysis, migration verification, tests, dan race
    tests lulus.
25. Frontend format, lint, type checking, tests, dan build lulus.
26. Local backend direstart setelah backend/config/migration changes dan process
    pengganti dikonfirmasi listening.
27. Minimal satu affected endpoint diverifikasi manual setelah restart dan
    frontend flow di-smoke-test.
28. API documentation diperbarui.
29. Project architecture documentation menjelaskan feature, global ownership,
    date-only persistence, paginated exact-date filter, cache identity/
    invalidation, index strategy, dan future capacity-resolution integration.
30. `AGENTS.md` hanya diubah bila implementasi memperkenalkan durable project-wide
    rule, bukan untuk behavior feature ini.
31. Completion report menyertakan documentation-impact assessment.
32. Database dan application mapping menjaga unique Date, termasuk pada
    concurrent writes, dengan conflict yang konsisten.

---

## 16. Documentation Impact

Future implementation wajib menilai dan memperbarui:

* `docs/project/architecture.md` untuk architecture project-specific Public
  Holiday. Root `docs/architecture.md` tetap dinilai, tetapi sesuai ownership
  dokumentasi repository, reusable baseline hanya diubah bila standard lintas
  project berubah.
* API documentation untuk endpoint, query, response, dan structured errors.
* `README.md` hanya jika setup atau usage berubah.
* `.env.example` hanya jika configuration baru diperkenalkan.
* Story ini bila product rule lain berubah atau requirement baru dikonfirmasi.
* `AGENTS.md` hanya jika terdapat durable project-wide rule baru.

Completion report harus menyebutkan dokumentasi yang diperbarui atau alasan
mengapa suatu dokumen tidak perlu berubah.

---

## 17. Locked Product Decisions

* Public Holiday adalah global scheduling constraint bagi seluruh Team Member
  dan project.
* Public Holidays memiliki page sendiri di group Team Configuration, bukan di Members.
* Aggregate memiliki ID, Start Date, End Date, derived weekday Dates,
  Description, Created At, dan Updated At.
* Start/End Date adalah required date-only ISO; Description required, trimmed, maksimum
  100 karakter berdasarkan konvensi repository.
* Past, current, dan future dates boleh dikelola.
* Public Holiday menghasilkan Execution Capacity dan Commitment Capacity `0` dan
  mengalahkan Capacity Override.
* Daily Capacity, Buffer, dan Capacity Override records tidak dimodifikasi.
* Mutation tidak otomatis menjalankan scheduler; latest confirmed data tersedia
  pada calculation berikutnya.
* CRUD lengkap tersedia; delete adalah confirmed hard delete.
* Backend pagination default `5`, max `100`; active/upcoming lebih dulu, lalu
  expired, dan tiap group diurut Start Date, End Date, ID ascending.
* Optional exact-date Holiday Date filter tersedia; text search dan date range
  tidak tersedia untuk MVP.
* Satu weekday Date hanya boleh dimiliki satu Public Holiday; weekend dilewati;
  conflict satu tanggal menolak seluruh transaction.
* Calendar menandai weekend dan configured Public Holiday secara konsisten.
* Today untuk grouping list mengikuti required `APP_TIMEZONE`.
* Loading/error/empty/no-results, cache consistency, accessibility, dan
  responsive behavior mengikuti `AGENTS.md` serta shared project patterns.

---

## 18. Unresolved Questions

Tidak ada unresolved question yang menghalangi implementasi story ini.
