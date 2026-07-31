# US-4.3 — View Group and Project Summary

## 1. User Story

**Sebagai** Engineering Lead,  
**Saya ingin** melihat ringkasan timeline dan penyelesaian effort pada sebuah Group maupun Project,  
**Sehingga** saya dapat memahami rentang rencana dan progres pekerjaan pada subtree atau keseluruhan Project tanpa membuka setiap Task satu per satu.

---

## 2. Business Context

SchedMind tidak mempunyai entity Group atau Task yang terpisah dari WBS:

- **Task** adalah Executable WBS, yaitu WBS leaf tanpa child.
- **Group** adalah Grouping WBS, yaitu WBS yang mempunyai child.

View Group saat ini hanya menampilkan jumlah direct item dan informasi bahwa detail Task dikelola pada Task di dalam Group. Informasi tersebut tidak membantu Engineering Lead memahami rencana maupun progres subtree Group.

Project adalah WBS level `0` tanpa root WBS record terpisah. Karena itu Edit Project harus menampilkan summary yang sama untuk keseluruhan Project, dengan seluruh Task dalam Project diperlakukan sebagai descendant dari WBS level `0`. Summary Project ditempatkan pada combined Edit Project form; Add Project tidak mempunyai confirmed Task subtree dan tidak menampilkan summary.

Story ini mengganti informasi View Group yang lama dan menambahkan Project Summary pada Edit Project dengan tiga read-only summaries yang dihitung dari **seluruh descendant Task secara rekursif**, tanpa membatasi kedalaman hierarchy:

1. Execution Timeline dan coverage.
2. Commitment Timeline dan coverage.
3. Effort Completion, termasuk disclosure untuk Task tanpa Effort.

Summary adalah projection dari current confirmed Task data. Summary bukan executable attribute milik Group atau editable Project field, bukan source of truth baru, dan tidak boleh dipersist sebagai duplicate aggregate state.

---

## 3. Terminology

| Product Term | Meaning |
| --- | --- |
| Project Structure | UI hierarchy WBS |
| Task | Executable WBS / WBS leaf |
| Group | Grouping WBS / WBS yang mempunyai child |
| Project Summary Root | Project sebagai logical WBS level `0`; tidak mempunyai root WBS record terpisah |
| Summary Subject | Selected Group pada View Group atau selected Project pada Edit Project |
| Direct Child | WBS tepat satu level di bawah Group atau Project Summary Root |
| Descendant | Seluruh WBS di bawah selected Group atau logical Project level `0` pada kedalaman berapa pun |
| Descendant Task | Seluruh descendant yang merupakan Executable WBS |
| Scheduled Task | Descendant Task yang mempunyai Start dan End lengkap untuk timeline terkait |
| Known Effort | Effort Task yang mempunyai persisted `effortMinutes` |
| Task Without Effort | Descendant Task dengan `effortMinutes == null` |
| Completed Task | Descendant Task dengan persisted `Actual End != null` |
| Effort Completion | Completed Known Effort dibagi Total Known Effort |
| Confirmed WBS Tree | WBS tree yang berasal dari confirmed backend response/cache, bukan unconfirmed draft atau preview |

---

## 4. Scope

### 4.1 In Scope

- Mengganti informational alert pada View Group dengan useful read-only summary.
- Menampilkan summary yang sama pada combined Edit Project form karena Project adalah WBS level `0`.
- Mengagregasi seluruh descendant Task secara rekursif untuk selected Group atau seluruh Task dalam selected Project.
- Menampilkan Execution Timeline aggregate.
- Menampilkan Execution schedule coverage.
- Menampilkan Commitment Timeline aggregate.
- Menampilkan Commitment schedule coverage.
- Menampilkan completed known effort, total known effort, dan percentage.
- Menampilkan jumlah descendant Task tanpa Effort dan menjelaskan bahwa Task tersebut dikeluarkan dari effort calculation.
- Menggunakan `Actual End` sebagai satu-satunya source of truth untuk completed effort.
- Memperbarui summary setelah confirmed WBS state berubah tanpa browser hard refresh.
- Menangani nested Group, Project tanpa Task, partially scheduled subtree, missing Effort, zero known effort, date-only formatting, local summary loading/failure, stale response, accessibility, dan responsive layout.
- Menggunakan existing shared wide Dialog variant untuk Edit Project agar fields dan summary tidak dipaksa ke dialog sempit.
- Menyediakan automated evidence untuk setiap Acceptance Criterion melalui Three-Level Confidence Standard di `AGENTS.md`.

### 4.2 Out of Scope

- Menambahkan editable fields pada Group atau editable aggregate fields pada Project.
- Menyimpan aggregate timeline, aggregate effort, atau percentage pada Group atau Project.
- Menambahkan persisted progress percentage atau remaining effort.
- Mengubah Effort menjadi remaining effort setelah Task completed.
- Mengubah source of truth completion dari `Actual End`.
- Menambahkan Task count completion sebagai primary progress metric.
- Menampilkan jumlah direct child, total Group, atau structure summary.
- Menampilkan Assignee summary, Role summary, dependency count, blocked state, Forecast, Delivery Impact, Project Health, atau Gantt data.
- Mengubah Execution atau Commitment scheduling algorithm.
- Menambahkan backend endpoint, database column, migration, aggregate table, atau new summary-specific production query hanya untuk summary ini.
- Mengubah Add Project menjadi summary view; Add Project belum mempunyai confirmed Task data.
- Mengubah Group rename, Task edit, Reopen Task, Project lifecycle, atau dependency behaviour.
- Menghitung summary dari unconfirmed schedule preview atau unsaved Task draft.

