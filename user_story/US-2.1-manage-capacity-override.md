# US-2.1 — Manage Capacity Override

## User Story

**Sebagai** Engineering Lead,
**Saya ingin** mengubah kapasitas engineer pada periode tertentu,
**Sehingga** planning menyesuaikan kondisi aktual.

Capacity Override merupakan bagian dari **Epic 2: Schedule Management**.

---

## Business Context

Daily Capacity pada Team Member merupakan kapasitas dasar. Kondisi aktual pada
periode tertentu dapat berbeda, misalnya karena perubahan ketersediaan yang
bersifat sementara.

Capacity Override menyimpan kapasitas pengganti untuk satu Team Member dalam
rentang tanggal tertentu tanpa mengubah Daily Capacity dasarnya. Data override
terbaru yang telah dikonfirmasi harus tersedia sebagai input pada eksekusi
Scheduling Engine berikutnya.

---

## UI Placement

Capacity Override diakses dari workflow **Members** untuk Team Member yang sedang
dibuka.

Rules:

- Tidak tersedia menu Capacity Override terpisah pada sidebar.
- Team Member sudah diketahui dari konteks halaman dan tidak perlu dipilih ulang.
- Form tidak menampilkan selector Team Member.
- List hanya menampilkan Capacity Override milik Team Member tersebut.
- Start Date dan End Date dipilih melalui satu field Date Range.

---

## Scope

Engineering Lead dapat:

1. Melihat daftar Capacity Override milik satu Team Member.
2. Membuka detail Capacity Override.
3. Menambahkan Capacity Override.
4. Mengubah Description, Start Date, End Date, dan Capacity.
5. Menghapus Capacity Override secara permanen.
6. Berpindah halaman pada daftar Capacity Override.

---

## Out of Scope

User story ini tidak mencakup:

- Menu sidebar khusus Capacity Override.
- Pencarian Capacity Override.
- Generic text search atau date-range filter. List hanya mendukung optional
  single-date filter `Effective Date`.
- Memindahkan Capacity Override ke Team Member lain.
- Mengubah Daily Capacity dasar Team Member.
- Mengelola Public Holiday.
- Menjalankan Scheduling Engine secara otomatis setelah mutation.
- Mendefinisikan algoritma penjadwalan atau menghitung ulang timeline pada saat
  mutation.
- Restore Capacity Override.

---

# Domain Model

## Capacity Override

| Field          | Type                        | Required | Source  | Rules                                             |
| -------------- | --------------------------- | -------: | ------- | ------------------------------------------------- |
| ID             | System-generated identifier |       Ya | System  | Dibuat otomatis dan tidak dapat diubah            |
| Team Member ID | Identifier                  |       Ya | Context | Pemilik Capacity Override dan tidak dapat diubah  |
| Description    | String                      |       Ya | User    | Alasan override, trimmed, maksimum 100 karakter   |
| Start Date     | Date                        |       Ya | User    | Date-only dan inklusif                            |
| End Date       | Date                        |       Ya | User    | Date-only, inklusif, dan tidak sebelum Start Date |
| Capacity       | Decimal hours               |       Ya | User    | `0` sampai `24`, dalam increment `0.5` jam        |
| Created At     | DateTime                    |       Ya | System  | Dibuat otomatis                                   |
| Updated At     | DateTime                    |       Ya | System  | Diperbarui otomatis                               |

Team Member ID merupakan bagian dari entity meskipun tidak ditampilkan sebagai
field yang harus dipilih pada form. Nilainya berasal dari konteks Members.

---

# Business Rules

## Description Rules

- Description wajib diisi dan menjelaskan alasan kapasitas diubah, misalnya
  cuti, training, atau support.
- Leading dan trailing whitespace di-trim sebelum persistence.
- Description tidak boleh kosong atau hanya whitespace setelah trim.
- Panjang maksimum adalah `100` karakter setelah trim, mengikuti konvensi
  master-data repository.
- Internal casing, punctuation, dan wording pengguna dipertahankan.

## Date Rules

- Start Date wajib diisi.
- End Date wajib diisi.
- Start Date dan End Date merupakan date-only values.
- Nilai API menggunakan format date-only `YYYY-MM-DD` yang valid.
- Kedua batas tanggal bersifat inklusif.
- Start Date boleh sama dengan End Date.
- End Date tidak boleh lebih awal daripada Start Date.
- Capacity Override boleh dibuat untuk tanggal lampau.
- Tidak terdapat batas tanggal masa depan.

## Date Range Interaction

- Form menggunakan satu Date Range field yang membuka calendar.
- Pilihan tanggal pertama menjadi Start Date.
- Pilihan tanggal kedua yang lebih besar dari Start Date menjadi End Date.
- Jika pilihan kedua lebih kecil dari Start Date, tanggal tersebut menggantikan
  Start Date dan calendar tetap menunggu pilihan End Date.
- Jika pilihan kedua sama dengan Start Date, Start Date dan End Date menggunakan
  tanggal yang sama sehingga range berlangsung satu hari.
- Calendar ditampilkan di bawah field ketika ruang viewport mencukupi dan
  otomatis berpindah ke atas ketika posisi bawah akan terpotong.
- Calendar hanya berpindah ke atas jika seluruh popup muat di ruang atas. Jika
  ruang atas dan bawah sama-sama tidak cukup, calendar ditampilkan ke bawah
  dengan content yang dapat di-scroll agar bagian atas popup tidak terpotong.
- Keputusan placement dan scroll dikunci selama calendar masih terbuka agar
  popup tidak berpindah saat user melakukan scroll atau memilih tanggal.
  Placement dihitung ulang setelah calendar ditutup dan dibuka kembali.
- Posisi calendar dihitung ulang ketika viewport atau posisi scroll berubah.

## Capacity Rules

- Capacity wajib diisi.
- Satuan Capacity adalah jam.
- Capacity minimum adalah `0`.
- Capacity maksimum adalah `24`.
- Capacity harus menggunakan increment `0.5` jam.
- Contoh valid: `0`, `0.5`, `6`, `7.5`, `24`.
- Contoh tidak valid: `-0.5`, `7.2`, `24.5`.

## Capacity Form Interaction

- Add dan Edit menggunakan mode form eksklusif. Selama form terbuka, list,
  Effective Date filter, pagination, Add action, dan row actions tidak
  ditampilkan atau dapat dioperasikan.
- Cancel menutup form dan mengembalikan list beserta filter dan pagination state
  sebelumnya.
- Capacity draft tidak diubah ketika pengguna masih mengetik.
- Capacity input hanya menerima digit dan maksimal satu titik (`.`) sebagai
  pemisah desimal. Huruf, koma, tanda minus, dan titik tambahan diabaikan.
- Ketika field kehilangan fokus, frontend membulatkan nilai ke kelipatan `0.5`
  terdekat.
- Normalisasi yang sama dijalankan ulang saat submit, termasuk ketika field
  belum kehilangan fokus.
- Setelah normalisasi, frontend tetap memvalidasi required value, minimum `0`,
  maksimum `24`, dan increment `0.5`.
- Jika submission gagal, nilai hasil normalisasi tetap tersedia di form.
- Backend tidak melakukan auto-correction dan tetap menolak request API yang
  tidak memenuhi increment `0.5`.

## Ownership Rules

