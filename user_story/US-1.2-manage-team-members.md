# US-1.2 — Manage Team Members

> **Product decision update — US-6.1:** Member Buffer now reduces Execution Capacity.
> A single Commitment Capacity no longer exists at Team Member master level because
> Commitment Capacity also requires the owning Project Buffer. Scheduler-derived
> Base Execution Capacity preserves the existing `0.5`-hour rounding rule after Member Buffer is applied.

> **Product decision update — US-6.2:** Daily Capacity and Member Buffer are
> scheduling-impacting mutations. Before confirmed save, the server simulates
> transitive cross-project impact. Open-only impact requires grouped Project-name
> confirmation and server revalidation. Any impacted Locked Project blocks the
> change atomically. Independent Projects are not recalculated. Member Buffer does not alter Actual BAU Capacity; it remains an Execution/Commitment planning input.

## User Story

**Sebagai** Engineering Lead,
**Saya ingin** mengelola data engineer beserta kapasitas hariannya,
**Sehingga** scheduler dapat menghitung timeline berdasarkan kapasitas engineer yang tersedia.

---

## Business Context

Team Member merupakan resource yang dapat dipilih sebagai assignee pada Executable Leaf.

Data Team Member digunakan oleh Scheduling Engine sebagai constraint untuk menentukan:

- Kapasitas kerja harian.
- Execution Timeline.
- Commitment Timeline.
- Forecast Timeline.
- Urutan task untuk assignee yang sama.
- Alokasi task lintas project.

---

## Scope

Engineering Lead dapat:

1. Melihat daftar Team Member.
2. Menambahkan Team Member.
3. Mengubah data Team Member.
4. Menghapus Team Member yang belum digunakan.
5. Menentukan Role.
6. Menentukan Daily Capacity.
7. Menentukan Buffer.
8. Melihat Base Execution Capacity sebagai preview dari Daily Capacity dan Member Buffer.
9. Meninjau grouped impacted Project warning sebelum menyimpan Daily Capacity atau Member Buffer yang memengaruhi schedule lain.

---

## Out of Scope

User story ini tidak mencakup:

- Capacity Override.
- Public Holiday.
- Pengaturan kapasitas per hari tertentu.
- Penugasan task kepada Team Member.
- Concrete scheduling algorithm; this story only owns impact preview/confirmation coordination for capacity mutations.
- Menampilkan hasil timeline.
- Import atau sinkronisasi user dari sistem lain.
- Authentication atau user account management.

Capacity Override dan Public Holiday harus dibuat melalui user story terpisah.

---

# Domain Model

## Team Member

| Field                    | Type                        | Required | Source       | Rules                                           |
| ------------------------ | --------------------------- | -------: | ------------ | ----------------------------------------------- |
| ID                       | System-generated identifier |       Ya | System       | Dibuat otomatis dan tidak dapat diubah          |
| Name                     | String                      |       Ya | User         | Di-trim dan tidak boleh kosong                  |
| Role ID                  | Identifier                  |       Ya | User         | Harus mereferensikan Role yang tersedia         |
| Daily Capacity           | Decimal hours               |       Ya | User         | Kapasitas dasar untuk Execution Timeline        |
| Member Buffer Percentage | Decimal percentage          |       Ya | User/default | Default 20%; API field tetap `bufferPercentage` |
| Created At               | DateTime                    |       Ya | System       | Dibuat otomatis                                 |
| Updated At               | DateTime                    |       Ya | System       | Diperbarui otomatis                             |

---

# Capacity Definitions

## Daily Capacity

Daily Capacity adalah kapasitas kerja dasar Team Member dalam satu hari kerja.

Satuan:

```text
Hours
```

Contoh:

```text
8
7.5
6
```

Daily Capacity adalah kapasitas dasar sebelum precedence Public Holiday,
minimum active Capacity Override per Member/Date, dan Member Buffer diterapkan
oleh Scheduling Engine.

---

## Member Buffer

Member Buffer mengurangi kapasitas yang dapat digunakan pada Execution Timeline.
Nilai ini dimiliki Team Member dan berlaku setelah scheduler menentukan Resolved
Daily Capacity untuk tanggal terkait.

Default:

```text
20%
```

Formula scheduler:

```text
Raw Execution Capacity =
Resolved Daily Capacity × (1 - Member Buffer Percentage / 100)

Execution Capacity =
MROUND(Raw Execution Capacity, 0.5)
```

Contoh preview pada master Team Member, sebelum Public Holiday atau Capacity
Override tanggal tertentu diketahui:

```text
Daily Capacity: 8 jam
Member Buffer: 20%

Base Execution Capacity:
MROUND(8 × 80%, 0.5) = 6.5 jam
```

Base Execution Capacity dibulatkan ke kelipatan `0.5` jam terdekat setelah Member Buffer diterapkan. Input Daily Capacity juga tetap menggunakan increment `0.5` jam sesuai validation rule.

Commitment Capacity tidak dapat dihitung hanya dari Team Member karena juga
membutuhkan Project Buffer:

```text
Commitment Capacity =
MROUND(
  Resolved Daily Capacity
  × (1 - Member Buffer Percentage / 100)
  × (1 - Project Buffer Percentage / 100),
  0.5
)
```

Karena Project Buffer dapat berbeda antar-Project, Team Member master tidak
menampilkan satu Commitment Capacity global.