---

## 5. Locked Product Rules

### 5.1 Recursive Descendant Scope

- Untuk View Group, calculation dimulai dari selected Group dan menelusuri seluruh `children` secara rekursif.
- Untuk Edit Project, calculation dimulai dari Project Summary Root pada WBS level `0` dan menelusuri seluruh top-level WBS nodes beserta `children` secara rekursif.
- Hanya descendant **Task** yang masuk ke calculation.
- Nested Group tidak dihitung sebagai Task dan tidak menyumbang Effort, dates, atau completion secara langsung.
- Direct child dan deeper descendant mempunyai bobot yang sama sebagai Task source.
- Selected Group atau Project sendiri tidak dihitung sebagai Task.
- Nilai pada `Group.executable`, bila muncul akibat stale atau malformed projection, tidak boleh digunakan untuk summary.
- Traversal tidak mempunyai product depth limit.
- Urutan child tidak memengaruhi hasil aggregate.

Contoh:

```text
Group A
├── Task 1
├── Group B
│   ├── Task 2
│   └── Group C
│       └── Task 3
└── Task 4
```

Summary Group A menggunakan Task 1, Task 2, Task 3, dan Task 4.
Summary Group B menggunakan Task 2 dan Task 3.
Summary Group C menggunakan Task 3.

Project Summary untuk Project yang memiliki tree di atas menggunakan Task 1, Task 2, Task 3, dan Task 4, regardless of berapa top-level WBS root yang dimiliki Project.

### 5.2 Read-Only Derived Projection

- Group tetap tidak boleh memiliki executable attributes.
- Project Summary tidak menjadi writable Project setting dan tidak menggunakan Project form draft sebagai aggregate source.
- Summary dihitung dari current confirmed descendant Task projection.
- View Group summary tidak mempunyai Save action. Project Summary berada di dalam Edit Project dialog tetapi tetap read-only dan bukan bagian dari Project update payload.
- Membuka atau menutup View Group atau Edit Project tidak melakukan summary mutation request.
- Tidak ada summary field baru pada WBS atau Project domain entity, persistence model, atau API DTO.
- Tidak ada separate Group atau Project Summary aggregate.
- Frontend menggunakan satu pure deterministic helper pada WBS feature domain/presentation boundary untuk menghasilkan view model dari confirmed tree; Group dan Project presentation tidak boleh menduplikasi calculation logic.
- Backend tetap source of truth untuk Task Effort, Actual End, Execution Timeline, dan Commitment Timeline.
- Summary harus direcompute ketika confirmed WBS tree berubah; summary tidak boleh menyimpan copy independen yang dapat menjadi stale.

### 5.3 Execution Timeline Aggregate

Execution aggregate dihitung hanya dari descendant Task dengan complete Execution pair:

```text
executionStart != null AND executionEnd != null
```

Rules:

- Aggregate Start = earliest descendant `executionStart` dari scheduled Execution Tasks.
- Aggregate End = latest descendant `executionEnd` dari scheduled Execution Tasks.
- Start dan End berasal dari semua scheduled descendant Task; keduanya tidak harus berasal dari Task yang sama.
- Task tanpa complete Execution pair tidak menyumbang date ke aggregate.
- Task tanpa complete Execution pair tetap masuk total descendant Task untuk coverage.
- Execution calculation tidak menggunakan Commitment dates.
- Actual End tidak mengganti Execution dates dalam summary.
- Date comparison menggunakan date-only value, bukan browser local timestamp.

### 5.4 Commitment Timeline Aggregate

Commitment aggregate dihitung hanya dari descendant Task dengan complete Commitment pair:

```text
commitmentStart != null AND commitmentEnd != null
```

Rules:

- Aggregate Start = earliest descendant `commitmentStart` dari scheduled Commitment Tasks.
- Aggregate End = latest descendant `commitmentEnd` dari scheduled Commitment Tasks.
- Start dan End berasal dari semua scheduled descendant Task; keduanya tidak harus berasal dari Task yang sama.
- Task tanpa complete Commitment pair tidak menyumbang date ke aggregate.
- Task tanpa complete Commitment pair tetap masuk total descendant Task untuk coverage.
- Commitment calculation tidak menggunakan Execution dates.
- Actual End tidak mengganti Commitment dates dalam summary.
- Date comparison menggunakan date-only value, bukan browser local timestamp.

### 5.5 Schedule Coverage

Coverage dihitung terpisah untuk Execution dan Commitment.

```text
Execution Coverage  = execution-scheduled descendant Task count / total descendant Task count
Commitment Coverage = commitment-scheduled descendant Task count / total descendant Task count
```

A Task dianggap scheduled untuk timeline terkait hanya bila Start dan End lengkap.

Required copy:

```text
3 of 5 tasks scheduled
```

Rules:

- Coverage numerator adalah count Task dengan complete pair pada timeline terkait.
- Coverage denominator adalah seluruh descendant Task, termasuk Task tanpa Assignee, tanpa Effort, completed, dan unscheduled.
- Execution dan Commitment coverage tidak boleh digabung menjadi satu angka.
- Coverage bukan percentage dan tidak perlu progress bar.
- `5 of 5 tasks scheduled` tetap ditampilkan ketika coverage penuh.
- Ketika tidak ada Task yang scheduled, tampilkan `0 of 5 tasks scheduled`.
- Timeline range ditampilkan hanya bila numerator lebih dari zero.
- Ketika numerator zero, date value menampilkan `Not scheduled`, bukan fake date, zero date, atau Project date.