- Setiap Capacity Override dimiliki tepat oleh satu Team Member.
- Team Member ditentukan dari parent resource pada endpoint dan konteks Members.
- Ownership tidak dapat dipindahkan melalui update.
- Update request tidak menerima `teamMemberId`.
- Request update yang mengirim `teamMemberId` ditolak sebagai invalid request.
- Periode yang sama boleh dimiliki oleh Team Member yang berbeda.

## Overlap Rules

Untuk Team Member yang sama, dua periode Capacity Override tidak boleh memiliki
tanggal yang beririsan.

Dengan periode inklusif, periode dianggap overlap ketika:

```text
existing.startDate <= candidate.endDate
AND
existing.endDate >= candidate.startDate
```

Contoh:

| Existing              | Candidate             | Result  |
| --------------------- | --------------------- | ------- |
| 2026-07-03—2026-07-04 | 2026-07-04—2026-07-05 | Overlap |
| 2026-07-03—2026-07-04 | 2026-07-05—2026-07-06 | Allowed |
| 2026-07-03—2026-07-06 | 2026-07-04—2026-07-05 | Overlap |
| 2026-07-04—2026-07-04 | 2026-07-04—2026-07-04 | Overlap |

Adjacent periods diperbolehkan ketika tidak berbagi tanggal. Saat update,
Capacity Override yang sedang diubah harus dikecualikan dari pemeriksaan overlap
terhadap dirinya sendiri.

## Delete Rules

- Delete menggunakan hard delete untuk MVP.
- Direct Delete Capacity Override tetap hard delete.
- Ketika parent Team Member dihapus, Capacity Override ikut di-soft-delete
  secara atomik sebagai bagian dari lifecycle parent, bukan melalui direct
  Delete Capacity Override.
- Delete harus dikonfirmasi oleh pengguna.
- Cancel tidak mengirim request delete.
- Setelah berhasil dihapus, Capacity Override tidak lagi dapat diambil.

---

# Capacity Resolution

Pada tanggal tertentu, urutan precedence kapasitas adalah:

1. Public Holiday → effective daily capacity `0`, meskipun terdapat Capacity
   Override pada tanggal tersebut.
2. Capacity Override.
3. Team Member Daily Capacity.

Capacity Override menggantikan Daily Capacity hanya untuk Team Member dan
tanggal yang termasuk dalam periode override. Capacity Override tidak mengubah
nilai Daily Capacity dasar yang tersimpan pada Team Member.

Create, update, dan delete Capacity Override tidak otomatis menjalankan
Scheduling Engine. Mutation yang sudah dikonfirmasi harus tersedia untuk dibaca
oleh Scheduling Engine pada eksekusi berikutnya.

---

# List Behaviour

- List selalu scoped ke satu Team Member.
- List menggunakan backend pagination.
- Default page adalah `1`.
- Default page size adalah `5`.
- Response menyediakan `page`, `pageSize`, dan `total`.
- UI menampilkan halaman aktif, navigasi yang tersedia, serta rentang item dan
  total data.
- Search tidak tersedia karena list sudah scoped ke satu Team Member dan
  Capacity Override tidak memiliki nama human-readable yang perlu dicari.
- Optional filter **Effective Date** menampilkan override dengan inclusive
  predicate `startDate <= effectiveDate AND endDate >= effectiveDate`.
- Tanpa Effective Date, list paginated normal tetap ditampilkan.
- Filter dijalankan backend dan tidak memuat dataset tanpa batas untuk difilter
  di frontend.
- Mengubah atau menghapus Effective Date mereset pagination ke page `1`.
- Effective Date dipertahankan saat berpindah page dan setelah mutation.
- Effective Date menggunakan calendar yang konsisten dengan Date Range field,
  tetapi selesai setelah satu tanggal dipilih.
- Calendar Effective Date menggunakan month navigation dan viewport-aware
  placement yang sama dengan calendar Date Range.
- Ketika filter aktif tanpa hasil, UI menampilkan no-results state yang berbeda
  dari empty state dan menyediakan Clear action.
- Initial load menampilkan skeleton atau loader lokal yang mempertahankan bentuk
  content.
- Empty state berbeda dari error state.
- Error state menjelaskan kegagalan dan menyediakan Retry.
- Mutation yang berhasil memperbarui list dan metadata pagination tanpa hard
  refresh.
- Data yang masih berguna tetap terlihat selama background refresh jika aman.
- List diurutkan berdasarkan Start Date secara ascending, kemudian End Date
  secara ascending, kemudian ID secara ascending.
- Setiap list item menampilkan Description sebagai title.
- Date range dan Capacity ditampilkan sebagai subtitle.

## Effective Date Rules

- Effective Date bersifat optional dan menggunakan date-only `YYYY-MM-DD`.
- Tanggal sama dengan Start Date atau End Date termasuk match.
- Tanggal di dalam period termasuk match; tanggal di luar period tidak match.
- Invalid Effective Date query ditolak dengan structured `400 Bad Request` dan
  code `INVALID_EFFECTIVE_DATE`.
- Filter dipertahankan selama pagination serta create, update, dan delete.
- Mutation menginvalidasi seluruh cache list filtered dan unfiltered agar stale
  response tidak dapat muncul kembali.

---

# Acceptance Criteria

## AC-1 — Akses dari workflow Members

**Given** Engineering Lead membuka Team Member pada workflow Members
**When** bagian Capacity Override dibuka
**Then** sistem menampilkan Capacity Override milik Team Member tersebut
**And** Team Member tidak perlu dipilih ulang
**And** tidak tersedia menu Capacity Override terpisah pada sidebar.

## AC-2 — Initial loading state

**Given** data Capacity Override belum selesai dimuat
**When** bagian Capacity Override pertama kali ditampilkan
**Then** sistem menampilkan skeleton atau loader lokal
**And** global application shell tetap tersedia
**And** layout tidak mengalami pergeseran yang tidak perlu.

## AC-3 — Menampilkan daftar scoped

**Given** Team Member memiliki Capacity Override
**When** list berhasil dimuat
**Then** hanya Capacity Override milik Team Member tersebut yang ditampilkan
**And** setiap item menampilkan Description sebagai title
**And** Start Date, End Date, dan Capacity dalam jam sebagai subtitle
**And** list diurutkan berdasarkan Start Date ascending, lalu End Date ascending
**And** pagination menampilkan page, visible range, dan total.

## AC-4 — Empty state

**Given** Team Member belum memiliki Capacity Override
**When** list berhasil dimuat
**Then** sistem menampilkan empty state
**And** menyediakan aksi untuk menambahkan Capacity Override.

## AC-5 — List gagal dimuat dan Retry

**Given** backend gagal memuat list
**When** response kegagalan diterima
**Then** sistem menampilkan error yang dapat dipahami tanpa detail internal
**And** menyediakan aksi Retry
**When** Retry dipilih dan request berikutnya berhasil
**Then** list ditampilkan.

## AC-6 — Pagination

**Given** Team Member memiliki lebih dari lima Capacity Override
**When** list dibuka tanpa parameter pagination
**Then** backend mengembalikan page `1` dengan page size `5`
**And** frontend dapat meminta halaman sebelumnya atau berikutnya ketika tersedia
**And** backend tidak memuat dataset tanpa batas.

## AC-7 — Membuat Capacity Override valid

**Given** Team Member tersedia
**When** Engineering Lead menyimpan Description, Start Date, End Date, dan Capacity yang valid
**Then** sistem membuat Capacity Override untuk Team Member dari konteks
**And** list yang terlihat diperbarui tanpa hard refresh
**And** sistem menampilkan konfirmasi keberhasilan
**And** Daily Capacity dasar Team Member tidak berubah.

