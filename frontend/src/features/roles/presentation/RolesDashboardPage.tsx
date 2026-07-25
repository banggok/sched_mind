import { useState } from 'react'

import { Breadcrumb } from '../../../app/Breadcrumb'
import { PageContent } from '../../../app/PageContent'
import { SearchField } from '../../../shared/presentation/SearchField'
import { PaginationControls } from '../../../shared/presentation/PaginationControls'
import { ListSkeleton } from '../../../shared/presentation/ListSkeleton'
import { useDebouncedValue } from '../../../shared/presentation/useDebouncedValue'
import type { RolesGateway } from '../application/rolesGateway'
import type { Role } from '../domain/role'
import { DeleteRoleDialog } from './DeleteRoleDialog'
import { RoleFormDialog } from './RoleFormDialog'
import { useRoleManagement } from './useRoleManagement'

export function RolesDashboardPage({
  gateway,
}: {
  gateway: RolesGateway
}) {
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const debouncedSearch = useDebouncedValue(search.trim())
  const searchPending = search.trim() !== debouncedSearch
  const management = useRoleManagement(gateway, {
    search: debouncedSearch,
    page,
    pageSize: 5,
  })

  return (
    <>
      <PageContent>
        <section className="min-w-0">
          <div className="flex items-center justify-between gap-4">
            <Breadcrumb activePage="roles" />
            <button
              className="shrink-0 rounded-lg bg-[#2A93D6] px-4 py-2.5 text-sm font-extrabold text-white shadow-md shadow-[#2A93D6]/20 hover:bg-[#0C4DA2] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#0C4DA2]"
              onClick={management.openCreate}
            >
              + Add Role
            </button>
          </div>

          <section className="mt-6 overflow-hidden rounded-3xl border border-[#EBF0F5] bg-white shadow-sm">
            <div className="flex flex-col gap-4 border-b border-[#EBF0F5] px-6 py-5 sm:flex-row sm:items-center sm:justify-between">
              <h2 className="text-lg font-black">Roles</h2>
              <SearchField
                label="Search roles"
                value={search}
                onChange={(value) => { setSearch(value); setPage(1) }}
              />
            </div>

            {management.loading || searchPending ? (
              <ListSkeleton label="Loading roles" />
            ) : management.pageError ? (
              <div className="p-10 text-center">
                <p className="font-bold text-rose-600" role="alert">
                  {management.pageError}
                </p>
                <button
                  className="mt-4 text-sm font-extrabold text-[#2A93D6]"
                  onClick={() => void management.retry()}
                >
                  Try again
                </button>
              </div>
            ) : management.roles.length === 0 && debouncedSearch === '' ? (
              <div className="p-12 text-center">
                <span
                  aria-hidden="true"
                  className="mx-auto grid size-14 place-items-center rounded-2xl bg-[#E5F5FF] text-2xl text-[#2A93D6]"
                >
                  ◫
                </span>
                <h3 className="mt-5 text-xl font-black">No roles yet</h3>
                <p className="mx-auto mt-2 max-w-sm text-[#6D6E70]">
                  Add the first team role to start organizing your team.
                </p>
                <button
                  className="mt-5 text-sm font-extrabold text-[#2A93D6]"
                  onClick={management.openCreate}
                >
                  Add Role
                </button>
              </div>
            ) : management.roles.length === 0 ? (
              <div className="p-12 text-center">
                <h3 className="text-xl font-black">No matching roles</h3>
                <p className="mt-2 text-[#6D6E70]">
                  Try another search term.
                </p>
                <button
                  className="mt-5 text-sm font-extrabold text-[#2A93D6]"
                  onClick={() => { setSearch(''); setPage(1) }}
                >
                  Clear search
                </button>
              </div>
            ) : (
              <ul className="divide-y divide-[#EBF0F5]">
                {management.roles.map((role) => (
                  <RoleRow
                    key={role.id}
                    role={role}
                    onEdit={() => management.openEdit(role)}
                    onDelete={() => management.openDelete(role)}
                  />
                ))}
              </ul>
            )}
            {!management.loading && !searchPending && !management.pageError && management.total > 0 ? (
              <PaginationControls page={page} pageSize={5} total={management.total} onPageChange={setPage} />
            ) : null}
          </section>
        </section>
      </PageContent>

      {management.form ? (
        <RoleFormDialog
          mode={management.form.mode}
          role={management.form.role}
          name={management.name}
          error={management.fieldError}
          submitting={management.submitting}
          onNameChange={management.setName}
          onSubmit={() => void management.submitForm()}
          onClose={management.closeForm}
        />
      ) : null}

      {management.deleting ? (
        <DeleteRoleDialog
          role={management.deleting}
          error={management.deleteError}
          submitting={management.submitting}
          onCancel={management.closeDelete}
          onConfirm={() => void management.confirmDelete()}
        />
      ) : null}

      {management.notification ? (
        <div
          className="fixed right-5 bottom-5 z-50 flex max-w-sm items-center gap-4 rounded-2xl bg-[#0C162D] px-5 py-4 text-sm font-bold text-white shadow-2xl"
          role="status"
        >
          <span>{management.notification}</span>
          <button
            type="button"
            className="grid size-8 shrink-0 place-items-center rounded-lg text-lg hover:bg-white/10 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white"
            aria-label="Dismiss notification"
            onClick={() => management.setNotification('')}
          >
            ×
          </button>
        </div>
      ) : null}
    </>
  )
}

function RoleRow({
  role,
  onEdit,
  onDelete,
}: {
  role: Role
  onEdit(): void
  onDelete(): void
}) {
  return (
    <li className="flex flex-col items-stretch justify-between gap-4 px-6 py-5 sm:flex-row sm:items-center">
      <div className="flex min-w-0 items-center gap-4">
        <span className="grid size-11 shrink-0 place-items-center rounded-xl bg-[#E5F5FF] font-black text-[#2A93D6]">
          {role.name.slice(0, 1).toUpperCase()}
        </span>
        <div className="min-w-0">
          <h3 className="truncate font-extrabold">{role.name}</h3>
          <p className="mt-1 text-xs text-[#98999A]">
            Updated {formatRoleDate(role.updatedAt)}
          </p>
        </div>
      </div>
      <div className="flex gap-2 self-end sm:self-auto">
        <button
          className="rounded-xl border border-[#D3DCE5] px-3 py-2 text-sm font-bold hover:border-[#2A93D6] hover:text-[#2A93D6]"
          aria-label={`Edit ${role.name}`}
          onClick={onEdit}
        >
          Edit
        </button>
        <button
          className="rounded-xl px-3 py-2 text-sm font-bold text-rose-600 hover:bg-rose-50"
          aria-label={`Delete ${role.name}`}
          onClick={onDelete}
        >
          Delete
        </button>
      </div>
    </li>
  )
}

const roleDateFormatter = new Intl.DateTimeFormat('en-GB', {
  day: 'numeric',
  month: 'short',
  year: 'numeric',
})

function formatRoleDate(date: Date): string {
  return roleDateFormatter.format(date)
}
