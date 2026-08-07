# US-4.4 — Configure Group Scheduling and Lock Scope

> **Authority and supersession:** This story is the authoritative requirement
> for Group-scoped scheduling configuration and Group Lock/Reopen. It
> supersedes wording in US-3.1, US-3.3, US-4.1, US-4.2, US-4.3, US-5.1, US-6.1,
> US-6.2, US-6.3, US-6.5, and US-7.1 that assumes Automatic Scheduling, Scheduling
> Start Date, or planning lock eligibility can only be resolved from the owning
> Project.
>
> Project Priority, Project Buffer, Member capacity/buffer, Capacity Override,
> Public Holiday, Task Capacity Allocation Percentage, Manual Dependency,
> Actual Allocation, and the concrete allocation algorithms remain owned by
> their existing stories. This story does not create Group Priority, Group
> Buffer, or a separate Group capacity pool.

## 1. User Story

**Sebagai** Engineering Lead,
**Saya ingin** sebuah Group dapat memakai scheduling configuration dan
Lock/Reopen yang terpisah dari Project,
**Sehingga** saya dapat mengelola satu subtree WBS secara independen tanpa
memaksa seluruh Project memakai mode scheduling atau lifecycle yang sama.

---

## 2. Product Decision

Project tetap menyediakan default scheduling configuration, tetapi bukan lagi
satu-satunya effective configuration untuk seluruh Task.

Target resolution:

```text
Task Effective Scheduling Configuration
= nearest ancestor Group scheduling override
  else inherited ancestor Group effective configuration
  else Project scheduling configuration
```

Target lifecycle protection:

```text
Task Effective Lifecycle
= Closed, if owning Project is Closed
  else Locked, if owning Project is Locked
  else Locked, if any ancestor Group is locally Locked
  else Open
```

Group configuration berlaku terhadap seluruh descendant Task sampai sebuah
nested Group membentuk scheduling override baru. Override resolution berlaku
per field: Automatic Scheduling selalu custom pada source=`override`, sedangkan
Scheduling Start Date hanya custom bila non-null; null tetap mengikuti parent
chain. Parent Project/Group lifecycle protection selalu merupakan hard ceiling
dan tidak dapat dibypass oleh child.

---

## 3. Scope

### 3.1 In Scope

- Group scheduling source: `inherit` atau `override`.
- Group override untuk:
  - Automatic Scheduling;
  - Scheduling Start Date, dengan `null` sebagai field-level inheritance ke
    parent Scheduling Start Date.
- Group local lifecycle:
  - Open;
  - Locked;
  - explicit Reopen (`Locked → Open`).
- Nested Group inheritance dan nearest-ancestor resolution.
- Effective Task editability dan scheduler activation berdasarkan resolved Group
  configuration/lifecycle.
- Group Lock baseline protection dan Actual Date factual exception.
- Group-aware scheduling-impact warning/blocking, including impact terhadap
  sibling Group di Project yang sama.
- Group-aware Reopen closure bila protected Group/Project lain harus ikut
  dibuka.
- Move/reorder/create/conversion behaviour ketika effective configuration
  berubah.
- Existing Home Group dialog composition, validation, rollback, concurrency,
  stale-response protection, accessibility, and Three-Level Confidence.

### 3.2 Out of Scope

- Group Priority.
- Group Buffer; Project Buffer tetap Project-level dan berlaku pada seluruh Task
  Project sesuai US-3.3/US-6.1.
- Group-specific Team Member capacity, Capacity Override, Public Holiday, atau
  capacity pool.
- Group Closed status atau historical Group archive.
- Per-Task override untuk Automatic Scheduling atau Scheduling Start Date.
- Cross-Project WBS move.
- Per-Group Forecast, Health, atau Delivery Impact redesign.
- Per-Group permission model.
- Mengubah manual-only dependency contract US-6.4.

---

## 4. Terminology

| Term | Meaning |
| --- | --- |
| Group | Grouping WBS / non-leaf WBS node |
| Scheduling Source | `inherit` atau `override`; `override` selalu meng-customize Automatic Scheduling, sedangkan Scheduling Start Date hanya di-customize bila non-null |
| Inherited Scheduling Configuration | Effective scheduling configuration dari parent Group terdekat, atau Project bila tidak ada parent Group |
| Group Scheduling Override | Group-owned Automatic Scheduling plus optional non-null Scheduling Start Date; null Start Date tetap resolve ke parent chain |
| Effective Scheduling Configuration | Automatic Scheduling + Scheduling Start Date yang benar-benar dipakai sebuah Task setelah inheritance resolution |
| Local Group Status | Persisted `open` atau `locked` milik Group itu sendiri |
| Effective Lifecycle | Open/Locked/Closed state yang benar-benar berlaku pada Task/Group setelah Project dan ancestor lock precedence |
| Scheduling Scope Owner | Project atau Group boundary yang menyediakan effective scheduling override/lifecycle protection untuk sekumpulan Task |
| Protected Group Baseline | Confirmed Execution/Commitment dates and allocations descendant Task yang tidak boleh dimutasi scheduler selama Group effectively Locked |

---

## 5. Domain Model

### 5.1 Group Scheduling Configuration

Grouping WBS menambahkan scheduling configuration berikut:

| Field | Type | Required | Default | Rules |
| --- | --- | ---: | --- | --- |
| Scheduling Source | `inherit \| override` | Ya | `inherit` | Menentukan apakah Group mengikuti parent effective values atau memakai values sendiri |
| Automatic Scheduling Override | Nullable Boolean | Conditional | `null` | Wajib bernilai ketika source=`override`; tidak digunakan ketika `inherit` |
| Scheduling Start Date Override | Nullable date-only | Conditional | `null` | Non-null meng-override inherited date; `null` berarti inherit Scheduling Start Date dari parent chain; SQL `DATE`, API `YYYY-MM-DD` |
| Local Group Status | `open \| locked` | Ya | `open` | Hanya berubah melalui Group Lock/Reopen command |

