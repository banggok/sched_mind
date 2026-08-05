import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { PortfolioGateway } from "../features/portfolio/application/portfolioGateway";
import type {
  PortfolioProject,
  PortfolioProjectionResult,
  PortfolioRow,
} from "../features/portfolio/domain/portfolio";
import type { ProjectsGateway } from "../features/projects/application/projectsGateway";
import type { Project } from "../features/projects/domain/project";
import type { WBSGateway } from "../features/wbs/application/wbsGateway";
import type { WBSNode } from "../features/wbs/domain/wbs";

const mockedGateways = vi.hoisted(() => {
  vi.stubEnv("VITE_API_BASE_URL", "/api");
  const portfolio = {
    activeProjects: vi.fn<PortfolioGateway["activeProjects"]>(),
    projection: vi.fn<PortfolioGateway["projection"]>(),
    savedFilters: vi.fn<PortfolioGateway["savedFilters"]>(),
    createSavedFilter: vi.fn<PortfolioGateway["createSavedFilter"]>(),
    updateSavedFilter: vi.fn<PortfolioGateway["updateSavedFilter"]>(),
    deleteSavedFilter: vi.fn<PortfolioGateway["deleteSavedFilter"]>(),
  } satisfies PortfolioGateway;
  const projects = {
    list: vi.fn<ProjectsGateway["list"]>(),
    get: vi.fn<ProjectsGateway["get"]>(),
    create: vi.fn<ProjectsGateway["create"]>(),
    rename: vi.fn<ProjectsGateway["rename"]>(),
    update: vi.fn<ProjectsGateway["update"]>(),
    changeStatus: vi.fn<ProjectsGateway["changeStatus"]>(),
    bulkReopen: vi.fn<ProjectsGateway["bulkReopen"]>(),
    movePriority: vi.fn<ProjectsGateway["movePriority"]>(),
    updateSettings: vi.fn<ProjectsGateway["updateSettings"]>(),
    delete: vi.fn<ProjectsGateway["delete"]>(),
  } satisfies ProjectsGateway;
  const wbs = {
    tree: vi.fn<WBSGateway["tree"]>(),
    allocations: vi.fn<WBSGateway["allocations"]>(),
    create: vi.fn<WBSGateway["create"]>(),
    rename: vi.fn<WBSGateway["rename"]>(),
    reorder: vi.fn<WBSGateway["reorder"]>(),
    move: vi.fn<WBSGateway["move"]>(),
    remove: vi.fn<WBSGateway["remove"]>(),
    updateExecutable: vi.fn<WBSGateway["updateExecutable"]>(),
    previewExecutableSchedule: vi.fn<WBSGateway["previewExecutableSchedule"]>(),
    complete: vi.fn<WBSGateway["complete"]>(),
    reopen: vi.fn<WBSGateway["reopen"]>(),
  } satisfies WBSGateway;

  return {
    portfolio,
    projects,
    wbs,
    createPortfolio: vi.fn(() => portfolio),
    createProjects: vi.fn(() => projects),
    createWBS: vi.fn(() => wbs),
  };
});

vi.mock("../features/portfolio/infrastructure/httpPortfolioGateway", () => ({
  createHTTPPortfolioGateway: mockedGateways.createPortfolio,
}));
vi.mock("../features/projects/infrastructure/httpProjectsGateway", () => ({
  createHTTPProjectsGateway: mockedGateways.createProjects,
}));
vi.mock("../features/wbs/infrastructure/httpWBSGateway", () => ({
  createHTTPWBSGateway: mockedGateways.createWBS,
}));

import { advanceScheduleProjectionVersion } from "../shared/infrastructure/scheduleProjectionClock";
import { App } from "./App";

const alphaPortfolioProject: PortfolioProject = {
  id: "alpha",
  name: "Alpha",
  status: "open",
  priority: 1,
  scheduleVersion: 1,
};

const alphaProject: Project = {
  id: "alpha",
  name: "Alpha",
  status: "open",
  autoCalculateDate: true,
  automaticScheduling: true,
  schedulingStartDate: "2026-08-03",
  projectBuffer: 20,
  scheduleVersion: 1,
  priority: 1,
  createdAt: new Date("2026-08-01T00:00:00Z"),
  updatedAt: new Date("2026-08-01T00:00:00Z"),
};

