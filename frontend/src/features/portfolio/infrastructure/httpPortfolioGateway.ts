import {
  currentScheduleProjectionVersion,
  subscribeScheduleProjectionVersion,
} from "../../../shared/infrastructure/scheduleProjectionClock";
import type { PortfolioGateway } from "../application/portfolioGateway";
import {
  PortfolioOperationError,
  type PortfolioDependency,
  type PortfolioHoliday,
  type PortfolioProject,
  type PortfolioProjection,
  type PortfolioProjectionResult,
  type PortfolioRow,
  type SavedPortfolioFilter,
} from "../domain/portfolio";

export function createHTTPPortfolioGateway(
  apiBaseURL: string,
): PortfolioGateway {
  let projectionRequest = 0;
  let metadataGeneration = 0;
  let savedFiltersGeneration = 0;
  let activeProjectsPending: Promise<PortfolioProject[]> | undefined;
  let savedFiltersPending: Promise<SavedPortfolioFilter[]> | undefined;
  let observedProjectionVersion = currentScheduleProjectionVersion();
  subscribeScheduleProjectionVersion(() => {
    observedProjectionVersion = currentScheduleProjectionVersion();
    projectionRequest += 1;
    metadataGeneration += 1;
    activeProjectsPending = undefined;
  });

  function loadActiveProjects(): Promise<PortfolioProject[]> {
    if (activeProjectsPending) return activeProjectsPending;
    const generation = metadataGeneration;
    const request = fetch(`${apiBaseURL}/portfolio/projects`)
      .then(readJSON)
      .then(envelopeData)
      .then((data) => {
        if (generation !== metadataGeneration) throw staleResponse();
        if (!Array.isArray(data)) throw unexpectedResponse();
        return data.map(readProject);
      });
    activeProjectsPending = request;
    request.then(
      () => {
        if (activeProjectsPending === request)
          activeProjectsPending = undefined;
      },
      () => {
        if (activeProjectsPending === request)
          activeProjectsPending = undefined;
      },
    );
    return request;
  }

  function loadSavedFilters(): Promise<SavedPortfolioFilter[]> {
    if (savedFiltersPending) return savedFiltersPending;
    const generation = savedFiltersGeneration;
    const request = fetch(`${apiBaseURL}/portfolio/saved-filters`)
      .then(readJSON)
      .then(envelopeData)
      .then((data) => {
        if (generation !== savedFiltersGeneration) throw staleResponse();
        if (!Array.isArray(data)) throw unexpectedResponse();
        return data.map(readSavedFilter);
      });
    savedFiltersPending = request;
    request.then(
      () => {
        if (savedFiltersPending === request) savedFiltersPending = undefined;
      },
      () => {
        if (savedFiltersPending === request) savedFiltersPending = undefined;
      },
    );
    return request;
  }

  function invalidateSavedFilters() {
    savedFiltersGeneration += 1;
    savedFiltersPending = undefined;
  }

  return {
    activeProjects(signal) {
      return observeRequest(loadActiveProjects(), signal);
    },
    async projection(projectIds, projection, from, to, signal) {
      const identity = ++projectionRequest;
      const version = currentScheduleProjectionVersion();
      observedProjectionVersion = version;
      const query = new URLSearchParams({ projection, from, to });
      projectIds.forEach((id) => query.append("projectId", id));
      const data = envelopeData(
        await readJSON(
          await fetch(`${apiBaseURL}/portfolio?${query}`, { signal }),
        ),
      );
      if (
        identity !== projectionRequest ||
        version !== currentScheduleProjectionVersion() ||
        observedProjectionVersion !== version
      )
        throw staleResponse();
      return readProjection(data);
    },
    savedFilters(signal) {
      return observeRequest(loadSavedFilters(), signal);
    },
    async createSavedFilter(name, projectIds) {
      const data = envelopeData(
        await readJSON(
          await fetch(`${apiBaseURL}/portfolio/saved-filters`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ name, projectIds }),
          }),
        ),
      );
      const result = readSavedFilter(data);
      invalidateSavedFilters();
      return result;
    },
    async updateSavedFilter(id, version, projectIds) {
      const data = envelopeData(
        await readJSON(
          await fetch(
            `${apiBaseURL}/portfolio/saved-filters/${encodeURIComponent(id)}`,
            {
              method: "PUT",
              headers: { "Content-Type": "application/json" },
              body: JSON.stringify({ version, projectIds }),
            },
          ),
        ),
      );
      const result = readSavedFilter(data);
      invalidateSavedFilters();
      return result;
    },
    async deleteSavedFilter(id, version) {
      const query = new URLSearchParams({ version: String(version) });
      const response = await fetch(
        `${apiBaseURL}/portfolio/saved-filters/${encodeURIComponent(id)}?${query}`,
        { method: "DELETE" },
      );
      if (!response.ok) await throwError(response);
      invalidateSavedFilters();
    },
  };
}

