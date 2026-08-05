import { afterEach, describe, expect, it, vi } from "vitest";
import { SprintAPIError } from "../application/sprintsGateway";
import { createHTTPSprintsGateway } from "./httpSprintsGateway";

afterEach(() => vi.unstubAllGlobals());

describe("HTTP Sprints gateway", () => {
  it("maps list dates/status and sends Version with Start and Delete", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            data: [
              {
                id: "s1",
                name: "Sprint",
                startDate: "2026-08-04",
                endDate: "2026-08-05",
                status: "planned",
                version: 1,
                createdAt: "2026-08-01T00:00:00Z",
                updatedAt: "2026-08-01T00:00:00Z",
              },
            ],
            page: 1,
            pageSize: 10,
            total: 1,
          }),
          { status: 200 },
        ),
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            data: {
              id: "s1",
              name: "Sprint",
              startDate: "2026-08-04",
              endDate: "2026-08-05",
              status: "started",
              version: 2,
              createdAt: "2026-08-01T00:00:00Z",
              updatedAt: "2026-08-02T00:00:00Z",
            },
          }),
          { status: 200 },
        ),
      )
      .mockResolvedValueOnce(new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetchMock);
    const gateway = createHTTPSprintsGateway("http://api.test/api");
    const page = await gateway.list({ search: "", page: 1, pageSize: 10 });
    expect(page.items[0]).toMatchObject({
      id: "s1",
      status: "planned",
      memberIds: [],
      taskIds: [],
    });
    await gateway.start("s1", 1);
    await gateway.delete("s1", 2);
    expect(fetchMock.mock.calls[1]?.[1]).toMatchObject({
      method: "POST",
      body: JSON.stringify({ version: 1 }),
    });
    expect(fetchMock.mock.calls[2]?.[0]).toBe(
      "http://api.test/api/sprints/s1?version=2",
    );
  });

  it("posts unsaved Sprint context when listing draft Task candidates_AC40", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(
        new Response(
          JSON.stringify({ data: [], page: 1, pageSize: 20, total: 0 }),
          { status: 200 },
        ),
      );
    vi.stubGlobal("fetch", fetchMock);
    const gateway = createHTTPSprintsGateway("http://api.test/api");

    await gateway.draftCandidates(
      {
        startDate: "2026-08-04",
        endDate: "2026-08-08",
        memberIds: ["member-1"],
        excludedTaskIds: ["task-1"],
      },
      1,
    );

    expect(fetchMock).toHaveBeenCalledWith(
      "http://api.test/api/sprints/task-candidates?page=1&pageSize=20&search=",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({
          startDate: "2026-08-04",
          endDate: "2026-08-08",
          memberIds: ["member-1"],
          excludedTaskIds: ["task-1"],
        }),
      }),
    );
  });

  it("normalizes nullable Task collections before Sprint Planning rendering", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            data: {
              members: [
                {
                  id: "member-1",
                  name: "Harry",
                  roleName: "Engineer",
                  dailyCapacity: [{ date: "2026-08-04", minutes: 480 }],
                  dailySummaries: [
                    {
                      date: "2026-08-04",
                      capacityMinutes: 480,
                      selectedAllocationMinutes: 120,
                      remainingMinutes: 360,
                      overcapacityMinutes: 0,
                    },
                  ],
                  capacityMinutes: 480,
                  inSprintAllocationMinutes: 120,
                  remainingMinutes: 360,
                  overcapacityMinutes: 0,
                  totalAllocationMinutes: 120,
                },
              ],
              tasks: [
                {
                  reason: "mandatory",
                  task: {
                    id: "task-1",
                    projectId: "project-1",
                    projectName: "Alpha",
                    projectStatus: "open",
                    projectPriority: 2,
                    name: "API",
                    wbsOrder: "1",
                    wbsPath: "1.2",
                    wbsRank: 3,
                    dailyPlanOrderDate: "2026-08-04",
                    completed: false,
                    allocations: null,
                    inSprintAllocationMinutes: 0,
                    outsideAllocationMinutes: 0,
                    totalAllocationMinutes: 0,
                    warnings: null,
                  },
                },
              ],
              totals: {
                capacityMinutes: 0,
                selectedMemberAllocationMinutes: 0,
                remainingMinutes: 0,
                overcapacityMinutes: 0,
                needsReviewAllocationMinutes: 0,
                needsReviewDailyAllocation: null,
                allTaskInSprintMinutes: 0,
                allTaskTotalMinutes: 0,
                dailySummaries: null,
              },
              projectionToken: "projection-1",
            },
          }),
          { status: 200 },
        ),
      ),
    );
    const gateway = createHTTPSprintsGateway("http://api.test/api");

    const suggestion = await gateway.suggest({
      startDate: "2026-08-04",
      endDate: "2026-08-08",
      memberIds: ["member-1"],
    });

    expect(suggestion.members[0]).toMatchObject({
      totalAllocationMinutes: 120,
      dailySummaries: [
        {
          date: "2026-08-04",
          capacityMinutes: 480,
          selectedAllocationMinutes: 120,
          remainingMinutes: 360,
          overcapacityMinutes: 0,
        },
      ],
    });
    expect(suggestion.tasks[0]?.task).toMatchObject({
      projectPriority: 2,
      wbsPath: "1.2",
      wbsRank: 3,
      dailyPlanOrderDate: "2026-08-04",
      warnings: [],
      allocations: [],
    });
    expect(suggestion.totals.needsReviewDailyAllocation).toEqual([]);
    expect(suggestion.totals.dailySummaries).toEqual([]);
  });

  it("retains structured overlap details from the stable API error_AC65", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            code: "SPRINT_MEMBER_OVERLAP",
            message: "overlap",
            details: {
              sprintId: "sprint-2",
              sprintName: "Release Sprint",
              startDate: "2026-08-08",
              endDate: "2026-08-15",
              members: [{ id: "member-1", name: "Harry" }],
            },
          }),
          { status: 409 },
        ),
      ),
    );
    const gateway = createHTTPSprintsGateway("http://api.test/api");

    const error = await gateway
      .create({
        name: "August",
        startDate: "2026-08-04",
        endDate: "2026-08-10",
        memberIds: ["member-1"],
        taskIds: [],
      })
      .catch((reason: unknown) => reason);

    expect(error).toBeInstanceOf(SprintAPIError);
    if (!(error instanceof SprintAPIError))
      throw new Error("expected API error");
    expect(error.details).toMatchObject({
      sprintId: "sprint-2",
      sprintName: "Release Sprint",
      members: [{ id: "member-1", name: "Harry" }],
    });
  });
});
