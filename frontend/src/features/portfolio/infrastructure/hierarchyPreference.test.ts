import { describe, expect, it, vi } from "vitest";
import {
  loadPortfolioCollapsedRows,
  portfolioCollapsedRowsPreferenceKey,
  savePortfolioCollapsedRows,
} from "./hierarchyPreference";

describe("Home collapsed-row browser preference", () => {
  it("restores unique non-empty row identities and ignores invalid entries_DeltaD12", () => {
    const storage = {
      getItem: vi
        .fn()
        .mockReturnValue(
          JSON.stringify(["alpha", "delivery", "alpha", "", 42, null]),
        ),
    };

    expect(loadPortfolioCollapsedRows(storage)).toEqual(
      new Set(["alpha", "delivery"]),
    );
    expect(storage.getItem).toHaveBeenCalledWith(
      portfolioCollapsedRowsPreferenceKey,
    );
  });

  it("falls back to fully expanded when the preference is missing malformed or unreadable_DeltaD12", () => {
    expect(
      loadPortfolioCollapsedRows({ getItem: vi.fn().mockReturnValue(null) }),
    ).toEqual(new Set());
    expect(
      loadPortfolioCollapsedRows({
        getItem: vi.fn().mockReturnValue(JSON.stringify({ alpha: true })),
      }),
    ).toEqual(new Set());
    expect(
      loadPortfolioCollapsedRows({
        getItem: vi.fn().mockReturnValue("not-json"),
      }),
    ).toEqual(new Set());
    expect(
      loadPortfolioCollapsedRows({
        getItem: vi.fn(() => {
          throw new Error("storage blocked");
        }),
      }),
    ).toEqual(new Set());
  });

  it("stores a deterministic identity list and ignores unavailable writes_DeltaD12", () => {
    const storage = { setItem: vi.fn() };
    savePortfolioCollapsedRows(new Set(["delivery", "alpha"]), storage);

    expect(storage.setItem).toHaveBeenCalledWith(
      portfolioCollapsedRowsPreferenceKey,
      JSON.stringify(["alpha", "delivery"]),
    );
    expect(() =>
      savePortfolioCollapsedRows(new Set(["alpha"]), {
        setItem: vi.fn(() => {
          throw new Error("quota exceeded");
        }),
      }),
    ).not.toThrow();
  });
});
