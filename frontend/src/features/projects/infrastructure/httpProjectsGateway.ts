import { RequestCache } from "../../../shared/infrastructure/RequestCache";
import type { PageResult } from "../../../shared/application/pagination";
import {
  ProjectOperationError,
  type ProjectsGateway,
} from "../application/projectsGateway";
import type { Project } from "../domain/project";

interface ProjectDTO {
  id: string;
  name: string;
  status: "open" | "locked" | "closed";
  startDate: string | null;
  endDate: string | null;
  autoCalculateDate: boolean;
  autoDependencyByAssignee: boolean;
  automaticScheduling: boolean;
  schedulingStartDate: string | null;
  projectBuffer: number;
  projectPriority: number;
  closedAt: string | null;
  createdAt: string;
  updatedAt: string;
}
interface ItemDTO {
  data: ProjectDTO;
}
interface ListDTO {
  data: ProjectDTO[];
  page: number;
  pageSize: number;
  total: number;
}
interface ErrorDTO {
  code: string;
  message: string;
  field?: string;
}

export function createHTTPProjectsGateway(apiBaseURL: string): ProjectsGateway {
  const lists = new Map<string, RequestCache<PageResult<Project>>>();
  const invalidate = () => {
    lists.forEach((cache) => cache.invalidate());
    lists.clear();
  };
  const mutation = async (
    path: string,
    method: string,
    body?: object,
  ): Promise<Project> => {
    const response = await fetch(`${apiBaseURL}${path}`, {
      method,
      headers: body ? { "Content-Type": "application/json" } : undefined,
      body: body ? JSON.stringify(body) : undefined,
    });
    const payload = await read<ItemDTO>(response);
    invalidate();
    return mapProject(payload.data);
  };
  return {
    async list(query, signal) {
      const parameters = new URLSearchParams({
        search: query.search,
        page: String(query.page),
        pageSize: String(query.pageSize),
      });
      const key = parameters.toString();
      const cache = lists.get(key) ?? new RequestCache<PageResult<Project>>();
      lists.set(key, cache);
      return cache.run(async () => {
        const payload = await read<ListDTO>(
          await fetch(`${apiBaseURL}/projects?${parameters}`),
        );
        return {
          items: payload.data.map(mapProject),
          page: payload.page,
          pageSize: payload.pageSize,
          total: payload.total,
        };
      }, signal);
    },
    async get(id) {
      return mapProject(
        (
          await read<ItemDTO>(
            await fetch(`${apiBaseURL}/projects/${encodeURIComponent(id)}`),
          )
        ).data,
      );
    },
    create(name, automaticScheduling, schedulingStartDate, projectBuffer) {
      return mutation("/projects", "POST", {
        name,
        automaticScheduling,
        schedulingStartDate: schedulingStartDate ?? null,
        projectBuffer,
      });
    },
    update(id, name, automaticScheduling, schedulingStartDate, projectBuffer) {
      return mutation(`/projects/${encodeURIComponent(id)}`, "PUT", {
        name,
        automaticScheduling,
        schedulingStartDate: schedulingStartDate ?? null,
        projectBuffer,
      });
    },
    changeStatus(id, status) {
      return mutation(`/projects/${encodeURIComponent(id)}/status`, "POST", {
        status,
      });
    },
    movePriority(id, direction) {
      return mutation(`/projects/${encodeURIComponent(id)}/priority`, "POST", {
        direction,
      });
    },
    updateSettings(
      id,
      automaticScheduling,
      schedulingStartDate,
      projectBuffer,
    ) {
      return mutation(`/projects/${encodeURIComponent(id)}/settings`, "PATCH", {
        automaticScheduling,
        schedulingStartDate: schedulingStartDate ?? null,
        projectBuffer,
      });
    },
    async delete(id) {
      const response = await fetch(
        `${apiBaseURL}/projects/${encodeURIComponent(id)}`,
        { method: "DELETE" },
      );
      if (!response.ok) await throwError(response);
      invalidate();
    },
  };
}

async function read<T>(response: Response): Promise<T> {
  if (!response.ok) await throwError(response);
  return (await response.json()) as T;
}
async function throwError(response: Response): Promise<never> {
  try {
    const payload = (await response.json()) as ErrorDTO;
    throw new ProjectOperationError(
      payload.code,
      payload.message,
      payload.field,
    );
  } catch (error: unknown) {
    if (error instanceof ProjectOperationError) throw error;
    throw new ProjectOperationError(
      "UNEXPECTED_RESPONSE",
      "The server returned an unexpected response",
    );
  }
}
function mapProject(dto: ProjectDTO): Project {
  return {
    id: dto.id,
    name: dto.name,
    status: dto.status,
    startDate: dto.startDate ?? undefined,
    endDate: dto.endDate ?? undefined,
    autoCalculateDate: dto.autoCalculateDate,
    autoDependencyByAssignee: dto.autoDependencyByAssignee,
    automaticScheduling: dto.automaticScheduling,
    schedulingStartDate: dto.schedulingStartDate ?? undefined,
    projectBuffer: dto.projectBuffer,
    priority: dto.projectPriority,
    closedAt: dto.closedAt ? new Date(dto.closedAt) : undefined,
    createdAt: new Date(dto.createdAt),
    updatedAt: new Date(dto.updatedAt),
  };
}
