export interface PublicHoliday {
  id: string;
  startDate: string;
  endDate: string;
  description: string;
  createdAt: Date;
  updatedAt: Date;
}
export interface PublicHolidayInput {
  startDate: string;
  endDate: string;
  description: string;
}
export type PublicHolidayField = "startDate" | "endDate" | "description";
export class PublicHolidayValidationError extends Error {
  constructor(
    readonly field: PublicHolidayField,
    readonly code: string,
    message: string,
  ) {
    super(message);
  }
}
export function validatePublicHoliday(
  input: PublicHolidayInput,
): PublicHolidayInput {
  if (!input.startDate)
    throw new PublicHolidayValidationError(
      "startDate",
      "PUBLIC_HOLIDAY_START_DATE_REQUIRED",
      "Start date is required",
    );
  if (!isDateOnly(input.startDate))
    throw new PublicHolidayValidationError(
      "startDate",
      "INVALID_START_DATE",
      "Start date must be valid",
    );
  if (!input.endDate || !isDateOnly(input.endDate))
    throw new PublicHolidayValidationError(
      "endDate",
      "INVALID_END_DATE",
      "End date is required and must be valid",
    );
  if (input.endDate < input.startDate)
    throw new PublicHolidayValidationError(
      "endDate",
      "INVALID_DATE_RANGE",
      "End date must be on or after start date",
    );
  if (!hasWeekday(input.startDate, input.endDate))
    throw new PublicHolidayValidationError(
      "startDate",
      "NO_WORKING_DATES",
      "Select a range containing at least one weekday",
    );
  const description = input.description.trim();
  if (!description)
    throw new PublicHolidayValidationError(
      "description",
      "PUBLIC_HOLIDAY_DESCRIPTION_REQUIRED",
      "Description is required",
    );
  if ([...description].length > 100)
    throw new PublicHolidayValidationError(
      "description",
      "PUBLIC_HOLIDAY_DESCRIPTION_TOO_LONG",
      "Description must not exceed 100 characters",
    );
  return { startDate: input.startDate, endDate: input.endDate, description };
}
function hasWeekday(start: string, end: string) {
  const date = new Date(`${start}T00:00:00Z`);
  const last = new Date(`${end}T00:00:00Z`);
  while (date <= last) {
    if (date.getUTCDay() !== 0 && date.getUTCDay() !== 6) return true;
    date.setUTCDate(date.getUTCDate() + 1);
  }
  return false;
}
function isDateOnly(value: string) {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) return false;
  const [year, month, day] = value.split("-").map(Number);
  const date = new Date(Date.UTC(year, month - 1, day));
  return (
    date.getUTCFullYear() === year &&
    date.getUTCMonth() === month - 1 &&
    date.getUTCDate() === day
  );
}
