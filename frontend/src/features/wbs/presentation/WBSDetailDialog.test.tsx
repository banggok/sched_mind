import { StrictMode } from "react";
import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { DependenciesGateway } from "../../dependencies/application/dependenciesGateway";
import type { RolesGateway } from "../../roles/application/rolesGateway";
import type { TeamMembersGateway } from "../../team-members/application/teamMembersGateway";
import type { SchedulePreview, WBSGateway } from "../application/wbsGateway";
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
