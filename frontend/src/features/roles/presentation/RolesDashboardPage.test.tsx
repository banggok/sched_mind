import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { AppShell } from "../../../app/AppShell";
import type { RoleAuditGateway } from "../application/rolesGateway";
import type { Role } from "../domain/role";
import { RolesDashboardPage } from "./RolesDashboardPage";

describe("RolesDashboardPage", () => {
  it("shows the available workspace navigation and current location", async () => {
    const user = userEvent.setup();
    const fixture = gatewayFixture([]);
    render(
      <AppShell activePage="roles">
        <RolesDashboardPage gateway={fixture.gateway} />
      </AppShell>,
    );

    expect(
      screen.getByRole("link", { name: "SchedMind home" }).getAttribute("href"),
    ).toBe("/");
    const toggle = screen.getByRole("button", { name: "Expand navigation" });
    expect(toggle.getAttribute("aria-expanded")).toBe("false");
    await user.click(toggle);
    expect(
      screen
        .getByRole("button", { name: "Collapse navigation" })
        .getAttribute("aria-expanded"),
    ).toBe("true");

    const navigation = screen.getByRole("navigation", {
      name: "Application navigation",
    });
    expect(
      within(navigation).getByRole("region", {
        name: "Team Configuration",
      }),
    ).toBeTruthy();
    const current = within(navigation).getByRole("link", { name: "Roles" });
    await user.click(within(navigation).getByRole("link", { name: "Members" }));
    expect(
      screen
        .getByRole("button", { name: "Collapse navigation" })
        .getAttribute("aria-expanded"),
    ).toBe("true");

    expect(current.getAttribute("href")).toBe("#roles");
    expect(current.getAttribute("aria-current")).toBe("page");
    expect(
      within(screen.getByRole("navigation", { name: "Breadcrumb" })).getByText(
        "Roles",
      ),
    ).toBeTruthy();
    expect(await screen.findByText("No roles yet")).toBeTruthy();
  });

  it("renders empty state, validates input, and creates a role", async () => {
    const user = userEvent.setup();
    const fixture = gatewayFixture([]);
    render(<RolesDashboardPage gateway={fixture.gateway} />);

    expect(await screen.findByText("No roles yet")).toBeTruthy();
    await user.click(screen.getAllByRole("button", { name: "Add Role" })[0]);
    await user.click(screen.getByRole("button", { name: "Add role" }));

    expect(await screen.findByText("Role name is required")).toBeTruthy();
    expect(fixture.gateway.create).not.toHaveBeenCalled();

    await user.type(screen.getByLabelText("Role name"), "  Backend  ");
    await user.click(screen.getByRole("button", { name: "Add role" }));

    expect(await screen.findByText("Role created successfully")).toBeTruthy();
    expect(fixture.gateway.create).toHaveBeenCalledWith("Backend");
    expect(await screen.findByText("Backend")).toBeTruthy();
  });

  it("filters roles and distinguishes no search results from no data", async () => {
    const user = userEvent.setup();
    const fixture = gatewayFixture([
      role("android", "Android"),
      role("backend", "Backend"),
    ]);
    render(<RolesDashboardPage gateway={fixture.gateway} />);

    const search = await screen.findByLabelText("Search roles");
    await user.type(search, "back");
    expect(screen.getByRole("status", { name: "Loading roles" })).toBeTruthy();
    await waitFor(() =>
      expect(
        screen.queryByRole("status", { name: "Loading roles" }),
      ).toBeNull(),
    );
    expect(await screen.findByText("Backend")).toBeTruthy();
    expect(screen.queryByText("Android")).toBeNull();

    await user.clear(search);
    await user.type(search, "web");
    expect(await screen.findByText("No matching roles")).toBeTruthy();
    await user.click(screen.getByRole("button", { name: "Clear search" }));
    expect(await screen.findByText("Android")).toBeTruthy();
  });

  it("edits a role and refreshes the list", async () => {
    const user = userEvent.setup();
    const fixture = gatewayFixture([role("backend", "Backend")]);
    render(<RolesDashboardPage gateway={fixture.gateway} />);

    await user.click(
      await screen.findByRole("button", { name: "Edit Backend" }),
    );
    const input = screen.getByLabelText("Role name");
    await user.clear(input);
    await user.type(input, "Backend Engineer");
    await user.click(screen.getByRole("button", { name: "Save changes" }));

    expect(await screen.findByText("Role updated successfully")).toBeTruthy();
    expect(fixture.gateway.update).toHaveBeenCalledWith(
      "backend",
      "Backend Engineer",
    );
    expect(await screen.findByText("Backend Engineer")).toBeTruthy();
  });

  it("shows the specified duplicate-name message without exposing API details", async () => {
    const user = userEvent.setup();
    const fixture = gatewayFixture([]);
    fixture.gateway.create = vi.fn().mockRejectedValue({
      code: "ROLE_NAME_ALREADY_EXISTS",
      message: "unique index roles_name_ci_unique",
    });
    render(<RolesDashboardPage gateway={fixture.gateway} />);

    await screen.findByText("No roles yet");
    await user.click(screen.getAllByRole("button", { name: "Add Role" })[0]);
    await user.type(screen.getByLabelText("Role name"), "Backend");
    await user.click(screen.getByRole("button", { name: "Add role" }));

    expect(await screen.findByText("Role name already exists")).toBeTruthy();
    expect(screen.queryByText("unique index roles_name_ci_unique")).toBeNull();
  });

  it("cancels deletion and reports an in-use failure", async () => {
    const user = userEvent.setup();
    const fixture = gatewayFixture([role("backend", "Backend")]);
    fixture.gateway.delete = vi
      .fn()
      .mockRejectedValue({ code: "ROLE_IN_USE", message: "database detail" });
    render(<RolesDashboardPage gateway={fixture.gateway} />);

    await user.click(
      await screen.findByRole("button", { name: "Delete Backend" }),
    );
    const firstDialog = screen.getByRole("alertdialog");
    await user.click(
      within(firstDialog).getByRole("button", { name: "Cancel" }),
    );
    expect(fixture.gateway.delete).not.toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: "Delete Backend" }));
    const secondDialog = screen.getByRole("alertdialog");
    await user.click(
      within(secondDialog).getByRole("button", { name: "Delete role" }),
    );

    expect(
      await screen.findByText(
        "Role is assigned to one or more active team members and cannot be deleted",
      ),
    ).toBeTruthy();
    expect(screen.getByRole("alertdialog")).toBeTruthy();
    expect(screen.queryByText("database detail")).toBeNull();
    expect(fixture.roles).toHaveLength(1);
  });

  it("reports Task usage separately from Member usage when deletion is blocked", async () => {
    const user = userEvent.setup();
    const fixture = gatewayFixture([role("backend", "Backend")]);
    fixture.gateway.delete = vi.fn().mockRejectedValue({
      code: "ROLE_IN_USE_BY_TASK",
      message: "database detail",
    });
    render(<RolesDashboardPage gateway={fixture.gateway} />);

    await user.click(
      await screen.findByRole("button", { name: "Delete Backend" }),
    );
    await user.click(
      within(screen.getByRole("alertdialog")).getByRole("button", {
        name: "Delete role",
      }),
    );

    expect(
      await screen.findByText(
        "Role is assigned to one or more tasks and cannot be deleted",
      ),
    ).toBeTruthy();
    expect(screen.queryByText("database detail")).toBeNull();
  });

  it("lists active members using a role for audit", async () => {
    const user = userEvent.setup();
    const fixture = gatewayFixture([role("backend", "Backend")]);
    const auditMembers = [
      { id: "member-1", name: "Ayu" },
      { id: "member-2", name: "Bima" },
    ];
    fixture.gateway.listMembers = vi
      .fn()
      .mockImplementation(async (_roleId, query) => {
        const matching = auditMembers.filter((member) =>
          member.name.toLowerCase().startsWith(query.search.toLowerCase()),
        );
        return {
          items: matching,
          page: query.page,
          pageSize: query.pageSize,
          total: matching.length,
        };
      });
    render(<RolesDashboardPage gateway={fixture.gateway} />);

    await user.click(
      await screen.findByRole("button", { name: "View members using Backend" }),
    );

    const dialog = screen.getByRole("dialog", {
      name: "Members using Backend",
    });
    expect(within(dialog).getByText("Ayu")).toBeTruthy();
    expect(within(dialog).getByText("Bima")).toBeTruthy();
    expect(within(dialog).getByText("1–2 of 2")).toBeTruthy();
    expect(fixture.gateway.listMembers).toHaveBeenCalledWith(
      "backend",
      { search: "", page: 1, pageSize: 10 },
      expect.any(AbortSignal),
    );

    await user.type(within(dialog).getByLabelText("Search members"), "bi");
    expect(await within(dialog).findByText("Bima")).toBeTruthy();
    expect(within(dialog).queryByText("Ayu")).toBeNull();
    expect(fixture.gateway.listMembers).toHaveBeenLastCalledWith(
      "backend",
      { search: "bi", page: 1, pageSize: 10 },
      expect.any(AbortSignal),
    );

    await user.click(within(dialog).getByRole("button", { name: "Close" }));
    await waitFor(() =>
      expect(
        screen.queryByRole("dialog", { name: "Members using Backend" }),
      ).toBeNull(),
    );
  });

  it("shows an audit empty state when no active member uses the role", async () => {
    const user = userEvent.setup();
    const fixture = gatewayFixture([role("qa", "QA")]);
    render(<RolesDashboardPage gateway={fixture.gateway} />);

    await user.click(
      await screen.findByRole("button", { name: "View members using QA" }),
    );

    expect(
      await screen.findByText("No active members use this role"),
    ).toBeTruthy();
    expect(
      screen.getByText(
        "No active team member uses this role. Tasks may still reference it.",
      ),
    ).toBeTruthy();
  });

  it("keeps audit failures local and retries without exposing backend details", async () => {
    const user = userEvent.setup();
    const fixture = gatewayFixture([role("backend", "Backend")]);
    fixture.gateway.listMembers = vi
      .fn()
      .mockRejectedValueOnce(new Error("postgres connection refused"))
      .mockResolvedValueOnce({
        items: [{ id: "member-1", name: "Ayu" }],
        page: 1,
        pageSize: 10,
        total: 1,
      });
    render(<RolesDashboardPage gateway={fixture.gateway} />);

    await user.click(
      await screen.findByRole("button", { name: "View members using Backend" }),
    );

    const dialog = screen.getByRole("dialog", {
      name: "Members using Backend",
    });
    expect(
      await within(dialog).findByText(
        "Unable to load members for this role. Try again.",
      ),
    ).toBeTruthy();
    expect(
      within(dialog).queryByText("postgres connection refused"),
    ).toBeNull();

    await user.click(within(dialog).getByRole("button", { name: "Try again" }));
    expect(await within(dialog).findByText("Ayu")).toBeTruthy();
    expect(fixture.gateway.listMembers).toHaveBeenCalledTimes(2);
  });

  it("shows a stale-role audit error instead of an empty member list", async () => {
    const user = userEvent.setup();
    const fixture = gatewayFixture([role("backend", "Backend")]);
    fixture.gateway.listMembers = vi.fn().mockRejectedValue({
      code: "ROLE_NOT_FOUND",
      message: "database detail",
    });
    render(<RolesDashboardPage gateway={fixture.gateway} />);

    await user.click(
      await screen.findByRole("button", { name: "View members using Backend" }),
    );

    const dialog = screen.getByRole("dialog", {
      name: "Members using Backend",
    });
    expect(
      await within(dialog).findByText(
        "This role no longer exists. Close this dialog and refresh the list.",
      ),
    ).toBeTruthy();
    expect(
      within(dialog).queryByText("No active members use this role"),
    ).toBeNull();
    expect(within(dialog).queryByText("database detail")).toBeNull();
  });

  it("maps a technical load failure and retries near the affected content", async () => {
    const user = userEvent.setup();
    const fixture = gatewayFixture([role("backend", "Backend")]);
    fixture.gateway.list = vi
      .fn()
      .mockRejectedValueOnce(new Error("postgres connection refused"))
      .mockResolvedValueOnce({
        items: [...fixture.roles],
        page: 1,
        pageSize: 5,
        total: fixture.roles.length,
      });

    render(<RolesDashboardPage gateway={fixture.gateway} />);

    expect(await screen.findByText("Unable to load roles")).toBeTruthy();
    expect(screen.queryByText("postgres connection refused")).toBeNull();
    await user.click(screen.getByRole("button", { name: "Try again" }));
    expect(await screen.findByText("Backend")).toBeTruthy();
  });

  it("prevents duplicate submission while create is pending", async () => {
    const user = userEvent.setup();
    const fixture = gatewayFixture([]);
    let resolveCreate: ((role: Role) => void) | undefined;
    fixture.gateway.create = vi.fn(
      () =>
        new Promise<Role>((resolve) => {
          resolveCreate = resolve;
        }),
    );
    render(<RolesDashboardPage gateway={fixture.gateway} />);

    await screen.findByText("No roles yet");
    await user.click(screen.getAllByRole("button", { name: "Add Role" })[0]);
    await user.type(screen.getByLabelText("Role name"), "Backend");
    const submit = screen.getByRole("button", { name: "Add role" });
    await user.click(submit);

    expect(
      (
        screen.getByRole("button", {
          name: "Saving…",
        }) as HTMLButtonElement
      ).disabled,
    ).toBe(true);
    await user.click(screen.getByRole("button", { name: "Saving…" }));
    expect(fixture.gateway.create).toHaveBeenCalledTimes(1);

    if (!resolveCreate) {
      throw new Error("create promise was not started");
    }
    resolveCreate(role("backend", "Backend"));
    expect(await screen.findByText("Role created successfully")).toBeTruthy();
  });

  it("supports Escape and restores focus without losing a dirty draft silently", async () => {
    const user = userEvent.setup();
    const fixture = gatewayFixture([]);
    const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
    render(<RolesDashboardPage gateway={fixture.gateway} />);

    await screen.findByText("No roles yet");
    const trigger = screen.getAllByRole("button", { name: "Add Role" })[0];
    await user.click(trigger);
    const input = screen.getByLabelText("Role name");
    expect(document.activeElement).toBe(input);

    await user.type(input, "Backend");
    await user.keyboard("{Escape}");
    expect(confirm).toHaveBeenCalledWith("Discard your unsaved role changes?");
    expect(screen.getByRole("dialog")).toBeTruthy();
    expect((input as HTMLInputElement).value).toBe("Backend");

    confirm.mockReturnValue(true);
    await user.keyboard("{Escape}");
    await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
    expect(document.activeElement).toBe(trigger);
    confirm.mockRestore();
  });
});

