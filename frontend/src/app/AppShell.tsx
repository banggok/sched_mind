import { useEffect, useState, type ReactNode } from 'react'

import { ApplicationSidebar } from './ApplicationSidebar'
import type { ApplicationPage } from './navigation'
import { TopBar } from './TopBar'

export function AppShell({
  activePage,
  children,
}: {
  activePage: ApplicationPage
  children: ReactNode
}) {
  const [sidebarOpen, setSidebarOpen] = useState(readSidebarState)

  useEffect(() => {
    try {
      window.localStorage.setItem(
        sidebarStateStorageKey,
        sidebarOpen ? 'expanded' : 'collapsed',
      )
    } catch {
      // Storage may be unavailable; the in-memory state remains usable.
    }
  }, [sidebarOpen])

  return (
    <div className="min-h-screen bg-[#F7F9FC] text-[#101828]">
      <ApplicationSidebar
        activePage={activePage}
        open={sidebarOpen}
        onOpenChange={setSidebarOpen}
      />
      <main
        className={`min-h-screen transition-[margin] duration-200 motion-reduce:transition-none ${
          sidebarOpen ? 'lg:ml-64' : 'ml-0'
        }`}
      >
        <TopBar />
        {children}
      </main>
    </div>
  )
}

const sidebarStateStorageKey = 'schedmind.application-sidebar'

function readSidebarState(): boolean {
  try {
    return window.localStorage.getItem(sidebarStateStorageKey) === 'expanded'
  } catch {
    return false
  }
}
