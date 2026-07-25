import {
  normalizeTeamMemberInput,
  sortTeamMembers,
  type TeamMember,
  type TeamMemberInput,
} from '../domain/teamMember'
import type { TeamMembersGateway } from './teamMembersGateway'
import type { PageQuery, PageResult } from '../../../shared/application/pagination'

export async function listTeamMembers(
  gateway: TeamMembersGateway,
  query: PageQuery,
  signal?: AbortSignal,
): Promise<PageResult<TeamMember>> {
  const result = await gateway.list(query, signal)
  return { ...result, items: sortTeamMembers(result.items) }
}

export function createTeamMember(
  gateway: TeamMembersGateway,
  input: TeamMemberInput,
): Promise<TeamMember> {
  return gateway.create(normalizeTeamMemberInput(input))
}

export function updateTeamMember(
  gateway: TeamMembersGateway,
  id: string,
  input: TeamMemberInput,
): Promise<TeamMember> {
  return gateway.update(id, normalizeTeamMemberInput(input))
}

export function deleteTeamMember(
  gateway: TeamMembersGateway,
  id: string,
): Promise<void> {
  return gateway.delete(id)
}
