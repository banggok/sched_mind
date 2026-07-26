import { describe, expect, it } from "vitest";
import { validatePublicHoliday } from "./publicHoliday";

describe("public holiday validation", () => {
  it("trims valid input", () =>
    expect(
      validatePublicHoliday({
        startDate: "2026-08-17",
        endDate: "2026-08-18",
        description: " Holiday ",
      }),
    ).toEqual({
      startDate: "2026-08-17",
      endDate: "2026-08-18",
      description: "Holiday",
    }));

  it.each([
    {
      startDate: "",
      endDate: "2026-08-17",
      description: "Holiday",
      code: "PUBLIC_HOLIDAY_START_DATE_REQUIRED",
    },
    {
      startDate: "2026-02-30",
      endDate: "2026-03-01",
      description: "Holiday",
      code: "INVALID_START_DATE",
    },
    {
      startDate: "2026-08-17",
      endDate: "2026-08-17",
      description: " ",
      code: "PUBLIC_HOLIDAY_DESCRIPTION_REQUIRED",
    },
    {
      startDate: "2026-08-17",
      endDate: "2026-08-17",
      description: "a".repeat(101),
      code: "PUBLIC_HOLIDAY_DESCRIPTION_TOO_LONG",
    },
  ])("rejects invalid input", ({ startDate, endDate, description, code }) =>
    expect(() =>
      validatePublicHoliday({ startDate, endDate, description }),
    ).toThrowError(expect.objectContaining({ code })),
  );
});