### 5.6 Effort Completion

Effort Completion menggunakan persisted Effort dan persisted Actual End dari seluruh descendant Task.

Definitions:

```text
Total Known Effort = sum(effortMinutes) for descendant Tasks with effortMinutes != null
Completed Known Effort = sum(effortMinutes) for descendant Tasks where:
  effortMinutes != null AND Actual End != null
Effort Completion Percentage = Completed Known Effort / Total Known Effort × 100
```

Rules:

- `Actual End` adalah satu-satunya completion source of truth.
- Unfinished Task dengan Effort masuk denominator tetapi tidak masuk numerator.
- Completed Task dengan Effort masuk numerator dan denominator.
- Task tanpa Effort tidak masuk numerator maupun denominator.
- Completed Task tanpa Effort tetap dihitung sebagai Task Without Effort dan tidak boleh diberi invented effort.
- Reopen Task menghapus Actual End; setelah confirmed Reopen, Effort Task tersebut tetap dalam denominator tetapi keluar dari numerator.
- Mengubah Actual End tanpa mengubah Effort dapat mengubah completed known effort dan percentage.
- Mengubah Effort dapat mengubah numerator, denominator, dan percentage sesuai current Actual End.
- Summary tidak menyimpan completed effort atau percentage.
- Summary tidak menghitung remaining effort per Task.
- Summary tidak mengubah original Effort setelah completion.

Required primary copy when Total Known Effort is greater than zero:

```text
24 of 40 hours completed (60%)
```

The word `hours` may follow the shared compact hour formatting convention, but the UI must preserve half-hour precision and must not round `0.5h` to a whole hour.

### 5.7 Effort Percentage Precision

- Calculation menggunakan integer minutes untuk menghindari floating-point accumulation error.
- Percentage ditampilkan dengan maksimum satu decimal place.
- Percentage dibulatkan ke nearest one decimal place.
- Trailing `.0` dihilangkan.
- Exact completion menampilkan `100%`.
- Zero completed known effort dengan positive total menampilkan `0%`.

Examples:

| Completed | Total | Display |
| ---: | ---: | ---: |
| 24h | 40h | 60% |
| 1h | 3h | 33.3% |
| 0h | 12.5h | 0% |
| 12.5h | 12.5h | 100% |

### 5.8 Task Without Effort Disclosure

Task Without Effort count adalah jumlah seluruh descendant Task dengan `effortMinutes == null`, regardless of completion state.

When count is greater than zero, display supporting copy:

```text
2 tasks without effort are excluded from this calculation.
```

Singular copy:

```text
1 task without effort is excluded from this calculation.
```

Rules:

- Disclosure berada pada Effort Completion section, bukan separate structure section.
- Disclosure harus selalu terlihat ketika count lebih dari zero.
- Percentage tetap dihitung dari Known Effort dan harus dipahami sebagai completion dari known/estimated effort only.
- UI tidak boleh menampilkan percentage seolah-olah seluruh Task mempunyai Effort tanpa disclosure.
- Task Without Effort tidak boleh dianggap `0 hours`, karena hal tersebut akan menyembunyikan incomplete planning data.

### 5.9 Zero Known Effort

Ketika `Total Known Effort == 0`:

- Percentage tidak dapat dihitung dan tidak boleh ditampilkan sebagai `0%`.
- Primary value menampilkan `Effort completion unavailable`.
- Supporting value menampilkan Task Without Effort disclosure.
- UI tidak boleh melakukan division by zero.
- UI tidak boleh menampilkan `0 of 0 hours completed (0%)` karena wording tersebut menyiratkan valid estimate.

For a valid Group, zero known effort means every descendant Task has no Effort.

### 5.10 Empty and Defensive States

A Project may validly have zero Task because creation does not create a root WBS record or child automatically.

- When Edit Project has zero confirmed Task, display `No tasks are available for this project.`
- Do not display `0 of 0 tasks scheduled`, effort percentage, or fake timeline values.
- This is an ordinary empty state, not a data-integrity failure.

For a valid Group, finite hierarchy normally contains at least one descendant Task because a child without children is a Task. Frontend must still handle malformed, stale, or transient Group projection safely:

- If selected node is Group but traversal finds zero descendant Task, display `No descendant tasks are available for this group.`
- Do not crash, recurse indefinitely, mutate data, or show misleading aggregate values.
- Existing backend tree invariants remain authoritative; this story does not add cycle-repair behaviour.

### 5.11 Date Formatting

- API date tetap `YYYY-MM-DD`.
- UI menggunakan established shared date-only formatter.
- Formatting harus menetapkan UTC/date-only semantics agar tanggal tidak bergeser karena browser timezone.
- Range menggunakan separator yang accessible dan konsisten, misalnya:

```text
01 Aug 2026 – 12 Aug 2026
```

- Story ini tidak mengubah locale policy aplikasi.
- Start/End labels tetap tersedia secara semantic; range tidak boleh hanya mengandalkan visual punctuation untuk screen reader meaning.

### 5.12 Refresh and Confirmed-State Behaviour

Group dan Project Summary harus merefleksikan confirmed state setelah operation yang mengubah descendant projection, termasuk:

- Task create/delete/move atau structural conversion.
- Task Effort change.
- Task Actual End set.
- Reopen Task.
- Execution/Commitment manual timeline change.
- Automatic scheduling recalculation yang menghasilkan dates baru.
- Assignee/dependency/settings/capacity/holiday change yang melalui established flows menghasilkan confirmed WBS dates baru.

