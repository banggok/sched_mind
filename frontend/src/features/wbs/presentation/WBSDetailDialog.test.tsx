import { StrictMode } from "react";
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { DependenciesGateway } from "../../dependencies/application/dependenciesGateway";
import type { RolesGateway } from "../../roles/application/rolesGateway";
import type { TeamMembersGateway } from "../../team-members/application/teamMembersGateway";
import type {
  AssigneeRecommendationResult,
  SchedulePreview,
  WBSGateway,
} from "../application/wbsGateway";
import type { WBSNode } from "../domain/wbs";
import { formatDateOnly } from "../../../shared/presentation/formatDateOnly";
import { WBSDetailDialog } from "./WBSDetailDialog";

describe("WBSDetailDialog option loading", () => {
  it("does not report Strict Mode cancellation as a load failure", async () => {
    const rolesGateway = {
      list: vi.fn((_query, signal?: AbortSignal) =>
        abortable(
          {
            items: [
              {
                id: "role",
                name: "Backend",
                createdAt: new Date(),
                updatedAt: new Date(),
              },
            ],
            page: 1,
            pageSize: 100,
            total: 1,
          },
          signal,
        ),
      ),
    } as unknown as RolesGateway;
    const membersGateway = {
      list: vi.fn((_query, signal?: AbortSignal) =>
        abortable({ items: [], page: 1, pageSize: 100, total: 0 }, signal),
      ),
    } as unknown as TeamMembersGateway;
    const gateway = {} as WBSGateway;
    const node: WBSNode = {
      id: "task",
      projectId: "project",
      name: "API",
      position: 1,
      hasChildren: false,
      executable: { lagDays: 0, executionTimeline: {}, commitmentTimeline: {} },
      children: [],
    };
    render(
      <StrictMode>
        <WBSDetailDialog
          project={{
            id: "project",
            name: "Alpha",
            status: "open",
            autoCalculateDate: true,
            automaticScheduling: true,
            projectBuffer: 20,
            scheduleVersion: 0,
            priority: 1,
            createdAt: new Date(),
            updatedAt: new Date(),
          }}
          node={node}
          gateway={gateway}
          rolesGateway={rolesGateway}
          membersGateway={membersGateway}
          onClose={() => undefined}
          onChanged={() => undefined}
          onReopened={() => undefined}
        />
      </StrictMode>,
    );
    await waitFor(() =>
      expect(screen.getByRole("option", { name: "Backend" })).not.toBeNull(),
    );
    expect(
      screen.queryByText("Role and member options could not be loaded."),
    ).toBeNull();
  });

  it("uses the wide responsive task layout before relying on vertical scrolling", () => {
    const node: WBSNode = {
      id: "task",
      projectId: "project",
      name: "API",
      position: 1,
      hasChildren: false,
      executable: {
        lagDays: 0,
        executionTimeline: {},
        commitmentTimeline: {},
      },
      children: [],
    };
    const rolesGateway = {
      list: vi
        .fn()
        .mockResolvedValue({ items: [], page: 1, pageSize: 100, total: 0 }),
    } as unknown as RolesGateway;
    const membersGateway = {
      list: vi
        .fn()
        .mockResolvedValue({ items: [], page: 1, pageSize: 100, total: 0 }),
    } as unknown as TeamMembersGateway;

    render(
      <WBSDetailDialog
        project={{
          id: "project",
          name: "Alpha",
          status: "open",
          autoCalculateDate: true,
          automaticScheduling: true,
          projectBuffer: 20,
          scheduleVersion: 0,
          priority: 1,
          createdAt: new Date(),
          updatedAt: new Date(),
        }}
        node={node}
        gateway={{} as WBSGateway}
        rolesGateway={rolesGateway}
        membersGateway={membersGateway}
        onClose={() => undefined}
        onChanged={() => undefined}
        onReopened={() => undefined}
      />,
    );

    const dialog = screen.getByRole("dialog", { name: "Edit Task" });
    expect(dialog.className).toContain("dialog-panel-wide");

    const form = screen.getByLabelText("Name").closest("form");
    expect(form).not.toBeNull();
    expect(form?.className).toContain("md:grid-cols-2");

    expect(
      screen.getByRole("group", { name: "Generated schedule" }),
    ).not.toBeNull();
  });

  it("saves the task name and executable details through one update", async () => {
    const node: WBSNode = {
      id: "task",
      projectId: "project",
      name: "API",
      position: 1,
      hasChildren: false,
      executable: { lagDays: 0, executionTimeline: {}, commitmentTimeline: {} },
      children: [],
    };
    const gateway = {
      updateExecutable: vi.fn().mockResolvedValue(undefined),
    } as unknown as WBSGateway;
    const rolesGateway = {
      list: vi
        .fn()
        .mockResolvedValue({ items: [], page: 1, pageSize: 100, total: 0 }),
    } as unknown as RolesGateway;
    const membersGateway = {
      list: vi
        .fn()
        .mockResolvedValue({ items: [], page: 1, pageSize: 100, total: 0 }),
    } as unknown as TeamMembersGateway;
    render(
      <WBSDetailDialog
        project={{
          id: "project",
          name: "Alpha",
          status: "open",
          autoCalculateDate: true,
          automaticScheduling: true,
          projectBuffer: 20,
          scheduleVersion: 0,
          priority: 1,
          createdAt: new Date(),
          updatedAt: new Date(),
        }}
        node={node}
        gateway={gateway}
        rolesGateway={rolesGateway}
        membersGateway={membersGateway}
        onClose={() => undefined}
        onChanged={() => undefined}
        onReopened={() => undefined}
      />,
    );
    fireEvent.change(screen.getByLabelText("Name"), {
      target: { value: "Backend API" },
    });
    const effort = screen.getByLabelText("Effort (hours)");
    fireEvent.change(effort, { target: { value: "6.6666" } });
    expect((effort as HTMLInputElement).value).toBe("6.6666");
    fireEvent.blur(effort);
    expect((effort as HTMLInputElement).value).toBe("6.5");
    fireEvent.change(effort, { target: { value: "6.5x" } });
    expect((effort as HTMLInputElement).value).toBe("6.5");
    fireEvent.change(screen.getByLabelText("Lag (days)"), {
      target: { value: "2" },
    });
    expect(
      screen.getByRole("button", {
        name: "Execution timeline: Select start and end date",
      }),
    ).toBeTruthy();
    const capacityAllocation = screen.getByRole("spinbutton", {
      name: "Capacity Allocation (%)",
    }) as HTMLInputElement;
    expect(capacityAllocation.value).toBe("100");
    expect(
      screen.getByText(
        "Maximum planned capacity per day. Remaining capacity may be used by other tasks.",
      ),
    ).toBeTruthy();
    expect(
      screen.getByRole("button", {
        name: "Commitment timeline: Select start and end date",
      }),
    ).toBeTruthy();
    expect(
      screen.getByRole("button", {
        name: "Actual Date: Select start and end date",
      }),
    ).toBeTruthy();
    fireEvent.submit(screen.getByLabelText("Name").closest("form")!);
    await waitFor(() =>
      expect(gateway.updateExecutable).toHaveBeenCalledWith(
        "project",
        "task",
        expect.objectContaining({
          name: "Backend API",
          effortHours: 6.5,
          lagDays: 2,
          capacityAllocationPercentage: 100,
        }),
      ),
    );
    vi.mocked(gateway.updateExecutable).mockClear();
    fireEvent.change(capacityAllocation, { target: { value: "0" } });
    fireEvent.submit(screen.getByLabelText("Name").closest("form")!);
    expect(screen.getByRole("alert").textContent).toContain(
      "whole number from 1 to 100",
    );
    expect(gateway.updateExecutable).not.toHaveBeenCalled();
  });

  it("US-6.1 AC-2 AC-27 validates Lag and exposes generated unscheduled state accessibly", async () => {
    const node: WBSNode = {
      id: "task",
      projectId: "project",
      name: "API",
      position: 1,
      hasChildren: false,
      executable: {
        lagDays: 0,
        executionTimeline: {},
        commitmentTimeline: {},
        executionUnscheduledReason: "Missing assignee",
        commitmentUnscheduledReason: "Zero commitment capacity",
      },
      children: [],
    };
    const gateway = {
      updateExecutable: vi.fn().mockResolvedValue(undefined),
    } as unknown as WBSGateway;
    const rolesGateway = {
      list: vi
        .fn()
        .mockResolvedValue({ items: [], page: 1, pageSize: 100, total: 0 }),
    } as unknown as RolesGateway;
    const membersGateway = {
      list: vi
        .fn()
        .mockResolvedValue({ items: [], page: 1, pageSize: 100, total: 0 }),
    } as unknown as TeamMembersGateway;

    render(
      <WBSDetailDialog
        project={{
          id: "project",
          name: "Alpha",
          status: "open",
          autoCalculateDate: true,
          automaticScheduling: true,
          projectBuffer: 20,
          scheduleVersion: 0,
          priority: 1,
          createdAt: new Date(),
          updatedAt: new Date(),
        }}
        node={node}
        gateway={gateway}
        rolesGateway={rolesGateway}
        membersGateway={membersGateway}
        onClose={() => undefined}
        onChanged={() => undefined}
        onReopened={() => undefined}
      />,
    );

    const execution = screen.getByRole("button", {
      name: "Execution timeline: Select start and end date",
    });
    const commitment = screen.getByRole("button", {
      name: "Commitment timeline: Select start and end date",
    });
    expect((execution as HTMLButtonElement).disabled).toBe(true);
    expect((commitment as HTMLButtonElement).disabled).toBe(true);
    const status = screen.getByRole("status", {
      name: "Automatic schedule status",
    });
    expect(status.textContent).toContain("Missing assignee");
    expect(status.textContent).toContain("Zero commitment capacity");

    fireEvent.change(screen.getByLabelText("Lag (days)"), {
      target: { value: "" },
    });
    fireEvent.submit(screen.getByLabelText("Name").closest("form")!);
    expect(
      await screen.findByText(
        "Lag must be a non-negative whole number of days.",
      ),
    ).not.toBeNull();
    expect(gateway.updateExecutable).not.toHaveBeenCalled();
  });

  it("previews generated dates on blur before Save without persisting the draft", async () => {
    const node: WBSNode = {
      id: "task",
      projectId: "project",
      name: "Build API",
      position: 1,
      hasChildren: false,
      executable: {
        roleId: "role",
        assigneeId: "member",
        effortMinutes: 480,
        lagDays: 0,
        executionTimeline: { start: "2026-08-01", end: "2026-08-01" },
        commitmentTimeline: { start: "2026-08-01", end: "2026-08-02" },
      },
      children: [],
    };
    const preview: SchedulePreview = {
      task: {
        ...node,
        executable: {
          ...node.executable,
          effortMinutes: 600,
          executionTimeline: { start: "2026-08-03", end: "2026-08-04" },
          commitmentTimeline: { start: "2026-08-03", end: "2026-08-05" },
        },
      },
    };
    const updateExecutable = vi.fn().mockResolvedValue(undefined);
    const gateway = {
      previewExecutableSchedule: vi.fn().mockResolvedValue(preview),
      updateExecutable,
    } as unknown as WBSGateway;
    const dependenciesGateway = {
      list: vi.fn().mockResolvedValue({ blockedBy: [], blocks: [] }),
      candidates: vi.fn(),
      create: vi.fn(),
      remove: vi.fn(),
      keepAsManual: vi.fn(),
      invalidateTask: vi.fn(),
    } as unknown as DependenciesGateway;
    const rolesGateway = {
      list: vi.fn().mockResolvedValue({
        items: [
          {
            id: "role",
            name: "Backend",
            createdAt: new Date(),
            updatedAt: new Date(),
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
            name: "Harry",
            role: { id: "role", name: "Backend" },
            dailyCapacity: 8,
            bufferPercentage: 20,
            baseExecutionCapacity: 6.5,
            createdAt: new Date(),
            updatedAt: new Date(),
          },
        ],
        page: 1,
        pageSize: 100,
        total: 1,
      }),
    } as unknown as TeamMembersGateway;

    render(
      <WBSDetailDialog
        project={{
          id: "project",
          name: "Alpha",
          status: "open",
          autoCalculateDate: true,
          automaticScheduling: true,
          projectBuffer: 20,
          scheduleVersion: 0,
          priority: 1,
          createdAt: new Date(),
          updatedAt: new Date(),
        }}
        node={node}
        gateway={gateway}
        dependenciesGateway={dependenciesGateway}
        rolesGateway={rolesGateway}
        membersGateway={membersGateway}
        onClose={() => undefined}
        onChanged={() => undefined}
        onReopened={() => undefined}
      />,
    );

    await screen.findByRole("option", { name: "Harry" });
    const effort = screen.getByLabelText("Effort (hours)");
    fireEvent.change(effort, { target: { value: "10" } });
    expect(gateway.previewExecutableSchedule).not.toHaveBeenCalled();
    expect(gateway.updateExecutable).not.toHaveBeenCalled();

    fireEvent.blur(effort);

    await waitFor(() =>
      expect(gateway.previewExecutableSchedule).toHaveBeenCalledWith(
        "project",
        "task",
        {
          roleId: "role",
          assigneeId: "member",
          effortHours: 10,
          lagDays: 0,
          capacityAllocationPercentage: 100,
        },
        expect.any(AbortSignal),
      ),
    );
    expect(gateway.updateExecutable).not.toHaveBeenCalled();
    const status = screen.getByRole("status", {
      name: "Automatic schedule status",
    });
    await waitFor(() => {
      expect(status.textContent).toContain(formatDateOnly("2026-08-03"));
      expect(status.textContent).toContain(formatDateOnly("2026-08-05"));
    });
    expect(status.textContent).toContain("Unconfirmed schedule preview");
    expect(status.textContent).toContain("Save confirms the draft");

    fireEvent.submit(screen.getByLabelText("Name").closest("form")!);
    await waitFor(() =>
      expect(gateway.updateExecutable).toHaveBeenCalledOnce(),
    );
    const confirmedInput = updateExecutable.mock.calls[0]?.[2];
    if (!confirmedInput) throw new Error("confirmed input was not captured");
    expect(confirmedInput).toEqual(
      expect.objectContaining({
        name: "Build API",
        roleId: "role",
        assigneeId: "member",
        effortHours: 10,
        lagDays: 0,
      }),
    );
    expect(confirmedInput.executionStart).toBeUndefined();
    expect(confirmedInput.executionEnd).toBeUndefined();
    expect(confirmedInput.commitmentStart).toBeUndefined();
    expect(confirmedInput.commitmentEnd).toBeUndefined();
  });

  it("US-6.1 AC-38 skips a new blur preview when Save is the focus target", async () => {
    const user = userEvent.setup();
    const node = recommendationNode();
    const previewExecutableSchedule = vi.fn().mockResolvedValue({ task: node });
    const updateExecutable = vi.fn().mockResolvedValue(undefined);
    const gateway = recommendationGateway(node, {
      previewExecutableSchedule,
      updateExecutable,
    });

    renderRecommendationDialog(node, gateway);
    await screen.findByRole("option", { name: "Ayu" });

    const effort = screen.getByLabelText("Effort (hours)");
    await user.clear(effort);
    await user.type(effort, "10");
    expect(document.activeElement).toBe(effort);

    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => expect(updateExecutable).toHaveBeenCalledOnce());
    expect(previewExecutableSchedule).not.toHaveBeenCalled();
    expect(updateExecutable).toHaveBeenCalledWith(
      "project",
      "task",
      expect.objectContaining({
        effortHours: 10,
        roleId: "role",
        assigneeId: "a",
      }),
    );
  });

  it("ignores an older schedule preview that resolves after a newer draft", async () => {
    const node: WBSNode = {
      id: "task",
      projectId: "project",
      name: "Build API",
      position: 1,
      hasChildren: false,
      executable: {
        roleId: "role",
        assigneeId: "member",
        effortMinutes: 480,
        lagDays: 0,
        executionTimeline: { start: "2026-08-01", end: "2026-08-01" },
        commitmentTimeline: { start: "2026-08-01", end: "2026-08-02" },
      },
      children: [],
    };
    const first = deferred<SchedulePreview>();
    const second = deferred<SchedulePreview>();
    const previewSignals: Array<AbortSignal | undefined> = [];
    let previewRequest = 0;
    const previewExecutableSchedule = vi.fn(
      (
        _projectId: string,
        _id: string,
        _input: unknown,
        signal?: AbortSignal,
      ): Promise<SchedulePreview> => {
        previewSignals.push(signal);
        previewRequest += 1;
        return previewRequest === 1 ? first.promise : second.promise;
      },
    );
    const gateway = {
      previewExecutableSchedule,
      updateExecutable: vi.fn(),
    } as unknown as WBSGateway;
    const rolesGateway = {
      list: vi.fn().mockResolvedValue({
        items: [
          {
            id: "role",
            name: "Backend",
            createdAt: new Date(),
            updatedAt: new Date(),
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
            name: "Harry",
            role: { id: "role", name: "Backend" },
            dailyCapacity: 8,
            bufferPercentage: 20,
            baseExecutionCapacity: 6.5,
            createdAt: new Date(),
            updatedAt: new Date(),
          },
        ],
        page: 1,
        pageSize: 100,
        total: 1,
      }),
    } as unknown as TeamMembersGateway;

    render(
      <WBSDetailDialog
        project={{
          id: "project",
          name: "Alpha",
          status: "open",
          autoCalculateDate: true,
          automaticScheduling: true,
          projectBuffer: 20,
          scheduleVersion: 0,
          priority: 1,
          createdAt: new Date(),
          updatedAt: new Date(),
        }}
        node={node}
        gateway={gateway}
        rolesGateway={rolesGateway}
        membersGateway={membersGateway}
        onClose={() => undefined}
        onChanged={() => undefined}
        onReopened={() => undefined}
      />,
    );

    await screen.findByRole("option", { name: "Harry" });
    const effort = screen.getByLabelText("Effort (hours)");

    fireEvent.change(effort, { target: { value: "9" } });
    fireEvent.blur(effort);
    await waitFor(() =>
      expect(previewExecutableSchedule).toHaveBeenCalledTimes(1),
    );

    fireEvent.change(effort, { target: { value: "10" } });
    fireEvent.blur(effort);
    await waitFor(() =>
      expect(previewExecutableSchedule).toHaveBeenCalledTimes(2),
    );

    expect(previewSignals[0]?.aborted).toBe(true);

    second.resolve({
      task: {
        ...node,
        executable: {
          ...node.executable,
          effortMinutes: 600,
          executionTimeline: { start: "2026-08-10", end: "2026-08-11" },
          commitmentTimeline: { start: "2026-08-10", end: "2026-08-12" },
        },
      },
    });
    const status = screen.getByRole("status", {
      name: "Automatic schedule status",
    });
    await waitFor(() =>
      expect(status.textContent).toContain(formatDateOnly("2026-08-12")),
    );

    first.resolve({
      task: {
        ...node,
        executable: {
          ...node.executable,
          effortMinutes: 540,
          executionTimeline: { start: "2026-08-03", end: "2026-08-04" },
          commitmentTimeline: { start: "2026-08-03", end: "2026-08-05" },
        },
      },
    });
    await waitFor(() => {
      expect(status.textContent).toContain(formatDateOnly("2026-08-12"));
      expect(status.textContent).not.toContain(formatDateOnly("2026-08-05"));
    });
  });

  it("preserves manual dependency and returns an unscheduled preview when Assignee is cleared", async () => {
    const missingAssigneeReason =
      "Task requires an Assignee before it can be scheduled.";
    const node: WBSNode = {
      id: "task",
      projectId: "project",
      name: "Build API",
      position: 2,
      hasChildren: false,
      executable: {
        roleId: "role",
        assigneeId: "member",
        effortMinutes: 480,
        lagDays: 0,
        executionTimeline: { start: "2026-08-01", end: "2026-08-01" },
        commitmentTimeline: { start: "2026-08-01", end: "2026-08-02" },
      },
      children: [],
    };
    const gateway = {
      previewExecutableSchedule: vi.fn().mockResolvedValue({
        task: {
          ...node,
          executable: {
            ...node.executable,
            assigneeId: undefined,
            executionTimeline: {},
            commitmentTimeline: {},
            executionUnscheduledReason: missingAssigneeReason,
            commitmentUnscheduledReason: missingAssigneeReason,
          },
        },
      } satisfies SchedulePreview),
      updateExecutable: vi.fn(),
    } as unknown as WBSGateway;
    const dependenciesGateway = {
      list: vi.fn().mockResolvedValue({
        blockedBy: [
          {
            id: "automatic-1",
            source: "automatic",
            manualRemovable: false,
            task: {
              id: "task-1",
              name: "Task 1",
              projectId: "project",
              projectName: "Alpha",
              hierarchyPath: "",
              completed: false,
            },
          },
        ],
        blocks: [],
      }),
      candidates: vi.fn(),
      create: vi.fn(),
      remove: vi.fn(),
      keepAsManual: vi.fn(),
      invalidateTask: vi.fn(),
    } as unknown as DependenciesGateway;
    const rolesGateway = {
      list: vi.fn().mockResolvedValue({
        items: [
          {
            id: "role",
            name: "Backend",
            createdAt: new Date(),
            updatedAt: new Date(),
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
            name: "Harry",
            role: { id: "role", name: "Backend" },
            dailyCapacity: 8,
            bufferPercentage: 20,
            baseExecutionCapacity: 6.5,
            createdAt: new Date(),
            updatedAt: new Date(),
          },
        ],
        page: 1,
        pageSize: 100,
        total: 1,
      }),
    } as unknown as TeamMembersGateway;

    render(
      <WBSDetailDialog
        project={{
          id: "project",
          name: "Alpha",
          status: "open",
          autoCalculateDate: true,
          automaticScheduling: true,
          projectBuffer: 20,
          scheduleVersion: 0,
          priority: 1,
          createdAt: new Date(),
          updatedAt: new Date(),
        }}
        node={node}
        gateway={gateway}
        dependenciesGateway={dependenciesGateway}
        rolesGateway={rolesGateway}
        membersGateway={membersGateway}
        onClose={() => undefined}
        onChanged={() => undefined}
        onReopened={() => undefined}
      />,
    );

    const dependencies = await screen.findByRole("region", {
      name: "Dependencies",
    });
    expect(within(dependencies).getByText("Task 1")).not.toBeNull();

    const assignee = await screen.findByLabelText("Assignee");
    fireEvent.change(assignee, { target: { value: "" } });
    fireEvent.blur(assignee);

    await waitFor(() =>
      expect(gateway.previewExecutableSchedule).toHaveBeenCalledWith(
        "project",
        "task",
        {
          roleId: "role",
          assigneeId: undefined,
          effortHours: 8,
          lagDays: 0,
          capacityAllocationPercentage: 100,
        },
        expect.any(AbortSignal),
      ),
    );
    const status = screen.getByRole("status", {
      name: "Automatic schedule status",
    });
    await waitFor(() => {
      expect(status.textContent).toContain(missingAssigneeReason);
      expect(status.textContent).not.toContain(formatDateOnly("2026-08-01"));
      expect(within(dependencies).getByText("Task 1")).not.toBeNull();
    });
  });
});

