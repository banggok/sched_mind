import { RequestCache } from "../../../shared/infrastructure/RequestCache";
import type { PublicHolidaysGateway } from "../application/publicHolidaysGateway";
import type { PublicHoliday } from "../domain/publicHoliday";
interface DTO {
  id: string;
  startDate: string;
  endDate: string;
  description: string;
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
export class PublicHolidaysAPIError extends Error {
  constructor(
    readonly code: string,
    message: string,
    readonly field?: string,
  ) {
    super(message);
  }
}
export function createHTTPPublicHolidaysGateway(
  base: string,
): PublicHolidaysGateway {
  const caches = new Map<
    string,
    RequestCache<{
      items: PublicHoliday[];
      page: number;
      pageSize: number;
      total: number;
    }>
  >();
  const calendarCaches = new Map<string, RequestCache<string[]>>();
  const invalidate = () => {
    caches.forEach((cache) => cache.invalidate());
    caches.clear();
    calendarCaches.forEach((cache) => cache.invalidate());
    calendarCaches.clear();
  };
  const path = `${base}/public-holidays`;
  return {
    async list(query, signal) {
      const parameters = new URLSearchParams({
        page: String(query.page),
        pageSize: String(query.pageSize),
      });
      if (query.holidayDate) parameters.set("holidayDate", query.holidayDate);
      const key = parameters.toString();
      const cache = caches.get(key) ?? new RequestCache();
      caches.set(key, cache);
      return cache.run(async () => {
        const payload = await read<ListDTO>(
          await fetch(`${path}?${parameters}`),
        );
        return { ...payload, items: payload.data.map(map) };
      }, signal);
    },
    async get(id) {
      return map(
        (await read<ItemDTO>(await fetch(`${path}/${encodeURIComponent(id)}`)))
          .data,
      );
    },
    async create(input) {
      const value = map(
        (
          await read<ItemDTO>(
            await fetch(path, {
              method: "POST",
              headers: { "Content-Type": "application/json" },
              body: JSON.stringify(input),
            }),
          )
        ).data,
      );
      invalidate();
      return value;
    },
    async update(id, input) {
      const value = map(
        (
          await read<ItemDTO>(
            await fetch(`${path}/${encodeURIComponent(id)}`, {
              method: "PUT",
              headers: { "Content-Type": "application/json" },
              body: JSON.stringify(input),
            }),
          )
        ).data,
      );
      invalidate();
      return value;
    },
    async delete(id) {
      const response = await fetch(`${path}/${encodeURIComponent(id)}`, {
        method: "DELETE",
      });
      if (!response.ok) await fail(response);
      invalidate();
    },
    async calendar(startDate, endDate) {
      const parameters = new URLSearchParams({ startDate, endDate });
      const key = parameters.toString();
      const cache = calendarCaches.get(key) ?? new RequestCache<string[]>();
      calendarCaches.set(key, cache);
      return cache.run(async () => {
        const payload = await read<{ data: string[] }>(
          await fetch(`${path}/calendar?${parameters}`),
        );
        return payload.data;
      });
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
    throw new PublicHolidaysAPIError(value.code, value.message, value.field);
  } catch (error) {
    if (error instanceof PublicHolidaysAPIError) throw error;
    throw new PublicHolidaysAPIError(
      "UNEXPECTED_RESPONSE",
      "The server returned an unexpected response",
    );
  }
}
function map(value: DTO): PublicHoliday {
  return {
    ...value,
    createdAt: new Date(value.createdAt),
    updatedAt: new Date(value.updatedAt),
  };
}
