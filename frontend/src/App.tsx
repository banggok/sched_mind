import { useEffect, useState } from 'react'

import { checkBackendHealth, type HealthStatus } from './health'

const palette = [
  { name: 'Navy', value: '#0C162D', className: 'bg-[#0C162D]' },
  { name: 'EDTS Blue', value: '#2A93D6', className: 'bg-[#2A93D6]' },
  { name: 'Sky', value: '#50ABE5', className: 'bg-[#50ABE5]' },
  { name: 'Deep Blue', value: '#0C4DA2', className: 'bg-[#0C4DA2]' },
  { name: 'Cloud', value: '#F7F9FC', className: 'bg-[#F7F9FC]' },
]

const schedule = [
  {
    time: '09:00',
    title: 'Product alignment',
    meta: '45 min · Studio room',
    accent: 'bg-[#2A93D6]',
  },
  {
    time: '11:30',
    title: 'Focus block',
    meta: '90 min · Do not disturb',
    accent: 'bg-[#50ABE5]',
  },
  {
    time: '15:00',
    title: 'Design review',
    meta: '30 min · 4 participants',
    accent: 'bg-[#0C4DA2]',
  },
]

function CalendarIcon() {
  return (
    <svg
      aria-hidden="true"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      className="size-5"
    >
      <path d="M7 3v3m10-3v3M4 9h16M5 5h14a1 1 0 0 1 1 1v14H4V6a1 1 0 0 1 1-1Z" />
    </svg>
  )
}

