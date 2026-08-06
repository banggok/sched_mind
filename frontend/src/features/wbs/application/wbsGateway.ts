import type { WBSNode } from "../domain/wbs";
export class WBSOperationError extends Error {
  constructor(
    readonly code: string,
    message: string,
  ) {
    super(message);
  }
}
export interface WBSGateway {
  tree(projectId: string, signal?: AbortSignal): Promise<WBSNode[]>;
  allocations(
    projectId: string,
    id: string,
    signal?: AbortSignal,
  ): Promise<AllocationGroups>;
  subscribeToConfirmedChanges?(listener: () => void): () => void;
  create(
    projectId: string,
    parentId: string | undefined,
    name: string,
    confirmConversion: boolean,
  ): Promise<WBSNode>;
  createSibling(
    projectId: string,
    insertAfterId: string,
    name: string,
  ): Promise<WBSNode>;
  rename(projectId: string, id: string, name: string): Promise<void>;
  reorder(
    projectId: string,
    id: string,
    direction: "up" | "down",
  ): Promise<void>;
  place(
    projectId: string,
    id: string,
    targetSiblingId: string,
    placement: "before" | "after",
  ): Promise<void>;
  move(
    projectId: string,
    id: string,
    parentId: string | undefined,
    confirmConversion: boolean,
  ): Promise<void>;
  remove(projectId: string, id: string): Promise<void>;
  updateExecutable(
    projectId: string,
    id: string,
    input: ExecutableInput,
  ): Promise<void>;
  recommendAssignees?(
    projectId: string,
    id: string,
    input: AssigneeRecommendationInput,
    signal?: AbortSignal,
  ): Promise<AssigneeRecommendationResult>;
  previewExecutableSchedule(
    projectId: string,
    id: string,
    input: SchedulePreviewInput,
    signal?: AbortSignal,
  ): Promise<SchedulePreview>;
  complete(
    projectId: string,
    id: string,
    actualStart: string,
    actualEnd: string,
  ): Promise<void>;
  reopen(projectId: string, id: string): Promise<WBSNode>;
}
export interface ExecutableInput {
  name: string;
  roleId?: string;
  assigneeId?: string;
  effortHours?: number;
  lagDays: number;
  capacityAllocationPercentage?: number;
  executionStart?: string;
  executionEnd?: string;
  commitmentStart?: string;
  commitmentEnd?: string;
}

export interface SchedulePreview {
  task: WBSNode;
}

export interface SchedulePreviewInput {
  roleId: string;
  assigneeId?: string;
  effortHours: number;
  lagDays: number;
  capacityAllocationPercentage?: number;
}

export interface AllocationRow {
  date: string;
  allocatedMinutes: number;
  capacityMinutes: number;
  remainingMinutes: number;
  overcapacityMinutes: number;
  capacityAllocationPercentage: number;
  taskDailyLimitMinutes: number;
}

export interface AllocationGroups {
  execution: AllocationRow[];
  commitment: AllocationRow[];
  actual: AllocationRow[];
}

export interface AssigneeRecommendationInput {
  roleId: string;
  effortHours: number;
  lagDays: number;
  capacityAllocationPercentage: number;
  executionStart?: string;
}

export type AssigneeRecommendationMode = "automatic" | "manual-advisory";
export type AssigneeRecommendationRankGroup =
  "feasible" | "overcapacity" | "no-completion";

export interface AssigneeRecommendationItem {
  memberId: string;
  memberName: string;
  roleId: string;
  rankGroup: AssigneeRecommendationRankGroup;
  executionEnd?: string;
  remainingExecutionCapacityHours: number;
  incrementalOvercapacityHours: number;
  reasonCode?: string;
}

export interface AssigneeRecommendationResult {
  calculatedOnDate: string;
  snapshot: {
    projectScheduleVersions: Record<string, number>;
  };
  mode: AssigneeRecommendationMode;
  items: AssigneeRecommendationItem[];
}
