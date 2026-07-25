export function PaginationControls({
  page,
  pageSize,
  total,
  onPageChange,
}: {
  page: number
  pageSize: number
  total: number
  onPageChange(page: number): void
}) {
  const pages = Math.max(1, Math.ceil(total / pageSize))
  const start = total === 0 ? 0 : (page - 1) * pageSize + 1
  const end = Math.min(page * pageSize, total)
  return (
    <nav className="flex items-center justify-between border-t border-[#EBF0F5] px-6 py-4 text-sm" aria-label="Pagination">
      <span className="text-[#6D6E70]">{start}–{end} of {total}</span>
      <div className="flex items-center gap-3">
        <button disabled={page <= 1} className="rounded-lg border px-3 py-2 font-bold disabled:opacity-40" onClick={() => onPageChange(page - 1)}>Previous</button>
        <span aria-current="page">Page {page} of {pages}</span>
        <button disabled={page >= pages} className="rounded-lg border px-3 py-2 font-bold disabled:opacity-40" onClick={() => onPageChange(page + 1)}>Next</button>
      </div>
    </nav>
  )
}
