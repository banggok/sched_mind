import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { TeamMembersGateway } from "../../team-members/application/teamMembersGateway";
import type { SprintsGateway } from "../application/sprintsGateway";
import type { Sprint, SprintDetail } from "../domain/sprint";
import { SprintFormDialog } from "./SprintFormDialog";

const saved: Sprint = {
  id: "sprint-1",
  name: "August",
  startDate: "2026-08-04",
  endDate: "2026-08-08",
  status: "planned",
  version: 1,
  memberIds: ["member-1"],
  taskIds: [],
  createdAt: new Date(),
  updatedAt: new Date(),
};

function gateways() {
  const gateway: Pick<SprintsGateway, "create" | "update"> = {
    create: vi.fn().mockResolvedValue(saved),
    update: vi.fn().mockResolvedValue(saved),
  };
  const membersGateway: TeamMembersGateway = {
    list: vi.fn().mockResolvedValue({
      items: [
        {
          id: "member-1",
          name: "Harry",
          role: { id: "role-1", name: "Engineer" },
          dailyCapacity: 8,
          bufferPercentage: 20,
          baseExecutionCapacity: 6.4,
          createdAt: new Date(),
          updatedAt: new Date(),
        },
      ],
      page: 1,
      pageSize: 100,
      total: 1,
    }),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  };
  return { gateway, membersGateway };
}

async function selectDate(label: string, date: string) {
  await userEvent.click(
    screen.getByRole("button", { name: `${label}: Select date` }),
  );
  await userEvent.click(screen.getByRole("button", { name: date }));
}

describe("SprintFormDialog", () => {
  it("creates Details and Members without rendering Task Review in the modal", async () => {
    const { gateway, membersGateway } = gateways();
    const onSaved = vi.fn();
    render(
      <SprintFormDialog
        gateway={gateway}
        membersGateway={membersGateway}
        loadPublicHolidayDates={vi.fn().mockResolvedValue([])}
        onClose={vi.fn()}
        onSaved={onSaved}
      />,
    );
    await userEvent.type(screen.getByLabelText("Sprint Name"), "August");
    await selectDate("Start Date", "2026-08-04");
    await selectDate("End Date", "2026-08-08");
    await userEvent.click(
      screen.getByRole("button", { name: "Next: Members" }),
    );
    await userEvent.click(
      await screen.findByRole("checkbox", { name: /Harry/ }),
    );
    expect(screen.queryByText("Task Review")).toBeNull();
    await userEvent.click(
      screen.getByRole("button", { name: "Create Sprint" }),
    );
    await waitFor(() =>
      expect(gateway.create).toHaveBeenCalledWith({
        name: "August",
        startDate: "2026-08-04",
        endDate: "2026-08-08",
        memberIds: ["member-1"],
        taskIds: [],
      }),
    );
    expect(onSaved).toHaveBeenCalledWith(saved);
  });

  it("preserves existing Task relations when editing Details and Members", async () => {
    const { gateway, membersGateway } = gateways();
    const detail: SprintDetail = {
      sprint: {
        id: "sprint-1",
        name: "August",
        startDate: "2026-08-04",
        endDate: "2026-08-08",
        status: "planned",
        version: 3,
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
      tasks: [
        {
          id: "task-1",
          projectId: "project-1",
          projectName: "Alpha",
          projectStatus: "open",
          name: "API",
          wbsOrder: "1",
          assigneeId: "member-1",
          completed: false,
          allocations: [],
          inSprintAllocationMinutes: 0,
          outsideAllocationMinutes: 0,
          totalAllocationMinutes: 0,
          warnings: [],
        },
      ],
      totals: {
        capacityMinutes: 480,
        selectedMemberAllocationMinutes: 0,
        needsReviewAllocationMinutes: 0,
        allTaskInSprintMinutes: 0,
        allTaskTotalMinutes: 0,
      },
      projectionToken: "p",
    };
    render(
      <SprintFormDialog
        gateway={gateway}
        membersGateway={membersGateway}
        initialDetail={detail}
        loadPublicHolidayDates={vi.fn().mockResolvedValue([])}
        onClose={vi.fn()}
        onSaved={vi.fn()}
      />,
    );
    await userEvent.click(
      screen.getByRole("button", { name: "Next: Members" }),
    );
    await userEvent.click(screen.getByRole("button", { name: "Save Sprint" }));
    await waitFor(() =>
      expect(gateway.update).toHaveBeenCalledWith(
        "sprint-1",
        expect.objectContaining({ taskIds: ["task-1"], version: 3 }),
      ),
    );
  });

  it("uses the shared calendar holiday loader in Create Sprint", async () => {
    const { gateway, membersGateway } = gateways();
    const loadPublicHolidayDates = vi.fn().mockResolvedValue(["2026-08-17"]);
    render(
      <SprintFormDialog
        gateway={gateway}
        membersGateway={membersGateway}
        loadPublicHolidayDates={loadPublicHolidayDates}
        onClose={vi.fn()}
        onSaved={vi.fn()}
      />,
    );

    await userEvent.click(
      screen.getByRole("button", { name: "Start Date: Select date" }),
    );

    await waitFor(() =>
      expect(screen.getByRole("button", { name: "2026-08-17" }).title).toBe(
        "Holiday",
      ),
    );
    expect(loadPublicHolidayDates).toHaveBeenCalledWith(
      "2026-08-01",
      "2026-08-31",
    );
  });
});
