import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { SprintsGateway } from "../application/sprintsGateway";
import type { SprintDetail } from "../domain/sprint";
import { SprintTaskReview } from "./SprintTaskReview";

describe("SprintTaskReview", () => {
  it("renders Tasks by Member in the authoritative Project and WBS hierarchy without technical ordering labels", async () => {
    const detail: SprintDetail = {
      sprint: {
        id: "sprint-1",
        name: "August",
        startDate: "2026-08-04",
        endDate: "2026-08-08",
        status: "planned",
        version: 1,
      },
      members: [
        {
          id: "member-1",
          name: "Harry",
          roleName: "Engineer",
          dailyCapacity: [],
          capacityMinutes: 480,
          inSprintAllocationMinutes: 120,
          remainingMinutes: 360,
          overcapacityMinutes: 0,
        },
      ],
      tasks: [
        {
          id: "task-1",
          projectId: "project-1",
          projectName: "Alpha",
          projectStatus: "open",
          name: "API",
          wbsOrder: "1.1",
          assigneeId: "member-1",
          assigneeName: "Harry",
          completed: false,
          allocations: [],
          inSprintAllocationMinutes: 120,
          outsideAllocationMinutes: 0,
          totalAllocationMinutes: 120,
          warnings: [],
        },
      ],
      totals: {
        capacityMinutes: 480,
        selectedMemberAllocationMinutes: 120,
        needsReviewAllocationMinutes: 0,
        allTaskInSprintMinutes: 120,
        allTaskTotalMinutes: 120,
      },
      projectionToken: "p",
    };
    const gateway: Pick<SprintsGateway, "suggest" | "candidates" | "update"> = {
      suggest: vi.fn(),
      candidates: vi.fn(),
      update: vi.fn(),
    };
    const wbsGateway = {
      tree: vi.fn().mockResolvedValue([
        {
          id: "group-1",
          projectId: "project-1",
          name: "Backend",
          position: 1,
          hasChildren: true,
          executable: {
            lagDays: 0,
            executionTimeline: {},
            commitmentTimeline: {},
          },
          children: [
            {
              id: "task-1",
              projectId: "project-1",
              parentId: "group-1",
              name: "API",
              position: 1,
              hasChildren: false,
              executable: {
                lagDays: 0,
                executionTimeline: {},
                commitmentTimeline: {},
              },
              children: [],
            },
          ],
        },
      ]),
    };
    render(
      <SprintTaskReview
        detail={detail}
        gateway={gateway}
        wbsGateway={wbsGateway}
        onEdit={vi.fn()}
        onClose={vi.fn()}
        onChanged={vi.fn()}
      />,
    );
    const backendGroup = (await screen.findByText("Backend")).closest("li");
    expect(backendGroup).toBeTruthy();
    expect(within(backendGroup as HTMLElement).getByText("API")).toBeTruthy();
    const review = screen.getByRole("region", { name: "August" });
    expect(review.textContent).toContain("Alpha");
    expect(review.textContent).not.toContain("Level 0 · Project");
    expect(review.textContent).not.toContain("1.1");
    expect(
      screen.getByRole("button", { name: "Remove API from Sprint" }),
    ).toBeTruthy();
    expect(screen.getByLabelText("Harry Project structure")).toBeTruthy();
  });

  it("shows Add Task results next to the action and adds the selected Task", async () => {
    const detail: SprintDetail = {
      sprint: {
        id: "sprint-1",
        name: "August",
        startDate: "2026-08-04",
        endDate: "2026-08-08",
        status: "planned",
        version: 1,
      },
      members: [
        {
          id: "member-1",
          name: "Harry",
          roleName: "Engineer",
          dailyCapacity: [],
          capacityMinutes: 480,
          inSprintAllocationMinutes: 0,
          remainingMinutes: 480,
          overcapacityMinutes: 0,
        },
      ],
      tasks: [],
      totals: {
        capacityMinutes: 480,
        selectedMemberAllocationMinutes: 0,
        needsReviewAllocationMinutes: 0,
        allTaskInSprintMinutes: 0,
        allTaskTotalMinutes: 0,
      },
      projectionToken: "p",
    };
    const candidate = {
      id: "task-1",
      projectId: "project-1",
      projectName: "Alpha",
      projectStatus: "open",
      name: "API",
      wbsOrder: ":00000004",
      assigneeId: "member-1",
      assigneeName: "Harry",
      executionStart: "2026-08-06",
      executionEnd: "2026-08-10",
      completed: false,
      allocations: [],
      inSprintAllocationMinutes: 0,
      outsideAllocationMinutes: 0,
      totalAllocationMinutes: 0,
      warnings: [],
    };
    const gateway: Pick<SprintsGateway, "suggest" | "candidates" | "update"> = {
      suggest: vi.fn(),
      candidates: vi.fn().mockResolvedValue({
        items: [candidate],
        page: 1,
        pageSize: 20,
        total: 1,
      }),
      update: vi.fn(),
    };
    render(
      <SprintTaskReview
        detail={detail}
        gateway={gateway}
        wbsGateway={{ tree: vi.fn().mockResolvedValue([]) }}
        onEdit={vi.fn()}
        onClose={vi.fn()}
        onChanged={vi.fn()}
      />,
    );

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
    expect(candidateRegion.textContent).not.toContain(":00000004");
    expect(candidateRegion.textContent).toContain("2026-08-06");
    expect(candidateRegion.textContent).toContain("2026-08-10");
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
  });
});