Rules:

- Existing WBS invalidation/versioned request-cache contract tetap digunakan untuk View Group dan Edit Project.
- Older in-flight tree response tidak boleh mengembalikan summary ke stale confirmed values setelah mutation.
- Unconfirmed automatic schedule preview pada Edit Task tidak mengubah Group atau Project Summary.
- Unsaved Task draft tidak mengubah Group atau Project Summary.
- Hard browser refresh tidak boleh diperlukan setelah successful confirmed mutation. Bila Edit Project tetap terbuka setelah successful Save yang memicu scheduling, summary refreshes from the resulting confirmed WBS state.
- Story ini tidak memperluas mutation side effects; hanya existing confirmed WBS refresh behaviour yang digunakan.

### 5.13 Project Lifecycle

- View Group summary tersedia pada Open, Locked, dan Closed Project selama Project Structure dapat dibuka.
- Project Summary tersedia ketika Edit Project dibuka untuk Open, Locked, dan Closed Project.
- Summary selalu read-only untuk semua status; Project field edit permissions remain owned by US-3.1 and US-3.3.
- Locked dan Closed status tidak mengubah calculation rules.
- Closed Project summary menggunakan retained confirmed Task data dan tidak memicu scheduler.
- Story ini tidak mengubah Project visibility, Gantt exclusion, lifecycle transition, atau edit permissions.

### 5.14 UI Composition

#### View Group

View Group retains:

- Group Name as dialog heading;
- `Group` as supporting type label;
- `Close` action.

The obsolete informational alert is removed:

```text
This group contains <n> direct item items. Task details are managed on tasks inside this group.
```

#### Edit Project

- The existing Project Name and Project Settings remain in the combined Edit Project form.
- A semantic **Project Summary** region is shown after the editable/read-only Project fields and before form actions.
- The Project Summary is not submitted, validated as Project input, or reset from draft Project fields.
- Add Project does not show Project Summary.
- Edit Project uses the existing shared `Dialog` wide variant (`dialog-panel-wide` / `--container-dialog-wide`) rather than a new one-off width.
- The wide maximum is a desktop cap, not a fixed width. The dialog remains `width: 100%` inside the existing viewport padding and collapses naturally on narrow viewports.
- The nested OFF→ON confirmation remains the existing standard alert-dialog width unless its own content independently requires otherwise.

#### Shared Summary Sections

View Group and Project Summary display the same three semantic information groups and the same calculation/copy rules:

1. **Execution Timeline**
   - Aggregate date range or `Not scheduled`.
   - `<scheduled> of <total> tasks scheduled`.
2. **Commitment Timeline**
   - Aggregate date range or `Not scheduled`.
   - `<scheduled> of <total> tasks scheduled`.
3. **Effort Completion**
   - `<completed> of <total> hours completed (<percentage>%)`, or zero-known-effort state.
   - Task Without Effort disclosure when applicable.

At wide desktop size, the three sections may use a responsive multi-column arrangement to use available width. Reading order must remain Execution, Commitment, then Effort, and the layout must stack without horizontal scrolling when space is insufficient.

The sections may use existing cards, definition-list patterns, or another established read-only summary primitive. Feature code must reuse semantic tokens and existing shared primitives; it must not create one-off raw visual values or a competing card system.

### 5.15 Edit Project Summary Loading and Failure

Project list DTO does not carry all Task fields needed for separate Execution/Commitment coverage and Effort Completion. Project `startDate`/`endDate` are not an approved substitute because they cannot represent both timeline pairs or missing-Effort disclosure.

- Opening Edit Project reuses the existing confirmed WBS tree read contract for that Project.
- If a fresh confirmed tree is already cached, use it without a duplicate request.
- Otherwise the Edit Project form opens immediately and only the Project Summary region displays a shape-preserving local loading state.
- Summary loading must not delay access to Project fields or move focus away from the Project Name control.
- Summary read failure displays a local recoverable error and `Retry` within the summary region.
- Summary read failure does not erase Project form draft, close the dialog, or block Save/Cancel for otherwise valid Project fields.
- Retry performs only the summary read.
- A late older WBS response cannot replace a newer confirmed tree or summary.
- Loading and Retry do not perform scheduler, Project update, or other mutation requests.

### 5.16 Accessibility and Responsive Behaviour

- View Group and Edit Project use the existing shared Dialog focus contract.
- Edit Project opts into the existing wide variant; no new fixed pixel width or viewport overflow exception is allowed.
- Heading hierarchy, Project Summary region, and section labels are semantic.
- Values are not distinguished only by color.
- Screen readers can associate labels with values and supporting coverage/disclosure.
- Date range has accessible Start/End meaning.
- Long Project/Group names, large hour values, large Task counts, localized dates, loading errors, and Retry wrap without clipping.
- Summary does not cause uncontrolled horizontal scrolling on the minimum supported viewport.
- Form actions remain visible in logical reading order after the summary; Close/Cancel/Save remain keyboard accessible.
- Focus is returned to the trigger after closing. Loading or refreshing the summary does not steal focus.
- No live region is required on initial open; a user-triggered Retry result or confirmed external refresh must not silently preserve stale values.
- Automated accessibility checks use the existing axe/component-test convention, supplemented by keyboard and narrow-viewport acceptance evidence.

---

## 6. Calculation Examples

### 6.1 Nested Group with Partial Scheduling and Missing Effort

Input:

