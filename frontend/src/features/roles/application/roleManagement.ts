import { normalizeRoleName, sortRolesByName, type Role } from '../domain/role'
import type { RolesGateway } from './rolesGateway'

export async function listRoles(
  gateway: RolesGateway,
  signal?: AbortSignal,
): Promise<Role[]> {
  return sortRolesByName(await gateway.list(signal))
}

export async function createRole(
  gateway: RolesGateway,
  name: string,
): Promise<Role> {
  return gateway.create(normalizeRoleName(name))
}

export async function updateRole(
  gateway: RolesGateway,
  id: string,
  name: string,
): Promise<Role> {
  return gateway.update(id, normalizeRoleName(name))
}

export async function deleteRole(
  gateway: RolesGateway,
  id: string,
): Promise<void> {
  return gateway.delete(id)
}
