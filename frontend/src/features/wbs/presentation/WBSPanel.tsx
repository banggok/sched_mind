import { useEffect, useState } from "react";
import type { Project } from "../../projects/domain/project";
import { WBSOperationError, type WBSGateway } from "../application/wbsGateway";
import type { WBSNode } from "../domain/wbs";
import { Alert } from "../../../shared/presentation/Alert";
import { Button } from "../../../shared/presentation/Button";
import { Dialog } from "../../../shared/presentation/Dialog";
import { EmptyState } from "../../../shared/presentation/EmptyState";
import { FormField } from "../../../shared/presentation/FormField";
import { ListSkeleton } from "../../../shared/presentation/ListSkeleton";
import { Toast } from "../../../shared/presentation/Toast";
import type { RolesGateway } from "../../roles/application/rolesGateway";
import type { TeamMembersGateway } from "../../team-members/application/teamMembersGateway";
import { WBSDetailDialog } from "./WBSDetailDialog";
import type { DependenciesGateway } from "../../dependencies/application/dependenciesGateway";

type Draft = {
  mode: "create" | "rename" | "move";
  node?: WBSNode;
  parentId?: string;
};
export function WBSPanel({
  project,
  gateway,
  dependenciesGateway,
  rolesGateway,
  membersGateway,
  loadPublicHolidayDates,
  onClose,
}: {
  project: Project;
  gateway: WBSGateway;
  dependenciesGateway?: DependenciesGateway;
  rolesGateway: RolesGateway;
  membersGateway: TeamMembersGateway;
  loadPublicHolidayDates?(
    startDate: string,
    endDate: string,
  ): Promise<string[]>;
  onClose(): void;
}) {
  const [tree, setTree] = useState<WBSNode[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [reload, setReload] = useState(0);
  const [draft, setDraft] = useState<Draft>();
  const [name, setName] = useState("");
  const [target, setTarget] = useState("");
  const [busy, setBusy] = useState(false);
  const [operationError, setOperationError] = useState("");
  const [toast, setToast] = useState("");
  const [detail, setDetail] = useState<WBSNode>();
  const [conversion, setConversion] = useState(false);
  useEffect(() => {
    return gateway.subscribeToConfirmedChanges?.(() =>
      setReload((value) => value + 1),
    );
  }, [gateway]);
  useEffect(() => {
    const controller = new AbortController();
    setLoading(true);
    setError("");
    void gateway
      .tree(project.id, controller.signal)
      .then((confirmedTree) => {
        setTree(confirmedTree);
        setDetail((current) =>
          current ? findNode(confirmedTree, current.id) : undefined,
        );
      })
      .catch((reason: unknown) => {
        if (!(reason instanceof DOMException && reason.name === "AbortError"))
          setError("Project structure could not be loaded. Try again.");
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, [gateway, project.id, reload]);
  const all = flatten(tree);
  const refresh = (message: string) => {
    dependenciesGateway?.invalidateAll();
    setToast(message);
    setDraft(undefined);
    setConversion(false);
    setReload((v) => v + 1);
  };
  const mutate = async (action: () => Promise<void>, message: string) => {
    if (busy) return;
    setBusy(true);
    setOperationError("");
    try {
      await action();
      refresh(message);
    } catch (reason: unknown) {
      if (
        reason instanceof WBSOperationError &&
        reason.code === "WBS_CONVERSION_REQUIRED"
      )
        setConversion(true);
      else
        setOperationError(
          reason instanceof Error
            ? reason.message
            : "The item could not be updated.",
        );
    } finally {
      setBusy(false);
    }
  };
  const submit = () => {
    if (!draft) return;
    const selected = draft.node;
    if (draft.mode === "create")
      void mutate(
        () => gateway.create(project.id, draft.parentId, name, false),
        "Task added.",
      );
    else if (draft.mode === "rename" && selected)
      void mutate(
        () => gateway.rename(project.id, selected.id, name),
        "Item updated.",
      );
    else if (selected)
      void mutate(
        () => gateway.move(project.id, selected.id, target || undefined, false),
        "Item moved.",
      );
  };
  return (
    <Dialog titleID="wbs-title" onClose={onClose} wide>
      <div className="flex items-start justify-between gap-4">
        <div>
          <p className="text-sm font-bold text-action">
            Projects / {project.name}
          </p>
          <h2 id="wbs-title" className="mt-2 text-2xl font-extrabold">
            Project Structure
          </h2>
          <p className="mt-1 text-sm text-muted">
            Organise project work into groups and tasks.
          </p>
        </div>
        <Button onClick={onClose}>Close</Button>
      </div>
      <div className="mt-6 flex justify-end">
        <Button
          variant="primary"
          disabled={project.status === "closed"}
          onClick={() => {
            setDraft({ mode: "create" });
            setName("");
          }}
        >
          + Add Task
        </Button>
      </div>
      <div className="mt-5 rounded-panel border border-border-subtle">
        {loading && tree.length === 0 ? (
          <ListSkeleton label="Loading project structure" />
        ) : error ? (
          <div className="p-8 text-center">
            <Alert tone="danger">{error}</Alert>
            <Button className="mt-4" onClick={() => setReload((v) => v + 1)}>
              Try again
            </Button>
          </div>
        ) : tree.length === 0 ? (
          <EmptyState
            title="No tasks or groups yet"
            description="Add the first task to start planning this project."
            action={
              <Button
                variant="quiet"
                onClick={() => {
                  setDraft({ mode: "create" });
                  setName("");
                }}
              >
                Add Task
              </Button>
            }
          />
        ) : (
          <ul className="p-4" role="tree">
            {tree.map((node, index) => (
              <TreeNode
                key={node.id}
                node={node}
                first={index === 0}
                last={index === tree.length - 1}
                disabled={busy || project.status === "closed"}
                onCreate={(selected) => {
                  setDraft({ mode: "create", parentId: selected.id });
                  setName("");
                }}
                onRename={(selected) => {
                  setDraft({ mode: "rename", node: selected });
                  setName(selected.name);
                }}
                onMove={(selected) => {
                  setDraft({ mode: "move", node: selected });
                  setTarget("");
                }}
                onReorder={(selected, d) =>
                  void mutate(
                    () => gateway.reorder(project.id, selected.id, d),
                    "Item reordered.",
                  )
                }
                onDelete={(selected) =>
                  void mutate(
                    () => gateway.remove(project.id, selected.id),
                    "Task deleted.",
                  )
                }
                onDetail={setDetail}
              />
            ))}
          </ul>
        )}
      </div>
      {draft ? (
        <Dialog
          nested
          titleID="wbs-form-title"
          onClose={() => !busy && setDraft(undefined)}
        >
          <form
            onSubmit={(event) => {
              event.preventDefault();
              submit();
            }}
          >
            <h3 id="wbs-form-title" className="text-xl font-extrabold">
              {conversion
                ? "This task will become a group"
                : draft.mode === "create"
                  ? draft.parentId
                    ? "Add Child"
                    : "Add Task"
                  : draft.mode === "rename"
                    ? `Edit ${draft.node?.hasChildren ? "Group" : "Task"}`
                    : "Move Item"}
            </h3>
            <div className="mt-5">
              {conversion ? (
                <Alert tone="success">
                  {draft.mode === "move"
                    ? "The destination task will become a group. Its current task details will be moved into a new child task so no data is lost."
                    : `Adding a child will turn “${all.find((item) => item.id === draft.parentId)?.name ?? "this task"}” into a group. Its current task details will be moved to the new child task so no data is lost.`}
                </Alert>
              ) : draft.mode === "move" ? (
                <div>
                  <label
                    className="block text-label font-bold"
                    htmlFor="wbs-parent"
                  >
                    Destination group
                  </label>
                  <select
                    id="wbs-parent"
                    className="ui-input mt-2"
                    value={target}
                    onChange={(e) => setTarget(e.target.value)}
                  >
                    <option value="">Project root</option>
                    {all
                      .filter(
                        (n) =>
                          n.id !== draft.node?.id &&
                          !descendants(draft.node, n.id),
                      )
                      .map((n) => (
                        <option key={n.id} value={n.id}>
                          {n.name}
                        </option>
                      ))}
                  </select>
                </div>
              ) : (
                <FormField
                  label="Name"
                  id="wbs-name"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  autoFocus
                />
              )}
              {operationError ? (
                <Alert tone="danger" className="mt-4">
                  {operationError}
                </Alert>
              ) : null}
            </div>
            <div className="mt-6 flex justify-end gap-3">
              <Button
                type="button"
                disabled={busy}
                onClick={() => {
                  setConversion(false);
                  setDraft(undefined);
                }}
              >
                Cancel
              </Button>
              <Button
                type={conversion ? "button" : "submit"}
                variant="primary"
                loading={busy}
                onClick={
                  conversion
                    ? () => {
                        if (draft.mode === "create")
                          void mutate(
                            () =>
                              gateway.create(
                                project.id,
                                draft.parentId,
                                name,
                                true,
                              ),
                            "Child added and task converted to a group.",
                          );
                        else if (draft.node) {
                          const selected = draft.node;
                          void mutate(
                            () =>
                              gateway.move(
                                project.id,
                                selected.id,
                                target || undefined,
                                true,
                              ),
                            "Item moved and destination converted to a group.",
                          );
                        }
                      }
                    : undefined
                }
              >
                {conversion ? "Add Child and Convert" : "Save"}
              </Button>
            </div>
          </form>
        </Dialog>
      ) : null}
      {detail ? (
        <WBSDetailDialog
          key={`${detail.id}:${detail.executable.actualEnd ?? "unfinished"}`}
          project={project}
          node={detail}
          gateway={gateway}
          dependenciesGateway={dependenciesGateway}
          rolesGateway={rolesGateway}
          membersGateway={membersGateway}
          loadPublicHolidayDates={loadPublicHolidayDates}
          onClose={() => setDetail(undefined)}
          onChanged={(message) => {
            setDetail(undefined);
            refresh(message);
          }}
          onReopened={(confirmed, message) => {
            dependenciesGateway?.invalidateAll();
            setTree((current) => replaceNode(current, confirmed));
            setDetail(confirmed);
            setToast(message);
          }}
        />
      ) : null}
      <Toast message={toast} onDismiss={() => setToast("")} />
    </Dialog>
  );
}

function replaceNode(values: WBSNode[], replacement: WBSNode): WBSNode[] {
  return values.map((value) => {
    if (value.id === replacement.id) return replacement;
    if (value.children.length === 0) return value;
    return {
      ...value,
      children: replaceNode(value.children, replacement),
    };
  });
}

function TreeNode({
  node,
  first,
  last,
  disabled,
  onCreate,
  onRename,
  onMove,
  onReorder,
  onDelete,
  onDetail,
}: {
  node: WBSNode;
  first: boolean;
  last: boolean;
  disabled: boolean;
  onCreate(node: WBSNode): void;
  onRename(node: WBSNode): void;
  onMove(node: WBSNode): void;
  onReorder(node: WBSNode, d: "up" | "down"): void;
  onDelete(node: WBSNode): void;
  onDetail(node: WBSNode): void;
}) {
  return (
    <li
      role="treeitem"
      aria-expanded={node.hasChildren || undefined}
      className="py-2"
    >
      <div className="flex flex-wrap items-center gap-2 rounded-control border border-border-subtle p-3">
        <div className="mr-auto min-w-0">
          <strong>{node.name}</strong>
          <span className="ml-2 text-xs text-muted">
            {node.hasChildren ? "Group" : "Task"}
          </span>
          {node.executable.effortMinutes ? (
            <span className="ml-2 text-xs text-muted">
              {node.executable.effortMinutes / 60}h
            </span>
          ) : null}
        </div>
        <Button compact onClick={() => onDetail(node)}>
          {node.hasChildren ? "View Group" : "Edit Task"}
        </Button>
        <Button
          compact
          disabled={disabled || first}
          aria-label={`Move ${node.name} up`}
          onClick={() => onReorder(node, "up")}
        >
          ↑
        </Button>
        <Button
          compact
          disabled={disabled || last}
          aria-label={`Move ${node.name} down`}
          onClick={() => onReorder(node, "down")}
        >
          ↓
        </Button>
        <Button compact disabled={disabled} onClick={() => onCreate(node)}>
          Add Child
        </Button>
        {node.hasChildren ? (
          <Button
            compact
            disabled={disabled || Boolean(node.executable.actualEnd)}
            onClick={() => onRename(node)}
          >
            Edit Group
          </Button>
        ) : null}
        <Button compact disabled={disabled} onClick={() => onMove(node)}>
          Move Item
        </Button>
        {!node.hasChildren ? (
          <Button
            compact
            variant="danger"
            disabled={disabled || Boolean(node.executable.actualEnd)}
            onClick={() => onDelete(node)}
          >
            Delete Task
          </Button>
        ) : null}
      </div>
      {node.children.length ? (
        <ul role="group" className="ml-6 border-l border-border-subtle pl-3">
          {node.children.map((child, index) => (
            <TreeNode
              key={child.id}
              node={child}
              first={index === 0}
              last={index === node.children.length - 1}
              disabled={disabled}
              onCreate={onCreate}
              onRename={onRename}
              onMove={onMove}
              onReorder={onReorder}
              onDelete={onDelete}
              onDetail={onDetail}
            />
          ))}
        </ul>
      ) : null}
    </li>
  );
}
function findNode(nodes: WBSNode[], id: string): WBSNode | undefined {
  const pending = [...nodes];
  while (pending.length > 0) {
    const node = pending.pop();
    if (!node) continue;
    if (node.id === id) return node;
    pending.push(...node.children);
  }
  return undefined;
}

function flatten(nodes: WBSNode[]): WBSNode[] {
  return nodes.flatMap((node) => [node, ...flatten(node.children)]);
}
function descendants(node: WBSNode | undefined, id: string): boolean {
  return node ? flatten(node.children).some((child) => child.id === id) : false;
}
