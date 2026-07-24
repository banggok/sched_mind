import { useState } from 'react'

import schedMindAppIcon from '../../../assets/schedmind-app-icon.png'
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
  const management = useRoleManagement(gateway)
  const [sidebarOpen, setSidebarOpen] = useState(false)

  return (
    <div className="min-h-screen bg-[#F7F9FC] text-[#101828]">
      {sidebarOpen ? (
        <button
          type="button"
          className="fixed inset-0 z-30 bg-[#0C162D]/35 lg:hidden"
          aria-label="Close navigation"
          onClick={() => setSidebarOpen(false)}
        />
      ) : null}

      <button
        type="button"
        className="fixed top-4 left-[22px] z-50 grid size-9 place-items-center rounded-xl border border-[#D3DCE5] bg-white font-bold shadow-sm hover:border-[#2A93D6] hover:text-[#2A93D6]"
        aria-label={sidebarOpen ? 'Collapse navigation' : 'Expand navigation'}
        aria-expanded={sidebarOpen}
        aria-controls="application-sidebar"
        onClick={() => setSidebarOpen((open) => !open)}
      >
        ☰
      </button>

      <aside
        id="application-sidebar"
        className={`fixed inset-y-0 left-0 z-40 flex w-64 flex-col border-r border-[#EBF0F5] bg-white shadow-xl transition-transform duration-200 motion-reduce:transition-none ${
          sidebarOpen ? 'translate-x-0' : '-translate-x-full'
        }`}
        aria-label="Application sidebar"
      >
        <div className="h-[69px] shrink-0 border-b border-[#EBF0F5]" />

        <nav className="flex-1 p-3" aria-label="Team Management">
          <p className="overflow-hidden px-3 pt-2 pb-2 text-xs font-extrabold tracking-[0.14em] whitespace-nowrap text-[#98999A] uppercase">
            Team Management
          </p>
          <a
            className="flex items-center gap-3 rounded-xl bg-[#E5F5FF] px-3 py-3 text-sm font-extrabold text-[#0C4DA2]"
            href="#roles"
            aria-current="page"
            aria-label="Roles"
            title={sidebarOpen ? undefined : 'Roles'}
            onClick={() => {
              if (window.innerWidth < 1024) {
                setSidebarOpen(false)
              }
            }}
          >
            <span
              aria-hidden="true"
              className="grid size-8 shrink-0 place-items-center rounded-lg bg-white text-[#2A93D6]"
            >
              <RoleIcon />
            </span>
            <span>Roles</span>
          </a>
        </nav>
      </aside>

      <main
        className={`min-h-screen transition-[margin] duration-200 motion-reduce:transition-none ${
          sidebarOpen ? 'lg:ml-64' : 'ml-0'
        }`}
      >
      <header className="border-b border-[#EBF0F5] bg-white px-5 py-4 sm:px-8 lg:px-12">
        <div className="mx-auto flex h-9 max-w-7xl items-center gap-3 pl-14">
          <span
            className="grid size-9 shrink-0 place-items-center overflow-hidden rounded-xl"
            aria-hidden="true"
          >
            <img
              className="size-full scale-[1.3]"
              src={schedMindAppIcon}
              alt=""
            />
          </span>
          <span className="font-black">SchedMind</span>
        </div>
      </header>

      <div className="mx-auto max-w-7xl px-5 py-8 sm:px-8 lg:px-12">
        <section id="roles" className="min-w-0">
          <div className="flex items-center justify-between gap-4">
            <nav aria-label="Breadcrumb">
              <ol className="flex items-center gap-2 text-sm font-bold">
                <li className="text-[#6D6E70]">Team Management</li>
                <li aria-hidden="true" className="text-[#B2BAC4]">
                  /
                </li>
                <li>
                  <h1 className="text-[#101828]" aria-current="page">
                    Roles
                  </h1>
                </li>
              </ol>
            </nav>
            <button
              className="shrink-0 rounded-lg bg-[#2A93D6] px-4 py-2.5 text-sm font-extrabold text-white shadow-md shadow-[#2A93D6]/20 hover:bg-[#0C4DA2] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#0C4DA2]"
              onClick={management.openCreate}
            >
              + Add Role
            </button>
          </div>

          <section className="mt-6 overflow-hidden rounded-3xl border border-[#EBF0F5] bg-white shadow-sm">
            <div className="flex items-center justify-between border-b border-[#EBF0F5] px-6 py-5">
              <h2 className="text-lg font-black">Team roles</h2>
            </div>

            {management.loading ? (
              <div
                className="space-y-3 p-6"
                aria-label="Loading roles"
                aria-live="polite"
              >
                {[1, 2, 3].map((item) => (
                  <div
                    key={item}
                    className="h-16 animate-pulse rounded-2xl bg-[#F2F2F2] motion-reduce:animate-none"
                  />
                ))}
              </div>
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
            ) : management.roles.length === 0 ? (
              <div className="p-12 text-center">
                <span
                  aria-hidden="true"
                  className="mx-auto grid size-14 place-items-center rounded-2xl bg-[#E5F5FF] text-2xl text-[#2A93D6]"
                >
                  ◫
                </span>
                <h3 className="mt-5 text-xl font-black">No roles yet</h3>
                <p className="mx-auto mt-2 max-w-sm text-[#6D6E70]">
                  Add the first engineering role to start organizing your team.
                </p>
                <button
                  className="mt-5 text-sm font-extrabold text-[#2A93D6]"
                  onClick={management.openCreate}
                >
                  Add Role
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
          </section>
        </section>
      </div>

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
      </main>
    </div>
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

function RoleIcon() {
  return (
    <svg
      className="size-5"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <rect x="3" y="5" width="18" height="14" rx="3" />
      <circle cx="9" cy="11" r="2.25" />
      <path d="M5.75 16c.7-1.6 1.8-2.4 3.25-2.4s2.55.8 3.25 2.4" />
      <path d="M15 10h3" />
      <path d="M15 14h2" />
    </svg>
  )
}