`Scheduling Start Date Override = null` saat source=`override` **bukan** explicit no-anchor.
Nilai `null` berarti field Scheduling Start Date tetap mengikuti effective
Scheduling Start Date dari nearest parent Group; root Group jatuh ke Project.
Hanya bila seluruh parent chain sampai Project juga menghasilkan `null`,
effective Scheduling Start Date menjadi **no anchor**. Source=`override` tetap
diperlukan untuk membedakan Group yang meng-override Automatic Scheduling dari
Group yang sepenuhnya source=`inherit`.

### 5.2 Effective Resolution

Untuk setiap Group/Task:

1. Jika owning Project `closed`, effective lifecycle = `closed`.
2. Jika owning Project `locked`, effective lifecycle = `locked`.
3. Jika ada ancestor Group `locked`, effective lifecycle = `locked`.
4. Selain itu effective lifecycle = `open`.
5. Local Locked Group memakai frozen effective scheduling configuration yang
   dikonfirmasi saat Lock, regardless of later parent setting changes.
6. Untuk Open scheduling configuration, Group source=`override` memakai own
   Automatic Scheduling value. Scheduling Start Date memakai own non-null value;
   bila override date `null`, date di-resolve dari effective parent Group, lalu
   Project.
7. Open Group source=`inherit` memakai seluruh effective scheduling configuration
   parent Group; root Group jatuh ke Project.
8. Task memakai effective scheduling configuration immediate parent chain atau
   frozen locked ancestor scope yang berlaku.
9. Root Task tanpa parent Group selalu memakai Project scheduling configuration.

Nested Group dengan source=`override` memutus inheritance Automatic Scheduling
dan setiap non-null Scheduling Start Date yang dioverride untuk subtree-nya.
Scheduling Start Date override `null` tetap mewarisi date dari parent chain.
Override **tidak** dapat memutus lifecycle protection Project atau locked
ancestor Group.

### 5.3 Override Initialization and Reset

Ketika user mengubah Group dari `inherit → override`:

- draft Automatic Scheduling diinisialisasi dari current inherited effective
  value;
- draft Scheduling Start Date diinisialisasi dari current inherited effective
  value, termasuk `null`; bila user menyimpan `null`, field tersebut tetap
  mengikuti parent Scheduling Start Date setelah Save;
- hanya setelah user Save, source dan values menjadi confirmed override;
- bila values sama dengan inherited values, perubahan representation saja tidak
  dianggap scheduling-impacting.

Ketika user memilih **Use inherited scheduling settings**:

- source kembali `inherit`;
- stored override values boleh dihapus atau dipertahankan sebagai non-effective
  history sesuai implementation convention, tetapi tidak boleh mempengaruhi
  resolution;
- before/after **effective** values menentukan apakah scheduler/impact guard
  harus berjalan.

### 5.4 Configuration Ownership Boundary

Group scheduling fields adalah configuration pada Group, bukan executable Task
attributes. Group tetap tidak mempunyai:

- Assignee;
- Role;
- Effort;
- Lag;
- Task Capacity Allocation Percentage;
- manual/generated Execution/Commitment date pair;
- Actual Date.

US-4.3 summary tetap derived dari descendant Task dan tidak menjadi writable
Group timeline.

---

## 6. Effective Automatic Scheduling Rules

### 6.1 Effective ON

Task dengan effective Automatic Scheduling `ON`:

- memakai concrete scheduler US-6.1;
- Execution/Commitment dates read-only;
- memakai effective Scheduling Start Date sebagai scheduling anchor;
- bila effective Scheduling Start Date `null`, scheduler tidak mengarang anchor
  lain dan Task mengikuti existing unscheduled semantics US-3.3/US-6.1;
- Task mutation, dependency, WBS order, capacity, dan other established trigger
  menjalankan scheduler berdasarkan effective scope, bukan raw Project toggle.

### 6.2 Effective OFF

Task dengan effective Automatic Scheduling `OFF`:

- memakai manual Execution/Commitment rules existing;
- manual dates editable hanya bila effective lifecycle `open` dan Task sendiri
  eligible;
- Lag tetap tersimpan tetapi tidak mengubah manual timeline;
- Project Buffer tetap tersimpan pada owning Project, tidak menjadi Group
  override, dan tidak mengubah manual timeline.

### 6.3 Mixed Automatic and Manual Subtrees

Satu Project boleh memiliki kombinasi:

```text
Project ON
├─ Group A inherit  => ON
└─ Group B override => OFF
```

atau:

```text
Project OFF
├─ Group A inherit  => OFF
└─ Group B override => ON
```

Rules:

- Group tidak memperoleh priority baru.
- Seluruh Task tetap memakai owning Project Priority.
- Within one Project, visual WBS order tetap deterministic tie-breaker.
- Capacity Assignee tetap shared portfolio-wide; Group tidak menjadi capacity
  silo.
- Project Buffer percentage tetap dimiliki Project dan tidak dapat dioverride
  Group. Untuk Task yang **effectively Automatic ON**, Commitment scheduling
  memakai owning Project Buffer walaupun raw Project Automatic Scheduling=OFF.
  Untuk Task yang effectively OFF, buffer tidak mengubah manual timeline.
- Manual/fixed planning reservation dan automatic allocation memakai existing
  priority/WBS/fixed-reservation semantics. Manual mode tidak otomatis mendapat
  priority lebih tinggi hanya karena manual.
- Karena capacity tetap shared, perubahan Group A boleh mengubah allocation atau
  timeline Group B/Project lain dan wajib melalui impact rules Section 9.

### 6.4 Effective Scheduling Start Date

Untuk Task automatic:

```text
Effective Anchor
= nearest non-null Group Scheduling Start Date override
  else nearest parent Group effective Scheduling Start Date
  else Project Scheduling Start Date
  else no anchor
```

- `override + null` berarti **inherit Scheduling Start Date**, walaupun Automatic
  Scheduling pada Group tersebut tetap custom. Resolution berjalan ke nearest
  parent Group lalu Project; hanya seluruh chain `null` yang menghasilkan no
  anchor.
- Dependency readiness, Lag, zero-capacity dates, completed/Locked anchors, dan
  all existing scheduling rules tetap dapat mendorong Start lebih lambat dari
  effective anchor.
