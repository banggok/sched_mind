export interface WBSNode {
  id: string;
  projectId: string;
  parentId?: string;
  name: string;
  position: number;
  hasChildren: boolean;
  scheduling?: {
    version: number;
    source: "inherit" | "override";
    automaticScheduling?: boolean;
    schedulingStartDate?: string;
    localStatus: "open" | "locked";
    effectiveAutomaticScheduling: boolean;
    effectiveSchedulingStartDate?: string;
    inheritedAutomaticScheduling: boolean;
    inheritedSchedulingStartDate?: string;
    inheritedAutomaticSource: { id: string; name: string };
    inheritedStartDateSource: { id: string; name: string };
    automaticSource: { id: string; name: string };
    startDateSource: { id: string; name: string };
    effectiveLifecycle: "open" | "locked" | "closed";
    lockOwner?: { id: string; name: string };
  };
  executable: {
    roleId?: string;
    assigneeId?: string;
    effortMinutes?: number;
    lagDays: number;
    capacityAllocationPercentage?: number;
    executionTimeline: { start?: string; end?: string };
    commitmentTimeline: { start?: string; end?: string };
    executionUnscheduledReason?: string;
    commitmentUnscheduledReason?: string;
    actualStart?: string;
    actualEnd?: string;
  };
  children: WBSNode[];
}