## AC-8 — Same-day override

**Given** Team Member tersedia
**When** Start Date dan End Date menggunakan tanggal yang sama
**And** Capacity valid
**Then** sistem menerima dan menyimpan Capacity Override untuk tanggal tersebut.

**Given** calendar Date Range telah memiliki Start Date
**When** Engineering Lead memilih tanggal yang sama untuk pilihan kedua
**Then** calendar menyelesaikan range satu hari
**And** Start Date dan End Date menggunakan tanggal tersebut.

**Given** calendar Date Range telah memiliki Start Date
**When** Engineering Lead memilih tanggal kedua yang lebih awal
**Then** tanggal kedua menggantikan Start Date
**And** End Date tetap kosong
**And** calendar tetap menunggu pilihan End Date.

## AC-9 — Start Date wajib diisi

**Given** form Add atau Edit dibuka
**When** Start Date tidak diisi
**Then** penyimpanan ditolak
**And** error ditampilkan di dekat Start Date
**And** tidak ada data yang dibuat atau diubah.

## AC-10 — End Date wajib diisi

**Given** form Add atau Edit dibuka
**When** End Date tidak diisi
**Then** penyimpanan ditolak
**And** error ditampilkan di dekat End Date
**And** tidak ada data yang dibuat atau diubah.

## AC-11 — Date range harus valid

**Given** End Date lebih awal daripada Start Date
**When** form disimpan
**Then** sistem menolak penyimpanan
**And** tidak ada data yang dibuat atau diubah.

## AC-12 — Capacity wajib diisi

**Given** form Add atau Edit dibuka
**When** Capacity tidak diisi
**Then** sistem menolak penyimpanan
**And** error ditampilkan di dekat Capacity.

## AC-13 — Capacity nol diperbolehkan

**Given** periode valid
**When** Capacity diisi `0`
**Then** sistem menerima dan menyimpan Capacity Override.

## AC-14 — Capacity negatif ditolak

**Given** Engineering Lead mengisi Capacity lebih kecil dari `0`
**When** form disimpan
**Then** sistem menolak penyimpanan
**And** tidak ada data yang dibuat atau diubah.

## AC-15 — Batas maksimum Capacity

**Given** Engineering Lead mengisi Capacity `24`
**When** form disimpan
**Then** sistem menerima nilai tersebut.

**Given** Engineering Lead mengisi Capacity lebih besar dari `24`
**When** form disimpan
**Then** sistem menolak penyimpanan.

## AC-16 — Capacity menggunakan increment 0,5 jam

**Given** Engineering Lead mengisi Capacity dengan kelipatan `0.5`
**When** form disimpan
**Then** sistem menerima nilai tersebut.

**Given** Capacity bukan kelipatan `0.5`
**When** form disimpan
**Then** sistem menolak penyimpanan.

**Given** Engineering Lead sedang mengetik Capacity pada form
**Then** frontend mempertahankan draft tanpa transformasi
**When** field kehilangan fokus atau form disimpan
**Then** frontend membulatkan nilai ke kelipatan `0.5` terdekat
**And** menjalankan seluruh validasi setelah normalisasi.

## AC-17 — Overlap pada Team Member yang sama ditolak

**Given** Team Member memiliki Capacity Override pada periode tertentu
**When** create menggunakan periode yang beririsan pada minimal satu tanggal
**Then** sistem menolak create dengan conflict response
**And** existing Capacity Override tidak berubah.

## AC-18 — Adjacent periods diperbolehkan

**Given** existing period berakhir pada suatu tanggal
**When** period baru dimulai pada tanggal berikutnya tanpa tanggal yang sama
**Then** sistem menerima period baru.

## AC-19 — Periode sama untuk Team Member berbeda

**Given** Team Member A memiliki Capacity Override pada periode tertentu
**When** Capacity Override dengan periode sama dibuat untuk Team Member B
**Then** sistem menerima Capacity Override Team Member B.

## AC-20 — Mengubah Capacity Override

**Given** Capacity Override tersedia untuk Team Member
**When** Description, Start Date, End Date, atau Capacity diubah menggunakan nilai valid
**Then** sistem menyimpan perubahan
**And** ID dan Team Member ID tetap sama
**And** list diperbarui tanpa hard refresh
**And** Daily Capacity dasar Team Member tidak berubah.

## AC-21 — Update mengecualikan dirinya dari overlap check

**Given** Capacity Override tersedia
**When** data disimpan tanpa mengubah period atau dengan period valid yang tidak
beririsan dengan override lain
**Then** sistem tidak menganggap entity tersebut overlap dengan dirinya sendiri
**And** update berhasil.

## AC-22 — Update yang menyebabkan overlap ditolak

**Given** Team Member memiliki lebih dari satu Capacity Override
**When** satu override diubah sehingga beririsan dengan override lain
**Then** sistem menolak update dengan conflict response
**And** data sebelumnya tetap tersimpan utuh.

## AC-23 — Ownership tidak dapat dipindahkan

**Given** Capacity Override dimiliki Team Member A
**When** request update diproses
**Then** request body tidak menerima Team Member ID
**And** request yang tetap mengirim Team Member ID ditolak sebagai invalid request
**And** ownership tetap pada Team Member A.

## AC-24 — Parent/member mismatch

**Given** Capacity Override dimiliki Team Member A
**When** detail, update, atau delete diminta melalui path Team Member B
**Then** sistem menolak operasi
**And** Capacity Override tidak dibaca, diubah, atau dihapus
**And** sistem mengembalikan `404 Not Found` dengan code
`CAPACITY_OVERRIDE_NOT_FOUND`
**And** response tidak mengungkap data milik Team Member A.

## AC-25 — Team Member tidak ditemukan

**Given** parent Team Member ID tidak tersedia
**When** list, get, create, update, atau delete diproses
**Then** sistem mengembalikan `TEAM_MEMBER_NOT_FOUND`
**And** tidak ada mutation yang terjadi.

## AC-26 — Capacity Override tidak ditemukan

**Given** Team Member tersedia tetapi Capacity Override ID tidak tersedia
**When** get, update, atau delete diproses
**Then** sistem mengembalikan `CAPACITY_OVERRIDE_NOT_FOUND`
**And** tidak ada data lain yang berubah.

## AC-27 — Membuka konfirmasi delete

**Given** Engineering Lead memilih Delete
**When** dialog konfirmasi ditampilkan
**Then** Capacity Override belum dihapus
**And** dialog menjelaskan periode dan Capacity yang akan dihapus
**And** tersedia aksi Cancel dan Delete.

## AC-28 — Membatalkan delete

**Given** dialog konfirmasi delete ditampilkan
**When** Engineering Lead memilih Cancel
**Then** dialog ditutup
**And** tidak ada request delete yang dikirim
**And** Capacity Override tetap tersedia.

## AC-29 — Menghapus Capacity Override

**Given** dialog konfirmasi delete ditampilkan
**When** Engineering Lead mengonfirmasi Delete
**Then** sistem melakukan hard delete
**And** Capacity Override tidak lagi tersedia
**And** list dan pagination diperbarui tanpa hard refresh
**And** sistem menampilkan konfirmasi keberhasilan.

## AC-30 — Backend validation tetap berlaku

**Given** frontend validation dilewati
**When** request invalid dikirim langsung ke API
**Then** backend tetap menjalankan seluruh business validation
**And** data invalid tidak disimpan.

## AC-31 — Failed submission mempertahankan input

