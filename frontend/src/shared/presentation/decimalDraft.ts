export function parseDecimalDraft(value: string): number | undefined {
  const normalized = value.trim();
  if (!normalized) return undefined;
  const parsed = Number(normalized);
  return Number.isFinite(parsed) ? parsed : Number.NaN;
}

export function roundToHalfDraft(value: string): string {
  const parsed = parseDecimalDraft(value);
  if (parsed === undefined || !Number.isFinite(parsed)) return value;
  return String(Math.round(parsed * 2) / 2);
}

export function isDecimalDraft(value: string): boolean {
  return /^\d*(?:\.\d*)?$/.test(value);
}
