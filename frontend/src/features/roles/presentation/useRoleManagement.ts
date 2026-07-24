import { useCallback, useEffect, useState } from 'react'

import {
  createRole,
  deleteRole,
  listRoles,
  updateRole,
} from '../application/roleManagement'
import type { RolesGateway } from '../application/rolesGateway'
import { RoleNameError, type Role } from '../domain/role'

type FormMode = 'create' | 'edit'

interface FormState {
  mode: FormMode
  role?: Role
}

export function useRoleManagement(gateway: RolesGateway) {
  const [roles, setRoles] = useState<Role[]>([])
  const [loading, setLoading] = useState(true)
  const [pageError, setPageError] = useState('')
  const [form, setForm] = useState<FormState>()
  const [name, setName] = useState('')
  const [fieldError, setFieldError] = useState('')
  const [deleting, setDeleting] = useState<Role>()
  const [deleteError, setDeleteError] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [notification, setNotification] = useState('')

  const load = useCallback(
    async (signal?: AbortSignal) => {
      try {
        const result = await listRoles(gateway, signal)
        setRoles(result)
        setPageError('')
      } catch (error) {
        if (!(error instanceof DOMException && error.name === 'AbortError')) {
          setPageError(errorMessage(error, 'Unable to load roles'))
        }
      } finally {
        if (!signal?.aborted) {
          setLoading(false)
        }
      }
    },
    [gateway],
  )

  useEffect(() => {
    const controller = new AbortController()
    void load(controller.signal)
    return () => controller.abort()
  }, [load])

  useEffect(() => {
    if (!notification) {
      return
    }
    const timeout = window.setTimeout(() => setNotification(''), 5000)
    return () => window.clearTimeout(timeout)
  }, [notification])

  function openCreate() {
    setForm({ mode: 'create' })
    setName('')
    setFieldError('')
  }

  function openEdit(role: Role) {
    setForm({ mode: 'edit', role })
    setName(role.name)
    setFieldError('')
  }

  function openDelete(role: Role) {
    setDeleting(role)
    setDeleteError('')
  }

  function closeDelete() {
    if (!submitting) {
      setDeleting(undefined)
      setDeleteError('')
    }
  }

  function closeForm() {
    if (!submitting) {
      setForm(undefined)
      setFieldError('')
    }
  }

  async function submitForm() {
    if (!form) {
      return
    }

    setSubmitting(true)
    setFieldError('')
    try {
      if (form.mode === 'create') {
        await createRole(gateway, name)
        setNotification('Role created successfully')
      } else if (form.role) {
        await updateRole(gateway, form.role.id, name)
        setNotification('Role updated successfully')
      }
      setForm(undefined)
      await load()
    } catch (error) {
      setFieldError(
        errorMessage(
          error,
          form.mode === 'create'
            ? 'Unable to create role'
            : 'Unable to update role',
        ),
      )
    } finally {
      setSubmitting(false)
    }
  }

  async function confirmDelete() {
    if (!deleting) {
      return
    }

    setSubmitting(true)
    setDeleteError('')
    try {
      await deleteRole(gateway, deleting.id)
      setNotification('Role deleted successfully')
      setDeleting(undefined)
      await load()
    } catch (error) {
      setDeleteError(
        errorMessage(
          error,
          'Unable to delete this role. Check your connection and try again.',
        ),
      )
    } finally {
      setSubmitting(false)
    }
  }

  return {
    roles,
    loading,
    pageError,
    form,
    name,
    fieldError,
    deleting,
    deleteError,
    submitting,
    notification,
    setName,
    setNotification,
    openCreate,
    openEdit,
    openDelete,
    closeDelete,
    closeForm,
    submitForm,
    confirmDelete,
    retry: load,
  }
}

function errorMessage(error: unknown, fallback: string): string {
  if (error instanceof RoleNameError) {
    return error.message
  }
  const code =
    typeof error === 'object' &&
    error !== null &&
    'code' in error &&
    typeof error.code === 'string'
      ? error.code
      : ''
  switch (code) {
    case 'ROLE_NAME_REQUIRED':
      return 'Role name is required'
    case 'ROLE_NAME_TOO_LONG':
      return 'Role name must not exceed 100 characters'
    case 'ROLE_NAME_INVALID':
      return 'Role name contains unsupported characters'
    case 'ROLE_NAME_ALREADY_EXISTS':
      return 'Role name already exists'
    case 'ROLE_NOT_FOUND':
      return 'This role no longer exists. Refresh the list and try again.'
    case 'ROLE_IN_USE':
      return 'Role is assigned to one or more team members and cannot be deleted'
    default:
      return fallback
  }
}