**Given** form berisi input pengguna
**When** submission gagal karena validation, conflict, atau dependency failure
**Then** form tetap terbuka
**And** seluruh input tetap tersedia untuk diperbaiki atau dicoba kembali
**And** error ditampilkan pada lokasi yang relevan.

## AC-32 — Duplicate submission dicegah

**Given** create, update, atau delete sedang diproses
**When** pengguna mencoba menjalankan aksi yang sama kembali
**Then** kontrol aksi tersebut dinonaktifkan selama request berjalan
**And** hanya satu mutation dikirim.

## AC-33 — Concurrent overlapping create

**Given** dua create request untuk Team Member yang sama memiliki periode overlap
**And** keduanya diproses secara concurrent
**When** transaksi selesai
**Then** maksimal satu Capacity Override yang saling overlap tersimpan
**And** request lainnya menerima conflict response
**And** invariant overlap tetap terjaga.

## AC-34 — Transaction rollback

**Given** persistence atau dependency gagal saat mutation
**When** transaksi gagal
**Then** tidak ada partial mutation yang tersimpan
**And** list tetap merepresentasikan confirmed system state
**And** pengguna dapat mencoba kembali bila aman.

## AC-35 — Capacity resolution dan scheduler consistency

**Given** Capacity Override telah dikonfirmasi
**When** Scheduling Engine dijalankan berikutnya
**Then** data override terbaru tersedia sebagai input
**And** Public Holiday memiliki precedence lebih tinggi
**And** effective daily capacity bernilai `0` pada Public Holiday meskipun
terdapat Capacity Override
**And** Capacity Override memiliki precedence lebih tinggi daripada Daily Capacity
**And** mutation tidak menjalankan Scheduling Engine secara otomatis.

## AC-36 — Accessibility

**Given** Engineering Lead menggunakan keyboard atau assistive technology
**When** mengakses list, form, pagination, Retry, dan dialog delete
**Then** seluruh kontrol dapat dioperasikan dengan keyboard
**And** label, accessible name, focus state, validation message, serta perubahan
status dapat dipahami
**And** dialog mengelola dan mengembalikan focus dengan benar.

## AC-37 — Responsive behavior

**Given** workflow dibuka pada supported screen size
**When** viewport berubah
**Then** list, form, pagination, dan primary action tetap dapat digunakan
**And** required action tidak disembunyikan
**And** normal page content tidak menyebabkan horizontal scrolling.

## AC-38 — Membuka detail Capacity Override

**Given** Capacity Override tersedia untuk Team Member yang sedang dibuka
**When** Engineering Lead membuka detail atau form Edit
**Then** sistem mengambil Capacity Override melalui parent Team Member yang benar
**And** menampilkan Start Date, End Date, dan Capacity yang tersimpan
**And** Team Member tetap berasal dari context dan tidak dapat dipilih ulang.

## AC-39 — Format date-only harus valid

**Given** request mengirim Start Date atau End Date
**When** nilainya bukan date-only `YYYY-MM-DD` yang valid
**Then** sistem menolak request dengan `400 Bad Request`
**And** error code adalah `CAPACITY_OVERRIDE_INVALID_DATE`
**And** field menunjukkan `startDate` atau `endDate` yang invalid
**And** tidak ada data yang dibuat atau diubah.

## AC-40 — Filter Effective Date secara inclusive

**Given** Capacity Override memiliki period `2026-07-26`—`2026-07-28`
**When** Effective Date adalah Start Date, `2026-07-27`, atau End Date
**Then** override termasuk dalam hasil untuk masing-masing tanggal.

**When** Effective Date sebelum Start Date atau setelah End Date
**Then** override tidak termasuk dalam hasil.

## AC-41 — Effective Date optional dan backend-filtered

**Given** Effective Date kosong
**When** list dimuat
**Then** sistem menampilkan normal paginated list.

**Given** Effective Date dipilih
**When** list dimuat
**Then** frontend mengirim Effective Date ke backend
**And** backend menerapkan inclusive predicate sebelum count, limit, dan offset
**And** frontend tidak memuat unbounded list untuk melakukan filter lokal.

## AC-42 — Perubahan dan Clear filter

**Given** Engineering Lead berada pada page selain page `1`
**When** Effective Date berubah atau dihapus
**Then** pagination kembali ke page `1`.

**Given** Effective Date terpilih
**When** Engineering Lead memilih Clear
**Then** Effective Date dikosongkan
**And** normal paginated list dimuat kembali.

## AC-43 — Filter dipertahankan

**Given** Effective Date terpilih
**When** Engineering Lead berpindah page atau mutation berhasil
**Then** Effective Date tetap terpilih
**And** list direfresh menggunakan filter yang sama
**And** tidak diperlukan hard refresh.

## AC-44 — Filtered no-results state

**Given** Effective Date terpilih dan backend telah menyelesaikan response
**When** tidak ada override yang berlaku pada tanggal tersebut
**Then** UI menampilkan no-results state yang berbeda dari normal empty state
**And** menyebutkan Effective Date secara human-readable
**And** menyediakan Clear action.

Normal empty state menjelaskan bahwa Member menggunakan base Daily Capacity
ketika belum memiliki Capacity Override.

## AC-45 — Invalid Effective Date query

**Given** query `effectiveDate` bukan date-only `YYYY-MM-DD` yang valid
**When** list endpoint dipanggil
**Then** API mengembalikan `400 Bad Request`
**And** code `INVALID_EFFECTIVE_DATE`
**And** field `effectiveDate`
**And** repository list tidak dijalankan.

## AC-46 — Loading, retry, accessibility, dan responsive filter

**Given** filtered request sedang diproses pertama kali
**Then** UI tidak menampilkan no-results sebelum response selesai.

**When** safe background refresh berlangsung
**Then** visible result tetap ditampilkan sampai confirmed response tersedia.

**And** Effective Date serta Clear action keyboard-accessible, memiliki label
visible, dapat dipakai pada supported screen sizes, dan error menyediakan Retry.
**And** Effective Date membuka calendar dengan desain dan placement behavior
yang sama dengan Date Range, lalu menutup setelah satu tanggal dipilih.

## AC-47 — Add dan Edit merupakan mode eksklusif

**Given** list Capacity Override sedang ditampilkan
**When** Engineering Lead membuka form Add atau Edit
**Then** list, Effective Date filter, pagination, Add action, dan row actions
tidak ditampilkan dan tidak dapat dioperasikan
**And** header serta Close workflow tetap tersedia.

**When** Engineering Lead memilih Cancel
**Then** form ditutup
**And** list ditampilkan kembali dengan filter dan pagination state sebelumnya.

## AC-48 — Description wajib dan dinormalisasi

**Given** form Add atau Edit dibuka
**When** Description kosong, hanya whitespace, atau lebih dari `100` karakter
setelah trim
**Then** penyimpanan ditolak dengan field error yang sesuai
**And** tidak ada data yang dibuat atau diubah.

**When** Description valid memiliki leading atau trailing whitespace
**Then** whitespace tersebut di-trim sebelum persistence
**And** internal casing, punctuation, serta wording dipertahankan.

---

# API Contract

Endpoint mengikuti nested Team Member resource.

## List Capacity Overrides

```http
GET /api/team-members/{teamMemberId}/capacity-overrides?effectiveDate=2026-07-27&page=1&pageSize=5
```

### Success Response

