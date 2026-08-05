import type { PageResult } from "../../../shared/application/pagination";
import { RequestCache } from "../../../shared/infrastructure/RequestCache";
import {
  SprintAPIError,
  type SprintsGateway,
} from "../application/sprintsGateway";
import type {
  Sprint,
  SprintDetail,
  SprintSuggestion,
  SprintTaskProjection,
  SprintWriteInput,
} from "../domain/sprint";

interface SprintDTO {
  id: string;
  name: string;
  startDate: string;
  endDate: string;
  status: "planned" | "started";
  version: number;
  memberIds?: string[];
  taskIds?: string[];
  startedAt?: string;
  createdAt: string;
  updatedAt: string;
}
interface ListDTO {
  data: SprintDTO[];
  page: number;
  pageSize: number;
  total: number;
}
interface ItemDTO<T> {
  data: T;
}
type TaskProjectionDTO = Omit<
  SprintTaskProjection,
  "allocations" | "warnings"
> & {
  allocations?: SprintTaskProjection["allocations"] | null;
  warnings?: string[] | null;
};
interface SuggestionDTO extends Omit<SprintSuggestion, "tasks"> {
  tasks: Array<{
    task: TaskProjectionDTO;
    reason: "mandatory" | "capacity_fill";
  }>;
}
interface ErrorDTO {
  code: string;
  message: string;
  field?: string;
  details?: {
    sprintId: string;
    sprintName: string;
    startDate: string;
    endDate: string;
    members: Array<{ id: string; name: string }>;
  };
}

export function createHTTPSprintsGateway(apiBaseURL: string): SprintsGateway {
  const lists = new Map<string, RequestCache<PageResult<Sprint>>>();
  const invalidate = () => {
    lists.forEach((cache) => cache.invalidate());
    lists.clear();
  };
  return {
    list(query, signal) {
      const parameters = new URLSearchParams({
        search: query.search,
        page: String(query.page),
        pageSize: String(query.pageSize),
      });
      const key = parameters.toString();
      const cache = lists.get(key) ?? new RequestCache<PageResult<Sprint>>();
      lists.set(key, cache);
      return cache.run(async () => {
        const payload = await read<ListDTO>(
          await fetch(`${apiBaseURL}/sprints?${parameters}`, { signal }),
        );
        return {
          items: payload.data.map(mapSprint),
          page: payload.page,
          pageSize: payload.pageSize,
          total: payload.total,
        };
      }, signal);
    },
    async detail(id, signal) {
      return (
        await read<ItemDTO<SprintDetail>>(
          await fetch(`${apiBaseURL}/sprints/${encodeURIComponent(id)}`, {
            signal,
          }),
        )
      ).data;
    },
    async create(input) {
      const value = (
        await read<ItemDTO<SprintDTO>>(
          await fetch(`${apiBaseURL}/sprints`, request("POST", input)),
        )
      ).data;
      invalidate();
      return mapSprint(value);
    },
    async update(id, input) {
      const value = (
        await read<ItemDTO<SprintDTO>>(
          await fetch(
            `${apiBaseURL}/sprints/${encodeURIComponent(id)}`,
            request("PUT", input),
          ),
        )
      ).data;
      invalidate();
      return mapSprint(value);
    },
    async suggest(input, signal) {
      const value = (
        await read<ItemDTO<SuggestionDTO>>(
          await fetch(`${apiBaseURL}/sprints/suggestion`, {
            ...request("POST", input),
            signal,
          }),
        )
      ).data;
      return {
        ...value,
        tasks: value.tasks.map((item) => ({
          ...item,
          task: mapTaskProjection(item.task),
        })),
      };
    },
    async candidates(id, page, signal) {
      const parameters = new URLSearchParams({
        page: String(page),
        pageSize: "20",
        search: "",
      });
      return read<PageResultDTO<SprintTaskProjection>>(
        await fetch(
          `${apiBaseURL}/sprints/${encodeURIComponent(id)}/task-candidates?${parameters}`,
          { signal },
        ),
      ).then((payload) => ({
        items: payload.data,
        page: payload.page,
        pageSize: payload.pageSize,
        total: payload.total,
      }));
    },
    async draftCandidates(input, page, signal) {
      const parameters = new URLSearchParams({
        page: String(page),
        pageSize: "20",
        search: "",
      });
      const payload = await read<PageResultDTO<SprintTaskProjection>>(
        await fetch(`${apiBaseURL}/sprints/task-candidates?${parameters}`, {
          ...request("POST", input),
          signal,
        }),
      );
      return {
        items: payload.data,
        page: payload.page,
        pageSize: payload.pageSize,
        total: payload.total,
      };
    },
    async start(id, version) {
      const value = (
        await read<ItemDTO<SprintDTO>>(
          await fetch(`${apiBaseURL}/sprints/${encodeURIComponent(id)}/start`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ version }),
          }),
        )
      ).data;
      invalidate();
      return mapSprint(value);
    },
    async delete(id, version) {
      const response = await fetch(
        `${apiBaseURL}/sprints/${encodeURIComponent(id)}?version=${version}`,
        { method: "DELETE" },
      );
      if (!response.ok) await throwError(response);
      invalidate();
    },
  };
}

interface PageResultDTO<T> {
  data: T[];
  page: number;
  pageSize: number;
  total: number;
}
function request(
  method: string,
  body:
    | SprintWriteInput
    | Pick<SprintWriteInput, "startDate" | "endDate" | "memberIds">
    | (Pick<SprintWriteInput, "startDate" | "endDate" | "memberIds"> & {
        excludedTaskIds: string[];
      }),
): RequestInit {
  return {
    method,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  };
}

function mapSprint(value: SprintDTO): Sprint {
  return {
    ...value,
    memberIds: value.memberIds ?? [],
    taskIds: value.taskIds ?? [],
    startedAt: value.startedAt ? new Date(value.startedAt) : undefined,
    createdAt: new Date(value.createdAt),
    updatedAt: new Date(value.updatedAt),
  };
}
function mapTaskProjection(value: TaskProjectionDTO): SprintTaskProjection {
  return {
    ...value,
    allocations: value.allocations ?? [],
    warnings: value.warnings ?? [],
  };
}
async function read<T>(response: Response): Promise<T> {
  if (!response.ok) await throwError(response);
  return (await response.json()) as T;
}
async function throwError(response: Response): Promise<never> {
  try {
    const payload = (await response.json()) as ErrorDTO;
    throw new SprintAPIError(
      payload.code,
      payload.message,
      payload.field,
      payload.details,
    );
  } catch (error: unknown) {
    if (error instanceof SprintAPIError) throw error;
    throw new SprintAPIError(
      "UNEXPECTED_RESPONSE",
      "The server returned an unexpected response",
    );
  }
}
