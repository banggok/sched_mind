import {
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
  type RefObject,
} from "react";
import { Alert } from "../../../shared/presentation/Alert";
import { Button } from "../../../shared/presentation/Button";
import type { DependenciesGateway } from "../../dependencies/application/dependenciesGateway";
import type { ProjectsGateway } from "../../projects/application/projectsGateway";
import type { Project } from "../../projects/domain/project";
import type { RolesGateway } from "../../roles/application/rolesGateway";
import type { TeamMembersGateway } from "../../team-members/application/teamMembersGateway";
import type { WBSGateway } from "../../wbs/application/wbsGateway";
import { WBSPanel } from "../../wbs/presentation/WBSPanel";
import type { SprintsGateway } from "../application/sprintsGateway";
import type {
  DailyMinutes,
  SprintDetail,
  SprintMemberProjection,
  SprintTaskProjection,
} from "../domain/sprint";
import {
  deriveReviewProjection,
  sortDailyPlanTasks,
} from "./sprintTaskReviewProjection";

interface SprintTaskEditorDependencies {
  projectsGateway: Pick<ProjectsGateway, "get">;
  wbsGateway: WBSGateway;
  dependenciesGateway: DependenciesGateway;
  rolesGateway: RolesGateway;
  membersGateway: TeamMembersGateway;
  loadPublicHolidayDates?(
    startDate: string,
    endDate: string,
  ): Promise<string[]>;
}

interface EditingTask {
  request: number;
  project: Project;
  taskId: string;
}

