import { afterEach, describe, expect, it, vi } from "vitest";
import { advanceScheduleProjectionVersion } from "../../../shared/infrastructure/scheduleProjectionClock";
import { createHTTPPortfolioGateway } from "./httpPortfolioGateway";

afterEach(() => {
  vi.restoreAllMocks();
});

describe("httpPortfolioGateway", () => {
  it("rejects a projection response that became stale after a confirmed mutation", async () => {
    let resolveResponse: ((response: Response) => void) | undefined;
    vi.spyOn(globalThis, "fetch").mockImplementation(
      () =>
        new Promise<Response>((resolve) => {
          resolveResponse = resolve;
        }),
    );
    const gateway = createHTTPPortfolioGateway("/api");
    const pending = gateway.projection(
      ["project"],
      "execution",
      "2026-08-01",
      "2026-08-31",
    );

    advanceScheduleProjectionVersion();
    resolveResponse?.(
      new Response(
        JSON.stringify({
          data: {
            projection: "execution",
            projects: [],
            rows: [],
            dependencies: [],
            holidays: [],
          },
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );

    await expect(pending).rejects.toMatchObject({ name: "AbortError" });
  });

  it("rejects malformed portfolio payloads instead of trusting the transport boundary", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify({ data: { projection: "execution" } }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    const gateway = createHTTPPortfolioGateway("/api");

    await expect(
      gateway.projection(["project"], "execution", "2026-08-01", "2026-08-31"),
    ).rejects.toMatchObject({ code: "UNEXPECTED_RESPONSE" });
  });

  it("maps Task Role fields from the portfolio projection", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(
        JSON.stringify({
          data: {
            projection: "execution",
            projects: [],
            rows: [
              {
                id: "task",
                projectId: "project",
                kind: "task",
                name: "Test API",
                wbsNumber: "1",
                depth: 1,
                position: 1,
                roleId: "testing",
                roleName: "Testing",
                incompleteEffort: false,
                incompleteSchedule: false,
                hasChildren: false,
                completed: false,
              },
            ],
            dependencies: [],
            holidays: [],
          },
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );
    const gateway = createHTTPPortfolioGateway("/api");

    const result = await gateway.projection(
      ["project"],
      "execution",
      "2026-08-01",
      "2026-08-31",
    );

    expect(result.rows[0]).toMatchObject({
      roleId: "testing",
      roleName: "Testing",
    });
  });

  it("maps saved-filter optimistic concurrency failures", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(
        JSON.stringify({
          code: "SAVED_FILTER_CONFLICT",
          message: "saved filter changed concurrently",
        }),
        { status: 409, headers: { "Content-Type": "application/json" } },
      ),
    );
    const gateway = createHTTPPortfolioGateway("/api");

    await expect(
      gateway.updateSavedFilter("filter", 1, ["project"]),
    ).rejects.toMatchObject({ code: "SAVED_FILTER_CONFLICT" });
  });
});

it("shares Strict Mode metadata requests without aborting the underlying fetch", async () => {
  let response = deferred<Response>();
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockImplementation(() => response.promise);
  const gateway = createHTTPPortfolioGateway("/api");

  const firstProjectsController = new AbortController();
  const firstProjects = gateway.activeProjects(firstProjectsController.signal);
  const secondProjects = gateway.activeProjects();
  firstProjectsController.abort();

  await expect(firstProjects).rejects.toMatchObject({ name: "AbortError" });
  expect(fetchMock).toHaveBeenCalledTimes(1);
  response.resolve(emptyDataResponse());
  await expect(secondProjects).resolves.toEqual([]);

  response = deferred<Response>();
  const firstFiltersController = new AbortController();
  const firstFilters = gateway.savedFilters(firstFiltersController.signal);
  const secondFilters = gateway.savedFilters();
  firstFiltersController.abort();

  await expect(firstFilters).rejects.toMatchObject({ name: "AbortError" });
  expect(fetchMock).toHaveBeenCalledTimes(2);
  response.resolve(emptyDataResponse());
  await expect(secondFilters).resolves.toEqual([]);

  expect(fetchMock.mock.calls[0]?.[1]).toBeUndefined();
  expect(fetchMock.mock.calls[1]?.[1]).toBeUndefined();
});

function deferred<T>(): {
  promise: Promise<T>;
  resolve(value: T): void;
  reject(reason?: unknown): void;
} {
  let resolvePromise!: (value: T) => void;
  let rejectPromise!: (reason?: unknown) => void;
  const promise = new Promise<T>((resolve, reject) => {
    resolvePromise = resolve;
    rejectPromise = reject;
  });
  return {
    promise,
    resolve: resolvePromise,
    reject: rejectPromise,
  };
}

function emptyDataResponse(): Response {
  return new Response(JSON.stringify({ data: [] }), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });
}
