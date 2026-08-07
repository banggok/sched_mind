import { afterEach, describe, expect, it, vi } from "vitest";
import { createHTTPProjectsGateway } from "../../projects/infrastructure/httpProjectsGateway";
import { createHTTPWBSGateway } from "./httpWBSGateway";

afterEach(() => vi.unstubAllGlobals());

describe("HTTP WBS gateway tree cache", () => {
  it("deduplicates Strict Mode consumers into one network request", async () => {
    let resolveResponse: ((value: Response) => void) | undefined;
    const fetchMock = vi.fn(
      () =>
        new Promise<Response>((resolve) => {
          resolveResponse = resolve;
        }),
    );
    vi.stubGlobal("fetch", fetchMock);
    const gateway = createHTTPWBSGateway("/api");
    const firstController = new AbortController();
    const first = gateway.tree("project", firstController.signal);
    const second = gateway.tree("project");
    firstController.abort();
    resolveResponse?.(
      new Response(JSON.stringify({ data: [] }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    await expect(first).rejects.toMatchObject({ name: "AbortError" });
    await expect(second).resolves.toEqual([]);
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("returns the confirmed created node identity and invalidates the tree", async () => {
    const fetchMock = vi.fn(
      (_url: string | URL | Request, init?: RequestInit) =>
        Promise.resolve(
          init?.method
            ? new Response(
                JSON.stringify({ data: taskResponse(null, "new-task") }),
                {
                  status: 201,
                  headers: { "Content-Type": "application/json" },
                },
              )
            : new Response(JSON.stringify({ data: [] }), {
                status: 200,
                headers: { "Content-Type": "application/json" },
              }),
        ),
    );
    vi.stubGlobal("fetch", fetchMock);
    const gateway = createHTTPWBSGateway("/api");
    const confirmedChange = vi.fn();
    expect(gateway.subscribeToConfirmedChanges).toBeTypeOf("function");
    const unsubscribe = gateway.subscribeToConfirmedChanges?.(confirmedChange);
    await gateway.tree("project");
    await gateway.tree("project");
    expect(fetchMock).toHaveBeenCalledTimes(1);
    const created = await gateway.create("project", undefined, "Task", false);
    expect(created.id).toBe("new-task");
    expect(created.projectId).toBe("project");
    expect(created.name).toBe("Build API");
    expect(created.parentId).toBeUndefined();
    expect(created.executable.executionTimeline).toEqual({
      start: "2026-08-01",
      end: "2026-08-03",
    });
    expect(confirmedChange).toHaveBeenCalledTimes(1);
    await gateway.tree("project");
    expect(fetchMock).toHaveBeenCalledTimes(3);
    unsubscribe?.();
  });

  it("preserves the confirmed parent identity for a newly created child", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          data: { ...taskResponse(null, "new-child"), parentId: "group" },
        }),
        {
          status: 201,
          headers: { "Content-Type": "application/json" },
        },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);
    const gateway = createHTTPWBSGateway("/api");

    const created = await gateway.create("project", "group", "Child", false);

    expect(created.id).toBe("new-child");
    expect(created.parentId).toBe("group");
  });

  it("invalidates cached projections after confirmed create even when the response is malformed", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ data: [taskResponse()] }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      )
      .mockResolvedValueOnce(
        new Response("not-json", {
          status: 201,
          headers: { "Content-Type": "application/json" },
        }),
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ data: [] }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      );
    vi.stubGlobal("fetch", fetchMock);
    const gateway = createHTTPWBSGateway("/api");
    const confirmedChange = vi.fn();
    const unsubscribe = gateway.subscribeToConfirmedChanges?.(confirmedChange);

    await gateway.tree("project");
    await expect(
      gateway.create("project", undefined, "Task", false),
    ).rejects.toBeInstanceOf(Error);
    await expect(gateway.tree("project")).resolves.toEqual([]);

    expect(fetchMock).toHaveBeenCalledTimes(3);
    expect(confirmedChange).toHaveBeenCalledTimes(1);
    unsubscribe?.();
  });

  it("US-6.1 AC-2 AC-27 sends Task Lag through the API lag field", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetchMock);
    const gateway = createHTTPWBSGateway("/api");

    await gateway.updateExecutable("project", "task", {
      name: "Build API",
      effortHours: 8,
      lagDays: 3,
    });

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/projects/project/wbs/task/executable",
      expect.objectContaining({ method: "PUT" }),
    );
    const init = fetchMock.mock.calls[0]?.[1] as RequestInit;
    expect(JSON.parse(String(init.body))).toEqual({
      name: "Build API",
      effortHours: 8,
      lag: 3,
    });
    expect(String(init.body)).not.toContain("lagDays");
  });

  it("US-4.4 AC-5 AC-7 AC-37 sends the Group scheduling version and maps the next confirmed version", async () => {
    const response = {
      ...taskResponse(null, "group"),
      hasChildren: true,
      scheduling: {
        ...taskResponse(null, "group").scheduling,
        version: 6,
        source: "override",
        automaticScheduling: false,
        schedulingStartDate: null,
        effectiveAutomaticScheduling: false,
      },
      children: [taskResponse(null, "child")],
    };
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ data: response }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);
    const gateway = createHTTPWBSGateway("/api");

    const confirmed = await gateway.updateGroupScheduling("project", "group", {
      expectedVersion: 5,
      name: "Platform",
      schedulingSource: "override",
      automaticScheduling: false,
      schedulingStartDate: null,
    });

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/projects/project/wbs/group/scheduling",
      expect.objectContaining({ method: "PUT" }),
    );
    const init = fetchMock.mock.calls[0]?.[1] as RequestInit;
    expect(JSON.parse(String(init.body))).toEqual({
      expectedVersion: 5,
      name: "Platform",
      schedulingSource: "override",
      automaticScheduling: false,
      schedulingStartDate: null,
    });
    expect(confirmed.scheduling?.version).toBe(6);
    expect(confirmed.scheduling?.source).toBe("override");
    expect(confirmed.scheduling?.automaticScheduling).toBe(false);
    expect(confirmed.scheduling?.schedulingStartDate).toBeUndefined();
  });

  it("US-4.4 AC-35 AC-37 sends the Group lifecycle version with Lock", async () => {
    const response = {
      ...taskResponse(null, "group"),
      hasChildren: true,
      scheduling: {
        ...taskResponse(null, "group").scheduling,
        version: 8,
        localStatus: "locked",
        effectiveLifecycle: "locked",
        lockOwner: { id: "group", name: "Platform" },
      },
      children: [taskResponse(null, "child")],
    };
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ data: response }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);
    const gateway = createHTTPWBSGateway("/api");

    const confirmed = await gateway.changeGroupStatus(
      "project",
      "group",
      "locked",
      7,
    );

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/projects/project/wbs/group/status",
      expect.objectContaining({ method: "POST" }),
    );
    const init = fetchMock.mock.calls[0]?.[1] as RequestInit;
    expect(JSON.parse(String(init.body))).toEqual({
      status: "locked",
      expectedVersion: 7,
    });
    expect(confirmed.scheduling?.version).toBe(8);
    expect(confirmed.scheduling?.localStatus).toBe("locked");
    expect(confirmed.scheduling?.lockOwner).toEqual({
      id: "group",
      name: "Platform",
    });
  });

  it("US-6.1 AC-36 rejects an in-flight WBS projection after another gateway mutates scheduling state", async () => {
    let resolveOld: ((response: Response) => void) | undefined;
    const oldRequest = new Promise<Response>((resolve) => {
      resolveOld = resolve;
    });
    const fetchMock = vi
      .fn()
      .mockImplementationOnce(() => oldRequest)
      .mockResolvedValueOnce(new Response(null, { status: 204 }))
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ data: [] }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      );
    vi.stubGlobal("fetch", fetchMock);
    const wbsGateway = createHTTPWBSGateway("/api");
    const projectsGateway = createHTTPProjectsGateway("/api");

    const staleTree = wbsGateway.tree("project");
    await projectsGateway.delete("other-project");
    resolveOld?.(
      new Response(JSON.stringify({ data: [taskResponse()] }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );

    await expect(staleTree).rejects.toMatchObject({ name: "AbortError" });
    await expect(wbsGateway.tree("project")).resolves.toEqual([]);
    expect(fetchMock).toHaveBeenCalledTimes(3);
  });

  it("rejects a draft preview invalidated by a newer confirmed scheduling mutation", async () => {
    let resolvePreview: ((response: Response) => void) | undefined;
    const previewRequest = new Promise<Response>((resolve) => {
      resolvePreview = resolve;
    });
    const fetchMock = vi
      .fn()
      .mockImplementationOnce(() => previewRequest)
      .mockResolvedValueOnce(new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetchMock);
    const wbsGateway = createHTTPWBSGateway("/api");
    const projectsGateway = createHTTPProjectsGateway("/api");

    const preview = wbsGateway.previewExecutableSchedule("project", "task", {
      roleId: "role",
      assigneeId: "member",
      effortHours: 8,
      lagDays: 0,
    });
    await projectsGateway.delete("other-project");
    resolvePreview?.(
      new Response(JSON.stringify({ data: schedulePreviewResponse() }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );

    await expect(preview).rejects.toMatchObject({ name: "AbortError" });
  });

  it("omits a cleared Assignee from the preview request so the backend can reconcile the queue", async () => {
    const response = {
      task: {
        ...taskResponse(),
        executable: {
          ...taskResponse().executable,
          assigneeId: null,
          executionTimeline: { start: null, end: null },
          commitmentTimeline: { start: null, end: null },
          executionUnscheduledReason:
            "Task requires an Assignee before it can be scheduled.",
          commitmentUnscheduledReason:
            "Task requires an Assignee before it can be scheduled.",
        },
      },
      dependencies: { blockedBy: [], blocks: [] },
    };
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ data: response }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);
    const gateway = createHTTPWBSGateway("/api");

    const preview = await gateway.previewExecutableSchedule("project", "task", {
      roleId: "role",
      assigneeId: undefined,
      effortHours: 8,
      lagDays: 0,
    });

    const init = fetchMock.mock.calls[0]?.[1] as RequestInit;
    expect(JSON.parse(String(init.body))).toEqual({
      roleId: "role",
      effortHours: 8,
      lag: 0,
    });
    expect(preview.task.executable.assigneeId).toBeUndefined();
    expect(preview.task.executable.executionTimeline).toEqual({});
  });

  it("previews a scheduling draft without invalidating the confirmed WBS cache", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ data: [taskResponse()] }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            data: schedulePreviewResponse(),
          }),
          {
            status: 200,
            headers: { "Content-Type": "application/json" },
          },
        ),
      );
    vi.stubGlobal("fetch", fetchMock);
    const gateway = createHTTPWBSGateway("/api");
    const confirmedChange = vi.fn();
    const unsubscribe = gateway.subscribeToConfirmedChanges?.(confirmedChange);

    await gateway.tree("project");
    const preview = await gateway.previewExecutableSchedule("project", "task", {
      roleId: "role",
      assigneeId: "member",
      effortHours: 8,
      lagDays: 2,
    });
    await gateway.tree("project");

    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "/api/projects/project/wbs/task/executable/preview",
      expect.objectContaining({ method: "POST" }),
    );
    const init = fetchMock.mock.calls[1]?.[1] as RequestInit;
    expect(JSON.parse(String(init.body))).toEqual({
      roleId: "role",
      assigneeId: "member",
      effortHours: 8,
      lag: 2,
    });
    expect(preview.task.id).toBe("task");
    expect(preview.task.executable.executionTimeline).toEqual({
      start: "2026-08-01",
      end: "2026-08-03",
    });
    expect(confirmedChange).not.toHaveBeenCalled();
    unsubscribe?.();
  });

  it("US-6.5 AC-12 AC-25 sends one ranked recommendation batch and maps stable metadata", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          data: assigneeRecommendationResponse(),
        }),
        {
          status: 200,
          headers: { "Content-Type": "application/json" },
        },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);
    const gateway = createHTTPWBSGateway("/api");

    const result = await gateway.recommendAssignees!(
      "project",
      "task/with slash",
      {
        roleId: "role",
        effortHours: 8,
        lagDays: 2,
        capacityAllocationPercentage: 40,
        executionStart: "2026-08-10",
      },
    );

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/projects/project/wbs/task%2Fwith%20slash/assignee-recommendations",
      expect.objectContaining({ method: "POST" }),
    );
    const init = fetchMock.mock.calls[0]?.[1] as RequestInit;
    expect(JSON.parse(String(init.body))).toEqual({
      roleId: "role",
      effortHours: 8,
      capacityAllocationPercentage: 40,
      executionStart: "2026-08-10",
      lag: 2,
    });
    expect(result).toEqual({
      calculatedOnDate: "2026-08-05",
      snapshot: { projectScheduleVersions: { project: 12 } },
      mode: "manual-advisory",
      items: [
        {
          memberId: "member",
          memberName: "Ayu",
          roleId: "role",
          rankGroup: "feasible",
          executionEnd: "2026-08-10",
          remainingExecutionCapacityHours: 4,
          incrementalOvercapacityHours: 0,
          reasonCode: undefined,
        },
      ],
    });
  });

  it("US-6.5 AC-26 AC-28 rejects a recommendation invalidated by a confirmed scheduling mutation", async () => {
    let resolveRecommendation: ((response: Response) => void) | undefined;
    const recommendationRequest = new Promise<Response>((resolve) => {
      resolveRecommendation = resolve;
    });
    const fetchMock = vi
      .fn()
      .mockImplementationOnce(() => recommendationRequest)
      .mockResolvedValueOnce(new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetchMock);
    const wbsGateway = createHTTPWBSGateway("/api");
    const projectsGateway = createHTTPProjectsGateway("/api");

    const recommendation = wbsGateway.recommendAssignees!("project", "task", {
      roleId: "role",
      effortHours: 8,
      lagDays: 0,
      capacityAllocationPercentage: 100,
    });
    await projectsGateway.delete("other-project");
    resolveRecommendation?.(
      new Response(JSON.stringify({ data: assigneeRecommendationResponse() }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );

    await expect(recommendation).rejects.toMatchObject({ name: "AbortError" });
  });

  it("US-6.5 AC-25 AC-31 rejects malformed recommendation rows instead of trusting partial ranking", async () => {
    const malformed = assigneeRecommendationResponse();
    malformed.items[0].remainingExecutionCapacityHours = Number.NaN;
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ data: malformed }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      ),
    );
    const gateway = createHTTPWBSGateway("/api");

    await expect(
      gateway.recommendAssignees!("project", "task", {
        roleId: "role",
        effortHours: 8,
        lagDays: 0,
        capacityAllocationPercentage: 100,
      }),
    ).rejects.toThrow("WBS response is invalid. Try again.");
  });
});

