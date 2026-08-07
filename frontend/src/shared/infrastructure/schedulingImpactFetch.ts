export const schedulingImpactTokenHeader = "X-Scheduling-Impact-Token";

export type SchedulingImpactProject = {
  id: string;
  name: string;
};

export type SchedulingImpactGroup = {
  id: string;
  projectId: string;
  name: string;
  path: string;
};

export type SchedulingImpact = {
  code:
    | "SCHEDULING_IMPACT_CONFIRMATION_REQUIRED"
    | "SCHEDULING_LOCKED_PROJECT_IMPACT"
    | "SCHEDULING_LOCKED_SCOPE_IMPACT"
    | "SCHEDULING_SCOPE_REOPEN_CLOSURE_REQUIRED"
    | "SCHEDULING_IMPACT_STALE";
  message: string;
  token: string;
  lockedProjects: SchedulingImpactProject[];
  openProjects: SchedulingImpactProject[];
  lockedGroups: SchedulingImpactGroup[];
  openGroups: SchedulingImpactGroup[];
};

export type SchedulingImpactDecision = "confirm" | "cancel" | "reopen-all";
type Listener = (
  impact: SchedulingImpact,
  resolve: (decision: SchedulingImpactDecision) => void,
) => void;

let listener: Listener | undefined;

export function subscribeSchedulingImpact(next: Listener): () => void {
  listener = next;
  return () => {
    if (listener === next) listener = undefined;
  };
}

export class SchedulingImpactCancelledError extends Error {
  constructor() {
    super("Scheduling change was cancelled.");
    this.name = "SchedulingImpactCancelledError";
  }
}

export async function schedulingImpactFetch(
  input: RequestInfo | URL,
  init: RequestInit = {},
  options?: { reopenAll?: (impact: SchedulingImpact) => Promise<Response> },
): Promise<Response> {
  let token: string | undefined;
  for (let attempt = 0; attempt < 4; attempt += 1) {
    const headers = new Headers(init.headers);
    if (token) headers.set(schedulingImpactTokenHeader, token);
    const response = await fetch(input, { ...init, headers });
    const impact = await readSchedulingImpact(response);
    if (!impact) return response;

    const decision = await requestDecision(impact);
    if (
      impact.code === "SCHEDULING_LOCKED_PROJECT_IMPACT" ||
      impact.code === "SCHEDULING_LOCKED_SCOPE_IMPACT"
    ) {
      return response;
    }
    if (impact.code === "SCHEDULING_SCOPE_REOPEN_CLOSURE_REQUIRED") {
      if (decision !== "reopen-all") throw new SchedulingImpactCancelledError();
      if (!options?.reopenAll) return response;
      return options.reopenAll(impact);
    }
    if (decision !== "confirm") throw new SchedulingImpactCancelledError();
    token = impact.token;
  }
  throw new Error(
    "Scheduling impact changed repeatedly. Reload and try again.",
  );
}

async function readSchedulingImpact(
  response: Response,
): Promise<SchedulingImpact | undefined> {
  if (response.status !== 409) return undefined;
  const payload: unknown = await response
    .clone()
    .json()
    .catch(() => undefined);
  if (!isRecord(payload) || !isImpactCode(payload.code)) return undefined;
  const details = isRecord(payload.details) ? payload.details : undefined;
  if (!details || typeof details.token !== "string") return undefined;
  return {
    code: payload.code,
    message:
      typeof payload.message === "string"
        ? payload.message
        : "Scheduling impact requires review.",
    token: details.token,
    lockedProjects: readProjects(details.lockedProjects),
    openProjects: readProjects(details.openProjects),
    lockedGroups: readGroups(details.lockedGroups),
    openGroups: readGroups(details.openGroups),
  };
}

function requestDecision(
  impact: SchedulingImpact,
): Promise<SchedulingImpactDecision> {
  if (!listener) {
    throw new Error("Scheduling impact dialog is unavailable.");
  }
  return new Promise((resolve) => listener?.(impact, resolve));
}

function readProjects(value: unknown): SchedulingImpactProject[] {
  if (!Array.isArray(value)) return [];
  return value.flatMap((item) =>
    isRecord(item) &&
    typeof item.id === "string" &&
    typeof item.name === "string"
      ? [{ id: item.id, name: item.name }]
      : [],
  );
}

function readGroups(value: unknown): SchedulingImpactGroup[] {
  if (!Array.isArray(value)) return [];
  return value.flatMap((item) =>
    isRecord(item) &&
    typeof item.id === "string" &&
    typeof item.projectId === "string" &&
    typeof item.name === "string" &&
    typeof item.path === "string"
      ? [
          {
            id: item.id,
            projectId: item.projectId,
            name: item.name,
            path: item.path,
          },
        ]
      : [],
  );
}

function isImpactCode(value: unknown): value is SchedulingImpact["code"] {
  return (
    value === "SCHEDULING_IMPACT_CONFIRMATION_REQUIRED" ||
    value === "SCHEDULING_LOCKED_PROJECT_IMPACT" ||
    value === "SCHEDULING_LOCKED_SCOPE_IMPACT" ||
    value === "SCHEDULING_SCOPE_REOPEN_CLOSURE_REQUIRED" ||
    value === "SCHEDULING_IMPACT_STALE"
  );
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}
