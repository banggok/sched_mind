export function PaginationControls({
  page,
  pageSize,
  total,
  onPageChange,
}: {
  page: number;
  pageSize: number;
  total: number;
  onPageChange(page: number): void;
}) {
  const pages = Math.max(1, Math.ceil(total / pageSize));
  const start = total === 0 ? 0 : (page - 1) * pageSize + 1;
  const end = Math.min(page * pageSize, total);
  return (
    <nav
      className="flex items-center justify-between border-t border-border-subtle px-6 py-4 text-sm"
      aria-label="Pagination"
    >
      <span className="text-muted">
        {start}–{end} of {total}
      </span>
      <div className="flex items-center gap-3">
        <Button
          compact
          disabled={page <= 1}
          onClick={() => onPageChange(page - 1)}
        >
          Previous
        </Button>
        <span aria-current="page">
          Page {page} of {pages}
        </span>
        <Button
          compact
          disabled={page >= pages}
          onClick={() => onPageChange(page + 1)}
        >
          Next
        </Button>
      </div>
    </nav>
  );
}
import { Button } from "./Button";