```json
{
  "data": [
    {
      "id": "capacity-override-id",
      "teamMemberId": "member-id",
      "description": "Training",
      "startDate": "2026-07-03",
      "endDate": "2026-07-04",
      "capacity": 4,
      "createdAt": "2026-07-26T10:00:00Z",
      "updatedAt": "2026-07-26T10:00:00Z"
    }
  ],
  "page": 1,
  "pageSize": 5,
  "total": 1
}
```

Status sukses: `200 OK`.

Pagination rules:

- `page` default `1` dan harus lebih besar dari `0`.
- `pageSize` default `5` dan harus berada pada range `1` sampai `100` sesuai
  query convention repository.
- `effectiveDate` optional. Jika ada, nilainya harus berupa date-only
  `YYYY-MM-DD` dan filtering dilakukan sebelum count serta pagination.

## Get Capacity Override

```http
GET /api/team-members/{teamMemberId}/capacity-overrides/{capacityOverrideId}
```

### Success Response

```json
{
  "data": {
    "id": "capacity-override-id",
    "teamMemberId": "member-id",
    "description": "Training",
    "startDate": "2026-07-03",
    "endDate": "2026-07-04",
    "capacity": 4,
    "createdAt": "2026-07-26T10:00:00Z",
    "updatedAt": "2026-07-26T10:00:00Z"
  }
}
```

Status sukses: `200 OK`.

## Create Capacity Override

```http
POST /api/team-members/{teamMemberId}/capacity-overrides
Content-Type: application/json
```

```json
{
  "description": "Training",
  "startDate": "2026-07-03",
  "endDate": "2026-07-04",
  "capacity": 4
}
```

Status sukses: `201 Created`.

Response menggunakan bentuk Capacity Override yang sama dengan Get.

## Update Capacity Override

```http
PUT /api/team-members/{teamMemberId}/capacity-overrides/{capacityOverrideId}
Content-Type: application/json
```

```json
{
  "description": "Production support",
  "startDate": "2026-07-05",
  "endDate": "2026-07-06",
  "capacity": 6.5
}
```

Request body tidak menerima `teamMemberId`. Jika field tersebut dikirim, API
menolak request dengan `400 Bad Request` dan code `INVALID_REQUEST`.

Status sukses: `200 OK`.

Response menggunakan bentuk Capacity Override yang sama dengan Get.

## Delete Capacity Override

```http
DELETE /api/team-members/{teamMemberId}/capacity-overrides/{capacityOverrideId}
```

Status sukses: `204 No Content`.

## Status Kegagalan

| Condition                           |                    Status |
| ----------------------------------- | ------------------------: |
| Input atau pagination tidak valid   |           400 Bad Request |
| Team Member tidak ditemukan         |             404 Not Found |
| Capacity Override tidak ditemukan   |             404 Not Found |
| Periode overlap                     |              409 Conflict |
| Parent/member mismatch              |             404 Not Found |
| Persistence atau dependency failure | 500 Internal Server Error |

---

# Error Response

Seluruh API error menggunakan format konsisten:

```json
{
  "code": "CAPACITY_OVERRIDE_CAPACITY_INVALID_INCREMENT",
  "message": "Capacity must use 0.5-hour increments",
  "field": "capacity"
}
```

| Error Code                                     | Field                      | Condition                                      |
| ---------------------------------------------- | -------------------------- | ---------------------------------------------- |
| `CAPACITY_OVERRIDE_START_DATE_REQUIRED`        | `startDate`                | Start Date tidak diisi                         |
| `CAPACITY_OVERRIDE_END_DATE_REQUIRED`          | `endDate`                  | End Date tidak diisi                           |
| `CAPACITY_OVERRIDE_INVALID_DATE`               | `startDate` atau `endDate` | Nilai bukan date-only `YYYY-MM-DD` yang valid  |
| `CAPACITY_OVERRIDE_INVALID_DATE_RANGE`         | `endDate`                  | End Date sebelum Start Date                    |
| `CAPACITY_OVERRIDE_CAPACITY_REQUIRED`          | `capacity`                 | Capacity tidak diisi                           |
| `CAPACITY_OVERRIDE_CAPACITY_NEGATIVE`          | `capacity`                 | Capacity lebih kecil dari `0`                  |
| `CAPACITY_OVERRIDE_CAPACITY_EXCEEDS_LIMIT`     | `capacity`                 | Capacity lebih besar dari `24`                 |
| `CAPACITY_OVERRIDE_CAPACITY_INVALID_INCREMENT` | `capacity`                 | Capacity bukan kelipatan `0.5`                 |
| `CAPACITY_OVERRIDE_DESCRIPTION_REQUIRED`       | `description`              | Description kosong setelah trim                |
| `CAPACITY_OVERRIDE_DESCRIPTION_TOO_LONG`       | `description`              | Description melebihi 100 karakter              |
| `CAPACITY_OVERRIDE_OVERLAPS`                   | —                          | Periode overlap untuk Team Member sama         |
| `CAPACITY_OVERRIDE_NOT_FOUND`                  | —                          | Capacity Override tidak ditemukan              |
| `TEAM_MEMBER_NOT_FOUND`                        | —                          | Parent Team Member tidak ditemukan             |
| `INVALID_PAGE`                                 | `page`                     | Page tidak valid                               |
| `INVALID_PAGE_SIZE`                            | `pageSize`                 | Page size tidak valid                          |
| `INVALID_REQUEST`                              | —                          | Update mengirim field yang tidak diterima      |
| `INVALID_EFFECTIVE_DATE`                       | `effectiveDate`            | Filter bukan date-only `YYYY-MM-DD` yang valid |

Error internal tidak boleh mengekspos stack trace, query, detail database, atau
infrastructure. Frontend memetakan error menjadi pesan yang dapat dipahami dan
menampilkan Retry ketika operasi aman untuk dicoba kembali.

---

# Test Cases

## TC-1 — Akses Capacity Override dari Members

1. Buka Team Member A dari workflow Members.
2. Buka bagian Capacity Override.
3. Verifikasi Team Member tidak perlu dipilih dan tidak ada sidebar menu khusus.

**Expected:** Hanya workflow Capacity Override Team Member A yang ditampilkan.

## TC-2 — Initial loading state

1. Tunda response list.
2. Buka bagian Capacity Override.

**Expected:** Skeleton atau loader lokal tampil tanpa meremount global shell atau
menyebabkan layout shift yang tidak perlu.

## TC-3 — List scoped ke Team Member

1. Siapkan override untuk Team Member A dan B.
2. Minta list Team Member A.

**Expected:** Response `200` hanya memuat data A, diurutkan berdasarkan Start Date
ascending lalu End Date ascending, beserta metadata pagination.

## TC-4 — Empty state

1. Buka Team Member tanpa Capacity Override.

**Expected:** Empty state dan aksi Add ditampilkan.

## TC-5 — List failure dan Retry

1. Buat request list pertama gagal.
2. Pilih Retry dan buat request kedua berhasil.

**Expected:** Error yang dapat dipahami tampil, lalu digantikan list setelah Retry.

## TC-6 — Pagination default dan navigation

1. Siapkan lebih dari lima override.
2. Request list tanpa page dan pageSize.
3. Navigasi ke page berikutnya.

**Expected:** Page pertama memakai page size `5`; navigation meminta page yang
benar dan metadata tetap konsisten.

## TC-7 — Pagination invalid

Uji `page=0`, `pageSize=0`, dan `pageSize=101`.

**Expected:** API mengembalikan `400` dengan `INVALID_PAGE` atau
`INVALID_PAGE_SIZE` sesuai field tanpa menjalankan query list invalid.