| Task | Execution | Commitment | Effort | Actual End |
| --- | --- | --- | ---: | --- |
| Task A | 2026-08-01 → 2026-08-03 | 2026-08-01 → 2026-08-05 | 16h | Set |
| Task B | 2026-08-04 → 2026-08-06 | 2026-08-04 → 2026-08-08 | 24h | Empty |
| Task C | Empty | 2026-08-09 → 2026-08-12 | Empty | Set |
| Task D | 2026-08-08 → 2026-08-12 | Empty | 8h | Set |
| Task E | Empty | Empty | Empty | Empty |

Expected:

```text
Execution Timeline
01 Aug 2026 – 12 Aug 2026
3 of 5 tasks scheduled

Commitment Timeline
01 Aug 2026 – 12 Aug 2026
3 of 5 tasks scheduled

Effort Completion
24 of 48 hours completed (50%)
2 tasks without effort are excluded from this calculation.
```

Explanation:

- Completed known effort = Task A 16h + Task D 8h = 24h.
- Total known effort = Task A 16h + Task B 24h + Task D 8h = 48h.
- Task C completed without Effort is not assigned invented effort.
- Task C and Task E are disclosed as Task Without Effort.

### 6.2 No Scheduled Tasks

```text
Execution Timeline
Not scheduled
0 of 3 tasks scheduled

Commitment Timeline
Not scheduled
0 of 3 tasks scheduled
```

No Project date or placeholder date is used.

### 6.3 Every Task Without Effort

```text
Effort Completion
Effort completion unavailable
3 tasks without effort are excluded from this calculation.
```

### 6.4 Reopen Completed Task

Before confirmed Reopen:

```text
16 of 24 hours completed (66.7%)
```

After confirmed Reopen of the completed 16-hour Task:

```text
0 of 24 hours completed (0%)
```

The Task remains in Total Known Effort because original Effort is preserved.

### 6.5 Project as WBS Level 0

Given a Project has two top-level Groups and one top-level Task, Project Summary traverses all three roots and every nested descendant Task. It renders the same Execution, Commitment, coverage, Effort Completion, and missing-Effort rules as a Group summary over an equivalent subtree.

A Project with no WBS nodes displays:

```text
No tasks are available for this project.
```

It does not derive the two timeline summaries from Project `startDate`/`endDate`.

---

## 7. Acceptance Criteria

### Recursive Aggregation

**AC-1**  
**Given** a Group contains direct Tasks and nested Groups at multiple levels  
**When** Engineering Lead opens View Group  
**Then** every descendant Task at every depth contributes to the applicable summary calculation.

**AC-2**  
**Given** a Group contains nested Groups  
**When** summary is calculated  
**Then** nested Groups themselves do not count as Tasks and do not contribute executable values directly.

**AC-3**  
**Given** child order changes without descendant Task data changing  
**When** View Group is reopened or refreshed  
**Then** all summary values remain identical.

### Execution Timeline

**AC-4**  
**Given** some descendant Tasks have complete Execution pairs  
**When** the applicable Group or Project summary is opened  
**Then** Execution Start is the earliest complete descendant Execution Start and Execution End is the latest complete descendant Execution End.

**AC-5**  
**Given** only 3 of 5 descendant Tasks have complete Execution pairs  
**When** Execution summary is rendered  
**Then** the available aggregate range is shown and coverage reads `3 of 5 tasks scheduled`.

**AC-6**  
**Given** no descendant Task has a complete Execution pair  
**When** Execution summary is rendered  
**Then** it displays `Not scheduled` and `0 of <total> tasks scheduled` without inventing a date.

### Commitment Timeline

**AC-7**  
**Given** some descendant Tasks have complete Commitment pairs  
**When** the applicable Group or Project summary is opened  
**Then** Commitment Start is the earliest complete descendant Commitment Start and Commitment End is the latest complete descendant Commitment End.

**AC-8**  
**Given** only 3 of 5 descendant Tasks have complete Commitment pairs  
**When** Commitment summary is rendered  
**Then** the available aggregate range is shown and coverage reads `3 of 5 tasks scheduled`.

**AC-9**  
**Given** no descendant Task has a complete Commitment pair  
**When** Commitment summary is rendered  
**Then** it displays `Not scheduled` and `0 of <total> tasks scheduled` without inventing a date.

**AC-10**  
**Given** Execution and Commitment coverage differ  
**When** the applicable Group or Project summary is rendered  
**Then** each timeline displays its own range and coverage; neither reuses the other timeline's count or dates.

### Effort Completion

**AC-11**  
**Given** descendant Tasks have Known Effort and some have Actual End  
**When** Effort Completion is calculated  
**Then** completed known effort is the sum of Effort only for Tasks with Actual End, and total known effort is the sum of all Tasks with Effort.

**AC-12**  
**Given** completed known effort is 24 hours and total known effort is 40 hours  
**When** the applicable Group or Project summary is rendered  
**Then** it displays `24 of 40 hours completed (60%)`.

**AC-13**  
**Given** percentage has a non-integer result  
**When** percentage is rendered  
**Then** it is calculated from integer minutes, rounded to at most one decimal place, and trailing `.0` is removed.

**AC-14**  
**Given** one or more descendant Tasks have no Effort  
**When** Effort Completion is rendered  
**Then** those Tasks are excluded from completed/total known effort and an accurate singular/plural disclosure is displayed.

**AC-15**  
**Given** a completed descendant Task has no Effort  
**When** Effort Completion is calculated  
**Then** no effort is invented, the Task does not enter numerator or denominator, and it remains included in Task Without Effort disclosure.

