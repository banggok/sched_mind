import { formatDateOnly } from "../../../shared/presentation/formatDateOnly";
import type {
  TimelineSummary,
  WBSSummary as WBSSummaryModel,
} from "../domain/wbsSummary";

export function WBSSummary({
  summary,
  subject,
  idPrefix,
}: {
  summary: WBSSummaryModel;
  subject: "group" | "project";
  idPrefix: string;
}) {
  if (summary.taskCount === 0) {
    return (
      <p className="rounded-panel border border-border-subtle p-4 text-muted">
        {subject === "project"
          ? "No tasks are available for this project."
          : "No descendant tasks are available for this group."}
      </p>
    );
  }

  return (
    <div
      className="grid min-w-0 gap-4 lg:grid-cols-3"
      aria-label={`${subject === "project" ? "Project" : "Group"} summary details`}
    >
      <TimelineSection
        id={`${idPrefix}-execution`}
        title="Execution Timeline"
        timeline={summary.execution}
        totalTaskCount={summary.taskCount}
      />
      <TimelineSection
        id={`${idPrefix}-commitment`}
        title="Commitment Timeline"
        timeline={summary.commitment}
        totalTaskCount={summary.taskCount}
      />
      <section
        className="min-w-0 rounded-panel border border-border-subtle p-4"
        aria-labelledby={`${idPrefix}-effort-title`}
      >
        <h4 id={`${idPrefix}-effort-title`} className="font-extrabold">
          Effort Completion
        </h4>
        <p className="mt-3 break-words font-bold">
          {summary.completionPercentage === undefined
            ? "Effort completion unavailable"
            : `${formatHours(
                summary.completedKnownEffortMinutes,
              )} of ${formatHours(
                summary.totalKnownEffortMinutes,
              )} hours completed (${formatPercentage(
                summary.completionPercentage,
              )}%)`}
        </p>
        {summary.taskWithoutEffortCount > 0 ? (
          <p className="mt-2 break-words text-sm text-muted">
            {summary.taskWithoutEffortCount === 1
              ? "1 task without effort is excluded from this calculation."
              : `${summary.taskWithoutEffortCount} tasks without effort are excluded from this calculation.`}
          </p>
        ) : null}
      </section>
    </div>
  );
}

function TimelineSection({
  id,
  title,
  timeline,
  totalTaskCount,
}: {
  id: string;
  title: string;
  timeline: TimelineSummary;
  totalTaskCount: number;
}) {
  const hasRange = Boolean(timeline.start && timeline.end);
  const start = timeline.start ? formatDateOnly(timeline.start) : undefined;
  const end = timeline.end ? formatDateOnly(timeline.end) : undefined;

  return (
    <section
      className="min-w-0 rounded-panel border border-border-subtle p-4"
      aria-labelledby={`${id}-title`}
    >
      <h4 id={`${id}-title`} className="font-extrabold">
        {title}
      </h4>
      {hasRange ? (
        <p className="mt-3 break-words font-bold">
          <span className="sr-only">
            Start {start}; End {end}
          </span>
          <span aria-hidden="true">
            {start} – {end}
          </span>
        </p>
      ) : (
        <p className="mt-3 font-bold">Not scheduled</p>
      )}
      <p className="mt-2 break-words text-sm text-muted">
        {timeline.scheduledTaskCount} of {totalTaskCount} tasks scheduled
      </p>
    </section>
  );
}

function formatHours(minutes: number): string {
  const hours = minutes / 60;
  return Number.isInteger(hours)
    ? String(hours)
    : String(Number(hours.toFixed(2)));
}

function formatPercentage(value: number): string {
  return Number.isInteger(value) ? String(value) : value.toFixed(1);
}
