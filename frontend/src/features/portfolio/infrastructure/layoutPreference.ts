export const portfolioColumnWidthsPreferenceKey =
  "schedmind.home.column-widths.v1";

type ColumnWidthDefinition = {
  key: string;
  width: number;
  minimum: number;
  maximum: number;
};

type ResolvedColumnWidth<T extends ColumnWidthDefinition> = Omit<T, "width"> & {
  width: number;
};

export function loadPortfolioColumnWidths<T extends ColumnWidthDefinition>(
  defaults: readonly T[],
  storage?: Pick<Storage, "getItem">,
): ResolvedColumnWidth<T>[] {
  try {
    const raw = (storage ?? window.localStorage).getItem(
      portfolioColumnWidthsPreferenceKey,
    );
    if (!raw) return defaults.map((column) => ({ ...column }));
    const parsed: unknown = JSON.parse(raw);
    if (!isRecord(parsed)) return defaults.map((column) => ({ ...column }));

    return defaults.map((column) => {
      const storedWidth = parsed[column.key];
      if (typeof storedWidth !== "number" || !Number.isFinite(storedWidth))
        return { ...column };
      return {
        ...column,
        width: clamp(Math.round(storedWidth), column.minimum, column.maximum),
      };
    });
  } catch {
    return defaults.map((column) => ({ ...column }));
  }
}

export function savePortfolioColumnWidths(
  columns: readonly ColumnWidthDefinition[],
  storage?: Pick<Storage, "setItem">,
): void {
  try {
    const widths = Object.fromEntries(
      columns.map((column) => [
        column.key,
        clamp(Math.round(column.width), column.minimum, column.maximum),
      ]),
    );
    (storage ?? window.localStorage).setItem(
      portfolioColumnWidthsPreferenceKey,
      JSON.stringify(widths),
    );
  } catch {
    // Browser storage is optional. Current in-memory column widths remain active.
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function clamp(value: number, minimum: number, maximum: number): number {
  return Math.min(maximum, Math.max(minimum, value));
}
