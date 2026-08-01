import type {
  PageQuery,
  PageResult,
} from "../../../shared/application/pagination";
import {
  normalizeProjectName,
  validateProjectBuffer,
  type PriorityDirection,
  type Project,
  type ProjectStatus,
} from "../domain/project";
import type { ProjectsGateway } from "./projectsGateway";

export function listProjects(
  gateway: ProjectsGateway,
  query: PageQuery,
  signal?: AbortSignal,
): Promise<PageResult<Project>> {
  return gateway.list(query, signal);
}
export function createProject(
  gateway: ProjectsGateway,
  name: string,
  automaticScheduling: boolean,
  schedulingStartDate: string | undefined,
  projectBuffer: number,
): Promise<Project> {
  return gateway.create(
    normalizeProjectName(name),
    automaticScheduling,
    schedulingStartDate,
    validateProjectBuffer(projectBuffer),
  );
}
export function updateProject(
  gateway: ProjectsGateway,
  id: string,
  name: string,
  automaticScheduling: boolean,
  schedulingStartDate: string | undefined,
  projectBuffer: number,
): Promise<Project> {
  return gateway.update(
    id,
    normalizeProjectName(name),
    automaticScheduling,
    schedulingStartDate,
    validateProjectBuffer(projectBuffer),
  );
}
export function renameProject(
  gateway: ProjectsGateway,
  id: string,
  name: string,
): Promise<Project> {
  return gateway.rename(id, normalizeProjectName(name));
}
export function changeProjectStatus(
  gateway: ProjectsGateway,
  id: string,
  status: ProjectStatus,
): Promise<Project> {
  return gateway.changeStatus(id, status);
}
export function moveProjectPriority(
  gateway: ProjectsGateway,
  id: string,
  direction: PriorityDirection,
): Promise<Project> {
  return gateway.movePriority(id, direction);
}
export function deleteProject(
  gateway: ProjectsGateway,
  id: string,
): Promise<void> {
  return gateway.delete(id);
}
export function updateProjectSettings(
  gateway: ProjectsGateway,
  id: string,
  automaticScheduling: boolean,
  schedulingStartDate: string | undefined,
  projectBuffer: number,
): Promise<Project> {
  return gateway.updateSettings(
    id,
    automaticScheduling,
    schedulingStartDate,
    validateProjectBuffer(projectBuffer),
  );
}
