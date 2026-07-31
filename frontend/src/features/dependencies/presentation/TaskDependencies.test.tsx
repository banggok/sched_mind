import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { DependenciesGateway } from "../application/dependenciesGateway";
import type { DependencyDetail } from "../domain/dependency";
import { TaskDependencies } from "./TaskDependencies";

function gateway(): DependenciesGateway {
  return {
    list: vi.fn().mockResolvedValue({
      blockedBy: [
        {
          id: "dependency-a",
          source: "manual",
          manualRemovable: true,
          task: {
            id: "a",
            name: "API",
            projectId: "alpha",
            projectName: "Alpha",
            hierarchyPath: "Alpha > API",
            completed: true,
          },
        },
      ],
      blocks: [
        {
          id: "dependency-b",
          source: "manual",
          manualRemovable: true,
          task: {
            id: "b",
            name: "Build",
            projectId: "beta",
            projectName: "Beta",
            hierarchyPath: "Beta > Build",
            completed: false,
          },
        },
      ],
    }),
    candidates: vi.fn().mockResolvedValue({
      items: [
        {
          id: "c",
          name: "Client",
          projectId: "gamma",
          projectName: "Gamma",
          hierarchyPath: "Gamma > Client",
          completed: false,
        },
      ],
      page: 1,
      pageSize: 5,
      totalItems: 1,
    }),
    create: vi.fn().mockResolvedValue(undefined),
    remove: vi.fn().mockResolvedValue(undefined),
    keepAsManual: vi.fn().mockResolvedValue(undefined),
    invalidateTask: vi.fn(),
    invalidateAll: vi.fn(),
  };
}

