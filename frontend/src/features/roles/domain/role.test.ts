import { describe, expect, it } from "vitest";

import {
  normalizeRoleName,
  RoleNameError,
  sortRolesByName,
  type Role,
} from "./role";

describe("role domain", () => {
  it("trims and validates role names", () => {
    expect(normalizeRoleName("  Backend Engineer  ")).toBe("Backend Engineer");
    expect(normalizeRoleName("QA / Automation (L2)")).toBe(
      "QA / Automation (L2)",
    );
  });

  it("rejects empty, long, and unsupported names", () => {
    expect(() => normalizeRoleName("   ")).toThrowError(
      new RoleNameError("ROLE_NAME_REQUIRED", "Role name is required"),
    );
    expect(() => normalizeRoleName("a".repeat(101))).toThrow(
      "Role name must not exceed 100 characters",
    );
    expect(() => normalizeRoleName("Backend!")).toThrow(
      "Role name contains unsupported characters",
    );
  });

  it("sorts roles case-insensitively without mutating the source", () => {
    const roles = [
      role("qa", "QA"),
      role("backend", "backend"),
      role("frontend", "Frontend"),
    ];

    expect(sortRolesByName(roles).map((item) => item.name)).toEqual([
      "backend",
      "Frontend",
      "QA",
    ]);
    expect(roles[0].name).toBe("QA");
  });
});

function role(id: string, name: string): Role {
  return {
    id,
    name,
    createdAt: new Date("2026-07-24T10:00:00Z"),
    updatedAt: new Date("2026-07-24T10:00:00Z"),
  };
}
