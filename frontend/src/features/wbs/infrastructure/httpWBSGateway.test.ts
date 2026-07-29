import { afterEach, describe, expect, it, vi } from "vitest";
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
    await gateway.tree("project");
    await gateway.tree("project");
    expect(fetchMock).toHaveBeenCalledTimes(1);
    await gateway.create("project", undefined, "Task", false);
    await gateway.tree("project");
    expect(fetchMock).toHaveBeenCalledTimes(3);
  });
});

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
      executionTimeline: { start: "2026-08-01", end: "2026-08-03" },
      commitmentTimeline: { start: "2026-08-01", end: "2026-08-05" },
      actualEnd,
    },
    children: [],
  };
}

describe("HTTP WBS gateway reopen command", () => {
  it("AC-4 uses the dedicated endpoint without arbitrary payload and validates confirmed state", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ data: taskResponse(null, "task/with slash") }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
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
        executionTimeline: { start: "2026-08-01", end: "2026-08-03" },
        commitmentTimeline: { start: "2026-08-01", end: "2026-08-05" },
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
        new Response(
          JSON.stringify({ data: [taskResponse("2026-07-28")] }),
          {
            status: 200,
            headers: { "Content-Type": "application/json" },
          },
        ),
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
        new Response(
          JSON.stringify({ data: [taskResponse("2026-07-28")] }),
          {
            status: 200,
            headers: { "Content-Type": "application/json" },
          },
        ),
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
      new Response(
        JSON.stringify({ data: [taskResponse("2026-07-28")] }),
        {
          status: 200,
          headers: { "Content-Type": "application/json" },
        },
      ),
    );
    await expect(staleConsumer).rejects.toMatchObject({ name: "AbortError" });

    const confirmedTree = await gateway.tree("project");
    expect(confirmedTree[0]?.executable.actualEnd).toBeUndefined();
    expect(fetchMock).toHaveBeenCalledTimes(3);
  });
});
