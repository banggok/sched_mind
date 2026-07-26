import type {
  PageQuery,
  PageResult,
} from "../../../shared/application/pagination";

export type MemberRoleOption = { id: string; name: string };

export interface RoleOptionsGateway {
  list(
    query: PageQuery,
    signal?: AbortSignal,
  ): Promise<PageResult<MemberRoleOption>>;
}
