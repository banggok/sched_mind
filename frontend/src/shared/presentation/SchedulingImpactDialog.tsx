import { useEffect, useState } from "react";
import {
  subscribeSchedulingImpact,
  type SchedulingImpact,
} from "../infrastructure/schedulingImpactFetch";
import { Button } from "./Button";
import { Dialog } from "./Dialog";

type PendingImpact = {
  impact: SchedulingImpact;
  resolve(decision: "confirm" | "cancel"): void;
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

  const locked = pending.impact.code === "SCHEDULING_LOCKED_PROJECT_IMPACT";
  const stale = pending.impact.code === "SCHEDULING_IMPACT_STALE";
  const close = () => {
    pending.resolve("cancel");
    setPending(undefined);
  };
  const confirm = () => {
    pending.resolve("confirm");
    setPending(undefined);
  };

  return (
    <Dialog
      titleID="scheduling-impact-title"
      descriptionID="scheduling-impact-description"
      kind="alertdialog"
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
            : stale
              ? "Scheduling impact changed"
              : "Review scheduling impact"}
        </h2>
        <p
          id="scheduling-impact-description"
          className="mt-3 text-sm text-muted"
        >
          {locked
            ? "This change would alter one or more locked project schedules. Reopen those projects before continuing."
            : stale
              ? "The portfolio changed after the previous preview. Review the updated affected projects before confirming again."
              : "Saving this change will reschedule other projects. Review the affected projects before continuing."}
        </p>
        <ImpactGroup
          title="Locked projects"
          projects={pending.impact.lockedProjects}
        />
        <ImpactGroup
          title="Open projects"
          projects={pending.impact.openProjects}
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
            {locked ? "Close" : "Confirm and save"}
          </Button>
        </div>
      </div>
    </Dialog>
  );
}

function ImpactGroup({
  title,
  projects,
}: {
  title: string;
  projects: { id: string; name: string }[];
}) {
  if (projects.length === 0) return null;
  return (
    <section className="mt-5" aria-label={title}>
      <h3 className="text-sm font-bold">{title}</h3>
      <ul className="mt-2 list-disc space-y-1 pl-5 text-sm">
        {projects.map((project) => (
          <li key={project.id}>{project.name}</li>
        ))}
      </ul>
    </section>
  );
}
