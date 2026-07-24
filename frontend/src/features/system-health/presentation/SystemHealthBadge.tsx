export type HealthStatus = 'checking' | 'online' | 'offline'

export function SystemHealthBadge({ status }: { status: HealthStatus }) {
  const label = {
    checking: 'Checking API',
    online: 'Backend connected',
    offline: 'Backend offline',
  }[status]

  const dotClass = {
    checking: 'bg-[#50ABE5] animate-pulse',
    online: 'bg-emerald-400',
    offline: 'bg-rose-500',
  }[status]

  return (
    <div
      className="inline-flex items-center gap-2 rounded-full border border-[#2A93D6]/20 bg-[#E5F5FF] px-3 py-2 text-xs font-semibold text-[#115488]"
      role="status"
    >
      <span className={`size-2 rounded-full ${dotClass}`} />
      {label}
    </div>
  )
}

