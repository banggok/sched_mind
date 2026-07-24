# US-1.1 — Manage Roles

## User Story

**Sebagai** Engineering Lead,
**Saya ingin** mengelola daftar role tim,
**Sehingga** team member dan task dapat diklasifikasikan berdasarkan role yang sesuai.

---

## Scope

Engineering Lead dapat:

1. Melihat daftar role.
2. Menambahkan role baru.
3. Mengubah nama role.
4. Menghapus role yang belum digunakan.
5. Melihat kegagalan validasi secara jelas.

Role merupakan master data sederhana.

Role tidak memiliki atribut:

* Grade
* Squad
* Department

---

## Data Model

### Role

| Field      | Type                                  | Required | Rules                                                           |
| ---------- | ------------------------------------- | -------: | --------------------------------------------------------------- |
| ID         | UUID atau system-generated identifier |       Ya | Dibuat otomatis dan tidak dapat diubah                          |
| Name       | String                                |       Ya | Harus unik, setelah trim dan tanpa membedakan huruf besar-kecil |
| Created At | DateTime                              |       Ya | Dibuat otomatis                                                 |
| Updated At | DateTime                              |       Ya | Diperbarui otomatis                                             |

### Name Rules

* Minimum 1 karakter setelah whitespace di awal dan akhir dihapus.
* Maksimum 100 karakter.
* Tidak boleh hanya berisi whitespace.
* Tidak boleh duplikat secara case-insensitive.
* Sistem menyimpan nama setelah proses trim.
* Karakter huruf, angka, spasi, tanda hubung, garis miring, dan tanda kurung diperbolehkan.
* ID role tidak berubah ketika nama role diubah.

Contoh nama valid:

* Backend
* Frontend
* Android
* iOS
* QA
* Backend Engineer
* QA Automation

---

## Acceptance Criteria

### AC-1 — Menampilkan daftar role

**Given** Engineering Lead membuka halaman Roles
**When** data role berhasil dimuat
**Then** sistem menampilkan seluruh role yang tersedia
**And** setiap role menampilkan minimal nama role
**And** daftar role diurutkan berdasarkan nama secara ascending tanpa membedakan huruf besar-kecil.

---

### AC-2 — Empty state

**Given** belum terdapat role dalam sistem
**When** Engineering Lead membuka halaman Roles
**Then** sistem menampilkan empty state
**And** sistem menyediakan aksi untuk menambahkan role baru.

---

### AC-3 — Menambahkan role

**Given** Engineering Lead berada pada halaman Roles
**When** Engineering Lead memasukkan nama role yang valid dan menyimpan data
**Then** sistem membuat role baru
**And** sistem melakukan trim terhadap whitespace di awal dan akhir nama
**And** role baru muncul pada daftar role
**And** sistem menampilkan notifikasi bahwa role berhasil dibuat.

---

### AC-4 — Nama role wajib diisi

**Given** Engineering Lead membuka form penambahan atau perubahan role
**When** nama role kosong atau hanya berisi whitespace
**Then** sistem menolak penyimpanan
**And** sistem menampilkan pesan validasi `Role name is required`
**And** tidak ada data yang dibuat atau diubah.

---

### AC-5 — Panjang maksimum nama role

**Given** Engineering Lead membuka form penambahan atau perubahan role
**When** nama role setelah trim memiliki panjang lebih dari 100 karakter
**Then** sistem menolak penyimpanan
**And** sistem menampilkan pesan validasi `Role name must not exceed 100 characters`
**And** tidak ada data yang dibuat atau diubah.

---

### AC-6 — Nama role harus unik

**Given** role dengan nama `Backend` sudah tersedia
**When** Engineering Lead mencoba membuat atau mengubah role lain dengan nama `Backend`, `backend`, atau `BACKEND`
**Then** sistem menolak penyimpanan
**And** sistem menampilkan pesan validasi `Role name already exists`
**And** tidak ada data yang dibuat atau diubah.

---

### AC-7 — Mengubah nama role

**Given** sebuah role sudah tersedia
**When** Engineering Lead mengubah nama role menggunakan nama yang valid dan unik
**Then** sistem menyimpan nama baru
**And** ID role tetap sama
**And** referensi team member terhadap role tersebut tetap valid
**And** sistem menampilkan notifikasi bahwa role berhasil diperbarui.

---

### AC-8 — Tidak ada perubahan saat nama tetap sama

**Given** role `Backend` sudah tersedia
**When** Engineering Lead membuka form perubahan dan menyimpan nama `Backend`
**Then** sistem memperlakukan nilainya sebagai `Backend`
**And** sistem tidak menghasilkan duplicate-name error terhadap record yang sedang diubah
**And** data role tetap valid.

