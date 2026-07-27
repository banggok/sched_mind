import { useCallback, useEffect, useRef, useState } from "react";
import { Breadcrumb } from "../../../app/Breadcrumb";
import { PageContent } from "../../../app/PageContent";
import { SearchField } from "../../../shared/presentation/SearchField";
import { PaginationControls } from "../../../shared/presentation/PaginationControls";
import { useDebouncedValue } from "../../../shared/presentation/useDebouncedValue";
import { ListSkeleton } from "../../../shared/presentation/ListSkeleton";
import {
  createTeamMember,
  deleteTeamMember,
  listTeamMembers,
  updateTeamMember,
} from "../application/teamMemberManagement";
import type { TeamMembersGateway } from "../application/teamMembersGateway";
import {
  calculateCommitmentCapacity,
  normalizeTeamMemberInput,
  type TeamMember,
} from "../domain/teamMember";
import type {
  MemberRoleOption,
  RoleOptionsGateway,
} from "../application/roleOptionsGateway";
import { Button } from "../../../shared/presentation/Button";
import { Dialog } from "../../../shared/presentation/Dialog";
import { EmptyState } from "../../../shared/presentation/EmptyState";
import { ListSurface } from "../../../shared/presentation/ListSurface";
import { Alert } from "../../../shared/presentation/Alert";
import { FormField } from "../../../shared/presentation/FormField";
import {
  parseDecimalDraft,
  roundToHalfDraft,
} from "../../../shared/presentation/decimalDraft";

