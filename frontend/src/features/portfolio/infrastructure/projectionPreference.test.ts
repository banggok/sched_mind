import { describe, expect, it, vi } from "vitest";
import {
  loadPortfolioProjectionPreference,
  portfolioProjectionPreferenceKey,
  savePortfolioProjectionPreference,
} from "./projectionPreference";

describe("Home projection browser preference", () => {
  it("restores only the supported projection enum and defaults invalid values to Execution_DeltaD01", () => {
    for (const [stored, expected] of [
      ["execution", "execution"],
      ["commitment", "commitment"],
      [null, "execution"],
      ["forecast", "execution"],
      ["Commitment", "execution"],
    ] as const) {
      const storage = { getItem: vi.fn().mockReturnValue(stored) };
      expect(loadPortfolioProjectionPreference(storage)).toBe(expected);
      expect(storage.getItem).toHaveBeenCalledWith(
        portfolioProjectionPreferenceKey,
      );
    }
  });

  it("falls back to Execution when browser storage cannot be read_DeltaD01", () => {
    const storage = {
      getItem: vi.fn(() => {
        throw new Error("storage blocked");
      }),
    };

    expect(loadPortfolioProjectionPreference(storage)).toBe("execution");
  });

  it("stores only the applied enum and ignores unavailable writes_DeltaD01", () => {
    const storage = { setItem: vi.fn() };
    savePortfolioProjectionPreference("commitment", storage);
    expect(storage.setItem).toHaveBeenCalledWith(
      portfolioProjectionPreferenceKey,
      "commitment",
    );

    expect(() =>
      savePortfolioProjectionPreference("execution", {
        setItem: vi.fn(() => {
          throw new Error("quota exceeded");
        }),
      }),
    ).not.toThrow();
  });
});