describe("WBSDetailDialog assignee recommendation", () => {
  it("D02 AC2 renders Role and Assignee immediately after Lag and before timeline and dependency controls", async () => {
    const node = recommendationNode();
    const gateway = recommendationGateway(node);
    const dependenciesGateway = {
      list: vi.fn().mockResolvedValue({ blockedBy: [], blocks: [] }),
      candidates: vi.fn().mockResolvedValue([]),
      create: vi.fn(),
      remove: vi.fn(),
      keepAsManual: vi.fn(),
      invalidateTask: vi.fn(),
    } as unknown as DependenciesGateway;
    renderRecommendationDialog(node, gateway, { dependenciesGateway });

    const dependencies = await screen.findByRole("region", {
      name: "Dependencies",
    });
    const form = screen.getByLabelText("Name").closest("form");
    if (!form) throw new Error("Task form was not rendered");
    const controls = Array.from(
      form.querySelectorAll<
        HTMLInputElement | HTMLSelectElement | HTMLButtonElement
      >("input, select, button"),
    );
    const indexOf = (element: Element) =>
      controls.indexOf(
        element as HTMLInputElement | HTMLSelectElement | HTMLButtonElement,
      );
    const effort = screen.getByLabelText("Effort (hours)");
    const percentage = screen.getByLabelText("Capacity Allocation (%)");
    const lag = screen.getByLabelText("Lag (days)");
    const execution = screen.getByRole("button", {
      name: /Execution timeline:/,
    });
    const commitment = screen.getByRole("button", {
      name: /Commitment timeline:/,
    });
    const role = screen.getByLabelText("Role");
    const assignee = screen.getByLabelText("Assignee");
    const save = screen.getByRole("button", { name: "Save" });

    expect(indexOf(effort)).toBeLessThan(indexOf(percentage));
    expect(indexOf(percentage)).toBeLessThan(indexOf(lag));
    expect(indexOf(lag) + 1).toBe(indexOf(role));
    expect(indexOf(role) + 1).toBe(indexOf(assignee));
    expect(indexOf(assignee)).toBeLessThan(indexOf(execution));
    expect(indexOf(execution)).toBeLessThan(indexOf(commitment));
    expect(
      commitment.compareDocumentPosition(dependencies) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).not.toBe(0);
    expect(indexOf(commitment)).toBeLessThan(indexOf(save));
  });

  it("D03 AC3 keeps alphabetical options and skips API while prerequisites are incomplete", async () => {
    const node = recommendationNode({
      roleId: undefined,
      effortMinutes: undefined,
    });
    const recommendAssignees = vi.fn();
    const gateway = recommendationGateway(node, { recommendAssignees });
    renderRecommendationDialog(node, gateway);

    const assignee = await screen.findByLabelText("Assignee");
    fireEvent.focus(assignee);
    expect(recommendAssignees).not.toHaveBeenCalled();
    expect(
      screen.getByText(/Select Role and enter valid Effort/).textContent,
    ).toContain("Candidates remain alphabetical");
    const options = within(assignee).getAllByRole("option");
    expect(options.map((option) => option.textContent)).toEqual([
      "No assignee",
      "Ayu",
      "Bima",
      "Dewi",
      "Fajar",
    ]);
  });

  it("D01 D09 D15 AC12 AC13 AC14 AC15 AC16 AC23 AC25 sends one batch, renders metadata, and preserves percentage", async () => {
    const node = recommendationNode({
      assigneeId: undefined,
      capacityAllocationPercentage: 40,
    });
    const recommendAssignees = vi
      .fn()
      .mockResolvedValue(
        recommendationResult("automatic", [
          recommendationItem("b", "Bima", "feasible", "2026-08-11", 240, 0),
          recommendationItem("a", "Ayu", "overcapacity", "2026-08-10", 0, 120),
          recommendationItem("d", "Dewi", "no-completion"),
        ]),
      );
    const gateway = recommendationGateway(node, { recommendAssignees });
    renderRecommendationDialog(node, gateway);

    const assignee = await screen.findByLabelText("Assignee");
    fireEvent.focus(assignee);
    await waitFor(() => expect(recommendAssignees).toHaveBeenCalledTimes(1));
    expect(recommendAssignees).toHaveBeenCalledWith(
      "project",
      "task",
      {
        roleId: "role",
        effortHours: 8,
        lagDays: 0,
        capacityAllocationPercentage: 40,
        executionStart: undefined,
      },
      expect.any(AbortSignal),
    );
    expect(
      await within(assignee).findByRole("option", {
        name: `Bima — finishes ${formatDateOnly("2026-08-11")}, 4h remaining`,
      }),
    ).not.toBeNull();
    expect(
      within(assignee).getByRole("option", {
        name: `Ayu — finishes ${formatDateOnly("2026-08-10")}, adds 2h overcapacity`,
      }),
    ).not.toBeNull();
    expect(
      within(assignee).getByRole("option", {
        name: "Dewi — no projected completion",
      }),
    ).not.toBeNull();

    const percentage = screen.getByLabelText(
      "Capacity Allocation (%)",
    ) as HTMLInputElement;
    fireEvent.change(assignee, { target: { value: "b" } });
    expect(percentage.value).toBe("40");
    expect(
      within(assignee).getByRole("option", {
        name: `Bima — finishes ${formatDateOnly("2026-08-11")}, 4h remaining`,
      }),
    ).not.toBeNull();
    fireEvent.change(assignee, { target: { value: "" } });
    expect(percentage.value).toBe("40");
    expect(recommendAssignees).toHaveBeenCalledTimes(1);
  });

  it("D08 D10 D15 AC8 AC10 AC25 labels manual recommendations as advisory and ignores Manual Execution End", async () => {
    const node = recommendationNode({
      executionTimeline: { start: "2026-08-10", end: "2026-08-10" },
    });
    const recommendAssignees = vi
      .fn()
      .mockResolvedValue(
        recommendationResult("manual-advisory", [
          recommendationItem("b", "Bima", "feasible", "2026-08-11", 240, 0),
          recommendationItem("a", "Ayu", "feasible", "2026-08-12", 120, 0),
        ]),
      );
    const gateway = recommendationGateway(node, { recommendAssignees });
    renderRecommendationDialog(node, gateway, { automaticScheduling: false });

    const assignee = await screen.findByLabelText("Assignee");
    fireEvent.focus(assignee);

    await waitFor(() => expect(recommendAssignees).toHaveBeenCalledTimes(1));
    expect(recommendAssignees).toHaveBeenCalledWith(
      "project",
      "task",
      expect.objectContaining({
        executionStart: "2026-08-10",
      }),
      expect.any(AbortSignal),
    );
    expect(
      await within(assignee).findByRole("option", {
        name: `Bima — estimated ${formatDateOnly("2026-08-11")}, 4h remaining`,
      }),
    ).not.toBeNull();
    expect(
      screen.getByText(/Advisory estimates use confirmed scheduling state/),
    ).not.toBeNull();
    expect(screen.queryByText(/manual end|exceeds/i)).toBeNull();
  });

  it("D07 AC7 keeps candidates alphabetical when Automatic Scheduling has no anchor", async () => {
    const node = recommendationNode({ assigneeId: "b" });
    const missingAnchorItems = [
      recommendationItem("b", "Bima", "no-completion"),
      recommendationItem("a", "Ayu", "no-completion"),
      recommendationItem("d", "Dewi", "no-completion"),
    ].map((item) => ({
      ...item,
      reasonCode: "AUTOMATIC_ANCHOR_MISSING",
    }));
    const recommendAssignees = vi
      .fn()
      .mockResolvedValue(recommendationResult("automatic", missingAnchorItems));
    const gateway = recommendationGateway(node, { recommendAssignees });
    renderRecommendationDialog(node, gateway);

    const assignee = (await screen.findByLabelText(
      "Assignee",
    )) as HTMLSelectElement;
    fireEvent.focus(assignee);

    await screen.findByText(
      /Automatic Scheduling needs an effective Scheduling Start Date/,
    );
    expect(
      within(assignee)
        .getAllByRole("option")
        .map((option) => option.textContent),
    ).toEqual(["No assignee", "Ayu", "Bima", "Dewi"]);
    expect(assignee.value).toBe("b");
    expect(
      within(assignee).queryByRole("option", {
        name: /no projected completion/,
      }),
    ).toBeNull();
  });

  it("D09 D14 AC24 preserves a mismatched current Assignee visibly and last", async () => {
    const node = recommendationNode({
      assigneeId: "a",
      capacityAllocationPercentage: 40,
    });
    const gateway = recommendationGateway(node);
    renderRecommendationDialog(node, gateway, { includeSecondRole: true });

    await screen.findByRole("option", { name: "Ayu" });
    fireEvent.change(screen.getByLabelText("Role"), {
      target: { value: "other-role" },
    });

    const assignee = screen.getByLabelText("Assignee") as HTMLSelectElement;
    expect(assignee.value).toBe("a");
    expect(
      (screen.getByLabelText("Capacity Allocation (%)") as HTMLInputElement)
        .valueAsNumber,
    ).toBe(40);
    const options = within(assignee).getAllByRole("option");
    expect(options.at(-1)?.textContent).toBe(
      "Ayu — does not match selected Role",
    );
    expect(
      screen.getByText(/Current Assignee does not match selected Role/)
        .textContent,
    ).toContain("Replace or clear it before Save");
  });

  it("D16 AC26 AC28 disables stale rows and ignores an older response", async () => {
    const node = recommendationNode();
    const first = deferred<AssigneeRecommendationResult>();
    const second = deferred<AssigneeRecommendationResult>();
    const signals: AbortSignal[] = [];
    const recommendAssignees = vi
      .fn()
      .mockImplementation(
        (
          _projectID: string,
          _taskID: string,
          _input: unknown,
          signal: AbortSignal,
        ) => {
          signals.push(signal);
          return signals.length === 1 ? first.promise : second.promise;
        },
      );
    const gateway = recommendationGateway(node, { recommendAssignees });
    renderRecommendationDialog(node, gateway);

    const assignee = await screen.findByLabelText("Assignee");
    fireEvent.focus(assignee);
    await waitFor(() => expect(recommendAssignees).toHaveBeenCalledTimes(1));
    expect(
      (
        within(assignee).getByRole("option", {
          name: "Ayu",
        }) as HTMLOptionElement
      ).disabled,
    ).toBe(true);

    fireEvent.blur(assignee);
    fireEvent.change(screen.getByLabelText("Effort (hours)"), {
      target: { value: "10" },
    });
    fireEvent.focus(assignee);
    await waitFor(() => expect(recommendAssignees).toHaveBeenCalledTimes(2));
    expect(signals[0]?.aborted).toBe(true);

    second.resolve(
      recommendationResult("automatic", [
        recommendationItem("d", "Dewi", "feasible", "2026-08-09", 180, 0),
        recommendationItem("a", "Ayu", "feasible", "2026-08-10", 120, 0),
      ]),
    );
    await within(assignee).findByRole("option", {
      name: `Dewi — finishes ${formatDateOnly("2026-08-09")}, 3h remaining`,
    });

    first.resolve(
      recommendationResult("automatic", [
        recommendationItem("a", "Ayu", "feasible", "2026-08-01", 480, 0),
      ]),
    );
    await waitFor(() => {
      expect(
        within(assignee).queryByRole("option", {
          name: `Ayu — finishes ${formatDateOnly("2026-08-01")}, 8h remaining`,
        }),
      ).toBeNull();
      expect(
        within(assignee).getByRole("option", {
          name: `Dewi — finishes ${formatDateOnly("2026-08-09")}, 3h remaining`,
        }),
      ).not.toBeNull();
    });
  });

  it("D16 AC27 freezes visible order across a confirmed mutation until close and reopen", async () => {
    const node = recommendationNode();
    let confirmedChange: (() => void) | undefined;
    const recommendAssignees = vi
      .fn()
      .mockResolvedValueOnce(
        recommendationResult("automatic", [
          recommendationItem("b", "Bima", "feasible", "2026-08-10", 240, 0),
          recommendationItem("a", "Ayu", "feasible", "2026-08-11", 120, 0),
        ]),
      )
      .mockResolvedValueOnce(
        recommendationResult("automatic", [
          recommendationItem("a", "Ayu", "feasible", "2026-08-09", 300, 0),
          recommendationItem("b", "Bima", "feasible", "2026-08-12", 60, 0),
        ]),
      );
    const gateway = recommendationGateway(node, {
      recommendAssignees,
      subscribeToConfirmedChanges: (listener: () => void) => {
        confirmedChange = listener;
        return () => undefined;
      },
    });
    renderRecommendationDialog(node, gateway);

    const assignee = await screen.findByLabelText("Assignee");
    fireEvent.focus(assignee);
    await within(assignee).findByRole("option", {
      name: `Bima — finishes ${formatDateOnly("2026-08-10")}, 4h remaining`,
    });
    act(() => confirmedChange?.());
    expect(
      within(assignee)
        .getAllByRole("option")
        .map((option) => option.textContent),
    ).toEqual([
      "No assignee",
      `Bima — finishes ${formatDateOnly("2026-08-10")}, 4h remaining`,
      `Ayu — finishes ${formatDateOnly("2026-08-11")}, 2h remaining`,
    ]);
    expect(screen.getByText(/Close and reopen it/)).not.toBeNull();
    expect(
      (
        within(assignee).getByRole("option", {
          name: `Bima — finishes ${formatDateOnly("2026-08-10")}, 4h remaining`,
        }) as HTMLOptionElement
      ).disabled,
    ).toBe(true);

    fireEvent.blur(assignee);
    fireEvent.focus(assignee);
    await waitFor(() => expect(recommendAssignees).toHaveBeenCalledTimes(2));
    expect(
      await within(assignee).findByRole("option", {
        name: `Ayu — finishes ${formatDateOnly("2026-08-09")}, 5h remaining`,
      }),
    ).not.toBeNull();
  });

  it("D16 D17 AC31 falls back alphabetically, preserves selection, and retries latest batch", async () => {
    const node = recommendationNode({ assigneeId: "b" });
    const recommendAssignees = vi
      .fn()
      .mockRejectedValueOnce(new Error("unavailable"))
      .mockResolvedValueOnce(
        recommendationResult("automatic", [
          recommendationItem("d", "Dewi", "feasible", "2026-08-09", 240, 0),
          recommendationItem("b", "Bima", "feasible", "2026-08-10", 120, 0),
          recommendationItem("a", "Ayu", "feasible", "2026-08-11", 60, 0),
        ]),
      );
    const gateway = recommendationGateway(node, { recommendAssignees });
    renderRecommendationDialog(node, gateway);

    const assignee = (await screen.findByLabelText(
      "Assignee",
    )) as HTMLSelectElement;
    fireEvent.focus(assignee);
    await screen.findByText(
      "Assignee recommendation is temporarily unavailable.",
    );
    expect(assignee.value).toBe("b");
    expect(
      within(assignee)
        .getAllByRole("option")
        .map((option) => option.textContent),
    ).toEqual(["No assignee", "Ayu", "Bima", "Dewi"]);

    fireEvent.click(
      screen.getByRole("button", { name: "Retry recommendations" }),
    );
    await waitFor(() => expect(recommendAssignees).toHaveBeenCalledTimes(2));
    expect(
      await within(assignee).findByRole("option", {
        name: `Dewi — finishes ${formatDateOnly("2026-08-09")}, 4h remaining`,
      }),
    ).not.toBeNull();
    expect(assignee.value).toBe("b");
  });
});

