import type { TeamMember, TeamMemberInput } from "../domain/teamMember";
import type {
  PageQuery,
  PageResult,
} from "../../../shared/application/pagination";

export interface TeamMembersGateway {
  list(query: PageQuery, signal?: AbortSignal): Promise<PageResult<TeamMember>>;
  create(input: Required<TeamMemberInput>): Promise<TeamMember>;
  update(id: string, input: Required<TeamMemberInput>): Promise<TeamMember>;
  delete(id: string): Promise<void>;
}
