import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { DependenciesGateway } from "../../dependencies/application/dependenciesGateway";
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
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  } satisfies RolesGateway,
  membersGateway: {
    list: vi
      .fn()
      .mockResolvedValue({ items: [], page: 1, pageSize: 100, total: 0 }),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  } satisfies TeamMembersGateway,
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
    allocations: vi
      .fn()
      .mockResolvedValue({ execution: [], commitment: [], actual: [] }),
    create: vi.fn().mockResolvedValue(node("created", "Created")),
    rename: vi.fn().mockResolvedValue(undefined),
    reorder: vi.fn().mockResolvedValue(undefined),
    move: vi.fn().mockResolvedValue(undefined),
    remove: vi.fn().mockResolvedValue(undefined),
    updateExecutable: vi.fn().mockResolvedValue(undefined),
    previewExecutableSchedule: vi.fn(),
    complete: vi.fn().mockResolvedValue(undefined),
    reopen: vi.fn(),
  };
}

function dependenciesGateway(): DependenciesGateway {
  return {
    list: vi.fn().mockResolvedValue({ blockedBy: [], blocks: [] }),
    candidates: vi.fn().mockResolvedValue({
      items: [],
      page: 1,
      pageSize: 5,
      totalItems: 0,
    }),
    create: vi.fn().mockResolvedValue(undefined),
    remove: vi.fn().mockResolvedValue(undefined),
    invalidateTask: vi.fn(),
    invalidateAll: vi.fn(),
  };
}

