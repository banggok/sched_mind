export type PortfolioProjection = "execution" | "commitment";
export type PortfolioRowKind = "project" | "group" | "task";

export interface PortfolioProject {
  id: string;
  name: string;
  status: "open" | "locked";
  priority: number;
  scheduleVersion: number;
}

export interface PortfolioRow {
  id: string;
  projectId: string;
  parentId?: string;
  kind: PortfolioRowKind;
  name: string;
  wbsNumber: string;
  depth: number;
  position: number;
  status?: "open" | "locked";
  roleId?: string;
  roleName?: string;
  assigneeId?: string;
  assigneeName?: string;
  effortMinutes?: number;
  start?: string;
  end?: string;
  unscheduledReason?: string;
  incompleteEffort: boolean;
  incompleteSchedule: boolean;
  hasChildren: boolean;
  completed: boolean;
}

export interface PortfolioDependency {
  id: string;
  blockingTaskId: string;
  blockedTaskId: string;
}

export interface PortfolioHoliday {
  date: string;
  description: string;
}

export interface PortfolioProjectionResult {
  projection: PortfolioProjection;
  projects: PortfolioProject[];
  rows: PortfolioRow[];
  dependencies: PortfolioDependency[];
  holidays: PortfolioHoliday[];
  workingDayAnchor?: string;
}

export interface SavedPortfolioFilter {
  id: string;
  name: string;
  projectIds: string[];
  version: number;
  createdAt: Date;
  updatedAt: Date;
}

export class PortfolioOperationError extends Error {
  constructor(
    readonly code: string,
    message: string,
    readonly field?: string,
  ) {
    super(message);
  }
}
