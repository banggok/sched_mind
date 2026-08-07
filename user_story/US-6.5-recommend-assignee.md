# US-6.5 — Recommend Assignee by Simulated Completion

> **Product decision update — US-4.4:** Recommendation eligibility and simulation mode resolve from effective Task lifecycle and Group/Project scheduling configuration. Effective Automatic Scheduling ON uses the concrete scheduler with the effective Group/Project Scheduling Start Date; effective OFF uses the existing advisory manual simulation. Effectively Locked Tasks are not recommendable.

> **Authority:** This story owns Assignee recommendation for unfinished
> executable Tasks, including candidate eligibility, batch simulation, ranking,
> recommendation metadata, freshness, fallback behaviour, and Task-form field
> placement. It applies when an Engineering Lead chooses the first Assignee for
> a newly created Task and when an existing Task is edited.
>
> Automatic Scheduling `ON` must reuse the concrete scheduler owned by US-6.1
> and US-6.3 in rollback-only simulation mode. This story does not define a
> second automatic scheduling algorithm. Automatic Scheduling `OFF` uses the
> advisory simulation contract defined here because manual timelines remain
> authoritative and the normal scheduler does not generate Task dates.
>
> This story also revises US-6.3: Task Capacity Allocation Percentage is a Task
> attribute and is preserved when Assignee is selected, changed, or cleared. An
> Assignee change no longer resets the percentage to `100`.

## 1. User Story

**Sebagai** Engineering Lead,

**Saya ingin** daftar Assignee diurutkan berdasarkan simulasi siapa yang dapat
menyelesaikan Task lebih cepat,

**Sehingga** saya dapat memilih Assignee dengan informasi capacity yang relevan
tanpa mencoba setiap Member satu per satu atau mengubah schedule terlebih dahulu.

---

## 2. Business Context

Urutan alfabetis tidak membantu user memahami dampak assignment terhadap
completion date. SchedMind sudah memiliki sumber kebenaran untuk Project
Priority, WBS order, dependencies, daily capacity, overrides, holidays, buffers,
Task Capacity Allocation Percentage, fixed manual allocation, completed Actual
Allocation, dan Locked baseline. Assignee recommendation harus menggunakan
sumber yang sama agar urutan kandidat tidak berbeda dari schedule aktual.

Recommendation bersifat **advisory**:

- recommendation tidak melakukan reservation;
- recommendation tidak mengubah Task atau Project lain;
- recommendation tidak menjamin bahwa hasil Save akan identik bila confirmed
  portfolio state berubah setelah simulasi;
- scheduler dan mutation contract saat Save tetap menjadi sumber kebenaran;
- dampak terhadap lower-priority Task atau Project tidak menjadi objective
  ranking. First priority tetap first served.

Ranking utama adalah completion Task yang sedang dinilai. Bila completion date
sama, kandidat yang menyisakan lebih banyak Execution Capacity pada completion
Date ditempatkan lebih dahulu. Kandidat yang menambah overcapacity ditempatkan
setelah kandidat yang tidak menambah overcapacity, walaupun kandidat tersebut
mungkin selesai lebih cepat.

---

## 3. Scope

### 3.1 In Scope

- Assignee recommendation pada Task Details untuk:
  - first assignment pada Task baru yang sudah memiliki confirmed executable
    WBS identity;
  - Edit Task pada unfinished executable WBS.
- Open Projects dengan Automatic Scheduling `ON` atau `OFF`.
- Candidate filtering menggunakan Role contract existing.
- One batch request untuk seluruh candidate.
- Automatic `ON` rollback-only simulation menggunakan concrete scheduler aktual.
- Automatic `OFF` advisory simulation dengan anchor hierarchy dan manual-date
  rules pada story ini.
- Mengeluarkan Task yang sedang diedit dari baseline sebelum setiap candidate
  simulation untuk mencegah double allocation.
- Draft Effort, Lag, Capacity Allocation Percentage, Role, manual Execution
  Start, dan candidate Assignee sebagai input simulasi.
- Execution End sebagai completion metric.
- Remaining Execution Capacity pada simulated completion Date sebagai secondary
  metric.
- Incremental overcapacity terhadap baseline sebagai feasibility grouping.
- Deterministic name/ID tie-breaker.
- Metadata recommendation di setiap Assignee option.
- Latest-state request sebelum user memilih Assignee.
- Stale-response protection, loading, fallback, error recovery, accessibility,
  API contract, query review, dan Three-Level Confidence evidence.
- Menempatkan Role dan Assignee langsung setelah Lag pada Task form, dalam
  urutan Role lalu Assignee, sebelum timeline dan dependency controls.
- Mempertahankan Task Capacity Allocation Percentage ketika Assignee dipilih,
  diganti, atau dikosongkan.

### 3.2 Out of Scope

- Mengubah Project Priority, WBS order, dependency, capacity configuration, Task
  dates, allocation, atau schedule version dari recommendation command.
- Mengoptimalkan total portfolio completion, jumlah Task yang tergeser, fairness,
  workload balancing jangka panjang, atau konteks personal Member.
- Menilai skill, seniority, cost, preference, utilization target, velocity,
  historical performance, atau availability di luar capacity source existing.
