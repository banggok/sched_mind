import { schedulingImpactFetch } from "../../../shared/infrastructure/schedulingImpactFetch";
import { advanceScheduleProjectionVersion } from "../../../shared/infrastructure/scheduleProjectionClock";
import type { CapacityOverridesGateway } from "../application/capacityOverridesGateway";
import type { CapacityOverride } from "../domain/capacityOverride";
import { RequestCache } from "../../../shared/infrastructure/RequestCache";
interface DTO {
  id: string;
  teamMemberId: string;
  description: string;
  startDate: string;
  endDate: string;
  capacity: number;
  createdAt: string;
  updatedAt: string;
}
interface ListDTO {
  data: DTO[];
  page: number;
  pageSize: number;
  total: number;
}
interface ItemDTO {
  data: DTO;
}
interface ErrorDTO {
  code: string;
  message: string;
  field?: string;
}
export class CapacityOverridesAPIError extends Error {
  constructor(
    readonly code: string,
    message: string,
    readonly field?: string,
  ) {
    super(message);
  }
}
export function createHTTPCapacityOverridesGateway(
  base: string,
): CapacityOverridesGateway {
  const caches = new Map<
    string,
    RequestCache<{
      items: CapacityOverride[];
      page: number;
      pageSize: number;
      total: number;
    }>
  >();
  const invalidate = () => {
    caches.forEach((c) => c.invalidate());
    caches.clear();
  };
  const path = (memberId: string) =>
    `${base}/team-members/${encodeURIComponent(memberId)}/capacity-overrides`;
  return {
    async list(memberId, query, signal) {
      const parameters = new URLSearchParams({
        page: String(query.page),
        pageSize: String(query.pageSize),
      });
      if (query.effectiveDate)
        parameters.set("effectiveDate", query.effectiveDate);
      const key = `${memberId}:${parameters}`;
      const cache = caches.get(key) ?? new RequestCache();
      caches.set(key, cache);
      return cache.run(async () => {
        const response = await schedulingImpactFetch(
          `${path(memberId)}?${parameters}`,
        );
        const payload = await read<ListDTO>(response);
        return {
          items: payload.data.map(map),
          page: payload.page,
          pageSize: payload.pageSize,
          total: payload.total,
        };
      }, signal);
    },
    async get(memberId, id) {
      return map(
        (
          await read<ItemDTO>(
            await schedulingImpactFetch(
              `${path(memberId)}/${encodeURIComponent(id)}`,
            ),
          )
        ).data,
      );
    },
    async create(memberId, input) {
      const value = map(
        (
          await read<ItemDTO>(
            await schedulingImpactFetch(path(memberId), {
              method: "POST",
              headers: { "Content-Type": "application/json" },
              body: JSON.stringify(input),
            }),
          )
        ).data,
      );
      invalidate();
      advanceScheduleProjectionVersion();
      return value;
    },
    async update(memberId, id, input) {
      const value = map(
        (
          await read<ItemDTO>(
            await schedulingImpactFetch(
              `${path(memberId)}/${encodeURIComponent(id)}`,
              {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(input),
              },
            ),
          )
        ).data,
      );
      invalidate();
      advanceScheduleProjectionVersion();
      return value;
    },
    async delete(memberId, id) {
      const response = await schedulingImpactFetch(
        `${path(memberId)}/${encodeURIComponent(id)}`,
        { method: "DELETE" },
      );
      if (!response.ok) await fail(response);
      invalidate();
      advanceScheduleProjectionVersion();
    },
  };
}
async function read<T>(response: Response): Promise<T> {
  if (!response.ok) await fail(response);
  return response.json() as Promise<T>;
}
async function fail(response: Response): Promise<never> {
  try {
    const value = (await response.json()) as ErrorDTO;
    throw new CapacityOverridesAPIError(value.code, value.message, value.field);
  } catch (error) {
    if (error instanceof CapacityOverridesAPIError) throw error;
    throw new CapacityOverridesAPIError(
      "UNEXPECTED_RESPONSE",
      "The server returned an unexpected response",
    );
  }
}
function map(value: DTO): CapacityOverride {
  return {
    ...value,
    createdAt: new Date(value.createdAt),
    updatedAt: new Date(value.updatedAt),
  };
}
