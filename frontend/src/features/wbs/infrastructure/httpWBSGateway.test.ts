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