**AC-16**  
**Given** every descendant Task has no Effort  
**When** Effort Completion is rendered  
**Then** it displays `Effort completion unavailable`, displays missing-effort disclosure, and does not display a percentage or divide by zero.

**AC-17**  
**Given** a known-effort Task changes from unfinished to completed through confirmed Actual End  
**When** confirmed WBS state refreshes  
**Then** its Effort enters completed known effort and every open/reopened applicable Group or Project percentage updates without hard refresh.

**AC-18**  
**Given** a known-effort completed Task is successfully reopened  
**When** confirmed WBS state refreshes  
**Then** its Effort leaves completed known effort but remains in total known effort, and every open/reopened applicable Group or Project percentage updates without hard refresh.

### Projection, State, and Compatibility

**AC-19**  
**Given** a user edits an automatic Task and receives an unconfirmed schedule preview  
**When** a Group or Project Summary is observed before Save  
**Then** summary continues to use confirmed WBS data and does not reflect preview-only dates or draft Effort.

**AC-20**  
**Given** a confirmed descendant mutation or schedule recalculation changes dates, Effort, Actual End, or subtree membership  
**When** WBS confirmed state refreshes  
**Then** an open or subsequently opened View Group or Edit Project shows recomputed values without browser hard refresh.

**AC-21**  
**Given** an older WBS tree request resolves after a successful mutation and newer confirmed response  
**When** cache coordination completes  
**Then** the older response cannot restore stale Group or Project Summary values.

**AC-22**  
**Given** Engineering Lead opens View Group  
**When** no mutation is performed  
**Then** opening and closing the dialog sends no write request and persists no aggregate fields.

**AC-23**  
**Given** the Project is Open, Locked, or Closed  
**When** View Group is available  
**Then** the same summary calculations are read-only and no Project lifecycle or scheduler action is triggered.

### UX, Accessibility, and Defensive Behaviour

**AC-24**  
**Given** View Group is rendered  
**Then** the obsolete direct-item alert is absent and the dialog displays exactly the required Execution Timeline, Commitment Timeline, and Effort Completion information groups.

**AC-25**  
**Given** long localized dates, large Task counts, large hour values, or a narrow supported viewport  
**When** View Group is rendered  
**Then** content wraps without clipping or uncontrolled horizontal scrolling and Close remains usable.

**AC-26**  
**Given** a keyboard or screen-reader user opens View Group  
**When** they navigate the dialog  
**Then** heading, labels, values, coverage/disclosure, focus containment, Close action, and focus restoration satisfy the shared Dialog and accessibility contracts.

**AC-27**  
**Given** malformed or transient data marks a node as Group but contains no descendant Task  
**When** View Group is rendered  
**Then** a safe `No descendant tasks are available for this group.` state is shown without crash or misleading aggregate values.

### Project Summary and Wide Edit Form

**AC-28**  
**Given** Project is the logical WBS level `0` and has multiple top-level WBS roots  
**When** Engineering Lead opens Edit Project  
**Then** Project Summary recursively includes every Task in every root and nested descendant, using exactly the same timeline, coverage, effort, percentage, and missing-Effort rules as View Group.

**AC-29**  
**Given** Edit Project has loaded confirmed WBS data  
**When** Project Summary is rendered  
**Then** it displays Execution Timeline, Commitment Timeline, and Effort Completion in the same semantic order and with the same copy/precision contract as View Group.

**AC-30**  
**Given** a Project has no confirmed Task  
**When** Edit Project is opened  
**Then** Project Summary displays `No tasks are available for this project.` and does not display `0 of 0`, a percentage, or invented dates.

**AC-31**  
**Given** Edit Project opens without a fresh confirmed WBS tree in cache  
**When** summary data is being loaded  
**Then** the form opens immediately, Project fields remain usable according to lifecycle rules, focus remains on the form, and only the Project Summary region shows a local loading state.

**AC-32**  
**Given** Project Summary loading fails  
**When** Edit Project remains open  
**Then** a local recoverable error and Retry are shown without clearing Project draft, closing the dialog, or blocking valid Save/Cancel operations.

**AC-33**  
**Given** an older Project WBS response resolves after a newer confirmed response  
**When** Project Summary state coordinates  
**Then** the older response cannot restore stale summary values.

**AC-34**  
**Given** Engineering Lead opens Edit Project on a supported desktop viewport  
**Then** the dialog uses the existing shared wide variant and the Project fields plus summary are not constrained to the standard `28rem` dialog cap.

**AC-35**  
**Given** Edit Project is displayed on a narrow supported viewport  
**When** the wide dialog and summary sections reflow  
**Then** the dialog remains within viewport padding, summary sections stack in semantic order, no uncontrolled horizontal scrolling occurs, and form actions remain reachable.

**AC-36**  
**Given** Engineering Lead opens Add Project  
**Then** Project Summary is absent and the Add form is not required to use the wide variant.

**AC-37**  
**Given** Edit Project is Open, Locked, or Closed  
**When** Project Summary is displayed  
**Then** the summary remains read-only, follows the same calculation rules, and does not change existing field permissions or lifecycle behaviour.

**AC-38**  
**Given** Project `startDate` and `endDate` are present  
**When** Project Summary is calculated  
**Then** those generic Project fields are not used as a substitute for recursive Task Execution/Commitment pairs or Effort Completion.

---

## 8. Required Test Cases

### 8.1 Pure Aggregation Tests