const alphaRow: PortfolioRow = {
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

const createdNode: WBSNode = {
  id: "new-task",
  projectId: "alpha",
  name: "New Task",
  position: 201,
  hasChildren: false,
  executable: {
    lagDays: 0,
    executionTimeline: {},
    commitmentTimeline: {},
  },
  children: [],
};

function taskRow(index: number): PortfolioRow {
  return {
    id: `task-${index}`,
    projectId: "alpha",
    kind: "task",
    name: `Task ${index}`,
    wbsNumber: String(index + 1),
    depth: 1,
    position: index + 1,
    incompleteEffort: true,
    incompleteSchedule: true,
    hasChildren: false,
    completed: false,
  };
}

function newTaskRow(id = "new-task", name = "New Task"): PortfolioRow {
  return {
    id,
    projectId: "alpha",
    kind: "task",
    name,
    wbsNumber: "201",
    depth: 1,
    position: 201,
    incompleteEffort: true,
    incompleteSchedule: true,
    hasChildren: false,
    completed: false,
  };
}

function groupRow(name = "Phase 100"): PortfolioRow {
  return {
    ...taskRow(100),
    id: "group-100",
    kind: "group",
    name,
    hasChildren: true,
  };
}

function groupNode(name = "Phase 100"): WBSNode {
  return {
    id: "group-100",
    projectId: "alpha",
    name,
    position: 101,
    hasChildren: true,
    executable: {
      lagDays: 0,
      executionTimeline: {},
      commitmentTimeline: {},
    },
    children: [
      {
        ...createdNode,
        id: "group-child",
        parentId: "group-100",
        name: "Group child",
        position: 1,
      },
    ],
  };
}

function portfolioResult(rows: PortfolioRow[]): PortfolioProjectionResult {
  return {
    projection: "execution",
    projects: [alphaPortfolioProject],
    rows,
    dependencies: [],
    holidays: [],
  };
}

function deferred<Value>() {
  let resolve!: (value: Value | PromiseLike<Value>) => void;
  const promise = new Promise<Value>((next) => {
    resolve = next;
  });
  return { promise, resolve };
}

function configureCommonGateways() {
  mockedGateways.portfolio.activeProjects.mockReset();
  mockedGateways.portfolio.projection.mockReset();
  mockedGateways.portfolio.savedFilters.mockReset();
  mockedGateways.projects.list.mockReset();
  mockedGateways.projects.get.mockReset();
  mockedGateways.projects.update.mockReset();
  mockedGateways.wbs.tree.mockReset();
  mockedGateways.wbs.create.mockReset();
  mockedGateways.wbs.rename.mockReset();

  mockedGateways.portfolio.activeProjects.mockResolvedValue([
    alphaPortfolioProject,
  ]);
  mockedGateways.portfolio.savedFilters.mockResolvedValue([]);
  mockedGateways.projects.get.mockResolvedValue(alphaProject);
  mockedGateways.projects.update.mockResolvedValue(alphaProject);
  mockedGateways.projects.list.mockResolvedValue({
    items: [],
    page: 1,
    pageSize: 5,
    total: 0,
  });
  mockedGateways.wbs.tree.mockResolvedValue([]);
  mockedGateways.wbs.rename.mockResolvedValue(undefined);
}

async function openAndSubmitAddTask() {
  const user = userEvent.setup();
  const originalTrigger = await screen.findByRole("button", {
    name: "Add Task for Alpha",
  });
  await user.click(originalTrigger);
  const dialog = await screen.findByRole("dialog", { name: "Add Task" });
  await user.type(within(dialog).getByLabelText("Name"), "New Task");
  await user.click(within(dialog).getByRole("button", { name: "Save" }));
  return { originalTrigger, user };
}

describe("App Home row focus restoration", () => {
  beforeEach(() => {
    window.history.replaceState(null, "", "/");
    vi.spyOn(window, "scrollTo").mockImplementation(() => undefined);
    configureCommonGateways();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("IFD-HOME-FOCUS-02 focuses the newly created Task after its deferred virtualized projection arrives", async () => {
    const existingRows = [
      alphaRow,
      ...Array.from({ length: 200 }, (_, index) => taskRow(index)),
    ];
    const refreshedProjection = deferred<PortfolioProjectionResult>();
    mockedGateways.portfolio.projection
      .mockResolvedValueOnce(portfolioResult(existingRows))
      .mockImplementation(() => refreshedProjection.promise);
    mockedGateways.wbs.create.mockImplementation(async () => {
      advanceScheduleProjectionVersion();
      return createdNode;
    });

    render(<App />);
    const timeline = await screen.findByRole("region", {
      name: "Scrollable portfolio timeline",
    });
    Object.defineProperty(timeline, "clientHeight", {
      configurable: true,
      value: 320,
    });
    const { originalTrigger } = await openAndSubmitAddTask();

    await waitFor(() =>
      expect(mockedGateways.wbs.create).toHaveBeenCalledWith(
        "alpha",
        undefined,
        "New Task",
        false,
      ),
    );
    await waitFor(() =>
      expect(screen.queryByRole("dialog", { name: "Add Task" })).toBeNull(),
    );
    await waitFor(() =>
      expect(
        mockedGateways.portfolio.projection.mock.calls.length,
      ).toBeGreaterThan(1),
    );

    await act(async () => {
      refreshedProjection.resolve(
        portfolioResult([...existingRows, newTaskRow()]),
      );
    });

    const newTaskName = await screen.findByRole("button", {
      name: "New Task",
    });
    const treegrid = screen.getByRole("treegrid", {
      name: "Portfolio schedule rows",
    });
    await waitFor(() => {
      expect(document.activeElement).toBe(newTaskName);
      expect(document.activeElement).not.toBe(originalTrigger);
      expect(timeline.scrollTop).toBeGreaterThan(0);
      expect(treegrid.parentElement?.scrollTop).toBe(timeline.scrollTop);
    });
  });

  it("IFD-HOME-FOCUS-06 waits for the confirmed Group edit projection before focusing the retained row", async () => {
    const initialRows = [
      alphaRow,
      ...Array.from({ length: 100 }, (_, index) => taskRow(index)),
      groupRow(),
      ...Array.from({ length: 99 }, (_, index) => taskRow(index + 101)),
    ];
    const refreshedProjection = deferred<PortfolioProjectionResult>();
    mockedGateways.portfolio.projection
      .mockResolvedValueOnce(portfolioResult(initialRows))
      .mockImplementation(() => refreshedProjection.promise);
    mockedGateways.wbs.tree.mockResolvedValue([groupNode()]);
    mockedGateways.wbs.rename.mockImplementation(async () => {
      advanceScheduleProjectionVersion();
    });

    const user = userEvent.setup();
    const { container } = render(<App />);
    const timeline = await screen.findByRole("region", {
      name: "Scrollable portfolio timeline",
    });
    Object.defineProperty(timeline, "clientHeight", {
      configurable: true,
      value: 320,
    });
    timeline.scrollTop = 4_000;
    fireEvent.scroll(timeline);
    const group = await waitFor(() => {
      const row = container.querySelector<HTMLElement>(
        '[data-portfolio-row="group-100"]',
      );
      expect(row).not.toBeNull();
      if (!row) throw new Error("Group row was not rendered");
      return row;
    });
    await user.click(within(group).getByRole("button", { name: "Phase 100" }));

    const dialog = await screen.findByRole("dialog", { name: "Phase 100" });
    const name = within(dialog).getByLabelText("Name");
    await user.clear(name);
    await user.type(name, "Renamed Phase");
    await user.click(within(dialog).getByRole("button", { name: "Save" }));
    await waitFor(() =>
      expect(mockedGateways.wbs.rename).toHaveBeenCalledWith(
        "alpha",
        "group-100",
        "Renamed Phase",
      ),
    );
    await waitFor(() =>
      expect(screen.queryByRole("dialog", { name: "Phase 100" })).toBeNull(),
    );
    const temporaryFocus = screen.getByRole("button", {
      name: "Configure Gantt",
    });
    temporaryFocus.focus();
    await waitFor(() =>
      expect(
        mockedGateways.portfolio.projection.mock.calls.length,
      ).toBeGreaterThan(1),
    );
    expect(document.activeElement).toBe(temporaryFocus);

    await act(async () => {
      refreshedProjection.resolve(
        portfolioResult(
          initialRows.map((row) =>
            row.id === "group-100" ? { ...row, name: "Renamed Phase" } : row,
          ),
        ),
      );
      await refreshedProjection.promise;
    });

    const renamedRow = await waitFor(() => {
      const row = container.querySelector<HTMLElement>(
        '[data-portfolio-row="group-100"]',
      );
      expect(row).not.toBeNull();
      if (!row) throw new Error("Renamed Group row was not rendered");
      return row;
    });
    const renamedName = within(renamedRow).getByRole("button", {
      name: "Renamed Phase",
    });
    const treegrid = screen.getByRole("treegrid", {
      name: "Portfolio schedule rows",
    });
    await waitFor(() => {
      expect(document.activeElement).toBe(renamedName);
      expect(timeline.scrollTop).toBeGreaterThan(0);
      expect(treegrid.parentElement?.scrollTop).toBe(timeline.scrollTop);
    });
  });

  it("IFD-HOME-FOCUS-06 focuses the retained Project identity after its confirmed edit projection", async () => {
    const refreshedProjection = deferred<PortfolioProjectionResult>();
    mockedGateways.portfolio.projection
      .mockResolvedValueOnce(portfolioResult([alphaRow]))
      .mockImplementation(() => refreshedProjection.promise);
    mockedGateways.projects.update.mockImplementation(async () => {
      advanceScheduleProjectionVersion();
      return { ...alphaProject, name: "Renamed Alpha" };
    });

    const user = userEvent.setup();
    const { container } = render(<App />);
    const alphaName = await screen.findByRole("button", { name: "Alpha" });
    await user.click(alphaName);
    const dialog = await screen.findByRole("dialog", { name: "Edit Project" });
    const name = within(dialog).getByLabelText("Name");
    await user.clear(name);
    await user.type(name, "Renamed Alpha");
    await user.click(within(dialog).getByRole("button", { name: "Save" }));

    await waitFor(() =>
      expect(mockedGateways.projects.update).toHaveBeenCalledWith(
        "alpha",
        "Renamed Alpha",
        true,
        "2026-08-03",
        20,
      ),
    );
    await waitFor(() =>
      expect(screen.queryByRole("dialog", { name: "Edit Project" })).toBeNull(),
    );
    const temporaryFocus = screen.getByRole("button", {
      name: "Configure Gantt",
    });
    temporaryFocus.focus();
    await waitFor(() =>
      expect(
        mockedGateways.portfolio.projection.mock.calls.length,
      ).toBeGreaterThan(1),
    );
    expect(document.activeElement).toBe(temporaryFocus);

    await act(async () => {
      refreshedProjection.resolve({
        ...portfolioResult([{ ...alphaRow, name: "Renamed Alpha" }]),
        projects: [{ ...alphaPortfolioProject, name: "Renamed Alpha" }],
      });
      await refreshedProjection.promise;
    });

    const renamedRow = await waitFor(() => {
      const row = container.querySelector<HTMLElement>(
        '[data-portfolio-row="alpha"]',
      );
      if (!row) throw new Error("Renamed Project row was not rendered");
      return row;
    });
    const renamedName = within(renamedRow).getByRole("button", {
      name: "Renamed Alpha",
    });
    await waitFor(() => expect(document.activeElement).toBe(renamedName));
  });

  it("IFD-HOME-FOCUS-04 ignores an Add Task completion after navigation supersedes its overlay", async () => {
    const staleCreate = deferred<WBSNode>();
    mockedGateways.portfolio.projection
      .mockResolvedValueOnce(portfolioResult([alphaRow]))
      .mockResolvedValue(
        portfolioResult([alphaRow, newTaskRow("stale-task", "Stale Task")]),
      );
    mockedGateways.wbs.create.mockImplementation(() =>
      staleCreate.promise.then((node) => {
        advanceScheduleProjectionVersion();
        return node;
      }),
    );

    render(<App />);
    const { user } = await openAndSubmitAddTask();
    await waitFor(() =>
      expect(mockedGateways.wbs.create).toHaveBeenCalledTimes(1),
    );

    await user.click(screen.getByRole("link", { name: "Projects" }));
    await screen.findByText("No projects yet");
    const projectsLink = screen.getByRole("link", { name: "Projects" });
    expect(document.activeElement).toBe(projectsLink);

    await act(async () => {
      staleCreate.resolve({
        ...createdNode,
        id: "stale-task",
        name: "Stale Task",
      });
      await Promise.resolve();
    });
    expect(document.activeElement).toBe(projectsLink);

    const homeLink = screen.getByRole("link", { name: "Home" });
    await user.click(homeLink);
    const staleTaskName = await screen.findByRole("button", {
      name: "Stale Task",
    });
    await act(async () => {
      await Promise.resolve();
    });

    expect(document.activeElement).toBe(homeLink);
    expect(document.activeElement).not.toBe(staleTaskName);
  });
});
