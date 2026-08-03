import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import axe from "axe-core";
import { describe, expect, it, vi } from "vitest";
import type { WBSGateway } from "../../wbs/application/wbsGateway";
import type { PortfolioGateway } from "../application/portfolioGateway";
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

describe("US-7.1 Home portfolio Gantt acceptance workflow", () => {
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
    const addTaskButton = within(alphaRow).getByRole("button", {
      name: "Add Task for Alpha",
    });
    expect(addTaskButton.textContent).toBe("");
    expect(addTaskButton.getAttribute("title")).toBe("Add Task to Alpha");

    const lockedProjectRow = await findPortfolioRow(container, "beta");
    expect(
      within(lockedProjectRow).getByRole("button", { name: "Beta" }),
    ).toBeTruthy();
    expect(within(lockedProjectRow).getByText("Locked")).toBeTruthy();

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
    const addChildButton = within(testingRow).getByRole("button", {
      name: "Add Child for QA Pass",
    });
    expect(addChildButton.textContent).toBe("");
    expect(addChildButton.getAttribute("title")).toBe(
      "Add Child under QA Pass",
    );
    expect(addChildButton.innerHTML).not.toBe(addTaskButton.innerHTML);

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
    expect(
      within(alphaRow).getByRole("button", { name: "Add Task for Alpha" }),
    ).toBeTruthy();
    const alphaCells = within(alphaRow).getAllByRole("gridcell");
    const alphaNameCell = alphaCells[1];
    const alphaActionContainer = alphaNameCell.querySelector(
      '[data-row-actions="alpha"]',
    );
    expect(alphaActionContainer).toBeTruthy();
    expect(alphaActionContainer?.parentElement).toBe(alphaNameCell);
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
    expect(
      within(completedRow).queryByRole("button", {
        name: "Add Child for Completed",
      }),
    ).toBeNull();
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
      within(betaRow).queryByRole("button", { name: /Add Task/ }),
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
