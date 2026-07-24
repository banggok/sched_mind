import type { RolesGateway } from '../application/rolesGateway'
import type { Role } from '../domain/role'

interface RoleDTO {
  id: string
  name: string
  createdAt: string
  updatedAt: string
}

interface ListRolesDTO {
  data: RoleDTO[]
}

interface ErrorDTO {
  code: string
  message: string
  field?: string
}

export class RolesAPIError extends Error {
  constructor(
    readonly code: string,
    message: string,
    readonly field?: string,
  ) {
    super(message)
  }
}

export function createHTTPRolesGateway(apiBaseURL: string): RolesGateway {
  return {
    async list(signal) {
      const response = await fetch(`${apiBaseURL}/roles`, { signal })
      const payload = await readResponse<ListRolesDTO>(response)
      return payload.data.map(mapRole)
    },
    async create(name) {
      const response = await fetch(`${apiBaseURL}/roles`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name }),
      })
      return mapRole(await readResponse<RoleDTO>(response))
    },
    async update(id, name) {
      const response = await fetch(`${apiBaseURL}/roles/${encodeURIComponent(id)}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name }),
      })
      return mapRole(await readResponse<RoleDTO>(response))
    },
    async delete(id) {
      const response = await fetch(`${apiBaseURL}/roles/${encodeURIComponent(id)}`, {
        method: 'DELETE',
      })
      if (!response.ok) {
        await throwAPIError(response)
      }
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
  let payload: ErrorDTO | undefined
  try {
    payload = (await response.json()) as ErrorDTO
  } catch {
    throw new RolesAPIError(
      'UNEXPECTED_RESPONSE',
      'The server returned an unexpected response',
    )
  }
  throw new RolesAPIError(payload.code, payload.message, payload.field)
}

function mapRole(dto: RoleDTO): Role {
  return {
    id: dto.id,
    name: dto.name,
    createdAt: new Date(dto.createdAt),
    updatedAt: new Date(dto.updatedAt),
  }
}
