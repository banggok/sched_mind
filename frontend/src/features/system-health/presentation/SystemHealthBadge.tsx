export type HealthStatus = "checking" | "online" | "offline";

export function SystemHealthBadge({ status }: { status: HealthStatus }) {
  const label = {
    checking: "Checking API",
    online: "Backend connected",
    offline: "Backend offline",
  }[status];

  const dotClass = {
    checking: "bg-brand-light animate-pulse",
    online: "bg-status-online",
    offline: "bg-status-offline",
  }[status];

  return (
    <div
      className="inline-flex items-center gap-2 rounded-full border border-brand/20 bg-brand-soft px-3 py-2 text-xs font-semibold text-brand-dark"
      role="status"
    >
      <span className={`size-2 rounded-full ${dotClass}`} />
      {label}
    </div>
  );
}
