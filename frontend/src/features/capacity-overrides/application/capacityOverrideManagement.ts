import type { CapacityOverridesGateway } from "./capacityOverridesGateway";
import {
  validateCapacityOverride,
  type CapacityOverrideInput,
} from "../domain/capacityOverride";
export const listCapacityOverrides = (
  gateway: CapacityOverridesGateway,
  memberId: string,
  page: number,
  effectiveDate: string,
  signal?: AbortSignal,
) =>
  gateway.list(
    memberId,
    { page, pageSize: 5, effectiveDate: effectiveDate || undefined },
    signal,
  );
export const createCapacityOverride = (
  gateway: CapacityOverridesGateway,
  memberId: string,
  input: CapacityOverrideInput,
) => gateway.create(memberId, validateCapacityOverride(input));
export const updateCapacityOverride = (
  gateway: CapacityOverridesGateway,
  memberId: string,
  id: string,
  input: CapacityOverrideInput,
) => gateway.update(memberId, id, validateCapacityOverride(input));
export const deleteCapacityOverride = (
  gateway: CapacityOverridesGateway,
  memberId: string,
  id: string,
) => gateway.delete(memberId, id);
