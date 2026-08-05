import { act, render, screen, waitFor, within } from "@testing-library/react";
import type { ComponentProps } from "react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { DependenciesGateway } from "../../dependencies/application/dependenciesGateway";
import type { Project } from "../../projects/domain/project";
import type { RolesGateway } from "../../roles/application/rolesGateway";
import type { TeamMembersGateway } from "../../team-members/application/teamMembersGateway";
import type { WBSGateway } from "../../wbs/application/wbsGateway";
import type { WBSNode } from "../../wbs/domain/wbs";
import type { SprintsGateway } from "../application/sprintsGateway";
import type {
  SprintDetail,
  SprintMemberProjection,
  SprintSuggestion,
  SprintTaskProjection,
} from "../domain/sprint";
import { SprintTaskReview } from "./SprintTaskReview";
import { deriveReviewProjection } from "./sprintTaskReviewProjection";

type SprintTaskEditorDependencies = NonNullable<
  ComponentProps<typeof SprintTaskReview>["taskEditorDependencies"]
>;

afterEach(() => {
  window.location.hash = "";
});

function member(
  id: string,
  name: string,
  capacity: Array<{ date: string; minutes: number }>,
): SprintMemberProjection {
  return {
    id,
    name,
    roleName: "Engineer",
    dailyCapacity: capacity,
    dailySummaries: [],
    capacityMinutes: 0,
    inSprintAllocationMinutes: 0,
    remainingMinutes: 0,
    overcapacityMinutes: 0,
    totalAllocationMinutes: 0,
  };
}

function task(
  id: string,
  overrides: Partial<SprintTaskProjection> = {},
): SprintTaskProjection {
  return {
    id,
    projectId: "project-alpha",
    projectName: "Alpha",
    projectStatus: "open",
    projectPriority: 1,
    name: id,
    wbsOrder: "1",
    wbsPath: "1",
    wbsRank: 1,
    assigneeId: "member-1",
    assigneeName: "Harry",
    executionStart: "2026-08-04",
    executionEnd: "2026-08-04",
    dailyPlanOrderDate: "2026-08-04",
    completed: false,
    allocations: [{ date: "2026-08-04", minutes: 60 }],
    inSprintAllocationMinutes: 60,
    outsideAllocationMinutes: 0,
    totalAllocationMinutes: 60,
    warnings: [],
    ...overrides,
  };
}

function detail(
  members: SprintMemberProjection[],
  tasks: SprintTaskProjection[],
  startDate = "2026-08-04",
  endDate = "2026-08-08",
): SprintDetail {
  return {
    sprint: {
      id: "sprint-1",
      name: "August",
      startDate,
      endDate,
      status: "planned",
      version: 1,
    },
    members,
    tasks,
    totals: {
      capacityMinutes: 0,
      selectedMemberAllocationMinutes: 0,
      remainingMinutes: 0,
      overcapacityMinutes: 0,
      needsReviewAllocationMinutes: 0,
      needsReviewDailyAllocation: [],
      allTaskInSprintMinutes: 0,
      allTaskTotalMinutes: 0,
      dailySummaries: [],
    },
    projectionToken: "projection-1",
  };
}

function gateway(
  candidates: SprintTaskProjection[] = [],
): Pick<SprintsGateway, "suggest" | "candidates" | "update"> {
  return {
    suggest: vi.fn(),
    candidates: vi.fn().mockResolvedValue({
      items: candidates,
      page: 1,
      pageSize: 20,
      total: candidates.length,
    }),
    update: vi.fn().mockResolvedValue(undefined),
  };
}