## TC-8 — Create valid

1. Buat override `2026-07-03` sampai `2026-07-04`, Capacity `4`.

**Expected:** Response `201`; ownership mengikuti path Team Member; list terbarui
tanpa hard refresh; Daily Capacity dasar tidak berubah.

## TC-9 — Same-day override

1. Buat Start Date dan End Date `2026-07-03` dengan Capacity `4`.

**Expected:** Request berhasil.

## TC-9A — Memilih Date Range melalui calendar

1. Buka form Add Capacity Override.
2. Buka field Date Range.
3. Pilih `2026-07-04` sebagai pilihan pertama.
4. Pilih `2026-07-03` sebagai pilihan kedua.
5. Pilih `2026-07-03` sekali lagi.

**Expected:** Setelah langkah 4, Start Date berubah menjadi `2026-07-03` dan
calendar tetap menunggu End Date. Setelah langkah 5, calendar menyelesaikan
range satu hari dengan Start Date dan End Date `2026-07-03`.

## TC-10 — Start Date required

1. Kirim create dan update tanpa Start Date.

**Expected:** Response `400`, code `CAPACITY_OVERRIDE_START_DATE_REQUIRED`, tanpa
mutation.

## TC-11 — End Date required

1. Kirim create dan update tanpa End Date.

**Expected:** Response `400`, code `CAPACITY_OVERRIDE_END_DATE_REQUIRED`, tanpa
mutation.

## TC-12 — Invalid date range

1. Kirim Start Date `2026-07-04` dan End Date `2026-07-03`.

**Expected:** Response `400`, code `CAPACITY_OVERRIDE_INVALID_DATE_RANGE`.

## TC-13 — Capacity required

1. Kirim create dan update tanpa Capacity.

**Expected:** Response `400`, code `CAPACITY_OVERRIDE_CAPACITY_REQUIRED`.

## TC-14 — Capacity zero

1. Buat override dengan Capacity `0`.

**Expected:** Request berhasil dan nilai `0` tersimpan, bukan dianggap missing.

## TC-15 — Capacity negative

1. Kirim Capacity `-0.5`.

**Expected:** Response `400`, code `CAPACITY_OVERRIDE_CAPACITY_NEGATIVE`.

## TC-16 — Capacity boundary 24

1. Kirim Capacity `24`.

**Expected:** Request berhasil dan nilai tersimpan presisi.

## TC-17 — Capacity exceeds limit

1. Kirim Capacity `24.5`.

**Expected:** Response `400`, code
`CAPACITY_OVERRIDE_CAPACITY_EXCEEDS_LIMIT`.

## TC-18 — Capacity increment

1. Uji nilai valid `0.5`, `6`, dan `7.5`.
2. Uji nilai invalid `7.2`.

**Expected:** Nilai valid diterima; nilai invalid menerima `400` dengan
`CAPACITY_OVERRIDE_CAPACITY_INVALID_INCREMENT`.

## TC-19 — Overlap pada boundary inklusif

1. Simpan existing period `2026-07-03`—`2026-07-04`.
2. Buat candidate `2026-07-04`—`2026-07-05`.

**Expected:** Response `409`, code `CAPACITY_OVERRIDE_OVERLAPS`, existing data
tidak berubah.

## TC-20 — Enclosed dan enclosing overlap

1. Uji candidate di dalam existing period.
2. Uji candidate yang melingkupi existing period.

**Expected:** Kedua request ditolak dengan `CAPACITY_OVERRIDE_OVERLAPS`.

## TC-21 — Adjacent periods

1. Simpan existing period yang berakhir `2026-07-04`.
2. Buat candidate yang dimulai `2026-07-05`.

**Expected:** Request berhasil.

## TC-22 — Periode sama untuk Team Member berbeda

1. Simpan period untuk Team Member A.
2. Buat period identik untuk Team Member B.

**Expected:** Kedua data dapat tersimpan.

## TC-23 — Update success

1. Ubah period dan Capacity menggunakan nilai valid.

**Expected:** Response `200`; ID dan Team Member ID tetap; list terbarui; Daily
Capacity dasar tidak berubah.

## TC-24 — Update self-overlap exclusion

1. Simpan ulang override tanpa mengubah period.

**Expected:** Update berhasil dan tidak overlap dengan dirinya sendiri.

## TC-25 — Update overlap dengan entity lain

1. Siapkan dua period terpisah.
2. Ubah satu period agar overlap dengan period lainnya.

**Expected:** Response `409`; data sebelum update tetap utuh.

## TC-26 — Update tidak memindahkan ownership

1. Periksa bahwa request update tidak memiliki `teamMemberId`.
2. Update Capacity Override Team Member A.
3. Kirim request update lain yang tetap memuat `teamMemberId`.

**Expected:** Request valid mempertahankan Team Member ID A; request yang memuat
`teamMemberId` ditolak dengan `400 INVALID_REQUEST`.

## TC-27 — Parent/member mismatch

1. Gunakan ID override Team Member A melalui path Team Member B untuk get,
   update, dan delete.

**Expected:** Semua operasi menerima `404 CAPACITY_OVERRIDE_NOT_FOUND`; data A
tidak terungkap atau berubah.

## TC-28 — Parent Team Member tidak ditemukan

1. Uji list, get, create, update, dan delete dengan Team Member ID yang tidak ada.

**Expected:** Response `404`, code `TEAM_MEMBER_NOT_FOUND`, tanpa mutation.

## TC-29 — Capacity Override tidak ditemukan

1. Uji get, update, dan delete dengan override ID yang tidak ada pada parent valid.

**Expected:** Response `404`, code `CAPACITY_OVERRIDE_NOT_FOUND`.

## TC-30 — Delete confirmation

1. Pilih Delete.

**Expected:** Tidak ada request delete; dialog menampilkan period, Capacity,
Cancel, dan Delete; focus dikelola di dalam dialog.

## TC-31 — Delete cancellation

1. Pilih Delete lalu Cancel.

**Expected:** Dialog ditutup, focus kembali, tidak ada request delete, data tetap
tersedia.

## TC-32 — Delete success

1. Konfirmasi Delete.

**Expected:** Response `204`; data di-hard-delete; get berikutnya menghasilkan
not found; list dan pagination terbarui tanpa hard refresh.

## TC-33 — Backend validation langsung

1. Lewati frontend dan kirim seluruh kombinasi input invalid langsung ke API.

**Expected:** Backend menolak dengan status, code, dan field yang sesuai; tidak
ada data invalid tersimpan.

## TC-34 — Failed submission mempertahankan input

1. Isi form.
2. Buat create atau update gagal karena validation, overlap, dan dependency
   failure secara terpisah.

**Expected:** Draft tetap tersedia dan error actionable ditampilkan.

## TC-35 — Duplicate submission prevention

1. Klik Save atau Delete berulang ketika request pertama masih berjalan.

**Expected:** Hanya satu mutation dikirim; hanya kontrol terkait yang disabled.

## TC-36 — Concurrent overlapping create

1. Kirim dua create concurrent untuk parent yang sama dengan period overlap.

**Expected:** Maksimal satu berhasil; lainnya menerima `409 Conflict` dengan code
`CAPACITY_OVERRIDE_OVERLAPS`; repository tidak menyimpan dua period overlap.

## TC-37 — Transaction rollback

1. Simulasikan persistence/dependency failure setelah mutation dimulai.

