import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'

import type { RolesGateway } from '../application/rolesGateway'
import type { Role } from '../domain/role'
import { RolesDashboardPage } from './RolesDashboardPage'

describe('RolesDashboardPage', () => {
  it('shows the available workspace navigation and current location', async () => {
    const user = userEvent.setup()
    const fixture = gatewayFixture([])
    render(<RolesDashboardPage gateway={fixture.gateway} />)

    const toggle = screen.getByRole('button', { name: 'Expand navigation' })
    expect(toggle.getAttribute('aria-expanded')).toBe('false')
    await user.click(toggle)
    expect(
      screen.getByRole('button', { name: 'Collapse navigation' }).getAttribute(
        'aria-expanded',
      ),
    ).toBe('true')

    const navigation = screen.getByRole('navigation', {
      name: 'Team Management',
    })
    const current = within(navigation).getByRole('link', { name: 'Roles' })

    expect(current.getAttribute('href')).toBe('#roles')
    expect(current.getAttribute('aria-current')).toBe('page')
    expect(
      within(screen.getByRole('navigation', { name: 'Breadcrumb' })).getByText(
        'Roles',
      ),
    ).toBeTruthy()
    expect(await screen.findByText('No roles yet')).toBeTruthy()
  })

  it('renders empty state, validates input, and creates a role', async () => {
    const user = userEvent.setup()
    const fixture = gatewayFixture([])
    render(<RolesDashboardPage gateway={fixture.gateway} />)

    expect(await screen.findByText('No roles yet')).toBeTruthy()
    await user.click(screen.getAllByRole('button', { name: 'Add Role' })[0])
    await user.click(screen.getByRole('button', { name: 'Add role' }))

    expect(await screen.findByText('Role name is required')).toBeTruthy()
    expect(fixture.gateway.create).not.toHaveBeenCalled()

    await user.type(screen.getByLabelText('Role name'), '  Backend  ')
    await user.click(screen.getByRole('button', { name: 'Add role' }))

    expect(
      await screen.findByText('Role created successfully'),
    ).toBeTruthy()
    expect(fixture.gateway.create).toHaveBeenCalledWith('Backend')
    expect(await screen.findByText('Backend')).toBeTruthy()
  })

  it('edits a role and refreshes the list', async () => {
    const user = userEvent.setup()
    const fixture = gatewayFixture([role('backend', 'Backend')])
    render(<RolesDashboardPage gateway={fixture.gateway} />)

    await user.click(
      await screen.findByRole('button', { name: 'Edit Backend' }),
    )
    const input = screen.getByLabelText('Role name')
    await user.clear(input)
    await user.type(input, 'Backend Engineer')
    await user.click(screen.getByRole('button', { name: 'Save changes' }))

    expect(
      await screen.findByText('Role updated successfully'),
    ).toBeTruthy()
    expect(fixture.gateway.update).toHaveBeenCalledWith(
      'backend',
      'Backend Engineer',
    )
    expect(await screen.findByText('Backend Engineer')).toBeTruthy()
  })

  it('shows the specified duplicate-name message without exposing API details', async () => {
    const user = userEvent.setup()
    const fixture = gatewayFixture([])
    fixture.gateway.create = vi
      .fn()
      .mockRejectedValue({
        code: 'ROLE_NAME_ALREADY_EXISTS',
        message: 'unique index roles_name_ci_unique',
      })
    render(<RolesDashboardPage gateway={fixture.gateway} />)

    await screen.findByText('No roles yet')
    await user.click(screen.getAllByRole('button', { name: 'Add Role' })[0])
    await user.type(screen.getByLabelText('Role name'), 'Backend')
    await user.click(screen.getByRole('button', { name: 'Add role' }))

    expect(await screen.findByText('Role name already exists')).toBeTruthy()
    expect(screen.queryByText('unique index roles_name_ci_unique')).toBeNull()
  })

  it('cancels deletion and reports an in-use failure', async () => {
    const user = userEvent.setup()
    const fixture = gatewayFixture([role('backend', 'Backend')])
    fixture.gateway.delete = vi
      .fn()
      .mockRejectedValue({ code: 'ROLE_IN_USE', message: 'database detail' })
    render(<RolesDashboardPage gateway={fixture.gateway} />)

    await user.click(
      await screen.findByRole('button', { name: 'Delete Backend' }),
    )
    const firstDialog = screen.getByRole('alertdialog')
    await user.click(within(firstDialog).getByRole('button', { name: 'Cancel' }))
    expect(fixture.gateway.delete).not.toHaveBeenCalled()

    await user.click(screen.getByRole('button', { name: 'Delete Backend' }))
    const secondDialog = screen.getByRole('alertdialog')
    await user.click(
      within(secondDialog).getByRole('button', { name: 'Delete role' }),
    )

    expect(
      await screen.findByText(
        'Role is assigned to one or more team members and cannot be deleted',
      ),
    ).toBeTruthy()
    expect(screen.getByRole('alertdialog')).toBeTruthy()
    expect(screen.queryByText('database detail')).toBeNull()
    expect(fixture.roles).toHaveLength(1)
  })

  it('maps a technical load failure and retries near the affected content', async () => {
    const user = userEvent.setup()
    const fixture = gatewayFixture([role('backend', 'Backend')])
    fixture.gateway.list = vi
      .fn()
      .mockRejectedValueOnce(new Error('postgres connection refused'))
      .mockResolvedValueOnce([...fixture.roles])

    render(<RolesDashboardPage gateway={fixture.gateway} />)

    expect(await screen.findByText('Unable to load roles')).toBeTruthy()
    expect(screen.queryByText('postgres connection refused')).toBeNull()
    await user.click(screen.getByRole('button', { name: 'Try again' }))
    expect(await screen.findByText('Backend')).toBeTruthy()
  })

  it('prevents duplicate submission while create is pending', async () => {
    const user = userEvent.setup()
    const fixture = gatewayFixture([])
    let resolveCreate: ((role: Role) => void) | undefined
    fixture.gateway.create = vi.fn(
      () =>
        new Promise<Role>((resolve) => {
          resolveCreate = resolve
        }),
    )
    render(<RolesDashboardPage gateway={fixture.gateway} />)

    await screen.findByText('No roles yet')
    await user.click(screen.getAllByRole('button', { name: 'Add Role' })[0])
    await user.type(screen.getByLabelText('Role name'), 'Backend')
    const submit = screen.getByRole('button', { name: 'Add role' })
    await user.click(submit)

    expect(
      (screen.getByRole('button', {
        name: 'Saving…',
      }) as HTMLButtonElement).disabled,
    ).toBe(true)
    await user.click(screen.getByRole('button', { name: 'Saving…' }))
    expect(fixture.gateway.create).toHaveBeenCalledTimes(1)

    if (!resolveCreate) {
      throw new Error('create promise was not started')
    }
    resolveCreate(role('backend', 'Backend'))
    expect(
      await screen.findByText('Role created successfully'),
    ).toBeTruthy()
  })

  it('supports Escape and restores focus without losing a dirty draft silently', async () => {
    const user = userEvent.setup()
    const fixture = gatewayFixture([])
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(false)
    render(<RolesDashboardPage gateway={fixture.gateway} />)

    await screen.findByText('No roles yet')
    const trigger = screen.getAllByRole('button', { name: 'Add Role' })[0]
    await user.click(trigger)
    const input = screen.getByLabelText('Role name')
    expect(document.activeElement).toBe(input)

    await user.type(input, 'Backend')
    await user.keyboard('{Escape}')
    expect(confirm).toHaveBeenCalledWith('Discard your unsaved role changes?')
    expect(screen.getByRole('dialog')).toBeTruthy()
    expect((input as HTMLInputElement).value).toBe('Backend')

    confirm.mockReturnValue(true)
    await user.keyboard('{Escape}')
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
    expect(document.activeElement).toBe(trigger)
    confirm.mockRestore()
  })
})

function gatewayFixture(initialRoles: Role[]) {
  const roles = [...initialRoles]
  const gateway: RolesGateway = {
    list: vi.fn().mockImplementation(async () => [...roles]),
    create: vi.fn().mockImplementation(async (name: string) => {
      const created = role(`role-${roles.length + 1}`, name)
      roles.push(created)
      return created
    }),
    update: vi.fn().mockImplementation(async (id: string, name: string) => {
      const index = roles.findIndex((item) => item.id === id)
      const updated = { ...roles[index], name, updatedAt: new Date() }
      roles[index] = updated
      return updated
    }),
    delete: vi.fn().mockImplementation(async (id: string) => {
      const index = roles.findIndex((item) => item.id === id)
      roles.splice(index, 1)
    }),
  }
  return { gateway, roles }
}

function role(id: string, name: string): Role {
  return {
    id,
    name,
    createdAt: new Date('2026-07-24T10:00:00Z'),
    updatedAt: new Date('2026-07-24T10:00:00Z'),
  }
}