function recommendationNode(
  executable: Partial<WBSNode["executable"]> = {},
): WBSNode {
  return {
    id: "task",
    projectId: "project",
    name: "Build API",
    position: 1,
    hasChildren: false,
    executable: {
      roleId: "role",
      assigneeId: "a",
      effortMinutes: 480,
      lagDays: 0,
      capacityAllocationPercentage: 40,
      executionTimeline: {},
      commitmentTimeline: {},
      ...executable,
    },
    children: [],
  };
}

function recommendationGateway(
  node: WBSNode,
  overrides: Partial<WBSGateway> = {},
): WBSGateway {
  return {
    updateExecutable: vi.fn().mockResolvedValue(undefined),
    previewExecutableSchedule: vi.fn().mockResolvedValue({ task: node }),
    ...overrides,
  } as unknown as WBSGateway;
}

function renderRecommendationDialog(
  node: WBSNode,
  gateway: WBSGateway,
  options: {
    dependenciesGateway?: DependenciesGateway;
    includeSecondRole?: boolean;
    automaticScheduling?: boolean;
  } = {},
) {
  const roles = [
    {
      id: "role",
      name: "Backend",
      createdAt: new Date(),
      updatedAt: new Date(),
    },
  ];
  if (options.includeSecondRole) {
    roles.push({
      id: "other-role",
      name: "Frontend",
      createdAt: new Date(),
      updatedAt: new Date(),
    });
  }
  const rolesGateway = {
    list: vi.fn().mockResolvedValue({
      items: roles,
      page: 1,
      pageSize: 100,
      total: roles.length,
    }),
  } as unknown as RolesGateway;
  const membersGateway = {
    list: vi.fn().mockResolvedValue({
      items: [
        recommendationMember("b", "Bima", "role", "Backend"),
        recommendationMember("a", "Ayu", "role", "Backend"),
        recommendationMember("d", "Dewi", "role", "Backend"),
        recommendationMember("f", "Fajar", "other-role", "Frontend"),
      ],
      page: 1,
      pageSize: 100,
      total: 4,
    }),
  } as unknown as TeamMembersGateway;
  return render(
    <WBSDetailDialog
      project={{
        id: "project",
        name: "Alpha",
        status: "open",
        autoCalculateDate: true,
        automaticScheduling: options.automaticScheduling ?? true,
        projectBuffer: 20,
        scheduleVersion: 0,
        priority: 1,
        createdAt: new Date(),
        updatedAt: new Date(),
      }}
      node={node}
      gateway={gateway}
      dependenciesGateway={options.dependenciesGateway}
      rolesGateway={rolesGateway}
      membersGateway={membersGateway}
      onClose={() => undefined}
      onChanged={() => undefined}
      onReopened={() => undefined}
    />,
  );
}

