export function isHealthyResponse(value: unknown): boolean {
  if (typeof value !== 'object' || value === null) {
    return false
  }

  return 'status' in value && value.status === 'ok'
}

export async function checkBackendHealth(
  signal?: AbortSignal,
  fetcher: typeof fetch = fetch,
): Promise<boolean> {
  const response = await fetcher('/api/health', { signal })

  if (!response.ok) {
    return false
  }

  return isHealthyResponse(await response.json())
}
