# Task Capacity Allocation Implementation Evidence

This document owns traceability for US-6.3 and revised US-6.2 AC-7–AC-12. The
authoritative behavior remains in the user stories; this file records production
symbols and automated evidence without redefining those rules.

## Delta coverage

| Delta | Implemented scope |
| --- | --- |
| D1 | Persisted integer percentage, default/omission/reset, conversion, API, form, lifecycle, and allocation display. |
| D2 | Independently rounded daily limits, concurrent ordered allocation, safe blocker reconciliation, and fixed manual allocation. |
| D3 | Actual completion ignores planned percentage; Reopen returns to the persisted planned value. |
| D4 | Actual-Date-only and Actual-versus-Actual allocation, progressive balancing, backfill, equalized overcapacity, and immutable prior Actual rows. |
| D5 | PostgreSQL backfill/default/check migration and guarded rollback. |
| D6 | Atomic rollback, shared schedule-version/stale-response protection, accessible validation, and traceable evidence. |

## US-6.3 acceptance evidence

Unless a narrower command is shown, backend evidence is executed by
`cd backend && go test ./...` and frontend evidence by `cd frontend && npm test`.

| AC | Code inspection | Unit/integration evidence | Acceptance-level evidence |
| --- | --- | --- | --- |
| AC-1 | `domain.New`; WBS mapping and migration default | `TestCapacityAllocationPercentageRangeAndDefault_US63_AC1_AC2` | `TestCreateTaskAcceptanceDefaultsCapacityPercentageAndSkipsUnneededScheduler_US63_AC1_US6_AC29_US4_AC23` |
| AC-2 | `ValidateExecutable`; `parseCapacityAllocation`; form validation | `TestCapacityAllocationPercentageRangeAndDefault_US63_AC1_AC2`; `TestExecutableUpdateDistinguishesOmittedPercentageFromExplicitZero_US63_AC2_AC4` | WBS dialog invalid-value workflow prevents gateway mutation. |
| AC-3 | migration `000022_add_task_capacity_allocation.up.sql` | PostgreSQL 16 startup migration passed against a schema at version 21 with a pre-feature Task; the row became `100` and its planned dates were unchanged. The guarded down migration also rejected a custom `50` value without dropping the column. | Live HTTP smoke test returned `capacityAllocationPercentage: 100` for the migrated Task and for a newly created WBS. |
| AC-4 | HTTP raw-message distinction and repository preserve-existing behavior | `TestExecutableUpdateDistinguishesOmittedPercentageFromExplicitZero_US63_AC2_AC4`; `TestUpdateExecutablePreservesOmittedPercentageAndResetsOnAssigneeChange_US63_AC4_AC5` | Create HTTP acceptance plus update HTTP boundary test. |
| AC-5 | repository Assignee reset and WBS draft reset | `TestUpdateExecutablePreservesOmittedPercentageAndResetsOnAssigneeChange_US63_AC4_AC5` | WBS dialog Assignee-change preview workflow verifies visible reset and current request. |
| AC-6 | `taskDailyLimit` applies percentage after timeline capacity | `TestTaskDailyLimitRoundsHalfUpAndKeepsPositiveMinimum_US63_AC6_AC7` | Concurrent scheduling repository workflow persists rounded rows. |
| AC-7 | `taskDailyLimit` zero/minimum branch | `TestTaskDailyLimitRoundsHalfUpAndKeepsPositiveMinimum_US63_AC6_AC7` | Allocation workflow observes positive `0.5h` minimum. |
| AC-8 | ready sorting precedes daily limit allocation | `TestConcurrentSameAssigneeAllocationUsesDailyLimits_US63_AC9_AC11_AC15` | Automatic scheduling acceptance preserves Project/WBS order. |
| AC-9 | `simulateContiguous` uses remaining capacity and task limit | `TestConcurrentSameAssigneeAllocationUsesDailyLimits_US63_AC9_AC11_AC15` | Same test executes the persisted scheduling boundary. |
| AC-10 | percentages are independent; daily remaining capacity is authoritative | `TestConfiguredPercentagesAboveOneHundredRemainCapacitySafe_US63_AC10` | Same test re-reads `330/150` persisted rows against `480` capacity. |
| AC-11 | concurrent allocation and independent Start/End derivation | `TestConcurrentSameAssigneeAllocationUsesDailyLimits_US63_AC9_AC11_AC15` | Same repository workflow verifies A/B/C positive same-day rows and no serial relation. |
| AC-12 | no limit carry-over in per-date calculation | Concurrent allocation row assertions | A/B/C workflow re-reads later rows without accumulated limit. |
| AC-13 | displacement removes only conflicting mutable future rows | `TestPriorityPreservingConcurrentAllocationAndSafeBlockers_US63_AC13_AC16` | Ordering acceptance preserves earlier lower-priority allocation and capacity safety. |
| AC-14 | `dependenciesResolved` and `readiness` precede allocation | `TestDependencyAndLagReadinessRules_AC11_AC12_AC13_AC14_AC15` | Automatic scheduling workflow observes dependency-gated dates. |
| AC-15 | blocker derivation skips common earliest start | `TestConcurrentSameAssigneeAllocationUsesDailyLimits_US63_AC9_AC11_AC15` | Persisted dependency count remains zero for valid parallel start. |
| AC-16 | stable capacity-release blocker; continuing/manual candidates rejected | `TestPriorityPreservingConcurrentAllocationAndSafeBlockers_US63_AC13_AC16` | Gap workflow re-reads the deterministic automatic endpoint. |
| AC-17 | reconciliation removes stale automatic ownership and preserves manual ownership | `TestAutomaticOwnershipReconciliationPreservesManualGraph_AC22_AC23_AC24_AC25_AC26` | Real dependency workflow verifies one manual-only endpoint after reconciliation. |
| AC-18 | `reconstructFixed` evenly persists authoritative manual allocation, including overcapacity | `TestManualFixedAllocationMayExceedLimitAndReservesAutomaticCapacity_US63_AC18_AC19`; `TestManualFixedAllocationUsesEligibleZeroCapacityWeekdays_US63_AC18_AC20` | Repository workflow re-reads unchanged manual dates and fixed rows. |
| AC-19 | fixed rows reserve before mutable automatic rows | `TestManualFixedAllocationMayExceedLimitAndReservesAutomaticCapacity_US63_AC18_AC19` | Same workflow observes automatic `180/180/120` around fixed `300/300`. |
| AC-20 | no-working range persists all Effort on manual Start | `TestManualFixedAllocationUsesManualStartWhenRangeHasNoWorkingDate_US63_AC20` | Same repository workflow re-reads the weekend Start row. |
| AC-21 | Actual allocator has no percentage input and uses revised US-6.2 algorithm | `TestReplaceActualAllocationsProgressivelyRebalancesAroundExistingActualLoad_US62_AC8_AC9` with persisted `20%` | Completion allocation repository boundary re-reads uncapped balanced rows. |
| AC-22 | completion replaces planned rows; Reopen removes Actual and reschedules persisted percentage | Existing Reopen integration tests plus percentage persistence tests | Reopen HTTP/application workflows and scheduling acceptance. |
| AC-23 | domain/repository lifecycle guards precede percentage mutation | Existing completed/Locked/Closed WBS mutation tests | WBS HTTP lifecycle error contracts. |
| AC-24 | conversion copies percentage to child and resets Group to `100` non-applicable representation | `TestMoveConversionRetargetsDependenciesAndPreservesPercentage_US63_AC24` | Repository conversion boundary re-reads Group, child, and dependency endpoint. |
| AC-25 | WBS write and concrete scheduler share serialized transaction | `TestUpdateExecutableAndSchedulerShareTransactionRollback_AC27_AC28_AC35` now includes percentage rollback | Scheduling persistence-failure acceptance re-reads unchanged state. |
| AC-26 | preview abort/version guards and shared projection clock | WBS gateway stale-response tests | WBS dialog out-of-order preview workflow. |
| AC-27 | allocation API exposes capacity, percentage, daily limit, remaining/overcapacity | allocation gateway tests; WBS dialog accessibility tests | Expanded Task allocation section workflow. |
| AC-28 | This table links every AC to three confidence levels | Full backend/frontend validation commands | Completion status is reported only after all commands pass. |

