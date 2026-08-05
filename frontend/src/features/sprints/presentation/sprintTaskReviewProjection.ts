import type {
  SprintDailySummary,
  SprintMemberProjection,
  SprintTaskProjection,
  SprintTotals,
} from "../domain/sprint";

interface DerivedReview {
  members: SprintMemberProjection[];
  tasks: SprintTaskProjection[];
  needsReviewTasks: SprintTaskProjection[];
  needsReviewTaskIDs: Set<string>;
  totals: SprintTotals;
}

export function deriveReviewProjection(
  sourceMembers: SprintMemberProjection[],
  sourceTasks: SprintTaskProjection[],
  sprintStart: string,
  sprintEnd: string,
): DerivedReview {
  const tasks = sortDailyPlanTasks(sourceTasks);
  const selectedMemberIDs = new Set(sourceMembers.map((member) => member.id));
  const needsReviewTasks = tasks.filter(
    (task) =>
      task.warnings.length > 0 ||
      !task.assigneeId ||
      !selectedMemberIDs.has(task.assigneeId),
  );
  const needsReviewTaskIDs = new Set(needsReviewTasks.map((task) => task.id));
  const members = sourceMembers.map((member) => {
    const memberTasks = tasks.filter(
      (task) =>
        task.assigneeId === member.id && !needsReviewTaskIDs.has(task.id),
    );
    const selectedByDate = allocationByDate(
      memberTasks,
      sprintStart,
      sprintEnd,
    );
    const dailySummaries = member.dailyCapacity
      .filter((capacity) =>
        insideSprintDate(capacity.date, sprintStart, sprintEnd),
      )
      .map((capacity) => {
        const selectedAllocationMinutes =
          selectedByDate.get(capacity.date) ?? 0;
        return dailySummary(
          capacity.date,
          capacity.minutes,
          selectedAllocationMinutes,
        );
      });
    return {
      ...member,
      dailySummaries,
      capacityMinutes: sum(dailySummaries, "capacityMinutes"),
      inSprintAllocationMinutes: sum(
        dailySummaries,
        "selectedAllocationMinutes",
      ),
      remainingMinutes: sum(dailySummaries, "remainingMinutes"),
      overcapacityMinutes: sum(dailySummaries, "overcapacityMinutes"),
      totalAllocationMinutes: memberTasks.reduce(
        (total, task) => total + task.totalAllocationMinutes,
        0,
      ),
    };
  });
  const sprintDaily = aggregateDailySummaries(members);
  const needsReviewByDate = allocationByDate(
    needsReviewTasks,
    sprintStart,
    sprintEnd,
  );
  const needsReviewDailyAllocation = [...needsReviewByDate.entries()]
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([date, minutes]) => ({ date, minutes }));
  const totals: SprintTotals = {
    capacityMinutes: sum(sprintDaily, "capacityMinutes"),
    selectedMemberAllocationMinutes: sum(
      sprintDaily,
      "selectedAllocationMinutes",
    ),
    remainingMinutes: sum(sprintDaily, "remainingMinutes"),
    overcapacityMinutes: sum(sprintDaily, "overcapacityMinutes"),
    needsReviewAllocationMinutes: needsReviewDailyAllocation.reduce(
      (total, value) => total + value.minutes,
      0,
    ),
    needsReviewDailyAllocation,
    allTaskInSprintMinutes: tasks.reduce(
      (total, task) => total + task.inSprintAllocationMinutes,
      0,
    ),
    allTaskTotalMinutes: tasks.reduce(
      (total, task) => total + task.totalAllocationMinutes,
      0,
    ),
    dailySummaries: sprintDaily,
  };
  return { members, tasks, needsReviewTasks, needsReviewTaskIDs, totals };
}

export function sortDailyPlanTasks(
  sourceTasks: SprintTaskProjection[],
): SprintTaskProjection[] {
  return [...sourceTasks].sort((left, right) => {
    const leftGroup = dailyPlanGroup(left);
    const rightGroup = dailyPlanGroup(right);
    if (leftGroup !== rightGroup) return leftGroup - rightGroup;
    if (left.dailyPlanOrderDate && right.dailyPlanOrderDate) {
      const date = left.dailyPlanOrderDate.localeCompare(
        right.dailyPlanOrderDate,
      );
      if (date !== 0) return date;
    }
    if (left.projectPriority !== right.projectPriority)
      return left.projectPriority - right.projectPriority;
    if (left.wbsRank !== right.wbsRank) return left.wbsRank - right.wbsRank;
    return left.id.localeCompare(right.id);
  });
}

export function insideSprintDate(date: string, start: string, end: string) {
  return date >= start && date <= end;
}

function dailyPlanGroup(task: SprintTaskProjection) {
  if (!task.dailyPlanOrderDate) return 2;
  return task.completed ? 1 : 0;
}

function allocationByDate(
  tasks: SprintTaskProjection[],
  sprintStart: string,
  sprintEnd: string,
) {
  const values = new Map<string, number>();
  tasks.forEach((task) =>
    task.allocations.forEach((allocation) => {
      if (!insideSprintDate(allocation.date, sprintStart, sprintEnd)) return;
      values.set(
        allocation.date,
        (values.get(allocation.date) ?? 0) + allocation.minutes,
      );
    }),
  );
  return values;
}

function dailySummary(
  date: string,
  capacityMinutes: number,
  selectedAllocationMinutes: number,
): SprintDailySummary {
  return {
    date,
    capacityMinutes,
    selectedAllocationMinutes,
    remainingMinutes: Math.max(0, capacityMinutes - selectedAllocationMinutes),
    overcapacityMinutes: Math.max(
      0,
      selectedAllocationMinutes - capacityMinutes,
    ),
  };
}

function aggregateDailySummaries(members: SprintMemberProjection[]) {
  const values = new Map<string, SprintDailySummary>();
  members.forEach((member) =>
    member.dailySummaries.forEach((daily) => {
      const current = values.get(daily.date) ?? dailySummary(daily.date, 0, 0);
      values.set(daily.date, {
        date: daily.date,
        capacityMinutes: current.capacityMinutes + daily.capacityMinutes,
        selectedAllocationMinutes:
          current.selectedAllocationMinutes + daily.selectedAllocationMinutes,
        remainingMinutes: current.remainingMinutes + daily.remainingMinutes,
        overcapacityMinutes:
          current.overcapacityMinutes + daily.overcapacityMinutes,
      });
    }),
  );
  return [...values.values()].sort((left, right) =>
    left.date.localeCompare(right.date),
  );
}

function sum<T extends SprintDailySummary>(
  values: T[],
  field:
    | "capacityMinutes"
    | "selectedAllocationMinutes"
    | "remainingMinutes"
    | "overcapacityMinutes",
) {
  return values.reduce((total, value) => total + value[field], 0);
}