function observeRequest<T>(
  request: Promise<T>,
  signal?: AbortSignal,
): Promise<T> {
  if (!signal) return request;
  if (signal.aborted) return Promise.reject(staleResponse());
  return new Promise<T>((resolve, reject) => {
    const onAbort = () => {
      signal.removeEventListener("abort", onAbort);
      reject(staleResponse());
    };
    signal.addEventListener("abort", onAbort, { once: true });
    request.then(
      (value) => {
        signal.removeEventListener("abort", onAbort);
        if (signal.aborted) reject(staleResponse());
        else resolve(value);
      },
      (reason: unknown) => {
        signal.removeEventListener("abort", onAbort);
        reject(reason);
      },
    );
  });
}

function envelopeData(value: unknown): unknown {
  if (!isRecord(value) || !("data" in value)) throw unexpectedResponse();
  return value.data;
}

function readProject(value: unknown): PortfolioProject {
  if (
    !isRecord(value) ||
    typeof value.id !== "string" ||
    typeof value.name !== "string" ||
    !isProjectStatus(value.status) ||
    typeof value.priority !== "number" ||
    typeof value.scheduleVersion !== "number"
  )
    throw unexpectedResponse();
  return {
    id: value.id,
    name: value.name,
    status: value.status,
    priority: value.priority,
    scheduleVersion: value.scheduleVersion,
  };
}

function readProjection(value: unknown): PortfolioProjectionResult {
  if (
    !isRecord(value) ||
    !isProjection(value.projection) ||
    !Array.isArray(value.projects) ||
    !Array.isArray(value.rows) ||
    !Array.isArray(value.dependencies) ||
    !Array.isArray(value.holidays) ||
    !isOptionalString(value.workingDayAnchor)
  )
    throw unexpectedResponse();
  return {
    projection: value.projection,
    projects: value.projects.map(readProject),
    rows: value.rows.map(readRow),
    dependencies: value.dependencies.map(readDependency),
    holidays: value.holidays.map(readHoliday),
    workingDayAnchor: value.workingDayAnchor,
  };
}

function readRow(value: unknown): PortfolioRow {
  if (
    !isRecord(value) ||
    typeof value.id !== "string" ||
    typeof value.projectId !== "string" ||
    !isOptionalString(value.parentId) ||
    !isRowKind(value.kind) ||
    typeof value.name !== "string" ||
    typeof value.wbsNumber !== "string" ||
    typeof value.depth !== "number" ||
    typeof value.position !== "number" ||
    !(value.status === undefined || isProjectStatus(value.status)) ||
    !isOptionalString(value.roleId) ||
    !isOptionalString(value.roleName) ||
    !isOptionalString(value.assigneeId) ||
    !isOptionalString(value.assigneeName) ||
    !isOptionalNumber(value.effortMinutes) ||
    !isOptionalString(value.start) ||
    !isOptionalString(value.end) ||
    !isOptionalString(value.unscheduledReason) ||
    typeof value.incompleteEffort !== "boolean" ||
    typeof value.incompleteSchedule !== "boolean" ||
    typeof value.hasChildren !== "boolean" ||
    typeof value.completed !== "boolean" ||
    !isEffectiveLifecycle(value.effectiveLifecycle)
  )
    throw unexpectedResponse();
  return {
    id: value.id,
    projectId: value.projectId,
    parentId: value.parentId,
    kind: value.kind,
    name: value.name,
    wbsNumber: value.wbsNumber,
    depth: value.depth,
    position: value.position,
    status: value.status,
    roleId: value.roleId,
    roleName: value.roleName,
    assigneeId: value.assigneeId,
    assigneeName: value.assigneeName,
    effortMinutes: value.effortMinutes,
    start: value.start,
    end: value.end,
    unscheduledReason: value.unscheduledReason,
    incompleteEffort: value.incompleteEffort,
    incompleteSchedule: value.incompleteSchedule,
    hasChildren: value.hasChildren,
    completed: value.completed,
    effectiveLifecycle: value.effectiveLifecycle,
  };
}