---

### AC-9 — Menghapus role yang belum digunakan

**Given** sebuah role belum digunakan oleh team member mana pun
**When** Engineering Lead mengonfirmasi penghapusan role
**Then** sistem menghapus role tersebut
**And** role tidak lagi muncul pada daftar
**And** sistem menampilkan notifikasi bahwa role berhasil dihapus.

---

### AC-10 — Konfirmasi sebelum penghapusan

**Given** Engineering Lead memilih aksi hapus pada sebuah role
**When** dialog konfirmasi ditampilkan
**Then** sistem belum menghapus role tersebut
**And** dialog menampilkan nama role yang akan dihapus
**And** Engineering Lead dapat memilih membatalkan atau mengonfirmasi penghapusan.

---

### AC-11 — Membatalkan penghapusan

**Given** dialog konfirmasi penghapusan sedang ditampilkan
**When** Engineering Lead membatalkan penghapusan
**Then** dialog ditutup
**And** role tetap tersedia
**And** tidak ada perubahan pada database.

---

### AC-12 — Role yang digunakan tidak dapat dihapus

**Given** sebuah role sedang digunakan oleh minimal satu team member
**When** Engineering Lead mencoba menghapus role tersebut
**Then** sistem menolak penghapusan
**And** sistem mengembalikan conflict response
**And** sistem menampilkan pesan `Role is assigned to one or more team members and cannot be deleted`
**And** role dan seluruh referensinya tetap tersedia.

---

### AC-13 — Penanganan role yang tidak ditemukan

**Given** Engineering Lead mencoba mengubah atau menghapus ID role yang tidak tersedia
**When** request diproses
**Then** sistem mengembalikan not-found response
**And** sistem tidak membuat atau mengubah data apa pun.

---

### AC-14 — Konsistensi validasi frontend dan backend

**Given** input role tidak valid
**When** validasi frontend dilewati atau request dikirim langsung ke API
**Then** backend tetap menolak request menggunakan rule validasi yang sama
**And** database tidak menyimpan data tidak valid.

---

## API Contract

Endpoint dapat disesuaikan dengan konvensi repository, tetapi perilakunya harus setara dengan kontrak berikut.

### List Roles

```http
GET /api/roles
```

#### Success Response

```json
{
  "data": [
    {
      "id": "role-id",
      "name": "Backend",
      "createdAt": "2026-07-24T10:00:00Z",
      "updatedAt": "2026-07-24T10:00:00Z"
    }
  ]
}
```

Status:

```text
200 OK
```

---

### Create Role

```http
POST /api/roles
Content-Type: application/json
```

```json
{
  "name": "Backend"
}
```

Status sukses:

```text
201 Created
```

Status kegagalan:

| Condition                    |          Status |
| ---------------------------- | --------------: |
| Nama kosong atau tidak valid | 400 Bad Request |
| Nama role sudah tersedia     |    409 Conflict |

---

### Update Role

```http
PUT /api/roles/{roleId}
Content-Type: application/json
```

```json
{
  "name": "Backend Engineer"
}
```

Status sukses:

```text
200 OK
```

Status kegagalan:

| Condition                           |          Status |
| ----------------------------------- | --------------: |
| Nama tidak valid                    | 400 Bad Request |
| Role tidak ditemukan                |   404 Not Found |
| Nama role sudah digunakan role lain |    409 Conflict |

---

### Delete Role

```http
DELETE /api/roles/{roleId}
```

Status sukses:

```text
204 No Content
```

Status kegagalan:

| Condition                         |        Status |
| --------------------------------- | ------------: |
| Role tidak ditemukan              | 404 Not Found |
| Role sedang digunakan team member |  409 Conflict |

---

## Error Response

Seluruh error API menggunakan format konsisten:

```json
{
  "code": "ROLE_NAME_ALREADY_EXISTS",
  "message": "Role name already exists",
  "field": "name"
}
```

Contoh error code:

* `ROLE_NAME_REQUIRED`
* `ROLE_NAME_TOO_LONG`
* `ROLE_NAME_ALREADY_EXISTS`
* `ROLE_NOT_FOUND`
* `ROLE_IN_USE`

---

# Test Cases

## TC-1 — Menampilkan daftar role

**Precondition**

Role berikut tersedia:

* QA
* Backend
* Frontend

**Steps**

1. Buka halaman Roles.

**Expected Result**

1. Sistem menampilkan tiga role.
2. Role ditampilkan dengan urutan:

   * Backend
   * Frontend
   * QA
