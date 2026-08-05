import { schedulingImpactFetch } from "../../../shared/infrastructure/schedulingImpactFetch";
import {
  WBSOperationError,
  type AllocationGroups,
  type AllocationRow,
  type SchedulePreview,
  type WBSGateway,
} from "../application/wbsGateway";
import type { WBSNode } from "../domain/wbs";
import { RequestCache } from "../../../shared/infrastructure/RequestCache";
import {
  advanceScheduleProjectionVersion,
  currentScheduleProjectionVersion,
  subscribeScheduleProjectionVersion,
} from "../../../shared/infrastructure/scheduleProjectionClock";

const invalidResponseMessage = "WBS response is invalid. Try again.";

export function createHTTPWBSGateway(baseURL: string): WBSGateway {
  const trees = new Map<string, RequestCache<WBSNode[]>>();
  let observedProjectionVersion = currentScheduleProjectionVersion();
  const request = async (
    path: string,
    init?: RequestInit,
  ): Promise<unknown> => {
    const response = await schedulingImpactFetch(`${baseURL}${path}`, init);
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
  const invalidateAll = (advance: boolean) => {
    if (advance) advanceScheduleProjectionVersion();
    observedProjectionVersion = currentScheduleProjectionVersion();
    trees.forEach((cache) => cache.invalidate());
    trees.clear();
  };
  const syncProjectionVersion = () => {
    if (observedProjectionVersion !== currentScheduleProjectionVersion()) {
      invalidateAll(false);
    }
  };
  const json = (method: string, body: object): RequestInit => ({
    method,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  return {
    subscribeToConfirmedChanges: subscribeScheduleProjectionVersion,
    tree: async (projectId, signal) => {
      syncProjectionVersion();
      const cache = trees.get(projectId) ?? new RequestCache<WBSNode[]>();
      trees.set(projectId, cache);
      const requestVersion = currentScheduleProjectionVersion();
      return cache.run(async () => {
        const data = await request(path(projectId));
        if (currentScheduleProjectionVersion() !== requestVersion) {
          throw staleResponseError();
        }
        if (!Array.isArray(data)) throw new Error(invalidResponseMessage);
        return data.map(mapNode);
      }, signal);
    },
    allocations: async (projectId, id, signal) => {
      syncProjectionVersion();
      const requestVersion = currentScheduleProjectionVersion();
      const data = await request(
        `${path(projectId)}/${encodeURIComponent(id)}/allocations`,
        { signal },
      );
      if (currentScheduleProjectionVersion() !== requestVersion) {
        throw staleResponseError();
      }
      return mapAllocationGroups(data);
    },
    create: async (projectId, parentId, name, confirmConversion) => {
      const response = await schedulingImpactFetch(
        `${baseURL}${path(projectId)}`,
        json("POST", { parentId, name, confirmConversion }),
      );
      if (!response.ok) {
        const body: unknown = await response.json().catch(() => undefined);
        throw operationError(body);
      }

      // The command may already be persisted even when its response cannot be
      // decoded. Invalidate before parsing so Home and WBS cannot retain stale
      // projections after a confirmed success.
      invalidateAll(true);
      const payload: unknown = await response.json();
      if (!isRecord(payload) || !("data" in payload)) {
        throw new Error(invalidResponseMessage);
      }
      const created = mapNode(payload.data);
      const expectedParentID = parentId ?? undefined;
      if (
        created.id.trim() === "" ||
        created.projectId !== projectId ||
        created.parentId !== expectedParentID
      ) {
        throw new Error(invalidResponseMessage);
      }
      return created;
    },
    rename: async (projectId, id, name) => {
      await request(`${path(projectId)}/${id}`, json("PUT", { name }));
      invalidateAll(true);
    },
    reorder: async (projectId, id, direction) => {
      await request(
        `${path(projectId)}/${id}/reorder`,
        json("POST", { direction }),
      );
      invalidateAll(true);
    },
    move: async (projectId, id, parentId, confirmConversion) => {
      await request(
        `${path(projectId)}/${id}/move`,
        json("POST", { parentId, confirmConversion }),
      );
      invalidateAll(true);
    },
    remove: async (projectId, id) => {
      await request(`${path(projectId)}/${id}`, {
        method: "DELETE",
      });
      invalidateAll(true);
    },
    updateExecutable: async (projectId, id, input) => {
      const { lagDays, ...fields } = input;
      await request(
        `${path(projectId)}/${id}/executable`,
        json("PUT", { ...fields, lag: lagDays }),
      );
      invalidateAll(true);
    },
    previewExecutableSchedule: async (projectId, id, input, signal) => {
      const requestVersion = currentScheduleProjectionVersion();
      const { lagDays, ...fields } = input;
      const data = await request(
        `${path(projectId)}/${encodeURIComponent(id)}/executable/preview`,
        {
          ...json("POST", { ...fields, lag: lagDays }),
          signal,
        },
      );
      if (currentScheduleProjectionVersion() !== requestVersion) {
        throw staleResponseError();
      }
      return mapSchedulePreview(data, projectId, id);
    },
    complete: async (projectId, id, actualStart, actualEnd) => {
      await request(
        `${path(projectId)}/${id}/actual-date`,
        json("POST", { actualStart, actualEnd }),
      );
      invalidateAll(true);
    },
    reopen: async (projectId, id) => {
      try {
        const response = await schedulingImpactFetch(
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
        invalidateAll(true);
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

function mapAllocationGroups(value: unknown): AllocationGroups {
  if (
    !isRecord(value) ||
    !Array.isArray(value.execution) ||
    !Array.isArray(value.commitment) ||
    !Array.isArray(value.actual)
  ) {
    throw new Error(invalidResponseMessage);
  }
  return {
    execution: value.execution.map(mapAllocationRow),
    commitment: value.commitment.map(mapAllocationRow),
    actual: value.actual.map(mapAllocationRow),
  };
}

function mapAllocationRow(value: unknown): AllocationRow {
  if (
    !isRecord(value) ||
    typeof value.date !== "string" ||
    !isNonNegativeInteger(value.allocatedMinutes) ||
    !isNonNegativeInteger(value.capacityMinutes) ||
    !isNonNegativeInteger(value.remainingMinutes) ||
    !isNonNegativeInteger(value.overcapacityMinutes) ||
    !isNonNegativeInteger(value.capacityAllocationPercentage) ||
    !isNonNegativeInteger(value.taskDailyLimitMinutes)
  ) {
    throw new Error(invalidResponseMessage);
  }
  return {
    date: value.date,
    allocatedMinutes: value.allocatedMinutes,
    capacityMinutes: value.capacityMinutes,
    remainingMinutes: value.remainingMinutes,
    overcapacityMinutes: value.overcapacityMinutes,
    capacityAllocationPercentage: value.capacityAllocationPercentage,
    taskDailyLimitMinutes: value.taskDailyLimitMinutes,
  };
}

function isNonNegativeInteger(value: unknown): value is number {
  return typeof value === "number" && Number.isInteger(value) && value >= 0;
}

function mapSchedulePreview(
  value: unknown,
  projectId: string,
  id: string,
): SchedulePreview {
  if (!isRecord(value) || !("task" in value)) {
    throw new Error(invalidResponseMessage);
  }
  return {
    task: mapPreviewNode(value.task, projectId, id),
  };
}

function mapPreviewNode(
  value: unknown,
  projectId: string,
  id: string,
): WBSNode {
  const preview = mapNode(value);
  if (
    preview.id !== id ||
    preview.projectId !== projectId ||
    preview.hasChildren ||
    preview.children.length !== 0
  ) {
    throw new Error(invalidResponseMessage);
  }
  return preview;
}

function mapReopenedNode(
  value: unknown,
  projectId: string,
  id: string,
): WBSNode {
  if (
    !isRecord(value) ||
    !isRecord(value.executable) ||
    !("actualStart" in value.executable) ||
    !("actualEnd" in value.executable) ||
    value.executable.actualStart !== null ||
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
    lagDays: optionalNonNegativeInteger(value.lag) ?? 0,
    capacityAllocationPercentage:
      optionalPositivePercentage(value.capacityAllocationPercentage) ?? 100,
    executionTimeline: mapTimeline(value.executionTimeline),
    commitmentTimeline: mapTimeline(value.commitmentTimeline),
    executionUnscheduledReason: optionalString(
      value.executionUnscheduledReason,
    ),
    commitmentUnscheduledReason: optionalString(
      value.commitmentUnscheduledReason,
    ),
    actualStart: optionalString(value.actualStart),
    actualEnd: optionalString(value.actualEnd),
  };
}

function optionalPositivePercentage(value: unknown): number | undefined {
  return typeof value === "number" &&
    Number.isInteger(value) &&
    value >= 1 &&
    value <= 100
    ? value
    : undefined;
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

function optionalNonNegativeInteger(value: unknown): number | undefined {
  if (value === undefined || value === null) return undefined;
  if (typeof value !== "number" || !Number.isInteger(value) || value < 0) {
    throw new Error(invalidResponseMessage);
  }
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
    case "INVALID_LAG":
      return "Lag must be a non-negative whole number of days.";
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
      return "The Task could not be found. Refresh Home and try again.";
    case "EXECUTABLE_TASK_REQUIRED":
      return "Only a Task can be reopened.";
    case "TASK_NOT_COMPLETED":
      return "This Task is already unfinished. Refresh Home and try again.";
    case "TASK_REOPEN_CONFLICT":
      return "The Task changed in another request. Refresh and try again.";
    case "TASK_REOPEN_FAILED":
      return "The Task could not be reopened. Try again.";
    case "SCHEDULING_INPUT_INCOMPLETE":
      return "Complete Role, Effort, and valid Lag to preview the schedule.";
    case "SCHEDULE_PREVIEW_UNAVAILABLE":
      return "Schedule preview is available only for an open unfinished Task with Automatic Scheduling on.";
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