- Effective anchor tidak pernah memaksa Task mulai lebih awal dari dependency
  atau capacity availability.

### 6.5 Project Setting Changes After Group Override

Ketika Project Automatic Scheduling atau Scheduling Start Date berubah:

- Task yang tetap resolve ke Project/inherited **Open** chain menerima effective
  change;
- locally Locked inherited Group tetap memakai frozen effective configuration
  sampai Group Reopen;
- Task di Group override dengan **non-null** Start Date tidak menerima direct
  Start Date change dari parent; custom Automatic Scheduling juga tetap unchanged;
- Group override dengan Start Date `null` tetap menerima perubahan effective
  Start Date dari parent chain;
- nested override mengikuti rule yang sama per field;
- indirect capacity/dependency propagation dari Task lain tetap dapat
  merecalculate Group override tanpa mengubah override values-nya;
- warning ditentukan dari actual timeline delta, bukan hanya configuration
  ownership.

Ketika parent Group override berubah, aturan yang sama berlaku terhadap Open
descendants. Locally Locked Group tetap memakai frozen effective configuration.
Nested overriding Group mempertahankan custom Automatic Scheduling dan non-null
custom Start Date, tetapi Start Date `null` tetap mengikuti perubahan parent
effective date.

---

## 7. Group Lock and Reopen

### 7.1 Group Lock Eligibility

Local Group `Open → Locked` diperbolehkan hanya bila:

- owning Project `open`;
- tidak ada locked ancestor Group;
- target masih Grouping WBS;
- mempunyai minimal satu descendant Executable Task;
- setiap unfinished descendant Executable Task mempunyai complete Execution dan
  Commitment pair serta tidak mempunyai unscheduled reason.

Lock adalah hard validation, bukan scheduler trigger. Failure mempertahankan
status dan seluruh confirmed state.

### 7.2 Locked Group Baseline

Setelah Group Lock berhasil:

- seluruh descendant Task berada dalam protected baseline, termasuk descendant
  di nested Group dengan scheduling override sendiri;
- current **effective Automatic Scheduling dan Scheduling Start Date** untuk
  local Locked Group juga dibekukan sebagai bagian dari protected scheduling
  baseline;
- Group source=`inherit` atau source=`override` dengan Start Date `null` tidak
  mulai memakai parent effective Start Date baru selama masih Locked; parent
  setting changes berlaku pada Open scopes saja;
- scheduler tidak boleh mengubah protected Execution/Commitment dates atau
  baseline allocation;
- planning fields, dependency mutation, WBS create/delete/move/reorder,
  executable/group conversion, Task Reopen, dan Group scheduling settings di
  protected subtree read-only;
- Group tidak menjadi capacity silo; protected allocations tetap menjadi fixed
  reservation bagi Open work;
- Group summary tetap readable.

Group Lock tidak menambah exception baru untuk Group rename. Existing WBS rename
rules tetap berlaku; configuration feature ini tidak memperluas metadata-edit
exception milik Locked Project Name.

### 7.3 Actual Date Factual Exception

Seperti Locked Project:

- complete Actual Date tetap dapat dicatat pada unfinished Task di Locked Group
  selama owning Project tidak Closed;
- protected Execution/Commitment baseline Group tidak berubah;
- Actual Allocation dapat berubah dan dapat merecalculate effective-Open work;
- timeline-only warning/confirmation mengikuti Section 9;
- Actual Date tidak secara implisit Reopen Group.

Jika owning Project sendiri Locked, Project-level Actual Date contract US-6.2
juga tetap berlaku dan Project lock tetap hard ceiling.

### 7.4 Group Reopen

Local Group `Locked → Open` adalah explicit lifecycle command.

- Hanya local lock target yang dibuka oleh command tersebut.
- Jika source=`inherit`, Reopen melepaskan frozen effective config dan
  me-resolve current parent effective Automatic Scheduling/Start Date sebelum
  recalculation.
- Jika source=`override`, confirmed Automatic Scheduling dan non-null own Start
  Date tetap berlaku; bila own Start Date `null`, Reopen me-resolve current parent
  effective Start Date sebelum recalculation.
- Nested Group yang mempunyai own local `locked` tetap Locked.
- Project atau ancestor Group yang masih Locked tetap membuat target effectively
  Locked; karena itu local Reopen tidak ditawarkan sebagai bypass. User harus
  Reopen locking ancestor/Project lebih dahulu.
- Reopen boleh menghasilkan valid unscheduled Task; Group tetap Open tetapi
  tidak dapat di-Lock kembali sampai lock eligibility terpenuhi.
- Reopen menjalankan required scheduling simulation dan mixed Group/Project
  closure pada Section 9.

### 7.5 Project Lifecycle Precedence

- Project `Locked` membuat seluruh subtree effectively Locked tanpa menghapus
  Group local status atau scheduling override.
- Project Reopen tidak otomatis mengubah Group local `locked` menjadi `open`.
- Setelah Project Reopen, local Locked Groups tetap protected anchors; Group
  lain kembali mengikuti effective scheduling configuration-nya.
- Project `Closed` membuat semua Group effectively Closed/read-only; Group
  Lock/Reopen tidak tersedia.
- Reopen Closed Project ke Open memulihkan Group local status dan scheduling
  override yang tersimpan; local Locked Groups tetap Locked.

---

## 8. Structural WBS Behaviour

### 8.1 Create Child and Add Sibling

- New Group boundary selalu mulai dengan Scheduling Source=`inherit` dan local
  status=`open`.
- Add Child di Group membuat Task yang langsung memakai effective configuration
  Group tersebut.
- Add Sibling tidak menyalin override dari anchor; sibling baru memakai
  effective configuration parent authoritative-nya.
- Executable→Grouping conversion membuat resulting Group source=`inherit`, lalu
  executable data yang dipindahkan ke child pertama tetap menerima effective
  parent configuration yang sama kecuali operation lain memang mengubah scope.

### 8.2 Move Task

Move Task ke parent baru:

- setelah move, Task resolve effective configuration dari destination chain;
- bila effective ON/OFF atau anchor berubah, conversion/recalculation mengikuti
  US-3.3/US-6.1 semantics dan Section 9 impact guard;
