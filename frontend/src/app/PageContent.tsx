import type { ReactNode } from 'react'

export function PageContent({ children }: { children: ReactNode }) {
  return (
    <div className="mx-auto max-w-7xl px-5 py-8 sm:px-8 lg:px-12">
      {children}
    </div>
  )
}