function recommendationMember(
  id: string,
  name: string,
  roleID: string,
  roleName: string,
) {
  return {
    id,
    name,
    role: { id: roleID, name: roleName },
    dailyCapacity: 8,
    bufferPercentage: 0,
    baseExecutionCapacity: 8,
    createdAt: new Date(),
    updatedAt: new Date(),
  };
}

function recommendationResult(
  mode: AssigneeRecommendationResult["mode"],
  items: AssigneeRecommendationResult["items"],
): AssigneeRecommendationResult {
  return {
    calculatedOnDate: "2026-08-05",
    snapshot: { projectScheduleVersions: { project: 12 } },
    mode,
    items,
  };
}

function recommendationItem(
  memberId: string,
  memberName: string,
  rankGroup: AssigneeRecommendationResult["items"][number]["rankGroup"],
  executionEnd?: string,
  remainingExecutionCapacityMinutes = 0,
  incrementalOvercapacityMinutes = 0,
): AssigneeRecommendationResult["items"][number] {
  return {
    memberId,
    memberName,
    roleId: "role",
    rankGroup,
    executionEnd,
    remainingExecutionCapacityHours: remainingExecutionCapacityMinutes / 60,
    incrementalOvercapacityHours: incrementalOvercapacityMinutes / 60,
    reasonCode:
      rankGroup === "no-completion" ? "NO_POSITIVE_CAPACITY" : undefined,
  };
}

