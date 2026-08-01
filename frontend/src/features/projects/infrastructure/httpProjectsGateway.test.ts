import { afterEach, describe, expect, it, vi } from "vitest";
import { ProjectOperationError } from "../application/projectsGateway";
import { createHTTPProjectsGateway } from "./httpProjectsGateway";

afterEach(() => vi.unstubAllGlobals());

describe("HTTP projects gateway", () => {
  it("maps list DTOs and includes query identity", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          data: [
            {
              id: "p1",
              name: "Alpha",
              status: "open",
              startDate: null,
              endDate: null,
              autoCalculateDate: true,
              automaticScheduling: true,
              schedulingStartDate: "2026-08-03",
              projectBuffer: 20,
              scheduleVersion: 3,
              projectPriority: 1,
              closedAt: null,
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
    const result = await createHTTPProjectsGateway("/api").list({
      search: "al",
      page: 1,
      pageSize: 5,
    });
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/projects?search=al&page=1&pageSize=5",
      expect.any(Object),
    );
    expect(result.items[0]).toMatchObject({
      name: "Alpha",
      priority: 1,
      schedulingStartDate: "2026-08-03",
      scheduleVersion: 3,
    });
  });

  it("maps structured mutation errors", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            code: "PROJECT_HAS_CHILDREN",
            message: "internal wording",
          }),
          { status: 409 },
        ),
      ),
    );
    await expect(
      createHTTPProjectsGateway("/api").delete("p1"),
    ).rejects.toBeInstanceOf(ProjectOperationError);
  });

  it("sends a name-only payload for Locked Project rename_US31_AC11_US62_AC16A", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          data: {
            id: "p1",
            name: "Renamed",
            status: "locked",
            startDate: null,
            endDate: null,
            autoCalculateDate: true,
            automaticScheduling: true,
            schedulingStartDate: "2026-08-03",
            projectBuffer: 20,
            scheduleVersion: 4,
            projectPriority: 1,
            closedAt: null,
            createdAt: "2026-07-26T00:00:00Z",
            updatedAt: "2026-08-01T00:00:00Z",
          },
        }),
        { status: 200 },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    await createHTTPProjectsGateway("/api").rename("p1", "Renamed");

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/projects/p1",
      expect.objectContaining({
        method: "PUT",
        body: JSON.stringify({ name: "Renamed" }),
      }),
    );
  });
});