export function SprintTaskReview({
  detail,
  gateway,
  taskEditorDependencies,
  onEdit,
  onClose,
  onChanged,
}: {
  detail: SprintDetail;
  gateway: Pick<SprintsGateway, "suggest" | "candidates" | "update">;
  taskEditorDependencies?: SprintTaskEditorDependencies;
  onEdit(): void;
  onClose(): void;
  onChanged(id: string): void;
}) {
  const [members, setMembers] = useState(detail.members);
  const [tasks, setTasks] = useState(detail.tasks);
  const [candidates, setCandidates] = useState<SprintTaskProjection[]>();
  const [loading, setLoading] = useState(false);
  const [candidateLoading, setCandidateLoading] = useState(false);
  const [candidateFeedback, setCandidateFeedback] = useState("");
  const [candidateFocusRequest, setCandidateFocusRequest] = useState(0);
  const [error, setError] = useState("");
  const [openingTaskId, setOpeningTaskId] = useState("");
  const [editingTask, setEditingTask] = useState<EditingTask>();
  const suggestionRequest = useRef(0);
  const candidateRequest = useRef(0);
  const taskEditorRequest = useRef(0);
  const suggestionController = useRef<AbortController | null>(null);
  const candidateController = useRef<AbortController | null>(null);
  const projectionTokenRef = useRef(detail.projectionToken);
  const candidateTitleRef = useRef<HTMLHeadingElement>(null);

  const review = useMemo(
    () =>
      deriveReviewProjection(
        members,
        tasks,
        detail.sprint.startDate,
        detail.sprint.endDate,
      ),
    [detail.sprint.endDate, detail.sprint.startDate, members, tasks],
  );
  const visibleDates = useMemo(
    () => reviewDates(detail.sprint.startDate, detail.sprint.endDate),
    [detail.sprint.endDate, detail.sprint.startDate],
  );

  useLayoutEffect(() => {
    projectionTokenRef.current = detail.projectionToken;
    suggestionRequest.current += 1;
    candidateRequest.current += 1;
    taskEditorRequest.current += 1;
    suggestionController.current?.abort();
    candidateController.current?.abort();
    suggestionController.current = null;
    candidateController.current = null;
    setOpeningTaskId("");
    setEditingTask(undefined);
    return () => {
      taskEditorRequest.current += 1;
      suggestionController.current?.abort();
      candidateController.current?.abort();
    };
  }, [detail.projectionToken]);

  useEffect(() => {
    setMembers(detail.members);
    setTasks(detail.tasks);
    setCandidates(undefined);
    setCandidateFeedback("");
    setLoading(false);
    setCandidateLoading(false);
    setError("");
  }, [detail.members, detail.projectionToken, detail.tasks]);

  useEffect(() => {
    if (!candidates) return;
    candidateTitleRef.current?.scrollIntoView?.({ block: "nearest" });
    candidateTitleRef.current?.focus();
  }, [candidates, candidateFocusRequest]);

  async function generateSuggestion() {
    candidateRequest.current += 1;
    candidateController.current?.abort();
    candidateController.current = null;
    suggestionController.current?.abort();
    const controller = new AbortController();
    suggestionController.current = controller;
    const request = ++suggestionRequest.current;
    const projectionToken = detail.projectionToken;
    setLoading(true);
    setCandidateLoading(false);
    setCandidates(undefined);
    setCandidateFeedback("");
    setError("");
    try {
      const value = await gateway.suggest(
        {
          startDate: detail.sprint.startDate,
          endDate: detail.sprint.endDate,
          memberIds: detail.members.map((member) => member.id),
        },
        controller.signal,
      );
      if (
        request !== suggestionRequest.current ||
        projectionToken !== projectionTokenRef.current
      )
        return;
      setMembers(value.members);
      setTasks(value.tasks.map((item) => item.task));
    } catch {
      if (
        controller.signal.aborted ||
        request !== suggestionRequest.current ||
        projectionToken !== projectionTokenRef.current
      )
        return;
      setError(
        "Task suggestion could not be generated. The current Sprint Planning selection is unchanged.",
      );
    } finally {
      if (suggestionController.current === controller) {
        suggestionController.current = null;
      }
      if (
        !controller.signal.aborted &&
        request === suggestionRequest.current &&
        projectionToken === projectionTokenRef.current
      ) {
        setLoading(false);
      }
    }
  }

  async function openTask(task: SprintTaskProjection) {
    if (!taskEditorDependencies || openingTaskId) return;
    const request = ++taskEditorRequest.current;
    setOpeningTaskId(task.id);
    setEditingTask(undefined);
    setError("");
    try {
      const project = await taskEditorDependencies.projectsGateway.get(
        task.projectId,
      );
      if (request !== taskEditorRequest.current) return;
      setEditingTask({ request, project, taskId: task.id });
    } catch {
      if (request !== taskEditorRequest.current) return;
      setError("The selected Task could not be opened. Please try again.");
    } finally {
      if (request === taskEditorRequest.current) setOpeningTaskId("");
    }
  }

  async function loadCandidates() {
    candidateController.current?.abort();
    const controller = new AbortController();
    candidateController.current = controller;
    const request = ++candidateRequest.current;
    const projectionToken = detail.projectionToken;
    setCandidateLoading(true);
    setCandidates(undefined);
    setCandidateFeedback("");
    setError("");
    try {
      const page = await gateway.candidates(
        detail.sprint.id,
        1,
        controller.signal,
      );
      if (
        request !== candidateRequest.current ||
        projectionToken !== projectionTokenRef.current
      )
        return;
      setCandidates(page.items);
    } catch {
      if (
        controller.signal.aborted ||
        request !== candidateRequest.current ||
        projectionToken !== projectionTokenRef.current
      )
        return;
      setError("Task candidates could not be loaded. Please try again.");
    } finally {
      if (candidateController.current === controller) {
        candidateController.current = null;
      }
      if (
        !controller.signal.aborted &&
        request === candidateRequest.current &&
        projectionToken === projectionTokenRef.current
      ) {
        setCandidateLoading(false);
      }
    }
  }

  async function save() {
    setLoading(true);
    setError("");
    try {
      await gateway.update(detail.sprint.id, {
        name: detail.sprint.name,
        startDate: detail.sprint.startDate,
        endDate: detail.sprint.endDate,
        memberIds: detail.members.map((member) => member.id),
        taskIds: review.tasks.map((task) => task.id),
        version: detail.sprint.version,
      });
      onChanged(detail.sprint.id);
    } catch {
      setError(
        "Sprint Planning could not be saved. Your selection is preserved.",
      );
    } finally {
      setLoading(false);
    }
  }

  return (
    <>
      <section
        className="mt-6 rounded-panel border border-border-strong bg-surface p-5"
        aria-labelledby="sprint-planning-title"
      >
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <p className="text-sm font-bold text-brand">Sprint Planning</p>
            <h2 id="sprint-planning-title" className="text-xl font-black">
              {detail.sprint.name}
            </h2>
            <p className="mt-1 text-sm text-muted">
              Execute Tasks in earliest positive allocation-date order across
              Projects. Same-Date tie-breakers are display order, not an exact
              intra-Day sequence.
            </p>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button onClick={onEdit}>Edit Details & Members</Button>
            <Button onClick={onClose}>Close Sprint Planning</Button>
          </div>
        </div>
        {error ? (
          <Alert tone="danger" className="mt-4">
            {error}
          </Alert>
        ) : null}
        <div className="mt-4 flex flex-wrap gap-2">
          <Button loading={loading} onClick={() => void generateSuggestion()}>
            {tasks.length ? "Regenerate Suggestion" : "Generate Suggestion"}
          </Button>
          <Button
            loading={candidateLoading}
            disabled={loading}
            onClick={() => void loadCandidates()}
          >
            Add Task
          </Button>
          <Button
            variant="primary"
            loading={loading}
            onClick={() => void save()}
          >
            Save Sprint Planning
          </Button>
        </div>
        {candidateLoading ? (
          <p className="mt-4 text-sm text-muted" role="status">
            Loading eligible Tasks…
          </p>
        ) : candidates ? (
          <CandidatePicker
            headingRef={candidateTitleRef}
            candidates={candidates}
            selectedTasks={review.tasks}
            feedback={candidateFeedback}
            onAdd={(candidate) => {
              setTasks((current) =>
                sortDailyPlanTasks([...current, candidate]),
              );
              setCandidateFeedback(
                `${candidate.name} from ${candidate.projectName} was added to Sprint Planning.`,
              );
              setCandidateFocusRequest((current) => current + 1);
            }}
          />
        ) : null}

        {review.members.map((member) => (
          <MemberDailyPlan
            key={member.id}
            member={member}
            tasks={review.tasks.filter(
              (task) =>
                task.assigneeId === member.id &&
                !review.needsReviewTaskIDs.has(task.id),
            )}
            dates={visibleDates}
            sprintStart={detail.sprint.startDate}
            sprintEnd={detail.sprint.endDate}
            onOpenTask={
              taskEditorDependencies ? (task) => void openTask(task) : undefined
            }
            openingTaskId={openingTaskId}
            onRemove={(id) =>
              setTasks((current) => current.filter((task) => task.id !== id))
            }
          />
        ))}

        {review.needsReviewTasks.length ? (
          <NeedsReviewDailyPlan
            tasks={review.needsReviewTasks}
            dates={visibleDates}
            sprintStart={detail.sprint.startDate}
            sprintEnd={detail.sprint.endDate}
            onOpenTask={
              taskEditorDependencies ? (task) => void openTask(task) : undefined
            }
            openingTaskId={openingTaskId}
            onRemove={(id) =>
              setTasks((current) => current.filter((task) => task.id !== id))
            }
          />
        ) : null}
      </section>
      {editingTask && taskEditorDependencies ? (
        <WBSPanel
          key={editingTask.request}
          project={editingTask.project}
          gateway={taskEditorDependencies.wbsGateway}
          dependenciesGateway={taskEditorDependencies.dependenciesGateway}
          rolesGateway={taskEditorDependencies.rolesGateway}
          membersGateway={taskEditorDependencies.membersGateway}
          loadPublicHolidayDates={taskEditorDependencies.loadPublicHolidayDates}
          initialNodeId={editingTask.taskId}
          onMutated={() => void generateSuggestion()}
          onClose={() => setEditingTask(undefined)}
        />
      ) : null}
    </>
  );
}

