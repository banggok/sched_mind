import type { WBSNode } from "../domain/wbs";
export class WBSOperationError extends Error {
  constructor(
    readonly code: string,
    message: string,
  ) {
    super(message);
  }
}
export interface WBSGateway {
  tree(projectId: string, signal?: AbortSignal): Promise<WBSNode[]>;
  create(
    projectId: string,
    parentId: string | undefined,
    name: string,
    confirmConversion: boolean,
  ): Promise<void>;
  rename(projectId: string, id: string, name: string): Promise<void>;
  reorder(
    projectId: string,
    id: string,
    direction: "up" | "down",
  ): Promise<void>;
  move(
    projectId: string,
    id: string,
    parentId: string | undefined,
    confirmConversion: boolean,
  ): Promise<void>;
  remove(projectId: string, id: string): Promise<void>;
  updateExecutable(
    projectId: string,
    id: string,
    input: ExecutableInput,
  ): Promise<void>;
  complete(projectId: string, id: string, actualEnd: string): Promise<void>;
  reopen(projectId: string, id: string): Promise<WBSNode>;
}
export interface ExecutableInput {
  name: string;
  roleId?: string;
  assigneeId?: string;
  effortHours?: number;
  executionStart?: string;
  executionEnd?: string;
  commitmentStart?: string;
  commitmentEnd?: string;
}
