import type {
  PortfolioProject,
  PortfolioProjection,
  PortfolioProjectionResult,
  SavedPortfolioFilter,
} from "../domain/portfolio";

export interface PortfolioGateway {
  activeProjects(signal?: AbortSignal): Promise<PortfolioProject[]>;
  projection(
    projectIds: string[],
    projection: PortfolioProjection,
    from: string,
    to: string,
    signal?: AbortSignal,
  ): Promise<PortfolioProjectionResult>;
  savedFilters(signal?: AbortSignal): Promise<SavedPortfolioFilter[]>;
  createSavedFilter(
    name: string,
    projectIds: string[],
  ): Promise<SavedPortfolioFilter>;
  updateSavedFilter(
    id: string,
    version: number,
    projectIds: string[],
  ): Promise<SavedPortfolioFilter>;
  deleteSavedFilter(id: string, version: number): Promise<void>;
}
