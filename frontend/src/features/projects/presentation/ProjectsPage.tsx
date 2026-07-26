import { useEffect, useMemo, useState, type FormEvent } from "react";
import { Breadcrumb } from "../../../app/Breadcrumb";
import { PageContent } from "../../../app/PageContent";
import { Button } from "../../../shared/presentation/Button";
import { Dialog } from "../../../shared/presentation/Dialog";
import { EmptyState } from "../../../shared/presentation/EmptyState";
import { FormField } from "../../../shared/presentation/FormField";
import { ListSkeleton } from "../../../shared/presentation/ListSkeleton";
import { ListSurface } from "../../../shared/presentation/ListSurface";
import { PaginationControls } from "../../../shared/presentation/PaginationControls";
import { SearchField } from "../../../shared/presentation/SearchField";
import { Toast } from "../../../shared/presentation/Toast";
import { useDebouncedValue } from "../../../shared/presentation/useDebouncedValue";
import {
  changeProjectStatus,
  createProject,
  deleteProject,
  listProjects,
  moveProjectPriority,
  updateProject,
} from "../application/projectManagement";
import {
  ProjectOperationError,
  type ProjectsGateway,
} from "../application/projectsGateway";
import {
  ProjectNameError,
  type PriorityDirection,
  type Project,
  type ProjectStatus,
} from "../domain/project";

type FormState = { mode: "create" | "edit"; project?: Project };
type CommandState = {
  kind: "lock" | "close" | "reopen" | "delete";
  project: Project;
};

