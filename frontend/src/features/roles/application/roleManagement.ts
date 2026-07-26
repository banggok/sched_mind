import { normalizeRoleName, sortRolesByName, type Role } from "../domain/role";
import type { RolesGateway } from "./rolesGateway";
import type {
  PageQuery,
  PageResult,
} from "../../../shared/application/pagination";

export async function listRoles(
  gateway: RolesGateway,
  query: PageQuery,
  signal?: AbortSignal,
): Promise<PageResult<Role>> {
  const result = await gateway.list(query, signal);
  return { ...result, items: sortRolesByName(result.items) };
}

export async function createRole(
  gateway: RolesGateway,
  name: string,
): Promise<Role> {
  return gateway.create(normalizeRoleName(name));
}

export async function updateRole(
  gateway: RolesGateway,
  id: string,
  name: string,
): Promise<Role> {
  return gateway.update(id, normalizeRoleName(name));
}

export async function deleteRole(
  gateway: RolesGateway,
  id: string,
): Promise<void> {
  return gateway.delete(id);
}
