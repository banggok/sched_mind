import { afterEach, describe, expect, it, vi } from "vitest";
import { createHTTPPublicHolidaysGateway } from "./httpPublicHolidaysGateway";
afterEach(() => vi.unstubAllGlobals());
describe("HTTP public holidays gateway", () => {
  it("includes filter in cache identity and invalidates all lists after mutation", async () => {
    const list = () =>
      new Response(
        JSON.stringify({ data: [], page: 1, pageSize: 5, total: 0 }),
        { status: 200 },
      );
    const item = () =>
      new Response(
        JSON.stringify({
          data: {
            id: "h",
            startDate: "2026-08-17",
            endDate: "2026-08-17",
            description: "Holiday",
            createdAt: "2026-07-26T00:00:00Z",
            updatedAt: "2026-07-26T00:00:00Z",
          },
        }),
        { status: 201 },
      );
    const fetchMock = vi
      .fn()
      .mockImplementation((_url, options) =>
        Promise.resolve(options?.method === "POST" ? item() : list()),
      );
    vi.stubGlobal("fetch", fetchMock);
    const gateway = createHTTPPublicHolidaysGateway("/api");
    await gateway.list({ page: 1, pageSize: 5 });
    await gateway.list({ page: 1, pageSize: 5, holidayDate: "2026-08-17" });
    await gateway.list({ page: 1, pageSize: 5, holidayDate: "2026-08-17" });
    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(String(fetchMock.mock.calls[1][0])).toContain(
      "holidayDate=2026-08-17",
    );
    await gateway.create({
      startDate: "2026-08-17",
      endDate: "2026-08-17",
      description: "Holiday",
    });
    await gateway.list({ page: 1, pageSize: 5 });
    expect(fetchMock).toHaveBeenCalledTimes(4);
  });
});