---

## Capacity Resolution Context

User story ini hanya mengelola kapasitas dasar Team Member.

Pada saat scheduler dijalankan, capacity mengikuti urutan berikut:

1. Public Holiday → Resolved Daily Capacity `0`.
2. Jika bukan Public Holiday dan satu atau lebih Capacity Override berlaku →
   gunakan Capacity terkecil dari seluruh active overrides untuk Member/Date.
3. Jika tidak ada active override → gunakan Team Member Daily Capacity.
4. Override result merupakan base capacity, bukan final capacity.
5. Terapkan Member Buffer dan pembulatan `0.5` jam → Execution Capacity.
6. Untuk Commitment, terapkan Member Buffer lalu owning Project Buffer ke
   Resolved Daily Capacity, kemudian bulatkan final Commitment Capacity ke
   kelipatan `0.5` jam.

Implementasi allocation dan timeline merupakan bagian dari US-6.1. Story ini
hanya mengelola Daily Capacity dan Member Buffer sebagai scheduler input.

---

# Implementation Decisions

Rule berikut merupakan keputusan implementasi agar perilaku sistem deterministik.

## Name Rules

- Nama di-trim sebelum divalidasi dan disimpan.
- Nama minimum 1 karakter setelah trim.
- Nama maksimum 100 karakter.
- Nama tidak wajib unik karena dua engineer dapat memiliki nama yang sama.
- ID menjadi identitas utama Team Member.

## Daily Capacity Rules

- Nilai minimum: lebih besar dari `0`.
- Nilai maksimum: `24`.
- Maksimal satu angka desimal.
- Increment yang diterima: `0.5` jam.
- Contoh valid: `0.5`, `6`, `7.5`, `8`, `24`.
- Contoh tidak valid: `0`, `-1`, `7.2`, `24.5`.

## Buffer Rules

- Nilai minimum: `0`.
- Nilai maksimum eksklusif: `100`.
- Maksimal dua angka desimal.
- Default saat tidak dikirim: `20`.
- Contoh valid: `0`, `10`, `20`, `25.5`, `99.99`.
- Contoh tidak valid: `-1`, `100`, `120`.

Member Buffer `100%` tidak diperbolehkan karena menghasilkan Execution Capacity `0` pada setiap hari non-holiday dan membuat Task tidak dapat dijadwalkan pada Execution maupun Commitment Timeline.

## Delete Rules

Team Member tidak dapat dihapus jika menjadi assignee task aktif. Task aktif
adalah task yang WBS level `0` atau Project root-nya belum berstatus `Closed`.
Jika minimal satu active-assignment projection tersedia, penghapusan ditolak
untuk menjaga pekerjaan aktif.

Jika tidak memiliki task aktif, Team Member, seluruh Capacity Override miliknya,
dan seluruh relasi Sprint Member di-soft-delete/dihapus dalam satu transaksi.
Sprint membership saja tidak boleh memblokir penghapusan Member. Sprint Task
yang tetap tersimpan tidak dihapus; pada pembacaan Sprint berikutnya Task tersebut
mengikuti warning `Needs Review` milik US-8.1. Assignment dan persisted timeline
untuk Project yang sudah `Closed` tetap dipertahankan. Nama Team Member yang
soft-deleted boleh digunakan kembali. Restore belum termasuk scope.

Sampai Project Management menyediakan WBS root dan status lifecycle,
`executable_leaves` merupakan provisional active-assignment projection.
Keputusan ini wajib direview ketika Project Management diimplementasikan.

---

# Derived Values

## Base Execution Capacity Preview

Pada master Team Member:

```text
Raw Base Execution Capacity =
Daily Capacity × (1 - Member Buffer Percentage / 100)

Base Execution Capacity =
MROUND(Raw Base Execution Capacity, 0.5)
```

Preview ini tidak memperhitungkan:

- Public Holiday.
- Capacity Override.
- Project Freeze.

Base Execution Capacity:

- Bukan source of truth yang disimpan.
- Dibulatkan secara deterministik ke kelipatan `0.5` jam terdekat.
- Harus konsisten di backend dan frontend bila ditampilkan.

## Project-specific Commitment Capacity

Commitment Capacity hanya dihitung dalam konteks Project oleh US-6.1:

```text
Commitment Capacity =
MROUND(
  Resolved Daily Capacity
  × (1 - Member Buffer Percentage / 100)
  × (1 - Project Buffer Percentage / 100),
  0.5
)
```

Team Member master tidak menyimpan atau menampilkan satu Commitment Capacity
global karena nilai tersebut berbeda menurut Project Buffer.

---

# Acceptance Criteria

## AC-1 — Menampilkan daftar Team Member

**Given** Engineering Lead membuka halaman Team Member Management
**When** data berhasil dimuat
**Then** sistem menampilkan seluruh Team Member
**And** setiap Team Member menampilkan:

- Name.
- Role.
- Daily Capacity.
- Buffer Percentage.
- Base Execution Capacity.

**And** daftar diurutkan berdasarkan Name secara ascending tanpa membedakan huruf besar-kecil
**And** jika nama sama, urutan ditentukan berdasarkan ID secara ascending
**And** sistem menyediakan pencarian Name secara case-insensitive
**And** ketika tidak ada Team Member yang cocok, sistem menampilkan no-results state yang berbeda dari empty state
**And** sistem menyediakan aksi untuk menghapus pencarian.

