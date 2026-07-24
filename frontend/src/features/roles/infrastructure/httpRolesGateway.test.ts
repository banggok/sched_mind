import { afterEach, describe, expect, it, vi } from 'vitest'

import { createHTTPRolesGateway, RolesAPIError } from './httpRolesGateway'

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('HTTP roles gateway', () => {
  it('maps API DTOs into domain roles', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            data: [
              {
                id: 'role-id',
                name: 'Backend',
                createdAt: '2026-07-24T10:00:00Z',
                updatedAt: '2026-07-24T10:00:00Z',
              },
            ],
          }),
          { status: 200 },
        ),
      ),
    )

    const roles = await createHTTPRolesGateway(
      'https://api.example.test/v1',
    ).list()

    expect(roles[0]).toMatchObject({ id: 'role-id', name: 'Backend' })
    expect(roles[0].createdAt).toBeInstanceOf(Date)
    expect(fetch).toHaveBeenCalledWith(
      'https://api.example.test/v1/roles',
      expect.any(Object),
    )
  })

  it('maps API errors without exposing raw DTOs', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            code: 'ROLE_IN_USE',
            message:
              'Role is assigned to one or more team members and cannot be deleted',
          }),
          { status: 409 },
        ),
      ),
    )

    await expect(
      createHTTPRolesGateway('/configured-api').delete('role-id'),
    ).rejects.toEqual(
      new RolesAPIError(
        'ROLE_IN_USE',
        'Role is assigned to one or more team members and cannot be deleted',
      ),
    )
  })
})
