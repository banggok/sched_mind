import { afterEach, describe, expect, it, vi } from "vitest";
import { createHTTPDependenciesGateway } from "./httpDependenciesGateway";

afterEach(() => vi.unstubAllGlobals());

function detail(completed: boolean) {
  return {
    blockedBy: [
      {
        id: "incoming-link",
        source: "manual",
        manualRemovable: true,
        task: {
          id: "reopened-task",
          name: "Build API",
          projectId: "project",
          projectName: "Alpha",
          hierarchyPath: "Alpha > Build API",
          completed,
        },
      },
    ],
    blocks: [],
  };
}

describe("dependency cache invalidation after Task reopen", () => {
  it("AC-15 prevents an older completed marker from becoming cached after invalidateAll", async () => {
    let resolveOld: ((response: Response) => void) | undefined;
    const oldRequest = new Promise<Response>((resolve) => {
      resolveOld = resolve;
    });
    const fetchMock = vi
      .fn()
      .mockImplementationOnce(() => oldRequest)
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ data: detail(false) }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      );
    vi.stubGlobal("fetch", fetchMock);
    const gateway = createHTTPDependenciesGateway("/api");

    const staleConsumer = gateway.list("other-task");
    gateway.invalidateAll();
    resolveOld?.(
      new Response(JSON.stringify({ data: detail(true) }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    await expect(staleConsumer).rejects.toMatchObject({ name: "AbortError" });

    const refreshed = await gateway.list("other-task");
    expect(refreshed.blockedBy[0]?.task.completed).toBe(false);
    expect(refreshed.blockedBy[0]?.id).toBe("incoming-link");
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it("AC-15 rejects an old dependency-candidate projection after invalidateAll", async () => {
    let resolveOld: ((response: Response) => void) | undefined;
    const oldRequest = new Promise<Response>((resolve) => {
      resolveOld = resolve;
    });
    const fetchMock = vi
      .fn()
      .mockImplementationOnce(() => oldRequest)
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            data: { items: [], page: 1, pageSize: 5, totalItems: 0 },
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        ),
      );
    vi.stubGlobal("fetch", fetchMock);
    const gateway = createHTTPDependenciesGateway("/api");

    const staleConsumer = gateway.candidates(
      "other-task",
      "blockedBy",
      "",
      1,
      5,
    );
    gateway.invalidateAll();
    resolveOld?.(
      new Response(
        JSON.stringify({
          data: {
            items: [detail(true).blockedBy[0].task],
            page: 1,
            pageSize: 5,
            totalItems: 1,
          },
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );
    await expect(staleConsumer).rejects.toMatchObject({ name: "AbortError" });

    await expect(
      gateway.candidates("other-task", "blockedBy", "", 1, 5),
    ).resolves.toEqual({ items: [], page: 1, pageSize: 5, totalItems: 0 });
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });
});

describe("automatic dependency ownership contract", () => {
  it("US-6.1 AC-23 AC-24 maps shared ownership once with scheduler expected start", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            data: {
              blockedBy: [],
              blocks: [
                {
                  id: "shared-link",
                  source: "both",
                  manualRemovable: true,
                  task: {
                    id: "blocked",
                    name: "Build API",
                    projectId: "project",
                    projectName: "Alpha",
                    hierarchyPath: "Alpha > Build API",
                    completed: false,
                    expectedStart: "2026-08-10",
                  },
                },
              ],
            },
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        ),
      ),
    );
    const gateway = createHTTPDependenciesGateway("/api");

    const result = await gateway.list("blocking");

    expect(result.blocks).toEqual([
      expect.objectContaining({
        id: "shared-link",
        source: "both",
        manualRemovable: true,
        task: expect.objectContaining({ expectedStart: "2026-08-10" }),
      }),
    ]);
  });

  it("US-6.1 AC-25 keeps an automatic relation as manual through the dedicated command", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetchMock);
    const gateway = createHTTPDependenciesGateway("/api");

    await gateway.keepAsManual("automatic/link");

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/dependencies/automatic%2Flink/keep-manual",
      expect.objectContaining({ method: "POST" }),
    );
  });

  it("US-6.1 AC-23 rejects a dependency source outside manual, automatic, or both", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            data: {
              blockedBy: [
                {
                  id: "invalid",
                  source: "manual+automatic",
                  manualRemovable: true,
                  task: detail(false).blockedBy[0].task,
                },
              ],
              blocks: [],
            },
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        ),
      ),
    );
    const gateway = createHTTPDependenciesGateway("/api");

    await expect(gateway.list("task")).rejects.toThrow(
      "Dependency response is invalid.",
    );
  });
});