---

## AC-2 — Empty state

**Given** belum terdapat Team Member
**When** Engineering Lead membuka halaman Team Member Management
**Then** sistem menampilkan empty state
**And** sistem menyediakan aksi `Add Team Member`.
**And** aksi tersebut membuka form yang sama dengan aksi utama pada header halaman.

---

## AC-3 — Menambahkan Team Member dengan data valid

**Given** minimal satu Role tersedia
**When** Engineering Lead mengisi:

- Name valid.
- Role valid.
- Daily Capacity valid.
- Buffer valid.

**And** menyimpan form
**Then** sistem membuat Team Member
**And** sistem menyimpan Name setelah trim
**And** Team Member muncul pada daftar
**And** Base Execution Capacity preview dihitung otomatis
**And** sistem menampilkan notifikasi keberhasilan.

---

## AC-4 — Default Buffer

**Given** Engineering Lead membuka form Add Team Member
**When** form pertama kali ditampilkan
**Then** Buffer Percentage bernilai default `20`.

**When** request create tidak mengirim Buffer Percentage
**Then** backend menggunakan Buffer Percentage `20`.

---

## AC-5 — Name wajib diisi

**Given** form Add atau Edit Team Member dibuka
**When** Engineering Lead mengetik Name
**Then** frontend mempertahankan draft tanpa transformasi
**When** field Name kehilangan fokus atau form disimpan
**Then** frontend mengubah huruf alfabet pertama setiap kata menjadi huruf kapital
**And** frontend tidak mengubah huruf lainnya secara otomatis.

**When** Name kosong atau hanya whitespace
**Then** sistem menolak penyimpanan
**And** menampilkan pesan:

```text
Team member name is required
```

**And** tidak ada data yang dibuat atau diubah.

---

## AC-6 — Batas maksimum Name

**Given** form Add atau Edit Team Member dibuka
**When** Name setelah trim memiliki panjang lebih dari 100 karakter
**Then** sistem menolak penyimpanan
**And** menampilkan pesan:

```text
Team member name must not exceed 100 characters
```

---

## AC-7 — Role wajib dipilih

**Given** form Add atau Edit Team Member dibuka
**When** form Add pertama kali ditampilkan
**Then** Role belum terpilih
**And** Engineering Lead dapat mencari Role berdasarkan nama
**And** pilihan hanya berasal dari Role yang tersedia.
**And** pencarian kosong menampilkan seluruh Role
**And** setiap huruf yang diketik memfilter Role berdasarkan awalan nama tanpa membedakan huruf besar-kecil
**And** ketika Engineering Lead menekan Enter pada pencarian Role, sistem memilih hasil teratas
**And** Enter tersebut tidak menyimpan atau mengirim form.
**And** setelah Role dipilih dengan Enter, fokus berpindah ke field Daily Capacity.

**When** Role tidak dipilih
**Then** sistem menolak penyimpanan
**And** menampilkan pesan:

```text
Role is required
```

---

## AC-8 — Role harus tersedia

**Given** request mereferensikan Role ID yang tidak tersedia
**When** request create atau update diproses
**Then** sistem menolak request
**And** mengembalikan not-found response
**And** tidak ada Team Member yang dibuat atau diubah.

---

## AC-9 — Daily Capacity wajib diisi

**Given** form Add atau Edit Team Member dibuka
**When** Daily Capacity tidak diisi
**Then** sistem menolak penyimpanan
**And** menampilkan pesan:

```text
Daily capacity is required
```

---

## AC-10 — Daily Capacity harus lebih besar dari nol

**Given** Engineering Lead memasukkan Daily Capacity `0` atau nilai negatif
**When** data disimpan
**Then** sistem menolak penyimpanan
**And** menampilkan pesan:

```text
Daily capacity must be greater than 0
```

---

## AC-11 — Daily Capacity tidak boleh melebihi 24 jam

**Given** Engineering Lead memasukkan Daily Capacity lebih besar dari `24`
**When** data disimpan
**Then** sistem menolak penyimpanan
**And** menampilkan pesan:

```text
Daily capacity must not exceed 24 hours
```

---

## AC-12 — Daily Capacity harus dalam increment 0,5 jam

**Given** Engineering Lead mengisi Daily Capacity pada form
**Then** input menerima nilai desimal, termasuk nilai seperti `7.5`
**And** frontend mempertahankan draft tanpa transformasi ketika nilai masih diketik
**When** field kehilangan fokus atau form disimpan
**Then** frontend membulatkan nilai ke kelipatan `0.5` terdekat.

**Given** Engineering Lead mengisi Buffer Percentage pada form
**Then** frontend mempertahankan draft tanpa transformasi ketika nilai masih diketik
**When** field kehilangan fokus atau form disimpan
**Then** frontend membulatkan nilai ke kelipatan `0.5` terdekat.

**Given** request API mengirim Daily Capacity yang bukan kelipatan `0.5`
**When** request diproses backend
**Then** sistem menolak penyimpanan
**And** menampilkan pesan:

```text
Daily capacity must use 0.5-hour increments
```

---

## AC-13 — Buffer wajib valid