function isEffectiveLifecycle(
  value: unknown,
): value is "open" | "locked" | "closed" {
  return value === "open" || value === "locked" || value === "closed";
}

function readDependency(value: unknown): PortfolioDependency {
  if (
    !isRecord(value) ||
    typeof value.id !== "string" ||
    typeof value.blockingTaskId !== "string" ||
    typeof value.blockedTaskId !== "string"
  )
    throw unexpectedResponse();
  return {
    id: value.id,
    blockingTaskId: value.blockingTaskId,
    blockedTaskId: value.blockedTaskId,
  };
}

function readHoliday(value: unknown): PortfolioHoliday {
  if (
    !isRecord(value) ||
    typeof value.date !== "string" ||
    typeof value.description !== "string"
  )
    throw unexpectedResponse();
  return { date: value.date, description: value.description };
}

function readSavedFilter(value: unknown): SavedPortfolioFilter {
  if (
    !isRecord(value) ||
    typeof value.id !== "string" ||
    typeof value.name !== "string" ||
    !Array.isArray(value.projectIds) ||
    !value.projectIds.every((id) => typeof id === "string") ||
    typeof value.version !== "number" ||
    typeof value.createdAt !== "string" ||
    typeof value.updatedAt !== "string"
  )
    throw unexpectedResponse();
  const createdAt = new Date(value.createdAt);
  const updatedAt = new Date(value.updatedAt);
  if (Number.isNaN(createdAt.getTime()) || Number.isNaN(updatedAt.getTime()))
    throw unexpectedResponse();
  return {
    id: value.id,
    name: value.name,
    projectIds: [...value.projectIds],
    version: value.version,
    createdAt,
    updatedAt,
  };
}

function staleResponse(): DOMException {
  return new DOMException("Stale portfolio response", "AbortError");
}

async function readJSON(response: Response): Promise<unknown> {
  if (!response.ok) await throwError(response);
  try {
    return await response.json();
  } catch {
    throw unexpectedResponse();
  }
}

async function throwError(response: Response): Promise<never> {
  try {
    const payload: unknown = await response.json();
    if (
      isRecord(payload) &&
      typeof payload.code === "string" &&
      typeof payload.message === "string" &&
      isOptionalString(payload.field)
    )
      throw new PortfolioOperationError(
        payload.code,
        payload.message,
        payload.field,
      );
  } catch (error: unknown) {
    if (error instanceof PortfolioOperationError) throw error;
  }
  throw unexpectedResponse();
}

function unexpectedResponse(): PortfolioOperationError {
  return new PortfolioOperationError(
    "UNEXPECTED_RESPONSE",
    "The server returned an unexpected response",
  );
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function isOptionalString(value: unknown): value is string | undefined {
  return value === undefined || typeof value === "string";
}

function isOptionalNumber(value: unknown): value is number | undefined {
  return value === undefined || typeof value === "number";
}

function isProjectStatus(value: unknown): value is PortfolioProject["status"] {
  return value === "open" || value === "locked";
}

function isProjection(value: unknown): value is PortfolioProjection {
  return value === "execution" || value === "commitment";
}

function isRowKind(value: unknown): value is PortfolioRow["kind"] {
  return value === "project" || value === "group" || value === "task";
}
