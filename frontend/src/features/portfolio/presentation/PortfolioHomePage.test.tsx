import {
  act,
  createEvent,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import axe from "axe-core";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { advanceScheduleProjectionVersion } from "../../../shared/infrastructure/scheduleProjectionClock";
import type { WBSGateway } from "../../wbs/application/wbsGateway";
import type { PortfolioGateway } from "../application/portfolioGateway";
import { portfolioCollapsedRowsPreferenceKey } from "../infrastructure/hierarchyPreference";
import { portfolioColumnWidthsPreferenceKey } from "../infrastructure/layoutPreference";
import type {
  PortfolioProject,
  PortfolioProjection,
  PortfolioProjectionResult,
  SavedPortfolioFilter,
} from "../domain/portfolio";
import { PortfolioHomePage } from "./PortfolioHomePage";

const projects: PortfolioProject[] = [
  {
    id: "alpha",
    name: "Alpha",
    status: "open",
    priority: 1,
    scheduleVersion: 3,
  },
  {
    id: "beta",
    name: "Beta",
    status: "locked",
    priority: 2,
    scheduleVersion: 8,
  },
];

const focusFilter: SavedPortfolioFilter = {
  id: "focus",
  name: "Focus",
  projectIds: ["beta"],
  version: 2,
  createdAt: new Date("2026-08-01T00:00:00Z"),
  updatedAt: new Date("2026-08-01T00:00:00Z"),
};

function projection(
  selected: string[],
  value: PortfolioProjection,
): PortfolioProjectionResult {
  const rows: PortfolioProjectionResult["rows"] = [];
  if (selected.includes("alpha")) {
    rows.push(
      {
        id: "alpha",
        projectId: "alpha",
        kind: "project",
        name: "Alpha",
        wbsNumber: "",
        depth: 0,
        position: 0,
        status: "open",
        effortMinutes: 480,
        start: "2026-08-03",
        end: "2026-08-05",
        incompleteEffort: false,
        incompleteSchedule: false,
        hasChildren: true,
        completed: false,
      },
      {
        id: "delivery",
        projectId: "alpha",
        kind: "group",
        name: "Delivery",
        wbsNumber: "1",
        depth: 1,
        position: 1,
        effortMinutes: 480,
        start: "2026-08-03",
        end: "2026-08-05",
        incompleteEffort: false,
        incompleteSchedule: false,
        hasChildren: true,
        completed: false,
      },
      {
        id: "completed",
        projectId: "alpha",
        parentId: "delivery",
        kind: "task",
        name: "Completed",
        wbsNumber: "1.1",
        depth: 2,
        position: 1,
        roleId: "development",
        roleName: "Development",
        assigneeId: "member",
        assigneeName: "Alice",
        effortMinutes: 480,
        start: "2026-08-03",
        end: "2026-08-05",
        incompleteEffort: false,
        incompleteSchedule: false,
        hasChildren: false,
        completed: true,
      },
      {
        id: "testing",
        projectId: "alpha",
        parentId: "delivery",
        kind: "task",
        name: "QA Pass",
        wbsNumber: "1.2",
        depth: 2,
        position: 2,
        roleId: "testing",
        roleName: "Testing",
        assigneeId: "member",
        assigneeName: "Alice",
        effortMinutes: 240,
        start: "2026-08-04",
        end: "2026-08-05",
        incompleteEffort: false,
        incompleteSchedule: false,
        hasChildren: false,
        completed: false,
      },
    );
  }
  if (selected.includes("beta")) {
    rows.push({
      id: "beta",
      projectId: "beta",
      kind: "project",
      name: "Beta",
      wbsNumber: "",
      depth: 0,
      position: 0,
      status: "locked",
      start: "2026-08-06",
      end: "2026-08-07",
      incompleteEffort: true,
      incompleteSchedule: false,
      hasChildren: false,
      completed: false,
    });
  }
  return {
    projection: value,
    projects: projects.filter((project) => selected.includes(project.id)),
    rows,
    dependencies: selected.includes("alpha")
      ? [
          {
            id: "dependency",
            blockingTaskId: "completed",
            blockedTaskId: "completed",
          },
        ]
      : [],
    holidays: [{ date: "2026-08-04", description: "Holiday" }],
    workingDayAnchor: "2026-08-03",
  };
}

function gateway(): PortfolioGateway {
  return {
    activeProjects: vi.fn().mockResolvedValue(projects),
    savedFilters: vi.fn().mockResolvedValue([focusFilter]),
    projection: vi
      .fn()
      .mockImplementation((selected: string[], value: PortfolioProjection) =>
        Promise.resolve(projection(selected, value)),
      ),
    createSavedFilter: vi.fn().mockResolvedValue(focusFilter),
    updateSavedFilter: vi
      .fn()
      .mockImplementation((_id: string, version: number, selected: string[]) =>
        Promise.resolve({
          ...focusFilter,
          projectIds: selected,
          version: version + 1,
        }),
      ),
    deleteSavedFilter: vi.fn().mockResolvedValue(undefined),
  };
}

function wbsGateway(): WBSGateway {
  return {
    tree: vi.fn().mockResolvedValue([]),
    allocations: vi.fn().mockResolvedValue({
      execution: [],
      commitment: [],
      actual: [],
    }),
    create: vi.fn().mockResolvedValue(undefined),
    createSibling: vi.fn().mockResolvedValue(undefined),
    rename: vi.fn().mockResolvedValue(undefined),
    reorder: vi.fn().mockResolvedValue(undefined),
    place: vi.fn().mockResolvedValue(undefined),
    move: vi.fn().mockResolvedValue(undefined),
    remove: vi.fn().mockResolvedValue(undefined),
    updateExecutable: vi.fn().mockResolvedValue(undefined),
    previewExecutableSchedule: vi.fn(),
    complete: vi.fn().mockResolvedValue(undefined),
    reopen: vi.fn(),
  };
}

async function findPortfolioRow(
  container: HTMLElement,
  rowID: string,
): Promise<HTMLElement> {
  await waitFor(() =>
    expect(
      container.querySelector(`[data-portfolio-row="${rowID}"]`),
    ).not.toBeNull(),
  );
  const row = container.querySelector<HTMLElement>(
    `[data-portfolio-row="${rowID}"]`,
  );
  if (!row) throw new Error(`Portfolio row ${rowID} not found`);
  return row;
}

function deferred<Value>() {
  let resolve!: (value: Value | PromiseLike<Value>) => void;
  const promise = new Promise<Value>((next) => {
    resolve = next;
  });
  return { promise, resolve };
}

describe("US-7.1 Home portfolio Gantt acceptance workflow", () => {
  beforeEach(() => window.localStorage.clear());
  it("loads one portfolio projection, preserves hierarchy actions, filters projects, and switches projection", async () => {
    const user = userEvent.setup();
    const portfolioGateway = gateway();
    const structureGateway = wbsGateway();
    const openProject = vi.fn();
    const openWBS = vi.fn();
    const { container } = render(
      <PortfolioHomePage
        gateway={portfolioGateway}
        wbsGateway={structureGateway}
        onOpenProject={openProject}
        onProjectCommand={vi.fn()}
        onOpenWBS={openWBS}
      />,
    );

    expect(
      await screen.findByRole("heading", { name: "All Active Projects" }),
    ).toBeTruthy();
    expect(screen.queryByRole("navigation", { name: "Breadcrumb" })).toBeNull();
    expect(
      screen.queryByRole("heading", { name: "Portfolio Gantt" }),
    ).toBeNull();
    expect(screen.queryByLabelText("Saved filter")).toBeNull();
    expect(
      screen.queryByRole("switch", { name: "Execution projection" }),
    ).toBeNull();
    await waitFor(() =>
      expect(portfolioGateway.projection).toHaveBeenCalledWith(
        ["alpha", "beta"],
        "execution",
        expect.any(String),
        expect.any(String),
        expect.any(AbortSignal),
      ),
    );
    const alphaRow = await findPortfolioRow(container, "alpha");
    expect(portfolioGateway.projection).toHaveBeenCalledTimes(1);
    expect(
      screen
        .getByRole("treegrid", { name: "Portfolio schedule rows" })
        .getAttribute("aria-colcount"),
    ).toBe("7");
    expect(screen.queryByText("Actions", { selector: "span" })).toBeNull();
    expect(screen.getByLabelText("Timeline months")).toBeTruthy();
    expect(screen.getByText("Aug 2026")).toBeTruthy();
    expect(
      screen.getByText("2026-08-03", { selector: "span.sr-only" }),
    ).toBeTruthy();
    expect(
      screen.queryByText("2026-07-27", { selector: "span.sr-only" }),
    ).toBeNull();
    expect(
      within(alphaRow).getByRole("button", { name: "Alpha" }),
    ).toBeTruthy();
    const alphaCells = within(alphaRow).getAllByRole("gridcell");
    expect(alphaCells).toHaveLength(7);
    expect(alphaCells[2].textContent).toBe("");
    expect(alphaCells[5].textContent).toBe("3 Aug 2026");
    expect(alphaCells[6].textContent).toBe("5 Aug 2026");
    const projectAddChildButton = within(alphaRow).getByRole("button", {
      name: "Add Child for Alpha",
    });
    expect(projectAddChildButton.textContent).toContain("Add Child");
    expect(projectAddChildButton.getAttribute("title")).toBe(
      "Add Child under Alpha",
    );
    expect(
      within(alphaRow).queryByRole("button", { name: /Add Sibling/ }),
    ).toBeNull();

    const lockedProjectRow = await findPortfolioRow(container, "beta");
    expect(
      within(lockedProjectRow).getByRole("button", { name: "Beta" }),
    ).toBeTruthy();
    expect(within(lockedProjectRow).getByText("Locked")).toBeTruthy();
    expect(
      within(lockedProjectRow).queryByRole("button", { name: /Reorder/ }),
    ).toBeNull();
    expect(
      within(lockedProjectRow).queryByRole("button", { name: /Add Child/ }),
    ).toBeNull();

    const completedRow = await findPortfolioRow(container, "completed");
    expect(
      within(completedRow).getByRole("button", { name: "Completed" }),
    ).toBeTruthy();
    expect(within(completedRow).getByText("Development")).toBeTruthy();
    const deliveryRow = await findPortfolioRow(container, "delivery");
    expect(within(deliveryRow).getAllByRole("gridcell")[2].textContent).toBe(
      "",
    );
    expect(
      within(completedRow).getByRole("button", {
        name: "Add Sibling for Completed",
      }),
    ).toBeTruthy();
    expect(
      within(completedRow).queryByRole("button", {
        name: "Add Child for Completed",
      }),
    ).toBeNull();

    await user.click(screen.getByRole("button", { name: "Configure Gantt" }));
    const filterDialog = await screen.findByRole("dialog", {
      name: "Gantt configuration",
    });
    expect(
      within(filterDialog).getByRole("heading", { name: "Projects" }),
    ).toBeTruthy();
    expect(
      within(filterDialog).getByRole("heading", { name: "Roles" }),
    ).toBeTruthy();
    const roleSection = within(filterDialog)
      .getByRole("heading", { name: "Roles" })
      .closest("section");
    if (!roleSection) throw new Error("Roles filter section not found");
    await user.click(
      within(roleSection).getByRole("button", { name: "Clear All" }),
    );
    await user.click(
      within(roleSection).getByRole("checkbox", { name: "Development" }),
    );
    await user.click(
      within(filterDialog).getByRole("button", { name: "Apply" }),
    );
    await waitFor(() =>
      expect(
        container.querySelector('[data-portfolio-row="testing"]'),
      ).toBeNull(),
    );
    const filteredCompletedRow = await findPortfolioRow(container, "completed");
    expect(within(filteredCompletedRow).getByText("Development")).toBeTruthy();

    await user.click(screen.getByRole("button", { name: "Configure Gantt" }));
    const restoreFilterDialog = await screen.findByRole("dialog", {
      name: "Gantt configuration",
    });
    const rolesSection = within(restoreFilterDialog)
      .getByRole("heading", { name: "Roles" })
      .closest("section");
    if (!rolesSection) throw new Error("Roles filter section not found");
    await user.click(
      within(rolesSection).getByRole("button", { name: "Select All" }),
    );
    await user.click(
      within(restoreFilterDialog).getByRole("button", { name: "Apply" }),
    );
    const testingRow = await findPortfolioRow(container, "testing");
    expect(within(testingRow).getByText("Testing")).toBeTruthy();
    const addSiblingButton = within(testingRow).getByRole("button", {
      name: "Add Sibling for QA Pass",
    });
    const addChildButton = within(testingRow).getByRole("button", {
      name: "Add Child for QA Pass",
    });
    expect(addSiblingButton.textContent).toContain("Add Sibling");
    expect(addSiblingButton.getAttribute("title")).toBe(
      "Add Sibling after QA Pass",
    );
    expect(addChildButton.textContent).toContain("Add Child");
    expect(addChildButton.getAttribute("title")).toBe(
      "Add Child under QA Pass",
    );
    expect(within(testingRow).getAllByText("⊕")).toHaveLength(1);
    const testingNameStack = testingRow.querySelector(
      '[data-row-name-stack="testing"]',
    );
    const testingNameLine = testingRow.querySelector(
      '[data-row-name-line="testing"]',
    );
    const testingCreateActions = testingRow.querySelector(
      '[data-row-create-actions="testing"]',
    );
    expect(testingNameStack).toBeTruthy();
    expect(testingNameStack?.className).toContain("flex-col");
    expect(testingNameStack?.children[0]).toBe(testingNameLine);
    expect(testingNameStack?.children[1]).toBe(testingCreateActions);
    expect(testingNameLine?.contains(addSiblingButton)).toBe(false);
    expect(testingCreateActions?.contains(addSiblingButton)).toBe(true);
    expect(testingCreateActions?.className).toContain("absolute");
    expect(testingCreateActions?.className).toContain("w-max");
    expect(testingCreateActions?.className).not.toContain("overflow-hidden");
    expect(testingNameStack?.className).toContain("overflow-visible");
    const testingTimelineRow = container.querySelector<HTMLElement>(
      '[data-portfolio-timeline-row="testing"]',
    );
    expect(testingRow.style.height).toBe("40px");
    expect(testingTimelineRow?.style.height).toBe("40px");

    fireEvent.mouseEnter(testingRow);
    await waitFor(() => {
      expect(testingRow.style.height).toBe("64px");
      expect(testingTimelineRow?.style.height).toBe("64px");
    });

    fireEvent.mouseLeave(testingRow);
    await waitFor(() => {
      expect(testingRow.style.height).toBe("40px");
      expect(testingTimelineRow?.style.height).toBe("40px");
    });

    fireEvent.keyDown(document, { key: "Tab" });
    addSiblingButton.focus();
    await waitFor(() => {
      expect(testingRow.style.height).toBe("64px");
      expect(testingTimelineRow?.style.height).toBe("64px");
    });
    addSiblingButton.blur();
    await waitFor(() => {
      expect(testingRow.style.height).toBe("40px");
      expect(testingTimelineRow?.style.height).toBe("40px");
    });

    const deliveryRowForHoverInteraction = await findPortfolioRow(
      container,
      "delivery",
    );
    const deliveryTimelineRowForHoverInteraction =
      container.querySelector<HTMLElement>(
        '[data-portfolio-timeline-row="delivery"]',
      );
    fireEvent.mouseEnter(deliveryRowForHoverInteraction);
    await waitFor(() => {
      expect(deliveryRowForHoverInteraction.style.height).toBe("64px");
      expect(deliveryTimelineRowForHoverInteraction?.style.height).toBe("64px");
    });
    await user.click(
      within(deliveryRowForHoverInteraction).getByRole("button", {
        name: "Collapse Delivery",
      }),
    );
    fireEvent.mouseLeave(deliveryRowForHoverInteraction);
    await waitFor(() => {
      expect(deliveryRowForHoverInteraction.style.height).toBe("40px");
      expect(deliveryTimelineRowForHoverInteraction?.style.height).toBe("40px");
    });
    await user.click(
      within(deliveryRowForHoverInteraction).getByRole("button", {
        name: "Expand Delivery",
      }),
    );

    const testingRowAfterExpand = await findPortfolioRow(container, "testing");
    const testingTimelineRowAfterExpand = container.querySelector<HTMLElement>(
      '[data-portfolio-timeline-row="testing"]',
    );
    const testingCreateActionsAfterExpand = testingRowAfterExpand.querySelector(
      '[data-row-create-actions="testing"]',
    );
    fireEvent.mouseEnter(testingRowAfterExpand);
    await waitFor(() =>
      expect(testingRowAfterExpand.style.height).toBe("64px"),
    );
    await user.click(
      within(testingRowAfterExpand).getByRole("button", {
        name: "More actions for QA Pass",
      }),
    );
    expect(
      screen.getByRole("menu", { name: "Actions for QA Pass" }),
    ).toBeTruthy();
    fireEvent.mouseLeave(testingRowAfterExpand);
    await waitFor(() => {
      expect(testingRowAfterExpand.style.height).toBe("40px");
      expect(testingTimelineRowAfterExpand?.style.height).toBe("40px");
      expect(testingCreateActionsAfterExpand?.className).toContain("opacity-0");
    });
    fireEvent.pointerDown(document.body, { pointerType: "mouse" });
    await waitFor(() =>
      expect(
        screen.queryByRole("menu", { name: "Actions for QA Pass" }),
      ).toBeNull(),
    );

    const lockedProjectRowForHover = await findPortfolioRow(container, "beta");
    const lockedTimelineRow = container.querySelector<HTMLElement>(
      '[data-portfolio-timeline-row="beta"]',
    );
    expect(
      lockedProjectRowForHover.querySelector(
        '[data-row-create-actions="beta"]',
      ),
    ).toBeNull();
    fireEvent.mouseEnter(lockedProjectRowForHover);
    await waitFor(() => {
      expect(lockedProjectRowForHover.style.height).toBe("40px");
      expect(lockedTimelineRow?.style.height).toBe("40px");
    });
    within(lockedProjectRowForHover)
      .getByRole("button", { name: "More actions for Beta" })
      .focus();
    await waitFor(() => {
      expect(lockedProjectRowForHover.style.height).toBe("40px");
      expect(lockedTimelineRow?.style.height).toBe("40px");
    });

    await user.click(screen.getByRole("button", { name: "Collapse Delivery" }));
    await waitFor(() =>
      expect(
        container.querySelector('[data-portfolio-row="completed"]'),
      ).toBeNull(),
    );
    await user.click(screen.getByRole("button", { name: "Expand Delivery" }));
    const expandedTestingRow = await findPortfolioRow(container, "testing");
    expect(
      within(expandedTestingRow).getByRole("button", { name: "QA Pass" }),
    ).toBeTruthy();

    await user.click(screen.getByRole("button", { name: "Configure Gantt" }));
    const configurationDialog = await screen.findByRole("dialog", {
      name: "Gantt configuration",
    });
    await user.selectOptions(
      within(configurationDialog).getByLabelText("Saved filter"),
      "focus",
    );
    await user.click(
      within(configurationDialog).getByRole("button", { name: "Save" }),
    );
    await waitFor(() =>
      expect(portfolioGateway.updateSavedFilter).toHaveBeenCalledWith(
        "focus",
        2,
        ["beta"],
      ),
    );
    await user.click(
      within(configurationDialog).getByRole("switch", {
        name: "Commitment projection",
      }),
    );
    await user.click(
      within(configurationDialog).getByRole("button", { name: "Apply" }),
    );
    await waitFor(() =>
      expect(portfolioGateway.projection).toHaveBeenCalledWith(
        ["beta"],
        "commitment",
        expect.any(String),
        expect.any(String),
        expect.any(AbortSignal),
      ),
    );
    expect(await screen.findByRole("heading", { name: "Focus" })).toBeTruthy();

    expect(screen.queryByLabelText("From")).toBeNull();
    expect(screen.queryByLabelText("To")).toBeNull();
    expect(screen.queryByRole("button", { name: "Apply range" })).toBeNull();

    expect((await axe.run(container)).violations).toEqual([]);
  });

  it("US-7.1 AC-56A AC-59A renders same-day Finish-to-Start Tasks as ordered non-overlapping bar slots", async () => {
    const portfolioGateway = gateway();
    vi.mocked(portfolioGateway.projection).mockResolvedValue({
      projection: "execution",
      projects: [projects[0]],
      rows: [
        {
          id: "alpha",
          projectId: "alpha",
          kind: "project",
          name: "Alpha",
          wbsNumber: "",
          depth: 0,
          position: 0,
          status: "open",
          effortMinutes: 960,
          start: "2026-08-03",
          end: "2026-08-06",
          incompleteEffort: false,
          incompleteSchedule: false,
          hasChildren: true,
          completed: false,
        },
        {
          id: "task-3",
          projectId: "alpha",
          kind: "task",
          name: "Task 3",
          wbsNumber: "1",
          depth: 1,
          position: 1,
          roleId: "development",
          roleName: "Development",
          assigneeId: "member",
          assigneeName: "Alice",
          effortMinutes: 480,
          start: "2026-08-03",
          end: "2026-08-05",
          incompleteEffort: false,
          incompleteSchedule: false,
          hasChildren: false,
          completed: false,
        },
        {
          id: "task-4",
          projectId: "alpha",
          kind: "task",
          name: "Task 4",
          wbsNumber: "2",
          depth: 1,
          position: 2,
          roleId: "development",
          roleName: "Development",
          assigneeId: "member",
          assigneeName: "Alice",
          effortMinutes: 480,
          start: "2026-08-05",
          end: "2026-08-06",
          incompleteEffort: false,
          incompleteSchedule: false,
          hasChildren: false,
          completed: false,
        },
      ],
      dependencies: [
        {
          id: "same-day-dependency",
          blockingTaskId: "task-3",
          blockedTaskId: "task-4",
        },
      ],
      holidays: [],
      workingDayAnchor: "2026-08-03",
    });

    const { container } = render(
      <PortfolioHomePage
        gateway={portfolioGateway}
        wbsGateway={wbsGateway()}
        onOpenProject={vi.fn()}
        onProjectCommand={vi.fn()}
        onOpenWBS={vi.fn()}
      />,
    );

    const blockingTimeline = await screen.findByLabelText(
      "Task 3: 2026-08-03 through 2026-08-05",
    );
    const blockedTimeline = screen.getByLabelText(
      "Task 4: 2026-08-05 through 2026-08-06",
    );
    const blockingBar = blockingTimeline.querySelector<HTMLElement>("span");
    const blockedBar = blockedTimeline.querySelector<HTMLElement>("span");
    expect(blockingBar).toBeTruthy();
    expect(blockedBar).toBeTruthy();
    const blockingRight =
      Number.parseFloat(blockingBar?.style.left ?? "0") +
      Number.parseFloat(blockingBar?.style.width ?? "0");
    const blockedLeft = Number.parseFloat(blockedBar?.style.left ?? "0");
    expect(blockingRight).toBeLessThan(blockedLeft);

    const arrow = container.querySelector<SVGPathElement>(
      '[data-portfolio-dependency="same-day-dependency"]',
    );
    expect(arrow).toBeTruthy();
    const horizontalPoints = [
      ...(arrow?.getAttribute("d") ?? "").matchAll(/H ([\d.]+)/g),
    ].map((match) => Number.parseFloat(match[1]));
    expect(horizontalPoints).toHaveLength(2);
    expect(horizontalPoints[1]).toBeCloseTo(blockedLeft);
    expect(horizontalPoints[0]).toBeLessThan(horizontalPoints[1]);
  });

  it("bounds rendered rows for a large portfolio while preserving total row semantics", async () => {
    const largeRows: PortfolioProjectionResult["rows"] = [
      {
        id: "alpha",
        projectId: "alpha",
        kind: "project",
        name: "Alpha",
        wbsNumber: "",
        depth: 0,
        position: 0,
        status: "open",
        incompleteEffort: false,
        incompleteSchedule: true,
        hasChildren: true,
        completed: false,
      },
      ...Array.from({ length: 200 }, (_, index) => ({
        id: `task-${index}`,
        projectId: "alpha",
        kind: "task" as const,
        name: `Task ${index}`,
        wbsNumber: String(index + 1),
        depth: 1,
        position: index + 1,
        incompleteEffort: true,
        incompleteSchedule: true,
        hasChildren: false,
        completed: false,
      })),
    ];
    const portfolioGateway = gateway();
    vi.mocked(portfolioGateway.projection).mockResolvedValue({
      projection: "execution",
      projects: [projects[0]],
      rows: largeRows,
      dependencies: [],
      holidays: [],
    });

    const { container } = render(
      <PortfolioHomePage
        gateway={portfolioGateway}
        wbsGateway={wbsGateway()}
        onOpenProject={vi.fn()}
        onProjectCommand={vi.fn()}
        onOpenWBS={vi.fn()}
      />,
    );

    await screen.findByRole("button", { name: "Task 0" });
    expect(
      container.querySelectorAll("[data-portfolio-row]").length,
    ).toBeLessThanOrEqual(80);
    expect(
      screen
        .getByRole("treegrid", { name: "Portfolio schedule rows" })
        .getAttribute("aria-rowcount"),
    ).toBe("201");
  });

  it.each([
    {
      action: "Move Up",
      direction: "up",
      adjacentIndex: 99,
      expectedWBS: "100",
    },
    {
      action: "Move Down",
      direction: "down",
      adjacentIndex: 101,
      expectedWBS: "102",
    },
  ] as const)(
    "IFD-HOME-FOCUS-05 waits for the reordered projection before focusing the row after $action",
    async ({ action, direction, adjacentIndex, expectedWBS }) => {
      const user = userEvent.setup();
      const projectRow: PortfolioProjectionResult["rows"][number] = {
        id: "alpha",
        projectId: "alpha",
        kind: "project",
        name: "Alpha",
        wbsNumber: "",
        depth: 0,
        position: 0,
        status: "open",
        incompleteEffort: false,
        incompleteSchedule: true,
        hasChildren: true,
        completed: false,
      };
      const taskRows: PortfolioProjectionResult["rows"] = Array.from(
        { length: 200 },
        (_, index) => ({
          id: `task-${index}`,
          projectId: "alpha",
          kind: "task" as const,
          name: `Task ${index}`,
          wbsNumber: String(index + 1),
          depth: 1,
          position: index + 1,
          incompleteEffort: true,
          incompleteSchedule: true,
          hasChildren: false,
          completed: false,
        }),
      );
      const largeRows = [projectRow, ...taskRows];
      const swappedTasks = [...taskRows];
      [swappedTasks[100], swappedTasks[adjacentIndex]] = [
        swappedTasks[adjacentIndex],
        swappedTasks[100],
      ];
      const reorderedRows = [
        projectRow,
        ...swappedTasks.map((row, index) => ({
          ...row,
          position: index + 1,
          wbsNumber: String(index + 1),
        })),
      ];
      const refreshedProjection = deferred<PortfolioProjectionResult>();
      const portfolioGateway = gateway();
      vi.mocked(portfolioGateway.projection)
        .mockResolvedValueOnce({
          projection: "execution",
          projects: [projects[0]],
          rows: largeRows.map((row) => ({ ...row })),
          dependencies: [],
          holidays: [],
        })
        .mockImplementation(() => refreshedProjection.promise);
      const structureGateway = wbsGateway();
      vi.mocked(structureGateway.reorder).mockImplementation(async () => {
        advanceScheduleProjectionVersion();
      });
      const { container } = render(
        <PortfolioHomePage
          gateway={portfolioGateway}
          wbsGateway={structureGateway}
          onOpenProject={vi.fn()}
          onProjectCommand={vi.fn()}
          onOpenWBS={vi.fn()}
        />,
      );

      await screen.findByRole("button", { name: "Task 0" });
      const timeline = screen.getByRole("region", {
        name: "Scrollable portfolio timeline",
      });
      Object.defineProperty(timeline, "clientHeight", {
        configurable: true,
        value: 320,
      });
      timeline.scrollTop = 4_000;
      fireEvent.scroll(timeline);

      const targetRow = await findPortfolioRow(container, "task-100");
      await user.click(
        within(targetRow).getByRole("button", {
          name: "More actions for Task 100",
        }),
      );
      await user.click(
        within(
          screen.getByRole("menu", { name: "Actions for Task 100" }),
        ).getByRole("menuitem", { name: action }),
      );

      await waitFor(() =>
        expect(structureGateway.reorder).toHaveBeenCalledWith(
          "alpha",
          "task-100",
          direction,
        ),
      );
      const oldTrigger = within(targetRow).getByRole("button", {
        name: "More actions for Task 100",
      });
      screen.getByRole("button", { name: "Configure Gantt" }).focus();
      expect(document.activeElement).not.toBe(oldTrigger);
      await waitFor(() =>
        expect(
          vi.mocked(portfolioGateway.projection).mock.calls.length,
        ).toBeGreaterThan(1),
      );

      await act(async () => {
        refreshedProjection.resolve({
          projection: "execution",
          projects: [projects[0]],
          rows: reorderedRows,
          dependencies: [],
          holidays: [],
        });
        await refreshedProjection.promise;
      });
      const refreshedRow = await findPortfolioRow(container, "task-100");
      const refreshedName = within(refreshedRow).getByRole("button", {
        name: "Task 100",
      });
      const treegrid = screen.getByRole("treegrid", {
        name: "Portfolio schedule rows",
      });

      await waitFor(() => {
        expect(timeline.scrollTop).toBe(4_000);
        expect(treegrid.parentElement?.scrollTop).toBe(4_000);
        expect(
          within(refreshedRow).getAllByRole("gridcell")[0].textContent,
        ).toBe(expectedWBS);
        expect(document.activeElement).toBe(refreshedName);
        expect(document.activeElement).not.toBe(oldTrigger);
        expect(document.activeElement?.closest("[data-portfolio-row]")).toBe(
          refreshedRow,
        );
      });
    },
  );

  it("IFD-HOME-FOCUS-05 restores a usable row focus without refreshing when reorder fails", async () => {
    const user = userEvent.setup();
    const portfolioGateway = gateway();
    const structureGateway = wbsGateway();
    vi.mocked(structureGateway.reorder).mockRejectedValue(
      new Error("The item could not be reordered. Try again."),
    );
    const { container } = render(
      <PortfolioHomePage
        gateway={portfolioGateway}
        wbsGateway={structureGateway}
        onOpenProject={vi.fn()}
        onProjectCommand={vi.fn()}
        onOpenWBS={vi.fn()}
      />,
    );

    const completedRow = await findPortfolioRow(container, "completed");
    await user.click(
      within(completedRow).getByRole("button", {
        name: "More actions for Completed",
      }),
    );
    await user.click(
      within(
        screen.getByRole("menu", { name: "Actions for Completed" }),
      ).getByRole("menuitem", { name: "Move Down" }),
    );

    expect((await screen.findByRole("status")).textContent).toContain(
      "The item could not be reordered. Try again.",
    );
    const retainedRow = await findPortfolioRow(container, "completed");
    const retainedName = within(retainedRow).getByRole("button", {
      name: "Completed",
    });
    await waitFor(() => expect(document.activeElement).toBe(retainedName));
    expect(within(retainedRow).getAllByRole("gridcell")[0].textContent).toBe(
      "1.1",
    );
    expect(portfolioGateway.projection).toHaveBeenCalledTimes(1);
  });

  it("IFD-HOME-FOCUS-02 focuses the newly added virtualized Task after its refreshed projection arrives", async () => {
    const user = userEvent.setup();
    const existingRows: PortfolioProjectionResult["rows"] = [
      {
        id: "alpha",
        projectId: "alpha",
        kind: "project",
        name: "Alpha",
        wbsNumber: "",
        depth: 0,
        position: 0,
        status: "open",
        incompleteEffort: false,
        incompleteSchedule: true,
        hasChildren: true,
        completed: false,
      },
      ...Array.from({ length: 200 }, (_, index) => ({
        id: `task-${index}`,
        projectId: "alpha",
        kind: "task" as const,
        name: `Task ${index}`,
        wbsNumber: String(index + 1),
        depth: 1,
        position: index + 1,
        incompleteEffort: true,
        incompleteSchedule: true,
        hasChildren: false,
        completed: false,
      })),
    ];
    const createdRow: PortfolioProjectionResult["rows"][number] = {
      id: "new-task",
      projectId: "alpha",
      kind: "task",
      name: "New Task",
      wbsNumber: "201",
      depth: 1,
      position: 201,
      incompleteEffort: true,
      incompleteSchedule: true,
      hasChildren: false,
      completed: false,
    };
    let includeCreatedRow = false;
    const portfolioGateway = gateway();
    vi.mocked(portfolioGateway.projection).mockImplementation(() =>
      Promise.resolve({
        projection: "execution",
        projects: [projects[0]],
        rows: [
          ...existingRows.map((row) => ({ ...row })),
          ...(includeCreatedRow ? [{ ...createdRow }] : []),
        ],
        dependencies: [],
        holidays: [],
      }),
    );
    const structureGateway = wbsGateway();
    const focusHandled = vi.fn();
    const sharedProps = {
      gateway: portfolioGateway,
      wbsGateway: structureGateway,
      onOpenProject: vi.fn(),
      onProjectCommand: vi.fn(),
      onOpenWBS: vi.fn(),
      onFocusRequestHandled: focusHandled,
    };
    const { container, rerender } = render(
      <PortfolioHomePage {...sharedProps} />,
    );

    await screen.findByRole("button", { name: "Task 0" });
    const timeline = screen.getByRole("region", {
      name: "Scrollable portfolio timeline",
    });
    Object.defineProperty(timeline, "clientHeight", {
      configurable: true,
      value: 320,
    });
    await user.click(screen.getByRole("button", { name: "Collapse Alpha" }));
    await waitFor(() =>
      expect(screen.queryByRole("button", { name: "Task 0" })).toBeNull(),
    );

    includeCreatedRow = true;
    rerender(
      <PortfolioHomePage
        {...sharedProps}
        focusRequest={{ key: 1, rowId: "new-task" }}
      />,
    );
    act(() => {
      advanceScheduleProjectionVersion();
    });

    const createdPortfolioRow = await findPortfolioRow(container, "new-task");
    const createdName = within(createdPortfolioRow).getByRole("button", {
      name: "New Task",
    });
    const treegrid = screen.getByRole("treegrid", {
      name: "Portfolio schedule rows",
    });
    await waitFor(() => {
      expect(document.activeElement).toBe(createdName);
      expect(timeline.scrollTop).toBeGreaterThan(0);
      expect(treegrid.parentElement?.scrollTop).toBe(timeline.scrollTop);
      expect(focusHandled).toHaveBeenCalledWith(1);
    });
  });

  it("IFD-HOME-FOCUS-03 resolves focus to the visible Project when a Role filter excludes the created Task", async () => {
    const user = userEvent.setup();
    const portfolioGateway = gateway();
    let includeCreatedRow = false;
    vi.mocked(portfolioGateway.projection).mockImplementation(
      (selected, value) => {
        const result = projection(selected, value);
        return Promise.resolve({
          ...result,
          rows: includeCreatedRow
            ? [
                ...result.rows,
                {
                  id: "filtered-new-task",
                  projectId: "alpha",
                  kind: "task" as const,
                  name: "New unassigned Task",
                  wbsNumber: "3",
                  depth: 1,
                  position: 3,
                  incompleteEffort: true,
                  incompleteSchedule: true,
                  hasChildren: false,
                  completed: false,
                },
              ]
            : result.rows,
        });
      },
    );
    const focusHandled = vi.fn();
    const sharedProps = {
      gateway: portfolioGateway,
      wbsGateway: wbsGateway(),
      onOpenProject: vi.fn(),
      onProjectCommand: vi.fn(),
      onOpenWBS: vi.fn(),
      onFocusRequestHandled: focusHandled,
    };
    const { container, rerender } = render(
      <PortfolioHomePage {...sharedProps} />,
    );
    await findPortfolioRow(container, "completed");

    await user.click(screen.getByRole("button", { name: "Configure Gantt" }));
    const filterDialog = await screen.findByRole("dialog", {
      name: "Gantt configuration",
    });
    const roleSection = within(filterDialog)
      .getByRole("heading", { name: "Roles" })
      .closest("section");
    if (!roleSection) throw new Error("Roles filter section not found");
    await user.click(
      within(roleSection).getByRole("button", { name: "Clear All" }),
    );
    await user.click(
      within(roleSection).getByRole("checkbox", { name: "Development" }),
    );
    await user.click(
      within(filterDialog).getByRole("button", { name: "Apply" }),
    );

    const projectionCallsBeforeCreate = vi.mocked(portfolioGateway.projection)
      .mock.calls.length;
    includeCreatedRow = true;
    rerender(
      <PortfolioHomePage
        {...sharedProps}
        focusRequest={{ key: 2, rowId: "filtered-new-task" }}
      />,
    );
    act(() => {
      advanceScheduleProjectionVersion();
    });

    await waitFor(() =>
      expect(
        vi.mocked(portfolioGateway.projection).mock.calls.length,
      ).toBeGreaterThan(projectionCallsBeforeCreate),
    );
    const alphaRow = await findPortfolioRow(container, "alpha");
    const alphaName = within(alphaRow).getByRole("button", { name: "Alpha" });
    await waitFor(() => {
      expect(
        container.querySelector('[data-portfolio-row="filtered-new-task"]'),
      ).toBeNull();
      expect(document.activeElement).toBe(alphaName);
      expect(focusHandled).toHaveBeenCalledWith(2);
    });
  });

  it("keeps timeline bars read-only and routes row actions through the canonical Home workflows", async () => {
    const user = userEvent.setup();
    const openProject = vi.fn();
    const openWBS = vi.fn();
    const projectCommand = vi.fn();
    const structureGateway = wbsGateway();
    const { container } = render(
      <PortfolioHomePage
        gateway={gateway()}
        wbsGateway={structureGateway}
        onOpenProject={openProject}
        onProjectCommand={projectCommand}
        onOpenWBS={openWBS}
      />,
    );

    const alphaRow = await findPortfolioRow(container, "alpha");
    expect(screen.queryByText("Actions", { selector: "span" })).toBeNull();
    expect(
      screen.queryByRole("separator", { name: "Resize Actions column" }),
    ).toBeNull();
    const alphaTimeline = screen.getByLabelText(
      "Alpha: 2026-08-03 through 2026-08-05",
    );
    expect(alphaTimeline.tagName).toBe("DIV");
    await user.click(alphaTimeline);
    expect(openProject).not.toHaveBeenCalled();
    await user.click(within(alphaRow).getByRole("button", { name: "Alpha" }));
    expect(openProject).toHaveBeenCalledWith("alpha");

    const alphaActions = within(alphaRow).getByRole("button", {
      name: "More actions for Alpha",
    });
    expect(alphaActions.className).toContain("absolute");
    expect(alphaActions.className).toContain("right-0");
    expect(
      within(alphaRow).getByRole("button", { name: "Add Child for Alpha" }),
    ).toBeTruthy();
    const alphaCells = within(alphaRow).getAllByRole("gridcell");
    const alphaNameCell = alphaCells[1];
    const alphaActionContainer = alphaNameCell.querySelector(
      '[data-row-actions="alpha"]',
    );
    expect(alphaActionContainer).toBeTruthy();
    const alphaNameStack = alphaNameCell.matches(
      '[data-row-name-stack="alpha"]',
    )
      ? alphaNameCell
      : null;
    const alphaCreateActions = alphaNameCell.querySelector(
      '[data-row-create-actions="alpha"]',
    );
    expect(alphaActionContainer?.parentElement).toBe(alphaNameStack);
    expect(alphaCreateActions?.parentElement).toBe(alphaNameStack);
    expect(alphaCreateActions?.className).toContain("absolute");
    expect(alphaCreateActions?.className).toContain("w-max");
    expect(alphaCreateActions?.className).not.toContain("overflow-hidden");
    expect(alphaNameStack?.className).toContain("overflow-visible");
    expect(alphaNameStack?.children[0]).toBe(
      alphaNameCell.querySelector('[data-row-name-line="alpha"]'),
    );
    expect(alphaNameStack?.children[1]).toBe(alphaCreateActions);
    expect(alphaActionContainer?.className).toContain("opacity-0");
    expect(within(alphaNameCell).getAllByRole("button")).toHaveLength(4);

    alphaActions.focus();
    await user.keyboard("{Enter}");
    const alphaMenu = screen.getByRole("menu", { name: "Actions for Alpha" });
    expect(document.activeElement).toBe(
      within(alphaMenu).getByRole("menuitem", { name: "Lock" }),
    );
    await user.keyboard("{Escape}");
    expect(
      screen.queryByRole("menu", { name: "Actions for Alpha" }),
    ).toBeNull();
    expect(document.activeElement).toBe(alphaActions);

    await user.keyboard("{Enter}");
    await user.click(
      within(screen.getByRole("menu", { name: "Actions for Alpha" })).getByRole(
        "menuitem",
        { name: "Lock" },
      ),
    );
    expect(projectCommand).toHaveBeenCalledWith("lock", "alpha");

    const completedRow = await findPortfolioRow(container, "completed");
    const addCompletedSibling = within(completedRow).getByRole("button", {
      name: "Add Sibling for Completed",
    });
    expect(
      within(completedRow).queryByRole("button", {
        name: "Add Child for Completed",
      }),
    ).toBeNull();
    await user.click(addCompletedSibling);
    expect(openWBS).toHaveBeenCalledWith({
      projectId: "alpha",
      createAfterId: "completed",
    });
    await user.click(
      within(completedRow).getByRole("button", {
        name: "More actions for Completed",
      }),
    );
    const completedActions = screen.getByRole("menu", {
      name: "Actions for Completed",
    });
    expect(
      within(completedActions).queryByRole("menuitem", { name: "Delete" }),
    ).toBeNull();
    await user.click(
      within(completedActions).getByRole("menuitem", { name: "Move to…" }),
    );
    expect(openWBS).toHaveBeenCalledWith({
      projectId: "alpha",
      moveNodeId: "completed",
    });

    const betaRow = await findPortfolioRow(container, "beta");
    expect(
      within(betaRow).queryByRole("button", { name: /Add Child|Add Sibling/ }),
    ).toBeNull();
    await user.click(
      within(betaRow).getByRole("button", { name: "More actions for Beta" }),
    );
    const betaActions = screen.getByRole("menu", {
      name: "Actions for Beta",
    });
    expect(
      within(betaActions).getByRole("menuitem", { name: "Reopen" }),
    ).toBeTruthy();
    expect(
      within(betaActions).getByRole("menuitem", { name: "Close" }),
    ).toBeTruthy();
  });

  it("executes reorder and confirmed Delete from Home and refreshes the projection", async () => {
    const user = userEvent.setup();
    const portfolioGateway = gateway();
    const structureGateway = wbsGateway();
    const { container } = render(
      <PortfolioHomePage
        gateway={portfolioGateway}
        wbsGateway={structureGateway}
        onOpenProject={vi.fn()}
        onProjectCommand={vi.fn()}
        onOpenWBS={vi.fn()}
      />,
    );

    let completedRow = await findPortfolioRow(container, "completed");
    await user.click(
      within(completedRow).getByRole("button", {
        name: "More actions for Completed",
      }),
    );
    await user.click(
      within(
        screen.getByRole("menu", { name: "Actions for Completed" }),
      ).getByRole("menuitem", { name: "Move Down" }),
    );
    await waitFor(() =>
      expect(structureGateway.reorder).toHaveBeenCalledWith(
        "alpha",
        "completed",
        "down",
      ),
    );
    await waitFor(() =>
      expect(portfolioGateway.projection).toHaveBeenCalledTimes(2),
    );

    const testingRow = await findPortfolioRow(container, "testing");
    await user.click(
      within(testingRow).getByRole("button", {
        name: "More actions for QA Pass",
      }),
    );
    await user.click(
      within(
        screen.getByRole("menu", { name: "Actions for QA Pass" }),
      ).getByRole("menuitem", { name: "Delete" }),
    );
    const confirmation = await screen.findByRole("alertdialog", {
      name: "Delete QA Pass?",
    });
    await user.click(
      within(confirmation).getByRole("button", { name: "Delete Task" }),
    );
    await waitFor(() =>
      expect(structureGateway.remove).toHaveBeenCalledWith("alpha", "testing"),
    );
    await waitFor(() =>
      expect(portfolioGateway.projection).toHaveBeenCalledTimes(3),
    );

    completedRow = await findPortfolioRow(container, "completed");
    expect(
      within(completedRow).getByRole("button", { name: "Completed" }),
    ).toBeTruthy();
  });

  it("keeps Delete confirmation open and preserves the projection when deletion fails", async () => {
    const user = userEvent.setup();
    const portfolioGateway = gateway();
    const structureGateway = wbsGateway();
    vi.mocked(structureGateway.remove).mockRejectedValue(
      new Error("Task could not be deleted because it changed elsewhere."),
    );
    const { container } = render(
      <PortfolioHomePage
        gateway={portfolioGateway}
        wbsGateway={structureGateway}
        onOpenProject={vi.fn()}
        onProjectCommand={vi.fn()}
        onOpenWBS={vi.fn()}
      />,
    );

    const testingRow = await findPortfolioRow(container, "testing");
    await user.click(
      within(testingRow).getByRole("button", {
        name: "More actions for QA Pass",
      }),
    );
    await user.click(
      within(
        screen.getByRole("menu", { name: "Actions for QA Pass" }),
      ).getByRole("menuitem", { name: "Delete" }),
    );
    const confirmation = await screen.findByRole("alertdialog", {
      name: "Delete QA Pass?",
    });
    await user.click(
      within(confirmation).getByRole("button", { name: "Delete Task" }),
    );

    expect((await screen.findByRole("status")).textContent).toContain(
      "Task could not be deleted because it changed elsewhere.",
    );
    expect(
      screen.getByRole("alertdialog", { name: "Delete QA Pass?" }),
    ).toBeTruthy();
    expect(portfolioGateway.projection).toHaveBeenCalledTimes(1);
  });

  it("keeps a Locked Task editable by Name entry while exposing no structural actions", async () => {
    const portfolioGateway = gateway();
    const lockedProjection = projection(["alpha", "beta"], "execution");
    lockedProjection.rows = [
      ...lockedProjection.rows.map((row) =>
        row.id === "beta" ? { ...row, hasChildren: true } : row,
      ),
      {
        id: "locked-task",
        projectId: "beta",
        kind: "task",
        name: "Locked Task",
        wbsNumber: "1",
        depth: 1,
        position: 1,
        effortMinutes: 120,
        start: "2026-08-06",
        end: "2026-08-07",
        incompleteEffort: false,
        incompleteSchedule: false,
        hasChildren: false,
        completed: false,
      },
    ];
    vi.mocked(portfolioGateway.projection).mockResolvedValue(lockedProjection);
    const openWBS = vi.fn();
    const { container } = render(
      <PortfolioHomePage
        gateway={portfolioGateway}
        wbsGateway={wbsGateway()}
        onOpenProject={vi.fn()}
        onProjectCommand={vi.fn()}
        onOpenWBS={openWBS}
      />,
    );

    const lockedTask = await findPortfolioRow(container, "locked-task");
    await userEvent.click(
      within(lockedTask).getByRole("button", { name: "Locked Task" }),
    );
    expect(openWBS).toHaveBeenCalledWith({
      projectId: "beta",
      nodeId: "locked-task",
    });
    expect(
      within(lockedTask).queryByRole("button", { name: /Add Child/ }),
    ).toBeNull();
    expect(
      within(lockedTask).queryByRole("button", { name: /More actions/ }),
    ).toBeNull();
  });

  it("disables adjacent reorder under a restrictive Role filter but keeps Move to available", async () => {
    const user = userEvent.setup();
    const openWBS = vi.fn();
    const { container } = render(
      <PortfolioHomePage
        gateway={gateway()}
        wbsGateway={wbsGateway()}
        onOpenProject={vi.fn()}
        onProjectCommand={vi.fn()}
        onOpenWBS={openWBS}
      />,
    );
    await findPortfolioRow(container, "completed");

    await user.click(screen.getByRole("button", { name: "Configure Gantt" }));
    const filterDialog = await screen.findByRole("dialog", {
      name: "Gantt configuration",
    });
    const roleSection = within(filterDialog)
      .getByRole("heading", { name: "Roles" })
      .closest("section");
    if (!roleSection) throw new Error("Roles section not found");
    await user.click(
      within(roleSection).getByRole("button", { name: "Clear All" }),
    );
    await user.click(
      within(roleSection).getByRole("checkbox", { name: "Development" }),
    );
    await user.click(
      within(filterDialog).getByRole("button", { name: "Apply" }),
    );

    const completedRow = await findPortfolioRow(container, "completed");
    const trigger = within(completedRow).getByRole("button", {
      name: "More actions for Completed",
    });
    expect(trigger.getAttribute("title")).toBe("More actions for Completed");
    await user.click(trigger);
    const actions = screen.getByRole("menu", {
      name: "Actions for Completed",
    });
    const moveUp = within(actions).getByRole("menuitem", { name: /Move Up/ });
    const moveDown = within(actions).getByRole("menuitem", {
      name: /Move Down/,
    });
    const moveTo = within(actions).getByRole("menuitem", { name: "Move to…" });
    expect(moveUp.getAttribute("aria-disabled")).toBe("true");
    expect(moveUp.getAttribute("title")).toBe(
      "Show all roles to reorder WBS items.",
    );
    expect(moveDown.getAttribute("aria-disabled")).toBe("true");
    expect(moveDown.getAttribute("title")).toBe(
      "Show all roles to reorder WBS items.",
    );
    expect(moveTo.getAttribute("aria-disabled")).toBe("false");
    await user.click(moveTo);
    expect(openWBS).toHaveBeenCalledWith({
      projectId: "alpha",
      moveNodeId: "completed",
    });

    await user.click(screen.getByRole("button", { name: "Configure Gantt" }));
    const restoreDialog = await screen.findByRole("dialog", {
      name: "Gantt configuration",
    });
    const restoreRoleSection = within(restoreDialog)
      .getByRole("heading", { name: "Roles" })
      .closest("section");
    if (!restoreRoleSection) throw new Error("Roles section not found");
    await user.click(
      within(restoreRoleSection).getByRole("button", { name: "Select All" }),
    );
    await user.click(
      within(restoreDialog).getByRole("button", { name: "Apply" }),
    );

    const restoredRow = await findPortfolioRow(container, "completed");
    await user.click(
      within(restoredRow).getByRole("button", {
        name: "More actions for Completed",
      }),
    );
    expect(
      within(screen.getByRole("menu", { name: "Actions for Completed" }))
        .getByRole("menuitem", { name: "Move Down" })
        .getAttribute("aria-disabled"),
    ).toBe("false");
  });
});

describe("US-7.1 projection preference and drag reorder deltas", () => {
  beforeEach(() => window.localStorage.clear());
  afterEach(() => vi.restoreAllMocks());

  it("collapses and expands the loaded hierarchy from one contextual Name-header action_DeltaD11D13", async () => {
    const user = userEvent.setup();
    const portfolioGateway = gateway();
    const structureGateway = wbsGateway();
    window.localStorage.setItem(
      portfolioCollapsedRowsPreferenceKey,
      JSON.stringify(["outside-project"]),
    );
    const firstRender = render(
      <PortfolioHomePage
        gateway={portfolioGateway}
        wbsGateway={structureGateway}
        onOpenProject={vi.fn()}
        onProjectCommand={vi.fn()}
        onOpenWBS={vi.fn()}
      />,
    );
    await findPortfolioRow(firstRender.container, "delivery");

    await user.click(screen.getByRole("button", { name: "Collapse all rows" }));

    await waitFor(() =>
      expect(
        firstRender.container.querySelector('[data-portfolio-row="delivery"]'),
      ).toBeNull(),
    );
    expect(
      screen.getByRole("button", { name: "Expand all rows" }),
    ).toBeTruthy();
    expect(structureGateway.create).not.toHaveBeenCalled();
    expect(structureGateway.createSibling).not.toHaveBeenCalled();
    expect(structureGateway.reorder).not.toHaveBeenCalled();
    expect(structureGateway.place).not.toHaveBeenCalled();
    expect(structureGateway.move).not.toHaveBeenCalled();
    expect(portfolioGateway.projection).toHaveBeenCalledTimes(1);
    await waitFor(() =>
      expect(
        JSON.parse(
          window.localStorage.getItem(portfolioCollapsedRowsPreferenceKey) ??
            "[]",
        ),
      ).toEqual(["alpha", "delivery", "outside-project"]),
    );

    firstRender.unmount();
    const secondRender = render(
      <PortfolioHomePage
        gateway={gateway()}
        wbsGateway={wbsGateway()}
        onOpenProject={vi.fn()}
        onProjectCommand={vi.fn()}
        onOpenWBS={vi.fn()}
      />,
    );
    const alphaRow = await findPortfolioRow(secondRender.container, "alpha");
    expect(alphaRow.getAttribute("aria-expanded")).toBe("false");
    expect(
      secondRender.container.querySelector('[data-portfolio-row="delivery"]'),
    ).toBeNull();

    await user.click(screen.getByRole("button", { name: "Expand all rows" }));
    await findPortfolioRow(secondRender.container, "completed");
    expect(
      screen.getByRole("button", { name: "Collapse all rows" }),
    ).toBeTruthy();
    await waitFor(() =>
      expect(
        window.localStorage.getItem(portfolioCollapsedRowsPreferenceKey),
      ).toBe(JSON.stringify(["outside-project"])),
    );
  });

  it("persists automatic ancestor expansion required by a confirmed row focus_DeltaD12", async () => {
    window.localStorage.setItem(
      portfolioCollapsedRowsPreferenceKey,
      JSON.stringify(["alpha", "delivery"]),
    );
    const handled = vi.fn();
    const { container } = render(
      <PortfolioHomePage
        gateway={gateway()}
        wbsGateway={wbsGateway()}
        onOpenProject={vi.fn()}
        onProjectCommand={vi.fn()}
        onOpenWBS={vi.fn()}
        focusRequest={{ key: 1, rowId: "completed" }}
        onFocusRequestHandled={handled}
      />,
    );

    await findPortfolioRow(container, "completed");
    await waitFor(() => expect(handled).toHaveBeenCalledWith(1));
    await waitFor(() =>
      expect(
        window.localStorage.getItem(portfolioCollapsedRowsPreferenceKey),
      ).toBe("[]"),
    );
  });

  it("restores an individual collapsed row after Home remount and expands safely from malformed storage_DeltaD12", async () => {
    const user = userEvent.setup();
    const firstRender = render(
      <PortfolioHomePage
        gateway={gateway()}
        wbsGateway={wbsGateway()}
        onOpenProject={vi.fn()}
        onProjectCommand={vi.fn()}
        onOpenWBS={vi.fn()}
      />,
    );
    const deliveryRow = await findPortfolioRow(
      firstRender.container,
      "delivery",
    );
    await user.click(
      within(deliveryRow).getByRole("button", { name: "Collapse Delivery" }),
    );
    await waitFor(() =>
      expect(
        firstRender.container.querySelector('[data-portfolio-row="completed"]'),
      ).toBeNull(),
    );
    await waitFor(() =>
      expect(
        window.localStorage.getItem(portfolioCollapsedRowsPreferenceKey),
      ).toBe(JSON.stringify(["delivery"])),
    );

    firstRender.unmount();
    const secondRender = render(
      <PortfolioHomePage
        gateway={gateway()}
        wbsGateway={wbsGateway()}
        onOpenProject={vi.fn()}
        onProjectCommand={vi.fn()}
        onOpenWBS={vi.fn()}
      />,
    );
    const restoredDelivery = await findPortfolioRow(
      secondRender.container,
      "delivery",
    );
    expect(restoredDelivery.getAttribute("aria-expanded")).toBe("false");
    expect(
      secondRender.container.querySelector('[data-portfolio-row="completed"]'),
    ).toBeNull();

    secondRender.unmount();
    window.localStorage.setItem(
      portfolioCollapsedRowsPreferenceKey,
      "not-json",
    );
    const thirdRender = render(
      <PortfolioHomePage
        gateway={gateway()}
        wbsGateway={wbsGateway()}
        onOpenProject={vi.fn()}
        onProjectCommand={vi.fn()}
        onOpenWBS={vi.fn()}
      />,
    );
    await findPortfolioRow(thirdRender.container, "completed");
  });

  it("keeps current-session hierarchy interaction usable when preference writes fail_DeltaD12", async () => {
    const user = userEvent.setup();
    const writeFailure = vi
      .spyOn(Storage.prototype, "setItem")
      .mockImplementation(() => {
        throw new Error("storage unavailable");
      });
    const { container } = render(
      <PortfolioHomePage
        gateway={gateway()}
        wbsGateway={wbsGateway()}
        onOpenProject={vi.fn()}
        onProjectCommand={vi.fn()}
        onOpenWBS={vi.fn()}
      />,
    );
    const deliveryRow = await findPortfolioRow(container, "delivery");

    await user.click(
      within(deliveryRow).getByRole("button", { name: "Collapse Delivery" }),
    );

    await waitFor(() =>
      expect(
        container.querySelector('[data-portfolio-row="completed"]'),
      ).toBeNull(),
    );
    expect(writeFailure).toHaveBeenCalledWith(
      portfolioCollapsedRowsPreferenceKey,
      JSON.stringify(["delivery"]),
    );
    expect(screen.queryByRole("alert")).toBeNull();
  });

  it("hides the bulk hierarchy action when the loaded hierarchy has no expandable row_DeltaD11", async () => {
    const noHierarchyGateway = gateway();
    noHierarchyGateway.activeProjects = vi
      .fn()
      .mockResolvedValue(projects.filter((project) => project.id === "beta"));
    noHierarchyGateway.projection = vi
      .fn()
      .mockResolvedValue(projection(["beta"], "execution"));

    const { container } = render(
      <PortfolioHomePage
        gateway={noHierarchyGateway}
        wbsGateway={wbsGateway()}
        onOpenProject={vi.fn()}
        onProjectCommand={vi.fn()}
        onOpenWBS={vi.fn()}
      />,
    );
    await findPortfolioRow(container, "beta");

    expect(screen.queryByRole("button", { name: /all rows/i })).toBeNull();
  });

  it("restores the latest resized Project Grid column widths after navigating away and back_DeltaD10", async () => {
    const firstRender = render(
      <PortfolioHomePage
        gateway={gateway()}
        wbsGateway={wbsGateway()}
        onOpenProject={vi.fn()}
        onProjectCommand={vi.fn()}
        onOpenWBS={vi.fn()}
      />,
    );
    await screen.findByRole("heading", { name: "All Active Projects" });

    const nameResize = await screen.findByRole("separator", {
      name: "Resize Name column",
    });
    expect(nameResize.getAttribute("aria-valuenow")).toBe("190");
    fireEvent.mouseDown(nameResize, { clientX: 100 });
    fireEvent.mouseMove(window, { clientX: 160 });
    fireEvent.mouseUp(window);

    await waitFor(() =>
      expect(nameResize.getAttribute("aria-valuenow")).toBe("250"),
    );
    await waitFor(() =>
      expect(
        JSON.parse(
          window.localStorage.getItem(portfolioColumnWidthsPreferenceKey) ??
            "{}",
        ),
      ).toMatchObject({ name: 250 }),
    );

    firstRender.unmount();
    render(
      <PortfolioHomePage
        gateway={gateway()}
        wbsGateway={wbsGateway()}
        onOpenProject={vi.fn()}
        onProjectCommand={vi.fn()}
        onOpenWBS={vi.fn()}
      />,
    );
    await screen.findByRole("heading", { name: "All Active Projects" });
    const restoredNameResize = await screen.findByRole("separator", {
      name: "Resize Name column",
    });
    expect(restoredNameResize.getAttribute("aria-valuenow")).toBe("250");
  });

  it("remembers only a successfully applied projection across a Home remount and ignores a cancelled draft_DeltaD01", async () => {
    const user = userEvent.setup();
    const firstGateway = gateway();
    const firstRender = render(
      <PortfolioHomePage
        gateway={firstGateway}
        wbsGateway={wbsGateway()}
        onOpenProject={vi.fn()}
        onProjectCommand={vi.fn()}
        onOpenWBS={vi.fn()}
      />,
    );
    await screen.findByRole("heading", { name: "All Active Projects" });

    await user.click(screen.getByRole("button", { name: "Configure Gantt" }));
    let dialog = await screen.findByRole("dialog", {
      name: "Gantt configuration",
    });
    await user.click(
      within(dialog).getByRole("switch", {
        name: "Commitment projection",
      }),
    );
    expect(window.localStorage.getItem("schedmind.home.projection.v1")).toBe(
      null,
    );
    await user.click(within(dialog).getByRole("button", { name: "Apply" }));
    await waitFor(() =>
      expect(firstGateway.projection).toHaveBeenCalledWith(
        ["alpha", "beta"],
        "commitment",
        expect.any(String),
        expect.any(String),
        expect.any(AbortSignal),
      ),
    );
    expect(window.localStorage.getItem("schedmind.home.projection.v1")).toBe(
      "commitment",
    );

    firstRender.unmount();
    const restoredGateway = gateway();
    render(
      <PortfolioHomePage
        gateway={restoredGateway}
        wbsGateway={wbsGateway()}
        onOpenProject={vi.fn()}
        onProjectCommand={vi.fn()}
        onOpenWBS={vi.fn()}
      />,
    );
    await waitFor(() =>
      expect(restoredGateway.projection).toHaveBeenCalledWith(
        ["alpha", "beta"],
        "commitment",
        expect.any(String),
        expect.any(String),
        expect.any(AbortSignal),
      ),
    );

    await user.click(screen.getByRole("button", { name: "Configure Gantt" }));
    dialog = await screen.findByRole("dialog", {
      name: "Gantt configuration",
    });
    await user.click(
      within(dialog).getByRole("switch", { name: "Execution projection" }),
    );
    await user.click(within(dialog).getByRole("button", { name: "Cancel" }));

    expect(window.localStorage.getItem("schedmind.home.projection.v1")).toBe(
      "commitment",
    );
    expect(restoredGateway.projection).not.toHaveBeenCalledWith(
      ["alpha", "beta"],
      "execution",
      expect.any(String),
      expect.any(String),
      expect.any(AbortSignal),
    );
  });

  it("applies the projection for the current session when browser preference writes fail_DeltaD01", async () => {
    const user = userEvent.setup();
    const portfolioGateway = gateway();
    const writeFailure = vi
      .spyOn(Storage.prototype, "setItem")
      .mockImplementation(() => {
        throw new Error("storage unavailable");
      });

    render(
      <PortfolioHomePage
        gateway={portfolioGateway}
        wbsGateway={wbsGateway()}
        onOpenProject={vi.fn()}
        onProjectCommand={vi.fn()}
        onOpenWBS={vi.fn()}
      />,
    );
    await screen.findByRole("heading", { name: "All Active Projects" });

    await user.click(screen.getByRole("button", { name: "Configure Gantt" }));
    const dialog = await screen.findByRole("dialog", {
      name: "Gantt configuration",
    });
    await user.click(
      within(dialog).getByRole("switch", {
        name: "Commitment projection",
      }),
    );
    await user.click(within(dialog).getByRole("button", { name: "Apply" }));

    await waitFor(() =>
      expect(portfolioGateway.projection).toHaveBeenCalledWith(
        ["alpha", "beta"],
        "commitment",
        expect.any(String),
        expect.any(String),
        expect.any(AbortSignal),
      ),
    );
    expect(writeFailure).toHaveBeenCalledWith(
      "schedmind.home.projection.v1",
      "commitment",
    );
    expect(screen.queryByRole("alert")).toBeNull();
  });

  it("starts drag only from the leading handle, places an exact sibling, keeps same-position drops as no-ops, and disables reorder under a restrictive Role filter_DeltaD06_D08_D09", async () => {
    const user = userEvent.setup();
    const structureGateway = wbsGateway();
    const { container } = render(
      <PortfolioHomePage
        gateway={gateway()}
        wbsGateway={structureGateway}
        onOpenProject={vi.fn()}
        onProjectCommand={vi.fn()}
        onOpenWBS={vi.fn()}
      />,
    );

    const completedRow = await findPortfolioRow(container, "completed");
    const testingRow = await findPortfolioRow(container, "testing");
    expect(completedRow.hasAttribute("draggable")).toBe(false);
    expect(
      within(await findPortfolioRow(container, "alpha")).queryByRole("button", {
        name: "Reorder Alpha",
      }),
    ).toBeNull();
    const completedHandle = within(completedRow).getByRole("button", {
      name: "Reorder Completed",
    });
    expect(completedHandle.getAttribute("draggable")).toBe("true");

    Object.defineProperty(testingRow, "getBoundingClientRect", {
      configurable: true,
      value: () => ({
        top: 0,
        bottom: 40,
        left: 0,
        right: 400,
        width: 400,
        height: 40,
        x: 0,
        y: 0,
        toJSON: () => ({}),
      }),
    });
    const dataTransfer = {
      effectAllowed: "none",
      dropEffect: "none",
      setData: vi.fn(),
      getData: vi.fn(),
    };
    fireEvent.dragStart(completedHandle, { dataTransfer });
    fireEvent.dragOver(testingRow, { clientY: 30, dataTransfer });
    fireEvent.drop(testingRow, { clientY: 30, dataTransfer });

    await waitFor(() =>
      expect(structureGateway.place).toHaveBeenCalledWith(
        "alpha",
        "completed",
        "testing",
        "after",
      ),
    );
    await waitFor(() =>
      expect(document.activeElement).toBe(
        screen.getByRole("button", { name: "Completed" }),
      ),
    );

    const testingHandle = within(testingRow).getByRole("button", {
      name: "Reorder QA Pass",
    });
    Object.defineProperty(completedRow, "getBoundingClientRect", {
      configurable: true,
      value: () => ({
        top: 0,
        bottom: 40,
        left: 0,
        right: 400,
        width: 400,
        height: 40,
        x: 0,
        y: 0,
        toJSON: () => ({}),
      }),
    });
    fireEvent.dragStart(testingHandle, { dataTransfer });
    fireEvent.dragOver(completedRow, { clientY: 30, dataTransfer });
    fireEvent.drop(completedRow, { clientY: 30, dataTransfer });
    expect(structureGateway.place).toHaveBeenCalledTimes(1);

    await user.click(screen.getByRole("button", { name: "Configure Gantt" }));
    let dialog = await screen.findByRole("dialog", {
      name: "Gantt configuration",
    });
    const projectSection = within(dialog)
      .getByRole("heading", { name: "Projects" })
      .closest("section");
    if (!projectSection) throw new Error("Projects filter section not found");
    await user.click(
      within(projectSection).getByRole("button", { name: "Clear All" }),
    );
    await user.click(
      within(projectSection).getByRole("checkbox", { name: "Alpha" }),
    );
    await user.click(within(dialog).getByRole("button", { name: "Apply" }));

    const projectFilteredRow = await findPortfolioRow(container, "completed");
    const projectFilteredHandle = within(projectFilteredRow).getByRole(
      "button",
      { name: "Reorder Completed" },
    );
    expect(projectFilteredHandle.hasAttribute("disabled")).toBe(false);
    expect(projectFilteredHandle.getAttribute("draggable")).toBe("true");

    await user.click(screen.getByRole("button", { name: "Configure Gantt" }));
    dialog = await screen.findByRole("dialog", {
      name: "Gantt configuration",
    });
    const roleSection = within(dialog)
      .getByRole("heading", { name: "Roles" })
      .closest("section");
    if (!roleSection) throw new Error("Roles filter section not found");
    await user.click(
      within(roleSection).getByRole("button", { name: "Clear All" }),
    );
    await user.click(
      within(roleSection).getByRole("checkbox", { name: "Development" }),
    );
    await user.click(within(dialog).getByRole("button", { name: "Apply" }));

    const filteredCompletedRow = await findPortfolioRow(container, "completed");
    const disabledHandle = within(filteredCompletedRow).getByRole("button", {
      name: "Reorder Completed",
    });
    expect(disabledHandle.hasAttribute("disabled")).toBe(true);
    expect(disabledHandle.getAttribute("title")).toBe(
      "Show all roles to reorder WBS items.",
    );
    expect(
      within(filteredCompletedRow).getByRole("button", {
        name: "Add Sibling for Completed",
      }),
    ).toBeTruthy();
    await user.click(
      within(filteredCompletedRow).getByRole("button", {
        name: "More actions for Completed",
      }),
    );
    const moveDown = within(
      screen.getByRole("menu", { name: "Actions for Completed" }),
    ).getByRole("menuitem", { name: /Move Down/ });
    expect(moveDown.getAttribute("aria-disabled")).toBe("true");
    expect(moveDown.getAttribute("title")).toBe(
      "Show all roles to reorder WBS items.",
    );
  });

  it("supports touch-pointer reorder from the dedicated handle without making the row draggable_DeltaD06_D09", async () => {
    const structureGateway = wbsGateway();
    const { container } = render(
      <PortfolioHomePage
        gateway={gateway()}
        wbsGateway={structureGateway}
        onOpenProject={vi.fn()}
        onProjectCommand={vi.fn()}
        onOpenWBS={vi.fn()}
      />,
    );

    const completedRow = await findPortfolioRow(container, "completed");
    const testingRow = await findPortfolioRow(container, "testing");
    const completedHandle = within(completedRow).getByRole("button", {
      name: "Reorder Completed",
    });
    Object.defineProperty(testingRow, "getBoundingClientRect", {
      configurable: true,
      value: () => ({
        top: 0,
        bottom: 40,
        left: 0,
        right: 400,
        width: 400,
        height: 40,
        x: 0,
        y: 0,
        toJSON: () => ({}),
      }),
    });
    const originalElementFromPoint = document.elementFromPoint;
    Object.defineProperty(document, "elementFromPoint", {
      configurable: true,
      value: vi.fn(() => testingRow),
    });

    try {
      fireEvent.pointerDown(completedHandle, {
        pointerId: 7,
        pointerType: "touch",
        clientX: 10,
        clientY: 10,
      });
      fireEvent.pointerMove(completedHandle, {
        pointerId: 7,
        pointerType: "touch",
        clientX: 12,
        clientY: 30,
      });
      fireEvent.pointerUp(completedHandle, {
        pointerId: 7,
        pointerType: "touch",
        clientX: 12,
        clientY: 30,
      });

      await waitFor(() =>
        expect(structureGateway.place).toHaveBeenCalledWith(
          "alpha",
          "completed",
          "testing",
          "after",
        ),
      );
      expect(completedRow.hasAttribute("draggable")).toBe(false);
    } finally {
      Object.defineProperty(document, "elementFromPoint", {
        configurable: true,
        value: originalElementFromPoint,
      });
    }
  });

  it("cancels drag with Escape and refreshes the confirmed projection after a rejected placement_DeltaD09", async () => {
    const portfolioGateway = gateway();
    const structureGateway = wbsGateway();
    vi.mocked(structureGateway.place).mockRejectedValue(
      new Error("The reorder became stale."),
    );
    const { container } = render(
      <PortfolioHomePage
        gateway={portfolioGateway}
        wbsGateway={structureGateway}
        onOpenProject={vi.fn()}
        onProjectCommand={vi.fn()}
        onOpenWBS={vi.fn()}
      />,
    );

    const completedRow = await findPortfolioRow(container, "completed");
    const testingRow = await findPortfolioRow(container, "testing");
    Object.defineProperty(testingRow, "getBoundingClientRect", {
      configurable: true,
      value: () => ({
        top: 0,
        bottom: 40,
        left: 0,
        right: 400,
        width: 400,
        height: 40,
        x: 0,
        y: 0,
        toJSON: () => ({}),
      }),
    });
    const dataTransfer = {
      effectAllowed: "none",
      dropEffect: "none",
      setData: vi.fn(),
      getData: vi.fn(),
    };
    const completedHandle = within(completedRow).getByRole("button", {
      name: "Reorder Completed",
    });

    fireEvent.dragStart(completedHandle, { dataTransfer });
    fireEvent.keyDown(window, { key: "Escape" });
    fireEvent.drop(testingRow, { clientY: 30, dataTransfer });
    expect(structureGateway.place).not.toHaveBeenCalled();

    fireEvent.dragStart(completedHandle, { dataTransfer });
    fireEvent.dragOver(testingRow, { clientY: 30, dataTransfer });
    fireEvent.drop(testingRow, { clientY: 30, dataTransfer });

    await waitFor(() =>
      expect(structureGateway.place).toHaveBeenCalledWith(
        "alpha",
        "completed",
        "testing",
        "after",
      ),
    );
    await waitFor(() =>
      expect(portfolioGateway.projection).toHaveBeenCalledTimes(2),
    );
    expect(await screen.findByText("The reorder became stale.")).toBeTruthy();
    const refreshedCompletedRow = await findPortfolioRow(
      container,
      "completed",
    );
    const refreshedTestingRow = await findPortfolioRow(container, "testing");
    expect(
      refreshedCompletedRow.compareDocumentPosition(refreshedTestingRow) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).not.toBe(0);
    await waitFor(() =>
      expect(document.activeElement).toBe(
        within(refreshedCompletedRow).getByRole("button", {
          name: "Completed",
        }),
      ),
    );
  });

  it("auto-scrolls only the vertical row viewport while dragging near its edge_DeltaD09", async () => {
    const portfolioGateway = gateway();
    const longRows: PortfolioProjectionResult["rows"] = [
      {
        id: "alpha",
        projectId: "alpha",
        kind: "project",
        name: "Alpha",
        wbsNumber: "",
        depth: 0,
        position: 0,
        status: "open",
        incompleteEffort: false,
        incompleteSchedule: false,
        hasChildren: true,
        completed: false,
      },
      ...Array.from({ length: 100 }, (_, index) => ({
        id: `long-task-${index}`,
        projectId: "alpha",
        kind: "task" as const,
        name: `Long Task ${index}`,
        wbsNumber: String(index + 1),
        depth: 1,
        position: index + 1,
        incompleteEffort: true,
        incompleteSchedule: true,
        hasChildren: false,
        completed: false,
      })),
    ];
    vi.mocked(portfolioGateway.projection).mockResolvedValue({
      projection: "execution",
      projects: [projects[0]],
      rows: longRows,
      dependencies: [],
      holidays: [],
    });
    const { container } = render(
      <PortfolioHomePage
        gateway={portfolioGateway}
        wbsGateway={wbsGateway()}
        onOpenProject={vi.fn()}
        onProjectCommand={vi.fn()}
        onOpenWBS={vi.fn()}
      />,
    );

    const sourceRow = await findPortfolioRow(container, "long-task-0");
    const targetRow = await findPortfolioRow(container, "long-task-1");
    const treegrid = screen.getByRole("treegrid", {
      name: "Portfolio schedule rows",
    });
    const rowViewport = treegrid.parentElement;
    if (!rowViewport) throw new Error("Row viewport not found");
    const timeline = screen.getByRole("region", {
      name: "Scrollable portfolio timeline",
    });
    Object.defineProperty(rowViewport, "getBoundingClientRect", {
      configurable: true,
      value: () => ({
        top: 0,
        bottom: 120,
        left: 0,
        right: 400,
        width: 400,
        height: 120,
        x: 0,
        y: 0,
        toJSON: () => ({}),
      }),
    });
    Object.defineProperty(timeline, "clientHeight", {
      configurable: true,
      value: 120,
    });
    Object.defineProperty(targetRow, "getBoundingClientRect", {
      configurable: true,
      value: () => ({
        top: 80,
        bottom: 120,
        left: 0,
        right: 400,
        width: 400,
        height: 40,
        x: 0,
        y: 80,
        toJSON: () => ({}),
      }),
    });
    timeline.scrollTop = 0;
    timeline.scrollLeft = 75;
    const dataTransfer = {
      effectAllowed: "none",
      dropEffect: "none",
      setData: vi.fn(),
      getData: vi.fn(),
    };

    fireEvent.dragStart(
      within(sourceRow).getByRole("button", { name: "Reorder Long Task 0" }),
      { dataTransfer },
    );
    const dragOver = createEvent.dragOver(targetRow, { dataTransfer });
    Object.defineProperty(dragOver, "clientY", {
      configurable: true,
      value: 119,
    });
    fireEvent(targetRow, dragOver);

    expect(timeline.scrollTop).toBe(40);
    expect(rowViewport.scrollTop).toBe(40);
    expect(timeline.scrollLeft).toBe(75);
  });
});

describe("US-7.1 expanded Group drag boundary", () => {
  beforeEach(() => window.localStorage.clear());

  it("draws an after-Group drop boundary after the complete visible subtree and submits the Group identity_DeltaD07", async () => {
    const portfolioGateway = gateway();
    vi.mocked(portfolioGateway.projection).mockResolvedValue({
      projection: "execution",
      projects: [projects[0]],
      rows: [
        {
          id: "alpha",
          projectId: "alpha",
          kind: "project",
          name: "Alpha",
          wbsNumber: "",
          depth: 0,
          position: 0,
          status: "open",
          incompleteEffort: false,
          incompleteSchedule: false,
          hasChildren: true,
          completed: false,
        },
        {
          id: "group-one",
          projectId: "alpha",
          kind: "group",
          name: "Group One",
          wbsNumber: "1",
          depth: 1,
          position: 1,
          incompleteEffort: false,
          incompleteSchedule: false,
          hasChildren: true,
          completed: false,
        },
        {
          id: "group-one-child",
          projectId: "alpha",
          parentId: "group-one",
          kind: "task",
          name: "Group One Child",
          wbsNumber: "1.1",
          depth: 2,
          position: 1,
          incompleteEffort: true,
          incompleteSchedule: true,
          hasChildren: false,
          completed: false,
        },
        {
          id: "group-two",
          projectId: "alpha",
          kind: "group",
          name: "Group Two",
          wbsNumber: "2",
          depth: 1,
          position: 2,
          incompleteEffort: false,
          incompleteSchedule: false,
          hasChildren: true,
          completed: false,
        },
        {
          id: "group-two-child",
          projectId: "alpha",
          parentId: "group-two",
          kind: "task",
          name: "Group Two Child",
          wbsNumber: "2.1",
          depth: 2,
          position: 1,
          incompleteEffort: true,
          incompleteSchedule: true,
          hasChildren: false,
          completed: false,
        },
      ],
      dependencies: [],
      holidays: [],
    });
    const structureGateway = wbsGateway();
    const { container } = render(
      <PortfolioHomePage
        gateway={portfolioGateway}
        wbsGateway={structureGateway}
        onOpenProject={vi.fn()}
        onProjectCommand={vi.fn()}
        onOpenWBS={vi.fn()}
      />,
    );

    const sourceRow = await findPortfolioRow(container, "group-one");
    const targetRow = await findPortfolioRow(container, "group-two");
    const targetChildRow = await findPortfolioRow(container, "group-two-child");
    Object.defineProperty(targetRow, "getBoundingClientRect", {
      configurable: true,
      value: () => ({
        top: 0,
        bottom: 40,
        left: 0,
        right: 400,
        width: 400,
        height: 40,
        x: 0,
        y: 0,
        toJSON: () => ({}),
      }),
    });
    const dataTransfer = {
      effectAllowed: "none",
      dropEffect: "none",
      setData: vi.fn(),
      getData: vi.fn(),
    };

    fireEvent.dragStart(
      within(sourceRow).getByRole("button", { name: "Reorder Group One" }),
      { dataTransfer },
    );
    fireEvent.dragOver(targetRow, { clientY: 30, dataTransfer });

    expect(targetRow.className).not.toContain("border-b-2");
    expect(targetChildRow.className).toContain("border-b-2");

    fireEvent.drop(targetRow, { clientY: 30, dataTransfer });
    await waitFor(() =>
      expect(structureGateway.place).toHaveBeenCalledWith(
        "alpha",
        "group-one",
        "group-two",
        "after",
      ),
    );
  });
});

describe("US-7.1 drag pending state", () => {
  beforeEach(() => window.localStorage.clear());

  it("keeps the source row visibly pending until authoritative placement completes_DeltaD09", async () => {
    const pending = deferred<void>();
    const structureGateway = wbsGateway();
    vi.mocked(structureGateway.place).mockReturnValue(pending.promise);
    const { container } = render(
      <PortfolioHomePage
        gateway={gateway()}
        wbsGateway={structureGateway}
        onOpenProject={vi.fn()}
        onProjectCommand={vi.fn()}
        onOpenWBS={vi.fn()}
      />,
    );

    const sourceRow = await findPortfolioRow(container, "completed");
    const targetRow = await findPortfolioRow(container, "testing");
    Object.defineProperty(targetRow, "getBoundingClientRect", {
      configurable: true,
      value: () => ({
        top: 0,
        bottom: 40,
        left: 0,
        right: 400,
        width: 400,
        height: 40,
        x: 0,
        y: 0,
        toJSON: () => ({}),
      }),
    });
    const dataTransfer = {
      effectAllowed: "none",
      dropEffect: "none",
      setData: vi.fn(),
      getData: vi.fn(),
    };

    fireEvent.dragStart(
      within(sourceRow).getByRole("button", { name: "Reorder Completed" }),
      { dataTransfer },
    );
    fireEvent.dragOver(targetRow, { clientY: 30, dataTransfer });
    fireEvent.drop(targetRow, { clientY: 30, dataTransfer });

    await waitFor(() =>
      expect(sourceRow.getAttribute("aria-busy")).toBe("true"),
    );
    expect(sourceRow.className).toContain("bg-brand-soft");

    pending.resolve(undefined);
    await waitFor(() =>
      expect(sourceRow.hasAttribute("aria-busy")).toBe(false),
    );
  });
});