3. Tidak ada data yang hilang atau terduplikasi.

---

## TC-2 — Menampilkan empty state

**Precondition**

Tidak ada role dalam database.

**Steps**

1. Buka halaman Roles.

**Expected Result**

1. Sistem menampilkan empty state.
2. Tersedia tombol atau aksi `Add Role`.

---

## TC-3 — Membuat role valid

**Steps**

1. Buka form Add Role.
2. Masukkan `Backend`.
3. Simpan.

**Expected Result**

1. Request berhasil dengan status `201`.
2. Role `Backend` tersimpan.
3. Role muncul pada daftar.
4. ID, createdAt, dan updatedAt dibuat otomatis.

---

## TC-4 — Trim nama saat create

**Steps**

1. Buka form Add Role.
2. Masukkan `  Backend  `.
3. Simpan.

**Expected Result**

1. Role tersimpan sebagai `Backend`.
2. Whitespace di awal dan akhir tidak disimpan.

---

## TC-5 — Nama kosong

**Test Data**

```text
""
```

**Expected Result**

1. Penyimpanan ditolak.
2. API mengembalikan `400`.
3. Error code adalah `ROLE_NAME_REQUIRED`.
4. Tidak ada role baru dalam database.

---

## TC-6 — Nama hanya whitespace

**Test Data**

```text
"     "
```

**Expected Result**

1. Penyimpanan ditolak.
2. API mengembalikan `400`.
3. Error code adalah `ROLE_NAME_REQUIRED`.
4. Tidak ada role baru dalam database.

---

## TC-7 — Nama tepat 100 karakter

**Steps**

1. Masukkan nama role sepanjang tepat 100 karakter.
2. Simpan.

**Expected Result**

1. Role berhasil disimpan.
2. API mengembalikan `201`.

---

## TC-8 — Nama lebih dari 100 karakter

**Steps**

1. Masukkan nama role sepanjang 101 karakter.
2. Simpan.

**Expected Result**

1. Penyimpanan ditolak.
2. API mengembalikan `400`.
3. Error code adalah `ROLE_NAME_TOO_LONG`.

---

## TC-9 — Duplicate dengan huruf yang sama

**Precondition**

Role `Backend` sudah tersedia.

**Steps**

1. Tambahkan role bernama `Backend`.

**Expected Result**

1. Penyimpanan ditolak.
2. API mengembalikan `409`.
3. Error code adalah `ROLE_NAME_ALREADY_EXISTS`.
4. Jumlah role tidak berubah.

---

## TC-10 — Duplicate case-insensitive

**Precondition**

Role `Backend` sudah tersedia.

**Steps**

1. Tambahkan role bernama `backend`.

**Expected Result**

1. Penyimpanan ditolak.
2. API mengembalikan `409`.
3. Database tetap hanya memiliki satu role Backend.

---

## TC-11 — Duplicate setelah trim

**Precondition**

Role `Backend` sudah tersedia.

**Steps**

1. Tambahkan role bernama `BACKEND`.

**Expected Result**

1. Input di-trim menjadi `BACKEND`.
2. Penyimpanan ditolak sebagai duplikat.
3. API mengembalikan `409`.

---

## TC-12 — Mengubah nama role

**Precondition**

Role `Backend` tersedia.

**Steps**

1. Edit role `Backend`.
2. Ubah menjadi `Backend Engineer`.
3. Simpan.

**Expected Result**

1. API mengembalikan `200`.
2. Nama berubah menjadi `Backend Engineer`.
3. ID role tidak berubah.
4. updatedAt berubah.
5. createdAt tidak berubah.

---

## TC-13 — Update menggunakan nama record sendiri

**Precondition**

Role `Backend` tersedia.

**Steps**

1. Edit role `Backend`.
2. Masukkan `backend`.
3. Simpan.

**Expected Result**

1. Sistem tidak menganggap record tersebut duplikat terhadap dirinya sendiri.
2. Penyimpanan berhasil.
3. Nama tersimpan setelah trim sesuai normalisasi yang ditetapkan.

---

## TC-14 — Update menjadi nama role lain

**Precondition**

Role berikut tersedia:

* Backend
* Frontend

**Steps**

1. Edit role `Frontend`.
2. Ubah nama menjadi `backend`.
3. Simpan.

**Expected Result**

1. Penyimpanan ditolak.
2. API mengembalikan `409`.
3. Nama role tetap `Frontend`.

---

## TC-15 — Update role tidak ditemukan

**Steps**

1. Kirim update terhadap ID role yang tidak tersedia.

**Expected Result**

