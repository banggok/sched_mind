import type { TeamMembersGateway } from '../application/teamMembersGateway'
import type { TeamMember, TeamMemberInput } from '../domain/teamMember'
import { RequestCache } from '../../../shared/infrastructure/RequestCache'

interface TeamMemberDTO {
  id: string
  name: string
  role: { id: string; name: string }
  dailyCapacity: number
  bufferPercentage: number
  commitmentCapacity: number
  createdAt: string
  updatedAt: string
}

interface ListDTO {
  data: TeamMemberDTO[]
  page: number
  pageSize: number
  total: number
}

interface ItemDTO {
  data: TeamMemberDTO
}

interface ErrorDTO {
  code: string
  message: string
  field?: string
}

export class TeamMembersAPIError extends Error {
  constructor(
    readonly code: string,
    message: string,
    readonly field?: string,
  ) {
    super(message)
  }
}

export interface HTTPTeamMembersGateway extends TeamMembersGateway {
  invalidateListCache(): void
}

export function createHTTPTeamMembersGateway(
  apiBaseURL: string,
): HTTPTeamMembersGateway {
  const listRequests = new Map<string, RequestCache<ReturnType<TeamMembersGateway['list']> extends Promise<infer T> ? T : never>>()
  const invalidateLists = () => {
    listRequests.forEach((request) => request.invalidate())
    listRequests.clear()
  }
  return {
    invalidateListCache: invalidateLists,
    async list(query, signal) {
      const parameters = new URLSearchParams({
        search: query.search,
        page: String(query.page),
        pageSize: String(query.pageSize),
      })
      const key = parameters.toString()
      const listRequest = listRequests.get(key) ?? new RequestCache()
      listRequests.set(key, listRequest)
      return listRequest.run(async () => {
        const response = await fetch(`${apiBaseURL}/team-members?${parameters}`, {})
        const payload = await readResponse<ListDTO>(response)
        return {
          items: payload.data.map(mapTeamMember),
          page: payload.page,
          pageSize: payload.pageSize,
          total: payload.total,
        }
      }, signal)
    },
    async create(input) {
      const response = await fetch(`${apiBaseURL}/team-members`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(input),
      })
      const member = mapTeamMember((await readResponse<ItemDTO>(response)).data)
      invalidateLists()
      return member
    },
    async update(id, input) {
      const response = await fetch(
        `${apiBaseURL}/team-members/${encodeURIComponent(id)}`,
        {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(input),
        },
      )
      const member = mapTeamMember((await readResponse<ItemDTO>(response)).data)
      invalidateLists()
      return member
    },
    async delete(id) {
      const response = await fetch(
        `${apiBaseURL}/team-members/${encodeURIComponent(id)}`,
        { method: 'DELETE' },
      )
      if (!response.ok) {
        await throwAPIError(response)
      }
      invalidateLists()
    },
  }
}

async function readResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    await throwAPIError(response)
  }
  return (await response.json()) as T
}

async function throwAPIError(response: Response): Promise<never> {
  try {
    const payload = (await response.json()) as ErrorDTO
    throw new TeamMembersAPIError(payload.code, payload.message, payload.field)
  } catch (error) {
    if (error instanceof TeamMembersAPIError) {
      throw error
    }
    throw new TeamMembersAPIError(
      'UNEXPECTED_RESPONSE',
      'The server returned an unexpected response',
    )
  }
}

function mapTeamMember(dto: TeamMemberDTO): TeamMember {
  return {
    ...dto,
    role: { ...dto.role },
    createdAt: new Date(dto.createdAt),
    updatedAt: new Date(dto.updatedAt),
  }
}