**Expected:** Tidak ada partial data; state sebelum transaksi tetap utuh; error
tidak mengekspos detail internal.

## TC-38 — Cache invalidation atau state refresh setelah mutation

1. Muat list dan simpan sebagai cached state.
2. Create, update, lalu delete secara terpisah.

**Expected:** Visible list dan metadata pagination mengikuti confirmed server
state tanpa hard refresh atau stale data.

## TC-39 — Capacity resolution dan scheduler input

1. Siapkan Daily Capacity dan Capacity Override berbeda.
2. Verifikasi override tidak mengubah Daily Capacity.
3. Ambil input pada Scheduling Engine berikutnya.
4. Verifikasi precedence ketika Public Holiday juga berlaku.

**Expected:** Data terbaru tersedia; Public Holiday menghasilkan effective daily
capacity `0` meskipun terdapat override; jika bukan Public Holiday, precedence
Capacity Override lalu Daily Capacity diterapkan; mutation tidak menjalankan
scheduler otomatis.

## TC-40 — Accessibility

1. Operasikan workflow dengan keyboard dan assistive technology.

**Expected:** Focus order, labels, errors, status, pagination, Retry, dan dialog
dapat dipahami dan dioperasikan.

## TC-41 — Responsive behavior

1. Uji supported desktop dan mobile viewport.

**Expected:** List, form, pagination, dan seluruh required action tetap usable
tanpa horizontal scrolling pada normal page content.

## TC-42 — Get detail success

1. Siapkan Capacity Override untuk Team Member A.
2. Buka detail atau form Edit melalui Team Member A.

**Expected:** Response `200` memuat ID, Team Member ID, Start Date, End Date, dan
Capacity yang benar; UI tidak menampilkan selector Team Member.

## TC-43 — Invalid date-only value

1. Kirim Start Date `2026-02-30`.
2. Kirim End Date `26-07-2026`.
3. Kirim nilai DateTime `2026-07-26T10:00:00Z` sebagai Start Date.

**Expected:** Setiap request menerima `400 Bad Request` dengan code
`CAPACITY_OVERRIDE_INVALID_DATE`; field menunjukkan input yang invalid; tidak ada
mutation yang tersimpan.

## TC-44 — Effective Date inclusive boundaries dan inside period

1. Simpan override `2026-07-26`—`2026-07-28`.
2. Filter secara terpisah menggunakan `2026-07-26`, `2026-07-27`, dan
   `2026-07-28`.

**Expected:** Override muncul pada ketiga hasil dengan metadata filtered yang
benar.

## TC-45 — Effective Date outside period

1. Filter menggunakan tanggal sebelum Start Date.
2. Filter menggunakan tanggal setelah End Date.

**Expected:** Override tidak muncul pada kedua hasil.

## TC-46 — Effective Date scoped, paginated, dan deterministik

1. Siapkan matching override untuk beberapa Team Member.
2. Request filtered page dengan page size tertentu.

**Expected:** Hanya data parent Member yang muncul; total, limit, dan offset
mengikuti filtered dataset; urutan Start Date, End Date, lalu ID ascending.

## TC-47 — Filter control, no-results, dan Clear

1. Pilih Effective Date yang tidak memiliki matching override.
2. Tunggu backend response selesai.
3. Pilih Clear.

**Expected:** Request page `1` memuat `effectiveDate`; no-results menyebut tanggal
dan menyediakan Clear; Clear meminta unfiltered page `1` dan normal list kembali.

## TC-48 — Filter preservation dan cache invalidation

1. Pilih Effective Date.
2. Berpindah page.
3. Jalankan create, update, dan delete secara terpisah.

**Expected:** Effective Date tetap terpilih dan dikirim pada pagination serta
refresh; seluruh filtered/unfiltered cache invalid; stale in-flight response
tidak merepopulasi list.

## TC-49 — Invalid Effective Date API query

1. Request list menggunakan `effectiveDate=2026-02-30`.

**Expected:** Response `400`, code `INVALID_EFFECTIVE_DATE`, field
`effectiveDate`, dan repository tidak dipanggil.

## TC-50 — Filter loading, retry, accessibility, dan responsive behavior

1. Tunda filtered response, lalu simulasikan failure dan Retry.
2. Operasikan filter serta Clear dengan keyboard pada desktop dan mobile.

**Expected:** Loading tampil tanpa premature no-results; existing result tetap
terlihat saat safe refresh; Retry berhasil; visible label dan controls tetap
accessible serta usable.

## TC-51 — Exclusive Add dan Edit mode

1. Buka form Add dari list yang memiliki filter atau pagination state.
2. Verifikasi list controls dan row actions tidak tersedia, lalu pilih Cancel.
3. Buka form Edit dan ulangi verifikasi tersebut, lalu pilih Cancel.

**Expected:** Selama Add atau Edit, hanya form workflow yang dapat dioperasikan;
Cancel mengembalikan list dengan filter dan pagination state sebelumnya.

## TC-52 — Description validation dan presentation

1. Uji Description kosong, whitespace-only, 100 karakter, dan 101 karakter.
2. Simpan `  Training  ` dan buka list serta Edit.

**Expected:** Nilai kosong/whitespace dan 101 karakter ditolak; 100 karakter
diterima; outer whitespace di-trim; list menampilkan `Training` sebagai title
serta date range dan Capacity sebagai subtitle; Edit memuat Description terbaru.

## Acceptance Criteria Traceability

| Acceptance Criteria | Corresponding Test Case(s) |
| ------------------- | -------------------------- |
| AC-1—AC-5           | TC-1—TC-5                  |
| AC-6                | TC-6, TC-7                 |
| AC-7—AC-16          | TC-8—TC-18                 |
| AC-17               | TC-19, TC-20               |
| AC-18—AC-32         | TC-21—TC-35                |
| AC-33               | TC-36                      |
| AC-34               | TC-37                      |
| AC-35               | TC-39                      |
| AC-36               | TC-40                      |
| AC-37               | TC-41                      |
| AC-38               | TC-42                      |
| AC-39               | TC-43                      |
| AC-40               | TC-44, TC-45               |
| AC-41               | TC-44—TC-46                |
| AC-42               | TC-47                      |
| AC-43               | TC-48                      |
| AC-44               | TC-47                      |
| AC-45               | TC-49                      |
| AC-46               | TC-50                      |
| AC-47               | TC-51                      |
| AC-48               | TC-52                      |

TC-38 memverifikasi state consistency lintas create, update, dan delete yang
diwajibkan oleh AC-7, AC-20, serta AC-29.

---

# Required Automated Tests

## Domain Unit Tests

Wajib menguji:

- Required Start Date dan End Date.
- Required, trim, dan batas panjang Description `100` karakter.
- Validasi date-only `YYYY-MM-DD`, termasuk tanggal kalender yang tidak valid.
- Inclusive date range dan same-day period.
- End Date sebelum Start Date.
- Capacity `0`, nilai negatif, batas `24`, lebih dari `24`, dan increment `0.5`.
- Overlap pada awal, akhir, enclosed, enclosing, dan same-day period.
- Adjacent periods tidak dianggap overlap.
- Update overlap check mengecualikan entity sendiri.
- Ownership tidak berubah saat update.
- Capacity Override tidak mengubah Daily Capacity dasar.

## Application Tests

Wajib menguji:

