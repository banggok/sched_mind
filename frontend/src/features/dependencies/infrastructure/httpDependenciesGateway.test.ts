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