function assigneeRecommendationResponse() {
  return {
    calculatedOnDate: "2026-08-05",
    snapshot: { projectScheduleVersions: { project: 12 } },
    mode: "manual-advisory" as const,
    items: [
      {
        memberId: "member",
        memberName: "Ayu",
        roleId: "role",
        rankGroup: "feasible" as const,
        executionEnd: "2026-08-10",
        remainingExecutionCapacityHours: 4,
        incrementalOvercapacityHours: 0,
        reasonCode: null,
      },
    ],
  };
}

function schedulePreviewResponse() {
  return {
    task: taskResponse(null, "task", "project"),
    dependencies: {
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
            expectedStart: "2026-08-01",
          },
        },
      ],
      blocks: [],
    },
  };
}

function taskResponse(
  actualEnd: string | null = null,
  id = "task",
  projectId = "project",
) {
  return {
    id,
    projectId,
    name: "Build API",
    position: 1,
    hasChildren: false,
    scheduling: {
      version: 0,
      source: "inherit",
      automaticScheduling: null,
      schedulingStartDate: null,
      localStatus: "open",
      effectiveAutomaticScheduling: true,
      effectiveSchedulingStartDate: "2026-08-01",
      inheritedAutomaticScheduling: true,
      inheritedSchedulingStartDate: "2026-08-01",
      inheritedAutomaticSource: { id: projectId, name: "Alpha" },
      inheritedStartDateSource: { id: projectId, name: "Alpha" },
      automaticSource: { id: projectId, name: "Alpha" },
      startDateSource: { id: projectId, name: "Alpha" },
      effectiveLifecycle: "open",
      lockOwner: null,
    },
    executable: {
      capacityAllocationPercentage: 100,
      roleId: null,
      assigneeId: null,
      effortMinutes: 390,
      lag: 0,
      executionTimeline: { start: "2026-08-01", end: "2026-08-03" },
      commitmentTimeline: { start: "2026-08-01", end: "2026-08-05" },
      executionUnscheduledReason: null,
      commitmentUnscheduledReason: null,
      actualStart: actualEnd,
      actualEnd,
    },
    children: [],
  };
}