function CandidatePicker({
  headingRef,
  candidates,
  selectedTasks,
  feedback,
  onAdd,
}: {
  headingRef: RefObject<HTMLHeadingElement | null>;
  candidates: SprintTaskProjection[];
  selectedTasks: SprintTaskProjection[];
  feedback: string;
  onAdd(task: SprintTaskProjection): void;
}) {
  const available = candidates.filter(
    (candidate) => !selectedTasks.some((task) => task.id === candidate.id),
  );
  return (
    <section
      className="mt-4 rounded-panel border border-border-strong p-4"
      aria-labelledby="sprint-candidates-title"
    >
      <h3
        ref={headingRef}
        id="sprint-candidates-title"
        className="font-bold outline-none"
        tabIndex={-1}
      >
        Eligible Tasks
      </h3>
      {feedback ? (
        <p className="mt-2 text-sm font-bold text-success" role="status">
          {feedback}
        </p>
      ) : null}
      {available.length === 0 ? (
        <p className="mt-2 text-muted">No eligible Tasks found.</p>
      ) : (
        <ul className="mt-3 max-h-80 space-y-2 overflow-y-auto">
          {available.map((candidate) => (
            <li
              key={candidate.id}
              className="flex items-center justify-between gap-3 rounded-control border border-border-subtle p-3"
            >
              <span className="min-w-0">
                <span className="block text-sm text-muted">
                  {candidate.projectName}
                </span>
                <strong className="block max-w-[50ch] whitespace-normal break-words">
                  {candidate.name}
                </strong>
                <span className="block text-sm text-muted">
                  {candidate.assigneeName ?? "Unassigned"}
                </span>
                <span className="block text-sm text-muted">
                  {
                    formatTaskDateRange(
                      candidate.executionStart,
                      candidate.executionEnd,
                    ).visible
                  }
                </span>
              </span>
              <Button
                compact
                aria-label={`Add ${candidate.name} from ${candidate.projectName} to Sprint`}
                onClick={() => onAdd(candidate)}
              >
                Add
              </Button>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

function MemberDailyPlan({
  member,
  tasks,
  dates,
  sprintStart,
  sprintEnd,
  onOpenTask,
  openingTaskId,
  onRemove,
}: {
  member: SprintMemberProjection;
  tasks: SprintTaskProjection[];
  dates: string[];
  sprintStart: string;
  sprintEnd: string;
  onOpenTask?(task: SprintTaskProjection): void;
  openingTaskId?: string;
  onRemove(id: string): void;
}) {
  return (
    <section
      className="mt-5 rounded-panel border border-border-subtle p-4"
      aria-labelledby={`review-member-${member.id}`}
    >
      <h3 id={`review-member-${member.id}`} className="font-black">
        {member.name}
      </h3>
      <p className="text-sm text-muted">
        <span aria-hidden="true">
          Capacity: {hours(member.inSprintAllocationMinutes)} of{" "}
          {hours(member.capacityMinutes)}
        </span>
        <span className="sr-only">
          Sprint Usage Capacity: {hours(member.inSprintAllocationMinutes)};
          Sprint Execution Capacity: {hours(member.capacityMinutes)}
        </span>
      </p>
      <DailyPlanTable
        identity={`${member.name} daily working plan`}
        memberName={member.name}
        dates={dates}
        sprintStart={sprintStart}
        sprintEnd={sprintEnd}
        dailyCapacity={member.dailyCapacity}
        tasks={tasks}
        onOpenTask={onOpenTask}
        openingTaskId={openingTaskId}
        onRemove={onRemove}
      />
      {tasks.length === 0 ? (
        <p className="mt-3 text-sm text-muted">No selected Tasks.</p>
      ) : null}
    </section>
  );
}

function NeedsReviewDailyPlan({
  tasks,
  dates,
  sprintStart,
  sprintEnd,
  onOpenTask,
  openingTaskId,
  onRemove,
}: {
  tasks: SprintTaskProjection[];
  dates: string[];
  sprintStart: string;
  sprintEnd: string;
  onOpenTask?(task: SprintTaskProjection): void;
  openingTaskId?: string;
  onRemove(id: string): void;
}) {
  return (
    <section
      className="mt-5 rounded-panel border border-warning p-4"
      aria-labelledby="review-needs-review"
    >
      <h3 id="review-needs-review" className="font-black">
        Needs Review
      </h3>
      <p className="text-sm text-muted">
        These Tasks remain Sprint members, but their allocation is excluded from
        selected-member utilization.
      </p>
      <div
        className="mt-3 overflow-x-auto rounded-control border border-border-subtle"
        role="region"
        aria-label="Needs Review daily allocation table"
        tabIndex={0}
      >
        <table className="min-w-max border-collapse text-sm">
          <thead>
            <tr>
              <th className="sticky left-0 z-10 min-w-80 bg-surface px-3 py-2 text-left">
                Task context
              </th>
              {dates.map((date) => (
                <DateHeader
                  key={date}
                  date={date}
                  sprintStart={sprintStart}
                  sprintEnd={sprintEnd}
                />
              ))}
            </tr>
          </thead>
          <tbody>
            {tasks.map((task) => (
              <TaskRow
                key={task.id}
                task={task}
                memberName="Needs Review"
                dates={dates}
                onOpenTask={onOpenTask}
                openingTaskId={openingTaskId}
                onRemove={onRemove}
                showAssigneeContext
              />
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}

function DailyPlanTable({
  identity,
  memberName,
  dates,
  sprintStart,
  sprintEnd,
  dailyCapacity,
  tasks,
  onOpenTask,
  openingTaskId,
  onRemove,
}: {
  identity: string;
  memberName: string;
  dates: string[];
  sprintStart: string;
  sprintEnd: string;
  dailyCapacity: DailyMinutes[];
  tasks: SprintTaskProjection[];
  onOpenTask?(task: SprintTaskProjection): void;
  openingTaskId?: string;
  onRemove(id: string): void;
}) {
  const capacityByDate = new Map(
    dailyCapacity.map((capacity) => [capacity.date, capacity.minutes]),
  );
  const zeroCapacityDates = new Set(
    dates.filter((date) => (capacityByDate.get(date) ?? 0) === 0),
  );
  return (
    <div
      className="mt-3 overflow-x-auto rounded-control border border-border-subtle"
      role="region"
      aria-label={identity}
      tabIndex={0}
    >
      <table
        className="min-w-max border-collapse text-sm"
        aria-label={identity}
      >
        <thead>
          <tr>
            <th className="sticky left-0 z-10 min-w-80 bg-surface px-3 py-2 text-left">
              Task context
            </th>
            {dates.map((date) => (
              <DateHeader
                key={date}
                date={date}
                sprintStart={sprintStart}
                sprintEnd={sprintEnd}
                noCapacity={zeroCapacityDates.has(date)}
                memberName={memberName}
              />
            ))}
          </tr>
        </thead>
        <tbody>
          <tr>
            <th
              scope="row"
              className="sticky left-0 z-10 bg-surface px-3 py-2 text-left font-bold"
            >
              Capacity
            </th>
            {dates.map((date) => {
              const value = capacityByDate.get(date) ?? 0;
              const noCapacity = zeroCapacityDates.has(date);
              return (
                <td
                  key={date}
                  data-date={date}
                  data-no-capacity={noCapacity ? "true" : undefined}
                  className={`border-l border-border-subtle px-3 py-2 text-right ${
                    noCapacity ? "bg-warning-soft text-warning" : ""
                  }`.trim()}
                  aria-label={`${memberName}, ${formatReviewDate(date)}, Daily Capacity: ${hours(
                    value,
                  )}${noCapacity ? ", no capacity" : ""}`}
                >
                  {noCapacity ? (
                    <strong>
                      <span className="block">0h</span>
                    </strong>
                  ) : (
                    hours(value)
                  )}
                </td>
              );
            })}
          </tr>
          {tasks.map((task) => (
            <TaskRow
              key={task.id}
              task={task}
              memberName={memberName}
              dates={dates}
              onOpenTask={onOpenTask}
              openingTaskId={openingTaskId}
              onRemove={onRemove}
              zeroCapacityDates={zeroCapacityDates}
            />
          ))}
        </tbody>
      </table>
    </div>
  );
}

function TaskRow({
  task,
  memberName,
  dates,
  onOpenTask,
  openingTaskId,
  onRemove,
  zeroCapacityDates = new Set<string>(),
  showAssigneeContext = false,
}: {
  task: SprintTaskProjection;
  memberName: string;
  dates: string[];
  onOpenTask?(task: SprintTaskProjection): void;
  openingTaskId?: string;
  onRemove(id: string): void;
  zeroCapacityDates?: Set<string>;
  showAssigneeContext?: boolean;
}) {
  const allocationByDate = new Map(
    task.allocations.map((value) => [value.date, value.minutes]),
  );
  return (
    <tr data-task-id={task.id}>
      <th
        scope="row"
        className="sticky left-0 z-10 w-[50ch] min-w-[50ch] max-w-[50ch] bg-surface px-3 py-3 text-left align-top"
      >
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            {onOpenTask ? (
              <button
                type="button"
                className="block max-w-[50ch] whitespace-normal break-words text-left font-bold text-brand-strong underline-offset-4 hover:underline disabled:cursor-wait disabled:text-disabled"
                disabled={Boolean(openingTaskId)}
                aria-busy={openingTaskId === task.id ? "true" : undefined}
                onClick={() => onOpenTask(task)}
              >
                {task.name}
              </button>
            ) : (
              <strong className="block max-w-[50ch] whitespace-normal break-words">
                {task.name}
              </strong>
            )}
            <span className="block text-xs text-muted">
              {task.parentName} ({task.projectStatus})
            </span>
            <TaskMetadata task={task} showAssignee={showAssigneeContext} />
            {task.completed ? (
              <span className="mt-1 block text-xs font-bold">Completed</span>
            ) : null}
          </div>
          <Button
            compact
            aria-label={`Remove ${task.name} from Sprint`}
            onClick={() => onRemove(task.id)}
          >
            Remove
          </Button>
        </div>
        {task.warnings.map((warning) => (
          <Alert key={warning} tone="warning" className="mt-2">
            {warning}
          </Alert>
        ))}
      </th>
      {dates.map((date) => {
        const minutes = allocationByDate.get(date) ?? 0;
        const noCapacity = zeroCapacityDates.has(date);
        return (
          <td
            key={date}
            data-date={date}
            data-no-capacity={noCapacity ? "true" : undefined}
            className={`border-l border-border-subtle px-3 py-3 text-right align-top ${
              noCapacity ? "bg-warning-soft" : ""
            }`.trim()}
            aria-label={`${memberName}, ${task.name}, ${task.parentName}, ${formatReviewDate(
              date,
            )}, Sprint allocation: ${hours(minutes)}`}
          >
            {minutes > 0 ? hours(minutes) : "—"}
          </td>
        );
      })}
    </tr>
  );
}

function TaskMetadata({
  task,
  showAssignee,
}: {
  task: SprintTaskProjection;
  showAssignee: boolean;
}) {
  const execution = formatTaskDateRange(task.executionStart, task.executionEnd);
  const commitment = formatTaskDateRange(
    task.commitmentStart,
    task.commitmentEnd,
  );
  return (
    <dl className="mt-2 grid grid-cols-[auto_minmax(0,1fr)] gap-x-2 gap-y-0.5 text-xs text-muted">
      {showAssignee ? (
        <MetadataRow
          label="Assignee"
          value={task.assigneeName ?? "Unassigned"}
        />
      ) : null}
      <MetadataRow label="Effort" value={formatEffort(task.effortMinutes)} />
      <MetadataRow
        label="Execution"
        value={execution.visible}
        accessibleValue={execution.accessible}
      />
      <MetadataRow
        label="Commitment"
        value={commitment.visible}
        accessibleValue={commitment.accessible}
      />
    </dl>
  );
}

function MetadataRow({
  label,
  value,
  accessibleValue,
}: {
  label: string;
  value: string;
  accessibleValue?: string;
}) {
  return (
    <>
      <dt className="font-bold text-foreground">{label}</dt>
      <dd className="min-w-0 break-words">
        {accessibleValue ? (
          <>
            <span aria-hidden="true">{value}</span>
            <span className="sr-only">{accessibleValue}</span>
          </>
        ) : (
          value
        )}
      </dd>
    </>
  );
}

function DateHeader({
  date,
  sprintStart,
  sprintEnd,
  noCapacity = false,
  memberName,
}: {
  date: string;
  sprintStart: string;
  sprintEnd: string;
  noCapacity?: boolean;
  memberName?: string;
}) {
  let context = "Sprint Date";
  if (date === sprintStart && date === sprintEnd)
    context = "Sprint Start & End";
  else if (date === sprintStart) context = "Sprint Start";
  else if (date === sprintEnd) context = "Sprint End";
  return (
    <th
      scope="col"
      data-date={date}
      data-no-capacity={noCapacity ? "true" : undefined}
      aria-label={
        noCapacity
          ? `${memberName ?? "Member"}, ${formatReviewDate(
              date,
            )}, daily capacity 0h, no capacity`
          : undefined
      }
      className={`min-w-28 border-l border-border-subtle px-3 py-2 text-right ${
        noCapacity ? "bg-warning-soft text-warning" : ""
      }`.trim()}
    >
      <span className="block">{formatReviewDate(date)}</span>
      <span className="block text-xs font-normal text-muted">{context}</span>
      {noCapacity ? (
        <span className="block text-xs font-bold">No capacity</span>
      ) : null}
    </th>
  );
}

function reviewDates(sprintStart: string, sprintEnd: string): string[] {
  return dateRange(sprintStart, sprintEnd);
}

function dateRange(start: string, end: string) {
  const values: string[] = [];
  const current = new Date(`${start}T00:00:00Z`);
  const last = new Date(`${end}T00:00:00Z`);
  while (current <= last) {
    values.push(current.toISOString().slice(0, 10));
    current.setUTCDate(current.getUTCDate() + 1);
  }
  return values;
}

function hours(minutes: number) {
  return `${(minutes / 60).toFixed(minutes % 60 === 0 ? 0 : 1)}h`;
}

const reviewMonthLabels = [
  "Jan",
  "Feb",
  "Mar",
  "Apr",
  "May",
  "Jun",
  "Jul",
  "Aug",
  "Sep",
  "Oct",
  "Nov",
  "Dec",
] as const;

function formatReviewDate(value: string) {
  const parsed = parseReviewDate(value);
  if (!parsed) return value;
  return `${parsed.dayPadded} ${parsed.monthLabel} ${parsed.year}`;
}

function formatTaskDateRange(start?: string, end?: string) {
  if (!start || !end) {
    return { visible: "Not scheduled", accessible: undefined };
  }
  const startDate = parseReviewDate(start);
  const endDate = parseReviewDate(end);
  const accessible = `Start: ${formatReviewDate(start)}; End: ${formatReviewDate(
    end,
  )}`;
  if (!startDate || !endDate) {
    return {
      visible: `${formatReviewDate(start)}–${formatReviewDate(end)}`,
      accessible,
    };
  }
  if (start === end) {
    return {
      visible: `${startDate.day} ${startDate.monthLabel} ${startDate.year}`,
      accessible,
    };
  }
  if (startDate.year === endDate.year && startDate.month === endDate.month) {
    return {
      visible: `${startDate.day}–${endDate.day} ${startDate.monthLabel} ${startDate.year}`,
      accessible,
    };
  }
  if (startDate.year === endDate.year) {
    return {
      visible: `${startDate.day} ${startDate.monthLabel}–${endDate.day} ${endDate.monthLabel} ${startDate.year}`,
      accessible,
    };
  }
  return {
    visible: `${startDate.day} ${startDate.monthLabel} ${startDate.year}–${endDate.day} ${endDate.monthLabel} ${endDate.year}`,
    accessible,
  };
}

function parseReviewDate(value: string) {
  const [year, monthValue, dayValue] = value.split("-");
  const month = Number(monthValue);
  const day = Number(dayValue);
  const monthLabel = reviewMonthLabels[month - 1];
  if (!year || !monthLabel || !Number.isInteger(day) || day < 1 || day > 31) {
    return undefined;
  }
  return {
    year,
    month,
    day,
    dayPadded: String(day).padStart(2, "0"),
    monthLabel,
  };
}

function formatEffort(minutes?: number) {
  return minutes === undefined ? "Not set" : hours(minutes);
}
