export type SprintStatus = "planned" | "started";

export interface Sprint {
  id: string;
  name: string;
  startDate: string;
  endDate: string;
  status: SprintStatus;
  version: number;
  memberIds: string[];
  taskIds: string[];
  startedAt?: Date;
  createdAt: Date;
  updatedAt: Date;
}

export interface SprintDetail {
  sprint: Pick<
    Sprint,
    "id" | "name" | "startDate" | "endDate" | "status" | "version" | "startedAt"
  >;
  projectionToken: string;
  members: SprintMemberProjection[];
  tasks: SprintTaskProjection[];
  totals: SprintTotals;
}

export interface DailyMinutes {
  date: string;
  minutes: number;
}

export interface SprintDailySummary {
  date: string;
  capacityMinutes: number;
  selectedAllocationMinutes: number;
  remainingMinutes: number;
  overcapacityMinutes: number;
}

export interface SprintMemberProjection {
  id: string;
  name: string;
  roleName: string;
  capacityMinutes: number;
  inSprintAllocationMinutes: number;
  remainingMinutes: number;
  overcapacityMinutes: number;
  totalAllocationMinutes: number;
  dailyCapacity: DailyMinutes[];
  dailySummaries: SprintDailySummary[];
}

export interface SprintTaskProjection {
  id: string;
  projectId: string;
  projectName: string;
  projectStatus: string;
  projectPriority: number;
  name: string;
  wbsOrder: string;
  wbsPath: string;
  wbsRank: number;
  assigneeId?: string;
  assigneeName?: string;
  executionStart?: string;
  executionEnd?: string;
  dailyPlanOrderDate?: string;
  completed: boolean;
  allocations: DailyMinutes[];
  inSprintAllocationMinutes: number;
  outsideAllocationMinutes: number;
  totalAllocationMinutes: number;
  warnings: string[];
}

export interface SprintTotals {
  capacityMinutes: number;
  selectedMemberAllocationMinutes: number;
  remainingMinutes: number;
  overcapacityMinutes: number;
  needsReviewAllocationMinutes: number;
  needsReviewDailyAllocation: DailyMinutes[];
  allTaskInSprintMinutes: number;
  allTaskTotalMinutes: number;
  dailySummaries: SprintDailySummary[];
}

export interface SprintSuggestion {
  members: SprintMemberProjection[];
  tasks: Array<{
    task: SprintTaskProjection;
    reason: "mandatory" | "capacity_fill";
  }>;
  totals: SprintTotals;
  projectionToken: string;
}

export interface SprintWriteInput {
  name: string;
  startDate: string;
  endDate: string;
  memberIds: string[];
  taskIds: string[];
  version?: number;
}
