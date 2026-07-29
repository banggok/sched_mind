import { useEffect, useRef, useState } from "react";
import { Alert } from "../../../shared/presentation/Alert";
import { Button } from "../../../shared/presentation/Button";
import { ListSkeleton } from "../../../shared/presentation/ListSkeleton";
import type { DependenciesGateway } from "../application/dependenciesGateway";
import type {
  DependencyDetail,
  DependencyDirection,
  DependencyTask,
} from "../domain/dependency";

const pageSize = 5;

export function TaskDependencies({
  taskId,
  gateway,
  readOnly,
}: {
  taskId: string;
  gateway: DependenciesGateway;
  readOnly: boolean;
}) {
  const [detail, setDetail] = useState<DependencyDetail>();
  const [loading, setLoading] = useState(true);
  const initialLoading = loading && !detail;
  const refreshing = loading && !!detail;
  const [error, setError] = useState("");
  const [version, setVersion] = useState(0);
  const [direction, setDirection] = useState<DependencyDirection>();
  const [mutationBusy, setMutationBusy] = useState(false);

  // React state disables the UI after render; the ref closes the same-tick event race.
  const mutationLock = useRef(false);
  const [success, setSuccess] = useState("");

  useEffect(() => {
    const controller = new AbortController();

    setLoading(true);
    setError("");

    void gateway
      .list(taskId, controller.signal)
      .then(setDetail)
      .catch((reason: unknown) => {
        if (!(reason instanceof DOMException && reason.name === "AbortError")) {
          setError("Dependencies could not be loaded.");
        }
      })
      .finally(() => {
        if (!controller.signal.aborted) {
          setLoading(false);
        }
      });

    return () => controller.abort();
  }, [gateway, taskId, version]);

  const changed = (message: string) => {
    setSuccess(message);
    gateway.invalidateTask(taskId);
    setDirection(undefined);
    setVersion((value) => value + 1);
  };

  const remove = async (id: string) => {
    if (mutationLock.current) {
      return;
    }

    mutationLock.current = true;
    setMutationBusy(true);
    setError("");

    try {
      await gateway.remove(id);
      changed("Dependency removed.");
    } catch (reason: unknown) {
      setError(
        reason instanceof Error
          ? reason.message
          : "Dependency could not be removed.",
      );
    } finally {
      mutationLock.current = false;
      setMutationBusy(false);
    }
  };

  return (
    <section
      className="border-t border-border-subtle pt-5"
      aria-labelledby="dependencies-title"
    >
      <h4 id="dependencies-title" className="font-extrabold">
        Dependencies
      </h4>

      {refreshing ? (
        <p className="mt-1 text-sm text-muted" role="status">
          Refreshing…
        </p>
      ) : null}

      {initialLoading ? (
        <ListSkeleton label="Loading dependencies" rows={2} />
      ) : error && !detail ? (
        <Alert tone="danger">
          {error}{" "}
          <Button
            type="button"
            compact
            className="ml-2"
            onClick={() => setVersion((value) => value + 1)}
          >
            Retry
          </Button>
        </Alert>
      ) : detail ? (
        <div className="mt-4 grid gap-5">
          <RelationSection
            title="Blocked by"
            values={detail.blockedBy}
            readOnly={readOnly}
            busy={mutationBusy}
            onDelete={(id) => void remove(id)}
            onAdd={() => setDirection("blockedBy")}
          />

          <RelationSection
            title="Blocks"
            values={detail.blocks}
            readOnly={readOnly}
            busy={mutationBusy}
            onDelete={(id) => void remove(id)}
            onAdd={() => setDirection("blocks")}
          />

          {error ? <Alert tone="danger">{error}</Alert> : null}
          {success ? <Alert tone="success">{success}</Alert> : null}

          {direction ? (
            <CandidatePicker
              taskId={taskId}
              direction={direction}
              gateway={gateway}
              onCancel={() => setDirection(undefined)}
              onCreated={() => changed("Dependency added.")}
            />
          ) : null}
        </div>
      ) : null}
    </section>
  );
}

function formatExpectedStart(value?: string): string {
  if (!value) {
    return "Not scheduled";
  }

  return new Intl.DateTimeFormat("en-GB", {
    day: "2-digit",
    month: "short",
    year: "numeric",
    timeZone: "UTC",
  }).format(new Date(value));
}

function UnlinkIcon() {
  return (
    <svg
      aria-hidden="true"
      className="size-5"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="M9 17H7a5 5 0 0 1 0-10h3" />
      <path d="M15 7h2a5 5 0 0 1 0 10h-3" />
      <line x1="8" y1="12" x2="16" y2="12" />
      <line x1="4" y1="4" x2="20" y2="20" />
    </svg>
  );
}