export function ProjectsPage({ gateway }: { gateway: ProjectsGateway }) {
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);
  const [items, setItems] = useState<Project[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [reload, setReload] = useState(0);
  const [form, setForm] = useState<FormState>();
  const [command, setCommand] = useState<CommandState>();
  const [name, setName] = useState("");
  const [fieldError, setFieldError] = useState("");
  const [operationError, setOperationError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [notification, setNotification] = useState("");
  const debouncedSearch = useDebouncedValue(search.trim());
  const searchPending = search.trim() !== debouncedSearch;
  const query = useMemo(
    () => ({ search: debouncedSearch, page, pageSize: 5 }),
    [debouncedSearch, page],
  );

  useEffect(() => {
    const controller = new AbortController();
    setLoading(true);
    setError("");
    void listProjects(gateway, query, controller.signal)
      .then((result) => {
        setItems(result.items);
        setTotal(result.total);
        const lastPage = Math.max(1, Math.ceil(result.total / result.pageSize));
        if (page > lastPage) setPage(lastPage);
      })
      .catch((reason: unknown) => {
        if (!(reason instanceof DOMException && reason.name === "AbortError"))
          setError("Projects could not be loaded. Try again.");
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, [gateway, page, query, reload]);

  function refresh(message: string) {
    setNotification(message);
    setReload((value) => value + 1);
  }
  function openCreate() {
    setForm({ mode: "create" });
    setName("");
    setFieldError("");
    setOperationError("");
  }
  function openEdit(project: Project) {
    setForm({ mode: "edit", project });
    setName(project.name);
    setFieldError("");
    setOperationError("");
  }
  async function submitForm() {
    if (!form || submitting) return;
    setSubmitting(true);
    setFieldError("");
    setOperationError("");
    try {
      if (form.mode === "create") await createProject(gateway, name);
      else if (form.project)
        await updateProject(gateway, form.project.id, name);
      setForm(undefined);
      refresh(form.mode === "create" ? "Project added." : "Project updated.");
    } catch (reason: unknown) {
      if (
        reason instanceof ProjectNameError ||
        (reason instanceof ProjectOperationError && reason.field === "name")
      )
        setFieldError(userMessage(reason));
      else setOperationError(userMessage(reason));
    } finally {
      setSubmitting(false);
    }
  }
  async function confirmCommand() {
    if (!command || submitting) return;
    setSubmitting(true);
    setOperationError("");
    try {
      if (command.kind === "delete")
        await deleteProject(gateway, command.project.id);
      else
        await changeProjectStatus(
          gateway,
          command.project.id,
          targetStatus(command.kind),
        );
      const message =
        command.kind === "delete"
          ? "Project deleted."
          : command.kind === "lock"
            ? "Project locked."
            : command.kind === "close"
              ? "Project closed."
              : "Project reopened.";
      setCommand(undefined);
      refresh(message);
    } catch (reason: unknown) {
      setOperationError(userMessage(reason));
    } finally {
      setSubmitting(false);
    }
  }
  async function move(project: Project, direction: PriorityDirection) {
    if (submitting) return;
    setSubmitting(true);
    setOperationError("");
    try {
      await moveProjectPriority(gateway, project.id, direction);
      refresh("Project priority updated.");
    } catch (reason: unknown) {
      setNotification(userMessage(reason));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <>
      <PageContent>
        <section className="min-w-0">
          <div className="page-header">
            <Breadcrumb activePage="projects" />
            <Button variant="primary" onClick={openCreate}>
              + Add Project
            </Button>
          </div>
          <ListSurface
            title="Projects"
            controls={
              <SearchField
                label="Search projects"
                value={search}
                onChange={(value) => {
                  setSearch(value);
                  setPage(1);
                }}
              />
            }
          >
            {(loading && items.length === 0) || searchPending ? (
              <ListSkeleton label="Loading projects" />
            ) : error ? (
              <div className="p-10 text-center">
                <p className="font-bold text-danger" role="alert">
                  {error}
                </p>
                <Button
                  variant="quiet"
                  className="mt-4"
                  onClick={() => setReload((value) => value + 1)}
                >
                  Try again
                </Button>
              </div>
            ) : items.length === 0 && !debouncedSearch ? (
              <EmptyState
                title="No projects yet"
                description="Add the first project to begin planning delivery."
                action={
                  <Button variant="quiet" onClick={openCreate}>
                    Add Project
                  </Button>
                }
              />
            ) : items.length === 0 ? (
              <EmptyState
                title="No matching projects"
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
                {items.map((project, index) => (
                  <ProjectRow
                    key={project.id}
                    project={project}
                    first={page === 1 && index === 0}
                    busy={submitting}
                    onEdit={() => openEdit(project)}
                    onCommand={(kind) => {
                      setOperationError("");
                      setCommand({ kind, project });
                    }}
                    onMove={(direction) => void move(project, direction)}
                  />
                ))}
              </ul>
            )}
            {!loading && !searchPending && !error && total > 0 ? (
              <PaginationControls
                page={page}
                pageSize={5}
                total={total}
                onPageChange={setPage}
              />
            ) : null}
          </ListSurface>
        </section>
      </PageContent>
      {form ? (
        <ProjectForm
          form={form}
          name={name}
          fieldError={fieldError}
          operationError={operationError}
          submitting={submitting}
          onName={setName}
          onClose={() => !submitting && setForm(undefined)}
          onSubmit={() => void submitForm()}
        />
      ) : null}
      {command ? (
        <CommandDialog
          command={command}
          error={operationError}
          submitting={submitting}
          onClose={() => !submitting && setCommand(undefined)}
          onConfirm={() => void confirmCommand()}
        />
      ) : null}
      <Toast message={notification} onDismiss={() => setNotification("")} />
    </>
  );
}

function ProjectRow({
  project,
  first,
  busy,
  onEdit,
  onCommand,
  onMove,
}: {
  project: Project;
  first: boolean;
  busy: boolean;
  onEdit(): void;
  onCommand(kind: CommandState["kind"]): void;
  onMove(direction: PriorityDirection): void;
}) {
  const active = project.status !== "closed";
  return (
    <li className="flex flex-col gap-4 px-6 py-5 lg:flex-row lg:items-center lg:justify-between">
      <div className="min-w-0">
        <div className="flex flex-wrap items-center gap-2">
          <h3 className="truncate font-extrabold">{project.name}</h3>
          <span className="rounded-action bg-surface-muted px-2 py-1 text-xs font-bold capitalize">
            {project.status}
          </span>
        </div>
        <p className="mt-1 text-xs text-subtle">
          Priority {project.priority} · {project.startDate ?? "No start date"} —{" "}
          {project.endDate ?? "No end date"}
        </p>
      </div>
      <div className="flex flex-wrap justify-end gap-2">
        {active ? (
          <>
            <Button
              compact
              aria-label={`Move ${project.name} up`}
              disabled={busy || first}
              onClick={() => onMove("up")}
            >
              ↑
            </Button>
            <Button
              compact
              aria-label={`Move ${project.name} down`}
              disabled={busy}
              onClick={() => onMove("down")}
            >
              ↓
            </Button>
            <Button compact disabled={busy} onClick={onEdit}>
              Edit
            </Button>
          </>
        ) : null}
        {project.status === "open" ? (
          <Button compact disabled={busy} onClick={() => onCommand("lock")}>
            Lock
          </Button>
        ) : null}
        {active ? (
          <Button compact disabled={busy} onClick={() => onCommand("close")}>
            Close
          </Button>
        ) : (
          <Button compact disabled={busy} onClick={() => onCommand("reopen")}>
            Reopen
          </Button>
        )}
        {active ? (
          <Button
            compact
            variant="danger"
            disabled={busy}
            onClick={() => onCommand("delete")}
          >
            Delete
          </Button>
        ) : null}
      </div>
    </li>
  );
}

function ProjectForm({
  form,
  name,
  fieldError,
  operationError,
  submitting,
  onName,
  onClose,
  onSubmit,
}: {
  form: FormState;
  name: string;
  fieldError: string;
  operationError: string;
  submitting: boolean;
  onName(value: string): void;
  onClose(): void;
  onSubmit(): void;
}) {
  function submit(event: FormEvent) {
    event.preventDefault();
    onSubmit();
  }
  return (
    <Dialog titleID="project-form-title" onClose={onClose}>
      <h2 id="project-form-title" className="text-dialog-title font-black">
        {form.mode === "create" ? "Add project" : `Edit ${form.project?.name}`}
      </h2>
      <form className="mt-7" noValidate onSubmit={submit}>
        <FormField
          id="project-name"
          name="name"
          label="Project name"
          required
          data-autofocus
          value={name}
          error={fieldError}
          help={
            form.mode === "create"
              ? "The project starts Open at the lowest priority."
              : "Up to 100 characters."
          }
          maxLength={101}
          onChange={(event) => onName(event.target.value)}
        />
        {operationError ? (
          <p className="mt-4 text-sm font-semibold text-danger" role="alert">
            {operationError}
          </p>
        ) : null}
        <div className="form-actions">
          <Button type="button" disabled={submitting} onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" variant="primary" loading={submitting}>
            {submitting ? "Saving…" : "Save"}
          </Button>
        </div>
      </form>
    </Dialog>
  );
}

function CommandDialog({
  command,
  error,
  submitting,
  onClose,
  onConfirm,
}: {
  command: CommandState;
  error: string;
  submitting: boolean;
  onClose(): void;
  onConfirm(): void;
}) {
  const content = commandCopy(command);
  return (
    <Dialog
      titleID="project-command-title"
      kind="alertdialog"
      closeOnBackdrop={false}
      onClose={onClose}
    >
      <h2 id="project-command-title" className="text-dialog-title font-black">
        {content.title}
      </h2>
      <p className="mt-3 leading-7 text-muted">{content.description}</p>
      {error ? (
        <p className="mt-4 text-sm font-semibold text-danger" role="alert">
          {error}
        </p>
      ) : null}
      <div className="form-actions">
        <Button disabled={submitting} onClick={onClose}>
          Cancel
        </Button>
        <Button
          data-autofocus
          variant={command.kind === "delete" ? "danger-solid" : "primary"}
          loading={submitting}
          onClick={onConfirm}
        >
          {submitting ? "Working…" : content.action}
        </Button>
      </div>
    </Dialog>
  );
}
function targetStatus(
  kind: Exclude<CommandState["kind"], "delete">,
): ProjectStatus {
  return kind === "lock" ? "locked" : kind === "close" ? "closed" : "open";
}
function commandCopy(command: CommandState) {
  if (command.kind === "lock")
    return {
      title: `Lock ${command.project.name}?`,
      description:
        "Current Execution and Commitment dates become protected baselines. Forecast remains dynamic.",
      action: "Lock project",
    };
  if (command.kind === "close")
    return {
      title: `Close ${command.project.name}?`,
      description:
        "Every executable task must have an Actual End. Closed projects become read-only and leave scheduling and Gantt.",
      action: "Close project",
    };
  if (command.kind === "reopen")
    return {
      title: `Reopen ${command.project.name}?`,
      description:
        "The project becomes active, editable, schedulable, and visible in Gantt again.",
      action: "Reopen project",
    };
  return {
    title: `Delete ${command.project.name}?`,
    description:
      "This permanently removes a childless project. Projects with planning children cannot be deleted.",
    action: "Delete project",
  };
}
function userMessage(reason: unknown): string {
  if (reason instanceof ProjectNameError) return reason.message;
  if (reason instanceof ProjectOperationError) {
    const messages: Record<string, string> = {
      PROJECT_NAME_ALREADY_EXISTS: "Project name already exists",
      PROJECT_CANNOT_CLOSE_WITHOUT_TASKS:
        "A project without tasks cannot be closed. Delete it instead.",
      PROJECT_CANNOT_LOCK_WITHOUT_TASKS:
        "Add at least one task before locking this project.",
      PROJECT_CANNOT_CLOSE_WITH_ACTIVE_TASKS:
        "Finish every task by setting Actual End before closing this project.",
      PROJECT_HAS_CHILDREN:
        "This project has planning children and cannot be deleted. Complete its work and close it instead.",
      PROJECT_PRIORITY_MOVE_NOT_ALLOWED:
        "This project cannot move further in that direction.",
    };
    return (
      messages[reason.code] ??
      "The project action could not be completed. Try again."
    );
  }
  return "The project action could not be completed. Try again.";
}