- One direct Task.
- Multiple direct Tasks.
- Nested Groups at three or more levels.
- Mixed direct and deeply nested Tasks.
- Child reorder does not change result.
- Group executable payload is ignored.
- Execution earliest Start and latest End come from different Tasks.
- Commitment earliest Start and latest End come from different Tasks.
- Execution and Commitment coverage differ.
- Partially scheduled subtree.
- No Execution scheduled.
- No Commitment scheduled.
- All Tasks scheduled.
- Known Effort completed/unfinished mix.
- Completed Task without Effort.
- Unfinished Task without Effort.
- All Tasks without Effort.
- Half-hour totals.
- Percentage with integer output.
- Percentage with one-decimal output.
- Zero completed effort with positive total.
- Exact 100 percent.
- Defensive zero-descendant Group.
- Project-level aggregation across multiple top-level roots.
- Empty Project with zero Task.
- Project generic `startDate`/`endDate` ignored by summary calculation.
- Group and Project adapters produce identical result for the same Task set.

### 8.2 Component Tests

- View Group replaces obsolete direct-item alert.
- Dialog renders three required summary sections.
- Correct date-only formatting without timezone shift.
- Correct `3 of 5 tasks scheduled` copy per timeline.
- Correct hour totals and percentage.
- Singular Task Without Effort copy.
- Plural Task Without Effort copy.
- Zero-known-effort unavailable state.
- No write gateway method is called when View Group opens/closes.
- Summary is read-only for Open, Locked, and Closed Project.
- Unconfirmed Task preview does not alter Group or Project Summary.
- Confirmed Actual End updates summary.
- Confirmed Reopen updates summary.
- Confirmed structural change updates descendant scope.
- Stale response cannot restore old summary.
- Keyboard open/close and focus restoration.
- axe accessibility assertion.
- Narrow viewport and long-content containment.
- Edit Project displays the same three sections as View Group.
- Edit Project uses shared wide Dialog variant; Add Project remains summary-free.
- Project Summary local skeleton does not block form interaction or steal focus.
- Project Summary local error and Retry preserve draft and Save/Cancel behaviour.
- Wide-to-stacked responsive layout and form-action reachability.

### 8.3 Acceptance-Level Tests

At least one Project Structure workflow must prove recursively aggregated output from a realistic tree that includes:

- a direct Task;
- a nested Group with deeper Tasks;
- partial Execution scheduling;
- different Commitment coverage;
- completed and unfinished known Effort;
- one completed Task without Effort;
- one unfinished Task without Effort.

The workflow must observe the exact aggregate ranges, both coverage values, completed/total known effort, percentage, and missing-effort disclosure.

A second acceptance workflow must prove confirmed Actual End and Reopen Task transitions update the same open/reopened Group and owning Project Summary without hard refresh and that unconfirmed preview does not affect either.

A third acceptance workflow must open Edit Project for a realistic multi-root WBS, verify the exact Project Summary, wide desktop layout, narrow stacked layout, and empty Project state. It must also prove local summary loading/error/Retry does not block or clear the Project form and that an older response cannot restore stale values.

---

## 9. Three-Level Confidence Requirements

Every AC requires all three confidence levels defined by `AGENTS.md`:

1. Code Inspection.
2. Unit or Integration Test.
3. Acceptance-Level Test.

An implementation agent must maintain explicit AC traceability. Pure calculations should be proven at the lowest meaningful deterministic boundary, while observable View Group and Edit Project behaviour must also be proven through their component/acceptance workflows.

A broad test that opens the dialog but does not assert the exact recursive calculations is not sufficient acceptance evidence.

The implementation is not complete when:

- only snapshot/UI text tests exist;
- only a pure helper is tested without the actual View Group and Edit Project workflows;
- only the happy path exists without missing Effort and partial schedule cases;
- stale state or unconfirmed preview behaviour is unverified;
- local validation has not passed.

---

## 10. Architecture and Data Contract

### 10.1 Frontend Ownership

The current WBS tree response already contains every required descendant field:

- `children`;
- `hasChildren`;
- `effortMinutes`;
- `executionTimeline.start/end`;
- `commitmentTimeline.start/end`;
- `actualEnd`.

Therefore this story is implemented as a frontend read-model calculation over the confirmed WBS tree. View Group passes the selected subtree roots; Edit Project passes all Project top-level roots into the same aggregation helper.

- Use one pure, deterministic, typed aggregation helper.
- Keep calculation ownership within the WBS feature boundary and expose only an intentional context-free summary contract to Project presentation; Project feature must not import WBS presentation internals or copy the algorithm.
- Do not use `any` or bypass typing.
- Do not mutate `WBSNode` or descendant arrays during traversal.
- Avoid repeated full-tree scans inside render when one traversal can return all summary values.
- Complexity should be O(number of descendants) time and O(depth) traversal stack or equivalent iterative space.
- Do not introduce speculative generic reporting infrastructure.

### 10.2 Backend, API, and Persistence

No backend, API, migration, or summary-specific production query change is required because the existing per-Project WBS tree contract returns the complete roots/subtrees and source fields. Edit Project may perform or reuse that existing read; this is not permission to add an aggregate endpoint.

Implementation must stop and report a requirement/architecture mismatch instead of adding duplicate persisted aggregates if repository inspection shows the UI no longer receives a complete confirmed subtree.

Existing Project `startDate`/`endDate` remain untouched for their current API/list contract, but they are insufficient and forbidden as the source for the two separate summary timelines.

No new endpoint such as the following is approved:

```text
GET /api/projects/{projectId}/wbs/{groupId}/summary
```