**Given** Engineering Lead memasukkan Buffer Percentage
**When** nilainya lebih kecil dari `0` atau sama dengan/lebih besar dari `100`
**Then** sistem menolak penyimpanan
**And** menampilkan pesan:

```text
Buffer must be between 0 and less than 100
```

---

## AC-14 — Menghitung Base Execution Capacity

**Given** Daily Capacity dan Buffer valid
**When** Team Member ditampilkan atau disimpan
**Then** sistem menghitung Base Execution Capacity menggunakan Daily Capacity dan Member Buffer
**And** hasil dibulatkan ke kelipatan `0.5` jam terdekat
**And** frontend dan backend menghasilkan nilai yang sama.

---

## AC-15 — Mengubah data Team Member

**Given** Team Member tersedia
**When** Engineering Lead mengubah Name, Role, Daily Capacity, atau Buffer menggunakan nilai valid
**Then** sistem menyimpan perubahan
**And** ID Team Member tetap sama
**And** Created At tetap sama
**And** Updated At diperbarui
**And** Base Execution Capacity preview dihitung ulang
**And** referensi task terhadap Team Member tetap valid.

---

## AC-16 — Perubahan capacity menggunakan impact guard

**Given** Team Member digunakan sebagai assignee
**When** Daily Capacity atau Buffer diperbarui
**Then** nilai terbaru menjadi input yang digunakan pada scheduling berikutnya
**And** scheduler tidak menggunakan Daily Capacity atau Member Buffer lama yang tersimpan atau ter-cache
**And** cross-project warning/blocking/revalidation follows US-6.2.

Confirmed Daily Capacity or Member Buffer update follows US-6.2 impact preview/confirmation. After confirmation, the mutation and all allowed impacted Open-Project recalculation are persisted atomically; Locked impact blocks save.

---

## AC-17 — Membuka konfirmasi penghapusan

**Given** Engineering Lead memilih Delete pada Team Member
**When** dialog konfirmasi ditampilkan
**Then** Team Member belum dihapus
**And** dialog menampilkan nama Team Member
**And** dialog menjelaskan bahwa Member dan Capacity Override dikeluarkan dari
active planning, historical records dipertahankan, dan hanya task aktif yang
dapat menghalangi penghapusan
**And** tersedia aksi Cancel dan Delete.

---

## AC-18 — Membatalkan penghapusan

**Given** dialog konfirmasi penghapusan ditampilkan
**When** Engineering Lead memilih Cancel
**Then** dialog ditutup
**And** tidak ada request delete yang dikirim
**And** Team Member tetap tersedia.

---

## AC-19 — Menghapus Team Member yang tidak digunakan

**Given** Team Member tidak memiliki task aktif
**When** Engineering Lead mengonfirmasi penghapusan
**Then** sistem melakukan soft delete pada Team Member dan seluruh Capacity Override miliknya serta menghapus seluruh relasi Sprint Member dalam satu transaksi
**And** Team Member tidak lagi muncul dalam daftar
**And** sistem menampilkan notifikasi keberhasilan.

---

## AC-20 — Menolak penghapusan Team Member yang menjadi assignee

**Given** Team Member menjadi assignee task yang Project/WBS level `0` belum `Closed`
**When** Engineering Lead mencoba menghapus Team Member
**Then** sistem menolak penghapusan
**And** mengembalikan conflict response
**And** menampilkan pesan:

```text
Team member is assigned to one or more tasks and cannot be deleted
```

**And** Team Member serta seluruh referensinya tetap tersedia.

---

## AC-21 — Cascade soft delete Capacity Override

**Given** Team Member tidak memiliki task aktif dan memiliki Capacity Override
**When** Engineering Lead mengonfirmasi penghapusan
**Then** sistem menerima penghapusan
**And** Team Member serta seluruh Capacity Override miliknya di-soft-delete dan seluruh relasi Sprint Member dihapus secara atomik
**And** data tersebut tidak tersedia pada operational list, detail, selector, atau scheduling input baru.

---

## AC-22 — Team Member tidak ditemukan

**Given** request update atau delete menggunakan Team Member ID yang tidak tersedia
**When** request diproses
**Then** sistem mengembalikan not-found response
**And** tidak ada data lain yang dibuat, diubah, atau dihapus.

---

## AC-23 — Validasi backend tetap berlaku

**Given** frontend validation dilewati
**When** request invalid dikirim langsung ke API
**Then** backend tetap menjalankan seluruh domain validation
**And** database tidak menyimpan data invalid.

---

## AC-24 — Referential integrity setelah Role diubah

**Given** Team Member menggunakan Role Backend
**When** Team Member diubah menggunakan Role Frontend
**Then** referensi Role berubah ke Role Frontend
**And** ID Team Member tetap sama
**And** task yang menggunakan Team Member sebagai assignee tetap valid.

**When** nama Role yang direferensikan Team Member diubah
**Then** daftar Team Member menampilkan nama Role terbaru
**And** form Edit Team Member memilih Role menggunakan nama terbaru
**And** pengguna tidak perlu melakukan hard refresh.

---

## AC-25 — Daftar Team Member menggunakan pagination

**Given** daftar Team Member tersedia
**When** Engineering Lead membuka halaman Members
**Then** sistem meminta halaman pertama dengan ukuran default 5
**And** sistem menampilkan rentang item, jumlah total item, halaman aktif, dan jumlah halaman
**And** Engineering Lead dapat berpindah ke halaman sebelumnya atau berikutnya ketika tersedia.