- Reserving candidate capacity setelah user melihat atau memilih recommendation.
- Menjamin recommendation tetap valid setelah confirmed portfolio state berubah.
- Mengganti scheduler aktual untuk Automatic Scheduling `ON`.
- Mengubah manual Execution/Commitment dates pada Automatic Scheduling `OFF`.
- Menggunakan manual Execution End sebagai completion hasil simulasi.
- Warning ketika advisory Estimated Finish berbeda dari manual Execution End.
- Recommendation untuk completed Task, Locked Project, Closed Project, atau Group.
- Multi-assignee Task.
- Custom user-defined ranking weight.
- Recommendation berdasarkan Commitment End, Forecast, Sprint, atau Actual End.

---

## 4. Terminology

| Term | Definition |
| --- | --- |
| Candidate | Active Member yang dapat ditampilkan pada Assignee control menurut Role dan lifecycle rules |
| Eligible Candidate | Candidate yang sesuai dengan selected Role dan dapat disimulasikan sebagai Assignee |
| Current Mismatched Assignee | Assignee existing yang tidak lagi sesuai selected Role; tetap terlihat untuk mencegah implicit data loss tetapi tidak ikut ranking normal |
| Recommendation Baseline | Confirmed portfolio scheduling state setelah allocation/projection Task yang sedang diedit dikeluarkan |
| Candidate Simulation | Rollback-only calculation yang menerapkan draft Task ke satu candidate di atas baseline yang sama |
| Projected Finish | Simulated Execution End dari concrete scheduler ketika Automatic Scheduling `ON` |
| Estimated Finish | Advisory simulated Execution End ketika Automatic Scheduling `OFF` |
| Candidate Remaining Execution Capacity | Execution Capacity tersisa pada candidate completion Date segera setelah higher-priority/fixed work dan candidate Task dialokasikan, sebelum lower-priority work menggunakan sisa tersebut |
| Incremental Overcapacity | Tambahan Execution overcapacity candidate Member/Date dibanding Recommendation Baseline |
| Feasible Candidate | Candidate dengan completion dan zero Incremental Overcapacity |
| Overcapacity Candidate | Candidate dengan completion tetapi positive Incremental Overcapacity |
| No Projected Completion | Candidate yang tidak menghasilkan simulated Execution End dalam scheduler horizon atau gagal karena candidate-specific schedulability |
| Advisory Anchor | Date-only start basis untuk Automatic Scheduling `OFF`; tidak dipersist sebagai Project atau Task scheduling anchor |
| Recommendation Snapshot | Satu confirmed schedule/capacity/version basis yang digunakan bersama oleh seluruh candidate dalam satu batch |

---

## 5. Applicability and Candidate Contract

### 5.1 Task Lifecycle

Recommendation tersedia hanya ketika:

- owning Project berstatus `Open`;
- WBS adalah unfinished executable Task;
- Task form writable;
- user sedang memilih Assignee pertama atau mengedit Assignee existing.

Recommendation tidak tersedia untuk:

- Grouping WBS;
- completed Task sebelum Reopen;
- Locked Project;
- Closed Project.

Read-only states tetap menampilkan confirmed Assignee tanpa recommendation
request.

### 5.2 Create and Edit Coverage

- Recommendation berlaku pada first assignment untuk Task baru dan reassignment
  pada Task existing.
- Current WBS create flow boleh tetap membuat Name-only executable WBS terlebih
  dahulu. Ketika confirmed Task identity tersedia dan Task Details dibuka untuk
  first assignment, workflow tersebut dihitung sebagai Create coverage untuk
  story ini.
- Recommendation tidak memerlukan duplicate Task aggregate atau speculative
  persisted Task.
- Bila implementasi memilih virtual pre-persist create draft, virtual Task harus
  menggunakan parent dan deterministic WBS position yang sama dengan hasil Save;
  virtual simulation tidak boleh menghasilkan ordering yang berbeda.

### 5.3 Role Filtering

- `No assignee` selalu menjadi option pertama dan tidak ikut ranking.
- Bila Role dipilih, normal candidates adalah active Members dengan Role tersebut.
- Bila Role belum dipilih, recommendation tidak dijalankan; candidate list
  mengikuti existing Role/Assignee behaviour dan diurutkan alfabetis.
- Selecting an Assignee may continue to set its Role according to US-4.1, but
  recommendation requires a valid selected Role before simulation.
- Role change tidak boleh menghapus current Assignee secara implisit.
- Bila current Assignee tidak sesuai Role baru:
  - current Assignee tetap terlihat dengan `Does not match selected Role`;
  - current Assignee ditempatkan setelah eligible candidates dan candidates
    tanpa completion;
  - current Assignee tidak dianggap recommendation;
  - Save tetap mengikuti existing backend mismatch validation dan mengharuskan
    user mengganti atau mengosongkan Assignee.

### 5.4 Duplicate Names and Stable Order

Member Name tidak unique. Setiap final tie menggunakan:

1. normalized case-insensitive Member Name ascending;
2. stable Member ID ascending.

UI harus menampilkan cukup Role/context untuk membedakan Member dengan nama sama
bila diperlukan oleh existing Member option contract.

---

## 6. Required Draft Inputs

Recommendation request hanya dijalankan ketika seluruh input berikut valid:

- Role selected dan valid;
- Effort valid menurut US-4.1;
- Capacity Allocation Percentage valid `1–100` menurut US-6.3;
- Lag valid menurut US-6.1;
- Project identity dan confirmed WBS identity/position tersedia;
- candidate Member projection berhasil dimuat.

Manual Execution Start boleh kosong karena Automatic Scheduling `OFF` memiliki
fallback anchor. Manual Execution End tidak diperlukan untuk recommendation.