function HealthBadge({ status }: { status: HealthStatus }) {
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

export function App() {
  const [healthStatus, setHealthStatus] =
    useState<HealthStatus>('checking')

  useEffect(() => {
    const controller = new AbortController()

    checkBackendHealth(controller.signal)
      .then((healthy) => setHealthStatus(healthy ? 'online' : 'offline'))
      .catch((error: unknown) => {
        if (!(error instanceof DOMException && error.name === 'AbortError')) {
          setHealthStatus('offline')
        }
      })

    return () => controller.abort()
  }, [])

  return (
    <main className="min-h-screen bg-white text-[#101828]">
      <header className="border-b border-[#EBF0F5] bg-white px-5 py-5 sm:px-8 lg:px-12">
        <div className="mx-auto flex max-w-7xl items-center justify-between">
          <a
            href="#"
            className="flex items-center gap-3 font-black tracking-[-0.04em]"
          >
            <span className="grid size-9 place-items-center rounded-xl bg-[#E5F5FF] text-[#2A93D6]">
              <CalendarIcon />
            </span>
            SchedMind
          </a>
          <nav className="hidden items-center gap-8 text-sm font-semibold text-[#6D6E70] sm:flex">
            <a href="#schedule" className="transition hover:text-[#2A93D6]">
              Schedule
            </a>
            <a href="#system" className="transition hover:text-[#2A93D6]">
              Design system
            </a>
          </nav>
          <button className="rounded-xl border border-[#2A93D6] bg-white px-5 py-2.5 text-sm font-bold text-[#2A93D6] transition hover:-translate-y-0.5 hover:bg-[#E5F5FF]">
            New event
          </button>
        </div>
      </header>

      <section className="px-5 py-14 sm:px-8 sm:py-20 lg:px-12">
        <div className="mx-auto grid max-w-7xl gap-12 lg:grid-cols-[1.05fr_0.95fr] lg:items-center">
          <div>
            <div className="mb-7 flex flex-wrap items-center gap-3">
              <span className="rounded-full bg-[#E5F5FF] px-3 py-2 text-xs font-extrabold tracking-[0.12em] text-[#0C4DA2] uppercase">
                Design direction 01
              </span>
              <HealthBadge status={healthStatus} />
            </div>
            <h1 className="max-w-3xl text-5xl leading-[0.96] font-black tracking-[-0.055em] text-balance sm:text-7xl lg:text-[5.5rem]">
              Smarter Planning.
              <br />
              <span className="text-[#2A93D6]">Better Delivery.</span>
            </h1>
            <p className="mt-7 max-w-xl text-lg leading-8 text-[#6D6E70]">
              Turn complex schedules into clear priorities, focused execution,
              and reliable delivery.
            </p>
            <div className="mt-9 flex flex-wrap gap-3">
              <button className="rounded-xl bg-[#2A93D6] px-6 py-3.5 text-sm font-extrabold text-white shadow-lg shadow-[#2A93D6]/20 transition hover:-translate-y-0.5 hover:bg-[#0C4DA2]">
                Plan my week
              </button>
              <button className="rounded-xl border border-[#2A93D6] bg-white px-6 py-3.5 text-sm font-extrabold text-[#2A93D6] transition hover:bg-[#E5F5FF]">
                View calendar
              </button>
            </div>
          </div>

          <div
            id="schedule"
            className="relative overflow-hidden rounded-[2rem] bg-[#0C162D] p-5 text-white shadow-2xl shadow-[#0C162D]/20 sm:p-7"
          >
            <div className="absolute -top-16 -right-12 size-48 rounded-full bg-[#2A93D6]/35 blur-3xl" />
            <div className="relative">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-xs font-bold tracking-[0.16em] text-white/45 uppercase">
                    Today
                  </p>
                  <h2 className="mt-1 text-2xl font-extrabold">
                    Friday, July 24
                  </h2>
                </div>
                <span className="rounded-2xl bg-white/8 px-3 py-2 text-sm font-bold">
                  3 events
                </span>
              </div>

              <div className="mt-8 space-y-3">
                {schedule.map((event) => (
                  <article
                    key={event.time}
                    className="grid grid-cols-[3.5rem_0.35rem_1fr] items-center gap-4 rounded-2xl border border-white/8 bg-white/6 p-4"
                  >
                    <time className="text-sm font-bold text-white/55">
                      {event.time}
                    </time>
                    <span
                      className={`h-12 w-1 rounded-full ${event.accent}`}
                    />
                    <div>
                      <h3 className="font-bold">{event.title}</h3>
                      <p className="mt-1 text-sm text-white/45">{event.meta}</p>
                    </div>
                  </article>
                ))}
              </div>

              <div className="mt-5 flex items-center justify-between rounded-2xl bg-[#2A93D6] p-4">
                <div>
                  <p className="text-xs font-bold text-white/60">
                    Focus protected
                  </p>
                  <p className="mt-1 font-extrabold">2h 15m available</p>
                </div>
                <span className="text-2xl" aria-hidden="true">
                  ↗
                </span>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section
        id="system"
        className="border-t border-[#EBF0F5] bg-[#F7F9FC] px-5 py-14 sm:px-8 lg:px-12"
      >
        <div className="mx-auto max-w-7xl">
          <div className="grid gap-8 lg:grid-cols-[0.7fr_1.3fr]">
            <div>
              <p className="text-xs font-extrabold tracking-[0.16em] text-[#2A93D6] uppercase">
                Visual language
              </p>
              <h2 className="mt-3 text-4xl font-black tracking-[-0.04em]">
                Clear, confident,
                <br />
                unmistakably simple.
              </h2>
              <p className="mt-5 max-w-sm leading-7 text-[#6D6E70]">
                Clear blues communicate confidence and technology. White space
                keeps complex schedules approachable and easy to scan.
              </p>
            </div>

            <div className="grid gap-4 sm:grid-cols-5">
              {palette.map((color) => (
                <article
                  key={color.name}
                  className="overflow-hidden rounded-2xl border border-[#EBF0F5] bg-white shadow-sm"
                >
                  <div className={`h-28 ${color.className}`} />
                  <div className="p-4">
                    <h3 className="text-sm font-extrabold">{color.name}</h3>
                    <p className="mt-1 font-mono text-[0.68rem] text-[#98999A]">
                      {color.value}
                    </p>
                  </div>
                </article>
              ))}
            </div>
          </div>

          <div className="mt-12 grid gap-5 md:grid-cols-3">
            <article className="rounded-3xl border border-[#EBF0F5] bg-white p-6">
              <p className="text-xs font-bold text-[#98999A] uppercase">
                Typography
              </p>
              <p className="mt-8 text-5xl font-black tracking-[-0.05em]">Aa</p>
              <p className="mt-3 text-sm text-[#6D6E70]">
                System sans · Bold, friendly, direct
              </p>
            </article>
            <article className="rounded-3xl border border-[#2A93D6]/20 bg-[#2A93D6] p-6 text-white">
              <p className="text-xs font-bold text-white/60 uppercase">
                Success state
              </p>
              <div className="mt-8 flex items-center gap-3">
                <span className="grid size-11 place-items-center rounded-full bg-white/15 text-xl">
                  ✓
                </span>
                <div>
                  <p className="font-extrabold">Time reserved</p>
                  <p className="text-sm text-white/60">Calendar updated</p>
                </div>
              </div>
            </article>
            <article className="rounded-3xl border border-[#C1D3E5] bg-[#E5F5FF] p-6">
              <p className="text-xs font-bold text-[#0C4DA2] uppercase">
                Attention state
              </p>
              <p className="mt-8 text-xl font-extrabold">
                You have a 15 minute overlap.
              </p>
              <button className="mt-5 text-sm font-extrabold underline decoration-2 underline-offset-4">
                Resolve conflict
              </button>
            </article>
          </div>
        </div>
      </section>
    </main>
  )
}
