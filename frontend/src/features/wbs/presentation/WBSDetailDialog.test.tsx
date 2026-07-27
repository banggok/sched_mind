import { StrictMode } from "react";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { RolesGateway } from "../../roles/application/rolesGateway";
import type { TeamMembersGateway } from "../../team-members/application/teamMembersGateway";
import type { WBSGateway } from "../application/wbsGateway";
import type { WBSNode } from "../domain/wbs";
import { WBSDetailDialog } from "./WBSDetailDialog";

describe("WBSDetailDialog option loading", () => {
  it("does not report Strict Mode cancellation as a load failure", async () => {
    const rolesGateway = {
      list: vi.fn((_query, signal?: AbortSignal) =>
        abortable(
          {
            items: [
              {
                id: "role",
                name: "Backend",
                createdAt: new Date(),
                updatedAt: new Date(),
              },
            ],
            page: 1,
            pageSize: 100,
            total: 1,
          },
          signal,
        ),
      ),
    } as unknown as RolesGateway;
    const membersGateway = {
      list: vi.fn((_query, signal?: AbortSignal) =>
        abortable({ items: [], page: 1, pageSize: 100, total: 0 }, signal),
      ),
    } as unknown as TeamMembersGateway;
    const gateway = {} as WBSGateway;
    const node: WBSNode = {
      id: "task",
      projectId: "project",
      name: "API",
      position: 1,
      hasChildren: false,
      executable: { executionTimeline: {}, commitmentTimeline: {} },
      children: [],
    };
    render(
      <StrictMode>
        <WBSDetailDialog
          project={{
            id: "project",
            name: "Alpha",
            status: "open",
            autoCalculateDate: true,
            autoDependencyByAssignee: true,
            automaticScheduling: true,
            projectBuffer: 20,
            priority: 1,
            createdAt: new Date(),
            updatedAt: new Date(),
          }}
          node={node}
          gateway={gateway}
          rolesGateway={rolesGateway}
          membersGateway={membersGateway}
          onClose={() => undefined}
          onChanged={() => undefined}
        />
      </StrictMode>,
    );
    await waitFor(() =>
      expect(screen.getByRole("option", { name: "Backend" })).not.toBeNull(),
    );
    expect(
      screen.queryByText("Role and member options could not be loaded."),
    ).toBeNull();
  });

  it("saves the task name and executable details through one update", async () => {
    const node: WBSNode = {
      id: "task",
      projectId: "project",
      name: "API",
      position: 1,
      hasChildren: false,
      executable: { executionTimeline: {}, commitmentTimeline: {} },
      children: [],
    };
    const gateway = {
      updateExecutable: vi.fn().mockResolvedValue(undefined),
    } as unknown as WBSGateway;
    const rolesGateway = {
      list: vi
        .fn()
        .mockResolvedValue({ items: [], page: 1, pageSize: 100, total: 0 }),
    } as unknown as RolesGateway;
    const membersGateway = {
      list: vi
        .fn()
        .mockResolvedValue({ items: [], page: 1, pageSize: 100, total: 0 }),
    } as unknown as TeamMembersGateway;
    render(
      <WBSDetailDialog
        project={{
          id: "project",
          name: "Alpha",
          status: "open",
          autoCalculateDate: true,
          autoDependencyByAssignee: true,
          automaticScheduling: true,
          projectBuffer: 20,
          priority: 1,
          createdAt: new Date(),
          updatedAt: new Date(),
        }}
        node={node}
        gateway={gateway}
        rolesGateway={rolesGateway}
        membersGateway={membersGateway}
        onClose={() => undefined}
        onChanged={() => undefined}
      />,
    );
    fireEvent.change(screen.getByLabelText("Name"), {
      target: { value: "Backend API" },
    });
    const effort = screen.getByLabelText("Effort (hours)");
    fireEvent.change(effort, { target: { value: "6.6666" } });
    expect((effort as HTMLInputElement).value).toBe("6.6666");
    fireEvent.blur(effort);
    expect((effort as HTMLInputElement).value).toBe("6.5");
    fireEvent.change(effort, { target: { value: "6.5x" } });
    expect((effort as HTMLInputElement).value).toBe("6.5");
    expect(
      screen.getByRole("button", {
        name: "Execution timeline: Select start and end date",
      }),
    ).toBeTruthy();
    expect(
      screen.getByRole("button", {
        name: "Commitment timeline: Select start and end date",
      }),
    ).toBeTruthy();
    expect(
      screen.getByRole("button", { name: "Actual End: Select date" }),
    ).toBeTruthy();
    fireEvent.submit(screen.getByLabelText("Name").closest("form")!);
    await waitFor(() =>
      expect(gateway.updateExecutable).toHaveBeenCalledWith(
        "project",
        "task",
        expect.objectContaining({ name: "Backend API", effortHours: 6.5 }),
      ),
    );
  });
});

function abortable<T>(value: T, signal?: AbortSignal): Promise<T> {
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => resolve(value), 0);
    signal?.addEventListener(
      "abort",
      () => {
        clearTimeout(timer);
        reject(new DOMException("aborted", "AbortError"));
      },
      { once: true },
    );
  });
}
