export interface Role {
  id: string;
  name: string;
  createdAt: Date;
  updatedAt: Date;
}

export type RoleNameErrorCode =
  "ROLE_NAME_REQUIRED" | "ROLE_NAME_TOO_LONG" | "ROLE_NAME_INVALID";

export class RoleNameError extends Error {
  constructor(
    readonly code: RoleNameErrorCode,
    message: string,
  ) {
    super(message);
  }
}

export function normalizeRoleName(name: string): string {
  const normalized = name.trim();
  if (normalized.length === 0) {
    throw new RoleNameError("ROLE_NAME_REQUIRED", "Role name is required");
  }
  if ([...normalized].length > 100) {
    throw new RoleNameError(
      "ROLE_NAME_TOO_LONG",
      "Role name must not exceed 100 characters",
    );
  }
  if (!/^[\p{L}\p{N} \-/()]+$/u.test(normalized)) {
    throw new RoleNameError(
      "ROLE_NAME_INVALID",
      "Role name contains unsupported characters",
    );
  }
  return normalized;
}

export function sortRolesByName(roles: Role[]): Role[] {
  return [...roles].sort((left, right) =>
    left.name.localeCompare(right.name, undefined, { sensitivity: "base" }),
  );
}
