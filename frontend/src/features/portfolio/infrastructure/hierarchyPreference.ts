export const portfolioCollapsedRowsPreferenceKey =
  "schedmind.home.collapsed-rows.v1";

export function loadPortfolioCollapsedRows(
  storage?: Pick<Storage, "getItem">,
): Set<string> {
  try {
    const raw = (storage ?? window.localStorage).getItem(
      portfolioCollapsedRowsPreferenceKey,
    );
    if (!raw) return new Set();
    const parsed: unknown = JSON.parse(raw);
    if (!Array.isArray(parsed)) return new Set();
    return new Set(
      parsed.filter(
        (value): value is string =>
          typeof value === "string" && value.length > 0,
      ),
    );
  } catch {
    return new Set();
  }
}

export function savePortfolioCollapsedRows(
  collapsed: ReadonlySet<string>,
  storage?: Pick<Storage, "setItem">,
): void {
  try {
    (storage ?? window.localStorage).setItem(
      portfolioCollapsedRowsPreferenceKey,
      JSON.stringify([...collapsed].sort()),
    );
  } catch {
    // Browser storage is optional. Current in-memory hierarchy state remains active.
  }
}