describe("WBS direct Home action controller", () => {
  it("opens a Task directly and exposes dependency state without Project Structure", async () => {
    const task = node("task", "Backend API");
    const deps = dependenciesGateway();
    vi.mocked(deps.list).mockResolvedValue({
      blockedBy: [],
      blocks: [
        {
          id: "dependency",
          task: {
            id: "deploy",
            name: "Deploy",
            projectId: "project",
            projectName: "Alpha",
            hierarchyPath: "Alpha > Deploy",
            completed: false,
          },
        },
      ],
    });

    render(
      <WBSPanel
        project={project}
        gateway={gateway([task])}
        dependenciesGateway={deps}
        {...options}
        initialNodeId="task"
        onClose={() => undefined}
      />,
    );

    const detail = await screen.findByRole("dialog", { name: "Edit Task" });
    expect(screen.queryByText("Project Structure")).toBeNull();
    expect(
      (within(detail).getByLabelText("Name") as HTMLInputElement).value,
    ).toBe("Backend API");
    const blocksHeading = await within(detail).findByRole("heading", {
      name: "Blocks",
    });
    const blocksSection = blocksHeading.parentElement?.parentElement;
    expect(blocksSection).not.toBeNull();
    if (!blocksSection) return;
    expect(within(blocksSection).getByText("Deploy")).not.toBeNull();
    expect(deps.list).toHaveBeenCalledWith("task", expect.any(AbortSignal));
  });

  it("keeps confirmed Task data visible while dependencies load", async () => {
    let resolveDependencies:
      | ((value: Awaited<ReturnType<DependenciesGateway["list"]>>) => void)
      | undefined;
    const pendingDependencies = new Promise<
      Awaited<ReturnType<DependenciesGateway["list"]>>
    >((resolve) => {
      resolveDependencies = resolve;
    });
    const deps = dependenciesGateway();
    vi.mocked(deps.list).mockReturnValue(pendingDependencies);

    render(
      <WBSPanel
        project={project}
        gateway={gateway([node("task", "Backend API")])}
        dependenciesGateway={deps}
        {...options}
        initialNodeId="task"
        onClose={() => undefined}
      />,
    );

    const detail = await screen.findByRole("dialog", { name: "Edit Task" });
    expect(
      (within(detail).getByLabelText("Name") as HTMLInputElement).value,
    ).toBe("Backend API");
    expect(
      within(detail).getByRole("status", { name: "Loading dependencies" }),
    ).not.toBeNull();
    resolveDependencies?.({ blockedBy: [], blocks: [] });
    await waitFor(() =>
      expect(within(detail).getAllByText("No dependencies.")).toHaveLength(2),
    );
  });

  it("does not render the removed standalone panel when no Home action is supplied", async () => {
    render(
      <WBSPanel
        project={project}
        gateway={gateway([])}
        {...options}
        onClose={() => undefined}
      />,
    );

    expect(
      await screen.findByRole("dialog", { name: "WBS item unavailable" }),
    ).not.toBeNull();
    expect(screen.getByText(/Choose one WBS action from Home/)).not.toBeNull();
    expect(screen.queryByText("Project Structure")).toBeNull();
    expect(screen.queryByRole("tree")).toBeNull();
  });

  it("opens the shared Group summary and rename dialog directly", async () => {
    vi.mocked(options.rolesGateway.list).mockClear();
    vi.mocked(options.membersGateway.list).mockClear();
    const group = node("group", "Development", [node("task", "Backend API")]);
    const api = gateway([group]);
    const onMutated = vi.fn();
    const onClose = vi.fn();
    render(
      <WBSPanel
        project={project}
        gateway={api}
        {...options}
        initialNodeId="group"
        onMutated={onMutated}
        onClose={onClose}
      />,
    );

    const dialog = await screen.findByRole("dialog", { name: "Development" });
    const name = within(dialog).getByLabelText("Name");
    fireEvent.change(name, { target: { value: "Delivery" } });
    fireEvent.submit(name.closest("form")!);
    await waitFor(() =>
      expect(api.rename).toHaveBeenCalledWith("project", "group", "Delivery"),
    );
    expect(options.rolesGateway.list).not.toHaveBeenCalled();
    expect(options.membersGateway.list).not.toHaveBeenCalled();
    expect(onMutated).toHaveBeenCalledWith("group");
    expect(onMutated.mock.invocationCallOrder[0]).toBeLessThan(
      onClose.mock.invocationCallOrder[0],
    );
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("does not report a Group edit mutation when rename fails", async () => {
    const group = node("group", "Development", [node("task", "Backend API")]);
    const api = gateway([group]);
    vi.mocked(api.rename).mockRejectedValue(
      new Error("Group could not be updated. Try again."),
    );
    const onMutated = vi.fn();
    const onClose = vi.fn();
    render(
      <WBSPanel
        project={project}
        gateway={api}
        {...options}
        initialNodeId="group"
        onMutated={onMutated}
        onClose={onClose}
      />,
    );

    const dialog = await screen.findByRole("dialog", { name: "Development" });
    const name = within(dialog).getByLabelText("Name");
    fireEvent.change(name, { target: { value: "Delivery" } });
    fireEvent.submit(name.closest("form")!);

    expect(
      await within(dialog).findByText("Group could not be updated. Try again."),
    ).toBeTruthy();
    expect(onMutated).not.toHaveBeenCalled();
    expect(onClose).not.toHaveBeenCalled();
  });

  it("opens Move to with destinations from the full authoritative tree", async () => {
    const source = node("source", "Source");
    const destination = node("destination", "Destination", [
      node("nested", "Nested"),
    ]);
    const api = gateway([source, destination]);
    const onMutated = vi.fn();
    const onClose = vi.fn();
    render(
      <WBSPanel
        project={project}
        gateway={api}
        {...options}
        initialMoveNodeId="source"
        onMutated={onMutated}
        onClose={onClose}
      />,
    );

    const dialog = await screen.findByRole("dialog", { name: "Move Item" });
    const destinationSelect =
      within(dialog).getByLabelText("Destination group");
    expect(within(destinationSelect).getByText("Destination")).not.toBeNull();
    expect(within(destinationSelect).getByText("Nested")).not.toBeNull();
    fireEvent.change(destinationSelect, { target: { value: "destination" } });
    fireEvent.submit(destinationSelect.closest("form")!);
    await waitFor(() =>
      expect(api.move).toHaveBeenCalledWith(
        "project",
        "source",
        "destination",
        false,
      ),
    );
    expect(onMutated).toHaveBeenCalledWith("source");
    expect(onMutated.mock.invocationCallOrder[0]).toBeLessThan(
      onClose.mock.invocationCallOrder[0],
    );
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("rejects a stale direct Move draft when the source disappears", async () => {
    let confirmedTree = [node("source", "Source"), node("target", "Target")];
    let notifyConfirmedChange: (() => void) | undefined;
    const api = gateway(confirmedTree);
    vi.mocked(api.tree).mockImplementation(() =>
      Promise.resolve(confirmedTree),
    );
    api.subscribeToConfirmedChanges = (listener) => {
      notifyConfirmedChange = listener;
      return () => undefined;
    };
    const onMutated = vi.fn();

    render(
      <WBSPanel
        project={project}
        gateway={api}
        {...options}
        initialMoveNodeId="source"
        onMutated={onMutated}
        onClose={() => undefined}
      />,
    );

    expect(
      await screen.findByRole("dialog", { name: "Move Item" }),
    ).not.toBeNull();
    confirmedTree = [node("target", "Target")];
    await act(async () => {
      notifyConfirmedChange?.();
    });

    expect(
      await screen.findByRole("dialog", { name: "WBS item unavailable" }),
    ).not.toBeNull();
    expect(
      screen.getByText(/selected WBS item no longer exists/),
    ).not.toBeNull();
    expect(api.move).not.toHaveBeenCalled();
    expect(onMutated).not.toHaveBeenCalled();
  });

  it("keeps the shared Group dialog read-only for a Locked Project", async () => {
    const locked = { ...project, status: "locked" as const };
    const group = node("group", "Development", [node("task", "Backend API")]);
    render(
      <WBSPanel
        project={locked}
        gateway={gateway([group])}
        {...options}
        initialNodeId="group"
        onClose={() => undefined}
      />,
    );

    const dialog = await screen.findByRole("dialog", { name: "Development" });
    expect(within(dialog).getByLabelText("Name").hasAttribute("disabled")).toBe(
      true,
    );
    expect(within(dialog).queryByRole("button", { name: "Save" })).toBeNull();
    expect(
      within(dialog).getByRole("button", { name: "Close" }),
    ).not.toBeNull();
  });

  it("opens a Locked Task with planning fields read-only and Actual Date writable", async () => {
    const locked = { ...project, status: "locked" as const };
    const task = node("task", "Backend API");
    const api = gateway([task]);
    const onClose = vi.fn();
    render(
      <WBSPanel
        project={locked}
        gateway={api}
        {...options}
        initialNodeId="task"
        onClose={onClose}
      />,
    );

    const dialog = await screen.findByRole("dialog", { name: "Edit Task" });
    expect(within(dialog).getByLabelText("Name").hasAttribute("disabled")).toBe(
      true,
    );
    const actualDate = within(dialog).getByRole("button", {
      name: "Actual Date: Select start and end date",
    });
    expect(actualDate.hasAttribute("disabled")).toBe(false);
    fireEvent.click(actualDate);
    const current = new Date();
    const currentMonth = `${current.getUTCFullYear()}-${String(
      current.getUTCMonth() + 1,
    ).padStart(2, "0")}`;
    const start = `${currentMonth}-14`;
    const end = `${currentMonth}-15`;
    fireEvent.click(screen.getByRole("button", { name: start }));
    fireEvent.click(screen.getByRole("button", { name: end }));
    fireEvent.click(
      within(dialog).getByRole("button", { name: "Mark completed" }),
    );
    await waitFor(() =>
      expect(api.complete).toHaveBeenCalledWith("project", "task", start, end),
    );
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("reports the created row identity before returning Home after Add Task", async () => {
    const api = gateway([]);
    const created = node("new-task", "API");
    vi.mocked(api.create).mockResolvedValue(created);
    const onCreated = vi.fn();
    const onClose = vi.fn();
    render(
      <WBSPanel
        project={project}
        gateway={api}
        {...options}
        initialCreateParentId={null}
        onCreated={onCreated}
        onClose={onClose}
      />,
    );

    const dialog = await screen.findByRole("dialog", { name: "Add Task" });
    fireEvent.change(within(dialog).getByLabelText("Name"), {
      target: { value: "API" },
    });
    fireEvent.submit(within(dialog).getByLabelText("Name").closest("form")!);
    await waitFor(() =>
      expect(api.create).toHaveBeenCalledWith(
        "project",
        undefined,
        "API",
        false,
      ),
    );
    expect(onCreated).toHaveBeenCalledWith(created);
    expect(onCreated.mock.invocationCallOrder[0]).toBeLessThan(
      onClose.mock.invocationCallOrder[0],
    );
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("does not report a created row when Add Task fails or is cancelled", async () => {
    const api = gateway([]);
    vi.mocked(api.create).mockRejectedValue(
      new Error("The Task could not be created. Try again."),
    );
    const onCreated = vi.fn();
    const onClose = vi.fn();
    render(
      <WBSPanel
        project={project}
        gateway={api}
        {...options}
        initialCreateParentId={null}
        onCreated={onCreated}
        onClose={onClose}
      />,
    );

    const dialog = await screen.findByRole("dialog", { name: "Add Task" });
    fireEvent.change(within(dialog).getByLabelText("Name"), {
      target: { value: "API" },
    });
    fireEvent.submit(within(dialog).getByLabelText("Name").closest("form")!);

    expect(
      await within(dialog).findByText(
        "The Task could not be created. Try again.",
      ),
    ).toBeTruthy();
    expect(onCreated).not.toHaveBeenCalled();
    expect(onClose).not.toHaveBeenCalled();

    fireEvent.click(within(dialog).getByRole("button", { name: "Cancel" }));
    expect(onCreated).not.toHaveBeenCalled();
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("does not report a created row when Add Task is dismissed with Escape", async () => {
    const onCreated = vi.fn();
    const onClose = vi.fn();
    render(
      <WBSPanel
        project={project}
        gateway={gateway([])}
        {...options}
        initialCreateParentId={null}
        onCreated={onCreated}
        onClose={onClose}
      />,
    );

    const dialog = await screen.findByRole("dialog", { name: "Add Task" });
    const name = within(dialog).getByLabelText("Name");
    await waitFor(() => expect(document.activeElement).toBe(name));
    fireEvent.keyDown(document, { key: "Escape" });

    expect(onCreated).not.toHaveBeenCalled();
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("shows explicit conversion confirmation for direct Add Child", async () => {
    const task = node("task", "Backend API");
    const created = node("converted-child", "API tests");
    const api = gateway([task]);
    vi.mocked(api.create)
      .mockRejectedValueOnce(
        new WBSOperationError("WBS_CONVERSION_REQUIRED", "technical message"),
      )
      .mockResolvedValueOnce(created);
    const onCreated = vi.fn();
    const onClose = vi.fn();
    render(
      <WBSPanel
        project={project}
        gateway={api}
        {...options}
        initialCreateParentId="task"
        onCreated={onCreated}
        onClose={onClose}
      />,
    );

    const dialog = await screen.findByRole("dialog", { name: "Add Child" });
    fireEvent.change(within(dialog).getByLabelText("Name"), {
      target: { value: "API tests" },
    });
    fireEvent.submit(within(dialog).getByLabelText("Name").closest("form")!);
    expect(
      await screen.findByRole("heading", {
        name: "This task will become a group",
      }),
    ).not.toBeNull();
    expect(
      screen.getByText(/current task details will be moved/),
    ).not.toBeNull();
    await userEvent.click(
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
    expect(onCreated).toHaveBeenCalledWith(created);
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
