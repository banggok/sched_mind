# US-3.3 Configure Project Settings

## User Story

**As an** Engineering Lead
**I want** to configure project scheduling settings
**So that** the Scheduling Engine follows the project-specific planning strategy.

---

## Business Context

This story allows an Engineering Lead to configure how a project participates in the Scheduling Engine.

The configuration determines whether the project is fully scheduler-driven or manually planned.

This story does **not** change project lifecycle (`Open`, `Locked`, `Closed`), WBS, or scheduling algorithms. It only controls how the Scheduling Engine treats the project.

---

# Scope

## In Scope

- Configure Automatic Scheduling.
- Configure the Project Scheduling Start Date.
- Configure Project Buffer (%).
- Validation rules.
- UI behaviour.
- Scheduler behaviour.
- Interaction with Project Status.
- Interaction with Execution, Commitment and Forecast timelines.

## Out of Scope

- Scheduling Engine implementation.
- Forecast algorithm.
- WBS management.
- Dependency management.
- Capacity management.
- Project lifecycle management.

---

# Product Rules

## Automatic Scheduling

Default: Enabled.

### Enabled

- Execution Timeline is calculated automatically.
- Commitment Timeline is calculated automatically.
- Execution and Commitment dates are read-only.

### Disabled

- Execution Timeline becomes manual.
- Commitment Timeline becomes manual.
- Forecast remains automatic.
- Engineering Lead manually maintains Execution and Commitment dates.

## Scheduling Start Date

- Scheduling Start Date is the single Project-level anchor for Automatic Scheduling.
- It is nullable and stored as SQL `DATE`; its API format is `YYYY-MM-DD`.
- It is optional while Automatic Scheduling is disabled.
- Automatic Scheduling may remain enabled without an anchor, but must keep the
  Execution and Commitment timelines empty and expose the warning
  `Automatic Scheduling requires a Project Scheduling Start Date.`
- The Scheduling Engine must not invent the first execution date. Epic 6 starts
  the first executable tasks without predecessors at the earliest working date
  allowed by this anchor and the other approved scheduling constraints.
- Scheduling Start Date belongs to Project, never to an Executable WBS.
- The Project form uses the shared calendar behaviour used by other date
  features: single-date selection, selected-date state, weekend/public-holiday
  marking, viewport-aware placement, scroll fallback, Escape, and outside-click
  dismissal. The calendar is disabled whenever Project settings are read-only.

## Project Buffer

- Default: 20%.
- Range: 0%–100%.
- Editable only when Automatic Scheduling is Enabled.
- Disabled (value preserved) when Automatic Scheduling is Disabled.
- Ignored by the Scheduling Engine while Automatic Scheduling is Disabled.

## Project Status

### Open

Project Settings may be modified.

### Locked

Project Settings are read-only while the Project is Locked. This restriction
must not introduce or imply a `Locked → Open` lifecycle transition. US-4.2 is a
narrow command exception: a completed Task may be reopened while the Project
remains Locked, without making Project Settings editable.

### Closed

Project Settings are read-only.

## Save Behaviour

Changing the Automatic Scheduling toggle does not immediately apply changes.

Changes take effect only after Save.

If OFF→ON scheduling coordination fails, the settings update and scheduler
coordination are rolled back as one atomic operation. The confirmed settings
remain unchanged and the API reports a safe operation failure.

## ON → OFF

After Save:

- Existing calculated Execution and Commitment timelines become the initial manual values.
- Engineering Lead may edit unfinished tasks manually.
- Project Buffer becomes inactive.

## OFF → ON

After Save:

- Scheduler recalculates only unfinished tasks.
- Tasks with Actual End remain unchanged.
- Manual timelines of unfinished tasks are replaced by scheduler-generated values.
- Stored Project Buffer becomes active again.

## Forecast

Forecast is independent from Automatic Scheduling and always remains active.


## Scheduling Calculation Rules

Automatic Scheduling affects only the Execution and Commitment scheduling engines.

### Execution Timeline

Execution Timeline is calculated using:

- Project Priority
- Dependencies
- Lag
- Public Holiday
- Capacity Override
- Daily Capacity

Execution Timeline is never affected by any buffer.

### Commitment Timeline

Commitment Timeline is calculated in the following order:

1. Use the Execution Timeline as the scheduling baseline.
2. Apply Member Buffer by deriving a Commitment Capacity for each engineer.
3. Recalculate the schedule using the Commitment Capacity.
4. Apply the Project Buffer (%) to produce the final Commitment Timeline.

Rules:

- Internal calculations use minute precision.
- Intermediate values are not rounded.
- Rounding is applied only to the final calculated Commitment Timeline where required.
- Project Buffer is ignored when Automatic Scheduling is disabled.
- Member Buffer is ignored when Automatic Scheduling is disabled.


