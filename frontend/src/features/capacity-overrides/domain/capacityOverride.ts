export interface CapacityOverride {
  id: string;
  teamMemberId: string;
  description: string;
  startDate: string;
  endDate: string;
  capacity: number;
  createdAt: Date;
  updatedAt: Date;
}
export interface CapacityOverrideInput {
  description: string;
  startDate: string;
  endDate: string;
  capacity: number | undefined;
}
export type CapacityOverrideField =
  "description" | "startDate" | "endDate" | "capacity";
export class CapacityOverrideValidationError extends Error {
  constructor(
    readonly field: CapacityOverrideField,
    readonly code: string,
    message: string,
  ) {
    super(message);
  }
}
export function validateCapacityOverride(
  input: CapacityOverrideInput,
): Required<CapacityOverrideInput> {
  const description = input.description.trim();
  if (!description)
    throw new CapacityOverrideValidationError(
      "description",
      "CAPACITY_OVERRIDE_DESCRIPTION_REQUIRED",
      "Description is required",
    );
  if ([...description].length > 100)
    throw new CapacityOverrideValidationError(
      "description",
      "CAPACITY_OVERRIDE_DESCRIPTION_TOO_LONG",
      "Description must not exceed 100 characters",
    );
  if (!input.startDate)
    throw new CapacityOverrideValidationError(
      "startDate",
      "CAPACITY_OVERRIDE_START_DATE_REQUIRED",
      "Start date is required",
    );
  if (!isDateOnly(input.startDate))
    throw new CapacityOverrideValidationError(
      "startDate",
      "CAPACITY_OVERRIDE_INVALID_DATE",
      "Start date must be a valid date in YYYY-MM-DD format",
    );
  if (!input.endDate)
    throw new CapacityOverrideValidationError(
      "endDate",
      "CAPACITY_OVERRIDE_END_DATE_REQUIRED",
      "End date is required",
    );
  if (!isDateOnly(input.endDate))
    throw new CapacityOverrideValidationError(
      "endDate",
      "CAPACITY_OVERRIDE_INVALID_DATE",
      "End date must be a valid date in YYYY-MM-DD format",
    );
  if (input.endDate < input.startDate)
    throw new CapacityOverrideValidationError(
      "endDate",
      "CAPACITY_OVERRIDE_INVALID_DATE_RANGE",
      "End date must not be earlier than start date",
    );
  if (input.capacity === undefined)
    throw new CapacityOverrideValidationError(
      "capacity",
      "CAPACITY_OVERRIDE_CAPACITY_REQUIRED",
      "Capacity is required",
    );
  if (input.capacity < 0)
    throw new CapacityOverrideValidationError(
      "capacity",
      "CAPACITY_OVERRIDE_CAPACITY_NEGATIVE",
      "Capacity must not be negative",
    );
  if (input.capacity > 24)
    throw new CapacityOverrideValidationError(
      "capacity",
      "CAPACITY_OVERRIDE_CAPACITY_EXCEEDS_LIMIT",
      "Capacity must not exceed 24 hours",
    );
  if (!Number.isInteger(input.capacity * 2))
    throw new CapacityOverrideValidationError(
      "capacity",
      "CAPACITY_OVERRIDE_CAPACITY_INVALID_INCREMENT",
      "Capacity must use 0.5-hour increments",
    );
  return { ...input, description } as Required<CapacityOverrideInput>;
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
