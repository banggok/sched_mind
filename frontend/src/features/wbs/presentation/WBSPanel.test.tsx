import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { DependenciesGateway } from "../../dependencies/application/dependenciesGateway";
import type { DependencyDetail } from "../../dependencies/domain/dependency";
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
  automaticScheduling: true,
  projectBuffer: 20,
  scheduleVersion: 0,
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
    executable: { lagDays: 0, executionTimeline: {}, commitmentTimeline: {} },
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
    previewExecutableSchedule: vi.fn(),
    complete: vi.fn(),
    reopen: vi.fn(),
  };
}

describe("WBS presentation terminology", () => {
  it("AC-1 shows one dependency from both directions through contextual Edit Task", async () => {
    const dependencyId = "dependency-ab";
    const taskA = node("a", "API");
    const taskB = node("b", "Build");
    const dependenciesGateway: DependenciesGateway = {
      list: vi.fn(async (taskId: string): Promise<DependencyDetail> => {
        if (taskId === taskA.id) {
          return {
            blockedBy: [],
            blocks: [
              {
                id: dependencyId,
                source: "manual",
                manualRemovable: true,
                task: {
                  id: taskB.id,
                  name: taskB.name,
                  projectId: project.id,
                  projectName: project.name,
                  hierarchyPath: `${project.name} > ${taskB.name}`,
                  completed: false,
                },
              },
            ],
          };
        }
        if (taskId === taskB.id) {
          return {
            blockedBy: [
              {
                id: dependencyId,
                source: "manual",
                manualRemovable: true,
                task: {
                  id: taskA.id,
                  name: taskA.name,
                  projectId: project.id,
                  projectName: project.name,
                  hierarchyPath: `${project.name} > ${taskA.name}`,
                  completed: false,
                },
              },
            ],
            blocks: [],
          };
        }
        throw new Error(`unexpected task ${taskId}`);
      }),
      candidates: vi.fn().mockResolvedValue({
        items: [],
        page: 1,
        pageSize: 5,
        totalItems: 0,
      }),
      create: vi.fn().mockResolvedValue(undefined),
      remove: vi.fn().mockResolvedValue(undefined),
      keepAsManual: vi.fn().mockResolvedValue(undefined),
      invalidateTask: vi.fn(),
      invalidateAll: vi.fn(),
    };

    render(
      <WBSPanel
        project={project}
        gateway={gateway([taskA, taskB])}
        dependenciesGateway={dependenciesGateway}
        {...options}
        onClose={() => undefined}
      />,
    );

    const tree = await screen.findByRole("tree");
    const apiTreeItem = within(tree)
      .getByText("API")
      .closest<HTMLElement>('[role="treeitem"]');

    expect(apiTreeItem).not.toBeNull();
    if (!apiTreeItem) {
      throw new Error("API tree item was not found");
    }

    fireEvent.click(
      within(apiTreeItem).getByRole("button", { name: "Edit Task" }),
    );

    let detailDialog = await screen.findByRole("dialog", { name: "Edit Task" });
    const blocksHeading = await within(detailDialog).findByRole("heading", {
      name: "Blocks",
    });
    const blocksSection = blocksHeading.parentElement?.parentElement;
    expect(blocksSection).not.toBeNull();
    if (!blocksSection) return;
    expect(within(blocksSection).getByText("Build")).not.toBeNull();

    fireEvent.click(
      within(detailDialog).getByRole("button", { name: "Close" }),
    );

    const buildTreeItem = within(tree)
      .getByText("Build")
      .closest<HTMLElement>('[role="treeitem"]');

    expect(buildTreeItem).not.toBeNull();
    if (!buildTreeItem) {
      throw new Error("Build tree item was not found");
    }

    fireEvent.click(
      within(buildTreeItem).getByRole("button", { name: "Edit Task" }),
    );

    detailDialog = await screen.findByRole("dialog", { name: "Edit Task" });
    await waitFor(() => {
      expect(dependenciesGateway.list).toHaveBeenCalledWith(
        taskB.id,
        expect.any(AbortSignal),
      );
    });
    const blockedByHeading = await within(detailDialog).findByRole("heading", {
      name: "Blocked by",
    });
    const blockedBySection = blockedByHeading.parentElement?.parentElement;
    expect(blockedBySection).not.toBeNull();
    if (!blockedBySection) return;
    expect(within(blockedBySection).getByText("API")).not.toBeNull();
  });

  it("AC-2 keeps confirmed Task data visible while dependencies are loading", async () => {
    let resolveDependencies:
      | ((value: Awaited<ReturnType<DependenciesGateway["list"]>>) => void)
      | undefined;
    const pendingDependencies = new Promise<
      Awaited<ReturnType<DependenciesGateway["list"]>>
    >((resolve) => {
      resolveDependencies = resolve;
    });
    const task = node("task", "Backend API");
    const dependenciesGateway: DependenciesGateway = {
      list: vi.fn().mockReturnValue(pendingDependencies),
      candidates: vi.fn().mockResolvedValue({
        items: [],
        page: 1,
        pageSize: 5,
        totalItems: 0,
      }),
      create: vi.fn().mockResolvedValue(undefined),
      remove: vi.fn().mockResolvedValue(undefined),
      keepAsManual: vi.fn().mockResolvedValue(undefined),
      invalidateTask: vi.fn(),
      invalidateAll: vi.fn(),
    };

    render(
      <WBSPanel
        project={project}
        gateway={gateway([task])}
        dependenciesGateway={dependenciesGateway}
        {...options}
        onClose={() => undefined}
      />,
    );

    const tree = await screen.findByRole("tree");
    const taskTreeItem = within(tree)
      .getByText("Backend API")
      .closest<HTMLElement>('[role="treeitem"]');

    expect(taskTreeItem).not.toBeNull();
    if (!taskTreeItem) {
      throw new Error("Backend API tree item was not found");
    }

    fireEvent.click(
      within(taskTreeItem).getByRole("button", { name: "Edit Task" }),
    );

    const detailDialog = await screen.findByRole("dialog", {
      name: "Edit Task",
    });

    const nameInput = within(detailDialog).getByLabelText(
      "Name",
    ) as HTMLInputElement;
    expect(nameInput.value).toBe("Backend API");
    expect(
      within(detailDialog).getByRole("status", {
        name: "Loading dependencies",
      }),
    ).not.toBeNull();
    expect(within(detailDialog).queryByText("No dependencies.")).toBeNull();

    resolveDependencies?.({ blockedBy: [], blocks: [] });

    await waitFor(() => {
      expect(
        within(detailDialog).getAllByText("No dependencies."),
      ).toHaveLength(2);
    });
  });

  it("AC-3 shows empty dependency states and Add actions through contextual Edit Task", async () => {
    const task = node("task", "Backend API");
    const dependenciesGateway: DependenciesGateway = {
      list: vi.fn().mockResolvedValue({ blockedBy: [], blocks: [] }),
      candidates: vi.fn().mockResolvedValue({
        items: [],
        page: 1,
        pageSize: 5,
        totalItems: 0,
      }),
      create: vi.fn().mockResolvedValue(undefined),
      remove: vi.fn().mockResolvedValue(undefined),
      keepAsManual: vi.fn().mockResolvedValue(undefined),
      invalidateTask: vi.fn(),
      invalidateAll: vi.fn(),
    };

    render(
      <WBSPanel
        project={project}
        gateway={gateway([task])}
        dependenciesGateway={dependenciesGateway}
        {...options}
        onClose={() => undefined}
      />,
    );

    const tree = await screen.findByRole("tree");
    const taskTreeItem = within(tree)
      .getByText("Backend API")
      .closest<HTMLElement>('[role="treeitem"]');

    expect(taskTreeItem).not.toBeNull();
    if (!taskTreeItem) {
      throw new Error("Backend API tree item was not found");
    }

    fireEvent.click(
      within(taskTreeItem).getByRole("button", { name: "Edit Task" }),
    );

    const detailDialog = await screen.findByRole("dialog", {
      name: "Edit Task",
    });

    const blockedByHeading = await within(detailDialog).findByRole("heading", {
      name: "Blocked by",
    });
    const blocksHeading = within(detailDialog).getByRole("heading", {
      name: "Blocks",
    });
    const blockedBySection = blockedByHeading.parentElement?.parentElement;
    const blocksSection = blocksHeading.parentElement?.parentElement;

    expect(blockedBySection).not.toBeNull();
    expect(blocksSection).not.toBeNull();
    if (!blockedBySection || !blocksSection) {
      throw new Error("Dependency sections were not found");
    }

    expect(
      within(blockedBySection).getByText("No dependencies."),
    ).not.toBeNull();
    expect(within(blocksSection).getByText("No dependencies.")).not.toBeNull();

    const blockedByAdd = within(blockedBySection).getByRole("button", {
      name: "Add",
    });
    const blocksAdd = within(blocksSection).getByRole("button", {
      name: "Add",
    });

    fireEvent.click(blockedByAdd);
    expect(within(detailDialog).getByLabelText("Search tasks")).not.toBeNull();
    await waitFor(() => {
      expect(dependenciesGateway.candidates).toHaveBeenCalledWith(
        task.id,
        "blockedBy",
        "",
        1,
        5,
        expect.any(AbortSignal),
      );
    });
    expect(
      await within(detailDialog).findByText("No matching tasks."),
    ).not.toBeNull();

    fireEvent.click(
      within(detailDialog).getByRole("button", { name: "Cancel" }),
    );
    fireEvent.click(blocksAdd);
    expect(within(detailDialog).getByLabelText("Search tasks")).not.toBeNull();
    await waitFor(() => {
      expect(dependenciesGateway.candidates).toHaveBeenCalledWith(
        task.id,
        "blocks",
        "",
        1,
        5,
        expect.any(AbortSignal),
      );
    });
    expect(
      await within(detailDialog).findByText("No matching tasks."),
    ).not.toBeNull();
  });

  it("AC-4 retries a failed dependency load through contextual Edit Task", async () => {
    const task = node("task", "Backend API");
    const dependenciesGateway: DependenciesGateway = {
      list: vi
        .fn()
        .mockRejectedValueOnce(new Error("database timeout"))
        .mockResolvedValueOnce({
          blockedBy: [
            {
              id: "dependency-auth-api",
              source: "manual",
              manualRemovable: true,
              task: {
                id: "authentication",
                name: "Authentication",
                projectId: "security",
                projectName: "Security",
                hierarchyPath: "Security > Authentication",
                completed: false,
              },
            },
          ],
          blocks: [],
        }),
      candidates: vi.fn().mockResolvedValue({
        items: [],
        page: 1,
        pageSize: 5,
        totalItems: 0,
      }),
      create: vi.fn().mockResolvedValue(undefined),
      remove: vi.fn().mockResolvedValue(undefined),
      keepAsManual: vi.fn().mockResolvedValue(undefined),
      invalidateTask: vi.fn(),
      invalidateAll: vi.fn(),
    };

    render(
      <WBSPanel
        project={project}
        gateway={gateway([task])}
        dependenciesGateway={dependenciesGateway}
        {...options}
        onClose={() => undefined}
      />,
    );

    const tree = await screen.findByRole("tree");
    const taskTreeItem = within(tree)
      .getByText("Backend API")
      .closest<HTMLElement>('[role="treeitem"]');

    expect(taskTreeItem).not.toBeNull();
    if (!taskTreeItem) {
      throw new Error("Backend API tree item was not found");
    }

    fireEvent.click(
      within(taskTreeItem).getByRole("button", { name: "Edit Task" }),
    );

    const detailDialog = await screen.findByRole("dialog", {
      name: "Edit Task",
    });

    expect(
      await within(detailDialog).findByText(
        /dependencies could not be loaded/i,
      ),
    ).not.toBeNull();
    expect(within(detailDialog).queryByText("database timeout")).toBeNull();

    fireEvent.click(
      within(detailDialog).getByRole("button", { name: "Retry" }),
    );

    expect(
      await within(detailDialog).findByText("Authentication"),
    ).not.toBeNull();

    await waitFor(() => {
      expect(
        within(detailDialog).queryByText(/dependencies could not be loaded/i),
      ).toBeNull();
      expect(dependenciesGateway.list).toHaveBeenCalledTimes(2);
    });
  });

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
