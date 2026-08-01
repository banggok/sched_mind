import { schedulingImpactFetch } from "../../../shared/infrastructure/schedulingImpactFetch";
import { RequestCache } from "../../../shared/infrastructure/RequestCache";
import {
  advanceScheduleProjectionVersion,
  currentScheduleProjectionVersion,
} from "../../../shared/infrastructure/scheduleProjectionClock";
import {
  DependencyOperationError,
  type DependenciesGateway,
} from "../application/dependenciesGateway";
import type {
  DependencyCandidatePage,
  DependencyDetail,
  DependencyRelation,
  DependencyTask,
} from "../domain/dependency";

export function createHTTPDependenciesGateway(
  baseURL: string,
): DependenciesGateway {
  const details = new Map<string, RequestCache<DependencyDetail>>();
  let projectionVersion = currentScheduleProjectionVersion();
  const invalidateAllCaches = (advance: boolean) => {
    if (advance) advanceScheduleProjectionVersion();
    projectionVersion = currentScheduleProjectionVersion();
    details.forEach((cache) => cache.invalidate());
    details.clear();
  };
  const syncProjectionVersion = () => {
    if (projectionVersion !== currentScheduleProjectionVersion()) {
      invalidateAllCaches(false);
    }
  };
  const request = async (
    path: string,
    init?: RequestInit,
  ): Promise<unknown> => {
    const response = await schedulingImpactFetch(`${baseURL}${path}`, init);
    if (!response.ok) {
      const body: unknown = await response.json().catch(() => undefined);
      throw mapError(body);
    }
    if (response.status === 204) return undefined;
    const body: unknown = await response.json();
    if (!isRecord(body) || !("data" in body))
      throw new Error("Dependency response is invalid.");
    return body.data;
  };
  return {
    list: (taskId, signal) => {
      syncProjectionVersion();
      const cache = details.get(taskId) ?? new RequestCache<DependencyDetail>();
      details.set(taskId, cache);
      const requestVersion = projectionVersion;
      return cache.run(async () => {
        const value = mapDetail(
          await request(`/tasks/${encodeURIComponent(taskId)}/dependencies`),
        );
        if (projectionVersion !== requestVersion) throw staleResponseError();
        return value;
      }, signal);
    },
    candidates: async (taskId, direction, search, page, pageSize, signal) => {
      syncProjectionVersion();
      const requestVersion = projectionVersion;
      const query = new URLSearchParams({
        taskId,
        direction,
        search,
        page: String(page),
        pageSize: String(pageSize),
      });
      const value = mapPage(
        await request(`/dependency-candidates?${query}`, { signal }),
      );
      if (projectionVersion !== requestVersion) throw staleResponseError();
      return value;
    },
    create: async (blockingTaskId, blockedTaskId) => {
      await request("/dependencies", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ blockingTaskId, blockedTaskId }),
      });
      invalidateAllCaches(true);
    },
    remove: async (dependencyId) => {
      await request(`/dependencies/${encodeURIComponent(dependencyId)}`, {
        method: "DELETE",
      });
      invalidateAllCaches(true);
    },
    keepAsManual: async (dependencyId) => {
      await request(
        `/dependencies/${encodeURIComponent(dependencyId)}/keep-manual`,
        { method: "POST" },
      );
      invalidateAllCaches(true);
    },
    invalidateTask: (taskId) => {
      details.get(taskId)?.invalidate();
      details.delete(taskId);
    },
    invalidateAll: () => {
      invalidateAllCaches(true);
      details.clear();
    },
  };
}

function mapDetail(value: unknown): DependencyDetail {
  if (
    !isRecord(value) ||
    !Array.isArray(value.blockedBy) ||
    !Array.isArray(value.blocks)
  )
    throw new Error("Dependency response is invalid.");
  return {
    blockedBy: value.blockedBy.map(mapRelation),
    blocks: value.blocks.map(mapRelation),
  };
}
function mapRelation(value: unknown): DependencyRelation {
  if (!isRecord(value) || typeof value.id !== "string")
    throw new Error("Dependency response is invalid.");
  if (
    value.source !== "manual" &&
    value.source !== "automatic" &&
    value.source !== "both"
  ) {
    throw new Error("Dependency response is invalid.");
  }
  if (typeof value.manualRemovable !== "boolean") {
    throw new Error("Dependency response is invalid.");
  }
  return {
    id: value.id,
    source: value.source,
    manualRemovable: value.manualRemovable,
    task: mapTask(value.task),
  };
}
function mapTask(value: unknown): DependencyTask {
  if (
    !isRecord(value) ||
    typeof value.id !== "string" ||
    typeof value.name !== "string" ||
    typeof value.projectId !== "string" ||
    typeof value.projectName !== "string" ||
    typeof value.hierarchyPath !== "string" ||
    typeof value.completed !== "boolean"
  ) {
    throw new Error("Dependency task response is invalid.");
  }
  return {
    id: value.id,
    name: value.name,
    projectId: value.projectId,
    projectName: value.projectName,
    hierarchyPath: value.hierarchyPath,
    completed: value.completed,
    expectedStart:
      typeof value.expectedStart === "string" ? value.expectedStart : undefined,
  };
}
function mapPage(value: unknown): DependencyCandidatePage {
  if (
    !isRecord(value) ||
    !Array.isArray(value.items) ||
    typeof value.page !== "number" ||
    typeof value.pageSize !== "number" ||
    typeof value.totalItems !== "number"
  )
    throw new Error("Dependency candidate response is invalid.");
  return {
    items: value.items.map(mapTask),
    page: value.page,
    pageSize: value.pageSize,
    totalItems: value.totalItems,
  };
}
function mapError(value: unknown): DependencyOperationError {
  if (!isRecord(value))
    return new DependencyOperationError(
      "UNEXPECTED_RESPONSE",
      "The dependency could not be updated. Try again.",
    );
  const code =
    typeof value.code === "string" ? value.code : "UNEXPECTED_RESPONSE";
  const path =
    isRecord(value.details) && Array.isArray(value.details.path)
      ? value.details.path.flatMap((step) =>
          isRecord(step) && typeof step.taskName === "string"
            ? [step.taskName]
            : [],
        )
      : [];
  const messages: Record<string, string> = {
    DEPENDENCY_SELF_REFERENCE: "A task cannot block itself.",
    DEPENDENCY_ALREADY_EXISTS: "This dependency already exists.",
    DEPENDENCY_EXECUTABLE_TASK_REQUIRED:
      "Dependencies can only connect tasks, not groups.",
    DEPENDENCY_CLOSED_PROJECT_TASK_NOT_ALLOWED:
      "Tasks in a closed project cannot be added.",
    DEPENDENCY_COMPLETED_TASK_CANNOT_BE_BLOCKED:
      "A completed task cannot be blocked by a new dependency.",
    DEPENDENCY_COMPLETED_HISTORY_READ_ONLY:
      "This historical dependency is read-only.",
    DEPENDENCY_AUTOMATIC_ONLY_READ_ONLY:
      "Automatic dependencies cannot be removed directly. Keep it as manual first if manual ownership is required.",
  };
  const message =
    code === "DEPENDENCY_CYCLE_DETECTED"
      ? `This creates a cycle${path.length ? `: ${path.join(" → ")}` : "."}`
      : (messages[code] ?? "The dependency could not be updated. Try again.");
  return new DependencyOperationError(code, message, path);
}
function staleResponseError(): DOMException {
  return new DOMException("The response was invalidated", "AbortError");
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}
