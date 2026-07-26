import type { PageResult } from "../../../shared/application/pagination";
import type {
  PublicHoliday,
  PublicHolidayInput,
} from "../domain/publicHoliday";
export interface PublicHolidayListQuery {
  page: number;
  pageSize: number;
  holidayDate?: string;
}
export interface PublicHolidaysGateway {
  list(
    query: PublicHolidayListQuery,
    signal?: AbortSignal,
  ): Promise<PageResult<PublicHoliday>>;
  get(id: string): Promise<PublicHoliday>;
  create(input: PublicHolidayInput): Promise<PublicHoliday>;
  update(id: string, input: PublicHolidayInput): Promise<PublicHoliday>;
  delete(id: string): Promise<void>;
  calendar(startDate: string, endDate: string): Promise<string[]>;
}
