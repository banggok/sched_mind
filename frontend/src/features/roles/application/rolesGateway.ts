import type { Role } from '../domain/role'

export interface RolesGateway {
  list(signal?: AbortSignal): Promise<Role[]>
  create(name: string): Promise<Role>
  update(id: string, name: string): Promise<Role>
  delete(id: string): Promise<void>
}
