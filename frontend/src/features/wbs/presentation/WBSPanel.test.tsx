import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { Project } from "../../projects/domain/project";
import type { RolesGateway } from "../../roles/application/rolesGateway";
import type { TeamMembersGateway } from "../../team-members/application/teamMembersGateway";
import { WBSOperationError, type WBSGateway } from "../application/wbsGateway";
import type { WBSNode } from "../domain/wbs";
import { WBSPanel } from "./WBSPanel";

const project: Project = {
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
};
const options = {
  rolesGateway: {
    list: vi
      .fn()
      .mockResolvedValue({ items: [], page: 1, pageSize: 100, total: 0 }),
  } as unknown as RolesGateway,
  membersGateway: {
    list: vi
      .fn()
      .mockResolvedValue({ items: [], page: 1, pageSize: 100, total: 0 }),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  } as unknown as TeamMembersGateway,
};
function node(id: string, name: string, children: WBSNode[] = []): WBSNode {
  return {
    id,
    projectId: "project",
    name,
    position: 1,
    hasChildren: children.length > 0,
    executable: { executionTimeline: {}, commitmentTimeline: {} },
    children,
  };
}
function gateway(tree: WBSNode[]): WBSGateway {
  return {
    tree: vi.fn().mockResolvedValue(tree),
    create: vi.fn().mockResolvedValue(undefined),
    rename: vi.fn(),
    reorder: vi.fn(),
    move: vi.fn(),
    remove: vi.fn(),
    updateExecutable: vi.fn(),
    complete: vi.fn(),
  };
}

describe("WBS presentation terminology", () => {
  it("uses Project Structure, Task, Group, and the product empty state", async () => {
    const empty = gateway([]);
    const view = render(
      <WBSPanel
        project={project}
        gateway={empty}
        {...options}
        onClose={() => undefined}
      />,
    );
    expect(
      await screen.findByRole("heading", { name: "Project Structure" }),
    ).not.toBeNull();
    expect(await screen.findByText("No tasks or groups yet")).not.toBeNull();
    expect(
      screen.getAllByRole("button", { name: /Add Task/ }).length,
    ).toBeGreaterThan(0);
    expect(view.container.textContent).not.toMatch(
      /Executable WBS|Grouping WBS|Add WBS/,
    );
    view.unmount();
    render(
      <WBSPanel
        project={project}
        gateway={gateway([
          node("group", "Development", [node("task", "Backend API")]),
        ])}
        {...options}
        onClose={() => undefined}
      />,
    );
    expect(await screen.findByText("Group")).not.toBeNull();
    expect(screen.getByText("Task")).not.toBeNull();
    expect(screen.getAllByRole("button", { name: "Edit Task" })).toHaveLength(
      1,
    );
  });
  it("shows an explicit user-facing conversion confirmation", async () => {
    const task = node("task", "Backend API");
    const api = gateway([task]);
    vi.mocked(api.create)
      .mockRejectedValueOnce(
        new WBSOperationError("WBS_CONVERSION_REQUIRED", "technical message"),
      )
      .mockResolvedValueOnce(undefined);
    render(
      <WBSPanel
        project={project}
        gateway={api}
        {...options}
        onClose={() => undefined}
      />,
    );
    await screen.findByText("Backend API");
    fireEvent.click(screen.getByRole("button", { name: "Add Child" }));
    fireEvent.change(screen.getByLabelText("Name"), {
      target: { value: "API tests" },
    });
    fireEvent.submit(screen.getByLabelText("Name").closest("form")!);
    expect(
      await screen.findByRole("heading", {
        name: "This task will become a group",
      }),
    ).not.toBeNull();
    expect(
      screen.getByText(/current task details will be moved/),
    ).not.toBeNull();
    fireEvent.click(
      screen.getByRole("button", { name: "Add Child and Convert" }),
    );
    await waitFor(() =>
      expect(api.create).toHaveBeenLastCalledWith(
        "project",
        "task",
        "API tests",
        true,
      ),
    );
  });
});