function deferred<T>(): {
  promise: Promise<T>;
  resolve(value: T): void;
  reject(reason?: unknown): void;
} {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise;
    reject = rejectPromise;
  });
  return { promise, resolve, reject };
}

function abortable<T>(value: T, signal?: AbortSignal): Promise<T> {
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => resolve(value), 0);
    signal?.addEventListener(
      "abort",
      () => {
        clearTimeout(timer);
        reject(new DOMException("aborted", "AbortError"));
      },
      { once: true },
    );
  });
}

describe("WBSDetailDialog Group scheduling", () => {
  it("US-4.4 AC-5 initializes an override draft from current effective values", async () => {
    const gateway = {
      updateGroupScheduling: vi.fn().mockResolvedValue(groupNode()),
    } as unknown as WBSGateway;
    renderGroupDialog(gateway, groupNode());

    fireEvent.click(
      screen.getByRole("button", { name: "Override scheduling settings" }),
    );
    fireEvent.change(screen.getByLabelText("Name"), {
      target: { value: "Platform Stream" },
    });

    expect(
      screen.queryByRole("button", { name: "Save scheduling" }),
    ).toBeNull();
    expect(screen.queryByRole("button", { name: "Save name" })).toBeNull();
    expect(screen.getByRole("button", { name: "Save" })).toBeTruthy();
    expect(
      screen
        .getByRole("switch", { name: "Group automatic scheduling" })
        .getAttribute("aria-checked"),
    ).toBe("true");
    expect(
      screen.getByRole("button", {
        name: `Scheduling Start Date: ${formatDateOnly("2026-08-10")}`,
      }),
    ).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() =>
      expect(gateway.updateGroupScheduling).toHaveBeenCalledWith(
        "project",
        "group",
        {
          expectedVersion: 0,
          name: "Platform Stream",
          schedulingSource: "override",
          automaticScheduling: true,
          schedulingStartDate: "2026-08-10",
        },
      ),
    );
  });

  it("US-4.4 Section 11.3 reuses OFF-to-ON confirmation before saving", async () => {
    const node = groupNode({
      effectiveAutomaticScheduling: false,
      inheritedAutomaticScheduling: false,
    });
    const gateway = {
      updateGroupScheduling: vi.fn().mockResolvedValue(node),
    } as unknown as WBSGateway;
    renderGroupDialog(gateway, node);

    fireEvent.click(
      screen.getByRole("button", { name: "Override scheduling settings" }),
    );
    fireEvent.click(
      screen.getByRole("switch", { name: "Group automatic scheduling" }),
    );
    fireEvent.click(screen.getByRole("button", { name: "Save" }));

    expect(gateway.updateGroupScheduling).not.toHaveBeenCalled();
    expect(
      screen.getByRole("alertdialog", {
        name: "Enable automatic scheduling for this Group?",
      }),
    ).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Enable and save" }));
    await waitFor(() =>
      expect(gateway.updateGroupScheduling).toHaveBeenCalled(),
    );
  });

  it("US-4.4 AC-7 confirms reset when inherited mode changes Custom OFF to ON", async () => {
    const node = groupNode({
      source: "override",
      automaticScheduling: false,
      schedulingStartDate: undefined,
      effectiveAutomaticScheduling: false,
      inheritedAutomaticScheduling: true,
    });
    const gateway = {
      updateGroupScheduling: vi.fn().mockResolvedValue(node),
    } as unknown as WBSGateway;
    renderGroupDialog(gateway, node);

    fireEvent.click(
      screen.getByRole("button", {
        name: "Use inherited scheduling settings",
      }),
    );
    fireEvent.click(screen.getByRole("button", { name: "Save" }));

    expect(gateway.updateGroupScheduling).not.toHaveBeenCalled();
    expect(
      screen.getByRole("alertdialog", {
        name: "Enable automatic scheduling for this Group?",
      }),
    ).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Enable and save" }));
    await waitFor(() =>
      expect(gateway.updateGroupScheduling).toHaveBeenCalledWith(
        "project",
        "group",
        {
          expectedVersion: 0,
          name: "Platform",
          schedulingSource: "inherit",
          automaticScheduling: undefined,
          schedulingStartDate: undefined,
        },
      ),
    );
  });

  it("US-4.4 AC-35 requires lifecycle confirmation before Lock", async () => {
    const node = groupNode();
    const gateway = {
      changeGroupStatus: vi.fn().mockResolvedValue(node),
    } as unknown as WBSGateway;
    renderGroupDialog(gateway, node);

    fireEvent.click(screen.getByRole("button", { name: "Lock Group" }));
    expect(gateway.changeGroupStatus).not.toHaveBeenCalled();
    expect(
      screen.getByRole("alertdialog", { name: "Lock Group?" }),
    ).toBeTruthy();

    fireEvent.click(
      within(
        screen.getByRole("alertdialog", { name: "Lock Group?" }),
      ).getByRole("button", { name: "Lock Group" }),
    );
    await waitFor(() =>
      expect(gateway.changeGroupStatus).toHaveBeenCalledWith(
        "project",
        "group",
        "locked",
        0,
      ),
    );
  });

  it("US-4.4 AC-36 shows the stronger lock owner and suppresses local lifecycle actions", () => {
    const node = groupNode({
      localStatus: "open",
      effectiveLifecycle: "locked",
      lockOwner: { id: "parent-group", name: "Platform Parent" },
    });
    const gateway = {} as unknown as WBSGateway;
    renderGroupDialog(gateway, node);

    expect(
      screen.getByText(
        "Planning fields are read-only while this scope is locked by Platform Parent.",
      ),
    ).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Lock Group" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Reopen Group" })).toBeNull();
  });
});