- List dan get scoped berdasarkan Team Member.
- Create, update, dan delete.
- Team Member not found dan Capacity Override not found.
- Parent/member mismatch tanpa mengungkap atau mengubah entity lain.
- Reject overlapping create dan update.
- Same period untuk Team Member berbeda.
- Ownership tidak dapat dipindahkan.
- Transaction rollback pada dependency failure.
- Mutation tidak menjalankan Scheduling Engine.
- Data terbaru tersedia untuk eksekusi Scheduling Engine berikutnya.
- Optional Effective Date diteruskan tanpa mengubah inclusive semantics.

Application unit tests menggunakan consumer-oriented repository test double dan
tidak bergantung pada transport atau framework database.

## Repository Integration Tests

Wajib menguji:

- Persist dan retrieve Capacity Override.
- Team Member relationship dan scoped query.
- Date-only dan decimal Capacity tersimpan secara presisi.
- Created At dibuat otomatis dan tetap saat update; Updated At diperbarui saat
  update.
- Backend pagination serta total matching records.
- Overlap detection untuk Team Member yang sama.
- Period sama untuk Team Member berbeda.
- Update self-exclusion.
- Hard delete.
- Concurrent overlapping create menjaga invariant.
- Transaction rollback.
- Query dan index strategy untuk parent-scoped list, overlap check, dan ordering
  setelah ordering diputuskan.
- Exact Start Date, inside period, exact End Date, before/after exclusion,
  parent scope, filtered count/limit/offset, dan deterministic ordering.
- Perilaku yang setara pada PostgreSQL dan MySQL sesuai database rules project.

## API Integration Tests

Wajib menguji:

- Seluruh nested HTTP method dan path.
- Request dan response field mapping.
- Required, trimmed, dan maximum-length Description.
- Date-only serialization.
- Invalid date-only value menghasilkan `CAPACITY_OVERRIDE_INVALID_DATE` dan field
  yang tepat.
- Success status `200`, `201`, dan `204`.
- Seluruh minimum error code dan HTTP status.
- Update body tidak menerima atau mengubah Team Member ID.
- Parent/member mismatch.
- Backend validation ketika frontend dilewati.
- Pagination default `5`, metadata, invalid page, dan invalid page size.
- Valid dan invalid Effective Date query serta filtered pagination metadata.
- Overlap conflict dan not-found behavior.
- Concurrent overlapping create.
- Database persistence dan rollback.

## Frontend Tests

Wajib menguji:

- Capacity Override diakses dari Members tanpa standalone sidebar menu.
- Initial skeleton atau loader lokal.
- Scoped list success.
- Description sebagai list title serta Date dan Capacity sebagai subtitle.
- Empty state dan Add action.
- List error dan Retry.
- Pagination default, visible range, serta previous/next navigation.
- Tidak menampilkan search.
- Create dan same-day override.
- Satu Date Range field membuka calendar dan memilih Start Date serta End Date
  melalui dua pilihan tanggal.
- Pilihan kedua yang lebih awal menggantikan Start Date dan tetap menunggu End
  Date.
- Pilihan kedua yang sama menghasilkan range satu hari.
- Calendar berpindah ke atas ketika ruang viewport di bawah field tidak cukup.
- Seluruh field validation dan boundary Capacity.
- Description required, trim, maximum length, dan draft preservation.
- Capacity draft tidak berubah saat diketik, dibulatkan ke increment `0.5` saat
  blur, dan dinormalisasi ulang saat submit.
- Capacity input menolak karakter selain digit dan satu titik desimal tanpa
  menghapus draft valid yang sudah diketik.
- Overlap error.
- Edit form dan ownership tetap berasal dari context.
- Add dan Edit menyembunyikan list, filter, pagination, Add action, serta row
  actions; Cancel memulihkan list state sebelumnya.
- Delete confirmation, cancellation, dan success.
- Not-found dan backend validation error mapping.
- Failed submission mempertahankan draft.
- Duplicate submission prevention.
- Cache invalidation atau equivalent refresh setelah create, update, dan delete.
- Loading feedback untuk setiap mutation.
- Keyboard interaction, focus management, accessible labels, dan dynamic status.
- Responsive behavior pada supported screen sizes.
- Effective Date control, page reset, preservation selama pagination/mutation,
  no-results, Clear, filtered retry, dan filtered cache invalidation.
- Effective Date menggunakan shared calendar pattern, memilih satu tanggal, dan
  mempertahankan viewport-aware placement yang sama dengan Date Range.

---

# Technical Completion Criteria

User story dianggap selesai jika:

1. Persistence Capacity Override dan relationship ke Team Member tersedia,
   termasuk Created At dan Updated At.
2. Date-only Start Date dan End Date tersimpan tanpa perubahan tanggal akibat
   timezone conversion.
3. Capacity disimpan sebagai decimal presisi dan bukan floating-point binary yang
   berisiko menghasilkan validation atau rounding error.
4. Invariant date range, Capacity, ownership, dan non-overlap berada pada business
   boundary yang sesuai dan tetap divalidasi backend.
5. Concurrent write strategy menjaga agar period overlap untuk Team Member yang
   sama tidak dapat tersimpan.
6. Create dan update bersifat transactional dan rollback tidak meninggalkan
   partial state.
7. List, get, create, update, dan hard delete tersedia melalui nested Team Member
   API.
8. List menggunakan backend pagination dengan default page size `5` dan response
   metadata `page`, `pageSize`, serta `total`.
9. Query parent-scoped list dan overlap check memiliki index strategy yang sesuai
   untuk PostgreSQL dan MySQL tanpa raw runtime query; list diurutkan berdasarkan
   Start Date ascending, End Date ascending, lalu ID ascending.
10. Domain dan application tidak bergantung pada HTTP, GORM, atau framework UI.
11. Transport hanya melakukan request validation, mapping, dan error mapping;
    business rule tidak berada pada handler.
12. Persistence model tidak diekspos sebagai domain model atau API DTO ketika
    responsibilities berbeda.
13. Frontend mengikuti feature boundary dan tidak memanggil HTTP langsung dari
    presentation ketika use case tersedia.
14. Mutation memperbarui visible list dan pagination tanpa hard refresh serta
    tidak meninggalkan stale data.
15. Failed form mempertahankan input dan duplicate submission dicegah.
16. Loading, empty, error, retry, success, accessibility, dan responsive states
    telah diimplementasikan dan diuji.
17. Mutation tidak menjalankan Scheduling Engine secara otomatis.
18. Data override terbaru tersedia bagi Scheduling Engine pada eksekusi berikutnya.
19. Public Holiday menghasilkan effective daily capacity `0` dan memiliki
    precedence di atas Capacity Override; Capacity Override memiliki precedence
    di atas Team Member Daily Capacity.
20. Seluruh mandatory automated tests tersedia dan lulus.
21. Go format, vet, test, race test, frontend test, lint, dan build lulus.
22. Dokumentasi API dan architecture diperbarui bila implementasi mengubah kontrak
    atau structural decision.
23. Optional Effective Date diterapkan backend sebelum count dan pagination
    menggunakan inclusive period predicate.
24. Effective Date menjadi bagian dari frontend request-cache identity; mutation
    menginvalidasi seluruh filtered dan unfiltered list cache.
25. Unfiltered list tetap backward compatible dan filtered no-results tidak
    menggantikan normal empty state.
26. Description tersimpan sebagai required `VARCHAR(100)`, divalidasi pada
    domain/backend boundary, dan dimapping melalui API serta frontend feature.
27. Existing Capacity Override mendapat migration fallback yang aman sebelum
    constraint `NOT NULL` diterapkan.
