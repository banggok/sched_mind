export type ProjectStatus = "open" | "locked" | "closed";
export type PriorityDirection = "up" | "down";

export interface Project {
  id: string;
  name: string;
  status: ProjectStatus;
  startDate?: string;
  endDate?: string;
  autoCalculateDate: boolean;
  autoDependencyByAssignee: boolean;
  priority: number;
  closedAt?: Date;
  createdAt: Date;
  updatedAt: Date;
}

export class ProjectNameError extends Error {
  constructor(
    readonly code: "PROJECT_NAME_REQUIRED" | "PROJECT_NAME_TOO_LONG",
    message: string,
  ) {
    super(message);
  }
}

export function normalizeProjectName(name: string): string {
  const normalized = name.trim();
  if (!normalized) {
    throw new ProjectNameError(
      "PROJECT_NAME_REQUIRED",
      "Project name is required",
    );
  }
  if ([...normalized].length > 100) {
    throw new ProjectNameError(
      "PROJECT_NAME_TOO_LONG",
      "Project name must not exceed 100 characters",
    );
  }
  return normalized;
}