Bila required input missing atau invalid:

- jangan menjalankan recommendation simulation;
- jangan menampilkan Projected/Estimated Finish palsu;
- eligible candidates ditampilkan alfabetis;
- selected Assignee tetap dipertahankan;
- helper text menyebut field yang harus dilengkapi, misalnya
  `Complete Role, Effort, Capacity Allocation, and Lag to calculate recommendations.`

Input validation tetap dijalankan pada frontend dan backend; frontend condition
bukan satu-satunya protection.

---

## 7. Shared Baseline Rules

### 7.1 Remove the Edited Task First

Untuk Automatic Scheduling `ON` dan `OFF`, setiap batch harus:

1. membaca confirmed portfolio state terbaru;
2. mengeluarkan seluruh planned Execution/Commitment allocation, generated/manual
   projection, dan candidate-specific capacity consumption milik Task yang sedang
   diedit dari Recommendation Baseline;
3. mempertahankan Task identity, Project/WBS order, explicit dependency endpoint,
   completed history, Locked anchors, dan data lain yang bukan output allocation
   Task tersebut;
4. menerapkan draft Task terhadap satu candidate;
5. menjalankan candidate simulation;
6. mengembalikan hasil tanpa persistence.

Tujuannya adalah mencegah current Assignee terlihat lebih sibuk karena Task yang
sama dihitung sebagai existing work sekaligus draft work.

Untuk first assignment pada Task baru yang belum memiliki allocation, removal
adalah no-op.

### 7.2 One Shared Snapshot

- Seluruh candidate dalam satu batch menggunakan Recommendation Snapshot yang sama.
- Backend tidak boleh menjalankan satu HTTP request terpisah per candidate.
- Confirmed state tidak boleh dibaca ulang dengan version yang berbeda di tengah
  batch.
- Candidate simulation boleh dijalankan berurutan atau paralel secara internal
  hanya bila hasil deterministic, transaction-safe, dan menggunakan snapshot
  yang identik.
- Recommendation response menyertakan snapshot/version identity dan backend
  calculation Date agar stale result dapat ditolak frontend.

### 7.3 No Side Effects

Recommendation command tidak boleh:

- persist draft Role, Assignee, Effort, Lag, percentage, atau dates;
- persist generated dates atau daily allocations;
- create/delete/retarget dependency;
- change Project dates, snapshots, status, priority, atau schedule version;
- trigger impact confirmation;
- invalidate confirmed cache as if mutation succeeded;
- reserve capacity;
- emit a success event representing confirmed Task change.

Rollback-only failure must leave all confirmed state unchanged.

---

## 8. Automatic Scheduling ON Simulation

### 8.1 Reuse the Concrete Scheduler

Automatic Scheduling `ON` recommendation must:

- apply candidate Assignee and current valid draft fields;
- execute the concrete portfolio scheduler owned by US-6.1 and US-6.3;
- use the same dependency readiness, Lag, Project Priority, visual WBS order,
  capacity resolution, buffers, Task Daily Limit, fixed allocations, completed
  Actual Allocation, Locked anchors, affected scope, rounding, and horizon;
- use rollback-only semantics equivalent to the established Task schedule preview;
- read the candidate Task's simulated Execution End and ranking metadata;
- roll back every temporary change.

This story must not copy, simplify, approximate, or independently reimplement
the automatic scheduler.

### 8.2 Missing Automatic Anchor

Scheduling Start Date behaviour remains owned by US-3.3, US-4.4, and US-6.1.

- Recommendation must not invent `today` or another anchor when effective
  Automatic Scheduling is `ON`.
- Bila concrete scheduler tidak menghasilkan Execution End karena effective
  Scheduling Start Date atau scheduler prerequisite lain tidak tersedia,
  recommendation ranking tidak dijalankan for that draft.
- Candidate list remains usable alphabetically and the UI explains the existing
  unscheduled prerequisite.

### 8.3 Priority Impact

Candidate Task participates at its actual Project Priority and visual WBS order.
The scheduler may move lower-priority mutable automatic work inside simulation
exactly as it would for a confirmed mutation.

Ranking does not penalize:

- number of lower-priority Tasks moved;
- amount of downstream delay;
- lower-priority Project completion changes.

First priority remains first served. Lower-priority impact is not a ranking
objective.

---

## 9. Automatic Scheduling OFF Advisory Simulation

### 9.1 Manual Dates Stay Authoritative

When Automatic Scheduling is `OFF`:

- confirmed/draft manual Execution and Commitment dates remain unchanged;
- recommendation does not write or preview replacement manual dates;
- manual Execution End is not used as the candidate completion result;
- no warning is shown when Estimated Finish is later or earlier than manual
  Execution End;
- Estimated Finish exists only to compare candidates.

### 9.2 Advisory Anchor Hierarchy

The earliest advisory basis is resolved as follows:

1. If draft Manual Execution Start exists, use it as the Task start lower bound.
2. Otherwise, if Project Scheduling Start Date exists, use the later of:
   - Project Scheduling Start Date;
   - `today` in `APP_TIMEZONE`.
3. Otherwise use `today` in `APP_TIMEZONE`.

Rules:

- `today` is calculated by the backend, not supplied as authoritative browser
  input.
- All candidates in one batch use the same resolved calendar Date.
- The Date is advisory only and is never persisted as Project Scheduling Start
  Date or Task date.
