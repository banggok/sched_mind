import { describe, expect, it, vi } from "vitest";
import {
  loadPortfolioColumnWidths,
  portfolioColumnWidthsPreferenceKey,
  savePortfolioColumnWidths,
} from "./layoutPreference";

const defaults = [
  { key: "name", width: 190, minimum: 140, maximum: 520 },
  { key: "role", width: 100, minimum: 80, maximum: 260 },
] as const;

describe("Home column-width browser preference", () => {
  it("restores known finite widths and clamps them to each column boundary_DeltaD10", () => {
    const storage = {
      getItem: vi
        .fn()
        .mockReturnValue(JSON.stringify({ name: 260, role: 999, unknown: 44 })),
    };

    expect(loadPortfolioColumnWidths(defaults, storage)).toEqual([
      { key: "name", width: 260, minimum: 140, maximum: 520 },
      { key: "role", width: 260, minimum: 80, maximum: 260 },
    ]);
    expect(storage.getItem).toHaveBeenCalledWith(
      portfolioColumnWidthsPreferenceKey,
    );
  });

  it("falls back per column for missing, malformed, or unreadable values_DeltaD10", () => {
    expect(
      loadPortfolioColumnWidths(defaults, {
        getItem: vi
          .fn()
          .mockReturnValue(JSON.stringify({ name: "wide", role: 120 })),
      }),
    ).toEqual([
      { key: "name", width: 190, minimum: 140, maximum: 520 },
      { key: "role", width: 120, minimum: 80, maximum: 260 },
    ]);

    expect(
      loadPortfolioColumnWidths(defaults, {
        getItem: vi.fn().mockReturnValue("not-json"),
      }),
    ).toEqual(defaults);
    expect(
      loadPortfolioColumnWidths(defaults, {
        getItem: vi.fn(() => {
          throw new Error("storage blocked");
        }),
      }),
    ).toEqual(defaults);
  });

  it("stores bounded widths and ignores unavailable writes_DeltaD10", () => {
    const storage = { setItem: vi.fn() };
    savePortfolioColumnWidths(
      [
        { key: "name", width: 260.4, minimum: 140, maximum: 520 },
        { key: "role", width: 999, minimum: 80, maximum: 260 },
      ],
      storage,
    );

    expect(storage.setItem).toHaveBeenCalledWith(
      portfolioColumnWidthsPreferenceKey,
      JSON.stringify({ name: 260, role: 260 }),
    );
    expect(() =>
      savePortfolioColumnWidths(defaults, {
        setItem: vi.fn(() => {
          throw new Error("quota exceeded");
        }),
      }),
    ).not.toThrow();
  });
});
