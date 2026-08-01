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
  create(
    name: string,
    automaticScheduling: boolean,
    schedulingStartDate: string | undefined,
    projectBuffer: number,
  ): Promise<Project>;
  rename(id: string, name: string): Promise<Project>;
  update(
    id: string,
    name: string,
    automaticScheduling: boolean,
    schedulingStartDate: string | undefined,
    projectBuffer: number,
  ): Promise<Project>;
  changeStatus(id: string, status: ProjectStatus): Promise<Project>;
  bulkReopen(rootProjectId: string, token: string): Promise<Project[]>;
  movePriority(id: string, direction: PriorityDirection): Promise<Project>;
  updateSettings(
    id: string,
    automaticScheduling: boolean,
    schedulingStartDate: string | undefined,
    projectBuffer: number,
  ): Promise<Project>;
  delete(id: string): Promise<void>;
}

export interface ReopenImpactProject {
  id: string;
  name: string;
  version: number;
}

export interface BulkReopenPlan {
  rootProjectId: string;
  lockedProjects: ReopenImpactProject[];
  openProjects: ReopenImpactProject[];
  token: string;
}

export class ProjectOperationError extends Error {
  constructor(
    readonly code: string,
    message: string,
    readonly field?: string,
    readonly bulkReopenPlan?: BulkReopenPlan,
  ) {
    super(message);
  }
}
