import { describe, expect, it } from "vitest";
import {
  CapacityOverrideValidationError,
  validateCapacityOverride,
} from "./capacityOverride";
describe("capacity override validation", () => {
  it.each([0, 0.5, 24])("accepts %s hours", (capacity) =>
    expect(
      validateCapacityOverride({
        startDate: "2026-07-03",
        endDate: "2026-07-03",
        capacity,
      }).capacity,
    ).toBe(capacity),
  );
  it.each([
    ["2026-02-30", "CAPACITY_OVERRIDE_INVALID_DATE"],
    ["2026-07-04", "CAPACITY_OVERRIDE_INVALID_DATE_RANGE"],
  ])("rejects invalid dates", (_value, code) => {
    expect(() =>
      validateCapacityOverride({
        startDate: _value === "2026-07-04" ? "2026-07-04" : _value,
        endDate: "2026-07-03",
        capacity: 4,
      }),
    ).toThrowError(expect.objectContaining({ code }));
  });
  it.each([
    [-0.5, "CAPACITY_OVERRIDE_CAPACITY_NEGATIVE"],
    [24.5, "CAPACITY_OVERRIDE_CAPACITY_EXCEEDS_LIMIT"],
    [7.2, "CAPACITY_OVERRIDE_CAPACITY_INVALID_INCREMENT"],
  ])("rejects %s", (capacity, code) => {
    try {
      validateCapacityOverride({
        startDate: "2026-07-03",
        endDate: "2026-07-03",
        capacity: capacity as number,
      });
    } catch (error) {
      expect(error).toBeInstanceOf(CapacityOverrideValidationError);
      expect((error as CapacityOverrideValidationError).code).toBe(code);
    }
  });
});