**When** Engineering Lead mengubah pencarian
**Then** pencarian dijalankan oleh backend tanpa membedakan huruf besar-kecil
**And** pagination kembali ke halaman pertama
**And** response hanya memuat halaman hasil yang diminta.

---

# API Contract

Endpoint dapat disesuaikan dengan konvensi repository. Perilakunya harus setara dengan kontrak berikut.

## List Team Members

```http
GET /api/team-members?search=har&page=1&pageSize=5
```

### Success Response

```json
{
  "data": [
    {
      "id": "member-id",
      "name": "Harry",
      "role": {
        "id": "role-id",
        "name": "Backend"
      },
      "dailyCapacity": 8,
      "bufferPercentage": 20,
      "baseExecutionCapacity": 6.5,
      "createdAt": "2026-07-24T10:00:00Z",
      "updatedAt": "2026-07-24T10:00:00Z"
    }
  ],
  "page": 1,
  "pageSize": 5,
  "total": 1
}
```

Status:

```text
200 OK
```

---

## Get Team Member

```http
GET /api/team-members/{teamMemberId}
```

Status sukses:

```text
200 OK
```

Status gagal:

| Condition                   |        Status |
| --------------------------- | ------------: |
| Team Member tidak ditemukan | 404 Not Found |

---

## Create Team Member

```http
POST /api/team-members
Content-Type: application/json
```

```json
{
  "name": "Harry",
  "roleId": "role-id",
  "dailyCapacity": 8,
  "bufferPercentage": 20
}
```

### Success Response

```json
{
  "data": {
    "id": "member-id",
    "name": "Harry",
    "role": {
      "id": "role-id",
      "name": "Backend"
    },
    "dailyCapacity": 8,
    "bufferPercentage": 20,
    "baseExecutionCapacity": 6.5,
    "createdAt": "2026-07-24T10:00:00Z",
    "updatedAt": "2026-07-24T10:00:00Z"
  }
}
```

Status:

```text
201 Created
```

Status kegagalan:

| Condition            |          Status |
| -------------------- | --------------: |
| Input tidak valid    | 400 Bad Request |
| Role tidak ditemukan |   404 Not Found |

---

## Update Team Member

```http
PUT /api/team-members/{teamMemberId}
Content-Type: application/json
```

```json
{
  "name": "Harry Wijaya",
  "roleId": "role-id",
  "dailyCapacity": 7.5,
  "bufferPercentage": 20
}
```

Status sukses:

```text
200 OK
```

Status kegagalan:

| Condition                   |          Status |
| --------------------------- | --------------: |
| Input tidak valid           | 400 Bad Request |
| Role tidak ditemukan        |   404 Not Found |
| Team Member tidak ditemukan |   404 Not Found |

---

## Delete Team Member

```http
DELETE /api/team-members/{teamMemberId}
```

Status sukses:

```text
204 No Content
```

Status kegagalan:

| Condition                       |        Status |
| ------------------------------- | ------------: |
| Team Member tidak ditemukan     | 404 Not Found |
| Team Member memiliki task aktif |  409 Conflict |

---

# Error Response

Seluruh API error menggunakan format konsisten:

```json
{
  "code": "DAILY_CAPACITY_INVALID_INCREMENT",
  "message": "Daily capacity must use 0.5-hour increments",
  "field": "dailyCapacity"
}
```

Error code minimum:

- `TEAM_MEMBER_NAME_REQUIRED`
- `TEAM_MEMBER_NAME_TOO_LONG`
- `ROLE_REQUIRED`
- `ROLE_NOT_FOUND`
- `DAILY_CAPACITY_REQUIRED`
- `DAILY_CAPACITY_NOT_POSITIVE`
- `DAILY_CAPACITY_EXCEEDS_LIMIT`
- `DAILY_CAPACITY_INVALID_INCREMENT`
- `BUFFER_OUT_OF_RANGE`
- `TEAM_MEMBER_NOT_FOUND`
- `TEAM_MEMBER_ASSIGNED_TO_TASK`

---

# Test Cases

## TC-1 — Menampilkan daftar Team Member

### Precondition

Data tersedia:

| Name  | Role     | Daily Capacity | Buffer |
| ----- | -------- | -------------: | -----: |
| Sandi | QA       |              8 |     20 |
| Harry | Backend  |              8 |     20 |
| Dewi  | Frontend |            7.5 |     10 |

### Steps

1. Buka halaman Team Member Management.

### Expected Result

1. Sistem menampilkan tiga Team Member.
2. Urutan berdasarkan Name:
   - Dewi.
   - Harry.
   - Sandi.

3. Setiap row menampilkan Role, Daily Capacity, Member Buffer, dan Base Execution Capacity.

---

## TC-2 — Empty state

### Precondition

Tidak ada Team Member.

### Steps

1. Buka halaman Team Member Management.

### Expected Result

1. Empty state ditampilkan.
2. Aksi `Add Team Member` tersedia.

---

## TC-3 — Membuat Team Member valid

### Precondition

Role Backend tersedia.

### Steps

1. Buka Add Team Member.
2. Isi Name `Harry`.
3. Pilih Role Backend.
4. Isi Daily Capacity `8`.
5. Biarkan Buffer `20`.
6. Simpan.

