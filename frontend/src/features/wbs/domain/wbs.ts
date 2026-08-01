export interface WBSNode {
  id: string;
  projectId: string;
  parentId?: string;
  name: string;
  position: number;
  hasChildren: boolean;
  executable: {
    roleId?: string;
    assigneeId?: string;
    effortMinutes?: number;
    lagDays: number;
    executionTimeline: { start?: string; end?: string };
    commitmentTimeline: { start?: string; end?: string };
    executionUnscheduledReason?: string;
    commitmentUnscheduledReason?: string;
    actualStart?: string;
    actualEnd?: string;
  };
  children: WBSNode[];
}