- source dan destination harus effectively Open;
- moving into/out of protected Locked scope ditolak;
- hierarchy mutation dan required scheduling commit/rollback atomically.

### 8.3 Move Group Subtree

- Group source=`override` membawa override values dan local status bersama
  subtree; parent change tidak mengubah own override values.
- Group source=`inherit` mulai mengikuti effective scheduling configuration
  destination parent setelah move.
- Nested overrides di dalam moved subtree tetap utuh.
- Locked Group atau Group di bawah effective lock tidak dapat dipindahkan.
- Destination effective lock menolak move.
- Effective scheduling changes caused by move use the same atomic impact flow.

### 8.4 Group-to-Executable Conversion

Existing WBS rule mengubah parent Group menjadi Executable ketika child terakhir
hilang. Group-specific scheduling override tidak boleh hilang diam-diam.

Karena itu:

- bila source Group mempunyai Scheduling Source=`override`, operation yang akan
  menghapus/memindahkan child terakhir ditolak;
- user harus terlebih dahulu memilih **Use inherited scheduling settings**;
- local Locked Group sudah menolak child mutation melalui lifecycle rule;
- setelah no local Group configuration remains, existing Group→Executable
  conversion berjalan dan resulting Task memakai parent effective configuration.

Error harus menjelaskan bahwa Group scheduling override harus di-reset sebelum
Group dapat berubah menjadi Task.

---

## 9. Scheduling Impact, Locked Protection, and Reopen Closure

### 9.1 Effective-Delta Trigger

Group scheduling mutation menjalankan scheduler/impact simulation hanya ketika
confirmed effective scheduling behaviour berubah untuk minimal satu Task, atau
ketika lifecycle Reopen memang memerlukan recalculation.

Contoh non-impacting representation change:

```text
Inherited: Automatic ON, Start 10 Aug
Override draft: Automatic ON, Start 10 Aug
```

Save boleh persist source=`override` tanpa scheduler, warning, atau schedule
version change karena effective Task behaviour tidak berubah.

### 9.2 Recalculation Scope Remains Transitive

Recalculation tetap dapat menembus:

- sibling Group di Project yang sama;
- Project-owned root Tasks;
- other Projects;
- manual/automatic scheduling boundaries;
- dependency/capacity connections.

Group boundary membatasi configuration inheritance, **bukan** propagation of
shared capacity or dependency impact.

### 9.3 Mutation Owner Exclusion for Group Changes

Existing US-6.2 rule yang mengecualikan seluruh mutation-owner Project dari
warning dipersempit untuk Group-scoped mutation:

- hanya descendant Task di target Group subtree yang dianggap owner scope dan
  dikeluarkan dari cross-scope warning;
- timeline changes pada Task di luar target subtree, walaupun masih dalam Project
  yang sama, tetap warning-eligible;
- allocation/readiness-only changes dengan unchanged Execution/Commitment Start/
  End tetap tidak ditampilkan.

Ini mencegah Group A secara diam-diam menggeser Group B hanya karena keduanya
berada di Project yang sama.

### 9.4 Warning Labels

Warning harus mengidentifikasi scheduling scope secara tidak ambigu:

```text
Locked Groups
- Project Alpha / Backend

Open Groups
- Project Alpha / Mobile

Locked Projects
- Project Beta

Open Projects
- Project Gamma
```

Rules:

- Group memakai qualified Project + Group path/name agar duplicate Group name
  tidak ambigu.
- Jika timeline-impacted Task berada langsung pada Project-owned scope, Project
  name digunakan.
- Nested Group warning tidak boleh menduplikasi scope yang sama hanya karena
  beberapa descendant Task berubah.
- UI tetap names-only; detailed per-Task reason tidak diperlukan.

### 9.5 Locked Group Evaluation

Locked Group adalah immutable scheduling anchor seperti Locked Project.

Ordinary mutation diblok bila counterfactual simulation menunjukkan minimal
satu protected descendant Task Execution/Commitment Start/End harus berubah.

Tidak memblok hanya karena:

- allocation shape berubah;
- remaining capacity berubah;
- readiness/internal state berubah;
- protected dates tetap sama.

Actual Date mempertahankan factual-data exception US-6.2.

### 9.6 Mixed Group/Project Reopen Closure

Required Reopen Closure dapat berisi Project dan Group.

Contoh:

```text
Reopen Group A
=> Locked Group B would need timeline change
=> B is required
=> opening A+B would require Locked Project C timeline change
=> Project C is required
```

Closure rules:

- connectivity/shared Assignee saja tidak cukup;
- hanya protected timeline delta yang menambah locked scope ke closure;
- expansion berjalan sampai fixed point;
- ancestor lock dependency harus dihormati: Group tidak dapat dibuka efektif
  tanpa required locking ancestor/Project;
- user tidak dapat memilih partial closure;
- **Reopen All** revalidates versions/state dan commits all required lifecycle +
  schedule changes atomically;
- valid unscheduled output tidak menggagalkan Reopen;
- failure rolls back semua Group/Project statuses, timelines, allocations, dan
  versions.

### 9.7 Priority Remains Project-Level

Group Lock/override tidak memberi Group priority baru. Required closure dan
counterfactual simulation tetap memakai:

```text
Manual Dependency readiness
→ Project Priority
→ visual WBS order
→ existing capacity/allocation rules
```

Maka higher-priority work yang protected dan tetap mempunyai dates sama tidak
perlu di-Reopen hanya karena lower-priority Group dibuka.

---

## 10. Dependency, Task Reopen, Capacity Allocation, and Recommendation Compatibility

### 10.1 Manual Dependency

Dependency mutation memerlukan kedua endpoint **effectively Open**.

- Task di Locked Group membuat relation endpoint read-only walaupun Project
  status Open.
- Project/ancestor Group lock juga membuat endpoint read-only.
- Existing manual-only dependency contract US-6.4 tidak berubah.

### 10.2 Reopen Completed Task

Reopen Task tersedia hanya bila target Task effectively Open.

