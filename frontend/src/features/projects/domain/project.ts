export type ProjectStatus = "open" | "locked" | "closed";
export type PriorityDirection = "up" | "down";

export interface Project {
  id: string;
  name: string;
  status: ProjectStatus;
  startDate?: string;
  endDate?: string;
  autoCalculateDate: boolean;
  automaticScheduling: boolean;
  schedulingStartDate?: string;
  projectBuffer: number;
  scheduleVersion: number;
  priority: number;
  closedAt?: Date;
  createdAt: Date;
  updatedAt: Date;
}

export class ProjectSettingsError extends Error {
  constructor(
    readonly field: "projectBuffer",
    message: string,
  ) {
    super(message);
  }
}

export function validateProjectBuffer(value: number): number {
  if (!Number.isInteger(value) || value < 0 || value > 100) {
    throw new ProjectSettingsError(
      "projectBuffer",
      "Project buffer must be a whole number between 0 and 100",
    );
  }
  return value;
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
