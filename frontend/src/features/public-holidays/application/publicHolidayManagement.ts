import type { PublicHolidayInput } from "../domain/publicHoliday";
import { validatePublicHoliday } from "../domain/publicHoliday";
import type { PublicHolidaysGateway } from "./publicHolidaysGateway";
export const listPublicHolidays = (
  gateway: PublicHolidaysGateway,
  page: number,
  holidayDate: string,
  signal?: AbortSignal,
) =>
  gateway.list(
    { page, pageSize: 5, holidayDate: holidayDate || undefined },
    signal,
  );
export const createPublicHoliday = (
  gateway: PublicHolidaysGateway,
  input: PublicHolidayInput,
) => gateway.create(validatePublicHoliday(input));
export const updatePublicHoliday = (
  gateway: PublicHolidaysGateway,
  id: string,
  input: PublicHolidayInput,
) => gateway.update(id, validatePublicHoliday(input));
export const deletePublicHoliday = (
  gateway: PublicHolidaysGateway,
  id: string,
) => gateway.delete(id);
