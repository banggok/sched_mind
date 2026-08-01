import { act, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import axe from "axe-core";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { DependenciesGateway } from "../../dependencies/application/dependenciesGateway";
import { createHTTPDependenciesGateway } from "../../dependencies/infrastructure/httpDependenciesGateway";
import type { Project } from "../../projects/domain/project";
import type { RolesGateway } from "../../roles/application/rolesGateway";
import type { TeamMembersGateway } from "../../team-members/application/teamMembersGateway";
import type { WBSGateway } from "../application/wbsGateway";
import type { WBSNode } from "../domain/wbs";
import { WBSPanel } from "./WBSPanel";

const rolesGateway = {
  list: vi.fn().mockResolvedValue({
    items: [
      {
        id: "role",
        name: "Backend Engineer",
        createdAt: new Date("2026-07-01T00:00:00Z"),
        updatedAt: new Date("2026-07-01T00:00:00Z"),
      },
    ],
    page: 1,
    pageSize: 100,
    total: 1,
  }),
} as unknown as RolesGateway;
const membersGateway = {
  list: vi.fn().mockResolvedValue({
    items: [
      {
        id: "member",
        name: "Rina",
        role: { id: "role", name: "Backend Engineer" },
        dailyCapacity: 8,
        bufferPercentage: 20,
        baseExecutionCapacity: 6.5,
        createdAt: new Date("2026-07-01T00:00:00Z"),
        updatedAt: new Date("2026-07-01T00:00:00Z"),
      },
    ],
    page: 1,
    pageSize: 100,
    total: 1,
  }),
} as unknown as TeamMembersGateway;

afterEach(() => {
  vi.unstubAllGlobals();
  Object.defineProperty(window, "innerWidth", {
    configurable: true,
    value: 1024,
  });
});

function project(status: Project["status"]): Project {
  return {
    id: "project",
    name: "Alpha",
    status,
    autoCalculateDate: true,
    automaticScheduling: true,
    projectBuffer: 20,
    scheduleVersion: 0,
    priority: 1,
    createdAt: new Date("2026-07-01T00:00:00Z"),
    updatedAt: new Date("2026-07-01T00:00:00Z"),
  };
}

function task(
  id: string,
  name: string,
  ...actualEndOverride: [string?]
): WBSNode {
  const actualEnd =
    actualEndOverride.length === 0 ? "2026-07-28" : actualEndOverride[0];
  const actualStart = actualEnd ? "2026-07-26" : undefined;
  return {
    id,
    projectId: "project",
    parentId: "group",
    name,
    position: 2,
    hasChildren: false,
    executable: {
      roleId: "role",
      assigneeId: "member",
      effortMinutes: 390,
      lagDays: 0,
      executionTimeline: { start: "2026-08-01", end: "2026-08-03" },
      commitmentTimeline: { start: "2026-08-01", end: "2026-08-05" },
      actualStart,
      actualEnd,
    },
    children: [],
  };
}

function group(): WBSNode {
  return {
    id: "group",
    projectId: "project",
    name: "Development",
    position: 1,
    hasChildren: true,
    executable: { lagDays: 0, executionTimeline: {}, commitmentTimeline: {} },
    children: [task("task", "Build API")],
  };
}

function wbsGateway(values: WBSNode[]): WBSGateway {
  return {
    tree: vi.fn().mockResolvedValue(values),
    allocations: vi
      .fn()
      .mockResolvedValue({ execution: [], commitment: [], actual: [] }),
    create: vi.fn().mockResolvedValue(undefined),
    rename: vi.fn().mockResolvedValue(undefined),
    reorder: vi.fn().mockResolvedValue(undefined),
    move: vi.fn().mockResolvedValue(undefined),
    remove: vi.fn().mockResolvedValue(undefined),
    updateExecutable: vi.fn(),
    previewExecutableSchedule: vi
      .fn()
      .mockImplementation(async (_projectId, id) => {
        const original = flatten(values).find((value) => value.id === id);
        if (!original) throw new Error("missing fixture");
        return {
          task: original,
          dependencies: { blockedBy: [], blocks: [] },
        };
      }),
    complete: vi.fn().mockResolvedValue(undefined),
    reopen: vi.fn().mockImplementation(async (_projectId, id) => {
      const original = flatten(values).find((value) => value.id === id);
      if (!original) throw new Error("missing fixture");
      return {
        ...original,
        executable: {
          ...original.executable,
          actualStart: undefined,
          actualEnd: undefined,
        },
      };
    }),
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
    keepAsManual: vi.fn().mockResolvedValue(undefined),
    invalidateTask: vi.fn(),
    invalidateAll: vi.fn(),
  };
}

function renderPanel(
  status: Project["status"],
  values: WBSNode[],
  gateway = wbsGateway(values),
  dependencyGateway?: DependenciesGateway,
  projectOverrides: Partial<Project> = {},
) {
  return {
    gateway,
    dependencyGateway,
    view: render(
      <WBSPanel
        project={{ ...project(status), ...projectOverrides }}
        gateway={gateway}
        dependenciesGateway={dependencyGateway}
        rolesGateway={rolesGateway}
        membersGateway={membersGateway}
        onClose={() => undefined}
      />,
    ),
  };
}

async function openTask(name: string) {
  const tree = await screen.findByRole("tree");
  const item = within(tree)
    .getByText(name)
    .closest<HTMLElement>('[role="treeitem"]');
  if (!item) throw new Error(`tree item ${name} not found`);
  await userEvent.click(
    within(item).getByRole("button", { name: "Edit Task" }),
  );
  return screen.findByRole("dialog", { name: "Edit Task" });
}

async function openConfirmation(detail: HTMLElement) {
  await userEvent.click(
    within(detail).getByRole("button", { name: "Reopen Task" }),
  );
  return screen.findByRole("alertdialog", { name: "Reopen Task?" });
}

describe("US-4.2 acceptance workflow through WBSPanel", () => {
  it("AC-1 AC-4 AC-5 AC-6 AC-7 AC-14 reopens a completed Open Task from Project Structure without hard reload", async () => {
    const values = [group()];
    const api = wbsGateway(values);
    renderPanel("open", values, api);
    let detail = await openTask("Build API");
    expect(
      (within(detail).getByLabelText("Name") as HTMLInputElement).disabled,
    ).toBe(true);
    expect(
      (within(detail).getByLabelText("Effort (hours)") as HTMLInputElement)
        .value,
    ).toBe("6.5");
    expect(
      (within(detail).getByLabelText("Role") as HTMLSelectElement).value,
    ).toBe("role");
    expect(
      (within(detail).getByLabelText("Assignee") as HTMLSelectElement).value,
    ).toBe("member");
    expect(
      (
        within(detail).getByRole("button", {
          name: /Execution timeline: .*2026.*2026/,
        }) as HTMLButtonElement
      ).disabled,
    ).toBe(true);
    expect(
      (
        within(detail).getByRole("button", {
          name: /Commitment timeline: .*2026.*2026/,
        }) as HTMLButtonElement
      ).disabled,
    ).toBe(true);
    const confirmation = await openConfirmation(detail);
    expect(within(confirmation).getByText("Build API")).toBeTruthy();
    expect(within(confirmation).getByText(/2026/)).toBeTruthy();
    expect(
      within(confirmation).getByText(
        /removes both Actual Start and Actual End/,
      ),
    ).toBeTruthy();
    await userEvent.click(
      within(confirmation).getByRole("button", { name: "Reopen Task" }),
    );

    detail = await screen.findByRole("dialog", { name: "Edit Task" });
    await waitFor(() =>
      expect(
        (within(detail).getByLabelText("Name") as HTMLInputElement).disabled,
      ).toBe(false),
    );
    expect(
      within(detail).queryByText(/Completed work is read-only/),
    ).toBeNull();
    expect(
      (within(detail).getByLabelText("Name") as HTMLInputElement).value,
    ).toBe("Build API");
    expect(
      (within(detail).getByLabelText("Effort (hours)") as HTMLInputElement)
        .value,
    ).toBe("6.5");
    expect(
      (within(detail).getByLabelText("Role") as HTMLSelectElement).value,
    ).toBe("role");
    expect(
      (within(detail).getByLabelText("Assignee") as HTMLSelectElement).value,
    ).toBe("member");
    expect(
      (
        within(detail).getByRole("button", {
          name: /Execution timeline: .*2026.*2026/,
        }) as HTMLButtonElement
      ).disabled,
    ).toBe(true);
    expect(
      (
        within(detail).getByRole("button", {
          name: /Commitment timeline: .*2026.*2026/,
        }) as HTMLButtonElement
      ).disabled,
    ).toBe(true);
    expect(api.reopen).toHaveBeenCalledWith("project", "task");
    expect(api.tree).toHaveBeenCalledTimes(1);
    expect(screen.getByText("Task reopened.")).toBeTruthy();
    expect(screen.getAllByText("Build API").length).toBeGreaterThan(0);
  });

  it("US-6.2 AC-16 hides Reopen for a completed Locked Task and preserves read-only planning fields", async () => {
    const values = [task("task", "Build API")];
    const api = wbsGateway(values);
    renderPanel("locked", values, api, undefined, {
      automaticScheduling: false,
    });
    const detail = await openTask("Build API");
    expect(
      within(detail).queryByRole("button", { name: "Reopen Task" }),
    ).toBeNull();
    expect(
      (within(detail).getByLabelText("Name") as HTMLInputElement).disabled,
    ).toBe(true);
    expect(
      (
        within(detail).getByRole("button", {
          name: "Save",
        }) as HTMLButtonElement
      ).disabled,
    ).toBe(true);
    expect(api.reopen).not.toHaveBeenCalled();
  });

  it("AC-3 cancel and Escape send no request, preserve state, and return focus", async () => {
    const user = userEvent.setup();
    const values = [task("task", "Build API")];
    const api = wbsGateway(values);
    renderPanel("open", values, api);
    const detail = await openTask("Build API");
    const trigger = within(detail).getByRole("button", { name: "Reopen Task" });
    const confirmation = await openConfirmation(detail);
    await user.click(
      within(confirmation).getByRole("button", { name: "Cancel" }),
    );
    expect(api.reopen).not.toHaveBeenCalled();
    expect(
      screen.queryByRole("alertdialog", { name: "Reopen Task?" }),
    ).toBeNull();
    expect(
      within(detail).getByText(/Completed work is read-only/),
    ).toBeTruthy();
    expect(document.activeElement).toBe(trigger);

    await user.click(trigger);
    await screen.findByRole("alertdialog", { name: "Reopen Task?" });
    await user.keyboard("{Escape}");
    expect(api.reopen).not.toHaveBeenCalled();
    expect(
      screen.queryByRole("alertdialog", { name: "Reopen Task?" }),
    ).toBeNull();
    expect(document.activeElement).toBe(trigger);
  });

  it("AC-2 AC-9 AC-13 does not offer Reopen for Closed, Grouping, or unfinished WBS", async () => {
    const closedView = renderPanel("closed", [task("task", "Closed Task")]);
    let detail = await openTask("Closed Task");
    expect(
      within(detail).queryByRole("button", { name: "Reopen Task" }),
    ).toBeNull();
    closedView.view.unmount();

    const groupView = renderPanel("open", [group()]);
    const tree = await screen.findByRole("tree");
    const groupItem = within(tree)
      .getByText("Development")
      .closest<HTMLElement>('[role="treeitem"]');
    if (!groupItem) throw new Error("group not found");
    await userEvent.click(
      within(groupItem).getByRole("button", { name: "View Group" }),
    );
    const groupDialog = await screen.findByRole("dialog", {
      name: "Development",
    });
    expect(
      within(groupDialog).queryByRole("button", { name: "Reopen Task" }),
    ).toBeNull();
    groupView.view.unmount();

    renderPanel("open", [task("task", "Unfinished", undefined)]);
    detail = await openTask("Unfinished");
    expect(
      within(detail).queryByRole("button", { name: "Reopen Task" }),
    ).toBeNull();
  });

  it("AC-11 AC-14 preserves completed state after Reopen failure and permits retry", async () => {
    const values = [task("task", "Build API")];
    const api = wbsGateway(values);
    vi.mocked(api.reopen)
      .mockRejectedValueOnce(
        new Error("The Task could not be reopened. Try again."),
      )
      .mockResolvedValueOnce(task("task", "Build API", undefined));
    renderPanel("open", values, api);
    let detail = await openTask("Build API");
    const confirmation = await openConfirmation(detail);
    await userEvent.click(
      within(confirmation).getByRole("button", { name: "Reopen Task" }),
    );
    expect(
      await within(confirmation).findByText(
        "The Task could not be reopened. Try again.",
      ),
    ).toBeTruthy();
    expect(
      within(detail).getByText(/Completed work is read-only/),
    ).toBeTruthy();

    await userEvent.click(
      within(confirmation).getByRole("button", { name: "Reopen Task" }),
    );
    detail = await screen.findByRole("dialog", { name: "Edit Task" });
    await waitFor(() =>
      expect(
        within(detail).queryByText(/Completed work is read-only/),
      ).toBeNull(),
    );
    expect(api.reopen).toHaveBeenCalledTimes(2);
  });

  it("AC-10 preserves dependency relation and refreshes its current completed marker", async () => {
    const target = task("target", "Build API");
    const other = task("other", "Deploy", undefined);
    let reopened = false;
    const api = wbsGateway([target, other]);
    vi.mocked(api.reopen).mockImplementation(async () => {
      reopened = true;
      return task("target", "Build API", undefined);
    });
    const deps = dependenciesGateway();
    vi.mocked(deps.list).mockImplementation(async (taskId) => {
      if (taskId !== "other") return { blockedBy: [], blocks: [] };
      return {
        blockedBy: [
          {
            id: "incoming-link",
            source: "manual",
            manualRemovable: true,
            task: {
              id: "target",
              name: "Build API",
              projectId: "project",
              projectName: "Alpha",
              hierarchyPath: "Alpha > Build API",
              completed: !reopened,
            },
          },
        ],
        blocks: [],
      };
    });
    renderPanel("open", [target, other], api, deps);

    let detail = await openTask("Deploy");
    expect(await within(detail).findByText("Alpha · Completed")).toBeTruthy();
    await userEvent.click(
      within(detail).getByRole("button", { name: "Close" }),
    );
    detail = await openTask("Build API");
    const confirmation = await openConfirmation(detail);
    await userEvent.click(
      within(confirmation).getByRole("button", { name: "Reopen Task" }),
    );
    detail = await screen.findByRole("dialog", { name: "Edit Task" });
    await userEvent.click(
      within(detail).getByRole("button", { name: "Close" }),
    );
    detail = await openTask("Deploy");
    expect(await within(detail).findByText("Build API")).toBeTruthy();
    expect(within(detail).queryByText("Alpha · Completed")).toBeNull();
    expect(deps.invalidateAll).toHaveBeenCalledTimes(1);
    expect(deps.list).toHaveBeenCalledTimes(4);
  });

  it("AC-12 duplicate activation sends one command and pending cannot close", async () => {
    const user = userEvent.setup();
    const values = [task("task", "Build API")];
    let resolveReopen: ((value: WBSNode) => void) | undefined;
    const pending = new Promise<WBSNode>((resolve) => {
      resolveReopen = resolve;
    });
    const api = wbsGateway(values);
    vi.mocked(api.reopen).mockReturnValue(pending);
    renderPanel("open", values, api);
    const detail = await openTask("Build API");
    const confirmation = await openConfirmation(detail);
    const confirm = within(confirmation).getByRole("button", {
      name: "Reopen Task",
    });
    act(() => {
      confirm.click();
      confirm.click();
    });
    expect(api.reopen).toHaveBeenCalledTimes(1);
    expect((confirm as HTMLButtonElement).disabled).toBe(true);
    expect(confirm.getAttribute("aria-busy")).toBe("true");
    await user.keyboard("{Escape}");
    expect(
      screen.getByRole("alertdialog", { name: "Reopen Task?" }),
    ).toBeTruthy();
    act(() => resolveReopen?.(task("task", "Build API", undefined)));
    await waitFor(() =>
      expect(
        screen.queryByRole("alertdialog", { name: "Reopen Task?" }),
      ).toBeNull(),
    );
  });

  it("AC-15 ignores an old dependency response after successful Reopen", async () => {
    const target = task("target", "Build API");
    const other = task("other", "Deploy", undefined);
    let resolveOld: ((response: Response) => void) | undefined;
    const oldResponse = new Promise<Response>((resolve) => {
      resolveOld = resolve;
    });
    let firstOtherRequest = true;
    const fetchMock = vi.fn((input: string | URL | Request) => {
      const url = String(input);
      if (url.endsWith("/tasks/other/dependencies") && firstOtherRequest) {
        firstOtherRequest = false;
        return oldResponse;
      }
      const data = url.endsWith("/tasks/other/dependencies")
        ? { blockedBy: [relation(false)], blocks: [] }
        : { blockedBy: [], blocks: [] };
      return Promise.resolve(
        new Response(JSON.stringify({ data }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      );
    });
    vi.stubGlobal("fetch", fetchMock);
    const deps = createHTTPDependenciesGateway("/api");
    const api = wbsGateway([target, other]);
    renderPanel("open", [target, other], api, deps);

    let detail = await openTask("Deploy");
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
    await userEvent.click(
      within(detail).getByRole("button", { name: "Close" }),
    );
    detail = await openTask("Build API");
    const confirmation = await openConfirmation(detail);
    await userEvent.click(
      within(confirmation).getByRole("button", { name: "Reopen Task" }),
    );
    resolveOld?.(
      new Response(
        JSON.stringify({
          data: { blockedBy: [relation(true)], blocks: [] },
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );
    detail = await screen.findByRole("dialog", { name: "Edit Task" });
    await userEvent.click(
      within(detail).getByRole("button", { name: "Close" }),
    );
    detail = await openTask("Deploy");
    expect(await within(detail).findByText("Build API")).toBeTruthy();
    expect(within(detail).queryByText(/Completed/)).toBeNull();
    expect(fetchMock).toHaveBeenCalledTimes(4);
  });

  it("AC-16 completes the Reopen workflow using keyboard activation only", async () => {
    const user = userEvent.setup();
    const values = [task("task", "Build API")];
    const api = wbsGateway(values);
    renderPanel("open", values, api);
    const tree = await screen.findByRole("tree");
    const item = within(tree)
      .getByText("Build API")
      .closest<HTMLElement>('[role="treeitem"]');
    if (!item) throw new Error("task not found");
    const edit = within(item).getByRole("button", { name: "Edit Task" });
    edit.focus();
    await user.keyboard("{Enter}");
    const detail = await screen.findByRole("dialog", { name: "Edit Task" });
    const trigger = within(detail).getByRole("button", { name: "Reopen Task" });
    trigger.focus();
    await user.keyboard("{Enter}");
    const confirmation = await screen.findByRole("alertdialog", {
      name: "Reopen Task?",
    });
    expect(document.activeElement).toBe(
      within(confirmation).getByRole("button", { name: "Cancel" }),
    );
    await user.keyboard("{Tab}{Enter}");
    await waitFor(() => expect(api.reopen).toHaveBeenCalledTimes(1));
    expect(await screen.findByText("Task reopened.")).toBeTruthy();
  });

  it("MVF-01 AC-3 AC-16 portals the active confirmation above Task detail with contained narrow-viewport content", async () => {
    Object.defineProperty(window, "innerWidth", {
      configurable: true,
      value: 360,
    });
    const values = [
      task(
        "task",
        "A Task With A Very Long Name That Must Wrap On A Narrow Viewport",
      ),
    ];
    renderPanel("open", values);
    const detail = await openTask(
      "A Task With A Very Long Name That Must Wrap On A Narrow Viewport",
    );
    const confirmation = await openConfirmation(detail);
    expect(confirmation.getAttribute("aria-modal")).toBe("true");
    expect(confirmation.getAttribute("aria-describedby")).toBe(
      "reopen-task-description",
    );
    expect(confirmation.className).toContain("dialog-panel");
    expect(detail.contains(confirmation)).toBe(false);
    const detailOverlay = detail.parentElement;
    const confirmationOverlay = confirmation.parentElement;
    expect(detailOverlay?.parentElement).toBe(document.body);
    expect(confirmationOverlay?.parentElement).toBe(document.body);
    expect(detailOverlay?.dataset.dialogDepth).toBe("1");
    expect(confirmationOverlay?.dataset.dialogDepth).toBe("2");
    expect(detail.getAttribute("aria-hidden")).toBe("true");
    expect(detail.getAttribute("aria-modal")).toBeNull();
    expect(confirmation.dataset.dialogActive).toBe("true");
    expect(detailOverlay?.style.getPropertyValue("--dialog-layer-offset")).toBe(
      "0",
    );
    expect(
      confirmationOverlay?.style.getPropertyValue("--dialog-layer-offset"),
    ).toBe("1");
    expect(document.activeElement?.closest('[role="alertdialog"]')).toBe(
      confirmation,
    );
    expect(
      within(confirmation).getByText(
        "A Task With A Very Long Name That Must Wrap On A Narrow Viewport",
      ).className,
    ).toContain("break-words");
    expect(
      within(confirmation).getByRole("button", { name: "Cancel" }),
    ).toBeTruthy();
    expect(
      within(confirmation).getByRole("button", { name: "Reopen Task" }),
    ).toBeTruthy();
    const results = await axe.run(document.body, {
      rules: { "color-contrast": { enabled: false } },
    });
    expect(results.violations).toEqual([]);
  });
});

function flatten(values: WBSNode[]): WBSNode[] {
  return values.flatMap((value) => [value, ...flatten(value.children)]);
}

function relation(completed: boolean) {
  return {
    id: "incoming-link",
    source: "manual",
    manualRemovable: true,
    task: {
      id: "target",
      name: "Build API",
      projectId: "project",
      projectName: "Alpha",
      hierarchyPath: "Alpha > Build API",
      completed,
    },
  };
}
