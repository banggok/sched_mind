import type { ReactNode } from 'react'

import {
  navigationItems,
  type ApplicationPage,
  type NavigationItem,
} from './navigation'

export function ApplicationSidebar({
  activePage,
  open,
  onOpenChange,
}: {
  activePage: ApplicationPage
  open: boolean
  onOpenChange(open: boolean): void
}) {
  return (
    <>
      {open ? (
        <button
          type="button"
          className="fixed inset-0 z-30 bg-[#0C162D]/35 lg:hidden"
          aria-label="Close navigation"
          onClick={() => onOpenChange(false)}
        />
      ) : null}
      <button
        type="button"
        className="fixed top-4 left-[22px] z-50 grid size-9 place-items-center rounded-xl border border-[#D3DCE5] bg-white font-bold shadow-sm hover:border-[#2A93D6] hover:text-[#2A93D6]"
        aria-label={open ? 'Collapse navigation' : 'Expand navigation'}
        aria-expanded={open}
        aria-controls="application-sidebar"
        onClick={() => onOpenChange(!open)}
      >
        ☰
      </button>
      <aside
        id="application-sidebar"
        className={`fixed inset-y-0 left-0 z-40 flex w-64 flex-col border-r border-[#EBF0F5] bg-white shadow-xl transition-transform duration-200 motion-reduce:transition-none ${
          open ? 'translate-x-0' : '-translate-x-full'
        }`}
        aria-label="Application sidebar"
      >
        <div className="h-[69px] shrink-0 border-b border-[#EBF0F5]" />
        <nav className="flex-1 p-3" aria-label="Application navigation">
          {groupNavigationItems().map((group) => (
            <section key={group.id} aria-labelledby={`nav-${group.id}`}>
              <p id={`nav-${group.id}`} className="overflow-hidden px-3 pt-2 pb-2 text-xs font-extrabold tracking-[0.14em] whitespace-nowrap text-[#98999A] uppercase">
                {group.label}
              </p>
              {group.items.map((item) => (
                <NavigationLink
                  key={item.id}
                  item={item}
                  activePage={activePage}
                >
                  {icons[item.id]}
                </NavigationLink>
              ))}
            </section>
          ))}
        </nav>
      </aside>
    </>
  )
}

function NavigationLink({
  item,
  activePage,
  children,
}: {
  item: NavigationItem
  activePage: ApplicationPage
  children: ReactNode
}) {
  const active = item.id === activePage
  return (
    <a
      className={`mt-2 flex items-center gap-3 rounded-xl px-3 py-3 text-sm font-extrabold ${
        active
          ? 'bg-[#E5F5FF] text-[#0C4DA2]'
          : 'text-[#344054] hover:bg-[#F2F4F7]'
      }`}
      href={item.href}
      aria-current={active ? 'page' : undefined}
    >
      <span
        aria-hidden="true"
        className={`grid size-8 shrink-0 place-items-center rounded-lg ${
          active ? 'bg-white text-[#2A93D6]' : 'bg-[#F2F4F7]'
        }`}
      >
        {children}
      </span>
      <span>{item.label}</span>
    </a>
  )
}

const icons: Record<ApplicationPage, ReactNode> = {
  roles: <RoleIcon />,
  'team-members': <TeamMembersIcon />,
}

function groupNavigationItems() {
  return navigationItems.reduce<
    Array<{
      id: string
      label: string
      items: NavigationItem[]
    }>
  >((groups, item) => {
    const existing = groups.find((group) => group.id === item.group)
    if (existing) existing.items.push(item)
    else
      groups.push({
        id: item.group,
        label: item.groupLabel,
        items: [item],
      })
    return groups
  }, [])
}

function RoleIcon() {
  return (
    <svg className="size-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <rect x="3" y="5" width="18" height="14" rx="3" />
      <circle cx="9" cy="11" r="2.25" />
      <path d="M5.75 16c.7-1.6 1.8-2.4 3.25-2.4s2.55.8 3.25 2.4M15 10h3M15 14h2" />
    </svg>
  )
}

function TeamMembersIcon() {
  return (
    <svg className="size-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="9" cy="8" r="3" />
      <path d="M3.5 19c.5-3.5 2.3-5.2 5.5-5.2s5 1.7 5.5 5.2M16 7.5a2.5 2.5 0 0 1 0 5M16.5 14.5c2.4.4 3.7 1.9 4 4.5" />
    </svg>
  )
}
