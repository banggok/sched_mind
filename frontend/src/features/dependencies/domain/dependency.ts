export type DependencyDirection = "blockedBy" | "blocks";
export type DependencySource = "manual" | "automatic" | "both";

export interface DependencyTask {
  id: string;
  name: string;
  projectId: string;
  projectName: string;
  hierarchyPath: string;
  completed: boolean;
  expectedStart?: string;
}

export interface DependencyRelation {
  id: string;
  source: DependencySource;
  manualRemovable: boolean;
  task: DependencyTask;
}

export interface DependencyDetail {
  blockedBy: DependencyRelation[];
  blocks: DependencyRelation[];
}

export interface DependencyCandidatePage {
  items: DependencyTask[];
  page: number;
  pageSize: number;
  totalItems: number;
}
