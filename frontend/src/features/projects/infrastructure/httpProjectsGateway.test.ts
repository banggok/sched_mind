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
              autoDependencyByAssignee: true,
              automaticScheduling: true,
              schedulingStartDate: "2026-08-03",
              projectBuffer: 20,
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
    );
    expect(result.items[0]).toMatchObject({
      name: "Alpha",
      priority: 1,
      schedulingStartDate: "2026-08-03",
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
});
