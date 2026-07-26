import type {
  PageQuery,
  PageResult,
} from "../../../shared/application/pagination";
import type {
  PriorityDirection,
  Project,
  ProjectStatus,
} from "../domain/project";

export interface ProjectsGateway {
  list(query: PageQuery, signal?: AbortSignal): Promise<PageResult<Project>>;
  get(id: string): Promise<Project>;
  create(name: string): Promise<Project>;
  update(id: string, name: string): Promise<Project>;
  changeStatus(id: string, status: ProjectStatus): Promise<Project>;
  movePriority(id: string, direction: PriorityDirection): Promise<Project>;
  delete(id: string): Promise<void>;
}

export class ProjectOperationError extends Error {
  constructor(
    readonly code: string,
    message: string,
    readonly field?: string,
  ) {
    super(message);
  }
}
