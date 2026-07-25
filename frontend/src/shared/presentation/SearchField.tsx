export function SearchField({
  label,
  value,
  onChange,
}: {
  label: string
  value: string
  onChange(value: string): void
}) {
  return (
    <label className="relative block w-full sm:max-w-xs">
      <span className="sr-only">{label}</span>
      <span
        aria-hidden="true"
        className="pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-[#6D6E70]"
      >
        ⌕
      </span>
      <input
        type="search"
        aria-label={label}
        className="w-full rounded-xl border border-[#D3DCE5] bg-white py-2 pr-3 pl-9 text-sm outline-none focus:border-[#2A93D6] focus:ring-2 focus:ring-[#2A93D6]/15"
        placeholder={label}
        value={value}
        onChange={(event) => onChange(event.target.value)}
      />
    </label>
  )
}
