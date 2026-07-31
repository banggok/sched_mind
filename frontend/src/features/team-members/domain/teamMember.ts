export interface TeamMemberRole {
  id: string;
  name: string;
}

export interface TeamMember {
  id: string;
  name: string;
  role: TeamMemberRole;
  dailyCapacity: number;
  bufferPercentage: number;
  baseExecutionCapacity: number;
  createdAt: Date;
  updatedAt: Date;
}

export interface TeamMemberInput {
  name: string;
  roleId: string;
  dailyCapacity: number | undefined;
  bufferPercentage: number | undefined;
}

export type TeamMemberField =
  "name" | "roleId" | "dailyCapacity" | "bufferPercentage";

export class TeamMemberValidationError extends Error {
  constructor(
    readonly field: TeamMemberField,
    readonly code: string,
    message: string,
  ) {
    super(message);
  }
}

export function normalizeTeamMemberInput(
  input: TeamMemberInput,
): Required<TeamMemberInput> {
  const name = input.name.trim();
  if (!name) {
    throw new TeamMemberValidationError(
      "name",
      "TEAM_MEMBER_NAME_REQUIRED",
      "Team member name is required",
    );
  }
  if ([...name].length > 100) {
    throw new TeamMemberValidationError(
      "name",
      "TEAM_MEMBER_NAME_TOO_LONG",
      "Team member name must not exceed 100 characters",
    );
  }
  if (!input.roleId) {
    throw new TeamMemberValidationError(
      "roleId",
      "ROLE_REQUIRED",
      "Role is required",
    );
  }
  if (input.dailyCapacity === undefined) {
    throw new TeamMemberValidationError(
      "dailyCapacity",
      "DAILY_CAPACITY_REQUIRED",
      "Daily capacity is required",
    );
  }
  if (input.dailyCapacity <= 0) {
    throw new TeamMemberValidationError(
      "dailyCapacity",
      "DAILY_CAPACITY_NOT_POSITIVE",
      "Daily capacity must be greater than 0",
    );
  }
  if (input.dailyCapacity > 24) {
    throw new TeamMemberValidationError(
      "dailyCapacity",
      "DAILY_CAPACITY_EXCEEDS_LIMIT",
      "Daily capacity must not exceed 24 hours",
    );
  }
  if (!Number.isInteger(input.dailyCapacity * 2)) {
    throw new TeamMemberValidationError(
      "dailyCapacity",
      "DAILY_CAPACITY_INVALID_INCREMENT",
      "Daily capacity must use 0.5-hour increments",
    );
  }
  const bufferPercentage = input.bufferPercentage ?? 20;
  if (
    bufferPercentage < 0 ||
    bufferPercentage >= 100 ||
    !Number.isInteger(bufferPercentage * 100)
  ) {
    throw new TeamMemberValidationError(
      "bufferPercentage",
      "BUFFER_OUT_OF_RANGE",
      "Buffer must be between 0 and less than 100",
    );
  }
  return {
    name,
    roleId: input.roleId,
    dailyCapacity: input.dailyCapacity,
    bufferPercentage,
  };
}

export function calculateBaseExecutionCapacity(
  dailyCapacity: number,
  bufferPercentage: number,
): number {
  const dailyHalfHours = Math.round(dailyCapacity * 2);
  const bufferBasisPoints = Math.round(bufferPercentage * 100);
  const rawHalfHours = (dailyHalfHours * (10_000 - bufferBasisPoints)) / 10_000;
  return Math.round(rawHalfHours) / 2;
}

export function sortTeamMembers(members: TeamMember[]): TeamMember[] {
  return [...members].sort((left, right) => {
    const byName = left.name.localeCompare(right.name, undefined, {
      sensitivity: "base",
    });
    return byName || left.id.localeCompare(right.id);
  });
}