function gatewayFixture(initialRoles: Role[]) {
  const roles = [...initialRoles];
  const gateway: RoleAuditGateway = {
    list: vi.fn().mockImplementation(async (query) => {
      const matching = roles.filter((item) =>
        item.name.toLowerCase().startsWith(query.search.toLowerCase()),
      );
      return {
        items: matching,
        page: query.page,
        pageSize: query.pageSize,
        total: matching.length,
      };
    }),
    listMembers: vi.fn().mockImplementation(async (_roleId, query) => ({
      items: [],
      page: query.page,
      pageSize: query.pageSize,
      total: 0,
    })),
    create: vi.fn().mockImplementation(async (name: string) => {
      const created = role(`role-${roles.length + 1}`, name);
      roles.push(created);
      return created;
    }),
    update: vi.fn().mockImplementation(async (id: string, name: string) => {
      const index = roles.findIndex((item) => item.id === id);
      const updated = { ...roles[index], name, updatedAt: new Date() };
      roles[index] = updated;
      return updated;
    }),
    delete: vi.fn().mockImplementation(async (id: string) => {
      const index = roles.findIndex((item) => item.id === id);
      roles.splice(index, 1);
    }),
  };
  return { gateway, roles };
}

function role(id: string, name: string): Role {
  return {
    id,
    name,
    createdAt: new Date("2026-07-24T10:00:00Z"),
    updatedAt: new Date("2026-07-24T10:00:00Z"),
  };
}