function RelationSection({
  title,
  values,
  readOnly,
  busy,
  onDelete,
  onAdd,
}: {
  title: string;
  values: DependencyDetail["blocks"];
  readOnly: boolean;
  busy: boolean;
  onDelete(id: string): void;
  onAdd(): void;
}) {
  return (
    <div>
      <div className="flex items-center justify-between gap-3">
        <h5 className="font-bold">{title}</h5>

        {!readOnly ? (
          <Button type="button" compact disabled={busy} onClick={onAdd}>
            Add
          </Button>
        ) : null}
      </div>

      {values.length === 0 ? (
        <p className="mt-2 text-sm text-muted">No dependencies.</p>
      ) : (
        <ul className="mt-2 space-y-2">
          {values.map(({ id, task }) => {
            const historical = title === "Blocks" && task.completed;

            return (
              <li
                key={id}
                className="rounded-control border border-border-subtle p-3"
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <strong className="block break-words">{task.name}</strong>

                    <p className="mt-1 break-words text-sm text-muted">
                      {task.projectName}
                      {task.completed ? " · Completed" : ""}
                      {title === "Blocks"
                        ? ` · ${formatExpectedStart(task.expectedStart)}`
                        : ""}
                    </p>
                  </div>

                  {!readOnly && !historical ? (
                    <button
                      type="button"
                      className="grid size-9 shrink-0 place-items-center rounded-control text-danger transition-colors hover:bg-danger-soft disabled:opacity-50"
                      aria-label={`Remove dependency for ${task.name}`}
                      title="Remove dependency"
                      disabled={busy}
                      onClick={() => onDelete(id)}
                    >
                      <UnlinkIcon />
                    </button>
                  ) : historical ? (
                    <span className="shrink-0 text-sm text-muted">
                      Historical
                    </span>
                  ) : null}
                </div>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}

function CandidatePicker({
  taskId,
  direction,
  gateway,
  onCancel,
  onCreated,
}: {
  taskId: string;
  direction: DependencyDirection;
  gateway: DependenciesGateway;
  onCancel(): void;
  onCreated(): void;
}) {
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);
  const [items, setItems] = useState<DependencyTask[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);

  // React state disables the UI after render; the ref closes the same-tick event race.
  const mutationLock = useRef(false);
  const [error, setError] = useState("");
  const [version, setVersion] = useState(0);

  useEffect(() => {
    const controller = new AbortController();

    const timer = window.setTimeout(() => {
      setLoading(true);
      setError("");

      void gateway
        .candidates(
          taskId,
          direction,
          search,
          page,
          pageSize,
          controller.signal,
        )
        .then((result) => {
          setItems(result.items);
          setTotal(result.totalItems);
        })
        .catch((reason: unknown) => {
          if (!(
            reason instanceof DOMException && reason.name === "AbortError"
          )) {
            setError("Tasks could not be loaded.");
          }
        })
        .finally(() => {
          if (!controller.signal.aborted) {
            setLoading(false);
          }
        });
    }, 250);

    return () => {
      window.clearTimeout(timer);
      controller.abort();
    };
  }, [gateway, taskId, direction, search, page, version]);

  const add = async (candidateId: string) => {
    if (mutationLock.current) {
      return;
    }

    mutationLock.current = true;
    setBusy(true);
    setError("");

    try {
      await gateway.create(
        direction === "blockedBy" ? candidateId : taskId,
        direction === "blockedBy" ? taskId : candidateId,
      );

      onCreated();
    } catch (reason: unknown) {
      setError(
        reason instanceof Error
          ? reason.message
          : "Dependency could not be added.",
      );
    } finally {
      mutationLock.current = false;
      setBusy(false);
    }
  };

  const pages = Math.max(1, Math.ceil(total / pageSize));

  return (
    <div
      className="rounded-control border border-border-subtle p-4"
      aria-label={`Add ${direction === "blockedBy" ? "Blocked by" : "Blocks"}`}
    >
      <label
        className="block text-label font-bold"
        htmlFor={`dependency-search-${direction}`}
      >
        Search tasks
      </label>

      <input
        id={`dependency-search-${direction}`}
        className="ui-input mt-2"
        value={search}
        autoFocus
        onChange={(event) => {
          setSearch(event.target.value);
          setPage(1);
        }}
        onKeyDown={(event) => {
          if (event.key === "Enter") {
            event.preventDefault();

            if (!loading && !busy && items[0]) {
              void add(items[0].id);
            }
          }
        }}
      />

      {loading ? (
        <ListSkeleton label="Loading tasks" rows={3} />
      ) : error ? (
        <Alert tone="danger">
          {error}{" "}
          <Button
            type="button"
            compact
            className="ml-2"
            onClick={() => setVersion((value) => value + 1)}
          >
            Retry
          </Button>
        </Alert>
      ) : items.length === 0 ? (
        <p className="mt-3 text-sm text-muted">No matching tasks.</p>
      ) : (
        <ul className="mt-3 space-y-2">
          {items.map((task) => (
            <li key={task.id} className="flex items-center gap-3">
              <span className="mr-auto">
                <strong>{task.name}</strong>

                <span className="block text-sm text-muted">
                  {task.hierarchyPath}
                  {task.completed ? " · Completed" : ""}
                </span>
              </span>

              <Button
                type="button"
                compact
                variant="primary"
                loading={busy}
                disabled={busy}
                onClick={() => void add(task.id)}
              >
                Select
              </Button>
            </li>
          ))}
        </ul>
      )}

      <div className="mt-4 flex items-center justify-between gap-2">
        <Button type="button" compact onClick={onCancel}>
          Cancel
        </Button>

        <div className="flex items-center gap-2">
          <Button
            type="button"
            compact
            disabled={page <= 1 || loading}
            onClick={() => setPage((value) => value - 1)}
          >
            Previous
          </Button>

          <span className="text-sm">
            Page {page} of {pages}
          </span>

          <Button
            type="button"
            compact
            disabled={page >= pages || loading}
            onClick={() => setPage((value) => value + 1)}
          >
            Next
          </Button>
        </div>
      </div>
    </div>
  );
}
