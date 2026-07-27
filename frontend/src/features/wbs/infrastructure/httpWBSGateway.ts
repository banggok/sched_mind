import { WBSOperationError, type WBSGateway } from "../application/wbsGateway";
import type { WBSNode } from "../domain/wbs";
import { RequestCache } from "../../../shared/infrastructure/RequestCache";

export function createHTTPWBSGateway(baseURL: string): WBSGateway {
  const trees = new Map<string, RequestCache<WBSNode[]>>();
  const request = async <T>(path: string, init?: RequestInit): Promise<T> => {
    const response = await fetch(`${baseURL}${path}`, init);
    if (!response.ok) {
      const body: unknown = await response.json().catch(() => undefined);
      throw operationError(body);
    }
    if (response.status === 204) return undefined as T;
    const payload: unknown = await response.json();
    if (typeof payload !== "object" || payload === null || !("data" in payload))
      throw new Error("WBS response is invalid. Try again.");
    return payload.data as T;
  };
  const path = (projectId: string) =>
    `/projects/${encodeURIComponent(projectId)}/wbs`;
  const invalidate = (projectId: string) => {
    trees.get(projectId)?.invalidate();
    trees.delete(projectId);
  };
  const json = (method: string, body: object): RequestInit => ({
    method,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  return {
    tree: async (projectId, signal) => {
      const cache = trees.get(projectId) ?? new RequestCache<WBSNode[]>();
      trees.set(projectId, cache);
      return cache.run(async () => {
        const data = await request<unknown>(path(projectId));
        if (!Array.isArray(data))
          throw new Error("WBS response is invalid. Try again.");
        return data.map(mapNode);
      }, signal);
    },
    create: async (projectId, parentId, name, confirmConversion) => {
      await request(
        path(projectId),
        json("POST", { parentId, name, confirmConversion }),
      );
      invalidate(projectId);
    },
    rename: async (projectId, id, name) => {
      await request(`${path(projectId)}/${id}`, json("PUT", { name }));
      invalidate(projectId);
    },
    reorder: async (projectId, id, direction) => {
      await request(
        `${path(projectId)}/${id}/reorder`,
        json("POST", { direction }),
      );
      invalidate(projectId);
    },
    move: async (projectId, id, parentId, confirmConversion) => {
      await request(
        `${path(projectId)}/${id}/move`,
        json("POST", { parentId, confirmConversion }),
      );
      invalidate(projectId);
    },
    remove: async (projectId, id) => {
      await request(`${path(projectId)}/${id}`, { method: "DELETE" });
      invalidate(projectId);
    },
    updateExecutable: async (projectId, id, input) => {
      await request(`${path(projectId)}/${id}/executable`, json("PUT", input));
      invalidate(projectId);
    },
    complete: async (projectId, id, actualEnd) => {
      await request(
        `${path(projectId)}/${id}/actual-end`,
        json("POST", { actualEnd }),
      );
      invalidate(projectId);
    },
  };
}
function mapNode(value: unknown): WBSNode {
  if (
    typeof value !== "object" ||
    value === null ||
    !("id" in value) ||
    typeof value.id !== "string" ||
    !("name" in value) ||
    typeof value.name !== "string" ||
    !("children" in value) ||
    !Array.isArray(value.children)
  )
    throw new Error("WBS response is invalid. Try again.");
  const source = value as Record<string, unknown>;
  const executable =
    typeof source.executable === "object" && source.executable !== null
      ? (source.executable as WBSNode["executable"])
      : { executionTimeline: {}, commitmentTimeline: {} };
  return {
    id: value.id,
    projectId: typeof source.projectId === "string" ? source.projectId : "",
    parentId: typeof source.parentId === "string" ? source.parentId : undefined,
    name: value.name,
    position: typeof source.position === "number" ? source.position : 0,
    hasChildren: source.hasChildren === true,
    executable,
    children: value.children.map(mapNode),
  };
}
function operationError(value: unknown): WBSOperationError {
  if (
    typeof value === "object" &&
    value !== null &&
    "message" in value &&
    typeof value.message === "string"
  ) {
    const code =
      "code" in value && typeof value.code === "string"
        ? value.code
        : "UNEXPECTED_RESPONSE";
    return new WBSOperationError(code, userMessage(code));
  }
  return new WBSOperationError(
    "UNEXPECTED_RESPONSE",
    "The item could not be updated. Try again.",
  );
}
function userMessage(code: string): string {
  switch (code) {
    case "WBS_HAS_CHILDREN":
      return "This group cannot be deleted while it still contains items. Move or delete its children first.";
    case "WBS_CONVERSION_REQUIRED":
      return "This task contains details and must be converted into a group.";
    case "WBS_MOVE_CYCLE":
      return "An item cannot be moved inside itself or one of its own children.";
    case "WBS_NAME_EXISTS":
      return "An item with this name already exists under the selected parent.";
    case "COMPLETED_TASK_READ_ONLY":
      return "Completed tasks are read-only.";
    case "PROJECT_CLOSED_READ_ONLY":
      return "Closed projects are read-only.";
    default:
      return "The item could not be updated. Check the form and try again.";
  }
}