Task dalam Locked Group harus menggunakan Group Reopen lebih dahulu. Task dalam
Project/ancestor lock harus membuka locking scope tersebut lebih dahulu. Reopen
Task tidak pernah menjadi Group Reopen implicit.

### 10.3 Task Capacity Allocation Percentage

Percentage planning mutation hanya tersedia ketika Task effectively Open.
Effective Automatic Scheduling menentukan apakah perubahan memicu generated
schedule atau tetap manual, dengan existing US-6.3 rules.

### 10.4 Assignee Recommendation

US-6.5 eligibility dan simulation memakai effective Task configuration:

- Task harus effectively Open;
- effective Automatic Scheduling `ON` memakai concrete scheduler simulation;
- effective Automatic Scheduling `OFF` memakai existing advisory manual
  simulation;
- automatic simulation memakai effective Scheduling Start Date;
- Group override tidak menciptakan recommendation algorithm baru.

---

## 11. UI / UX

### 11.1 Existing Group Dialog

Activating Group Name on Home tetap membuka satu shared Group dialog. Tidak ada
settings page baru.

Dialog Open Group menampilkan, dalam semantic order:

1. existing Group identity/name area;
2. **Scheduling** section;
3. existing read-only US-4.3 Group Summary, menggunakan horizontal three-card layout pada viewport besar seperti Project Summary;
4. actions.

Scheduling section menampilkan:

- source: `Inherited` atau `Custom`;
- inherited source label, misalnya `Project Alpha` atau `Group Platform`;
- effective Automatic Scheduling;
- effective Scheduling Start Date;
- local/effective lifecycle state;
- action **Override scheduling settings** ketika inherited;
- action **Use inherited scheduling settings** ketika custom;
- satu ordinary **Save** untuk menyimpan Group Name dan scheduling draft secara atomik;
- **Lock Group** atau **Reopen Group** bila lifecycle action eligible. Lifecycle action tetap terpisah dari ordinary Save; Group harus menyimpan ordinary draft terlebih dahulu sebelum Lock agar frozen baseline tidak mengabaikan unsaved Name/scheduling changes.

### 11.2 Inherited State

Ketika source=`inherit`:

- Automatic Scheduling dan Scheduling Start Date ditampilkan sebagai read-only
  effective values;
- source owner ditampilkan sehingga user tahu value berasal dari mana;
- memilih Override membuat editable draft initialized dari effective values;
- Project/parent change dapat memperbarui effective display setelah confirmed
  refresh.

### 11.3 Custom State

Ketika source=`override`:

- Automatic Scheduling dan Scheduling Start Date editable bila effective
  lifecycle Open;
- `null` Scheduling Start Date dapat disimpan dan berarti **Use parent Scheduling
  Start Date**;
- UI menjelaskan bahwa Automatic Scheduling adalah custom, sedangkan Scheduling
  Start Date hanya berhenti mengikuti parent ketika Group menyimpan non-null
  override date;
- ON/OFF confirmation semantics reuse US-3.3 for affected subtree.

### 11.4 Locked-by-Ancestor State

Jika Group effectively Locked karena Project/ancestor:

- UI menjelaskan source lock, misalnya `Locked by Project Alpha` atau
  `Locked by Group Platform`;
- Group scheduling settings read-only;
- local Reopen tidak ditawarkan sebagai bypass;
- user diarahkan ke lifecycle owner yang harus di-Reopen.

Jika Group locally Locked dan tidak ada stronger ancestor lock, **Reopen Group**
tersedia.

### 11.5 Feedback and Accessibility

- Scheduling source/effective values tidak dibedakan hanya dengan color.
- Controls mempunyai accessible labels/descriptions.
- Lock/Reopen dan override reset memakai confirmation ketika existing lifecycle
  or ON/OFF contract requires it.
- Impact warning focus management, stale refresh, Cancel, pending state, Retry,
  duplicate prevention, responsive layout, dan keyboard operation mengikuti
  shared patterns.
- Summary loading failure tidak menghapus scheduling draft atau lifecycle state.

---

## 12. API and Error Contract

Exact endpoint routing mengikuti repository architecture. Contract harus
membedakan source=`inherit` dari source=`override`; pada source=`override`,
`schedulingStartDate=null` mempunyai makna field-level inheritance ke parent
Scheduling Start Date.

Conceptual scheduling payload. Every scheduling/lifecycle write carries the latest confirmed Group scheduling `version`; a successful write returns the next confirmed version:

```json
{
  "expectedVersion": 4,
  "schedulingSource": "override",
  "automaticScheduling": true,
  "schedulingStartDate": "2026-08-10"
}
```

Override Automatic Scheduling while inheriting parent Start Date:

```json
{
  "expectedVersion": 4,
  "schedulingSource": "override",
  "automaticScheduling": true,
  "schedulingStartDate": null
}
```

Reset:

```json
{
  "expectedVersion": 4,
  "schedulingSource": "inherit"
}
```

Lifecycle remains dedicated commands, not generic settings field mutation. Lifecycle commands carry the same latest confirmed `expectedVersion`. Reopen-closure confirmation additionally carries the scheduling-impact token; both the Group version and impact token are revalidated before mutation.

Minimum stable error concepts:

| Error Code | HTTP | Condition |
| --- | ---: | --- |
| `GROUP_NOT_GROUPING_WBS` | 409 | Target is no longer a Group |
| `GROUP_LOCKED_READ_ONLY` | 409 | Mutation targets locally/effectively Locked Group subtree |
| `GROUP_CANNOT_LOCK_WITH_UNSCHEDULED_TASKS` | 409 | Group Lock eligibility fails |
| `GROUP_OVERRIDE_MUST_BE_RESET_BEFORE_TASK_CONVERSION` | 409 | Last-child removal would discard Group override |
| `GROUP_REOPEN_REQUIRED` | 409 | Planning mutation requires local Group Reopen |
| `SCHEDULING_LOCKED_SCOPE_IMPACT` | 409 | Ordinary mutation would require protected Group/Project timeline change |
| `SCHEDULING_SCOPE_REOPEN_CLOSURE_REQUIRED` | 409 | Reopen requires mixed Group/Project fixed-point closure |
| `SCHEDULING_IMPACT_STALE` | 409 | Effective config/status/version changed after preview |
| `GROUP_SCHEDULING_STALE` | 409 | Owner Group scheduling/lifecycle version changed before write confirmation |

