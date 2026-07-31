let confirmedVersion = 0;

export function currentScheduleProjectionVersion(): number {
  return confirmedVersion;
}

export function advanceScheduleProjectionVersion(): number {
  confirmedVersion += 1;
  return confirmedVersion;
}
