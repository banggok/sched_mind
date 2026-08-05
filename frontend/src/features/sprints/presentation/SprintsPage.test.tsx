import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import axe from "axe-core";
import { describe, expect, it, vi } from "vitest";
import type { SprintsGateway } from "../application/sprintsGateway";
import type { Sprint, SprintDetail } from "../domain/sprint";
import type { TeamMembersGateway } from "../../team-members/application/teamMembersGateway";
import { formatDateOnly } from "../../../shared/presentation/formatDateOnly";
import { SprintsPage } from "./SprintsPage";

const planned: Sprint = {
  id: "sprint-1",
  name: "August Sprint",
  startDate: "2026-08-04",
  endDate: "2026-08-08",
  status: "planned",
  version: 1,
  memberIds: ["member-1"],
  taskIds: ["task-1"],
  createdAt: new Date("2026-08-01T00:00:00Z"),
  updatedAt: new Date("2026-08-01T00:00:00Z"),
};

const secondSprint: Sprint = {
  ...planned,
  id: "sprint-2",
  name: "September Sprint",
  startDate: "2026-09-01",
  endDate: "2026-09-05",
};

function detail(sprint: Sprint): SprintDetail {
  return {
    sprint: {
      id: sprint.id,
      name: sprint.name,
      startDate: sprint.startDate,
      endDate: sprint.endDate,
      status: sprint.status,
      version: sprint.version,
    },
    projectionToken: "projection-1",
    members: [],
    tasks: [],
    totals: {
      capacityMinutes: 480,
      selectedMemberAllocationMinutes: 120,
      remainingMinutes: 360,
      overcapacityMinutes: 0,
      needsReviewAllocationMinutes: 0,
      needsReviewDailyAllocation: [],
      allTaskInSprintMinutes: 120,
      allTaskTotalMinutes: 180,
      dailySummaries: [],
    },
  };
}

function gateway(): SprintsGateway {
  return {
    list: vi
      .fn()
      .mockResolvedValue({ items: [planned], page: 1, pageSize: 10, total: 1 }),
    detail: vi.fn().mockResolvedValue(detail(planned)),
    create: vi.fn().mockResolvedValue(planned),
    update: vi.fn().mockResolvedValue(planned),
    suggest: vi.fn().mockResolvedValue({
      members: [],
      tasks: [],
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
      projectionToken: "p",
    }),
    candidates: vi
      .fn()
      .mockResolvedValue({ items: [], page: 1, pageSize: 20, total: 0 }),
    draftCandidates: vi
      .fn()
      .mockResolvedValue({ items: [], page: 1, pageSize: 20, total: 0 }),
    start: vi
      .fn()
      .mockResolvedValue({ ...planned, status: "started", version: 2 }),
    delete: vi.fn().mockResolvedValue(undefined),
  };
}

const membersGateway: TeamMembersGateway = {
  list: vi
    .fn()
    .mockResolvedValue({ items: [], page: 1, pageSize: 100, total: 0 }),
  create: vi.fn(),
  update: vi.fn(),
  delete: vi.fn(),
};
const loadPublicHolidayDates = vi.fn().mockResolvedValue([]);

