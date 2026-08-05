import { useEffect, useRef, useState } from "react";
import { Alert } from "../../../shared/presentation/Alert";
import { Button } from "../../../shared/presentation/Button";
import { Dialog } from "../../../shared/presentation/Dialog";
import { FormField } from "../../../shared/presentation/FormField";
import { ListSkeleton } from "../../../shared/presentation/ListSkeleton";
import type { DependenciesGateway } from "../../dependencies/application/dependenciesGateway";
import type { Project } from "../../projects/domain/project";
import type { RolesGateway } from "../../roles/application/rolesGateway";
import type { TeamMembersGateway } from "../../team-members/application/teamMembersGateway";
import { WBSOperationError, type WBSGateway } from "../application/wbsGateway";
import type { WBSNode } from "../domain/wbs";
import { WBSDetailDialog } from "./WBSDetailDialog";

type Draft = {
  mode: "create" | "move";
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
  initialNodeId,
  initialCreateParentId,
  initialMoveNodeId,
  onCreated,
  onMutated,
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
  initialNodeId?: string;
  initialCreateParentId?: string | null;
  initialMoveNodeId?: string;
  onCreated?(node: WBSNode): void;
  onMutated?(nodeId: string): void;
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
  const [detail, setDetail] = useState<WBSNode>();
  const [conversion, setConversion] = useState(false);
  const initialActionApplied = useRef(false);

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
        if (initialActionApplied.current) {
          if (initialNodeId) {
            const selected = findNode(confirmedTree, initialNodeId);
            if (selected) setDetail(selected);
            else {
              setDetail(undefined);
              setError("The selected WBS item no longer exists.");
            }
          } else if (
            initialCreateParentId !== undefined &&
            initialCreateParentId !== null &&
            !findNode(confirmedTree, initialCreateParentId)
          ) {
            setDraft(undefined);
            setError("The selected parent no longer exists.");
          } else if (initialMoveNodeId) {
            const selected = findNode(confirmedTree, initialMoveNodeId);
            if (selected)
              setDraft((current) =>
                current?.mode === "move"
                  ? { ...current, node: selected }
                  : current,
              );
            else {
              setDraft(undefined);
              setError("The selected WBS item no longer exists.");
            }
          }
          return;
        }

        const actionCount =
          Number(initialNodeId !== undefined) +
          Number(initialCreateParentId !== undefined) +
          Number(initialMoveNodeId !== undefined);
        initialActionApplied.current = true;
        if (actionCount !== 1) {
          setError("Choose one WBS action from Home and try again.");
          return;
        }
        if (initialNodeId) {
          const selected = findNode(confirmedTree, initialNodeId);
          if (selected) setDetail(selected);
          else setError("The selected WBS item no longer exists.");
          return;
        }
        if (initialCreateParentId !== undefined) {
          if (
            initialCreateParentId !== null &&
            !findNode(confirmedTree, initialCreateParentId)
          ) {
            setError("The selected parent no longer exists.");
            return;
          }
          setDraft({
            mode: "create",
            parentId: initialCreateParentId ?? undefined,
          });
          setName("");
          return;
        }
        if (initialMoveNodeId) {
          const selected = findNode(confirmedTree, initialMoveNodeId);
          if (selected) {
            setDraft({ mode: "move", node: selected });
            setTarget("");
          } else setError("The selected WBS item no longer exists.");
        }
      })
      .catch((reason: unknown) => {
        if (!(reason instanceof DOMException && reason.name === "AbortError"))
          setError("The WBS item could not be loaded. Try again.");
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, [
    gateway,
    initialCreateParentId,
    initialMoveNodeId,
    initialNodeId,
    project.id,
    reload,
  ]);

  const all = flatten(tree);
  const mutate = async <Result,>(
    action: () => Promise<Result>,
    onSuccess?: (result: Result) => void,
  ) => {
    if (busy) return;
    setBusy(true);
    setOperationError("");
    try {
      const result = await action();
      dependenciesGateway?.invalidateAll();
      onSuccess?.(result);
      onClose();
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
        onCreated,
      );
    else if (selected)
      void mutate(
        () => gateway.move(project.id, selected.id, target || undefined, false),
        () => onMutated?.(selected.id),
      );
  };

  return (
    <>
      {loading && !draft && !detail ? (
        <Dialog titleID="wbs-direct-loading-title" onClose={onClose}>
          <h2 id="wbs-direct-loading-title" className="text-xl font-extrabold">
            Loading WBS item
          </h2>
          <div className="mt-5">
            <ListSkeleton label="Loading WBS item" />
          </div>
        </Dialog>
      ) : error && !draft && !detail ? (
        <Dialog titleID="wbs-direct-error-title" onClose={onClose}>
          <h2 id="wbs-direct-error-title" className="text-xl font-extrabold">
            WBS item unavailable
          </h2>
          <Alert tone="danger" className="mt-4">
            {error}
          </Alert>
          <div className="mt-6 flex justify-end">
            <Button onClick={onClose}>Close</Button>
          </div>
        </Dialog>
      ) : null}
      {draft ? (
        <Dialog
          titleID="wbs-form-title"
          onClose={() => {
            if (!busy) onClose();
          }}
        >
          <form
            onSubmit={(event) => {
              event.preventDefault();
              submit();
            }}
          >
            <h2 id="wbs-form-title" className="text-xl font-extrabold">
              {conversion
                ? "This task will become a group"
                : draft.mode === "create"
                  ? draft.parentId
                    ? "Add Child"
                    : "Add Task"
                  : "Move Item"}
            </h2>
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
                    onChange={(event) => setTarget(event.target.value)}
                  >
                    <option value="">Project root</option>
                    {all
                      .filter(
                        (node) =>
                          node.id !== draft.node?.id &&
                          !descendants(draft.node, node.id),
                      )
                      .map((node) => (
                        <option key={node.id} value={node.id}>
                          {node.name}
                        </option>
                      ))}
                  </select>
                </div>
              ) : (
                <FormField
                  label="Name"
                  id="wbs-name"
                  maxLength={200}
                  value={name}
                  onChange={(event) => setName(event.target.value)}
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
                  onClose();
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
                            onCreated,
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
                            () => onMutated?.(selected.id),
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
          key={`${detail.id}:${detail.executable.actualStart ?? "unfinished"}:${detail.executable.actualEnd ?? "unfinished"}`}
          project={project}
          node={detail}
          gateway={gateway}
          dependenciesGateway={dependenciesGateway}
          rolesGateway={rolesGateway}
          membersGateway={membersGateway}
          loadPublicHolidayDates={loadPublicHolidayDates}
          nested={false}
          onClose={onClose}
          onChanged={() => {
            dependenciesGateway?.invalidateAll();
            onMutated?.(detail.id);
            onClose();
          }}
          onReopened={() => {
            dependenciesGateway?.invalidateAll();
            onMutated?.(detail.id);
            onClose();
          }}
        />
      ) : null}
    </>
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
