import { useCallback, useEffect, useMemo, useState } from "react";

import type {
  RoleAuditGateway,
  RoleMemberUsage,
} from "../application/rolesGateway";
import type { Role } from "../domain/role";
import { Button } from "../../../shared/presentation/Button";
import { Dialog } from "../../../shared/presentation/Dialog";
import { PaginationControls } from "../../../shared/presentation/PaginationControls";
import { SearchField } from "../../../shared/presentation/SearchField";
import { useDebouncedValue } from "../../../shared/presentation/useDebouncedValue";

const PAGE_SIZE = 10;

export function RoleMembersDialog({
  role,
  gateway,
  onClose,
}: {
  role: Role;
  gateway: RoleAuditGateway;
  onClose(): void;
}) {
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);
  const [members, setMembers] = useState<RoleMemberUsage[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [retryVersion, setRetryVersion] = useState(0);
  const debouncedSearch = useDebouncedValue(search.trim());
  const searchPending = search.trim() !== debouncedSearch;
  const query = useMemo(
    () => ({ search: debouncedSearch, page, pageSize: PAGE_SIZE }),
    [debouncedSearch, page],
  );

  useEffect(() => {
    const controller = new AbortController();
    setLoading(true);
    void gateway
      .listMembers(role.id, query, controller.signal)
      .then((result) => {
        if (controller.signal.aborted) return;
        setMembers(result.items);
        setTotal(result.total);
        setError("");
      })
      .catch((cause: unknown) => {
        if (controller.signal.aborted) return;
        if (cause instanceof DOMException && cause.name === "AbortError")
          return;
        setError(roleMemberError(cause));
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, [gateway, query, retryVersion, role.id]);

  const requestClose = useCallback(() => onClose(), [onClose]);
  const busy = loading || searchPending;

  return (
    <Dialog
      titleID="role-members-title"
      descriptionID="role-members-description"
      onClose={requestClose}
      wide
    >
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0">
          <p className="text-xs font-extrabold tracking-widest text-brand uppercase">
            Role audit
          </p>
          <h2
            id="role-members-title"
            className="mt-2 truncate text-dialog-title font-black tracking-tight"
          >
            Members using {role.name}
          </h2>
          <p
            id="role-members-description"
            className="mt-2 max-w-2xl text-sm leading-6 text-muted"
          >
            Active team members assigned to this role. Task usage is checked
            separately when the role is deleted.
          </p>
        </div>
        <Button
          type="button"
          variant="quiet"
          compact
          className="size-9 shrink-0 p-0 text-xl text-muted"
          aria-label="Close role member audit"
          onClick={requestClose}
        >
          ×
        </Button>
      </div>

      <div className="mt-6">
        <SearchField
          label="Search members"
          value={search}
          onChange={(value) => {
            setSearch(value);
            setPage(1);
          }}
        />
      </div>

      <section
        className="mt-4 overflow-hidden rounded-panel border border-border-subtle"
        aria-label={`Members using ${role.name}`}
      >
        {busy ? (
          <p className="p-6 text-sm text-muted" role="status">
            Loading members…
          </p>
        ) : error ? (
          <div className="p-6">
            <p className="font-semibold text-danger" role="alert">
              {error}
            </p>
            <Button
              type="button"
              variant="quiet"
              className="mt-3"
              onClick={() => setRetryVersion((value) => value + 1)}
            >
              Try again
            </Button>
          </div>
        ) : members.length === 0 ? (
          <div className="p-6">
            <p className="font-bold">
              {debouncedSearch === ""
                ? "No active members use this role"
                : "No matching members"}
            </p>
            <p className="mt-1 text-sm text-muted">
              {debouncedSearch === ""
                ? "No active team member uses this role. Tasks may still reference it."
                : "Try another member name."}
            </p>
          </div>
        ) : (
          <ul className="divide-y divide-border-subtle">
            {members.map((member) => (
              <li key={member.id} className="flex items-center gap-3 px-5 py-4">
                <span className="grid size-9 shrink-0 place-items-center rounded-control bg-brand-soft text-sm font-black text-brand">
                  {member.name.slice(0, 1).toUpperCase()}
                </span>
                <span className="min-w-0 truncate font-bold">
                  {member.name}
                </span>
              </li>
            ))}
          </ul>
        )}
        {!busy && !error && total > 0 ? (
          <PaginationControls
            page={page}
            pageSize={PAGE_SIZE}
            total={total}
            onPageChange={setPage}
          />
        ) : null}
      </section>

      <div className="form-actions">
        <Button type="button" onClick={requestClose}>
          Close
        </Button>
      </div>
    </Dialog>
  );
}

function roleMemberError(error: unknown): string {
  const code =
    typeof error === "object" &&
    error !== null &&
    "code" in error &&
    typeof error.code === "string"
      ? error.code
      : "";
  if (code === "ROLE_NOT_FOUND") {
    return "This role no longer exists. Close this dialog and refresh the list.";
  }
  return "Unable to load members for this role. Try again.";
}