Existing Project-specific error codes may be preserved for Project-only cases,
but API/UI must distinguish the lifecycle owner sufficiently to direct the user
to the correct Reopen action.

---

## 13. Acceptance Criteria

### AC-1 — Group defaults to inherited scheduling

**Given** a new Group is created
**Then** Scheduling Source is `inherit`
**And** it uses parent effective Automatic Scheduling and Scheduling Start Date
without copying an independent active value.

### AC-2 — Root inheritance falls back to Project

**Given** Group has no ancestor Group override
**Then** effective scheduling configuration comes from owning Project.

### AC-3 — Nested inheritance uses nearest ancestor

**Given** parent Group has override and child Group remains inherited
**Then** child and its non-overridden descendants use parent Group effective
configuration.

### AC-4 — Nested override wins per field

**Given** nested child Group has its own override
**Then** its subtree uses child Automatic Scheduling
**And** a non-null child Scheduling Start Date overrides parent/Project date
**And** a null child Scheduling Start Date inherits the nearest parent effective
date
**And** parent lifecycle lock still dominates.

### AC-5 — Override initializes from current effective values

**When** user chooses Override scheduling settings
**Then** draft ON/OFF and Start Date equal current inherited effective values.

### AC-6 — Null custom Start Date inherits parent anchor

**Given** Group saves source=`override` with Scheduling Start Date `null`
**When** its nearest parent Group has an effective Scheduling Start Date
**Then** the Group uses that parent effective date
**And** if no parent Group provides a date, resolution continues to Project
**And** only when the entire chain through Project is `null` does the Group have
no effective anchor.

### AC-7 — Reset restores live inheritance

**When** Custom Group chooses Use inherited scheduling settings
**Then** parent effective values become authoritative again
**And** future parent changes flow into that Group unless it overrides again.

### AC-8 — No-op representation change does not reschedule

**Given** inherited values and new custom values are identical
**When** override is saved
**Then** no scheduler/impact warning/schedule-version change occurs solely from
source metadata change.

### AC-9 — Effective ON uses concrete scheduler

**Given** Task effective Automatic Scheduling is ON
**Then** generated date, preview, trigger, and allocation rules use US-6.1 even
when owning Project raw toggle is OFF.

### AC-10 — Effective OFF uses manual scheduling

**Given** Task effective Automatic Scheduling is OFF
**Then** manual timeline rules apply even when owning Project raw toggle is ON.

### AC-11 — Project settings affect only inheriting direct scope

**When** Project Automatic Scheduling or Start Date changes
**Then** direct effective value change applies to inherited Tasks/Groups
**And** custom Group values remain unchanged
**And** indirect capacity/dependency recalculation may still propagate normally.

### AC-12 — Group does not create priority, buffer, or capacity isolation

**Then** Project Priority, WBS order, and shared Assignee capacity remain
unchanged product rules
**And** Group override alone never creates Group Priority, Group Buffer, or Group
capacity pool
**And** every effectively automatic Task uses the owning Project Buffer for
Commitment scheduling regardless of the raw Project Automatic Scheduling toggle.

### AC-13 — Group Lock validates subtree schedule

**When** Open Group is Locked
**Then** it must contain at least one Executable descendant
**And** every unfinished descendant is fully scheduled
**Else** Lock fails atomically.

### AC-14 — Locked Group protects complete subtree

**Given** Group locally Locked
**Then** descendant planning, WBS, dependency, and generated baseline are
immutable
**And** nested custom scheduling values do not bypass the lock.

### AC-15 — Project/ancestor lock is hard ceiling

**Given** Group local status Open but Project or ancestor Group Locked
**Then** Group/Tasks are effectively Locked
**And** child cannot Reopen itself to bypass that owner.

### AC-16 — Project Reopen preserves local Group locks

**When** Locked Project reopens
**Then** custom scheduling overrides remain stored
**And** locally Locked Groups remain Locked.

### AC-17 — Locked inherited Group freezes effective config

**Given** locally Locked Group source=`inherit`
**When** parent Project/Group scheduling values change
**Then** Locked Group keeps the effective Automatic Scheduling and Start Date
that were confirmed at Lock time
**And** its protected baseline is not silently reconfigured.

### AC-18 — Group Reopen re-resolves inheritance

**Given** locally Locked Group source=`inherit` and parent settings changed while
it was Locked
**When** Group Reopen succeeds
**Then** it resolves the current parent effective values and recalculates as
required.

### AC-19 — Group Reopen preserves nested local locks

**When** parent Group reopens
**Then** child Group with own local Locked status remains Locked.

### AC-20 — Actual Date remains factual in Locked Group

**Given** unfinished Task in Locked Group
**When** complete Actual Date is entered
**Then** protected baseline remains unchanged
**And** Actual Allocation may recalculate effective-Open work under US-6.2.

### AC-21 — Reopen Task requires effective Open lifecycle

**Given** completed Task in Locked Group
**Then** Task Reopen is unavailable/rejected until required Group/Project
lifecycle owner is reopened.

### AC-22 — Same-Project sibling impact is not silently excluded

**Given** Group A scheduling change shifts Task dates in Group B of same Project
**Then** Group B is warning-eligible
**And** only Group A owner subtree is excluded from warning.

### AC-23 — Allocation-only sibling change does not warn

**Given** Group A change recalculates Group B allocation but all Group B Task
Start/End dates remain unchanged
**Then** Group B is not shown in warning.

### AC-24 — Locked Group timeline impact blocks ordinary change

**Given** ordinary mutation would require a Locked Group protected Task Start/End
to change
**Then** save is blocked atomically
**And** allocation-only pressure with unchanged dates does not block.

### AC-25 — Reopen closure can contain Groups and Projects

**Given** Group Reopen transitively requires other locked scopes to move
**Then** fixed-point closure contains every required Group/Project
**And** user can only Reopen All or Cancel.

### AC-26 — Reopen closure respects priority

