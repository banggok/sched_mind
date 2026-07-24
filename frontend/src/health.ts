export type HealthStatus = 'checking' | 'online' | 'offline'

export function isHealthyResponse(value: unknown): boolean {
  if (typeof value !== 'object' || value === null) {
    return false
  }

  return 'status' in value && value.status === 'ok'
}

export async function checkBackendHealth(signal?: AbortSignal): Promise<boolean> {
  const response = await fetch('/api/health', { signal })

  if (!response.ok) {
    return false
  }

  return isHealthyResponse(await response.json())
}