describe("HTTP WBS gateway reopen command", () => {
  it("AC-4 uses the dedicated endpoint without arbitrary payload and validates confirmed state", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({ data: taskResponse(null, "task/with slash") }),
        {
          status: 200,
          headers: { "Content-Type": "application/json" },
        },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);
    const gateway = createHTTPWBSGateway("/api");

    const confirmed = await gateway.reopen("project", "task/with slash");

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/projects/project/wbs/task%2Fwith%20slash/reopen",
      expect.objectContaining({ method: "POST" }),
    );
    const init = fetchMock.mock.calls[0]?.[1] as RequestInit | undefined;
    expect(init?.body).toBeUndefined();
    expect(confirmed).toEqual({
      ...taskResponse(null, "task/with slash"),
      parentId: undefined,
      scheduling: {
        version: 0,
        source: "inherit",
        automaticScheduling: undefined,
        schedulingStartDate: undefined,
        localStatus: "open",
        effectiveAutomaticScheduling: true,
        effectiveSchedulingStartDate: "2026-08-01",
        inheritedAutomaticScheduling: true,
        inheritedSchedulingStartDate: "2026-08-01",
        inheritedAutomaticSource: { id: "project", name: "Alpha" },
        inheritedStartDateSource: { id: "project", name: "Alpha" },
        automaticSource: { id: "project", name: "Alpha" },
        startDateSource: { id: "project", name: "Alpha" },
        effectiveLifecycle: "open",
        lockOwner: undefined,
      },
      executable: {
        roleId: undefined,
        assigneeId: undefined,
        effortMinutes: 390,
        lagDays: 0,
        capacityAllocationPercentage: 100,
        executionTimeline: { start: "2026-08-01", end: "2026-08-03" },
        commitmentTimeline: { start: "2026-08-01", end: "2026-08-05" },
        executionUnscheduledReason: undefined,
        commitmentUnscheduledReason: undefined,
        actualStart: undefined,
        actualEnd: undefined,
      },
    });
  });

  it("AC-4 AC-14 rejects a completed projection from the Reopen command", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ data: taskResponse("2026-07-28") }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      ),
    );
    const gateway = createHTTPWBSGateway("/api");
    await expect(gateway.reopen("project", "task")).rejects.toMatchObject({
      code: "TASK_REOPEN_FAILED",
      message: "The Task could not be reopened. Try again.",
    });
  });

  it("AC-2 AC-9 AC-11 AC-12 AC-13 maps structured reopen errors to safe messages", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            code: "TASK_REOPEN_CONFLICT",
            message: "SQL infrastructure detail",
          }),
          {
            status: 409,
            headers: { "Content-Type": "application/json" },
          },
        ),
      ),
    );
    const gateway = createHTTPWBSGateway("/api");
    await expect(gateway.reopen("project", "task")).rejects.toMatchObject({
      code: "TASK_REOPEN_CONFLICT",
      message: "The Task changed in another request. Refresh and try again.",
    });
  });

  it("AC-4 AC-14 rejects an invalid reopen projection instead of trusting it", async () => {
    const invalid = taskResponse(null);
    invalid.executable.actualEnd = 42 as unknown as null;
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ data: invalid }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      ),
    );
    const gateway = createHTTPWBSGateway("/api");
    await expect(gateway.reopen("project", "task")).rejects.toMatchObject({
      code: "TASK_REOPEN_FAILED",
      message: "The Task could not be reopened. Try again.",
    });
  });

  it("AC-4 AC-5 AC-14 requires explicit null and preserved Task identity", async () => {
    const reopened = taskResponse(null);
    const executableWithoutActualStart: Partial<typeof reopened.executable> = {
      ...reopened.executable,
    };
    delete executableWithoutActualStart.actualStart;
    const executableWithoutActualEnd: Partial<typeof reopened.executable> = {
      ...reopened.executable,
    };
    delete executableWithoutActualEnd.actualEnd;
    const cases: Array<{ name: string; data: unknown }> = [
      {
        name: "missing actualStart",
        data: {
          ...reopened,
          executable: executableWithoutActualStart,
        },
      },
      {
        name: "missing actualEnd",
        data: {
          ...reopened,
          executable: executableWithoutActualEnd,
        },
      },
      {
        name: "different Task",
        data: taskResponse(null, "other-task"),
      },
      {
        name: "different Project",
        data: taskResponse(null, "task", "other-project"),
      },
      {
        name: "group projection",
        data: { ...taskResponse(null), hasChildren: true },
      },
      {
        name: "leaf projection with children",
        data: {
          ...taskResponse(null),
          children: [taskResponse(null, "unexpected-child")],
        },
      },
    ];

    for (const testCase of cases) {
      const fetchMock = vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ data: testCase.data }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      );
      vi.stubGlobal("fetch", fetchMock);
      const gateway = createHTTPWBSGateway("/api");
      await expect(
        gateway.reopen("project", "task"),
        testCase.name,
      ).rejects.toMatchObject({ code: "TASK_REOPEN_FAILED" });
      vi.unstubAllGlobals();
    }
  });

  it("AC-15 invalidates cached WBS after a successful HTTP response even when projection validation fails", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ data: [taskResponse("2026-07-28")] }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ data: taskResponse(null, "other") }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ data: [taskResponse(null)] }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      );
    vi.stubGlobal("fetch", fetchMock);
    const gateway = createHTTPWBSGateway("/api");

    await gateway.tree("project");
    await expect(gateway.reopen("project", "task")).rejects.toMatchObject({
      code: "TASK_REOPEN_FAILED",
    });
    const refreshed = await gateway.tree("project");

    expect(refreshed[0]?.executable.actualStart).toBeUndefined();
    expect(refreshed[0]?.executable.actualEnd).toBeUndefined();
    expect(fetchMock).toHaveBeenCalledTimes(3);
  });

  it("AC-15 invalidates cached WBS after a 2xx response with malformed JSON", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ data: [taskResponse("2026-07-28")] }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      )
      .mockResolvedValueOnce(
        new Response("not-json", {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ data: [taskResponse(null)] }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      );
    vi.stubGlobal("fetch", fetchMock);
    const gateway = createHTTPWBSGateway("/api");

    await gateway.tree("project");
    await expect(gateway.reopen("project", "task")).rejects.toMatchObject({
      code: "TASK_REOPEN_FAILED",
    });
    const refreshed = await gateway.tree("project");

    expect(refreshed[0]?.executable.actualStart).toBeUndefined();
    expect(refreshed[0]?.executable.actualEnd).toBeUndefined();
    expect(fetchMock).toHaveBeenCalledTimes(3);
  });

  it("AC-15 invalidates the tree and prevents an older in-flight response from becoming cached", async () => {
    let resolveOldTree: ((response: Response) => void) | undefined;
    const oldTree = new Promise<Response>((resolve) => {
      resolveOldTree = resolve;
    });
    const fetchMock = vi
      .fn()
      .mockImplementationOnce(() => oldTree)
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ data: taskResponse(null) }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ data: [taskResponse(null)] }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      );
    vi.stubGlobal("fetch", fetchMock);
    const gateway = createHTTPWBSGateway("/api");

    const staleConsumer = gateway.tree("project");
    await gateway.reopen("project", "task");
    resolveOldTree?.(
      new Response(JSON.stringify({ data: [taskResponse("2026-07-28")] }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    await expect(staleConsumer).rejects.toMatchObject({ name: "AbortError" });

    const confirmedTree = await gateway.tree("project");
    expect(confirmedTree[0]?.executable.actualEnd).toBeUndefined();
    expect(fetchMock).toHaveBeenCalledTimes(3);
  });
});

