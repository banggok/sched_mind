export function isHealthyResponse(value: unknown): boolean {
  if (typeof value !== "object" || value === null) {
    return false;
  }

  return "status" in value && value.status === "ok";
}

export function createBackendHealthChecker(
  apiBaseURL: string,
  fetcher: typeof fetch = fetch,
): (signal?: AbortSignal) => Promise<boolean> {
  return async (signal?: AbortSignal) => {
    const response = await fetcher(`${apiBaseURL}/health`, { signal });

    if (!response.ok) {
      return false;
    }

    return isHealthyResponse(await response.json());
  };
}