### Expected Result

1. API mengembalikan `201`.
2. Team Member tersimpan.
3. Base Execution Capacity bernilai `6.5`.
4. ID dan timestamp dibuat otomatis.

---

## TC-4 — Default Buffer pada form

### Steps

1. Buka Add Team Member.

### Expected Result

1. Buffer otomatis bernilai `20`.

---

## TC-5 — Default Buffer pada backend

### Steps

1. Kirim create request tanpa `bufferPercentage`.

### Expected Result

1. Request berhasil.
2. Buffer tersimpan sebagai `20`.
3. Base Execution Capacity dihitung menggunakan Member Buffer `20`.

---

## TC-6 — Trim Name

### Steps

1. Buat Team Member dengan Name `  Harry  `.
2. Simpan.

### Expected Result

1. Name tersimpan sebagai `Harry`.

---

## TC-7 — Name kosong

### Test Data

```text
""
```

### Expected Result

1. API mengembalikan `400`.
2. Error code `TEAM_MEMBER_NAME_REQUIRED`.
3. Tidak ada data tersimpan.

---

## TC-8 — Name hanya whitespace

### Test Data

```text
"     "
```

### Expected Result

1. API mengembalikan `400`.
2. Error code `TEAM_MEMBER_NAME_REQUIRED`.

---

## TC-9 — Name tepat 100 karakter

### Steps

1. Buat Team Member dengan Name sepanjang 100 karakter.

### Expected Result

1. Request berhasil.

---

## TC-10 — Name 101 karakter

### Steps

1. Buat Team Member dengan Name sepanjang 101 karakter.

### Expected Result

1. API mengembalikan `400`.
2. Error code `TEAM_MEMBER_NAME_TOO_LONG`.

---

## TC-11 — Nama Team Member boleh sama

### Steps

1. Buat Team Member `Andi`.
2. Buat Team Member lain dengan Name `Andi`.

### Expected Result

1. Kedua request berhasil.
2. Kedua Team Member memiliki ID berbeda.

---

## TC-12 — Role tidak diisi

### Steps

1. Kirim create request tanpa `roleId`.

### Expected Result

1. API mengembalikan `400`.
2. Error code `ROLE_REQUIRED`.

---

## TC-13 — Role tidak ditemukan

### Steps

1. Kirim create request dengan Role ID yang tidak tersedia.

### Expected Result

1. API mengembalikan `404`.
2. Error code `ROLE_NOT_FOUND`.
3. Team Member tidak dibuat.

---

## TC-14 — Daily Capacity valid

Uji masing-masing nilai:

```text
0.5
6
7.5
8
24
```

### Expected Result

1. Seluruh nilai diterima.
2. Nilai tersimpan secara presisi.

---

## TC-15 — Daily Capacity kosong

### Steps

1. Kirim create request tanpa `dailyCapacity`.

### Expected Result

1. API mengembalikan `400`.
2. Error code `DAILY_CAPACITY_REQUIRED`.

---

## TC-16 — Daily Capacity nol

### Test Data

```text
0
```

### Expected Result

1. API mengembalikan `400`.
2. Error code `DAILY_CAPACITY_NOT_POSITIVE`.

---

## TC-17 — Daily Capacity negatif

### Test Data

```text
-1
```

### Expected Result

1. API mengembalikan `400`.
2. Error code `DAILY_CAPACITY_NOT_POSITIVE`.

---

## TC-18 — Daily Capacity melebihi 24

### Test Data

```text
24.5
```

### Expected Result

1. API mengembalikan `400`.
2. Error code `DAILY_CAPACITY_EXCEEDS_LIMIT`.

---

## TC-19 — Daily Capacity bukan kelipatan 0,5

Uji nilai:

```text
7.2
7.25
8.1
```

### Expected Result

1. Seluruh request ditolak.
2. API mengembalikan `400`.
3. Error code `DAILY_CAPACITY_INVALID_INCREMENT`.

---

## TC-20 — Buffer nol

### Test Data

```text
0
```

### Expected Result

1. Request berhasil.
2. Base Execution Capacity sama dengan Daily Capacity.

---

## TC-21 — Buffer valid desimal

### Test Data

```text
25.5
```

### Expected Result

1. Request berhasil.
2. Buffer tersimpan secara presisi.

---

## TC-22 — Buffer negatif

### Test Data

```text
-1
```

### Expected Result

1. API mengembalikan `400`.
2. Error code `BUFFER_OUT_OF_RANGE`.

---

## TC-23 — Buffer sama dengan 100

### Test Data

```text
100
```

### Expected Result

1. API mengembalikan `400`.
2. Error code `BUFFER_OUT_OF_RANGE`.

---

## TC-24 — Buffer lebih besar dari 100

### Test Data

```text
120
```

### Expected Result

1. API mengembalikan `400`.
2. Error code `BUFFER_OUT_OF_RANGE`.

---

## TC-25 — Perhitungan Base Execution Capacity 8 jam

### Test Data

```text
Daily Capacity: 8
Buffer: 20
```

### Expected Result

```text
Base Execution Capacity: 6.5
Rounded to nearest 0.5 hour
```

---

## TC-26 — Perhitungan Base Execution Capacity 7,5 jam

### Test Data

```text
Daily Capacity: 7.5
Buffer: 20
```

