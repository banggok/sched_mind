import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import type { TeamMembersGateway } from "../application/teamMembersGateway";
import type { RoleOptionsGateway } from "../application/roleOptionsGateway";
import { TeamMembersDashboardPage } from "./TeamMembersDashboardPage";

describe("TeamMembersDashboardPage", () => {
  it("filters members and provides a clear no-results state", async () => {
    const user = userEvent.setup();
    const rolesGateway = emptyRolesGateway();
    const gateway: TeamMembersGateway = {
      list: vi.fn().mockImplementation(async (query) => {
        const items = [member("andi", "Andi"), member("budi", "Budi")].filter(
          (item) =>
            item.name.toLowerCase().startsWith(query.search.toLowerCase()),
        );
        return {
          items,
          page: query.page,
          pageSize: query.pageSize,
          total: items.length,
        };
      }),
      create: vi.fn(),
      update: vi.fn(),
      delete: vi.fn(),
    };
    render(
      <TeamMembersDashboardPage
        gateway={gateway}
        roleOptionsGateway={rolesGateway}
        onManageCapacity={vi.fn()}
      />,
    );

    const search = await screen.findByLabelText("Search members");
    await user.type(search, "bud");
    expect(
      screen.getByRole("status", { name: "Loading members" }),
    ).toBeTruthy();
    await waitFor(() =>
      expect(
        screen.queryByRole("status", { name: "Loading members" }),
      ).toBeNull(),
    );
    expect(await screen.findByText("Budi")).toBeTruthy();
    expect(screen.queryByText("Andi")).toBeNull();
    await user.clear(search);
    await user.type(search, "sari");
    expect(await screen.findByText("No matching members")).toBeTruthy();
    await user.click(screen.getByRole("button", { name: "Clear search" }));
    expect(await screen.findByText("Andi")).toBeTruthy();
  });

  it("starts with no role selected and accepts a .5 capacity draft", async () => {
    const user = userEvent.setup();
    const rolesGateway: RoleOptionsGateway = {
      list: vi.fn().mockResolvedValue({
        items: [
          {
            id: "backend",
            name: "Backend",
            createdAt: new Date("2026-07-25T00:00:00Z"),
            updatedAt: new Date("2026-07-25T00:00:00Z"),
          },
          {
            id: "android",
            name: "Android",
            createdAt: new Date("2026-07-25T00:00:00Z"),
            updatedAt: new Date("2026-07-25T00:00:00Z"),
          },
        ],
        page: 1,
        pageSize: 100,
        total: 2,
      }),
    };
    const gateway: TeamMembersGateway = {
      list: vi
        .fn()
        .mockResolvedValue({ items: [], page: 1, pageSize: 5, total: 0 }),
      create: vi.fn().mockResolvedValue({
        id: "member-1",
        name: "Rizki",
        role: { id: "backend", name: "Backend" },
        dailyCapacity: 7.5,
        bufferPercentage: 20,
        baseExecutionCapacity: 6,
        createdAt: new Date(),
        updatedAt: new Date(),
      }),
      update: vi.fn(),
      delete: vi.fn(),
    };

    render(
      <TeamMembersDashboardPage
        gateway={gateway}
        roleOptionsGateway={rolesGateway}
        onManageCapacity={vi.fn()}
      />,
    );

    await screen.findByText("No members yet");
    expect(screen.getByRole("button", { name: "Add Member" })).toBeTruthy();
    await user.click(screen.getByRole("button", { name: "+ Add Member" }));

    const role = screen.getByLabelText("Role") as HTMLInputElement;
    expect(role.value).toBe("");

    const name = screen.getByLabelText("Name") as HTMLInputElement;
    await user.type(name, "rizki triamoro");
    expect(name.value).toBe("rizki triamoro");
    await user.click(role);
    expect(name.value).toBe("Rizki Triamoro");
    expect(screen.getByRole("option", { name: "Backend" })).toBeTruthy();
    expect(screen.getByRole("option", { name: "Android" })).toBeTruthy();
    await user.type(role, "Back");
    expect(screen.getByRole("option", { name: "Backend" })).toBeTruthy();
    expect(screen.queryByRole("option", { name: "Android" })).toBeNull();
    await user.keyboard("{Enter}");
    expect(role.value).toBe("Backend");
    expect(gateway.create).not.toHaveBeenCalled();
    const capacity = screen.getByLabelText(
      "Daily capacity (hours)",
    ) as HTMLInputElement;
    expect(document.activeElement).toBe(capacity);
    await user.clear(capacity);
    await user.type(capacity, "7.3");
    expect(capacity.value).toBe("7.3");
    await user.tab();
    expect(capacity.value).toBe("7.5");
    const buffer = screen.getByLabelText("Buffer (%)") as HTMLInputElement;
    await user.clear(buffer);
    await user.type(buffer, "20.3");
    expect(buffer.value).toBe("20.3");
    await user.tab();
    expect(buffer.value).toBe("20.5");
    expect(
      screen.getByText("Base execution capacity: 6 hours/day"),
    ).toBeTruthy();

    await user.click(screen.getByRole("button", { name: "Save" }));
    expect(gateway.create).toHaveBeenCalledWith({
      name: "Rizki Triamoro",
      roleId: "backend",
      dailyCapacity: 7.5,
      bufferPercentage: 20.5,
    });
  });

  it("allows deleting a member whose child capacity overrides are cascaded by the backend", async () => {
    const user = userEvent.setup();
    let deleted = false;
    const gateway: TeamMembersGateway = {
      list: vi.fn().mockImplementation(async () => ({
        items: deleted ? [] : [member("harry", "Harry")],
        page: 1,
        pageSize: 5,
        total: deleted ? 0 : 1,
      })),
      create: vi.fn(),
      update: vi.fn(),
      delete: vi.fn().mockImplementation(async () => {
        deleted = true;
      }),
    };
    render(
      <TeamMembersDashboardPage
        gateway={gateway}
        roleOptionsGateway={emptyRolesGateway()}
        onManageCapacity={vi.fn()}
      />,
    );

    expect(await screen.findByText("Harry")).toBeTruthy();
    await user.click(screen.getByRole("button", { name: "Delete" }));
    expect(
      screen.getByText(
        /capacity overrides will be removed from active planning/i,
      ),
    ).toBeTruthy();
    await user.click(screen.getByRole("button", { name: "Delete member" }));

    await waitFor(() => expect(gateway.delete).toHaveBeenCalledWith("harry"));
    expect(await screen.findByText("No members yet")).toBeTruthy();
  });
});

function emptyRolesGateway(): RoleOptionsGateway {
  return {
    list: vi
      .fn()
      .mockResolvedValue({ items: [], page: 1, pageSize: 100, total: 0 }),
  };
}

function member(id: string, name: string) {
  return {
    id,
    name,
    role: { id: "backend", name: "Backend" },
    dailyCapacity: 8,
    bufferPercentage: 20,
    baseExecutionCapacity: 6.5,
    createdAt: new Date("2026-07-25T00:00:00Z"),
    updatedAt: new Date("2026-07-25T00:00:00Z"),
  };
}
