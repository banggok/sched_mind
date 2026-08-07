import type {
  RoleAuditGateway,
  RoleMemberUsage,
  RolesGateway,
} from "../application/rolesGateway";
import type { Role } from "../domain/role";
import { RequestCache } from "../../../shared/infrastructure/RequestCache";

interface RoleDTO {
  id: string;
  name: string;
  createdAt: string;
  updatedAt: string;
}

interface RoleMemberUsageDTO {
  id: string;
  name: string;
}

interface ListRoleMembersDTO {
  data: RoleMemberUsageDTO[];
  page: number;
  pageSize: number;
  total: number;
}

interface ListRolesDTO {
  data: RoleDTO[];
  page: number;
  pageSize: number;
  total: number;
}

interface ErrorDTO {
  code: string;
  message: string;
  field?: string;
}

export class RolesAPIError extends Error {
  constructor(
    readonly code: string,
    message: string,
    readonly field?: string,
  ) {
    super(message);
  }
}

export function createHTTPRolesGateway(
  apiBaseURL: string,
  onRolesChanged: () => void = () => undefined,
): RoleAuditGateway {
  const listRequests = new Map<
    string,
    RequestCache<
      ReturnType<RolesGateway["list"]> extends Promise<infer T> ? T : never
    >
  >();
  const invalidateLists = () => {
    listRequests.forEach((request) => request.invalidate());
    listRequests.clear();
  };
  return {
    async list(query, signal) {
      const parameters = new URLSearchParams({
        search: query.search,
        page: String(query.page),
        pageSize: String(query.pageSize),
      });
      const key = parameters.toString();
      const listRequest = listRequests.get(key) ?? new RequestCache();
      listRequests.set(key, listRequest);
      return listRequest.run(async () => {
        const response = await fetch(`${apiBaseURL}/roles?${parameters}`, {});
        const payload = await readResponse<ListRolesDTO>(response);
        return {
          items: payload.data.map(mapRole),
          page: payload.page,
          pageSize: payload.pageSize,
          total: payload.total,
        };
      }, signal);
    },
    async listMembers(roleId, query, signal) {
      const parameters = new URLSearchParams({
        search: query.search,
        page: String(query.page),
        pageSize: String(query.pageSize),
      });
      const response = await fetch(
        `${apiBaseURL}/roles/${encodeURIComponent(roleId)}/members?${parameters}`,
        { signal },
      );
      const payload = await readResponse<ListRoleMembersDTO>(response);
      return {
        items: payload.data.map(mapRoleMemberUsage),
        page: payload.page,
        pageSize: payload.pageSize,
        total: payload.total,
      };
    },
    async create(name) {
      const response = await fetch(`${apiBaseURL}/roles`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name }),
      });
      const role = mapRole(await readResponse<RoleDTO>(response));
      invalidateLists();
      onRolesChanged();
      return role;
    },
    async update(id, name) {
      const response = await fetch(
        `${apiBaseURL}/roles/${encodeURIComponent(id)}`,
        {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ name }),
        },
      );
      const role = mapRole(await readResponse<RoleDTO>(response));
      invalidateLists();
      onRolesChanged();
      return role;
    },
    async delete(id) {
      const response = await fetch(
        `${apiBaseURL}/roles/${encodeURIComponent(id)}`,
        {
          method: "DELETE",
        },
      );
      if (!response.ok) {
        await throwAPIError(response);
      }
      invalidateLists();
      onRolesChanged();
    },
  };
}

async function readResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    await throwAPIError(response);
  }
  return (await response.json()) as T;
}

async function throwAPIError(response: Response): Promise<never> {
  let payload: ErrorDTO | undefined;
  try {
    payload = (await response.json()) as ErrorDTO;
  } catch {
    throw new RolesAPIError(
      "UNEXPECTED_RESPONSE",
      "The server returned an unexpected response",
    );
  }
  throw new RolesAPIError(payload.code, payload.message, payload.field);
}

function mapRole(dto: RoleDTO): Role {
  return {
    id: dto.id,
    name: dto.name,
    createdAt: new Date(dto.createdAt),
    updatedAt: new Date(dto.updatedAt),
  };
}

function mapRoleMemberUsage(dto: RoleMemberUsageDTO): RoleMemberUsage {
  return { id: dto.id, name: dto.name };
}
