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

  it("invalidates the confirmed tree after mutation", async () => {
    const fetchMock = vi.fn(
      (_url: string | URL | Request, init?: RequestInit) =>
        Promise.resolve(
          init?.method
            ? new Response(null, { status: 204 })
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
    await gateway.create("project", undefined, "Task", false);
    expect(confirmedChange).toHaveBeenCalledTimes(1);
    await gateway.tree("project");
    expect(fetchMock).toHaveBeenCalledTimes(3);
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
    expect(preview.dependencies).toEqual({ blockedBy: [], blocks: [] });
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
    expect(preview.dependencies.blockedBy).toEqual([
      expect.objectContaining({
        id: "automatic-1",
        source: "automatic",
        task: expect.objectContaining({ id: "task-1", name: "Task 1" }),
      }),
    ]);
    expect(confirmedChange).not.toHaveBeenCalled();
    unsubscribe?.();
  });
});

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
    executable: {
      roleId: null,
      assigneeId: null,
      effortMinutes: 390,
      lag: 0,
      executionTimeline: { start: "2026-08-01", end: "2026-08-03" },
      commitmentTimeline: { start: "2026-08-01", end: "2026-08-05" },
      executionUnscheduledReason: null,
      commitmentUnscheduledReason: null,
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
      { method: "POST" },
    );
    const init = fetchMock.mock.calls[0]?.[1] as RequestInit | undefined;
    expect(init?.body).toBeUndefined();
    expect(confirmed).toEqual({
      ...taskResponse(null, "task/with slash"),
      parentId: undefined,
      executable: {
        roleId: undefined,
        assigneeId: undefined,
        effortMinutes: 390,
        lagDays: 0,
        executionTimeline: { start: "2026-08-01", end: "2026-08-03" },
        commitmentTimeline: { start: "2026-08-01", end: "2026-08-05" },
        executionUnscheduledReason: undefined,
        commitmentUnscheduledReason: undefined,
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
    const missingActualEnd = taskResponse(null);
    const executableWithoutActualEnd: Partial<
      typeof missingActualEnd.executable
    > = { ...missingActualEnd.executable };
    delete executableWithoutActualEnd.actualEnd;
    const cases: Array<{ name: string; data: unknown }> = [
      {
        name: "missing actualEnd",
        data: {
          ...missingActualEnd,
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