A future scale-driven change may introduce backend projection only through a separately approved requirement that defines freshness, transaction consistency, API compatibility, query/index review, and cache behaviour.

### 10.3 Query Review

This story introduces no new production query shape. Edit Project reuses the existing WBS tree read that Project Structure already requires. The completion report must explicitly state whether that response was reused from cache or requested on open and that no summary-specific query-review gate was triggered.

If an implementation nevertheless changes a production query, it must stop, explain why the approved no-query design is insufficient, obtain approval, and complete the mandatory query-review gate before proceeding.

---

## 11. Requirement Compatibility and Contradiction Check

### 11.1 US-4.1 Manage WBS

No contradiction when the boundary is preserved:

- US-4.1 states Grouping WBS cannot own executable attributes.
- US-4.3 displays a read-only derived projection from descendant Task attributes.
- Group does not become executable and no value is persisted on Group.

US-4.1 must link Group and Project summary ownership to this story.

### 11.2 US-4.2 Reopen Completed Task

US-4.2 excludes persisted percentage complete and remaining effort. US-4.3 does not persist either value and does not estimate remaining effort.

Compatibility rule:

- Reopen changes Actual End only.
- Every ancestor Group and the owning Project Effort Completion recompute from confirmed Task data.
- Original Effort remains unchanged.

US-4.2 wording must clarify that its out-of-scope statement does not prohibit this derived read-only summary.

### 11.3 US-6.1 Automatic Scheduling

No contradiction:

- US-6.1 owns generation of Task Execution and Commitment dates.
- US-4.3 only aggregates confirmed generated/manual Task dates.
- US-4.3 does not calculate daily allocation or schedule Task dates.
- Unconfirmed schedule preview remains excluded.

### 11.4 US-3.1 Create Project and US-3.3 Configure Project Settings

No contradiction when the boundaries are explicit:

- Project is already defined as logical WBS level `0`; US-4.3 uses that existing hierarchy meaning and does not create a root WBS record.
- Project Summary is read-only for all Project statuses and is not part of the Project create/update payload.
- Add Project has no confirmed Task subtree and therefore does not display summary.
- Edit Project becomes wide through the existing shared Dialog variant; this changes composition only, not Project validation or mutation semantics.
- Locked baselines and Closed read-only restrictions remain unchanged.
- Opening/loading/retrying summary triggers no scheduler or lifecycle mutation.
- Existing Project `startDate`/`endDate` are not equivalent to the two required recursive timeline pairs and are not reused as summary inputs.

### 11.5 Forecast, Delivery Impact, and Health

No contradiction with deferred scope because these values are not displayed or calculated in this story.

---

## 12. Documentation Impact

This story requires synchronized updates to:

- `user_story/US-3.1-create-project.md` for Project level `0`, Edit Project composition, and summary ownership.
- `user_story/US-3.3-configure-project-settings.md` for the wide Edit Project form and non-blocking read-only summary region.
- `user_story/US-4.1-manage-wbs.md` for Group/Project summary ownership and the Group executable-attribute boundary.
- `user_story/US-4.2-reopen-completed-task.md` to distinguish persisted percentage/remaining effort from this derived summary and to define refresh impact after Reopen.
- `docs/project/schedmind-context.md` to include the new authoritative story and summary invariant.
- `docs/project/architecture.md` to record the shared frontend recursive projection, existing WBS read reuse from Edit Project, wide Dialog composition, no new API/query/migration, and confirmed-state refresh behaviour.

Do not update reusable `docs/frontend-design-system.md` because this is product-specific summary content, not a new reusable design-system standard.

Do not update `AGENTS.md`; the applicable architecture, UX, documentation, and Three-Level Confidence rules already exist.

---

## 13. Agent Execution Rules

Before implementation, the agent must inspect completely:

- `AGENTS.md`.
- `docs/architecture.md`.
- `docs/architecture/frontend.md`.
- `docs/frontend-design-system.md`.
- `docs/project/README.md`.
- `docs/project/architecture.md`.
- `docs/project/schedmind-context.md`.
- US-3.1, US-3.3, US-4.1, US-4.2, and US-6.1.
- Existing Project form/page/gateway and tests.
- Existing WBS frontend domain, gateway, Project Structure panel, View Group dialog, request cache, date-only formatter, shared Dialog wide variant, and relevant tests.

The agent must:

- preserve the single WBS model;
- implement one deterministic recursive summary from confirmed descendant Tasks for both Group and Project;
- keep Group read-only and free of executable state, and keep Project Summary outside the update payload;
- reuse the existing WBS tree data/read contract and cache coordination;
- use the shared wide Dialog variant for Edit Project without changing Add Project unnecessarily;
- prevent stale and preview-only values from entering confirmed summary;
- use integer minutes for effort arithmetic;
- prove every AC with all three confidence levels;
- run the repository's complete relevant validation commands;
- stop at the first unmet AC or mandatory validation failure.

The agent must not:

- add a backend endpoint, migration, or persisted Group/Project aggregate without approval;
- count nested Groups as Tasks;
- calculate only direct children;
- treat missing Effort as zero;
- infer completion from dates other than Actual End;
- use Task count as the completion percentage;
- mutate or sort the source WBS tree during aggregation;
- reflect unconfirmed Task preview in Group or Project Summary;
- use Project `startDate`/`endDate` as a shortcut for separate recursive timeline summaries;
- duplicate the aggregation algorithm in the Project feature;
- add Forecast, Health, Delivery Impact, structure count, or dependency metrics;
- claim completion from helper tests alone.

---

## 14. Unresolved Questions

None.
