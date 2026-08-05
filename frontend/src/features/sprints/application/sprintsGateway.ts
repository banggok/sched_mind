import type {
  PageQuery,
  PageResult,
} from "../../../shared/application/pagination";
import type {
  Sprint,
  SprintDetail,
  SprintSuggestion,
  SprintTaskProjection,
  SprintWriteInput,
} from "../domain/sprint";

export interface SprintsGateway {
  list(query: PageQuery, signal?: AbortSignal): Promise<PageResult<Sprint>>;
  detail(id: string, signal?: AbortSignal): Promise<SprintDetail>;
  create(input: SprintWriteInput): Promise<Sprint>;
  update(id: string, input: Required<SprintWriteInput>): Promise<Sprint>;
  suggest(
    input: Pick<SprintWriteInput, "startDate" | "endDate" | "memberIds">,
    signal?: AbortSignal,
  ): Promise<SprintSuggestion>;
  candidates(
    id: string,
    page: number,
    signal?: AbortSignal,
  ): Promise<PageResult<SprintTaskProjection>>;
  draftCandidates(
    input: Pick<SprintWriteInput, "startDate" | "endDate" | "memberIds"> & {
      excludedTaskIds: string[];
    },
    page: number,
    signal?: AbortSignal,
  ): Promise<PageResult<SprintTaskProjection>>;
  start(id: string, version: number): Promise<Sprint>;
  delete(id: string, version: number): Promise<void>;
}

export interface SprintOverlapDetails {
  sprintId: string;
  sprintName: string;
  startDate: string;
  endDate: string;
  members: Array<{ id: string; name: string }>;
}

export class SprintAPIError extends Error {
  constructor(
    readonly code: string,
    message: string,
    readonly field?: string,
    readonly details?: SprintOverlapDetails,
  ) {
    super(message);
  }
}
