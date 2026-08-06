import type { PortfolioProjection } from "../domain/portfolio";

export const portfolioProjectionPreferenceKey = "schedmind.home.projection.v1";

export function loadPortfolioProjectionPreference(
  storage?: Pick<Storage, "getItem">,
): PortfolioProjection {
  try {
    const value = (storage ?? window.localStorage).getItem(
      portfolioProjectionPreferenceKey,
    );
    return value === "commitment" || value === "execution"
      ? value
      : "execution";
  } catch {
    return "execution";
  }
}

export function savePortfolioProjectionPreference(
  projection: PortfolioProjection,
  storage?: Pick<Storage, "setItem">,
): void {
  try {
    (storage ?? window.localStorage).setItem(
      portfolioProjectionPreferenceKey,
      projection,
    );
  } catch {
    // Browser storage is optional. The in-memory projection remains applied.
  }
}
