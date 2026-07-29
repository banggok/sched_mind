import type {
  DependencyCandidatePage,
  DependencyDetail,
  DependencyDirection,
} from "../domain/dependency";

export class DependencyOperationError extends Error {
  constructor(
    readonly code: string,
    message: string,
    readonly cyclePath: string[] = [],
  ) {
    super(message);
  }
}

export interface DependenciesGateway {
  list(taskId: string, signal?: AbortSignal): Promise<DependencyDetail>;
  candidates(
    taskId: string,
    direction: DependencyDirection,
    search: string,
    page: number,
    pageSize: number,
    signal?: AbortSignal,
  ): Promise<DependencyCandidatePage>;
  create(blockingTaskId: string, blockedTaskId: string): Promise<void>;
  remove(dependencyId: string): Promise<void>;
  invalidateTask(taskId: string): void;
  invalidateAll(): void;
}