**Given** higher-priority protected scope keeps identical protected dates
**When** lower-priority Group reopens
**Then** connectivity/shared Assignee alone does not add higher-priority scope to
closure.

### AC-27 — Move Task adopts destination effective config

**When** Task is moved between Groups
**Then** destination effective ON/OFF/anchor apply after move
**And** hierarchy + required schedule changes are atomic.

### AC-28 — Moved inherited Group adopts destination config

**When** source=`inherit` Group moves under another parent
**Then** it inherits destination effective scheduling values after move.

### AC-29 — Moved custom Group keeps own config

**When** source=`override` Group moves
**Then** own override values and nested overrides remain unchanged.

### AC-30 — Locked scope rejects structural move

**Given** source or destination is effectively Locked
**Then** move/reorder/conversion that mutates protected subtree is rejected.

### AC-31 — Last-child conversion cannot silently discard override

**Given** Group source=`override`
**When** operation would remove/move its final child and convert Group to Task
**Then** operation is rejected
**And** user must reset scheduling source to inherit first.

### AC-32 — Dependency mutation uses effective lifecycle

**Given** dependency endpoint Task belongs to Locked Group
**Then** dependency mutation is rejected even when owning Project is Open.

### AC-33 — Capacity percentage uses effective lifecycle/mode

**Given** Task in Group override
**Then** US-6.3 editability and scheduling behaviour use effective lifecycle and
Automatic Scheduling, not raw Project values.

### AC-34 — Assignee recommendation uses effective config

**Given** recommendable Task in Group override
**Then** US-6.5 chooses automatic/manual simulation and anchor from effective
Task configuration
**And** effectively Locked Task is not recommendable.

### AC-35 — Group dialog exposes source and lifecycle clearly

**When** Group dialog opens
**Then** user can distinguish inherited/custom scheduling values and the owner of
any effective lock
**And** eligible Override/Use inherited/Lock/Reopen actions are available without
new route navigation
**And** Group Name dan scheduling draft memakai satu Save action dan satu atomic write sehingga partial save tidak mungkin terjadi.

### AC-36 — Closed Project dominates Group

**Given** Project Closed
**Then** every Group is effectively Closed/read-only
**And** Group scheduling/lifecycle commands are rejected.

### AC-37 — Atomicity and stale protection

**Given** settings, lifecycle, impact set, hierarchy, or schedule version changes
concurrently
**Then** stale confirmation cannot overwrite newer state
**And** failed operations preserve confirmed Group config/status, WBS, timelines,
allocations, dependencies, and related Project state.

---

## 14. Required Edge-Case Test Matrix

| ID | Scenario | Expected |
| --- | --- | --- |
| G-01 | Project ON/date A; Group inherit | Group Tasks use ON/date A |
| G-02 | Project ON/date A; Group override OFF/date B | Group Tasks manual; date B stored but inactive under OFF |
| G-03 | Project OFF; Group override ON/date B | Group Tasks automatic from B |
| G-04 | Project OFF; Group override ON/date B; Project Buffer 20% | Group automatic Commitment uses the Project 20% Buffer |
| G-05 | Project date A; Group override ON/null | Group Tasks automatic using inherited Project date A |
| G-06 | Parent Group override ON/B; nested Group inherit | Nested Tasks use ON/B |
| G-07 | Parent Group override ON/B; nested Group override OFF/C | Nested subtree manual; parent siblings remain automatic |
| G-08 | Parent Project config changes while Group custom with non-null date | Custom Automatic + date unchanged; only indirect impact may propagate |
| G-08A | Parent Project date A→B; Group override ON/null | Group effective Start Date changes A→B and affected open subtree recalculates |
| G-09 | Parent Group config changes while nested Group custom with non-null date | Nested custom Automatic + date unchanged |
| G-09A | Parent Group date B→C; nested Group override ON/null | Nested Group effective Start Date changes B→C |
| G-10 | Inherit→Override with identical effective values | Persist source only; no scheduler/version warning |
| G-11 | Override→Inherit changes effective ON/OFF | Existing US-3.3 transition semantics applied to affected subtree |
| G-12 | Group Lock with one unscheduled descendant | Reject; no baseline/status change |
| G-13 | Group Lock with all descendants scheduled | Lock; immutable subtree baseline |
| G-14 | Project locks while Group custom Open | Entire subtree effectively Locked; custom values preserved |
| G-15 | Project reopens; Group local Open | Group returns effective Open with prior custom values |
| G-16 | Project reopens; Group local Locked | Group remains Locked |
| G-17 | Parent Group locks; nested Group local Open/custom | Nested subtree effectively Locked; custom values preserved |
| G-18 | Locally Locked inherited Group; parent scheduling changes | Locked Group keeps frozen effective ON/OFF + anchor |
| G-19 | Reopen that inherited Group after parent changed | Current parent effective config is re-resolved, then recalculated |
| G-20 | Parent Group reopens; nested Group local Locked | Nested remains Locked |
| G-21 | Actual Date on unfinished Locked-Group Task | Save factual data; baseline unchanged; Open impact recalculated |
| G-22 | Reopen completed Task in Locked Group | Reject until Group/ancestor lifecycle reopened |
| G-23 | Group A config shifts open sibling Group B dates | Warning lists qualified B |
| G-24 | Group A config only changes B allocation | No B warning |
| G-25 | Group A config would shift Locked Group B dates | Block ordinary save |
| G-26 | Group A config pressures Locked B but dates unchanged | Save allowed subject to other impact |
| G-27 | Reopen Group A requires Locked Group B and Project C | Reopen All closure A/B/C |
| G-28 | Higher-priority Locked A unchanged when lower-priority B reopens | A absent from closure |
| G-29 | Move Task inherited ON scope → custom OFF scope | Task adopts manual mode atomically |
| G-30 | Move inherited Group under custom parent | Entire inherited subtree adopts destination values |
| G-31 | Move custom Group under different parent | Custom values retained |
| G-32 | Try move Locked Group or into Locked Group | Reject |
| G-33 | Remove last child from custom Group | Reject until reset to inherit |
| G-34 | Add Sibling beside custom Group | New sibling does not copy Group override |
| G-35 | Executable→Group conversion | New Group inherits; moved child keeps equivalent effective config |
| G-36 | Dependency endpoint in locally Locked Group | Read-only/reject mutation |
| G-37 | Assignee recommendation Project OFF + Group ON | Concrete automatic simulation uses Group anchor |
| G-38 | Assignee recommendation Project ON + Group OFF | Existing manual advisory simulation |
| G-39 | Closed Project with custom/open Groups | All effectively Closed; no Group action |
| G-40 | Concurrent parent setting change after impact preview | Stale response; recompute effective config and warning |
| G-41 | Duplicate Group names under different parents impacted | Warning uses qualified paths, no ambiguity |