---

# Acceptance Criteria

1. Automatic Scheduling defaults to Enabled.
2. Project Buffer defaults to 20%.
3. Project Buffer accepts values from 0% to 100%.
4. Project Buffer is editable only when Automatic Scheduling is Enabled.
5. Project Buffer is disabled when Automatic Scheduling is Disabled.
6. Changes are applied only after Save.
7. Cancelling without saving preserves existing settings.
8. ON→OFF retains current calculated timelines as initial manual values.
9. OFF→ON recalculates only unfinished tasks.
10. Tasks with Actual End are never recalculated.
11. Forecast remains active regardless of Automatic Scheduling mode.
12. Locked projects reject Project Settings updates.
13. Closed projects reject Project Settings updates.
14. Project Name and Project Settings share one Add/Edit form; there is no separate Settings action.
15. Locked settings remain visible and read-only while Name retains its existing edit contract; Closed projects show the combined form entirely read-only without Save.
16. OFF→ON requires confirmation before Save. With Scheduling Start Date
    configured, scheduler failure rolls back the settings change; without it,
    the setting is saved, scheduling is not invoked, and the approved warning is shown.
17. Scheduling Start Date is visible and editable in the combined Project form whenever Project settings are editable.
18. Automatic Scheduling without Scheduling Start Date shows the approved warning and does not produce Execution or Commitment dates.

---

# API

Project create and update accept `automaticScheduling`, nullable `schedulingStartDate`, and `projectBuffer`
beside Name so the combined form is persisted atomically. Status remains absent
from those payloads and changes only through the lifecycle command endpoint.

PATCH /api/projects/{projectId}/settings

Body

```json
{
  "automaticScheduling": true,
  "schedulingStartDate": "2026-08-03",
  "projectBuffer": 20
}
```

---

# Test Cases

## Happy Path

- Enable Automatic Scheduling.
- Disable Automatic Scheduling.
- Change Project Buffer.
- Save changes.
- Re-enable Automatic Scheduling.
- Verify only unfinished tasks are recalculated.

## Validation

- Buffer below 0%.
- Buffer above 100%.
- Locked project update.
- Closed project update.
- Unsaved changes.
- Invalid Scheduling Start Date API format.
- Automatic Scheduling enabled with no Scheduling Start Date shows a warning and retains empty generated timelines.

## Regression

- Forecast continues working.
- Actual End tasks remain unchanged.
- Stored Project Buffer is reused after re-enabling.

---

# Required Automated Tests

- Domain
- Application
- Repository
- API
- Frontend
- Scheduler integration contract

---

# Documentation Impact

Update:

- Architecture
- Scheduling Engine
- Project Settings
- API documentation

---

# Locked Product Decisions

- Automatic Scheduling default = Enabled.
- Project Buffer default = 20%.
- Project Buffer range = 0–100%.
- Automatic Scheduling OFF makes Execution and Commitment manual.
- Forecast always remains active.
- Tasks with Actual End are immutable during recalculation.
- OFF→ON recalculates only unfinished tasks.
- Project Buffer is ignored while Automatic Scheduling is disabled.
- Locked and Closed projects cannot modify Project Settings.
- Changes apply only after Save.

---

# Unresolved Questions

None.

## Deferred Implementation

This story intentionally introduces the Project Settings feature and the scheduler integration contract only.

The actual scheduling behaviour is deferred because the Scheduling Engine cannot be fully implemented until the required planning model exists.

This story MUST implement:

- Project Settings persistence.
- API.
- Validation.
- UI.
- Status restrictions.
- Scheduler integration contract.
- Application port for scheduling recalculation.
- Automated tests verifying the scheduling port is invoked correctly.

This story MUST NOT implement:

- Execution Scheduler.
- Commitment Scheduler.
- Forecast Scheduler.
- Timeline recalculation.
- Actual scheduling algorithms.

The scheduling implementation is deferred to **Epic 6 – Scheduling Engine**.

The scheduler integration contract introduced by this story SHALL be fulfilled by the following stories:

- US-6.1 Execution Scheduling Engine
- US-6.2 Commitment Scheduling Engine
- US-6.3 Forecast Scheduling Engine

Those stories become executable only after the following MVP capabilities are completed:

- Project Management
- WBS Management
- Task Management
- Task Dependencies
- Actual End recording

Until those prerequisites exist, the scheduling contract introduced by this story shall be implemented using an application port (for example `RecalculateProjectSchedule`) and verified through automated tests only.

The current production composition uses a no-op implementation of this port
until Epic 6 supplies the concrete scheduler. This temporary adapter must not be
treated as evidence that timeline recalculation is implemented.