### Expected Result

```text
Base Execution Capacity: 6
Rounded to nearest 0.5 hour
```

---

## TC-27 — Base Execution Capacity half-hour result

### Test Data

```text
Daily Capacity: 8
Buffer: 25
```

### Expected Result

```text
Base Execution Capacity: 6
Rounded to nearest 0.5 hour
```

---

## TC-28 — Mengubah Team Member

### Precondition

Harry memiliki:

```text
Role: Backend
Daily Capacity: 8
Buffer: 20
```

### Steps

1. Ubah Name menjadi `Harry Wijaya`.
2. Ubah Role menjadi Frontend.
3. Ubah Daily Capacity menjadi `7.5`.
4. Ubah Buffer menjadi `10`.
5. Simpan.

### Expected Result

1. API mengembalikan `200`.
2. ID tidak berubah.
3. Created At tidak berubah.
4. Updated At berubah.
5. Base Execution Capacity menjadi:

```text
MROUND(7.5 × 90%, 0.5) = 7
```

Raw value `6.75` dibulatkan menjadi `7` sesuai aturan kelipatan `0.5` jam.

---

## TC-29 — Task tetap mereferensikan Team Member setelah update

### Precondition

1. Harry menjadi assignee Task A.
2. ID Harry adalah `member-1`.

### Steps

1. Ubah nama Harry.
2. Ubah Role Harry.
3. Buka Task A.

### Expected Result

1. Task A tetap mereferensikan `member-1`.
2. Nama dan Role terbaru ditampilkan.
3. Tidak terjadi orphan reference.

---

## TC-30 — Menghapus Team Member yang tidak digunakan

### Precondition

Team Member tidak memiliki task aktif dan boleh memiliki Capacity Override.

### Steps

1. Pilih Delete.
2. Konfirmasi.

### Expected Result

1. API mengembalikan `204`.
2. Team Member dan seluruh Capacity Override miliknya di-soft-delete.
3. Seluruh relasi Sprint Member miliknya dihapus tanpa menghapus Sprint Task.
4. Team Member tidak muncul pada daftar.

---

## TC-31 — Membatalkan penghapusan

### Steps

1. Pilih Delete.
2. Pilih Cancel.

### Expected Result

1. Tidak ada request delete.
2. Data tetap tersedia.

---

## TC-32 — Menolak delete Team Member yang menjadi assignee

### Precondition

Harry menjadi assignee Task A pada Project yang belum `Closed`.

### Steps

1. Hapus Harry.

### Expected Result

1. API mengembalikan `409`.
2. Error code `TEAM_MEMBER_ASSIGNED_TO_TASK`.
3. Harry dan Task A tidak berubah.

---

## TC-33 — Cascade soft delete Capacity Override

### Precondition

Harry memiliki minimal satu Capacity Override.

### Steps

1. Hapus Harry.

### Expected Result

1. API mengembalikan `204`.
2. Harry dan seluruh Capacity Override miliknya tidak tersedia pada operational query.
3. Seluruh relasi Sprint Member Harry dihapus; retained Sprint Task mengikuti Needs Review pada US-8.1.
4. Unscoped historical persistence mempertahankan Member dan Override beserta deletion timestamp.

---

## TC-34 — Update Team Member tidak ditemukan

### Steps

1. Kirim update untuk ID yang tidak tersedia.

### Expected Result

1. API mengembalikan `404`.
2. Error code `TEAM_MEMBER_NOT_FOUND`.

---

## TC-35 — Delete Team Member tidak ditemukan

### Steps

1. Kirim delete untuk ID yang tidak tersedia.

### Expected Result

1. API mengembalikan `404`.
2. Error code `TEAM_MEMBER_NOT_FOUND`.

---

## TC-36 — Backend menolak request invalid langsung

### Steps

Kirim langsung ke API:

1. Name kosong.
2. Role kosong.
3. Role tidak tersedia.
4. Daily Capacity nol.
5. Daily Capacity `7.2`.
6. Buffer `100`.

### Expected Result

1. Semua request invalid ditolak.
2. Status dan error code sesuai.
3. Database tidak menyimpan partial data.

---

## TC-37 — Transaction rollback

### Precondition

Role ID pada request tidak tersedia.

### Steps

1. Kirim request create Team Member.

### Expected Result

1. Request gagal.
2. Tidak ada row Team Member yang tersimpan.
3. Tidak ada derived data yang tersimpan sebagian.

---

## TC-38 — Capacity update impact and latest value

### Precondition

1. Harry memiliki Daily Capacity `8`.
2. Daily Capacity diubah menjadi `6`.

### Steps

1. Ambil data Team Member yang digunakan sebagai input scheduling.

### Expected Result

1. Scheduler menerima Daily Capacity `6`.
2. Scheduler tidak menerima nilai lama `8`.

Test ini dapat diimplementasikan sebagai application integration test tanpa menjalankan algoritma scheduler penuh.

---

## TC-39 — Rename Role tercermin pada Member

### Precondition

1. Harry menggunakan Role `Backend`.
2. Daftar Member pernah dimuat sehingga hasil dapat berada di cache frontend.

### Steps

1. Ubah nama Role `Backend` menjadi `Backend Engineer`.
2. Buka daftar Member tanpa melakukan hard refresh.
3. Buka form Edit Harry.

