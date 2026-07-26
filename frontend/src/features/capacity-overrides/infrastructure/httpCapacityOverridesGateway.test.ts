import { afterEach, describe, expect, it, vi } from "vitest";
import { createHTTPCapacityOverridesGateway } from "./httpCapacityOverridesGateway";
afterEach(() => vi.unstubAllGlobals());
describe("HTTP capacity override gateway", () => {
  it("maps paginated date-only data and caches identical list calls", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          data: [
            {
              id: "o",
              teamMemberId: "m",
              description: "Training",
              startDate: "2026-07-03",
              endDate: "2026-07-03",
              capacity: 0,
              createdAt: "2026-07-26T00:00:00Z",
              updatedAt: "2026-07-26T00:00:00Z",
            },
          ],
          page: 1,
          pageSize: 5,
          total: 1,
        }),
        { status: 200 },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);
    const gateway = createHTTPCapacityOverridesGateway("/api");
    const [one, two] = await Promise.all([
      gateway.list("m", { page: 1, pageSize: 5 }),
      gateway.list("m", { page: 1, pageSize: 5 }),
    ]);
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(one.items[0].capacity).toBe(0);
    expect(two.total).toBe(1);
  });
  it("uses nested paths and invalidates after mutation", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({ data: [], page: 1, pageSize: 5, total: 0 }),
          { status: 200 },
        ),
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            data: {
              id: "o",
              teamMemberId: "m",
              description: "Training",
              startDate: "2026-07-03",
              endDate: "2026-07-03",
              capacity: 4,
              createdAt: "2026-07-26T00:00:00Z",
              updatedAt: "2026-07-26T00:00:00Z",
            },
          }),
          { status: 201 },
        ),
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({ data: [], page: 1, pageSize: 5, total: 0 }),
          { status: 200 },
        ),
      );
    vi.stubGlobal("fetch", fetchMock);
    const gateway = createHTTPCapacityOverridesGateway("/api");
    await gateway.list("m", { page: 1, pageSize: 5 });
    await gateway.create("m", {
      description: "Training",
      startDate: "2026-07-03",
      endDate: "2026-07-03",
      capacity: 4,
    });
    await gateway.list("m", { page: 1, pageSize: 5 });
    expect(fetchMock.mock.calls[1][0]).toBe(
      "/api/team-members/m/capacity-overrides",
    );
    expect(fetchMock).toHaveBeenCalledTimes(3);
  });
});

describe("HTTP capacity override Effective Date cache", () => {
  it("includes the date in URL and cache identity and invalidates filtered lists after mutation", async () => {
    const listResponse = () =>
      new Response(
        JSON.stringify({ data: [], page: 1, pageSize: 5, total: 0 }),
        { status: 200 },
      );
    const itemResponse = () =>
      new Response(
        JSON.stringify({
          data: {
            id: "o",
            teamMemberId: "m",
            description: "Support",
            startDate: "2026-07-26",
            endDate: "2026-07-28",
            capacity: 3,
            createdAt: "2026-07-26T00:00:00Z",
            updatedAt: "2026-07-26T00:00:00Z",
          },
        }),
        { status: 201 },
      );
    const fetchMock = vi
      .fn()
      .mockImplementation((_url, options) =>
        Promise.resolve(
          options?.method === "POST" ? itemResponse() : listResponse(),
        ),
      );
    vi.stubGlobal("fetch", fetchMock);
    const gateway = createHTTPCapacityOverridesGateway("/api");
    await gateway.list("m", {
      page: 1,
      pageSize: 5,
      effectiveDate: "2026-07-27",
    });
    await gateway.list("m", { page: 1, pageSize: 5 });
    await gateway.list("m", {
      page: 1,
      pageSize: 5,
      effectiveDate: "2026-07-27",
    });
    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(String(fetchMock.mock.calls[0][0])).toContain(
      "effectiveDate=2026-07-27",
    );
    await gateway.create("m", {
      description: "Support",
      startDate: "2026-07-26",
      endDate: "2026-07-28",
      capacity: 3,
    });
    await gateway.list("m", {
      page: 1,
      pageSize: 5,
      effectiveDate: "2026-07-27",
    });
    expect(fetchMock).toHaveBeenCalledTimes(4);
  });
});
