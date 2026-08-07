import { afterEach, describe, expect, it, vi } from "vitest";
import {
  type SchedulingImpact,
  SchedulingImpactCancelledError,
  schedulingImpactFetch,
  schedulingImpactTokenHeader,
  subscribeSchedulingImpact,
} from "./schedulingImpactFetch";

const confirmationPayload = {
  code: "SCHEDULING_IMPACT_CONFIRMATION_REQUIRED",
  message: "review impact",
  details: {
    token: "token-1",
    lockedProjects: [],
    openProjects: [{ id: "project-b", name: "Project B" }],
  },
};

afterEach(() => {
  vi.restoreAllMocks();
});

describe("schedulingImpactFetch", () => {
  it("previews, confirms, and retries the same mutation with the impact token", async () => {
    const fetchMock = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValueOnce(
        new Response(JSON.stringify(confirmationPayload), {
          status: 409,
          headers: { "Content-Type": "application/json" },
        }),
      )
      .mockResolvedValueOnce(new Response(null, { status: 204 }));
    const unsubscribe = subscribeSchedulingImpact((_impact, resolve) =>
      resolve("confirm"),
    );

    const response = await schedulingImpactFetch("/api/tasks", {
      method: "POST",
      body: "payload",
    });

    expect(response.status).toBe(204);
    expect(fetchMock).toHaveBeenCalledTimes(2);
    const retry = fetchMock.mock.calls[1][1];
    expect(new Headers(retry?.headers).get(schedulingImpactTokenHeader)).toBe(
      "token-1",
    );
    expect(retry?.body).toBe("payload");
    unsubscribe();
  });

  it("US-6.2 D04 AC-24 replaces stale timeline-impact preview and confirms with the new token", async () => {
    const stalePayload = {
      code: "SCHEDULING_IMPACT_STALE",
      message: "scheduling impact changed after preview",
      details: {
        token: "token-2",
        lockedProjects: [],
        openProjects: [{ id: "project-c", name: "Project C" }],
      },
    };
    const fetchMock = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValueOnce(
        new Response(JSON.stringify(confirmationPayload), {
          status: 409,
          headers: { "Content-Type": "application/json" },
        }),
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify(stalePayload), {
          status: 409,
          headers: { "Content-Type": "application/json" },
        }),
      )
      .mockResolvedValueOnce(new Response(null, { status: 204 }));
    const reviewed: string[] = [];
    const unsubscribe = subscribeSchedulingImpact((impact, resolve) => {
      reviewed.push(
        `${impact.code}:${impact.openProjects
          .map((project) => project.id)
          .join(",")}`,
      );
      resolve("confirm");
    });

    const response = await schedulingImpactFetch("/api/tasks", {
      method: "POST",
      body: "payload",
    });

    expect(response.status).toBe(204);
    expect(reviewed).toEqual([
      "SCHEDULING_IMPACT_CONFIRMATION_REQUIRED:project-b",
      "SCHEDULING_IMPACT_STALE:project-c",
    ]);
    expect(fetchMock).toHaveBeenCalledTimes(3);
    expect(
      new Headers(fetchMock.mock.calls[1][1]?.headers).get(
        schedulingImpactTokenHeader,
      ),
    ).toBe("token-1");
    expect(
      new Headers(fetchMock.mock.calls[2][1]?.headers).get(
        schedulingImpactTokenHeader,
      ),
    ).toBe("token-2");
    unsubscribe();
  });

  it("does not retry a cancelled mutation", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify(confirmationPayload), {
        status: 409,
        headers: { "Content-Type": "application/json" },
      }),
    );
    const unsubscribe = subscribeSchedulingImpact((_impact, resolve) =>
      resolve("cancel"),
    );

    await expect(
      schedulingImpactFetch("/api/tasks", { method: "POST" }),
    ).rejects.toBeInstanceOf(SchedulingImpactCancelledError);
    expect(fetchMock).toHaveBeenCalledTimes(1);
    unsubscribe();
  });

  it("returns locked-impact responses without retrying the blocked mutation", async () => {
    const payload = {
      ...confirmationPayload,
      code: "SCHEDULING_LOCKED_PROJECT_IMPACT",
      details: {
        ...confirmationPayload.details,
        lockedProjects: [{ id: "locked", name: "Locked" }],
        openProjects: [],
      },
    };
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify(payload), {
        status: 409,
        headers: { "Content-Type": "application/json" },
      }),
    );
    const unsubscribe = subscribeSchedulingImpact((_impact, resolve) =>
      resolve("cancel"),
    );

    const response = await schedulingImpactFetch("/api/tasks", {
      method: "POST",
    });

    expect(response.status).toBe(409);
    expect(fetchMock).toHaveBeenCalledTimes(1);
    unsubscribe();
  });
});

describe("Group scheduling impact", () => {
  it("US-4.4 AC-22/AC-23 parses qualified Group impacts and forwards reopen closure", async () => {
    const closurePayload = {
      code: "SCHEDULING_SCOPE_REOPEN_CLOSURE_REQUIRED",
      message: "reopen closure required",
      details: {
        token: "group-token",
        lockedProjects: [{ id: "project-b", name: "Project B" }],
        openProjects: [],
        lockedGroups: [
          {
            id: "group-b",
            projectId: "project-b",
            name: "Backend",
            path: "Project B / Platform / Backend",
          },
        ],
        openGroups: [
          {
            id: "group-c",
            projectId: "project-c",
            name: "Client",
            path: "Project C / Client",
          },
        ],
      },
    };
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify(closurePayload), {
        status: 409,
        headers: { "Content-Type": "application/json" },
      }),
    );
    const reviewed: string[] = [];
    const unsubscribe = subscribeSchedulingImpact((impact, resolve) => {
      reviewed.push(
        `${impact.token}:${impact.lockedGroups[0]?.path}:${impact.openGroups[0]?.path}`,
      );
      resolve("reopen-all");
    });
    const reopenAll = vi.fn(async (impact: SchedulingImpact) => {
      expect(impact.token).toBe("group-token");
      return new Response(null, { status: 200 });
    });

    const response = await schedulingImpactFetch(
      "/api/groups/group-a/status",
      { method: "POST" },
      { reopenAll },
    );

    expect(response.status).toBe(200);
    expect(reviewed).toEqual([
      "group-token:Project B / Platform / Backend:Project C / Client",
    ]);
    expect(reopenAll).toHaveBeenCalledTimes(1);
    unsubscribe();
  });
});
