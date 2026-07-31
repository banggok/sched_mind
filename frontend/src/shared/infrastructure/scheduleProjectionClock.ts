let confirmedVersion = 0;

type ProjectionVersionListener = () => void;

const listeners = new Set<ProjectionVersionListener>();

export function currentScheduleProjectionVersion(): number {
  return confirmedVersion;
}

export function advanceScheduleProjectionVersion(): number {
  confirmedVersion += 1;
  listeners.forEach((listener) => listener());
  return confirmedVersion;
}

export function subscribeScheduleProjectionVersion(
  listener: ProjectionVersionListener,
): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}
