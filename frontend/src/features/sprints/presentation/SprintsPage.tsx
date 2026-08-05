import { useCallback, useEffect, useRef, useState } from "react";
import { Breadcrumb } from "../../../app/Breadcrumb";
import { PageContent } from "../../../app/PageContent";
import { Alert } from "../../../shared/presentation/Alert";
import { Button } from "../../../shared/presentation/Button";
import { Dialog } from "../../../shared/presentation/Dialog";
import { EmptyState } from "../../../shared/presentation/EmptyState";
import { ListSkeleton } from "../../../shared/presentation/ListSkeleton";
import { ListSurface } from "../../../shared/presentation/ListSurface";
import { PaginationControls } from "../../../shared/presentation/PaginationControls";
import { formatDateOnly } from "../../../shared/presentation/formatDateOnly";
import type { TeamMembersGateway } from "../../team-members/application/teamMembersGateway";
import type { WBSGateway } from "../../wbs/application/wbsGateway";
import type { SprintsGateway } from "../application/sprintsGateway";
import type { Sprint, SprintDetail } from "../domain/sprint";
import { SprintFormDialog } from "./SprintFormDialog";
import { SprintTaskReview } from "./SprintTaskReview";

export function SprintsPage({
  gateway,
  membersGateway,
  wbsGateway,
  loadPublicHolidayDates,
}: {
  gateway: SprintsGateway;
  membersGateway: TeamMembersGateway;
  wbsGateway: Pick<WBSGateway, "tree">;
  loadPublicHolidayDates(startDate: string, endDate: string): Promise<string[]>;
}) {
  const [items, setItems] = useState<Sprint[]>([]);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [detail, setDetail] = useState<SprintDetail>();
  const [editing, setEditing] = useState(false);
  const [detailError, setDetailError] = useState("");
  const [deleting, setDeleting] = useState<Sprint>();
  const [pendingID, setPendingID] = useState("");
  const [creating, setCreating] = useState(false);
  const requestID = useRef(0);

  const load = useCallback(
    async (targetPage: number) => {
      const current = ++requestID.current;
      setLoading(true);
      setError("");
      try {
        const result = await gateway.list({
          search: "",
          page: targetPage,
          pageSize: 10,
        });
        if (current !== requestID.current) return;
        setItems(result.items);
        setTotal(result.total);
      } catch {
        if (current === requestID.current)
          setError("Sprints could not be loaded. Please try again.");
      } finally {
        if (current === requestID.current) setLoading(false);
      }
    },
    [gateway],
  );

  useEffect(() => {
    void load(page);
  }, [load, page]);

  async function openDetail(sprint: Sprint): Promise<boolean> {
    const current = ++requestID.current;
    setDetailError("");
    setPendingID(sprint.id);
    try {
      const value = await gateway.detail(sprint.id);
      if (current === requestID.current) {
        setDetail(value);
        return true;
      }
    } catch {
      if (current === requestID.current)
        setDetailError("This Sprint could not be opened. Please try again.");
    } finally {
      if (current === requestID.current) setPendingID("");
    }
    return false;
  }

  async function refreshDetail(id: string) {
    const sprint = items.find((item) => item.id === id);
    if (sprint) await openDetail(sprint);
    await load(page);
  }

  async function start(sprint: Sprint) {
    if (pendingID) return;
    setPendingID(sprint.id);
    setError("");
    try {
      await gateway.start(sprint.id, sprint.version);
      await load(page);
    } catch {
      setError(
        "The Sprint could not be started. Please refresh and try again.",
      );
    } finally {
      setPendingID("");
    }
  }

  async function confirmDelete() {
    if (!deleting || pendingID) return;
    setPendingID(deleting.id);
    try {
      await gateway.delete(deleting.id, deleting.version);
      setDeleting(undefined);
      await load(page);
    } catch {
      setError(
        "The Sprint could not be deleted. Please refresh and try again.",
      );
    } finally {
      setPendingID("");
    }
  }

  return (
    <PageContent>
      <section className="min-w-0">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div className="min-w-0">
            <Breadcrumb activePage="sprints" />
            <p className="mt-2 max-w-2xl text-sm leading-6 text-muted">
              Plan focused delivery periods, then review Member capacity and
              selected Tasks.
            </p>
          </div>
          <Button
            variant="primary"
            className="shrink-0 self-start"
            onClick={() => setCreating(true)}
          >
            + Create New Sprint
          </Button>
        </div>
        <ListSurface
          title="Sprints"
          controls={
            !loading && total > 0 ? (
              <p className="text-sm font-bold text-muted">
                {total} {total === 1 ? "Sprint" : "Sprints"}
              </p>
            ) : undefined
          }
        >
          {loading && items.length > 0 ? (
            <p className="px-6 pt-4 text-sm font-bold text-brand" role="status">
              Refreshing Sprints…
            </p>
          ) : null}
          {error && items.length > 0 ? (
            <Alert tone="danger" className="m-4">
              {error}{" "}
              <button
                type="button"
                className="underline"
                onClick={() => void load(page)}
              >
                Retry
              </button>
            </Alert>
          ) : null}
          {loading && items.length === 0 ? (
            <ListSkeleton label="Loading Sprints" rows={5} />
          ) : error && items.length === 0 ? (
            <div className="p-10 text-center">
              <p className="font-bold text-danger" role="alert">
                {error}
              </p>
              <Button
                variant="quiet"
                className="mt-4"
                onClick={() => void load(page)}
              >
                Retry
              </Button>
            </div>
          ) : items.length === 0 && !error ? (
            <EmptyState
              title="No Sprints yet"
              description="Create a Sprint to review selected Members and scheduled Tasks."
              icon={
                <svg
                  className="size-7"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="1.8"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                >
                  <rect x="3" y="5" width="18" height="16" rx="2" />
                  <path d="M8 3v4M16 3v4M3 10h18M8 14h3M8 17h6" />
                </svg>
              }
              action={
                <Button variant="quiet" onClick={() => setCreating(true)}>
                  Create New Sprint
                </Button>
              }
            />
          ) : (
            <div
              className="overflow-x-auto focus:outline-none"
              role="region"
              aria-label="Sprint list"
              tabIndex={0}
            >
              <table className="w-full min-w-max text-left text-sm">
                <caption className="sr-only">
                  Sprints with date boundaries, status, and available actions
                </caption>
                <thead className="bg-surface-muted text-xs font-extrabold uppercase tracking-wide text-muted">
                  <tr>
                    <th className="whitespace-nowrap px-6 py-4" scope="col">
                      Sprint Name
                    </th>
                    <th className="whitespace-nowrap px-6 py-4" scope="col">
                      Start Date
                    </th>
                    <th className="whitespace-nowrap px-6 py-4" scope="col">
                      End Date
                    </th>
                    <th className="whitespace-nowrap px-6 py-4" scope="col">
                      Status
                    </th>
                    <th
                      className="whitespace-nowrap px-6 py-4 text-right"
                      scope="col"
                    >
                      Actions
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border-subtle">
                  {items.map((sprint) => (
                    <tr
                      key={sprint.id}
                      className="hover:bg-surface-muted focus-within:bg-surface-muted"
                    >
                      <th
                        className="whitespace-nowrap px-6 py-5 text-left align-middle"
                        scope="row"
                      >
                        <button
                          type="button"
                          className="font-extrabold text-brand-strong underline-offset-4 hover:underline disabled:text-disabled"
                          disabled={pendingID === sprint.id}
                          onClick={() => void openDetail(sprint)}
                        >
                          {sprint.name}
                        </button>
                      </th>
                      <td className="whitespace-nowrap px-6 py-5 align-middle text-text-secondary">
                        {formatDateOnly(sprint.startDate)}
                      </td>
                      <td className="whitespace-nowrap px-6 py-5 align-middle text-text-secondary">
                        {formatDateOnly(sprint.endDate)}
                      </td>
                      <td className="whitespace-nowrap px-6 py-5 align-middle">
                        <span
                          aria-label={`Status: ${sprint.status === "planned" ? "Planned" : "Started"}`}
                          className={`inline-flex items-center gap-2 rounded-action border px-3 py-1 text-xs font-extrabold ${
                            sprint.status === "planned"
                              ? "border-border-strong bg-surface-muted text-text-secondary"
                              : "border-success bg-success-soft text-success"
                          }`}
                        >
                          <span
                            aria-hidden="true"
                            className={`size-2 rounded-full ${
                              sprint.status === "planned"
                                ? "bg-subtle"
                                : "bg-success"
                            }`}
                          />
                          {sprint.status === "planned" ? "Planned" : "Started"}
                        </span>
                      </td>
                      <td className="whitespace-nowrap px-6 py-5 align-middle">
                        <div
                          className="flex flex-wrap justify-end gap-2"
                          role="group"
                          aria-label={`${sprint.name} actions`}
                        >
                          {sprint.status === "planned" ? (
                            <Button
                              compact
                              aria-label={`Start ${sprint.name}`}
                              disabled={Boolean(pendingID)}
                              onClick={() => void start(sprint)}
                            >
                              Start
                            </Button>
                          ) : null}
                          <Button
                            compact
                            aria-label={`Edit ${sprint.name}`}
                            disabled={Boolean(pendingID)}
                            onClick={() =>
                              void openDetail(sprint).then((opened) => {
                                if (opened) setEditing(true);
                              })
                            }
                          >
                            Edit
                          </Button>
                          <Button
                            compact
                            variant="danger"
                            aria-label={`Delete ${sprint.name}`}
                            disabled={Boolean(pendingID)}
                            onClick={() => setDeleting(sprint)}
                          >
                            Delete
                          </Button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
          {!loading && !error && total > 0 ? (
            <PaginationControls
              page={page}
              pageSize={10}
              total={total}
              onPageChange={setPage}
            />
          ) : null}
        </ListSurface>
        {detail ? (
          <SprintTaskReview
            detail={detail}
            gateway={gateway}
            wbsGateway={wbsGateway}
            onEdit={() => setEditing(true)}
            onClose={() => setDetail(undefined)}
            onChanged={(id) => void refreshDetail(id)}
          />
        ) : null}
      </section>
      {detail && editing ? (
        <SprintFormDialog
          gateway={gateway}
          membersGateway={membersGateway}
          initialDetail={detail}
          loadPublicHolidayDates={loadPublicHolidayDates}
          onClose={() => setEditing(false)}
          onSaved={(saved) => {
            setEditing(false);
            void refreshDetail(saved.id);
          }}
        />
      ) : null}
      {detailError ? (
        <Dialog titleID="sprint-error-title" onClose={() => setDetailError("")}>
          <h2 id="sprint-error-title" className="text-xl font-black">
            Sprint unavailable
          </h2>
          <Alert tone="danger" className="mt-4">
            {detailError}
          </Alert>
          <div className="form-actions">
            <Button onClick={() => setDetailError("")}>Close</Button>
          </div>
        </Dialog>
      ) : null}
      {deleting ? (
        <Dialog
          titleID="delete-sprint-title"
          kind="alertdialog"
          closeOnBackdrop={false}
          onClose={() => setDeleting(undefined)}
        >
          <h2 id="delete-sprint-title" className="text-xl font-black">
            Delete {deleting.name}?
          </h2>
          <p className="mt-3">
            This removes only the Sprint and its Member and Task associations.
            Projects and Tasks are not deleted.
          </p>
          <div className="form-actions">
            <Button
              disabled={Boolean(pendingID)}
              onClick={() => setDeleting(undefined)}
            >
              Cancel
            </Button>
            <Button
              variant="danger-solid"
              loading={Boolean(pendingID)}
              onClick={() => void confirmDelete()}
            >
              {pendingID ? "Deleting…" : "Delete Sprint"}
            </Button>
          </div>
        </Dialog>
      ) : null}
      {creating ? (
        <SprintFormDialog
          gateway={gateway}
          membersGateway={membersGateway}
          loadPublicHolidayDates={loadPublicHolidayDates}
          onClose={() => setCreating(false)}
          onSaved={(saved) => {
            setCreating(false);
            setPage(1);
            void load(1);
            void gateway.detail(saved.id).then(setDetail);
          }}
        />
      ) : null}
    </PageContent>
  );
}