- A recommendation recalculated on a later application Date may validly produce
  a different order without any Task mutation.
- If the Assignee control remains open across midnight, its displayed order is
  frozen until the control closes and a new batch is requested.

### 9.3 Manual Advisory Allocation

The advisory simulation must reuse existing scheduling primitives and
precedence rather than estimate `Effort / Daily Capacity` naively. It must
consider:

- explicit dependency readiness and Lag;
- candidate's actual Project Priority and WBS order;
- existing fixed manual allocations;
- higher-priority mutable/fixed capacity consumption;
- completed Actual Allocation;
- Locked baseline reservation;
- Public Holiday;
- minimum active Capacity Override;
- Member Daily Capacity and Member Buffer;
- Task Capacity Allocation Percentage and `0.5h` rounding;
- working Date and same-Date remaining-capacity rules;
- the scheduler horizon/configuration already used by the scheduling engine.

Candidate Task is placed as advisory Execution work beginning no earlier than
both its resolved advisory anchor and dependency readiness after Lag. Existing
manual ranges remain fixed and are not moved. Lower-priority impact is ignored
as an optimization objective.

Commitment dates/capacity are not ranking inputs.

### 9.4 No Project Start in the Past Bias

When no Manual Execution Start exists and Project Scheduling Start Date is in
the past, `today` wins through the `max(Project Start, today)` rule. A new manual
assignment must not appear able to finish in the past merely because the Project
anchor is old.

When Manual Execution Start is explicitly present, its date remains the lower
bound even when it is before today because the user has locked the manual
Execution range.

---

## 10. Capacity Allocation Preservation

Task Capacity Allocation Percentage is owned by the Task, not by an Assignee.
Therefore:

- first selecting an Assignee preserves the current draft percentage;
- changing Assignee preserves the current draft/persisted percentage;
- clearing Assignee preserves the current draft/persisted percentage;
- Role change that makes current Assignee mismatched also preserves percentage;
- legacy update omission preserves existing percentage even when Assignee changes;
- explicit valid percentage continues to replace the existing value;
- Add Task still starts at default `100` only because that is the Task default,
  not because an Assignee was selected.

Every candidate in a batch is simulated using the same visible draft percentage.
The system must not simulate non-current candidates at an implicit `100%`.

---

## 11. Ranking Metrics

### 11.1 Completion Metric

- Automatic Scheduling `ON`: use simulated **Execution End** and label it as
  Projected/`finishes`.
- Automatic Scheduling `OFF`: use advisory simulated **Execution End** and label
  it as `estimated`.
- Commitment End, manual Execution End, Sprint End, Forecast End, and Actual End
  do not affect ranking.

Earlier calendar Date is better.

### 11.2 Candidate Remaining Execution Capacity

For each candidate with completion:

```text
Candidate Remaining Execution Capacity
= Candidate's resolved Execution Capacity on simulated completion Date
  - higher-priority/fixed Execution consumption applicable before candidate
  - candidate Task allocation on that Date
```

Rules:

- measure immediately after candidate allocation;
- do not subtract lower-priority work that may consume the remaining capacity
  later in the full simulation;
- use hours, not percentage;
- use exact scheduling arithmetic and return display-safe decimal hours;
- larger remaining capacity is better when completion Date is equal;
- remaining capacity may be negative only for an Overcapacity Candidate.

This metric answers how much candidate capacity remains after completing the
Task at its scheduling turn, not final portfolio idle capacity after every
lower-priority Task is allocated.

### 11.3 Incremental Overcapacity

Baseline overcapacity must not automatically disqualify a candidate.

For candidate Member and each affected Execution Date:

```text
Date Incremental Overcapacity
= max(0, Simulated Overcapacity - Baseline Overcapacity)

Total Incremental Overcapacity
= sum(Date Incremental Overcapacity)
```

Where Recommendation Baseline is calculated after the edited Task's own
allocation is removed.

Rules:

- `0h` means Feasible Candidate;
- positive value means Overcapacity Candidate;
- existing overcapacity that candidate does not worsen is ignored;
- reducing overcapacity on one Date does not cancel new overcapacity on another
  Date;
- overcapacity caused only by downstream lower-priority changes on another
  Member is not a ranking input;
- UI may display total added hours, but magnitude is not an extra ordering key
  unless a future requirement approves it.

---

## 12. Deterministic Ranking

`No assignee` remains first and is outside ranking.

Normal candidates are ordered in these groups:

### Group 1 — Feasible Candidates

Candidates with simulated Execution End and `0h` Incremental Overcapacity:

1. earliest simulated Execution End;
2. largest Candidate Remaining Execution Capacity on completion Date;
3. normalized Member Name ascending;
4. Member ID ascending.

### Group 2 — Overcapacity Candidates

Candidates with simulated Execution End and positive Incremental Overcapacity:

1. earliest simulated Execution End;
2. largest Candidate Remaining Execution Capacity on completion Date;
3. normalized Member Name ascending;
4. Member ID ascending.

Every Group 1 candidate appears before every Group 2 candidate, even when a
Group 2 candidate has an earlier completion Date.

### Group 3 — No Projected Completion

Candidates without simulated Execution End:

1. normalized Member Name ascending;
2. Member ID ascending.

### Group 4 — Current Mismatched Assignee

Current Assignee that does not match selected Role is displayed last with a
warning and is not treated as a normal candidate.