function groupNode(
  schedulingOverrides: Partial<NonNullable<WBSNode["scheduling"]>> = {},
): WBSNode {
  return {
    id: "group",
    projectId: "project",
    name: "Platform",
    position: 1,
    hasChildren: true,
    scheduling: {
      version: 0,
      source: "inherit",
      automaticScheduling: undefined,
      schedulingStartDate: undefined,
      localStatus: "open",
      effectiveAutomaticScheduling: true,
      effectiveSchedulingStartDate: "2026-08-10",
      inheritedAutomaticScheduling: true,
      inheritedSchedulingStartDate: "2026-08-10",
      inheritedAutomaticSource: { id: "project", name: "Alpha" },
      inheritedStartDateSource: { id: "project", name: "Alpha" },
      automaticSource: { id: "project", name: "Alpha" },
      startDateSource: { id: "project", name: "Alpha" },
      effectiveLifecycle: "open",
      lockOwner: undefined,
      ...schedulingOverrides,
    },
    executable: {
      lagDays: 0,
      executionTimeline: {},
      commitmentTimeline: {},
    },
    children: [
      {
        id: "task",
        projectId: "project",
        parentId: "group",
        name: "Task",
        position: 1,
        hasChildren: false,
        executable: {
          lagDays: 0,
          effortMinutes: 60,
          executionTimeline: { start: "2026-08-10", end: "2026-08-10" },
          commitmentTimeline: { start: "2026-08-10", end: "2026-08-10" },
        },
        children: [],
      },
    ],
  };
}

function renderGroupDialog(gateway: WBSGateway, node: WBSNode) {
  const rolesGateway = {
    list: vi
      .fn()
      .mockResolvedValue({ items: [], page: 1, pageSize: 100, total: 0 }),
  } as unknown as RolesGateway;
  const membersGateway = {
    list: vi
      .fn()
      .mockResolvedValue({ items: [], page: 1, pageSize: 100, total: 0 }),
  } as unknown as TeamMembersGateway;
  return render(
    <WBSDetailDialog
      project={{
        id: "project",
        name: "Alpha",
        status: "open",
        autoCalculateDate: true,
        automaticScheduling: true,
        schedulingStartDate: "2026-08-10",
        projectBuffer: 20,
        scheduleVersion: 0,
        priority: 1,
        createdAt: new Date(),
        updatedAt: new Date(),
      }}
      node={node}
      gateway={gateway}
      rolesGateway={rolesGateway}
      membersGateway={membersGateway}
      onClose={() => undefined}
      onChanged={() => undefined}
      onReopened={() => undefined}
    />,
  );
}
