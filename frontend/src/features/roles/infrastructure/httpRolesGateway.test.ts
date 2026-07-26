import { afterEach, describe, expect, it, vi } from "vitest";

import { createHTTPRolesGateway, RolesAPIError } from "./httpRolesGateway";
import { createHTTPTeamMembersGateway } from "../../team-members/infrastructure/httpTeamMembersGateway";

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("HTTP roles gateway", () => {
  it("maps API DTOs into domain roles", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            data: [
              {
                id: "role-id",
                name: "Backend",
                createdAt: "2026-07-24T10:00:00Z",
                updatedAt: "2026-07-24T10:00:00Z",
              },
            ],
            page: 1,
            pageSize: 20,
            total: 1,
          }),
          { status: 200 },
        ),
      ),
    );

    const roles = await createHTTPRolesGateway(
      "https://api.example.test/v1",
    ).list(query);

    expect(roles.items[0]).toMatchObject({ id: "role-id", name: "Backend" });
    expect(roles.items[0].createdAt).toBeInstanceOf(Date);
    expect(fetch).toHaveBeenCalledWith(
      "https://api.example.test/v1/roles?search=&page=1&pageSize=20",
      expect.any(Object),
    );
  });

  it("reuses list results until a mutation invalidates them", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(
        new Response(
          JSON.stringify({ data: [], page: 1, pageSize: 20, total: 0 }),
          { status: 200 },
        ),
      );
    vi.stubGlobal("fetch", fetchMock);
    const gateway = createHTTPRolesGateway("/configured-api");

    await Promise.all([gateway.list(query), gateway.list(query)]);
    await gateway.list(query);

    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("loads a fresh list after a successful mutation", async () => {
    const roleDTO = {
      id: "backend",
      name: "Backend",
      createdAt: "2026-07-25T00:00:00Z",
      updatedAt: "2026-07-25T00:00:00Z",
    };
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({ data: [], page: 1, pageSize: 20, total: 0 }),
          { status: 200 },
        ),
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify(roleDTO), { status: 201 }),
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({ data: [roleDTO], page: 1, pageSize: 20, total: 1 }),
          { status: 200 },
        ),
      );
    vi.stubGlobal("fetch", fetchMock);
    const gateway = createHTTPRolesGateway("/configured-api");

    await gateway.list(query);
    await gateway.create("Backend");
    const roles = await gateway.list(query);

    expect(fetchMock).toHaveBeenCalledTimes(3);
    expect(roles.items).toHaveLength(1);
  });

  it("invalidates cached member projections after a role is renamed", async () => {
    let roleName = "Backend";
    const fetchMock = vi
      .fn()
      .mockImplementation(
        async (input: string | URL | Request, init?: RequestInit) => {
          const url = String(input);
          if (url.includes("/team-members?")) {
            return new Response(
              JSON.stringify({
                data: [
                  {
                    id: "member-id",
                    name: "Harry",
                    role: { id: "role-id", name: roleName },
                    dailyCapacity: 8,
                    bufferPercentage: 20,
                    commitmentCapacity: 6.5,
                    createdAt: "2026-07-25T00:00:00Z",
                    updatedAt: "2026-07-25T00:00:00Z",
                  },
                ],
                page: 1,
                pageSize: 5,
                total: 1,
              }),
              { status: 200 },
            );
          }
          if (url.endsWith("/roles/role-id") && init?.method === "PUT") {
            roleName = "Backend Engineer";
            return new Response(
              JSON.stringify({
                id: "role-id",
                name: roleName,
                createdAt: "2026-07-25T00:00:00Z",
                updatedAt: "2026-07-25T01:00:00Z",
              }),
              { status: 200 },
            );
          }
          throw new Error(`Unexpected request: ${url}`);
        },
      );
    vi.stubGlobal("fetch", fetchMock);
    const membersGateway = createHTTPTeamMembersGateway("/configured-api");
    const rolesGateway = createHTTPRolesGateway(
      "/configured-api",
      membersGateway.invalidateListCache,
    );
    const memberQuery = { search: "", page: 1, pageSize: 5 };

    const beforeRename = await membersGateway.list(memberQuery);
    await rolesGateway.update("role-id", "Backend Engineer");
    const afterRename = await membersGateway.list(memberQuery);

    expect(beforeRename.items[0].role.name).toBe("Backend");
    expect(afterRename.items[0].role.name).toBe("Backend Engineer");
    expect(
      fetchMock.mock.calls.filter(([url]) =>
        String(url).includes("/team-members?"),
      ),
    ).toHaveLength(2);
  });

  it("maps API errors without exposing raw DTOs", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            code: "ROLE_IN_USE",
            message:
              "Role is assigned to one or more team members and cannot be deleted",
          }),
          { status: 409 },
        ),
      ),
    );

    await expect(
      createHTTPRolesGateway("/configured-api").delete("role-id"),
    ).rejects.toEqual(
      new RolesAPIError(
        "ROLE_IN_USE",
        "Role is assigned to one or more team members and cannot be deleted",
      ),
    );
  });
});

const query = { search: "", page: 1, pageSize: 20 };