export function TeamMembersDashboardPage({
  gateway,
  roleOptionsGateway,
  onManageCapacity,
}: {
  gateway: TeamMembersGateway;
  roleOptionsGateway: RoleOptionsGateway;
  onManageCapacity(member: TeamMember): void;
}) {
  const [members, setMembers] = useState<TeamMember[]>([]);
  const [roles, setRoles] = useState<MemberRoleOption[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [editing, setEditing] = useState<TeamMember | null | undefined>();
  const [form, setForm] = useState({
    name: "",
    roleId: "",
    roleName: "",
    dailyCapacity: "8",
    bufferPercentage: "20",
  });
  const [saving, setSaving] = useState(false);
  const [deletingMember, setDeletingMember] = useState<TeamMember>();
  const [roleListOpen, setRoleListOpen] = useState(false);
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const debouncedSearch = useDebouncedValue(search.trim());
  const searchPending = search.trim() !== debouncedSearch;
  const dailyCapacityInputRef = useRef<HTMLInputElement>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const [memberPage, rolePage] = await Promise.all([
        listTeamMembers(gateway, {
          search: debouncedSearch,
          page,
          pageSize: 5,
        }),
        roleOptionsGateway.list({ search: "", page: 1, pageSize: 100 }),
      ]);
      setMembers(memberPage.items);
      setTotal(memberPage.total);
      setRoles(rolePage.items);
    } catch {
      setError("Members could not be loaded. Please try again.");
    } finally {
      setLoading(false);
    }
  }, [debouncedSearch, gateway, page, roleOptionsGateway]);
  useEffect(() => {
    void load();
  }, [load]);

  function open(member?: TeamMember) {
    setEditing(member ?? null);
    setForm(
      member
        ? {
            name: member.name,
            roleId: member.role.id,
            roleName: member.role.name,
            dailyCapacity: String(member.dailyCapacity),
            bufferPercentage: String(member.bufferPercentage),
          }
        : {
            name: "",
            roleId: "",
            roleName: "",
            dailyCapacity: "8",
            bufferPercentage: "20",
          },
    );
  }
  async function save(event: React.FormEvent) {
    event.preventDefault();
    setError("");
    let input;
    try {
      const normalizedName = capitalizeWords(form.name);
      const dailyCapacity = roundToHalfDraft(form.dailyCapacity);
      const bufferPercentage = roundToHalfDraft(form.bufferPercentage);
      setForm((current) => ({
        ...current,
        name: normalizedName,
        dailyCapacity,
        bufferPercentage,
      }));
      const selectedRole = findRoleByName(roles, form.roleName);
      if (!selectedRole || selectedRole.id !== form.roleId) {
        throw new Error("Select an available role");
      }
      input = normalizeTeamMemberInput({
        name: normalizedName,
        roleId: selectedRole.id,
        dailyCapacity: parseDecimalDraft(dailyCapacity),
        bufferPercentage: parseDecimalDraft(bufferPercentage),
      });
    } catch (reason) {
      setError(
        reason instanceof Error ? reason.message : "Check the form values.",
      );
      return;
    }
    setSaving(true);
    try {
      if (editing) await updateTeamMember(gateway, editing.id, input);
      else await createTeamMember(gateway, input);
      setEditing(undefined);
      await load();
    } catch {
      setError("The member could not be saved. Your input has been preserved.");
    } finally {
      setSaving(false);
    }
  }
  async function remove(member: TeamMember) {
    if (saving) return;
    setSaving(true);
    try {
      await deleteTeamMember(gateway, member.id);
      setDeletingMember(undefined);
      await load();
    } catch {
      setError(
        "This member could not be deleted. If they have active tasks, reassign them and try again.",
      );
    } finally {
      setSaving(false);
    }
  }
  const filteredRoles = filterRoles(roles, form.roleName);

  return (
    <>
      <PageContent>
        <div className="flex items-center justify-between gap-4">
          <Breadcrumb activePage="team-members" />
          <Button variant="primary" onClick={() => open()}>
            + Add Member
          </Button>
        </div>
        {error && editing === undefined && (
          <Alert tone="danger" className="mt-5">
            {error}
          </Alert>
        )}
        <ListSurface
          title="Members"
          controls={
            <SearchField
              label="Search members"
              value={search}
              onChange={(value) => {
                setSearch(value);
                setPage(1);
              }}
            />
          }
        >
          {loading || searchPending ? (
            <ListSkeleton label="Loading members" />
          ) : members.length === 0 && debouncedSearch === "" ? (
            <EmptyState
              title="No members yet"
              description="Add the first person to start planning capacity."
              icon={
                <svg
                  className="size-7"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="1.8"
                  strokeLinecap="round"
                >
                  <circle cx="9" cy="8" r="3" />
                  <path d="M3.5 19c.5-3.5 2.3-5.2 5.5-5.2s5 1.7 5.5 5.2M16 7.5a2.5 2.5 0 0 1 0 5M16.5 14.5c2.4.4 3.7 1.9 4 4.5" />
                </svg>
              }
              action={
                <Button variant="quiet" onClick={() => open()}>
                  Add Member
                </Button>
              }
            />
          ) : members.length === 0 ? (
            <EmptyState
              title="No matching members"
              description="Try another search term."
              action={
                <Button
                  variant="quiet"
                  onClick={() => {
                    setSearch("");
                    setPage(1);
                  }}
                >
                  Clear search
                </Button>
              }
            />
          ) : (
            <ul className="divide-y divide-border-subtle">
              {members.map((member) => (
                <li
                  className="flex flex-col gap-3 px-6 py-5 sm:flex-row sm:items-center sm:justify-between"
                  key={member.id}
                >
                  <div>
                    <h3 className="font-extrabold">{member.name}</h3>
                    <p className="text-sm text-muted">
                      {member.role.name} · {member.dailyCapacity}h daily ·{" "}
                      {member.bufferPercentage}% buffer ·{" "}
                      <strong>{member.commitmentCapacity}h commitment</strong>
                    </p>
                  </div>
                  <div className="flex flex-wrap gap-2">
                    <Button compact onClick={() => onManageCapacity(member)}>
                      Capacity Overrides
                    </Button>
                    <Button compact onClick={() => open(member)}>
                      Edit
                    </Button>
                    <Button
                      compact
                      variant="danger"
                      onClick={() => setDeletingMember(member)}
                    >
                      Delete
                    </Button>
                  </div>
                </li>
              ))}
            </ul>
          )}
          {!loading && !searchPending && total > 0 ? (
            <PaginationControls
              page={page}
              pageSize={5}
              total={total}
              onPageChange={setPage}
            />
          ) : null}
        </ListSurface>
      </PageContent>
      {editing !== undefined && (
        <Dialog
          titleID="member-form-title"
          onClose={() => {
            if (!saving) setEditing(undefined);
          }}
        >
          <form onSubmit={save} noValidate>
            <h2 id="member-form-title" className="text-dialog-title font-black">
              {editing ? "Edit member" : "Add member"}
            </h2>
            {error ? (
              <Alert tone="danger" className="mt-4">
                {error}
              </Alert>
            ) : null}
            <FormField
              id="member-name"
              label="Name"
              className="mt-5"
              autoFocus
              required
              maxLength={100}
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              onBlur={() =>
                setForm({ ...form, name: capitalizeWords(form.name) })
              }
            />
            <div className="relative mt-4">
              <label className="block text-sm font-bold">
                Role
                <input
                  required
                  type="search"
                  role="combobox"
                  aria-autocomplete="list"
                  aria-expanded={roleListOpen}
                  aria-controls="member-role-options"
                  autoComplete="off"
                  placeholder="Search roles"
                  className="ui-input mt-2"
                  value={form.roleName}
                  onFocus={() => setRoleListOpen(true)}
                  onBlur={() => setRoleListOpen(false)}
                  onChange={(e) => {
                    const roleName = e.target.value;
                    setForm({
                      ...form,
                      roleName,
                      roleId: findRoleByName(roles, roleName)?.id ?? "",
                    });
                    setRoleListOpen(true);
                  }}
                  onKeyDown={(event) => {
                    if (event.key !== "Enter") return;
                    event.preventDefault();
                    const role = filteredRoles[0];
                    if (role) {
                      setForm({
                        ...form,
                        roleName: role.name,
                        roleId: role.id,
                      });
                      setRoleListOpen(false);
                      dailyCapacityInputRef.current?.focus();
                    }
                  }}
                />
              </label>
              {roleListOpen ? (
                <ul
                  id="member-role-options"
                  role="listbox"
                  className="layer-popover absolute mt-1 max-h-44 w-full overflow-auto rounded-control border border-border-strong bg-surface p-1 shadow-floating"
                >
                  {filteredRoles.length ? (
                    filteredRoles.map((role) => (
                      <li key={role.id}>
                        <button
                          type="button"
                          role="option"
                          aria-selected={role.id === form.roleId}
                          className="w-full rounded-action px-3 py-2 text-left text-sm hover:bg-brand-soft"
                          onMouseDown={(event) => event.preventDefault()}
                          onClick={() => {
                            setForm({
                              ...form,
                              roleName: role.name,
                              roleId: role.id,
                            });
                            setRoleListOpen(false);
                            dailyCapacityInputRef.current?.focus();
                          }}
                        >
                          {role.name}
                        </button>
                      </li>
                    ))
                  ) : (
                    <li className="px-3 py-2 text-sm text-muted">
                      No roles found
                    </li>
                  )}
                </ul>
              ) : null}
            </div>
            <div className="grid grid-cols-2 gap-4">
              <label className="mt-4 block text-sm font-bold">
                Daily capacity (hours)
                <input
                  ref={dailyCapacityInputRef}
                  required
                  type="text"
                  inputMode="decimal"
                  className="ui-input mt-2"
                  value={form.dailyCapacity}
                  onChange={(e) =>
                    setForm({ ...form, dailyCapacity: e.target.value })
                  }
                  onBlur={() =>
                    setForm({
                      ...form,
                      dailyCapacity: roundToHalfDraft(form.dailyCapacity),
                    })
                  }
                />
              </label>
              <label className="mt-4 block text-sm font-bold">
                Buffer (%)
                <input
                  required
                  type="text"
                  inputMode="decimal"
                  className="ui-input mt-2"
                  value={form.bufferPercentage}
                  onChange={(e) =>
                    setForm({ ...form, bufferPercentage: e.target.value })
                  }
                  onBlur={() =>
                    setForm({
                      ...form,
                      bufferPercentage: roundToHalfDraft(form.bufferPercentage),
                    })
                  }
                />
              </label>
            </div>
            <p className="mt-4 rounded-control bg-brand-soft p-3 text-sm font-bold">
              Commitment capacity:{" "}
              {commitmentPreview(form.dailyCapacity, form.bufferPercentage)}{" "}
              hours/day
            </p>
            <div className="form-actions">
              <Button
                type="button"
                disabled={saving}
                onClick={() => setEditing(undefined)}
              >
                Cancel
              </Button>
              <Button variant="primary" loading={saving}>
                {saving ? "Saving…" : "Save"}
              </Button>
            </div>
          </form>
        </Dialog>
      )}
      {deletingMember ? (
        <Dialog
          titleID="delete-member-title"
          kind="alertdialog"
          closeOnBackdrop={false}
          onClose={() => {
            if (!saving) setDeletingMember(undefined);
          }}
        >
          <h2 id="delete-member-title" className="text-dialog-title font-black">
            Delete {deletingMember.name}?
          </h2>
          <p className="mt-3 text-muted">
            This member and their capacity overrides will be removed from active
            planning. Historical records will be retained. Members with active
            tasks cannot be deleted.
          </p>
          {error ? (
            <Alert tone="danger" className="mt-4">
              {error}
            </Alert>
          ) : null}
          <div className="form-actions">
            <Button
              data-autofocus
              disabled={saving}
              onClick={() => setDeletingMember(undefined)}
            >
              Cancel
            </Button>
            <Button
              variant="danger-solid"
              loading={saving}
              onClick={() => void remove(deletingMember)}
            >
              {saving ? "Deleting…" : "Delete member"}
            </Button>
          </div>
        </Dialog>
      ) : null}
    </>
  );
}

function findRoleByName(
  roles: MemberRoleOption[],
  name: string,
): MemberRoleOption | undefined {
  const normalized = name.trim().toLocaleLowerCase();
  return roles.find((role) => role.name.toLocaleLowerCase() === normalized);
}

function capitalizeWords(value: string): string {
  return value.replace(
    /(^|[\s-])(\p{L})/gu,
    (_, boundary: string, letter: string) =>
      `${boundary}${letter.toLocaleUpperCase()}`,
  );
}

function filterRoles(
  roles: MemberRoleOption[],
  query: string,
): MemberRoleOption[] {
  const normalized = query.trim().toLocaleLowerCase();
  if (!normalized) return roles;
  return roles.filter((role) =>
    role.name.toLocaleLowerCase().startsWith(normalized),
  );
}

function commitmentPreview(dailyDraft: string, bufferDraft: string): string {
  const daily = parseDecimalDraft(dailyDraft);
  const buffer = parseDecimalDraft(bufferDraft);
  if (
    daily === undefined ||
    buffer === undefined ||
    !Number.isFinite(daily) ||
    !Number.isFinite(buffer)
  ) {
    return "—";
  }
  return String(calculateCommitmentCapacity(daily, buffer));
}