No database row order, API iteration order, or frontend object order may become
an implicit tie-breaker.

---

## 13. No Projected Completion and Candidate Failure

A candidate may return `No projected completion` when:

- no positive Execution Capacity is found inside the existing scheduler horizon;
- capacity configuration is invalid or cannot be resolved;
- dependency/cycle state prevents candidate scheduling;
- candidate Member becomes unavailable during the request;
- the concrete scheduler produces an unscheduled reason;
- candidate-specific simulation fails without invalidating the whole confirmed
  portfolio state.

Rules:

- do not invent a completion Date;
- keep candidate selectable when existing Task validation permits it;
- place candidate in Group 3;
- expose a stable reason code for diagnostics/UI mapping;
- do not expose infrastructure details;
- one candidate failure should not discard successful results for other
  candidates unless the shared snapshot itself is invalid.

If the entire batch cannot be calculated, use the failure fallback in Section 17.

---

## 14. Task Form and Interaction

### 14.1 Field Placement

For editable unfinished Task planning fields:

- the visible and accessible order is Effort, Capacity Allocation Percentage,
  Lag, Role, then Assignee;
- Role appears immediately after Lag and immediately before Assignee;
- applicable manual/generated timeline inputs and the confirmed dependency editor
  appear after Role/Assignee;
- Actual Date remains a separate lifecycle action and may remain after the primary
  Save action; read-only allocation/history sections may retain their established
  composition.

Moving fields must preserve responsive layout, label association, tab order,
visible focus, and existing shared form primitives.

### 14.2 Option Labels

Examples for Automatic Scheduling `ON`:

```text
No assignee
Dewi — finishes 11 Aug 2026, 4h remaining
Tony — finishes 11 Aug 2026, 2h remaining
Alif — finishes 12 Aug 2026, adds 2h overcapacity
Gabby — no projected completion
```

Examples for Automatic Scheduling `OFF`:

```text
No assignee
Dewi — estimated 11 Aug 2026, 4h remaining
Tony — estimated 11 Aug 2026, 2h remaining
```

Mismatched current Assignee:

```text
Gabby — does not match selected Role
```

Rules:

- use shared date-only display format;
- remaining/overcapacity meaning must not rely on colour alone;
- long labels must remain understandable with keyboard and screen reader;
- selected value must remain stable when a new ranking arrives;
- recommendation metadata must not replace the Member's accessible name.

### 14.3 Incomplete Input State

When recommendation prerequisites are incomplete:

- show alphabetic candidate ordering;
- show no completion/capacity suffix;
- explain missing inputs;
- keep Assignee selection usable under existing validation.

### 14.4 Recalculation Triggers

Recommendation becomes stale when any of the following changes:

- Role;
- Effort;
- Capacity Allocation Percentage;
- Lag;
- Manual Execution Start;
- owning Project settings/status/version;
- confirmed dependency or WBS ordering;
- capacity/holiday/override/buffer state;
- confirmed scheduling mutation affecting the snapshot;
- application Date in `APP_TIMEZONE` for a manual fallback-anchor case.

Frontend may prefetch/debounce after valid draft changes, but immediately before
options become selectable it must ensure one latest batch request exists for the
current draft.

---

## 15. Freshness and Concurrency

### 15.1 Latest Result Before Selection

When user focuses, opens, or otherwise invokes the Assignee control:

1. mark any result based on older draft/snapshot as stale;
2. request one latest batch when no current result exists;
3. show `Calculating recommendations…` while the latest result is pending;
4. prevent selection from stale recommendation rows while pending;
5. enable ranked options only after the latest accepted response arrives;
6. if the request fails, enable alphabetic fallback options.

The implementation may prefetch earlier, but must not claim freshness from a
result invalidated by a newer draft or confirmed mutation.

### 15.2 Stable Open Interaction

- Once the option list is open, ordering is frozen for that open interaction.
- A result arriving while open may be held until the list closes.
- An older response resolving after a newer request is ignored.
- Closing/reopening may apply the newest accepted result.
- Crossing midnight does not reorder an already-open list.

### 15.3 Save Remains Authoritative

- Recommendation snapshot is not a reservation or confirmation token for Save.
- Save follows the existing latest-state mutation, impact, scheduler, locking,
  validation, and rollback rules.
- Save is not rejected solely because recommendation is older.
- Generated dates after Save may differ if portfolio state changed after
  recommendation.
- A normal Save conflict or scheduling failure preserves the Task draft according
  to existing Task mutation behaviour.

---

## 16. API Contract

Implementation extends the WBS Task boundary and must not create a separate
Assignee recommendation aggregate.

Recommended batch endpoint:

```text
POST /api/projects/{projectId}/wbs/{wbsId}/assignee-recommendations
```

The Task ID may represent an existing Task or a newly confirmed Name-only Task
before first assignment.

Minimum request fields:

```json
{
  "roleId": "role-id",
  "effortHours": 8,
  "capacityAllocationPercentage": 50,
  "lag": 0,
  "executionStart": "2026-08-10"
}
```

Rules:

- `executionStart` is optional and relevant to Automatic Scheduling `OFF`;
- `executionEnd` may be carried as Task draft data but must not affect ranking;
- candidate IDs are resolved by backend from current Role/Member state;
- browser-supplied `today` is not accepted as source of truth;
- request must not send one payload per candidate;
- unknown extra scheduling fields are rejected or ignored according to existing
  strict DTO convention, but must not silently influence ranking.

