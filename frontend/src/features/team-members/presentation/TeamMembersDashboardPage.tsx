import { useEffect, useRef, useState } from 'react'
import { Breadcrumb } from '../../../app/Breadcrumb'
import { PageContent } from '../../../app/PageContent'
import { SearchField } from '../../../shared/presentation/SearchField'
import { PaginationControls } from '../../../shared/presentation/PaginationControls'
import { useDebouncedValue } from '../../../shared/presentation/useDebouncedValue'
import { ListSkeleton } from '../../../shared/presentation/ListSkeleton'
import type { RolesGateway } from '../../roles/application/rolesGateway'
import type { Role } from '../../roles/domain/role'
import { createTeamMember, deleteTeamMember, listTeamMembers, updateTeamMember } from '../application/teamMemberManagement'
import type { TeamMembersGateway } from '../application/teamMembersGateway'
import { calculateCommitmentCapacity, normalizeTeamMemberInput, type TeamMember } from '../domain/teamMember'

export function TeamMembersDashboardPage({ gateway, rolesGateway }: { gateway: TeamMembersGateway; rolesGateway: RolesGateway }) {
  const [members, setMembers] = useState<TeamMember[]>([])
  const [roles, setRoles] = useState<Role[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [editing, setEditing] = useState<TeamMember | null | undefined>()
  const [form, setForm] = useState({
    name: '',
    roleId: '',
    roleName: '',
    dailyCapacity: '8',
    bufferPercentage: '20',
  })
  const [saving, setSaving] = useState(false)
  const [roleListOpen, setRoleListOpen] = useState(false)
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const debouncedSearch = useDebouncedValue(search.trim())
  const searchPending = search.trim() !== debouncedSearch
  const dailyCapacityInputRef = useRef<HTMLInputElement>(null)

  async function load() {
    setLoading(true); setError('')
    try {
      const [memberPage, rolePage] = await Promise.all([
        listTeamMembers(gateway, { search: debouncedSearch, page, pageSize: 5 }),
        rolesGateway.list({ search: '', page: 1, pageSize: 100 }),
      ])
      setMembers(memberPage.items); setTotal(memberPage.total); setRoles(rolePage.items)
    } catch { setError('Members could not be loaded. Please try again.') }
    finally { setLoading(false) }
  }
  useEffect(() => { void load() }, [debouncedSearch, page])

  function open(member?: TeamMember) {
    setEditing(member ?? null)
    setForm(member ? {
      name: member.name,
      roleId: member.role.id,
      roleName: member.role.name,
      dailyCapacity: String(member.dailyCapacity),
      bufferPercentage: String(member.bufferPercentage),
    } : {
      name: '',
      roleId: '',
      roleName: '',
      dailyCapacity: '8',
      bufferPercentage: '20',
    })
  }
  async function save(event: React.FormEvent) {
    event.preventDefault(); setError('')
    let input
    try {
      const normalizedName = capitalizeWords(form.name)
      const dailyCapacity = roundToHalfDraft(form.dailyCapacity)
      const bufferPercentage = roundToHalfDraft(form.bufferPercentage)
      setForm((current) => ({
        ...current,
        name: normalizedName,
        dailyCapacity,
        bufferPercentage,
      }))
      const selectedRole = findRoleByName(roles, form.roleName)
      if (!selectedRole || selectedRole.id !== form.roleId) {
        throw new Error('Select an available role')
      }
      input = normalizeTeamMemberInput({
        name: normalizedName,
        roleId: selectedRole.id,
        dailyCapacity: parseDecimalDraft(dailyCapacity),
        bufferPercentage: parseDecimalDraft(bufferPercentage),
      })
    } catch (reason) { setError(reason instanceof Error ? reason.message : 'Check the form values.'); return }
    setSaving(true)
    try {
      if (editing) await updateTeamMember(gateway, editing.id, input)
      else await createTeamMember(gateway, input)
      setEditing(undefined); await load()
    } catch { setError('The member could not be saved. Your input has been preserved.') }
    finally { setSaving(false) }
  }
  async function remove(member: TeamMember) {
    if (!window.confirm(`Delete ${member.name}? This cannot be undone.`)) return
    try { await deleteTeamMember(gateway, member.id); await load() }
    catch { setError('This member could not be deleted. Remove task assignments or capacity overrides first.') }
  }
  const filteredRoles = filterRoles(roles, form.roleName)

  return <>
    <PageContent>
      <div className="flex items-center justify-between gap-4"><Breadcrumb activePage="team-members" /><button className="rounded-lg bg-[#2A93D6] px-4 py-2.5 text-sm font-extrabold text-white" onClick={() => open()}>+ Add Member</button></div>
      {error && <p className="mt-5 rounded-xl bg-rose-50 p-4 font-bold text-rose-700" role="alert">{error}</p>}
      <section className="mt-6 overflow-hidden rounded-3xl border border-[#EBF0F5] bg-white shadow-sm">
        <div className="flex flex-col gap-4 border-b border-[#EBF0F5] px-6 py-5 sm:flex-row sm:items-center sm:justify-between">
          <h2 className="text-lg font-black">Members</h2>
          <SearchField label="Search members" value={search} onChange={(value) => { setSearch(value); setPage(1) }} />
        </div>
        {loading || searchPending ? <ListSkeleton label="Loading members" /> : members.length === 0 && debouncedSearch === '' ? <div className="p-12 text-center">
          <span aria-hidden="true" className="mx-auto grid size-14 place-items-center rounded-2xl bg-[#E5F5FF] text-[#2A93D6]">
            <svg className="size-7" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round">
              <circle cx="9" cy="8" r="3" />
              <path d="M3.5 19c.5-3.5 2.3-5.2 5.5-5.2s5 1.7 5.5 5.2M16 7.5a2.5 2.5 0 0 1 0 5M16.5 14.5c2.4.4 3.7 1.9 4 4.5" />
            </svg>
          </span>
          <h3 className="mt-5 text-xl font-black">No members yet</h3>
          <p className="mx-auto mt-2 max-w-sm text-[#6D6E70]">Add the first person to start planning capacity.</p>
          <button className="mt-5 text-sm font-extrabold text-[#2A93D6]" onClick={() => open()}>Add Member</button>
        </div> : members.length === 0 ? <div className="p-12 text-center">
          <h3 className="text-xl font-black">No matching members</h3>
          <p className="mt-2 text-[#6D6E70]">Try another search term.</p>
          <button className="mt-5 text-sm font-extrabold text-[#2A93D6]" onClick={() => { setSearch(''); setPage(1) }}>Clear search</button>
        </div> :
          <ul className="divide-y divide-[#EBF0F5]">{members.map(member => <li className="flex flex-col gap-3 px-6 py-5 sm:flex-row sm:items-center sm:justify-between" key={member.id}><div><h3 className="font-extrabold">{member.name}</h3><p className="text-sm text-[#6D6E70]">{member.role.name} · {member.dailyCapacity}h daily · {member.bufferPercentage}% buffer · <strong>{member.commitmentCapacity}h commitment</strong></p></div><div className="flex gap-2"><button className="rounded-lg border px-3 py-2 text-sm font-bold" onClick={() => open(member)}>Edit</button><button className="rounded-lg px-3 py-2 text-sm font-bold text-rose-600" onClick={() => void remove(member)}>Delete</button></div></li>)}</ul>}
        {!loading && !searchPending && total > 0 ? <PaginationControls page={page} pageSize={5} total={total} onPageChange={setPage} /> : null}
      </section>
    </PageContent>
    {editing !== undefined && <div className="fixed inset-0 z-50 grid place-items-center bg-[#0C162D]/45 p-4"><form className="w-full max-w-lg rounded-3xl bg-white p-6 shadow-2xl" onSubmit={save}><h2 className="text-xl font-black">{editing ? 'Edit member' : 'Add member'}</h2>
      <label className="mt-5 block text-sm font-bold">Name<input autoFocus required maxLength={100} className="mt-2 w-full rounded-xl border p-3" value={form.name} onChange={e => setForm({...form, name:e.target.value})} onBlur={() => setForm({...form, name:capitalizeWords(form.name)})}/></label>
      <div className="relative mt-4">
      <label className="block text-sm font-bold">Role<input required type="search" role="combobox" aria-autocomplete="list" aria-expanded={roleListOpen} aria-controls="member-role-options" autoComplete="off" placeholder="Search roles" className="mt-2 w-full rounded-xl border p-3" value={form.roleName} onFocus={() => setRoleListOpen(true)} onBlur={() => setRoleListOpen(false)} onChange={e => { const roleName = e.target.value; setForm({...form, roleName, roleId: findRoleByName(roles, roleName)?.id ?? ''}); setRoleListOpen(true) }} onKeyDown={event => {
        if (event.key !== 'Enter') return
        event.preventDefault()
        const role = filteredRoles[0]
        if (role) {
          setForm({...form, roleName: role.name, roleId: role.id})
          setRoleListOpen(false)
          dailyCapacityInputRef.current?.focus()
        }
      }}/></label>
      {roleListOpen ? <ul id="member-role-options" role="listbox" className="absolute z-10 mt-1 max-h-44 w-full overflow-auto rounded-xl border border-[#D3DCE5] bg-white p-1 shadow-xl">
        {filteredRoles.length ? filteredRoles.map(role => <li key={role.id} role="option" aria-selected={role.id === form.roleId}><button type="button" className="w-full rounded-lg px-3 py-2 text-left text-sm hover:bg-[#E5F5FF]" onMouseDown={event => event.preventDefault()} onClick={() => { setForm({...form, roleName:role.name, roleId:role.id}); setRoleListOpen(false); dailyCapacityInputRef.current?.focus() }}>{role.name}</button></li>) : <li className="px-3 py-2 text-sm text-[#6D6E70]">No roles found</li>}
      </ul> : null}
      </div>
      <div className="grid grid-cols-2 gap-4"><label className="mt-4 block text-sm font-bold">Daily capacity (hours)<input ref={dailyCapacityInputRef} required type="text" inputMode="decimal" className="mt-2 w-full rounded-xl border p-3" value={form.dailyCapacity} onChange={e => setForm({...form, dailyCapacity:e.target.value})} onBlur={() => setForm({...form, dailyCapacity:roundToHalfDraft(form.dailyCapacity)})}/></label><label className="mt-4 block text-sm font-bold">Buffer (%)<input required type="text" inputMode="decimal" className="mt-2 w-full rounded-xl border p-3" value={form.bufferPercentage} onChange={e => setForm({...form, bufferPercentage:e.target.value})} onBlur={() => setForm({...form, bufferPercentage:roundToHalfDraft(form.bufferPercentage)})}/></label></div>
      <p className="mt-4 rounded-xl bg-[#E5F5FF] p-3 text-sm font-bold">Commitment capacity: {commitmentPreview(form.dailyCapacity, form.bufferPercentage)} hours/day</p>
      <div className="mt-6 flex justify-end gap-3"><button type="button" disabled={saving} className="rounded-xl border px-4 py-2" onClick={() => setEditing(undefined)}>Cancel</button><button disabled={saving} className="rounded-xl bg-[#2A93D6] px-4 py-2 font-bold text-white">{saving ? 'Saving…' : 'Save'}</button></div>
    </form></div>}
  </>
}

function findRoleByName(roles: Role[], name: string): Role | undefined {
  const normalized = name.trim().toLocaleLowerCase()
  return roles.find(
    (role) => role.name.toLocaleLowerCase() === normalized,
  )
}

function capitalizeWords(value: string): string {
  return value.replace(
    /(^|[\s-])(\p{L})/gu,
    (_, boundary: string, letter: string) =>
      `${boundary}${letter.toLocaleUpperCase()}`,
  )
}

function roundToHalfDraft(value: string): string {
  const parsed = parseDecimalDraft(value)
  if (parsed === undefined || !Number.isFinite(parsed)) return value
  return String(Math.round(parsed * 2) / 2)
}

function filterRoles(roles: Role[], query: string): Role[] {
  const normalized = query.trim().toLocaleLowerCase()
  if (!normalized) return roles
  return roles.filter((role) =>
    role.name.toLocaleLowerCase().startsWith(normalized),
  )
}

function parseDecimalDraft(value: string): number | undefined {
  const normalized = value.trim()
  if (!normalized) return undefined
  const parsed = Number(normalized)
  return Number.isFinite(parsed) ? parsed : Number.NaN
}

function commitmentPreview(dailyDraft: string, bufferDraft: string): string {
  const daily = parseDecimalDraft(dailyDraft)
  const buffer = parseDecimalDraft(bufferDraft)
  if (
    daily === undefined ||
    buffer === undefined ||
    !Number.isFinite(daily) ||
    !Number.isFinite(buffer)
  ) {
    return '—'
  }
  return String(calculateCommitmentCapacity(daily, buffer))
}
