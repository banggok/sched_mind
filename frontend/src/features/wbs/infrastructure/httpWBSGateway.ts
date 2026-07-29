import { WBSOperationError, type WBSGateway } from "../application/wbsGateway";
import type { WBSNode } from "../domain/wbs";
import { RequestCache } from "../../../shared/infrastructure/RequestCache";

const invalidResponseMessage = "WBS response is invalid. Try again.";

export function createHTTPWBSGateway(baseURL: string): WBSGateway {
  const trees = new Map<string, RequestCache<WBSNode[]>>();
  const treeVersions = new Map<string, number>();
  const request = async (path: string, init?: RequestInit): Promise<unknown> => {
    const response = await fetch(`${baseURL}${path}`, init);
    if (!response.ok) {
      const body: unknown = await response.json().catch(() => undefined);
      throw operationError(body);
    }
    if (response.status === 204) return undefined;
    const payload: unknown = await response.json();
    if (!isRecord(payload) || !("data" in payload)) {
      throw new Error(invalidResponseMessage);
    }
    return payload.data;
  };
  const path = (projectId: string) =>
    `/projects/${encodeURIComponent(projectId)}/wbs`;
  const invalidate = (projectId: string) => {
    treeVersions.set(projectId, (treeVersions.get(projectId) ?? 0) + 1);
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
      const requestVersion = treeVersions.get(projectId) ?? 0;
      return cache.run(async () => {
        const data = await request(path(projectId));
        if ((treeVersions.get(projectId) ?? 0) !== requestVersion) {
          throw staleResponseError();
        }
        if (!Array.isArray(data)) throw new Error(invalidResponseMessage);
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
      await request(
        `${path(projectId)}/${id}`,
        json("PUT", { name }),
      );
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
      await request(`${path(projectId)}/${id}`, {
        method: "DELETE",
      });
      invalidate(projectId);
    },
    updateExecutable: async (projectId, id, input) => {
      await request(
        `${path(projectId)}/${id}/executable`,
        json("PUT", input),
      );
      invalidate(projectId);
    },
    complete: async (projectId, id, actualEnd) => {
      await request(
        `${path(projectId)}/${id}/actual-end`,
        json("POST", { actualEnd }),
      );
      invalidate(projectId);
    },
    reopen: async (projectId, id) => {
      try {
        const response = await fetch(
          `${baseURL}${path(projectId)}/${encodeURIComponent(id)}/reopen`,
          { method: "POST" },
        );
        if (!response.ok) {
          const body: unknown = await response.json().catch(() => undefined);
          throw operationError(body);
        }

        // A successful command may already have changed persisted state even if
        // its response body is malformed. Invalidate before parsing so neither
        // the previous tree nor an older in-flight response can become confirmed.
        invalidate(projectId);
        const payload: unknown = await response.json();
        if (!isRecord(payload) || !("data" in payload)) {
          throw new Error(invalidResponseMessage);
        }
        return mapReopenedNode(payload.data, projectId, id);
      } catch (reason: unknown) {
        if (reason instanceof WBSOperationError) throw reason;
        throw new WBSOperationError(
          "TASK_REOPEN_FAILED",
          "The Task could not be reopened. Try again.",
        );
      }
    },
  };
}


function mapReopenedNode(
  value: unknown,
  projectId: string,
  id: string,
): WBSNode {
  if (
    !isRecord(value) ||
    !isRecord(value.executable) ||
    !("actualEnd" in value.executable) ||
    value.executable.actualEnd !== null
  ) {
    throw new Error(invalidResponseMessage);
  }
  const confirmed = mapNode(value);
  if (
    confirmed.id !== id ||
    confirmed.projectId !== projectId ||
    confirmed.hasChildren ||
    confirmed.children.length !== 0
  ) {
    throw new Error(invalidResponseMessage);
  }
  return confirmed;
}

function mapNode(value: unknown): WBSNode {
  if (
    !isRecord(value) ||
    typeof value.id !== "string" ||
    typeof value.projectId !== "string" ||
    typeof value.name !== "string" ||
    typeof value.position !== "number" ||
    typeof value.hasChildren !== "boolean" ||
    !Array.isArray(value.children) ||
    !isRecord(value.executable)
  ) {
    throw new Error(invalidResponseMessage);
  }
  return {
    id: value.id,
    projectId: value.projectId,
    parentId: optionalString(value.parentId),
    name: value.name,
    position: value.position,
    hasChildren: value.hasChildren,
    executable: mapExecutable(value.executable),
    children: value.children.map(mapNode),
  };
}

function mapExecutable(value: Record<string, unknown>): WBSNode["executable"] {
  if (
    !isRecord(value.executionTimeline) ||
    !isRecord(value.commitmentTimeline)
  ) {
    throw new Error(invalidResponseMessage);
  }
  return {
    roleId: optionalString(value.roleId),
    assigneeId: optionalString(value.assigneeId),
    effortMinutes: optionalNumber(value.effortMinutes),
    executionTimeline: mapTimeline(value.executionTimeline),
    commitmentTimeline: mapTimeline(value.commitmentTimeline),
    actualEnd: optionalString(value.actualEnd),
  };
}

function mapTimeline(value: Record<string, unknown>): {
  start?: string;
  end?: string;
} {
  return {
    start: optionalString(value.start),
    end: optionalString(value.end),
  };
}

function optionalString(value: unknown): string | undefined {
  if (value === undefined || value === null) return undefined;
  if (typeof value !== "string") throw new Error(invalidResponseMessage);
  return value;
}

function optionalNumber(value: unknown): number | undefined {
  if (value === undefined || value === null) return undefined;
  if (typeof value !== "number" || !Number.isFinite(value)) {
    throw new Error(invalidResponseMessage);
  }
  return value;
}

function operationError(value: unknown): WBSOperationError {
  if (isRecord(value)) {
    const code =
      typeof value.code === "string" ? value.code : "UNEXPECTED_RESPONSE";
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
    case "TASK_NOT_FOUND":
      return "The Task could not be found. Refresh Project Structure.";
    case "EXECUTABLE_TASK_REQUIRED":
      return "Only a Task can be reopened.";
    case "TASK_NOT_COMPLETED":
      return "This Task is already unfinished. Refresh Project Structure.";
    case "TASK_REOPEN_CONFLICT":
      return "The Task changed in another request. Refresh and try again.";
    case "TASK_REOPEN_FAILED":
      return "The Task could not be reopened. Try again.";
    default:
      return "The item could not be updated. Check the form and try again.";
  }
}

function staleResponseError(): DOMException {
  return new DOMException("The response was invalidated", "AbortError");
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}