describe("HTTP WBS gateway sibling structural commands", () => {
  it("sends an authoritative insert-after anchor and returns the confirmed sibling_DeltaD03", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          data: { ...taskResponse(null, "new-sibling"), parentId: "group" },
        }),
        {
          status: 201,
          headers: { "Content-Type": "application/json" },
        },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);
    const gateway = createHTTPWBSGateway("/api");

    const created = await gateway.createSibling(
      "project",
      "anchor",
      "New sibling",
    );

    expect(created.id).toBe("new-sibling");
    expect(created.parentId).toBe("group");
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/projects/project/wbs",
      expect.objectContaining({ method: "POST" }),
    );
    const init = fetchMock.mock.calls[0]?.[1] as RequestInit;
    expect(JSON.parse(String(init.body))).toEqual({
      insertAfterWbsId: "anchor",
      name: "New sibling",
    });
    expect(String(init.body)).not.toContain("parentId");
    expect(String(init.body)).not.toContain("position");
  });

  it("sends source target and before-after placement for drag reorder_DeltaD05", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetchMock);
    const gateway = createHTTPWBSGateway("/api");

    await gateway.place("project", "source", "target", "after");

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/projects/project/wbs/source/reorder",
      expect.objectContaining({ method: "POST" }),
    );
    const init = fetchMock.mock.calls[0]?.[1] as RequestInit;
    expect(JSON.parse(String(init.body))).toEqual({
      targetSiblingId: "target",
      placement: "after",
    });
    expect(String(init.body)).not.toContain("direction");
    expect(String(init.body)).not.toContain("parentId");
    expect(String(init.body)).not.toContain("position");
  });
});