Minimum response shape:

```json
{
  "calculatedOnDate": "2026-08-05",
  "snapshot": {
    "projectScheduleVersions": {
      "project-id": 12
    }
  },
  "mode": "automatic",
  "items": [
    {
      "memberId": "member-id",
      "memberName": "Dewi",
      "roleId": "role-id",
      "rankGroup": "feasible",
      "executionEnd": "2026-08-11",
      "remainingExecutionCapacityHours": 4,
      "incrementalOvercapacityHours": 0,
      "reasonCode": null
    }
  ]
}
```

Allowed `mode` values:

- `automatic`;
- `manual-advisory`.

Allowed `rankGroup` values:

- `feasible`;
- `overcapacity`;
- `no-completion`.

Backend returns already-ranked deterministic items. Frontend must not derive a
conflicting rank from display strings.

Minimum stable errors:

| Error Code | HTTP | Condition |
| --- | ---: | --- |
| `ASSIGNEE_RECOMMENDATION_INPUT_INVALID` | 400 | Role, Effort, Lag, percentage, or Task context invalid |
| `TASK_NOT_RECOMMENDABLE` | 409 | Task is Group, completed, or owning Project is not Open |
| `ASSIGNEE_RECOMMENDATION_STALE` | 409 | Shared snapshot cannot be established consistently |
| `ASSIGNEE_RECOMMENDATION_UNAVAILABLE` | 503 | Batch simulation dependency unavailable; fallback remains allowed |

Candidate-specific no-completion is returned as an item reason, not a whole
request error.

---

## 17. Failure and Recovery

Recommendation is advisory and must not make Task editing unavailable.

If candidate/member loading fails:

- show recoverable error and Retry;
- preserve Task draft and selected Assignee;
- do not clear percentage or other fields.

If batch simulation fails:

- show `Assignee recommendation is temporarily unavailable.`;
- enable existing eligible candidates in deterministic alphabetic order;
- keep `No assignee` first;
- keep current mismatched Assignee visible when applicable;
- keep Save available subject to normal Task validation;
- Retry requests the latest batch rather than replaying a stale snapshot.

Duplicate pending requests for the same draft should be coalesced or cancelled.
No failure path may persist partial candidate simulation state.

---

## 18. Query, Performance, and Transaction Requirements

- One user interaction produces at most one candidate batch request.
- Candidate/member lookup must use bounded indexed Role/active-state access and
  must not issue one database query per Member.
- Scheduling inputs, allocation rows, dependency graph, capacity overrides,
  holidays, and Project versions must be loaded in bounded sets consistent with
  the existing scheduler query strategy.
- Internal per-candidate work must reuse the shared Recommendation Snapshot.
- A production query added or changed for this feature must pass the repository
  query-review gate in `docs/architecture/backend.md`.
- Automatic simulation runs inside existing scheduling serialization and
  transaction boundaries with deliberate rollback.
- Manual advisory simulation must not hold locks longer than necessary and must
  preserve deterministic results under concurrent confirmed mutations.
- Candidate count and scheduler horizon must remain bounded by existing product
  limits; do not introduce an unbounded date scan.
- Performance optimization must not replace exact scheduling semantics with an
  approximation unless a future requirement explicitly approves it.

---

## 19. Acceptance Criteria

### AC-1 — Recommendation is available for first assignment and edit

**Given** an unfinished executable Task in an Open Project
**When** user chooses its first Assignee or edits an existing Assignee
**Then** Assignee control can request ranked candidates
**And** Name-only Task creation remains valid before first assignment.

### AC-2 — Role and Assignee immediately follow Lag

**When** editable Task Details is rendered
**Then** Effort and Capacity Allocation appear before Lag
**And** Role appears immediately after Lag
**And** Assignee immediately follows Role
**And** applicable timeline and confirmed dependency controls appear after Assignee
**And** accessible tab order follows the visible planning order.

### AC-3 — Required draft data gates simulation

**Given** Role, Effort, Capacity Allocation, or Lag is missing/invalid
**When** Task form is used
**Then** recommendation API is not called
**And** candidates are alphabetical without fake completion metadata
**And** helper text identifies missing prerequisites.

### AC-4 — Edited Task allocation is removed for Automatic ON

**Given** Task already has confirmed automatic allocation for current Assignee
**When** recommendation batch is calculated
**Then** its old allocation is removed before every candidate simulation
**And** current Assignee is not penalized by double allocation.

### AC-5 — Edited Task allocation is removed for Automatic OFF

**Given** Task already has fixed manual allocation
**When** recommendation batch is calculated
**Then** its own fixed allocation is removed from baseline
**And** other fixed/manual/completed/Locked allocations remain authoritative.

### AC-6 — Automatic ON uses concrete scheduler

**Given** Automatic Scheduling is `ON`
**When** candidate is simulated
**Then** candidate result follows the same concrete scheduler rules as confirmed
Save
**And** no separate simplified algorithm is used
**And** all temporary state is rolled back.

### AC-7 — Automatic ON does not invent an anchor

**Given** effective Automatic Scheduling is `ON` and concrete scheduler
prerequisites do not yield Execution End
**When** recommendation is requested
**Then** system does not use today as a replacement effective scheduling anchor
**And** candidate ranking is unavailable/alphabetical with existing unscheduled
explanation.

### AC-8 — Automatic OFF uses Manual Execution Start first

