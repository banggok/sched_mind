import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import axe from "axe-core";
import { describe, expect, it, vi } from "vitest";
import { formatDateOnly } from "../../../shared/presentation/formatDateOnly";
import type { Project } from "../../projects/domain/project";
import type { RolesGateway } from "../../roles/application/rolesGateway";
import type { TeamMembersGateway } from "../../team-members/application/teamMembersGateway";
import type { WBSGateway } from "../application/wbsGateway";
import type { WBSNode } from "../domain/wbs";
import { WBSPanel } from "./WBSPanel";

const rolesGateway: RolesGateway = {
  list: vi.fn().mockResolvedValue({
    items: [
      {
        id: "role",
        name: "Engineer",
        createdAt: new Date("2026-07-01T00:00:00Z"),
        updatedAt: new Date("2026-07-01T00:00:00Z"),
      },
    ],
    page: 1,
    pageSize: 100,
    total: 1,
  }),
  create: vi.fn<RolesGateway["create"]>(),
  update: vi.fn<RolesGateway["update"]>(),
  delete: vi.fn<RolesGateway["delete"]>(),
};

const membersGateway: TeamMembersGateway = {
  list: vi.fn().mockResolvedValue({
    items: [
      {
        id: "member",
        name: "Rina",
        role: { id: "role", name: "Engineer" },
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
  create: vi.fn<TeamMembersGateway["create"]>(),
  update: vi.fn<TeamMembersGateway["update"]>(),
  delete: vi.fn<TeamMembersGateway["delete"]>(),
};

function project(status: Project["status"] = "open"): Project {
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
  input: Partial<WBSNode["executable"]> = {},
): WBSNode {
  return {
    id,
    projectId: "project",
    parentId: "delivery",
    name,
    position: 1,
    hasChildren: false,
    executable: {
      roleId: "role",
      assigneeId: "member",
      lagDays: 0,
      executionTimeline: {},
      commitmentTimeline: {},
      ...input,
    },
    children: [],
  };
}

function group(id: string, name: string, children: WBSNode[]): WBSNode {
  return {
    id,
    projectId: "project",
    name,
    position: 1,
    hasChildren: true,
    executable: {
      effortMinutes: 9999,
      actualStart: "2020-01-01",
      actualEnd: "2020-01-01",
      lagDays: 0,
      executionTimeline: { start: "2020-01-01", end: "2030-01-01" },
      commitmentTimeline: { start: "2020-01-01", end: "2030-01-01" },
    },
    children,
  };
}

function realisticTree(): WBSNode[] {
  return [
    group("delivery", "Delivery", [
      task("a", "Task A", {
        effortMinutes: 960,
        actualStart: "2026-08-01",
        actualEnd: "2026-08-03",
        executionTimeline: { start: "2026-08-01", end: "2026-08-03" },
        commitmentTimeline: { start: "2026-08-01", end: "2026-08-05" },
      }),
      group("nested", "Nested", [
        task("b", "Task B", {
          effortMinutes: 1440,
          executionTimeline: { start: "2026-08-04", end: "2026-08-06" },
          commitmentTimeline: { start: "2026-08-04", end: "2026-08-08" },
        }),
        group("deep", "Deep", [
          task("c", "Task C", {
            actualStart: "2026-08-09",
            actualEnd: "2026-08-10",
            commitmentTimeline: {
              start: "2026-08-09",
              end: "2026-08-12",
            },
          }),
          task("d", "Task D", {
            effortMinutes: 480,
            actualStart: "2026-08-08",
            actualEnd: "2026-08-12",
            executionTimeline: {
              start: "2026-08-08",
              end: "2026-08-12",
            },
          }),
        ]),
      ]),
      task("e", "Task E"),
    ]),
  ];
}

function mutableGateway(initial: WBSNode[]): {
  gateway: WBSGateway;
  current(): WBSNode[];
  confirmActualDate(id: string, actualStart: string, actualEnd: string): void;
} {
  let values = initial;
  const listeners = new Set<() => void>();
  const notify = () => listeners.forEach((listener) => listener());
  const gateway: WBSGateway = {
    tree: vi.fn(async () => values),
    allocations: vi
      .fn()
      .mockResolvedValue({ execution: [], commitment: [], actual: [] }),
    subscribeToConfirmedChanges: (listener) => {
      listeners.add(listener);
      return () => listeners.delete(listener);
    },
    create: vi.fn().mockResolvedValue(undefined),
    rename: vi.fn().mockResolvedValue(undefined),
    reorder: vi.fn().mockResolvedValue(undefined),
    move: vi.fn().mockResolvedValue(undefined),
    remove: vi.fn().mockResolvedValue(undefined),
    updateExecutable: vi.fn().mockResolvedValue(undefined),
    previewExecutableSchedule: vi.fn(async (_projectId, id, input) => {
      const current = flatten(values).find((value) => value.id === id);
      if (!current) throw new Error("missing Task fixture");
      return {
        task: {
          ...current,
          executable: {
            ...current.executable,
            effortMinutes: input.effortHours * 60,
            executionTimeline: {
              start: "2027-01-01",
              end: "2027-01-02",
            },
          },
        },
        dependencies: { blockedBy: [], blocks: [] },
      };
    }),
    complete: vi.fn(async (_projectId, id, actualStart, actualEnd) => {
      values = replace(values, id, (value) => ({
        ...value,
        executable: { ...value.executable, actualStart, actualEnd },
      }));
      notify();
    }),
    reopen: vi.fn(async (_projectId, id) => {
      let confirmed: WBSNode | undefined;
      values = replace(values, id, (value) => {
        confirmed = {
          ...value,
          executable: {
            ...value.executable,
            actualStart: undefined,
            actualEnd: undefined,
          },
        };
        return confirmed;
      });
      if (!confirmed) throw new Error("missing Task fixture");
      notify();
      return confirmed;
    }),
  };
  return {
    gateway,
    current: () => values,
    confirmActualDate: (id, actualStart, actualEnd) => {
      values = replace(values, id, (value) => ({
        ...value,
        executable: { ...value.executable, actualStart, actualEnd },
      }));
      notify();
    },
  };
}

function renderPanel(
  values: WBSNode[],
  status: Project["status"] = "open",
  gateway = mutableGateway(values).gateway,
) {
  return render(
    <WBSPanel
      project={project(status)}
      gateway={gateway}
      rolesGateway={rolesGateway}
      membersGateway={membersGateway}
      onClose={() => undefined}
    />,
  );
}

function getTreeItemAction(
  item: HTMLElement,
  name: "View Group" | "Edit Task",
): HTMLElement {
  const actionRow = item.firstElementChild;
  if (!(actionRow instanceof HTMLElement)) {
    throw new Error(`Action row for ${name} not found`);
  }
  return within(actionRow).getByRole("button", { name });
}

async function openGroup(name: string): Promise<HTMLElement> {
  const tree = await screen.findByRole("tree");
  const item = within(tree)
    .getByText(name)
    .closest<HTMLElement>('[role="treeitem"]');
  if (!item) throw new Error(`Group ${name} not found`);
  await userEvent.click(getTreeItemAction(item, "View Group"));
  return screen.findByRole("dialog", { name });
}

async function openTask(name: string): Promise<HTMLElement> {
  const tree = await screen.findByRole("tree");
  const item = within(tree)
    .getByText(name)
    .closest<HTMLElement>('[role="treeitem"]');
  if (!item) throw new Error(`Task ${name} not found`);
  await userEvent.click(getTreeItemAction(item, "Edit Task"));
  return screen.findByRole("dialog", { name: "Edit Task" });
}

describe("US-4.3 acceptance workflow through Project Structure", () => {
  it("AC-1..16 AC-22 AC-24..26 recursively renders exact Group summary accessibly and performs no write", async () => {
    const setup = mutableGateway(realisticTree());
    renderPanel(setup.current(), "open", setup.gateway);
    const deliveryItem = (
      await screen.findByText("Delivery")
    ).closest<HTMLElement>('[role="treeitem"]');
    if (!deliveryItem) throw new Error("Delivery Group not found");
    const trigger = getTreeItemAction(deliveryItem, "View Group");
    const dialog = await openGroup("Delivery");

    expect(
      within(dialog)
        .getAllByRole("heading", { level: 4 })
        .map((heading) => heading.textContent),
    ).toEqual([
      "Execution Timeline",
      "Commitment Timeline",
      "Effort Completion",
    ]);
    expect(
      within(dialog).getAllByText(
        `${formatDateOnly("2026-08-01")} – ${formatDateOnly("2026-08-12")}`,
      ),
    ).toHaveLength(2);
    expect(within(dialog).getAllByText("3 of 5 tasks scheduled")).toHaveLength(
      2,
    );
    expect(
      within(dialog).getByText("24 of 48 hours completed (50%)"),
    ).toBeTruthy();
    expect(
      within(dialog).getByText(
        "2 tasks without effort are excluded from this calculation.",
      ),
    ).toBeTruthy();
    expect(within(dialog).queryByText(/direct item/i)).toBeNull();
    expect(
      within(dialog).getAllByText(
        `Start ${formatDateOnly("2026-08-01")}; End ${formatDateOnly("2026-08-12")}`,
        { selector: ".sr-only" },
      ),
    ).toHaveLength(2);
    const summaryGrid = within(dialog).getByLabelText("Group summary details");
    expect(summaryGrid.className).toContain("min-w-0");
    expect(summaryGrid.classList.contains("grid-cols-3")).toBe(false);
    expect(summaryGrid.classList.contains("lg:grid-cols-3")).toBe(false);
    expect(
      within(dialog).getByText("24 of 48 hours completed (50%)").className,
    ).toContain("break-words");
    expect(
      within(dialog).getByText(
        "2 tasks without effort are excluded from this calculation.",
      ).className,
    ).toContain("break-words");

    const results = await axe.run(dialog, {
      rules: { "color-contrast": { enabled: false } },
    });
    expect(results.violations).toEqual([]);
    for (const write of [
      setup.gateway.create,
      setup.gateway.rename,
      setup.gateway.reorder,
      setup.gateway.move,
      setup.gateway.remove,
      setup.gateway.updateExecutable,
      setup.gateway.complete,
      setup.gateway.reopen,
    ]) {
      expect(write).not.toHaveBeenCalled();
    }

    await userEvent.click(
      within(dialog).getByRole("button", { name: "Close" }),
    );
    expect(screen.queryByRole("dialog", { name: "Delivery" })).toBeNull();
    expect(document.activeElement).toBe(trigger);
  });

  it("AC-23 AC-27 keeps the same read-only rule for every lifecycle and handles an empty malformed Group", async () => {
    for (const status of ["open", "locked", "closed"] as const) {
      const setup = mutableGateway([
        group("delivery", "Delivery", [
          task("known", "Known", { effortMinutes: 480 }),
        ]),
      ]);
      const view = renderPanel(setup.current(), status, setup.gateway);
      const dialog = await openGroup("Delivery");
      expect(
        within(dialog).getByText("0 of 8 hours completed (0%)"),
      ).toBeTruthy();
      expect(within(dialog).getAllByText("Not scheduled")).toHaveLength(2);
      expect(
        within(dialog).getAllByText("0 of 1 tasks scheduled"),
      ).toHaveLength(2);
      expect(setup.gateway.tree).toHaveBeenCalledTimes(1);
      view.unmount();
    }

    renderPanel([group("empty", "Empty Group", [])]);
    const emptyDialog = await openGroup("Empty Group");
    expect(
      within(emptyDialog).getByText(
        "No descendant tasks are available for this group.",
      ),
    ).toBeTruthy();
    expect(within(emptyDialog).queryByText(/tasks scheduled/)).toBeNull();
  });

  it("AC-14..16 renders unavailable Effort and plural disclosure without inventing zero Effort", async () => {
    renderPanel([
      group("delivery", "Unknown Effort", [
        task("completed", "Completed without Effort", {
          actualStart: "2026-08-01",
          actualEnd: "2026-08-01",
        }),
        task("unfinished", "Unfinished without Effort"),
      ]),
    ]);

    const dialog = await openGroup("Unknown Effort");
    expect(
      within(dialog).getByText("Effort completion unavailable"),
    ).toBeTruthy();
    expect(
      within(dialog).getByText(
        "2 tasks without effort are excluded from this calculation.",
      ),
    ).toBeTruthy();
    expect(within(dialog).queryByText(/completed \(.*%\)/)).toBeNull();
    expect(within(dialog).getAllByText("Not scheduled")).toHaveLength(2);
    expect(within(dialog).getAllByText("0 of 2 tasks scheduled")).toHaveLength(
      2,
    );
  });

  it("AC-12 AC-13 renders exact completion copy, one-decimal precision, and half-hour values", async () => {
    let view = renderPanel([
      group("delivery", "Exact Completion", [
        task("completed", "Completed", {
          effortMinutes: 1440,
          actualStart: "2026-08-01",
          actualEnd: "2026-08-01",
        }),
        task("unfinished", "Unfinished", { effortMinutes: 960 }),
      ]),
    ]);

    let dialog = await openGroup("Exact Completion");
    expect(
      within(dialog).getByText("24 of 40 hours completed (60%)"),
    ).toBeTruthy();
    view.unmount();

    view = renderPanel([
      group("delivery", "Half Hour", [
        task("completed", "Completed", {
          effortMinutes: 750,
          actualStart: "2026-08-01",
          actualEnd: "2026-08-01",
        }),
      ]),
    ]);
    dialog = await openGroup("Half Hour");
    expect(
      within(dialog).getByText("12.5 of 12.5 hours completed (100%)"),
    ).toBeTruthy();
    view.unmount();
  });

  it("AC-20 recomputes a currently open Group from a newer confirmed tree", async () => {
    const setup = mutableGateway([
      group("delivery", "Delivery", [
        task("target", "Target", { effortMinutes: 960 }),
        task("other", "Other", { effortMinutes: 480 }),
      ]),
    ]);
    renderPanel(setup.current(), "open", setup.gateway);

    const dialog = await openGroup("Delivery");
    expect(
      within(dialog).getByText("0 of 24 hours completed (0%)"),
    ).toBeTruthy();

    act(() => setup.confirmActualDate("target", "2026-08-01", "2026-08-01"));

    expect(
      await within(dialog).findByText("16 of 24 hours completed (66.7%)"),
    ).toBeTruthy();
    expect(screen.getByRole("dialog", { name: "Delivery" })).toBe(dialog);
  });

  it("AC-17..20 keeps preview draft unconfirmed, then refreshes completion and Reopen without hard reload", async () => {
    const values = [
      group("delivery", "Delivery", [
        task("target", "Target", { effortMinutes: 960 }),
        task("other", "Other", { effortMinutes: 480 }),
      ]),
    ];
    const setup = mutableGateway(values);
    renderPanel(setup.current(), "open", setup.gateway);

    let groupDialog = await openGroup("Delivery");
    expect(
      within(groupDialog).getByText("0 of 24 hours completed (0%)"),
    ).toBeTruthy();
    await userEvent.click(
      within(groupDialog).getByRole("button", { name: "Close" }),
    );

    let taskDialog = await openTask("Target");
    const effort = within(taskDialog).getByLabelText("Effort (hours)");
    fireEvent.change(effort, { target: { value: "40" } });
    fireEvent.blur(effort);
    await waitFor(() =>
      expect(setup.gateway.previewExecutableSchedule).toHaveBeenCalled(),
    );
    await userEvent.click(
      within(taskDialog).getByRole("button", { name: "Close" }),
    );

    groupDialog = await openGroup("Delivery");
    expect(
      within(groupDialog).getByText("0 of 24 hours completed (0%)"),
    ).toBeTruthy();
    expect(within(groupDialog).queryByText(/48 hours/)).toBeNull();
    await userEvent.click(
      within(groupDialog).getByRole("button", { name: "Close" }),
    );

    taskDialog = await openTask("Target");
    fireEvent.click(
      within(taskDialog).getByRole("button", {
        name: "Actual Date: Select start and end date",
      }),
    );
    const current = new Date();
    const currentMonth = `${current.getUTCFullYear()}-${String(current.getUTCMonth() + 1).padStart(2, "0")}`;
    const selectedStartDate = `${currentMonth}-14`;
    const selectedEndDate = `${currentMonth}-15`;
    fireEvent.click(screen.getByRole("button", { name: selectedStartDate }));
    fireEvent.click(screen.getByRole("button", { name: selectedEndDate }));
    await userEvent.click(
      within(taskDialog).getByRole("button", { name: "Mark completed" }),
    );
    await waitFor(() => expect(setup.gateway.complete).toHaveBeenCalled());
    await waitFor(() =>
      expect(
        vi.mocked(setup.gateway.tree).mock.calls.length,
      ).toBeGreaterThanOrEqual(2),
    );

    groupDialog = await openGroup("Delivery");
    expect(
      within(groupDialog).getByText("16 of 24 hours completed (66.7%)"),
    ).toBeTruthy();
    await userEvent.click(
      within(groupDialog).getByRole("button", { name: "Close" }),
    );

    taskDialog = await openTask("Target");
    await userEvent.click(
      within(taskDialog).getByRole("button", { name: "Reopen Task" }),
    );
    const confirmation = await screen.findByRole("alertdialog", {
      name: "Reopen Task?",
    });
    await userEvent.click(
      within(confirmation).getByRole("button", { name: "Reopen Task" }),
    );
    await waitFor(() => expect(setup.gateway.reopen).toHaveBeenCalled());
    await userEvent.click(
      within(screen.getByRole("dialog", { name: "Edit Task" })).getByRole(
        "button",
        { name: "Close" },
      ),
    );

    groupDialog = await openGroup("Delivery");
    expect(
      within(groupDialog).getByText("0 of 24 hours completed (0%)"),
    ).toBeTruthy();
  });
});

function flatten(values: WBSNode[]): WBSNode[] {
  return values.flatMap((value) => [value, ...flatten(value.children)]);
}

function replace(
  values: WBSNode[],
  id: string,
  update: (value: WBSNode) => WBSNode,
): WBSNode[] {
  return values.map((value) =>
    value.id === id
      ? update(value)
      : { ...value, children: replace(value.children, id, update) },
  );
}
