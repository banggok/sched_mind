export function SearchField({
  label,
  value,
  onChange,
}: {
  label: string;
  value: string;
  onChange(value: string): void;
}) {
  return (
    <label className="relative block w-full sm:max-w-xs">
      <span className="sr-only">{label}</span>
      <span
        aria-hidden="true"
        className="pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-muted"
      >
        ⌕
      </span>
      <input
        type="search"
        aria-label={label}
        className="ui-input min-h-10 py-2 pr-3 pl-9 text-label"
        placeholder={label}
        value={value}
        onChange={(event) => onChange(event.target.value)}
      />
    </label>
  );
}