describe("HTTP WBS gateway sibling conflict mapping", () => {
  it("maps a stale insert-after anchor, invalidates confirmed caches, and exposes a recoverable Home message_DeltaD03_D09", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ data: [taskResponse()] }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            code: "WBS_CREATE_ANCHOR_CONFLICT",
            message: "stale anchor",
          }),
          {
            status: 409,
            headers: { "Content-Type": "application/json" },
          },
        ),
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ data: [] }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      );
    vi.stubGlobal("fetch", fetchMock);
    const gateway = createHTTPWBSGateway("/api");
    const confirmedChange = vi.fn();
    const unsubscribe = gateway.subscribeToConfirmedChanges?.(confirmedChange);

    await gateway.tree("project");
    await expect(
      gateway.createSibling("project", "deleted-anchor", "New sibling"),
    ).rejects.toThrow(
      "The selected sibling is no longer available. Home will keep the confirmed order; refresh and try again.",
    );
    await expect(gateway.tree("project")).resolves.toEqual([]);

    expect(fetchMock).toHaveBeenCalledTimes(3);
    expect(confirmedChange).toHaveBeenCalledTimes(1);
    unsubscribe?.();
  });

  it("invalidates confirmed caches when target placement becomes stale_DeltaD05_D09", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ data: [taskResponse()] }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ code: "WBS_REORDER_TARGET_INVALID" }), {
          status: 409,
          headers: { "Content-Type": "application/json" },
        }),
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ data: [] }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      );
    vi.stubGlobal("fetch", fetchMock);
    const gateway = createHTTPWBSGateway("/api");

    await gateway.tree("project");
    await expect(
      gateway.place("project", "source", "deleted-target", "after"),
    ).rejects.toThrow(
      "The selected reorder target is no longer valid. Refresh Home and try again.",
    );
    await expect(gateway.tree("project")).resolves.toEqual([]);

    expect(fetchMock).toHaveBeenCalledTimes(3);
  });
});
