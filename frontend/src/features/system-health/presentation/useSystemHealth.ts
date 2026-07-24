import { useEffect, useState } from 'react'

import type { HealthStatus } from './SystemHealthBadge'

type CheckHealth = (signal?: AbortSignal) => Promise<boolean>

export function useSystemHealth(checkHealth: CheckHealth): HealthStatus {
  const [status, setStatus] = useState<HealthStatus>('checking')

  useEffect(() => {
    const controller = new AbortController()

    checkHealth(controller.signal)
      .then((healthy) => setStatus(healthy ? 'online' : 'offline'))
      .catch((error: unknown) => {
        if (!(error instanceof DOMException && error.name === 'AbortError')) {
          setStatus('offline')
        }
      })

    return () => controller.abort()
  }, [checkHealth])

  return status
}