1. API mengembalikan `404`.
2. Error code adalah `ROLE_NOT_FOUND`.
3. Tidak ada role baru yang dibuat.

---

## TC-16 — Menghapus role yang belum digunakan

**Precondition**

Role `Android` tersedia dan belum digunakan team member.

**Steps**

1. Pilih Delete pada role `Android`.
2. Konfirmasi penghapusan.

**Expected Result**

1. API mengembalikan `204`.
2. Role dihapus dari database.
3. Role tidak muncul pada daftar.

---

## TC-17 — Membatalkan penghapusan

**Precondition**

Role `Android` tersedia.

**Steps**

1. Pilih Delete.
2. Pilih Cancel pada dialog konfirmasi.

**Expected Result**

1. Tidak ada request delete yang dikirim.
2. Role tetap tersedia.
3. Database tidak berubah.

---

## TC-18 — Menghapus role yang digunakan team member

**Precondition**

1. Role `Backend` tersedia.
2. Minimal satu team member menggunakan role `Backend`.

**Steps**

1. Hapus role `Backend`.
2. Konfirmasi penghapusan.

**Expected Result**

1. API mengembalikan `409`.
2. Error code adalah `ROLE_IN_USE`.
3. Role tidak terhapus.
4. Referensi team member tetap valid.

---

## TC-19 — Menghapus role tidak ditemukan

**Steps**

1. Kirim delete terhadap ID role yang tidak tersedia.

**Expected Result**

1. API mengembalikan `404`.
2. Error code adalah `ROLE_NOT_FOUND`.
3. Tidak ada data lain yang berubah.

---

## TC-20 — Concurrent create nama yang sama

**Steps**

1. Kirim dua request create role `Backend` secara bersamaan.

**Expected Result**

1. Hanya satu request yang berhasil.
2. Request lainnya mengembalikan `409`.
3. Database hanya memiliki satu role dengan normalized name `Backend`.

---

## TC-21 — Backend menolak invalid request langsung

**Steps**

1. Lewati UI.
2. Kirim request API create dengan nama kosong.
3. Kirim request API create dengan nama duplikat.
4. Kirim request API create dengan nama lebih dari 100 karakter.

**Expected Result**

1. Seluruh request tidak valid ditolak oleh backend.
2. Tidak ada data tidak valid tersimpan.
3. Status dan error code sesuai jenis kegagalan.

---

## TC-22 — Referensi team member tetap valid setelah rename

**Precondition**

1. Role `Backend` tersedia.
2. Team member Harry menggunakan role tersebut.

**Steps**

1. Ubah nama role `Backend` menjadi `Backend Engineer`.
2. Buka detail team member Harry.

**Expected Result**

1. Team member Harry tetap mereferensikan ID role yang sama.
2. Nama role yang ditampilkan berubah menjadi `Backend Engineer`.
3. Tidak terjadi orphan reference.

---

# Required Automated Tests

Codex wajib membuat automated test pada minimal tiga lapisan berikut.

## Domain / Unit Test

Menguji:

* Nama role wajib diisi.
* Nama role di-trim.
* Panjang maksimum nama.
* Rename mempertahankan ID.
* Rule normalisasi nama.

## Application / Service Test

Menguji:

* Create role.
* Update role.
* Delete unused role.
* Reject duplicate role.
* Reject deleting role in use.
* Not-found handling.
* Concurrent duplicate protection.

## API / Integration Test

Menguji:

* HTTP status.
* Request dan response body.
* Error response format.
* Database persistence.
* Unique constraint case-insensitive.
* Referential integrity terhadap team member.

## Frontend Test

Menguji:

* Render list.
* Empty state.
* Create form.
* Inline validation.
* Edit role.
* Delete confirmation.
* Cancel deletion.
* Error saat role sedang digunakan.
* Refresh daftar setelah mutation berhasil.

---

# Technical Completion Criteria

User story dianggap selesai hanya jika:

1. Database migration untuk tabel role tersedia.
2. Unique constraint atau mekanisme database-level equivalent tersedia untuk mencegah duplicate race condition.
3. Domain dan application layer tidak bergantung pada HTTP atau UI framework.
4. API endpoint create, list, update, dan delete tersedia.
5. Validasi tidak hanya dilakukan pada frontend.
6. Referential integrity terhadap team member terjaga.
7. Seluruh test case prioritas utama terotomasi.
8. Seluruh test lulus.
9. Static analysis dan lint lulus.
10. Tidak terdapat business rule di transport atau persistence layer.
11. API error memiliki code yang konsisten dan dapat dipakai frontend.
12. Dokumentasi endpoint diperbarui.
