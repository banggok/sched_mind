import type { Role } from "../domain/role";
import type {
  PageQuery,
  PageResult,
} from "../../../shared/application/pagination";

export interface RoleMemberUsage {
  id: string;
  name: string;
}

export interface RolesGateway {
  list(query: PageQuery, signal?: AbortSignal): Promise<PageResult<Role>>;
  create(name: string): Promise<Role>;
  update(id: string, name: string): Promise<Role>;
  delete(id: string): Promise<void>;
}

export interface RoleAuditGateway extends RolesGateway {
  listMembers(
    roleId: string,
    query: PageQuery,
    signal?: AbortSignal,
  ): Promise<PageResult<RoleMemberUsage>>;
}