function taskEditorHarness(taskName = "API") {
  const project: Project = {
    id: "project-alpha",
    name: "Alpha",
    status: "open",
    autoCalculateDate: false,
    automaticScheduling: false,
    projectBuffer: 0,
    scheduleVersion: 1,
    priority: 1,
    createdAt: new Date("2026-08-01T00:00:00Z"),
    updatedAt: new Date("2026-08-01T00:00:00Z"),
  };
  const node: WBSNode = {
    id: taskName,
    projectId: project.id,
    name: taskName,
    position: 1,
    hasChildren: false,
    executable: {
      roleId: "role-engineer",
      assigneeId: "member-1",
      lagDays: 0,
      executionTimeline: {
        start: "2026-08-04",
        end: "2026-08-04",
      },
      commitmentTimeline: {
        start: "2026-08-04",
        end: "2026-08-04",
      },
    },
    children: [],
  };
  const wbsGateway = {
    tree: vi.fn().mockResolvedValue([node]),
    allocations: vi
      .fn()
      .mockResolvedValue({ execution: [], commitment: [], actual: [] }),
    create: vi.fn(),
    rename: vi.fn(),
    reorder: vi.fn(),
    move: vi.fn(),
    remove: vi.fn(),
    updateExecutable: vi.fn().mockResolvedValue(undefined),
    previewExecutableSchedule: vi.fn(),
    complete: vi.fn(),
    reopen: vi.fn(),
  } satisfies WBSGateway;
  const dependenciesGateway = {
    list: vi.fn().mockResolvedValue({ blockedBy: [], blocks: [] }),
    candidates: vi.fn().mockResolvedValue({
      items: [],
      page: 1,
      pageSize: 5,
      totalItems: 0,
    }),
    create: vi.fn(),
    remove: vi.fn(),
    invalidateTask: vi.fn(),
    invalidateAll: vi.fn(),
  } satisfies DependenciesGateway;
  const rolesGateway = {
    list: vi.fn().mockResolvedValue({
      items: [
        {
          id: "role-engineer",
          name: "Engineer",
          createdAt: new Date("2026-08-01T00:00:00Z"),
          updatedAt: new Date("2026-08-01T00:00:00Z"),
        },
      ],
      page: 1,
      pageSize: 100,
      total: 1,
    }),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  } satisfies RolesGateway;
  const membersGateway = {
    list: vi.fn().mockResolvedValue({
      items: [
        {
          id: "member-1",
          name: "Harry",
          role: { id: "role-engineer", name: "Engineer" },
          dailyCapacity: 8,
          bufferPercentage: 0,
          baseExecutionCapacity: 8,
          createdAt: new Date("2026-08-01T00:00:00Z"),
          updatedAt: new Date("2026-08-01T00:00:00Z"),
        },
      ],
      page: 1,
      pageSize: 100,
      total: 1,
    }),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  } satisfies TeamMembersGateway;
  const projectsGateway = {
    get: vi.fn().mockResolvedValue(project),
  };
  const dependencies: SprintTaskEditorDependencies = {
    projectsGateway,
    wbsGateway,
    dependenciesGateway,
    rolesGateway,
    membersGateway,
  };
  return { dependencies, projectsGateway, wbsGateway };
}