### Expected Result

1. Daftar menampilkan Role `Backend Engineer`.
2. Form Edit memilih Role `Backend Engineer`.
3. Team Member tetap mereferensikan Role ID yang sama.
4. Data Member atau projection yang lama tidak ditampilkan dari cache.

---

# Required Automated Tests

## Domain Unit Tests

Wajib menguji:

- Name validation dan trim.
- Daily Capacity minimum.
- Daily Capacity maksimum.
- Increment `0.5`.
- Buffer range.
- Default Buffer.
- Base Execution Capacity calculation.
- Rounded to nearest 0.5 hour.
- Update mempertahankan ID.
- Update mempertahankan Created At.

---

## Application Tests

Wajib menguji:

- Create Team Member.
- List Team Member.
- Get Team Member.
- Update Team Member.
- Delete unused Team Member.
- Reject Role not found.
- Reject Team Member not found.
- Reject delete saat menjadi assignee.
- Cascade soft delete Capacity Override saat Member dihapus.
- Transaction rollback.
- Nilai capacity terbaru tersedia sebagai scheduling input.

Repository dependency harus dimock atau menggunakan test double pada unit test application.

---

## Repository Integration Tests

Wajib menguji:

- Persist dan retrieve Team Member.
- Foreign key Role.
- Foreign key Executable Leaf assignee.
- Foreign key Capacity Override.
- Active task delete restriction.
- Transactional soft delete Member dan Capacity Override.
- Active query mengecualikan soft-deleted records.
- Nama Member dapat digunakan kembali setelah soft delete.
- Decimal capacity persistence.
- Timestamp behavior.
- Transaction rollback.

---

## API Integration Tests

Wajib menguji:

- HTTP method dan path.
- Request validation.
- HTTP status.
- Success response.
- Error response.
- Field mapping.
- Derived Base Execution Capacity.
- Database persistence.
- Not-found behavior.
- Conflict behavior.
- Backend search dan pagination, termasuk metadata serta default page size.

---

## Frontend Tests

Wajib menguji:

- Render list.
- Skeleton saat initial load dan selama remote search.
- Search list dan clear search.
- No-results state berbeda dari empty state.
- Pagination dan reset ke halaman pertama setelah search berubah.
- Empty state.
- Add form.
- Default Buffer.
- Searchable Role selector.
- Role selector default kosong pada Add form.
- Enter pada Role selector memilih hasil teratas tanpa submit form.
- Fokus berpindah ke Daily Capacity setelah pemilihan Role dengan Enter.
- Role list menampilkan seluruh Role saat pencarian kosong.
- Role list difilter berdasarkan awalan nama pada setiap perubahan input.
- Name tidak berubah saat diketik dan dikapitalisasi per kata saat blur.
- Daily Capacity dan Buffer tidak berubah saat diketik.
- Daily Capacity dibulatkan ke kelipatan `0.5` saat blur; Member Buffer mempertahankan decimal draft dan divalidasi sesuai range/precision rule.
- Normalisasi dan validasi form dijalankan ulang saat submit.
- Client-side validation.
- Base Execution Capacity preview.
- Edit form.
- Delete confirmation.
- Cancel deletion.
- Conflict error.
- Refresh atau cache invalidation setelah mutation.
- Rename Role memperbarui nama pada daftar dan selected Role pada form Edit tanpa hard refresh.

---

# Test Priority

## Priority 1 — Mandatory

- Create valid Team Member.
- Required field validation.
- Role not found.
- Daily Capacity validation.
- Buffer validation.
- Base Execution Capacity formula.
- Update Team Member.
- Active task delete restriction dan cascade soft delete.
- Backend validation.
- Referential integrity.

## Priority 2 — Important

- Empty state.
- Sorting.
- Name trim.
- Duplicate names.
- Timestamp behavior.
- Transaction rollback.
- Frontend notification.
- Cache invalidation.

---

# Technical Completion Criteria

User story dianggap selesai jika:

1. Migration tabel Team Member tersedia.
2. Foreign key ke Role tersedia.
3. Daily Capacity disimpan sebagai decimal yang presisi, bukan floating-point binary yang berisiko menghasilkan rounding error.
4. Buffer disimpan sebagai decimal yang presisi.
5. Base Execution Capacity preview dihitung sebagai derived value.
6. Base Execution Capacity dibulatkan secara deterministik ke kelipatan `0.5`; Commitment Capacity dihitung dari Resolved Daily Capacity dengan Member Buffer dan Project Buffer, lalu final Commitment Capacity dibulatkan secara independen ke kelipatan `0.5`.
7. Backend tetap memvalidasi seluruh input.
8. Create, list, get, update, dan transactional soft delete tersedia.
9. Referential integrity dengan active task assignment dan Capacity Override terjaga.
10. Business rule tidak diletakkan pada HTTP handler atau database adapter.
11. Application layer tidak bergantung pada framework transport.
12. Error code konsisten dan dapat dipakai frontend.
13. Seluruh mandatory automated test tersedia dan lulus.
14. Static analysis, lint, type check, dan test suite lulus.
15. Dokumentasi API diperbarui.
16. Perubahan Daily Capacity dan Buffer dapat dibaca oleh Scheduling Engine pada proses scheduling berikutnya.
17. Tidak terdapat field Grade, Squad, atau Department.

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