**Given** Automatic Scheduling is `OFF` and draft Manual Execution Start exists
**When** recommendation is calculated
**Then** that Date is the advisory start lower bound
**And** manual dates remain unchanged.

### AC-9 — Automatic OFF falls back to Project Start or today

**Given** Automatic Scheduling is `OFF` and Manual Execution Start is empty
**When** Project Scheduling Start Date exists
**Then** advisory anchor is the later of Project Start and backend today in
`APP_TIMEZONE`.

**Given** both dates are empty
**Then** advisory anchor is backend today in `APP_TIMEZONE`.

### AC-10 — Manual Execution End does not affect ranking

**Given** Automatic Scheduling is `OFF` with complete manual range
**When** recommendation is calculated
**Then** Estimated Finish is derived from advisory allocation
**And** manual Execution End is not used as completion metric
**And** no earlier/later warning is shown
**And** confirmed manual dates are unchanged.

### AC-11 — Same snapshot for all candidates

**When** one batch is calculated
**Then** every candidate uses one confirmed schedule/capacity/version snapshot
**And** one candidate cannot be based on a newer Project version than another.

### AC-12 — No one-request-per-candidate behaviour

**When** user invokes Assignee recommendation
**Then** frontend sends one batch request
**And** backend returns all candidate results deterministically.

### AC-13 — Percentage is preserved on first selection

**Given** Task draft Capacity Allocation is `40%` and Assignee is empty
**When** user selects first Assignee
**Then** draft remains `40%`
**And** selected candidate was simulated at `40%`.

### AC-14 — Percentage is preserved on reassignment

**Given** confirmed/draft Capacity Allocation is `40%`
**When** Assignee changes to another Member
**Then** percentage remains `40%`
**And** Save omission preserves `40%`
**And** no implicit `100%` reset occurs.

### AC-15 — Percentage is preserved on clear

**Given** confirmed/draft Capacity Allocation is `40%`
**When** Assignee is cleared
**Then** percentage remains `40%`
**And** Task follows existing missing-Assignee scheduling behaviour.

### AC-16 — Feasible candidates precede overcapacity candidates

**Given** Candidate A finishes earlier but adds overcapacity
**And** Candidate B finishes later without added overcapacity
**When** results are ranked
**Then** B appears before A.

### AC-17 — Earliest finish is primary inside a rank group

**Given** two feasible candidates with different simulated Execution End
**When** results are ranked
**Then** earlier Execution End appears first regardless of lower-priority
portfolio displacement.

### AC-18 — Remaining capacity breaks equal completion

**Given** two candidates in the same rank group with equal Execution End
**And** Candidate A leaves `4h` while Candidate B leaves `2h` on completion Date
**When** results are ranked
**Then** A appears first.

### AC-19 — Remaining capacity excludes lower-priority consumption

**Given** lower-priority work could consume candidate's residual capacity later
on the same Date
**When** tie-break metric is calculated
**Then** remaining capacity is measured immediately after candidate allocation
**And** lower-priority consumption does not erase the metric.

### AC-20 — Existing overcapacity is not a false penalty

**Given** baseline already has overcapacity
**When** candidate does not increase overcapacity on any affected Date
**Then** candidate is classified feasible.

### AC-21 — Added overcapacity is incremental

**Given** candidate increases overcapacity on at least one Date
**When** ranking is calculated
**Then** candidate is classified overcapacity
**And** reduction on another Date does not offset the added Date.

### AC-22 — Name and ID are final tie-breakers

**Given** candidates tie on group, Execution End, and remaining capacity
**Then** normalized name ascending is used
**And** stable Member ID resolves duplicate names.

### AC-23 — No projected completion stays selectable

**Given** candidate has no positive capacity inside scheduler horizon
**When** batch returns
**Then** candidate is labelled `No projected completion`
**And** appears after candidates with completion
**And** remains selectable when normal Task validation allows it.

### AC-24 — Current Role mismatch is preserved visibly

**Given** selected Role changes and current Assignee no longer matches
**Then** current Assignee is not cleared implicitly
**And** it appears last with mismatch warning
**And** Save requires replace or clear according to existing validation.

### AC-25 — Recommendation metadata differs by mode

**Given** Automatic Scheduling `ON`
**Then** option uses `finishes`/Projected Finish terminology.

**Given** Automatic Scheduling `OFF`
**Then** option uses `estimated` terminology
**And** no manual-end warning is added.

### AC-26 — Latest request before selection

**Given** prior result is stale after draft or confirmed state change
**When** user invokes Assignee control
**Then** latest batch is requested
**And** stale ranked options are not selectable while it is pending.

### AC-27 — Open option order is stable

**Given** Assignee options are open
**When** a newer response arrives or application Date crosses midnight
**Then** visible ordering does not change until control closes.

### AC-28 — Older response cannot overwrite newer result

**Given** two requests overlap
**When** older response resolves last
**Then** it is ignored
**And** selected Assignee and newest ranking remain unchanged.

### AC-29 — Selection does not create reservation

**When** user selects recommended Assignee
**Then** no Task/schedule state is persisted until Save
**And** candidate capacity is not reserved.

### AC-30 — Save remains source of truth

**Given** confirmed schedule changes after recommendation
**When** user saves
**Then** normal latest-state scheduler/mutation contract applies
**And** final dates may differ from recommendation
**And** recommendation age alone does not reject Save.

### AC-31 — Batch failure falls back safely