describe("TaskDependencies", () => {
  it("renders both directions and the unscheduled projection fallback", async () => {
    render(
      <TaskDependencies taskId="task" gateway={gateway()} readOnly={false} />,
    );
    expect(await screen.findByText("Blocked by")).not.toBeNull();
    expect(screen.getByText("Blocks")).not.toBeNull();
    expect(screen.getByText(/Alpha · Completed/)).not.toBeNull();
    expect(screen.getByText(/Beta · Not scheduled/)).not.toBeNull();
  });

  it("renders an unconfirmed automatic dependency preview without mutation controls", async () => {
    const value = gateway();
    render(
      <TaskDependencies
        taskId="task"
        gateway={value}
        readOnly={false}
        previewDetail={{
          blockedBy: [
            {
              id: "preview-automatic",
              source: "automatic",
              manualRemovable: false,
              task: {
                id: "task-1",
                name: "Task 1",
                projectId: "alpha",
                projectName: "Alpha",
                hierarchyPath: "Alpha > Task 1",
                completed: false,
                expectedStart: "2026-08-04",
              },
            },
          ],
          blocks: [],
        }}
      />,
    );

    expect(
      screen.getByText(
        "Unconfirmed dependency preview. Save confirms automatic dependencies.",
      ),
    ).not.toBeNull();
    expect(screen.getByText("Task 1")).not.toBeNull();
    expect(
      screen.getByLabelText("Dependency source: Automatic"),
    ).not.toBeNull();
    expect(screen.queryByRole("button", { name: "Add" })).toBeNull();
    expect(value.remove).not.toHaveBeenCalled();
    expect(value.keepAsManual).not.toHaveBeenCalled();
  });

  it("US-6.1 AC-23 AC-24 AC-25 exposes ownership and keeps automatic relation as manual", async () => {
    const value = gateway();
    vi.mocked(value.list)
      .mockResolvedValueOnce({
        blockedBy: [
          {
            id: "automatic-link",
            source: "automatic",
            manualRemovable: false,
            task: {
              id: "a",
              name: "API",
              projectId: "alpha",
              projectName: "Alpha",
              hierarchyPath: "Alpha > API",
              completed: false,
            },
          },
        ],
        blocks: [],
      })
      .mockResolvedValueOnce({
        blockedBy: [
          {
            id: "automatic-link",
            source: "both",
            manualRemovable: true,
            task: {
              id: "a",
              name: "API",
              projectId: "alpha",
              projectName: "Alpha",
              hierarchyPath: "Alpha > API",
              completed: false,
            },
          },
        ],
        blocks: [],
      });

    render(<TaskDependencies taskId="task" gateway={value} readOnly={false} />);

    expect(
      await screen.findByLabelText("Dependency source: Automatic"),
    ).not.toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Keep as Manual" }));

    await waitFor(() => {
      expect(value.keepAsManual).toHaveBeenCalledWith("automatic-link");
      expect(value.list).toHaveBeenCalledTimes(2);
    });
    expect(
      await screen.findByLabelText("Dependency source: Manual + Automatic"),
    ).not.toBeNull();
    expect(
      screen.getByRole("button", { name: "Remove manual ownership for API" }),
    ).not.toBeNull();
  });

  it("US-6.1 AC-24 removes only manual ownership from a shared relation", async () => {
    const value = gateway();
    vi.mocked(value.list).mockResolvedValueOnce({
      blockedBy: [
        {
          id: "shared-link",
          source: "both",
          manualRemovable: true,
          task: {
            id: "a",
            name: "API",
            projectId: "alpha",
            projectName: "Alpha",
            hierarchyPath: "Alpha > API",
            completed: false,
          },
        },
      ],
      blocks: [],
    });

    render(<TaskDependencies taskId="task" gateway={value} readOnly={false} />);

    expect(
      await screen.findByLabelText("Dependency source: Manual + Automatic"),
    ).not.toBeNull();
    fireEvent.click(
      screen.getByRole("button", { name: "Remove manual ownership for API" }),
    );

    await waitFor(() => {
      expect(value.remove).toHaveBeenCalledWith("shared-link");
    });
  });

  it("projects one created relation as Blocks for the blocker and Blocked by for the blocked task", async () => {
    const dependencyId = "dependency-ab";
    const taskA = {
      id: "a",
      name: "API",
      projectId: "alpha",
      projectName: "Alpha",
      hierarchyPath: "Alpha > API",
      completed: false,
    };
    const taskB = {
      id: "b",
      name: "Build",
      projectId: "beta",
      projectName: "Beta",
      hierarchyPath: "Beta > Build",
      completed: false,
    };
    let relationCreated = false;

    const value: DependenciesGateway = {
      list: vi.fn(async (taskId: string): Promise<DependencyDetail> => {
        if (!relationCreated) return { blockedBy: [], blocks: [] };
        if (taskId === taskA.id) {
          return {
            blockedBy: [],
            blocks: [
              {
                id: dependencyId,
                source: "manual",
                manualRemovable: true,
                task: taskB,
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
                task: taskA,
              },
            ],
            blocks: [],
          };
        }
        throw new Error(`unexpected task ${taskId}`);
      }),
      candidates: vi.fn(async () => ({
        items: [taskB],
        page: 1,
        pageSize: 5,
        totalItems: 1,
      })),
      create: vi.fn(async (blockingTaskId: string, blockedTaskId: string) => {
        expect(blockingTaskId).toBe(taskA.id);
        expect(blockedTaskId).toBe(taskB.id);
        relationCreated = true;
      }),
      remove: vi.fn().mockResolvedValue(undefined),
      keepAsManual: vi.fn().mockResolvedValue(undefined),
      invalidateTask: vi.fn(),
      invalidateAll: vi.fn(),
    };

    const view = render(
      <TaskDependencies taskId={taskA.id} gateway={value} readOnly={false} />,
    );

    await screen.findByText("Blocked by");
    fireEvent.click(screen.getAllByRole("button", { name: "Add" })[1]);
    fireEvent.click(await screen.findByRole("button", { name: "Select" }));

    const blocksHeading = await screen.findByRole("heading", {
      name: "Blocks",
    });
    const blocksSection = blocksHeading.parentElement?.parentElement;
    expect(blocksSection).not.toBeNull();
    if (!blocksSection) return;
    expect(await within(blocksSection).findByText("Build")).not.toBeNull();
    expect(within(blocksSection).queryByText("API")).toBeNull();

    view.rerender(
      <TaskDependencies taskId={taskB.id} gateway={value} readOnly={false} />,
    );

    await waitFor(() => {
      expect(value.list).toHaveBeenCalledWith(
        taskB.id,
        expect.any(AbortSignal),
      );
    });
    const blockedByHeading = screen.getByRole("heading", {
      name: "Blocked by",
    });
    const blockedBySection = blockedByHeading.parentElement?.parentElement;
    expect(blockedBySection).not.toBeNull();
    if (!blockedBySection) return;
    expect(await within(blockedBySection).findByText("API")).not.toBeNull();
    expect(within(blockedBySection).queryByText("Build")).toBeNull();
  });

  it("searches paginated candidates and creates through Blocks direction", async () => {
    const value = gateway();
    render(<TaskDependencies taskId="task" gateway={value} readOnly={false} />);
    await screen.findByText("Blocked by");
    fireEvent.click(screen.getAllByRole("button", { name: "Add" })[1]);
    const search = await screen.findByLabelText("Search tasks");
    fireEvent.change(search, { target: { value: "Cli" } });
    expect(await screen.findByText("Client")).not.toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Select" }));
    await waitFor(() => expect(value.create).toHaveBeenCalledWith("task", "c"));
  });

  it("sends only one create mutation when submission is triggered twice before render", async () => {
    let resolveCreate: (() => void) | undefined;
    const pendingCreate = new Promise<void>((resolve) => {
      resolveCreate = resolve;
    });
    const value = gateway();
    vi.mocked(value.create).mockReturnValueOnce(pendingCreate);

    render(<TaskDependencies taskId="task" gateway={value} readOnly={false} />);

    await screen.findByText("Blocked by");
    fireEvent.click(screen.getAllByRole("button", { name: "Add" })[1]);
    const select = await screen.findByRole("button", { name: "Select" });

    act(() => {
      select.dispatchEvent(new MouseEvent("click", { bubbles: true }));
      select.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    expect(value.create).toHaveBeenCalledTimes(1);
    expect(value.create).toHaveBeenCalledWith("task", "c");
    expect((select as HTMLButtonElement).disabled).toBe(true);

    resolveCreate?.();
    expect(await screen.findByText("Dependency added.")).not.toBeNull();
  });

  it("removes a dependency directly from the unlink action", async () => {
    const value = gateway();

    render(<TaskDependencies taskId="task" gateway={value} readOnly={false} />);

    const taskName = await screen.findByText("API");
    const relation = taskName.closest("li");

    expect(relation).not.toBeNull();

    fireEvent.click(
      within(relation!).getByRole("button", {
        name: "Remove manual ownership for API",
      }),
    );

    await waitFor(() => {
      expect(value.remove).toHaveBeenCalledTimes(1);
      expect(value.remove).toHaveBeenCalledWith("dependency-a");
    });

    expect(await screen.findByText("Dependency removed.")).not.toBeNull();
  });

  it("sends only one delete mutation when unlink is triggered twice before render", async () => {
    let resolveRemove: (() => void) | undefined;

    const pendingRemove = new Promise<void>((resolve) => {
      resolveRemove = resolve;
    });

    const value = gateway();

    vi.mocked(value.remove).mockReturnValueOnce(pendingRemove);

    render(<TaskDependencies taskId="task" gateway={value} readOnly={false} />);

    const taskName = await screen.findByText("API");
    const relation = taskName.closest("li");

    expect(relation).not.toBeNull();

    if (!relation) {
      return;
    }

    const removeButton = within(relation).getByRole("button", {
      name: "Remove manual ownership for API",
    });

    act(() => {
      removeButton.dispatchEvent(new MouseEvent("click", { bubbles: true }));

      removeButton.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    expect(value.remove).toHaveBeenCalledTimes(1);
    expect(value.remove).toHaveBeenCalledWith("dependency-a");
    expect((removeButton as HTMLButtonElement).disabled).toBe(true);

    resolveRemove?.();

    expect(await screen.findByText("Dependency removed.")).not.toBeNull();
  });

  it("shows both empty dependency states only after loading succeeds", async () => {
    let resolveList:
      | ((value: Awaited<ReturnType<DependenciesGateway["list"]>>) => void)
      | undefined;

    const pendingList = new Promise<
      Awaited<ReturnType<DependenciesGateway["list"]>>
    >((resolve) => {
      resolveList = resolve;
    });

    const value = gateway();
    vi.mocked(value.list).mockReturnValueOnce(pendingList);

    render(<TaskDependencies taskId="task" gateway={value} readOnly={false} />);

    expect(screen.queryByText("No dependencies.")).toBeNull();
    expect(
      screen.getByRole("status", { name: "Loading dependencies" }),
    ).not.toBeNull();

    resolveList?.({
      blockedBy: [],
      blocks: [],
    });

    await waitFor(() => {
      expect(screen.getAllByText("No dependencies.")).toHaveLength(2);
    });

    expect(screen.getAllByRole("button", { name: "Add" })).toHaveLength(2);
  });

  it("keeps confirmed dependencies visible while refreshing", async () => {
    let resolveRefresh:
      | ((value: Awaited<ReturnType<DependenciesGateway["list"]>>) => void)
      | undefined;

    const refreshPromise = new Promise<
      Awaited<ReturnType<DependenciesGateway["list"]>>
    >((resolve) => {
      resolveRefresh = resolve;
    });

    const value = gateway();

    vi.mocked(value.list)
      .mockResolvedValueOnce({
        blockedBy: [
          {
            id: "dependency-a",
            source: "manual",
            manualRemovable: true,
            task: {
              id: "a",
              name: "API",
              projectId: "alpha",
              projectName: "Alpha",
              hierarchyPath: "Alpha > API",
              completed: true,
            },
          },
        ],
        blocks: [
          {
            id: "dependency-b",
            source: "manual",
            manualRemovable: true,
            task: {
              id: "b",
              name: "Build",
              projectId: "beta",
              projectName: "Beta",
              hierarchyPath: "Beta > Build",
              completed: false,
            },
          },
        ],
      })
      .mockReturnValueOnce(refreshPromise);

    render(<TaskDependencies taskId="task" gateway={value} readOnly={false} />);

    expect(await screen.findByText("API")).not.toBeNull();
    expect(screen.getByText("Build")).not.toBeNull();

    fireEvent.click(screen.getAllByRole("button", { name: "Add" })[1]);

    const search = await screen.findByLabelText("Search tasks");
    fireEvent.change(search, { target: { value: "Cli" } });

    expect(await screen.findByText("Client")).not.toBeNull();

    fireEvent.click(screen.getByRole("button", { name: "Select" }));

    await waitFor(() => {
      expect(value.create).toHaveBeenCalledWith("task", "c");
      expect(value.list).toHaveBeenCalledTimes(2);
    });

    // Confirmed data must remain rendered while the refresh request is pending.
    expect(screen.queryByText("API")).not.toBeNull();
    expect(screen.queryByText("Build")).not.toBeNull();
    expect(screen.getByText("Refreshing…")).not.toBeNull();
    expect(screen.queryByText("No dependencies.")).toBeNull();

    resolveRefresh?.({
      blockedBy: [
        {
          id: "dependency-a",
          source: "manual",
          manualRemovable: true,
          task: {
            id: "a",
            name: "API",
            projectId: "alpha",
            projectName: "Alpha",
            hierarchyPath: "Alpha > API",
            completed: true,
          },
        },
      ],
      blocks: [
        {
          id: "dependency-b",
          source: "manual",
          manualRemovable: true,
          task: {
            id: "b",
            name: "Build",
            projectId: "beta",
            projectName: "Beta",
            hierarchyPath: "Beta > Build",
            completed: false,
          },
        },
        {
          id: "dependency-c",
          source: "manual",
          manualRemovable: true,
          task: {
            id: "c",
            name: "Client",
            projectId: "gamma",
            projectName: "Gamma",
            hierarchyPath: "Gamma > Client",
            completed: false,
          },
        },
      ],
    });

    await waitFor(() => {
      expect(screen.queryByText("Refreshing…")).toBeNull();
      expect(screen.getByText("Client")).not.toBeNull();
    });
  });

  it("recovers from dependency loading failure after retry", async () => {
    const value = gateway();

    vi.mocked(value.list)
      .mockRejectedValueOnce(new Error("database timeout"))
      .mockResolvedValueOnce({
        blockedBy: [
          {
            id: "dependency-a",
            source: "manual",
            manualRemovable: true,
            task: {
              id: "a",
              name: "API",
              projectId: "alpha",
              projectName: "Alpha",
              hierarchyPath: "Alpha > API",
              completed: true,
            },
          },
        ],
        blocks: [
          {
            id: "dependency-b",
            source: "manual",
            manualRemovable: true,
            task: {
              id: "b",
              name: "Build",
              projectId: "beta",
              projectName: "Beta",
              hierarchyPath: "Beta > Build",
              completed: false,
            },
          },
        ],
      });

    render(<TaskDependencies taskId="task" gateway={value} readOnly={false} />);

    expect(
      await screen.findByText(/dependencies could not be loaded/i),
    ).not.toBeNull();

    expect(screen.queryByText(/database timeout/i)).toBeNull();

    fireEvent.click(screen.getByRole("button", { name: "Retry" }));

    expect(await screen.findByText("API")).not.toBeNull();
    expect(screen.getByText("Build")).not.toBeNull();

    await waitFor(() => {
      expect(
        screen.queryByText(/dependencies could not be loaded/i),
      ).toBeNull();
    });

    expect(value.list).toHaveBeenCalledTimes(2);
  });

  it("shows task name, hierarchy, and completed status for candidates", async () => {
    const value = gateway();

    vi.mocked(value.candidates).mockResolvedValueOnce({
      items: [
        {
          id: "candidate",
          name: "Task 2",
          projectId: "ntb",
          projectName: "NTB",
          hierarchyPath: "NTB > Task 1 > Task 2",
          completed: true,
        },
      ],
      page: 1,
      pageSize: 5,
      totalItems: 1,
    });

    render(<TaskDependencies taskId="task" gateway={value} readOnly={false} />);

    await screen.findByText("Blocked by");

    fireEvent.click(screen.getAllByRole("button", { name: "Add" })[0]);

    expect(await screen.findByText("Task 2")).not.toBeNull();

    expect(
      screen.getByText("NTB > Task 1 > Task 2 · Completed"),
    ).not.toBeNull();
  });

  it("creates through Blocked by and refreshes the current task", async () => {
    const value = gateway();

    vi.mocked(value.list)
      .mockResolvedValueOnce({
        blockedBy: [],
        blocks: [],
      })
      .mockResolvedValueOnce({
        blockedBy: [
          {
            id: "dependency-c",
            source: "manual",
            manualRemovable: true,
            task: {
              id: "c",
              name: "Client",
              projectId: "gamma",
              projectName: "Gamma",
              hierarchyPath: "Gamma > Client",
              completed: false,
            },
          },
        ],
        blocks: [],
      });

    render(<TaskDependencies taskId="task" gateway={value} readOnly={false} />);

    await screen.findByText("Blocked by");

    // The first Add button belongs to the Blocked by section.
    fireEvent.click(screen.getAllByRole("button", { name: "Add" })[0]);

    const search = await screen.findByLabelText("Search tasks");
    fireEvent.change(search, { target: { value: "Cli" } });

    expect(await screen.findByText("Client")).not.toBeNull();

    fireEvent.click(screen.getByRole("button", { name: "Select" }));

    await waitFor(() => {
      expect(value.create).toHaveBeenCalledWith("c", "task");
      expect(value.list).toHaveBeenCalledTimes(2);
    });

    await waitFor(() => {
      expect(screen.getByText("Client")).not.toBeNull();
      expect(screen.queryByText("Refreshing…")).toBeNull();
    });
  });

  it("renders scheduler expected start projection from backend", async () => {
    const value = gateway();

    vi.mocked(value.list).mockResolvedValueOnce({
      blockedBy: [],
      blocks: [
        {
          id: "dependency-b",
          source: "manual",
          manualRemovable: true,
          task: {
            id: "b",
            name: "Build",
            projectId: "beta",
            projectName: "Beta",
            hierarchyPath: "Beta > Build",
            completed: false,
            expectedStart: "2026-07-18T00:00:00Z",
          },
        },
      ],
    });

    render(<TaskDependencies taskId="task" gateway={value} readOnly={false} />);

    expect(await screen.findByText("Build")).not.toBeNull();

    expect(screen.getByText("Beta · 18 Jul 2026")).not.toBeNull();

    expect(screen.queryByText(/Not scheduled/i)).toBeNull();
  });
});
