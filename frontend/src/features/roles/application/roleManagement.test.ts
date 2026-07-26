import { describe, expect, it, vi } from "vitest";

import type { Role } from "../domain/role";
import {
  createRole,
  deleteRole,
  listRoles,
  updateRole,
} from "./roleManagement";
import type { RolesGateway } from "./rolesGateway";

describe("role management workflows", () => {
  it("sorts list results and normalizes mutation input", async () => {
    const gateway = gatewayStub();
    gateway.list = vi.fn().mockResolvedValue({
      items: [role("qa", "QA"), role("backend", "Backend")],
      page: 1,
      pageSize: 20,
      total: 2,
    });
    gateway.create = vi.fn().mockResolvedValue(role("new", "Frontend"));
    gateway.update = vi
      .fn()
      .mockResolvedValue(role("new", "Frontend Engineer"));

    await expect(listRoles(gateway, query)).resolves.toMatchObject({
      items: [role("backend", "Backend"), role("qa", "QA")],
    });
    await createRole(gateway, "  Frontend  ");
    await updateRole(gateway, "new", " Frontend Engineer ");

    expect(gateway.create).toHaveBeenCalledWith("Frontend");
    expect(gateway.update).toHaveBeenCalledWith("new", "Frontend Engineer");
  });

  it("does not call dependencies for invalid input and delegates delete", async () => {
    const gateway = gatewayStub();

    await expect(createRole(gateway, "   ")).rejects.toThrow(
      "Role name is required",
    );
    expect(gateway.create).not.toHaveBeenCalled();

    await deleteRole(gateway, "role-id");
    expect(gateway.delete).toHaveBeenCalledWith("role-id");
  });
});

function gatewayStub(): RolesGateway {
  return {
    list: vi
      .fn()
      .mockResolvedValue({ items: [], page: 1, pageSize: 20, total: 0 }),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn().mockResolvedValue(undefined),
  };
}
const query = { search: "", page: 1, pageSize: 20 };
function role(id: string, name: string): Role {
  return {
    id,
    name,
    createdAt: new Date("2026-07-24T10:00:00Z"),
    updatedAt: new Date("2026-07-24T10:00:00Z"),
  };
}
