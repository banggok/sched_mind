import { useEffect, useState } from "react";
import {
  subscribeSchedulingImpact,
  type SchedulingImpact,
} from "../infrastructure/schedulingImpactFetch";
import { Button } from "./Button";
import { Dialog } from "./Dialog";

type PendingImpact = {
  impact: SchedulingImpact;
  resolve(decision: "confirm" | "cancel" | "reopen-all"): void;
};

export function SchedulingImpactDialog() {
  const [pending, setPending] = useState<PendingImpact>();
  useEffect(
    () =>
      subscribeSchedulingImpact((impact, resolve) => {
        setPending({ impact, resolve });
      }),
    [],
  );
  if (!pending) return null;

  const locked =
    pending.impact.code === "SCHEDULING_LOCKED_PROJECT_IMPACT" ||
    pending.impact.code === "SCHEDULING_LOCKED_SCOPE_IMPACT";
  const closure =
    pending.impact.code === "SCHEDULING_SCOPE_REOPEN_CLOSURE_REQUIRED";
  const stale = pending.impact.code === "SCHEDULING_IMPACT_STALE";
  const close = () => {
    pending.resolve("cancel");
    setPending(undefined);
  };
  const confirm = () => {
    pending.resolve(closure ? "reopen-all" : "confirm");
    setPending(undefined);
  };

  return (
    <Dialog
      titleID="scheduling-impact-title"
      descriptionID="scheduling-impact-description"
      kind="alertdialog"
      nested
      closeOnBackdrop={false}
      onClose={close}
    >
      <div className="p-6">
        <h2
          id="scheduling-impact-title"
          className="text-dialog-title font-black"
        >
          {locked
            ? "Change blocked by locked schedule"
            : closure
              ? "Reopen required scheduling scopes"
              : stale
                ? "Scheduling impact changed"
                : "Review scheduling impact"}
        </h2>
        <p
          id="scheduling-impact-description"
          className="mt-3 text-sm text-muted"
        >
          {locked
            ? "This change would shift protected task timelines in one or more locked scheduling scopes. Reopen the listed scopes before continuing."
            : closure
              ? "Reopening this Group requires other protected scheduling scopes to reopen so the resulting timeline can be recalculated atomically."
              : stale
                ? "The portfolio changed after the previous preview. Review the updated timeline-impacted projects and Groups before confirming again."
                : "Saving this change will shift task timelines outside the edited scope. Review the timeline-impacted projects and Groups before continuing."}
        </p>
        <ImpactGroup
          title="Locked projects"
          items={pending.impact.lockedProjects.map((project) => ({
            id: project.id,
            label: project.name,
          }))}
        />
        <ImpactGroup
          title="Open projects"
          items={pending.impact.openProjects.map((project) => ({
            id: project.id,
            label: project.name,
          }))}
        />
        <ImpactGroup
          title="Locked groups"
          items={pending.impact.lockedGroups.map((group) => ({
            id: group.id,
            label: group.path,
          }))}
        />
        <ImpactGroup
          title="Open groups"
          items={pending.impact.openGroups.map((group) => ({
            id: group.id,
            label: group.path,
          }))}
        />
        <div className="mt-6 flex justify-end gap-3">
          {!locked ? (
            <Button type="button" variant="secondary" onClick={close}>
              Cancel
            </Button>
          ) : null}
          <Button
            type="button"
            variant={locked ? "secondary" : "primary"}
            data-autofocus
            onClick={locked ? close : confirm}
          >
            {locked
              ? "Close"
              : closure
                ? "Reopen all"
                : stale
                  ? "Confirm updated impact"
                  : "Confirm and save"}
          </Button>
        </div>
      </div>
    </Dialog>
  );
}

function ImpactGroup({
  title,
  items,
}: {
  title: string;
  items: { id: string; label: string }[];
}) {
  if (items.length === 0) return null;
  return (
    <section className="mt-5" aria-label={title}>
      <h3 className="text-sm font-bold">{title}</h3>
      <ul className="mt-2 list-disc space-y-1 pl-5 text-sm">
        {items.map((item) => (
          <li key={item.id}>{item.label}</li>
        ))}
      </ul>
    </section>
  );
}
