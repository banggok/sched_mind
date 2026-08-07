import { afterEach, describe, expect, it, vi } from "vitest";
import {
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
