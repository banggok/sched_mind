import { useMemo, useState } from "react";

import { Breadcrumb } from "../../../app/Breadcrumb";
import { PageContent } from "../../../app/PageContent";
import { SearchField } from "../../../shared/presentation/SearchField";
import { PaginationControls } from "../../../shared/presentation/PaginationControls";
import { ListSkeleton } from "../../../shared/presentation/ListSkeleton";
import { useDebouncedValue } from "../../../shared/presentation/useDebouncedValue";
import { Button } from "../../../shared/presentation/Button";
import { EmptyState } from "../../../shared/presentation/EmptyState";
import { ListSurface } from "../../../shared/presentation/ListSurface";
import type { RolesGateway } from "../application/rolesGateway";
import type { Role } from "../domain/role";
import { DeleteRoleDialog } from "./DeleteRoleDialog";
import { RoleFormDialog } from "./RoleFormDialog";
import { useRoleManagement } from "./useRoleManagement";

export function RolesDashboardPage({ gateway }: { gateway: RolesGateway }) {
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);
  const debouncedSearch = useDebouncedValue(search.trim());
  const searchPending = search.trim() !== debouncedSearch;
  const query = useMemo(
    () => ({ search: debouncedSearch, page, pageSize: 5 }),
    [debouncedSearch, page],
  );
  const management = useRoleManagement(gateway, query);

  return (
    <>
      <PageContent>
        <section className="min-w-0">
          <div className="page-header">
            <Breadcrumb activePage="roles" />
            <Button
              variant="primary"
              className="shrink-0"
              onClick={management.openCreate}
            >
              + Add Role
            </Button>
          </div>

          <ListSurface
            title="Roles"
            controls={
              <SearchField
                label="Search roles"
                value={search}
                onChange={(value) => {
                  setSearch(value);
                  setPage(1);
                }}
              />
            }
          >
            {management.loading || searchPending ? (
              <ListSkeleton label="Loading roles" />
            ) : management.pageError ? (
              <div className="p-10 text-center">
                <p className="font-bold text-danger" role="alert">
                  {management.pageError}
                </p>
                <Button
                  variant="quiet"
                  className="mt-4"
                  onClick={() => void management.retry()}
                >
                  Try again
                </Button>
              </div>
            ) : management.roles.length === 0 && debouncedSearch === "" ? (
              <EmptyState
                title="No roles yet"
                description="Add the first team role to start organizing your team."
                icon="◫"
                action={
                  <Button variant="quiet" onClick={management.openCreate}>
                    Add Role
                  </Button>
                }
              />
            ) : management.roles.length === 0 ? (
              <EmptyState
                title="No matching roles"
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
            {!management.loading &&
            !searchPending &&
            !management.pageError &&
            management.total > 0 ? (
              <PaginationControls
                page={page}
                pageSize={5}
                total={management.total}
                onPageChange={setPage}
              />
            ) : null}
          </ListSurface>
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
          className="layer-notification fixed right-5 bottom-5 flex max-w-sm items-center gap-4 rounded-panel bg-overlay px-5 py-4 text-sm font-bold text-on-brand shadow-floating"
          role="status"
        >
          <span>{management.notification}</span>
          <button
            type="button"
            className="grid size-8 shrink-0 place-items-center rounded-action text-lg hover:bg-surface/10 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white"
            aria-label="Dismiss notification"
            onClick={() => management.setNotification("")}
          >
            ×
          </button>
        </div>
      ) : null}
    </>
  );
}

function RoleRow({
  role,
  onEdit,
  onDelete,
}: {
  role: Role;
  onEdit(): void;
  onDelete(): void;
}) {
  return (
    <li className="flex flex-col items-stretch justify-between gap-4 px-6 py-5 sm:flex-row sm:items-center">
      <div className="flex min-w-0 items-center gap-4">
        <span className="grid size-11 shrink-0 place-items-center rounded-control bg-brand-soft font-black text-brand">
          {role.name.slice(0, 1).toUpperCase()}
        </span>
        <div className="min-w-0">
          <h3 className="truncate font-extrabold">{role.name}</h3>
          <p className="mt-1 text-xs text-subtle">
            Updated {formatRoleDate(role.updatedAt)}
          </p>
        </div>
      </div>
      <div className="flex gap-2 self-end sm:self-auto">
        <Button compact aria-label={`Edit ${role.name}`} onClick={onEdit}>
          Edit
        </Button>
        <Button
          compact
          variant="danger"
          aria-label={`Delete ${role.name}`}
          onClick={onDelete}
        >
          Delete
        </Button>
      </div>
    </li>
  );
}

const roleDateFormatter = new Intl.DateTimeFormat("en-GB", {
  day: "numeric",
  month: "short",
  year: "numeric",
});

function formatRoleDate(date: Date): string {
  return roleDateFormatter.format(date);
}