**Given** batch simulation fails
**Then** draft and selected Assignee remain intact
**And** candidates become deterministic alphabetical options
**And** Save remains available subject to normal validation
**And** user sees a recoverable recommendation-unavailable message.

### AC-32 — Candidate-specific failure is isolated

**Given** one candidate cannot produce completion but shared snapshot is valid
**Then** successful candidate results remain available
**And** failed candidate returns a stable no-completion reason.

### AC-33 — No side effects

**When** recommendation succeeds or fails
**Then** Task fields, dependencies, dates, allocations, Project versions, and
confirmed caches remain unchanged.

### AC-34 — Application timezone owns today

**Given** clients in different browser timezones
**When** they request the same manual recommendation at the same application Date
**Then** backend `APP_TIMEZONE` resolves the same today anchor and ranking.

### AC-35 — Three-Level Confidence

**Then** every AC has traceable Code Inspection, Unit/Integration Test, and
Acceptance-Level Test evidence according to `AGENTS.md`.

---

## 20. Mandatory Test Scenarios

### 20.1 Domain/Application

- deterministic rank-group ordering;
- earlier finish before later finish inside same group;
- remaining capacity tie-break;
- case-insensitive name and ID tie-break;
- incremental overcapacity with existing baseline overcapacity;
- added overcapacity on one Date not offset by reduction elsewhere;
- percentage preservation on first select/change/clear;
- manual anchor with explicit Execution Start;
- manual anchor with future Project Start;
- manual anchor with past Project Start and today fallback;
- manual anchor with no Project Start;
- manual Execution End ignored;
- no-completion reason mapping;
- current Role mismatch placement;
- required-input gate.

### 20.2 Scheduler/Application Integration

- Automatic `ON` current Task allocation removed before current/candidate
  simulation;
- Automatic `OFF` fixed Task allocation removed while other fixed rows remain;
- same snapshot/version reused for all candidates;
- exact dependency, Lag, priority, WBS order, capacity override, holiday, buffer,
  percentage, completed, Locked, and fixed-allocation semantics;
- lower-priority automatic work may move in simulation without affecting ranking
  objective;
- no invented automatic anchor;
- manual advisory today uses `APP_TIMEZONE`;
- no persistence and no schedule-version change after success/failure;
- transaction rollback under injected scheduler failure;
- candidate-specific no-completion does not fail whole batch;
- bounded candidate and allocation queries without N+1 Member reads.

### 20.3 API

- one batch request returns ranked candidate DTOs;
- invalid draft returns stable 400;
- completed/Locked/Group returns stable conflict;
- automatic and manual-advisory mode mapping;
- calculated Date and snapshot/version identity;
- browser cannot override authoritative today;
- exact decimal-hour mapping;
- rollback leaves Task/dependency/allocation/version unchanged;
- shared-snapshot conflict returns stable error.

### 20.4 Frontend Component

- Role and Assignee render immediately after Lag with correct tab order;
- required-input helper and no API call;
- latest request on Assignee invocation;
- loading state prevents stale option selection;
- one request for all candidates;
- option labels for feasible, overcapacity, no-completion, and mismatch;
- percentage preserved on first assignment/change/clear;
- role change preserves mismatched Assignee visibly;
- selected Assignee remains stable when rank changes;
- open option ordering is frozen;
- older response ignored;
- batch failure fallback and Retry;
- keyboard, screen-reader name/metadata, visible focus, responsive long labels.

### 20.5 Acceptance-Level

1. Create a Task, enter Effort/percentage/Lag/Role, choose first Assignee from a
   ranked list, and verify percentage is preserved.
2. Edit an automatically scheduled Task and verify removing its old allocation
   prevents current Assignee double-counting.
3. Give two candidates the same Execution End but different remaining capacity
   and verify the larger remainder appears first.
4. Make an earlier candidate add overcapacity and verify every feasible candidate
   remains above it.
5. Use Automatic `OFF` with Manual Execution Start and verify Estimated Finish
   ranking while confirmed manual dates do not change.
6. Use Automatic `OFF` without Manual Execution Start and Project Start and verify
   backend today in `APP_TIMEZONE` is used.
7. Use an old Project Start and verify manual fallback does not estimate a new
   assignment in the past.
8. Remove Project Scheduling Start Date under Automatic `ON` and verify no today
   anchor is invented.
9. Change Role so current Assignee mismatches and verify it remains visible until
   user replaces or clears it.
10. Change Assignee on a `40%` Task and verify confirmed percentage remains `40%`.
11. Change capacity/dependency state while requests overlap and verify only the
    newest result becomes selectable.
12. Inject batch failure and verify alphabetical Assignee selection and Save
    remain usable with no confirmed side effects.

---

## 21. Definition of Done

- US-6.5 is linked from project context and applicable scheduling/WBS stories.
- US-6.3 reset wording, ACs, tests, context, and architecture are revised to
  preserve Task Capacity Allocation Percentage across Assignee changes.
- Automatic `ON` recommendation proves reuse of the concrete scheduler rather
  than a duplicate algorithm.
- Automatic `OFF` advisory anchor and allocation rules are deterministic and
  timezone-stable.
- Batch response is exact, side-effect free, snapshot-consistent, and free from
  per-candidate HTTP requests or N+1 candidate reads.
- Ranking, metadata, fallback, stale-response, and field-order behaviours are
  verified through all three confidence levels.
- Recommendation remains advisory; Save and the confirmed scheduler remain the
  final source of truth.
