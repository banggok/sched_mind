export function ListSkeleton({
  label,
  rows = 3,
}: {
  label: string
  rows?: number
}) {
  return (
    <div
      className="space-y-3 p-6"
      aria-label={label}
      aria-live="polite"
      role="status"
    >
      {Array.from({ length: rows }, (_, index) => (
        <div
          key={index}
          className="h-16 animate-pulse rounded-2xl bg-[#F2F2F2] motion-reduce:animate-none"
        />
      ))}
    </div>
  )
}
