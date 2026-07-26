import type {
  PageQuery,
  PageResult,
} from "../../../shared/application/pagination";
import type {
  CapacityOverride,
  CapacityOverrideInput,
} from "../domain/capacityOverride";
export interface CapacityOverrideListQuery extends Omit<PageQuery, "search"> {
  effectiveDate?: string;
}
export interface CapacityOverridesGateway {
  list(
    memberId: string,
    query: CapacityOverrideListQuery,
    signal?: AbortSignal,
  ): Promise<PageResult<CapacityOverride>>;
  get(memberId: string, id: string): Promise<CapacityOverride>;
  create(
    memberId: string,
    input: Required<CapacityOverrideInput>,
  ): Promise<CapacityOverride>;
  update(
    memberId: string,
    id: string,
    input: Required<CapacityOverrideInput>,
  ): Promise<CapacityOverride>;
  delete(memberId: string, id: string): Promise<void>;
}