describe("SprintTaskReview", () => {
  it("renders Member daily plans directly without Sprint summaries or date navigation_SPD09", () => {
    const harry = member("member-1", "Harry", [
      { date: "2026-08-04", minutes: 480 },
    ]);
    const sally = member("member-2", "Sally", [
      { date: "2026-08-04", minutes: 480 },
    ]);
    const tasks = [
      task("Low Project", {
        projectId: "project-low",
        projectName: "Low",
        projectPriority: 9,
        wbsPath: "2",
        wbsRank: 2,
        allocations: [{ date: "2026-08-04", minutes: 60 }],
      }),
      task("High WBS 2", {
        projectId: "project-high",
        projectName: "High",
        projectPriority: 1,
        wbsPath: "1.2",
        wbsRank: 2,
        allocations: [{ date: "2026-08-04", minutes: 60 }],
      }),
      task("High WBS 1", {
        projectId: "project-high",
        projectName: "High",
        projectPriority: 1,
        wbsPath: "1.1",
        wbsRank: 1,
        allocations: [
          { date: "2026-08-03", minutes: 60 },
          { date: "2026-08-04", minutes: 420 },
        ],
        dailyPlanOrderDate: "2026-08-03",
        inSprintAllocationMinutes: 420,
        outsideAllocationMinutes: 60,
        totalAllocationMinutes: 480,
      }),
      task("Completed", {
        completed: true,
        dailyPlanOrderDate: "2026-08-02",
        allocations: [{ date: "2026-08-04", minutes: 60 }],
      }),
      task("Non-member", {
        projectId: "project-external",
        projectName: "External",
        assigneeId: "member-3",
        assigneeName: "Tony",
        allocations: [{ date: "2026-08-04", minutes: 60 }],
        inSprintAllocationMinutes: 60,
        totalAllocationMinutes: 60,
        warnings: ["Assignee is not included in this Sprint."],
      }),
      task("Unreadable", {
        dailyPlanOrderDate: undefined,
        allocations: [],
        inSprintAllocationMinutes: 0,
        totalAllocationMinutes: 0,
        warnings: ["Execution allocation could not be resolved."],
      }),
      task("Sally Task", {
        projectId: "project-beta",
        projectName: "Beta",
        assigneeId: "member-2",
        assigneeName: "Sally",
        allocations: [{ date: "2026-08-04", minutes: 360 }],
        inSprintAllocationMinutes: 360,
        totalAllocationMinutes: 360,
      }),
    ];

    render(
      <SprintTaskReview
        detail={detail([harry, sally], tasks)}
        gateway={gateway()}
        onEdit={vi.fn()}
        onClose={vi.fn()}
        onChanged={vi.fn()}
      />,
    );

    expect(screen.getByText("Sprint Planning")).toBeTruthy();
    expect(
      screen.getByRole("button", { name: "Save Sprint Planning" }),
    ).toBeTruthy();
    expect(
      screen.getByRole("button", { name: "Close Sprint Planning" }),
    ).toBeTruthy();
    expect(screen.queryByText("Task Review")).toBeNull();
    const harryRegion = screen.getByRole("region", { name: "Harry" });
    const harryGrid = within(harryRegion).getByRole("region", {
      name: "Harry daily working plan",
    });
    expect(harryGrid.getAttribute("tabindex")).toBe("0");
    harryGrid.focus();
    expect(document.activeElement).toBe(harryGrid);
    expect(
      within(harryRegion)
        .getAllByRole("button", { name: /Remove .* from Sprint/ })
        .map((button) => button.getAttribute("aria-label")),
    ).toEqual([
      "Remove High WBS 1 from Sprint",
      "Remove High WBS 2 from Sprint",
      "Remove Low Project from Sprint",
      "Remove Completed from Sprint",
    ]);
    expect(harryRegion.textContent).toContain("High (open)");
    expect(harryRegion.textContent).not.toContain("WBS 1.1");
    expect(harryRegion.textContent).toContain("Completed");
    expect(harryRegion.textContent).toContain("Capacity: 8h");

    const needsReviewRegion = screen.getByRole("region", {
      name: "Needs Review",
    });
    expect(needsReviewRegion.textContent).toContain("Non-member");
    expect(needsReviewRegion.textContent).toContain("Unreadable");
    expect(screen.queryByText("Sprint daily summary")).toBeNull();
    expect(
      screen.queryByRole("group", { name: "Sprint Task totals" }),
    ).toBeNull();
    expect(screen.queryByRole("button", { name: "Previous Dates" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Next Dates" })).toBeNull();
    expect(
      screen.getByLabelText("Harry, 04 Aug 2026, Daily Capacity: 8h"),
    ).toBeTruthy();
    expect(
      screen.queryByLabelText(/Harry, .* Selected Allocation:/),
    ).toBeNull();
    expect(screen.queryByLabelText(/Harry, .* Remaining:/)).toBeNull();
    expect(screen.queryByLabelText(/Harry, .* Overcapacity:/)).toBeNull();
    expect(
      screen.queryByLabelText(
        "Harry, High WBS 1, High, 03 Aug 2026, Sprint allocation: 1h",
      ),
    ).toBeNull();
    expect(screen.queryByText("03 Aug 2026")).toBeNull();
    expect(screen.getAllByText("Sprint Start").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Sprint End").length).toBeGreaterThan(0);
  });

  it("orders unfinished readable Tasks before completed and unreadable Tasks across Sprint boundaries_SPD01_AC34To36", () => {
    const tasks = [
      task("Completed earliest", {
        completed: true,
        dailyPlanOrderDate: "2026-08-02",
        allocations: [{ date: "2026-08-02", minutes: 60 }],
        inSprintAllocationMinutes: 0,
        outsideAllocationMinutes: 60,
      }),
      task("After Sprint", {
        dailyPlanOrderDate: "2026-08-09",
        executionStart: "2026-08-09",
        executionEnd: "2026-08-09",
        allocations: [{ date: "2026-08-09", minutes: 60 }],
        inSprintAllocationMinutes: 0,
        outsideAllocationMinutes: 60,
      }),
      task("In Sprint", {
        dailyPlanOrderDate: "2026-08-04",
        allocations: [{ date: "2026-08-04", minutes: 60 }],
      }),
      task("Unreadable", {
        dailyPlanOrderDate: undefined,
        allocations: [],
        inSprintAllocationMinutes: 0,
        totalAllocationMinutes: 0,
        warnings: ["Execution allocation could not be resolved."],
      }),
      task("Overdue outside", {
        dailyPlanOrderDate: "2026-08-03",
        executionStart: "2026-08-03",
        executionEnd: "2026-08-03",
        allocations: [{ date: "2026-08-03", minutes: 60 }],
        inSprintAllocationMinutes: 0,
        outsideAllocationMinutes: 60,
      }),
    ];

    render(
      <SprintTaskReview
        detail={detail(
          [member("member-1", "Harry", [{ date: "2026-08-04", minutes: 480 }])],
          tasks,
        )}
        gateway={gateway()}
        onEdit={vi.fn()}
        onClose={vi.fn()}
        onChanged={vi.fn()}
      />,
    );

    expect(
      within(screen.getByRole("region", { name: "Harry" }))
        .getAllByRole("button", { name: /Remove .* from Sprint/ })
        .map((button) => button.getAttribute("aria-label")),
    ).toEqual([
      "Remove Overdue outside from Sprint",
      "Remove In Sprint from Sprint",
      "Remove After Sprint from Sprint",
      "Remove Completed earliest from Sprint",
    ]);
    expect(
      within(screen.getByRole("region", { name: "Needs Review" })).getByRole(
        "button",
        { name: "Remove Unreadable from Sprint" },
      ),
    ).toBeTruthy();
    expect(
      screen.queryByLabelText(
        "Harry, Overdue outside, Alpha, 03 Aug 2026, Sprint allocation: 1h",
      ),
    ).toBeNull();
    expect(
      screen.queryByLabelText(
        "Harry, After Sprint, Alpha, 09 Aug 2026, Sprint allocation: 1h",
      ),
    ).toBeNull();
    expect(screen.queryByRole("button", { name: "Previous Dates" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Next Dates" })).toBeNull();
  });

  it("shows Add Task results inline and adds the Task allocation row without aggregate summaries_SPD09_AC40To43And74", async () => {
    const api = gateway([
      task("API", {
        projectId: "project-1",
        projectName: "Alpha",
        wbsPath: "1.1",
        wbsRank: 2,
        executionStart: "2026-08-04",
        executionEnd: "2026-08-04",
        allocations: [{ date: "2026-08-04", minutes: 600 }],
        inSprintAllocationMinutes: 600,
        totalAllocationMinutes: 600,
      }),
    ]);

    render(
      <SprintTaskReview
        detail={detail(
          [member("member-1", "Harry", [{ date: "2026-08-04", minutes: 480 }])],
          [],
        )}
        gateway={api}
        onEdit={vi.fn()}
        onClose={vi.fn()}
        onChanged={vi.fn()}
      />,
    );

    const emptyMember = screen.getByRole("region", { name: "Harry" });
    expect(within(emptyMember).getByText("No selected Tasks.")).toBeTruthy();
    expect(
      screen.getByLabelText("Harry, 04 Aug 2026, Daily Capacity: 8h"),
    ).toBeTruthy();

    await userEvent.click(screen.getByRole("button", { name: "Add Task" }));
    const heading = await screen.findByRole("heading", {
      name: "Eligible Tasks",
    });
    const memberHeading = screen.getByRole("heading", { name: "Harry" });
    const review = screen.getByRole("region", { name: "August" });
    const candidateRegion = screen.getByRole("region", {
      name: "Eligible Tasks",
    });
    expect(document.activeElement).toBe(heading);
    expect(review.contains(candidateRegion)).toBe(true);
    expect(screen.queryByRole("dialog")).toBeNull();
    expect(
      heading.compareDocumentPosition(memberHeading) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
    expect(candidateRegion.textContent).toContain("Alpha");
    expect(candidateRegion.textContent).toContain("API");
    expect(candidateRegion.textContent).not.toContain("WBS");
    expect(candidateRegion.textContent).toContain("04 Aug 2026");

    await userEvent.click(
      screen.getByRole("button", {
        name: "Add API from Alpha to Sprint",
      }),
    );

    await waitFor(() =>
      expect(
        screen.getByRole("button", { name: "Remove API from Sprint" }),
      ).toBeTruthy(),
    );
    expect(screen.getByRole("status").textContent).toContain(
      "API from Alpha was added",
    );
    expect(document.activeElement).toBe(heading);
    expect(
      screen.getByLabelText(
        "Harry, API, Alpha, 04 Aug 2026, Sprint allocation: 10h",
      ),
    ).toBeTruthy();
    expect(screen.queryByText("Sprint daily summary")).toBeNull();
  });

  it("removes a Task while preserving Member capacity and the unsaved selection_SPD09_AC40To43", async () => {
    render(
      <SprintTaskReview
        detail={detail(
          [member("member-1", "Harry", [{ date: "2026-08-04", minutes: 480 }])],
          [
            task("API", {
              allocations: [{ date: "2026-08-04", minutes: 600 }],
              inSprintAllocationMinutes: 600,
              totalAllocationMinutes: 600,
            }),
          ],
        )}
        gateway={gateway()}
        onEdit={vi.fn()}
        onClose={vi.fn()}
        onChanged={vi.fn()}
      />,
    );

    expect(
      screen.getByLabelText(
        "Harry, API, Alpha, 04 Aug 2026, Sprint allocation: 10h",
      ),
    ).toBeTruthy();
    await userEvent.click(
      screen.getByRole("button", { name: "Remove API from Sprint" }),
    );

    expect(
      screen.queryByRole("button", { name: "Remove API from Sprint" }),
    ).toBeNull();
    expect(
      screen.getByLabelText("Harry, 04 Aug 2026, Daily Capacity: 8h"),
    ).toBeTruthy();
    expect(screen.queryByLabelText(/Harry, .* Overcapacity:/)).toBeNull();
  });

  it("preserves reviewed membership when suggestion generation fails_SPD07_AC77", async () => {
    const api = gateway();
    vi.mocked(api.suggest).mockRejectedValueOnce(
      new Error("canonical allocation unavailable"),
    );
    render(
      <SprintTaskReview
        detail={detail(
          [member("member-1", "Harry", [{ date: "2026-08-04", minutes: 480 }])],
          [task("Existing plan")],
        )}
        gateway={api}
        onEdit={vi.fn()}
        onClose={vi.fn()}
        onChanged={vi.fn()}
      />,
    );

    await userEvent.click(
      screen.getByRole("button", { name: "Regenerate Suggestion" }),
    );

    expect(
      await screen.findByText(
        "Task suggestion could not be generated. The current Sprint Planning selection is unchanged.",
      ),
    ).toBeTruthy();
    expect(
      screen.getByRole("button", { name: "Remove Existing plan from Sprint" }),
    ).toBeTruthy();
  });

  it("ignores a stale suggestion response after a newer Sprint projection is reviewed_SPD07_AC74", async () => {
    let resolveFirst!: (value: SprintSuggestion) => void;
    let resolveSecond!: (value: SprintSuggestion) => void;
    const first = new Promise<SprintSuggestion>((resolve) => {
      resolveFirst = resolve;
    });
    const second = new Promise<SprintSuggestion>((resolve) => {
      resolveSecond = resolve;
    });
    const api = gateway();
    let firstSignal: AbortSignal | undefined;
    vi.mocked(api.suggest)
      .mockImplementationOnce((_input, signal) => {
        firstSignal = signal;
        return first;
      })
      .mockImplementationOnce(() => second);
    const initial = detail(
      [member("member-1", "Harry", [{ date: "2026-08-04", minutes: 480 }])],
      [],
    );
    const current = {
      ...initial,
      sprint: { ...initial.sprint, version: 2 },
      projectionToken: "projection-2",
    };
    const view = render(
      <SprintTaskReview
        detail={initial}
        gateway={api}
        onEdit={vi.fn()}
        onClose={vi.fn()}
        onChanged={vi.fn()}
      />,
    );

    await userEvent.click(
      screen.getByRole("button", { name: "Generate Suggestion" }),
    );
    view.rerender(
      <SprintTaskReview
        detail={current}
        gateway={api}
        onEdit={vi.fn()}
        onClose={vi.fn()}
        onChanged={vi.fn()}
      />,
    );
    expect(firstSignal?.aborted).toBe(true);
    await waitFor(() =>
      expect(
        screen
          .getByRole("button", { name: "Generate Suggestion" })
          .getAttribute("disabled"),
      ).toBeNull(),
    );
    await userEvent.click(
      screen.getByRole("button", { name: "Generate Suggestion" }),
    );

    await act(async () => {
      resolveSecond({
        members: current.members,
        tasks: [{ task: task("Current plan"), reason: "mandatory" }],
        totals: current.totals,
        projectionToken: "suggestion-current",
      });
    });
    expect(
      await screen.findByRole("button", {
        name: "Remove Current plan from Sprint",
      }),
    ).toBeTruthy();

    await act(async () => {
      resolveFirst({
        members: initial.members,
        tasks: [{ task: task("Stale plan"), reason: "mandatory" }],
        totals: initial.totals,
        projectionToken: "suggestion-stale",
      });
    });
    await waitFor(() =>
      expect(
        screen.queryByRole("button", {
          name: "Remove Stale plan from Sprint",
        }),
      ).toBeNull(),
    );
    expect(
      screen.getByRole("button", { name: "Remove Current plan from Sprint" }),
    ).toBeTruthy();
  });

  it("ignores a stale candidate response after Sprint detail changes_SPD07_AC74", async () => {
    type CandidatePage = {
      items: SprintTaskProjection[];
      page: number;
      pageSize: number;
      total: number;
    };
    let resolveCandidates!: (value: CandidatePage) => void;
    const pending = new Promise<CandidatePage>((resolve) => {
      resolveCandidates = resolve;
    });
    const api = gateway();
    let candidateSignal: AbortSignal | undefined;
    vi.mocked(api.candidates).mockImplementationOnce((_id, _page, signal) => {
      candidateSignal = signal;
      return pending;
    });
    const initial = detail(
      [member("member-1", "Harry", [{ date: "2026-08-04", minutes: 480 }])],
      [],
    );
    const view = render(
      <SprintTaskReview
        detail={initial}
        gateway={api}
        onEdit={vi.fn()}
        onClose={vi.fn()}
        onChanged={vi.fn()}
      />,
    );

    await userEvent.click(screen.getByRole("button", { name: "Add Task" }));
    view.rerender(
      <SprintTaskReview
        detail={{
          ...initial,
          sprint: { ...initial.sprint, version: 2 },
          projectionToken: "projection-2",
        }}
        gateway={api}
        onEdit={vi.fn()}
        onClose={vi.fn()}
        onChanged={vi.fn()}
      />,
    );
    expect(candidateSignal?.aborted).toBe(true);
    await act(async () => {
      resolveCandidates({
        items: [task("Stale candidate")],
        page: 1,
        pageSize: 20,
        total: 1,
      });
    });

    expect(
      screen.queryByRole("button", {
        name: "Add Stale candidate from Alpha to Sprint",
      }),
    ).toBeNull();
    expect(screen.queryByRole("region", { name: "Eligible Tasks" })).toBeNull();
  });

  it("renders only Sprint Dates and ignores outside allocation columns_SPD10", () => {
    const dates = Array.from({ length: 40 }, (_, index) => {
      const date = new Date("2026-08-01T00:00:00Z");
      date.setUTCDate(date.getUTCDate() + index);
      return date.toISOString().slice(0, 10);
    });
    const allocation = dates.map((date) => ({ date, minutes: 60 }));
    const tasks = [
      task("First", {
        dailyPlanOrderDate: dates[0],
        allocations: allocation,
        inSprintAllocationMinutes: 31 * 60,
        outsideAllocationMinutes: 9 * 60,
        totalAllocationMinutes: 40 * 60,
      }),
      task("Second", {
        dailyPlanOrderDate: dates[1],
        allocations: allocation,
        inSprintAllocationMinutes: 31 * 60,
        outsideAllocationMinutes: 9 * 60,
        totalAllocationMinutes: 40 * 60,
        wbsRank: 2,
      }),
    ];

    render(
      <SprintTaskReview
        detail={detail(
          [
            member(
              "member-1",
              "Harry",
              dates.slice(0, 31).map((date) => ({ date, minutes: 480 })),
            ),
          ],
          tasks,
          "2026-08-01",
          "2026-08-31",
        )}
        gateway={gateway()}
        onEdit={vi.fn()}
        onClose={vi.fn()}
        onChanged={vi.fn()}
      />,
    );

    expect(screen.queryByRole("button", { name: "Previous Dates" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Next Dates" })).toBeNull();
    expect(screen.getAllByText("01 Aug 2026").length).toBeGreaterThan(0);
    expect(screen.getAllByText("31 Aug 2026").length).toBeGreaterThan(0);
    expect(screen.queryByText("01 Sep 2026")).toBeNull();
    expect(screen.queryByText("09 Sep 2026")).toBeNull();
    expect(
      screen.queryByLabelText(
        "Harry, First, Alpha, 09 Sep 2026, Sprint allocation: 1h",
      ),
    ).toBeNull();
    expect(
      within(
        screen.getByRole("region", { name: "Harry daily working plan" }),
      ).getAllByRole("columnheader"),
    ).toHaveLength(32);
    expect(
      within(screen.getByRole("region", { name: "Harry" }))
        .getAllByRole("button", { name: /Remove .* from Sprint/ })
        .map((button) => button.getAttribute("aria-label")),
    ).toEqual(["Remove First from Sprint", "Remove Second from Sprint"]);
  });

  it("formats dates, wraps long Task names, hides WBS and allocation metadata, and marks the whole zero-daily-capacity column_SPD09", () => {
    const longName =
      "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-long-task-name";
    render(
      <SprintTaskReview
        detail={detail(
          [
            member("member-1", "Harry", [
              { date: "2026-08-04", minutes: 480 },
              { date: "2026-08-05", minutes: 0 },
              { date: "2026-08-06", minutes: 480 },
              { date: "2026-08-07", minutes: 480 },
              { date: "2026-08-08", minutes: 480 },
            ]),
          ],
          [
            task(longName, {
              wbsPath: "9.4.2",
              executionStart: "2026-08-04",
              executionEnd: "2026-08-08",
              dailyPlanOrderDate: "2026-08-04",
              allocations: [{ date: "2026-08-04", minutes: 480 }],
              inSprintAllocationMinutes: 480,
              outsideAllocationMinutes: 0,
              totalAllocationMinutes: 480,
            }),
          ],
        )}
        gateway={gateway()}
        onEdit={vi.fn()}
        onClose={vi.fn()}
        onChanged={vi.fn()}
      />,
    );

    const taskName = screen.getByText(longName);
    expect(taskName.className).toContain("max-w-[50ch]");
    expect(taskName.className).toContain("break-words");
    const harryRegion = screen.getByRole("region", { name: "Harry" });
    expect(harryRegion.textContent).toContain("04 Aug 2026 – 08 Aug 2026");
    expect(harryRegion.textContent).not.toContain("WBS 9.4.2");
    expect(harryRegion.textContent).not.toContain("in Sprint");
    expect(harryRegion.textContent).not.toContain("outside ·");
    expect(harryRegion.textContent).not.toContain("First planned Date");
    const noCapacityHeader = within(harryRegion).getByRole("columnheader", {
      name: "Harry, 05 Aug 2026, daily capacity 0h, no capacity",
    });
    expect(noCapacityHeader.textContent).toContain("No capacity");

    const markedColumnCells = harryRegion.querySelectorAll(
      '[data-date="2026-08-05"][data-no-capacity="true"]',
    );
    expect(markedColumnCells).toHaveLength(3);
    markedColumnCells.forEach((cell) =>
      expect(cell.className).toContain("bg-warning-soft"),
    );
    expect(
      harryRegion.querySelectorAll(
        '[data-date="2026-08-04"][data-no-capacity="true"]',
      ),
    ).toHaveLength(0);
  });

  it("opens the shared Home Task editor and closes back to the same Sprint Planning without action_SPD11_AC80", async () => {
    const api = gateway();
    const editor = taskEditorHarness("API");
    const closeSprintPlanning = vi.fn();
    window.location.hash = "#sprints";
    render(
      <SprintTaskReview
        detail={detail(
          [member("member-1", "Harry", [{ date: "2026-08-04", minutes: 480 }])],
          [task("API")],
        )}
        gateway={api}
        taskEditorDependencies={editor.dependencies}
        onEdit={vi.fn()}
        onClose={closeSprintPlanning}
        onChanged={vi.fn()}
      />,
    );

    await userEvent.click(screen.getByRole("button", { name: "API" }));
    const dialog = await screen.findByRole("dialog", { name: "Edit Task" });
    expect(editor.projectsGateway.get).toHaveBeenCalledWith("project-alpha");
    expect(
      (within(dialog).getByLabelText("Name") as HTMLInputElement).value,
    ).toBe("API");

    await userEvent.click(
      within(dialog).getByRole("button", { name: "Close" }),
    );
    await waitFor(() =>
      expect(screen.queryByRole("dialog", { name: "Edit Task" })).toBeNull(),
    );
    expect(screen.getByText("Sprint Planning")).toBeTruthy();
    expect(screen.getByRole("button", { name: "API" })).toBeTruthy();
    expect(editor.wbsGateway.updateExecutable).not.toHaveBeenCalled();
    expect(api.suggest).not.toHaveBeenCalled();
    expect(api.update).not.toHaveBeenCalled();
    expect(closeSprintPlanning).not.toHaveBeenCalled();
    expect(window.location.hash).toBe("#sprints");
  });

  it("regenerates Sprint Planning after saving through the shared Home Task editor without navigating Home_SPD11_AC80", async () => {
    const current = detail(
      [member("member-1", "Harry", [{ date: "2026-08-04", minutes: 480 }])],
      [task("API")],
    );
    const api = gateway();
    vi.mocked(api.suggest).mockResolvedValue({
      members: current.members,
      tasks: [
        {
          task: task("API", { name: "Updated API" }),
          reason: "mandatory",
        },
      ],
      totals: current.totals,
      projectionToken: "projection-after-task-save",
    });
    const editor = taskEditorHarness("API");
    const closeSprintPlanning = vi.fn();
    window.location.hash = "#sprints";
    render(
      <SprintTaskReview
        detail={current}
        gateway={api}
        taskEditorDependencies={editor.dependencies}
        onEdit={vi.fn()}
        onClose={closeSprintPlanning}
        onChanged={vi.fn()}
      />,
    );

    await userEvent.click(screen.getByRole("button", { name: "API" }));
    const dialog = await screen.findByRole("dialog", { name: "Edit Task" });
    const name = within(dialog).getByLabelText("Name");
    await userEvent.clear(name);
    await userEvent.type(name, "Updated API");
    await userEvent.click(within(dialog).getByRole("button", { name: "Save" }));

    await waitFor(() =>
      expect(editor.wbsGateway.updateExecutable).toHaveBeenCalledWith(
        "project-alpha",
        "API",
        expect.objectContaining({ name: "Updated API" }),
      ),
    );
    await waitFor(() =>
      expect(api.suggest).toHaveBeenCalledWith(
        {
          startDate: "2026-08-04",
          endDate: "2026-08-08",
          memberIds: ["member-1"],
        },
        expect.any(AbortSignal),
      ),
    );
    expect(
      await screen.findByRole("button", { name: "Updated API" }),
    ).toBeTruthy();
    expect(screen.queryByRole("dialog", { name: "Edit Task" })).toBeNull();
    expect(window.location.hash).toBe("#sprints");
    expect(closeSprintPlanning).not.toHaveBeenCalled();
    expect(api.update).not.toHaveBeenCalled();
  });
});

describe("deriveReviewProjection", () => {
  it("sums daily remaining and overcapacity without aggregate netting_SPD03And04_AC19And79", () => {
    const result = deriveReviewProjection(
      [
        member("member-a", "A", [{ date: "2026-08-04", minutes: 480 }]),
        member("member-b", "B", [{ date: "2026-08-04", minutes: 480 }]),
      ],
      [
        task("A task", {
          assigneeId: "member-a",
          assigneeName: "A",
          allocations: [{ date: "2026-08-04", minutes: 600 }],
          inSprintAllocationMinutes: 600,
          totalAllocationMinutes: 600,
        }),
        task("B task", {
          assigneeId: "member-b",
          assigneeName: "B",
          allocations: [{ date: "2026-08-04", minutes: 360 }],
          inSprintAllocationMinutes: 360,
          totalAllocationMinutes: 360,
        }),
      ],
      "2026-08-04",
      "2026-08-04",
    );

    expect(result.totals).toMatchObject({
      capacityMinutes: 960,
      selectedMemberAllocationMinutes: 960,
      remainingMinutes: 120,
      overcapacityMinutes: 120,
    });
    expect(result.totals.dailySummaries).toEqual([
      {
        date: "2026-08-04",
        capacityMinutes: 960,
        selectedAllocationMinutes: 960,
        remainingMinutes: 120,
        overcapacityMinutes: 120,
      },
    ]);
  });
});