---

## 15. Required Automated Tests by Layer

### 15.1 Domain Tests

- Scheduling Source validation and null-as-parent-date inheritance semantics.
- Nearest-ancestor effective scheduling resolution.
- Project/ancestor lifecycle hard-ceiling resolution.
- Group Lock eligibility and protected subtree rules.
- Group→Executable override-loss guard.
- Effective-delta classification for no-op override representation changes.

### 15.2 Application Tests

- Inherit/override Save and reset coordination.
- Effective ON↔OFF transition scheduling for only affected subtree plus
  transitive scope.
- Group Lock/Reopen, Actual Date exception, and mixed Group/Project reopen
  closure.
- Same-Project sibling impact preview and Locked Group blocking.
- Move Task/Group across effective configuration boundaries.
- Atomic rollback and stale impact revalidation.

### 15.3 Repository Integration Tests

- Persist/reload source=`inherit` versus source=`override` with null Start Date
  retaining field-level parent-date inheritance semantics.
- Persist local Group status independently from Project status.
- Query/traversal resolves arbitrary nested override chains deterministically.
- Protected Group baseline remains unchanged during external recalculation.
- Mixed reopen closure and rollback preserve relation/status integrity.

### 15.4 API Integration Tests

- Group scheduling Get/Update payload mapping.
- Override/null/inherit distinction, including null Start Date parent-chain
  resolution.
- Lock eligibility errors.
- Effective locked mutation rejection.
- Same-Project Group impact warning payload.
- Mixed Group/Project reopen closure and stale-token response.
- Last-child conversion override guard.

### 15.5 Frontend and Acceptance Tests

- Existing Group dialog shows inherited source/effective values.
- Override draft initialization, Save, Cancel, reset-to-inherit.
- Null Start Date display as inherited-from-parent, including no-anchor only when
  the full chain is null.
- Local versus ancestor/project lock copy and disabled actions.
- Group Lock/Reopen flows without route change.
- Qualified same-Project Group impact warnings.
- Locked Group Actual Date flow and completed Task Reopen restriction.
- Move/create/conversion behaviour across configuration boundaries.
- Recommendation/editability follows effective config.
- Responsive, focus, keyboard, stale response, loading/error, and duplicate-submit
  behaviour.

Every affected Acceptance Criterion requires Code Inspection,
Unit/Integration, and Acceptance-Level evidence under the repository
Three-Level Confidence standard.

---

## 16. Documentation Compatibility

This story requires reconciliation of:

- US-3.3: Project scheduling values become defaults/fallback for inheriting
  Group/Task scope; Project Buffer remains Project-level.
- US-4.1: Group may own scheduling configuration/lifecycle boundary while still
  owning no executable Task attributes; Task manual editability and WBS mutation
  eligibility use effective config/lifecycle.
- US-4.2: Task Reopen requires effective Open lifecycle, not Project Open alone.
- US-4.3: Group dialog gains Scheduling + lifecycle controls while summary stays
  read-only/derived.
- US-5.1: dependency endpoint mutability uses effective lifecycle.
- US-6.1: scheduler activation and anchor resolution use effective Task
  configuration; priority/capacity algorithms remain unchanged.
- US-6.2: immutable protected scope and impact/reopen closure expand from Project
  only to Project + Group.
- US-6.3: percentage editability and trigger use effective lifecycle/mode.
- US-6.5: recommendation eligibility, simulation mode, and automatic anchor use
  effective Task configuration.

---

## 17. Locked Product Decisions

- Group configuration scope is Automatic Scheduling + Scheduling Start Date +
  Group Lock/Reopen only.
- Project Buffer remains Project-level; no Group Buffer.
- Project Priority remains Project-level; no Group Priority.
- Group scheduling inheritance is live, not copied, until explicit override.
- Nested inherited Group follows nearest ancestor effective Group configuration,
  then Project.
- Group source=`override` always customizes Automatic Scheduling; Scheduling Start
  Date is custom only when non-null.
- Custom `Scheduling Start Date = null` means inherit the nearest parent effective
  Scheduling Start Date, falling back through ancestor Groups to Project; only a
  fully-null chain means no anchor.
- Group local lifecycle is Open/Locked; no Group Closed status.
- Project Closed/Locked and ancestor Group Locked are hard ceilings.
- Group Lock protects the complete descendant subtree, including nested custom
  Groups.
- Local Locked Group freezes its resolved Automatic Scheduling and Scheduling
  Start Date. Any parent-derived field—including Start Date `null` on a custom
  Group—adopts newer parent values only when the Group is Reopened.
- Project Reopen never silently reopens locally Locked Groups.
- Group Reopen never silently reopens nested locally Locked Groups.
- Actual Date remains factual and allowed on unfinished Task in Locked Group;
  protected planning baseline does not move.
- Reopen completed Task requires effective Open lifecycle.
- Capacity remains shared; Group is not a resource pool/isolation boundary.
- Same-Project timeline impact outside mutation Group must warn; only the
  mutation Group subtree is owner-excluded.
- Locked Group timeline impact blocks ordinary mutation exactly like Locked
  Project; allocation-only pressure does not.
- Required Reopen Closure may contain both Groups and Projects and is atomic.
- Move of inherited Group adopts destination effective scheduling config; custom
  Group carries its override.
- A custom Group cannot silently become Executable by losing its last child;
  override must be reset to inherit first.
- No-op source/override representation changes with identical effective values do
  not invoke scheduler or warning.

---

## 18. Unresolved Questions

None.