## Revised US-6.2 Actual Allocation evidence

| AC | Production and evidence |
| --- | --- |
| AC-7 | `replaceActualAllocations` limits rows to Actual Date; `TestReplaceActualAllocationsUsesActualWindowAndEqualizesUnavoidableOvercapacity_US62_AC7_AC11` and non-working Actual-End fallback test. |
| AC-8 | `loadActualAllocationCalendar` selects only other completed Actual rows and excludes the current Task; progressive-rebalance test. |
| AC-9 | balanced half-hour forward pass and progressive remaining-date recalculation; progressive-rebalance test asserts `120/240/240`. |
| AC-10 | chronological spare-capacity backfill occurs before the equalization loop; deterministic excess tests. |
| AC-11 | smallest total overcapacity with latest-date tie-break; Actual-window overcapacity test. |
| AC-12 | per-date remaining capacity floors at zero without debt; existing completed row is re-read unchanged by the progressive-rebalance test. |

## Query review

Actual head-to-head load uses equality on `assignee_id` and `timeline`, excludes
one point `task_id`, joins the Task primary key to require complete Actual Date,
and groups by allocation date. The existing
`task_schedule_allocations_member_date_idx (assignee_id, timeline,
allocation_date, sequence, task_id)` supports the selective prefix and bounded
date attribution; `wbs_nodes` primary key supports the join. Scheduling scope
remains bounded to active shared-assignee/dependency closure. No new index is
introduced because the new percentage is not a query predicate and fixed/manual
rows reuse the existing Task/timeline/date primary key.
