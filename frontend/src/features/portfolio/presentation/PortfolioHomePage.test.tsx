import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import axe from "axe-core";
import { describe, expect, it, vi } from "vitest";
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
            source: "manual",
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
    const openProject = vi.fn();
    const openWBS = vi.fn();
    const { container } = render(
      <PortfolioHomePage
        gateway={portfolioGateway}
        onOpenProject={openProject}
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
    expect(screen.getByLabelText("2026-08-03")).toBeTruthy();
    expect(screen.queryByLabelText("2026-07-27")).toBeNull();
    expect(
      within(alphaRow).getByRole("button", { name: "Alpha" }),
    ).toBeTruthy();
    const addTaskButton = within(alphaRow).getByRole("button", {
      name: "Add Task",
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
    expect(
      within(completedRow).queryByRole("button", { name: "Add Child" }),
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
      name: "Add Child",
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
        onOpenProject={vi.fn()}
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
});