describe("SprintsPage", () => {
  it("renders deterministic list columns and opens live detail for either status_AC3And5", async () => {
    const api = gateway();
    const { container } = render(
      <SprintsPage
        gateway={api}
        membersGateway={membersGateway}
        loadPublicHolidayDates={loadPublicHolidayDates}
      />,
    );
    expect(
      await screen.findByRole("button", { name: "August Sprint" }),
    ).toBeTruthy();
    expect(
      screen.getAllByRole("columnheader").map((cell) => cell.textContent),
    ).toEqual(["Sprint Name", "Start Date", "End Date", "Status", "Actions"]);
    expect(
      screen.getByText(
        "Plan focused delivery periods, then review Member capacity and selected Tasks.",
      ),
    ).toBeTruthy();
    expect(screen.getByText("1 Sprint")).toBeTruthy();
    const listRegion = screen.getByRole("region", { name: "Sprint list" });
    expect(listRegion.tabIndex).toBe(0);
    const row = screen
      .getByRole("button", { name: "August Sprint" })
      .closest("tr");
    expect(row).toBeTruthy();
    const sprintRow = within(row as HTMLTableRowElement);
    expect(sprintRow.getByText(formatDateOnly(planned.startDate))).toBeTruthy();
    expect(sprintRow.getByText(formatDateOnly(planned.endDate))).toBeTruthy();
    expect(sprintRow.getByLabelText("Status: Planned")).toBeTruthy();
    expect(
      sprintRow.getByRole("group", { name: "August Sprint actions" }),
    ).toBeTruthy();
    expect(
      (
        await axe.run(container, {
          rules: {
            // jsdom has no layout engine, so contrast remains a browser gate.
            "color-contrast": { enabled: false },
          },
        })
      ).violations,
    ).toEqual([]);
    await userEvent.click(
      screen.getByRole("button", { name: "August Sprint" }),
    );
    expect(
      await screen.findByRole("heading", { name: "August Sprint", level: 2 }),
    ).toBeTruthy();
    expect(screen.queryByRole("dialog")).toBeNull();
    expect(api.detail).toHaveBeenCalledWith("sprint-1");
  });

  it("renders contextual Planned and Started status actions_AC3", async () => {
    const api = gateway();
    const started: Sprint = {
      ...secondSprint,
      status: "started",
      version: 2,
    };
    vi.mocked(api.list).mockResolvedValue({
      items: [planned, started],
      page: 1,
      pageSize: 10,
      total: 2,
    });
    render(
      <SprintsPage
        gateway={api}
        membersGateway={membersGateway}
        loadPublicHolidayDates={loadPublicHolidayDates}
      />,
    );

    const plannedRow = (
      await screen.findByRole("button", { name: planned.name })
    ).closest("tr");
    const startedRow = screen
      .getByRole("button", { name: started.name })
      .closest("tr");
    expect(plannedRow).toBeTruthy();
    expect(startedRow).toBeTruthy();
    const plannedActions = within(plannedRow as HTMLTableRowElement);
    const startedActions = within(startedRow as HTMLTableRowElement);
    expect(plannedActions.getByLabelText("Status: Planned")).toBeTruthy();
    expect(
      plannedActions.getByRole("button", { name: `Start ${planned.name}` }),
    ).toBeTruthy();
    expect(startedActions.getByLabelText("Status: Started")).toBeTruthy();
    expect(
      startedActions.queryByRole("button", { name: `Start ${started.name}` }),
    ).toBeNull();
    expect(
      startedActions.getByRole("group", { name: `${started.name} actions` }),
    ).toBeTruthy();
    expect(
      startedActions.getByRole("button", { name: `Edit ${started.name}` }),
    ).toBeTruthy();
    expect(
      startedActions.getByRole("button", { name: `Delete ${started.name}` }),
    ).toBeTruthy();
  });

  it("opens Add Task results inline from the main Sprint page_AC40", async () => {
    const api = gateway();
    render(
      <SprintsPage
        gateway={api}
        membersGateway={membersGateway}
        loadPublicHolidayDates={loadPublicHolidayDates}
      />,
    );
    await userEvent.click(
      await screen.findByRole("button", { name: "August Sprint" }),
    );
    await userEvent.click(screen.getByRole("button", { name: "Add Task" }));

    expect(
      await screen.findByRole("region", { name: "Eligible Tasks" }),
    ).toBeTruthy();
    expect(screen.getByText("No eligible Tasks found.")).toBeTruthy();
    expect(screen.queryByRole("dialog")).toBeNull();
    expect(api.candidates).toHaveBeenCalledWith(
      "sprint-1",
      1,
      expect.any(AbortSignal),
    );
  });

  it("starts Planned Sprint once and confirms Delete without duplicate submission_AC4And66To71", async () => {
    const api = gateway();
    render(
      <SprintsPage
        gateway={api}
        membersGateway={membersGateway}
        loadPublicHolidayDates={loadPublicHolidayDates}
      />,
    );
    await screen.findByText("August Sprint");
    await userEvent.click(
      screen.getByRole("button", { name: "Start August Sprint" }),
    );
    await waitFor(() => expect(api.start).toHaveBeenCalledWith("sprint-1", 1));
    await userEvent.click(
      screen.getByRole("button", { name: "Delete August Sprint" }),
    );
    const dialog = screen.getByRole("alertdialog", {
      name: "Delete August Sprint?",
    });
    expect(dialog).toBeTruthy();
    await userEvent.click(
      screen.getByRole("button", { name: "Delete Sprint" }),
    );
    await waitFor(() => expect(api.delete).toHaveBeenCalledWith("sprint-1", 1));
  });

  it("shows recoverable list failure without erasing the page_AC4And77", async () => {
    const api = gateway();
    vi.mocked(api.list)
      .mockRejectedValueOnce(new Error("offline"))
      .mockResolvedValueOnce({ items: [], page: 1, pageSize: 10, total: 0 });
    render(
      <SprintsPage
        gateway={api}
        membersGateway={membersGateway}
        loadPublicHolidayDates={loadPublicHolidayDates}
      />,
    );
    expect((await screen.findByRole("alert")).textContent).toContain(
      "Sprints could not be loaded",
    );
    expect(screen.queryByRole("navigation", { name: "Pagination" })).toBeNull();
    await userEvent.click(screen.getByRole("button", { name: "Retry" }));
    expect(await screen.findByText("No Sprints yet")).toBeTruthy();
  });

  it("does not restore an older Sprint detail response after a newer selection_AC74", async () => {
    const api = gateway();
    vi.mocked(api.list).mockResolvedValue({
      items: [planned, secondSprint],
      page: 1,
      pageSize: 10,
      total: 2,
    });
    let resolveFirst: (value: SprintDetail) => void = () => {
      throw new Error("first detail resolver not installed");
    };
    let resolveSecond: (value: SprintDetail) => void = () => {
      throw new Error("second detail resolver not installed");
    };
    vi.mocked(api.detail).mockImplementation((id) =>
      id === planned.id
        ? new Promise((resolve) => {
            resolveFirst = resolve;
          })
        : new Promise((resolve) => {
            resolveSecond = resolve;
          }),
    );
    render(
      <SprintsPage
        gateway={api}
        membersGateway={membersGateway}
        loadPublicHolidayDates={loadPublicHolidayDates}
      />,
    );
    await screen.findByRole("button", { name: planned.name });

    await userEvent.click(screen.getByRole("button", { name: planned.name }));
    await userEvent.click(
      screen.getByRole("button", { name: secondSprint.name }),
    );
    resolveSecond(detail(secondSprint));
    expect(
      await screen.findByRole("heading", {
        name: "September Sprint",
        level: 2,
      }),
    ).toBeTruthy();
    resolveFirst(detail(planned));
    await waitFor(() =>
      expect(
        screen.getByRole("heading", {
          name: "September Sprint",
          level: 2,
        }),
      ).toBeTruthy(),
    );
    expect(
      screen.queryByRole("heading", { name: "August Sprint", level: 2 }),
    ).toBeNull();
  });
});
