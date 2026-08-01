import type { WBSNode } from "./wbs";

export interface TimelineSummary {
  start?: string;
  end?: string;
  scheduledTaskCount: number;
}

export interface WBSSummary {
  taskCount: number;
  execution: TimelineSummary;
  commitment: TimelineSummary;
  completedKnownEffortMinutes: number;
  totalKnownEffortMinutes: number;
  taskWithoutEffortCount: number;
  completionPercentage?: number;
}

export function summarizeWBS(roots: readonly WBSNode[]): WBSSummary {
  const summary: WBSSummary = {
    taskCount: 0,
    execution: { scheduledTaskCount: 0 },
    commitment: { scheduledTaskCount: 0 },
    completedKnownEffortMinutes: 0,
    totalKnownEffortMinutes: 0,
    taskWithoutEffortCount: 0,
  };

  const stack: Array<{ nodes: readonly WBSNode[]; index: number }> = [
    { nodes: roots, index: 0 },
  ];
  while (stack.length > 0) {
    const frame = stack[stack.length - 1];
    if (!frame || frame.index >= frame.nodes.length) {
      stack.pop();
      continue;
    }

    const node = frame.nodes[frame.index];
    frame.index += 1;
    if (!node) continue;

    const isGroup = node.hasChildren || node.children.length > 0;
    if (isGroup) {
      if (node.children.length > 0) {
        stack.push({ nodes: node.children, index: 0 });
      }
      continue;
    }

    summary.taskCount += 1;
    includeTimeline(summary.execution, node.executable.executionTimeline);
    includeTimeline(summary.commitment, node.executable.commitmentTimeline);

    const effortMinutes = node.executable.effortMinutes;
    if (effortMinutes === undefined) {
      summary.taskWithoutEffortCount += 1;
      continue;
    }

    summary.totalKnownEffortMinutes += effortMinutes;
    if (node.executable.actualStart && node.executable.actualEnd) {
      summary.completedKnownEffortMinutes += effortMinutes;
    }
  }

  if (summary.totalKnownEffortMinutes > 0) {
    summary.completionPercentage = roundPercentage(
      summary.completedKnownEffortMinutes,
      summary.totalKnownEffortMinutes,
    );
  }

  return summary;
}

function includeTimeline(
  summary: TimelineSummary,
  timeline: WBSNode["executable"]["executionTimeline"],
): void {
  const { start, end } = timeline;
  if (!start || !end) return;

  summary.scheduledTaskCount += 1;
  if (!summary.start || start < summary.start) summary.start = start;
  if (!summary.end || end > summary.end) summary.end = end;
}

function roundPercentage(
  completedMinutes: number,
  totalMinutes: number,
): number {
  return Math.round((completedMinutes / totalMinutes) * 1000) / 10;
}
