import { useEffect, useRef, useState, type RefObject } from "react";
import { Alert } from "../../../shared/presentation/Alert";
import { Button } from "../../../shared/presentation/Button";
import type { WBSGateway } from "../../wbs/application/wbsGateway";
import type { WBSNode } from "../../wbs/domain/wbs";
import type { SprintsGateway } from "../application/sprintsGateway";
import type {
  SprintDetail,
  SprintMemberProjection,
  SprintTaskProjection,
} from "../domain/sprint";

export function SprintTaskReview({
  detail,
  gateway,
  wbsGateway,
  onEdit,
  onClose,
  onChanged,
}: {
  detail: SprintDetail;
  gateway: Pick<SprintsGateway, "suggest" | "candidates" | "update">;
  wbsGateway: Pick<WBSGateway, "tree">;
  onEdit(): void;
  onClose(): void;
  onChanged(id: string): void;
}) {
  const [members, setMembers] = useState(detail.members);
  const [tasks, setTasks] = useState(detail.tasks);
  const [candidates, setCandidates] = useState<SprintTaskProjection[]>();
  const [trees, setTrees] = useState<Record<string, WBSNode[]>>({});
  const [loading, setLoading] = useState(false);
  const [candidateLoading, setCandidateLoading] = useState(false);
  const [candidateFeedback, setCandidateFeedback] = useState("");
  const [candidateFocusRequest, setCandidateFocusRequest] = useState(0);
  const [error, setError] = useState("");
  const treeRequest = useRef(0);
  const candidateTitleRef = useRef<HTMLHeadingElement>(null);
  const selectedMemberIDs = new Set(members.map((member) => member.id));
  const needsReview = tasks.filter(
    (task) =>
      task.warnings.length > 0 ||
      !task.assigneeId ||
      !selectedMemberIDs.has(task.assigneeId),
  );

  useEffect(() => {
    setMembers(detail.members);
    setTasks(detail.tasks);
    setCandidates(undefined);
    setCandidateFeedback("");
  }, [detail]);

  useEffect(() => {
    const projectIDs = [...new Set(tasks.map((task) => task.projectId))];
    const request = ++treeRequest.current;
    if (projectIDs.length === 0) {
      setTrees({});
      return;
    }
    const controller = new AbortController();
    void Promise.all(
      projectIDs.map(
        async (projectID) =>
          [
            projectID,
            await wbsGateway.tree(projectID, controller.signal),
          ] as const,
      ),
    )
      .then((entries) => {
        if (request === treeRequest.current)
          setTrees(Object.fromEntries(entries));
      })
      .catch((reason: unknown) => {
        if (!(reason instanceof DOMException && reason.name === "AbortError"))
          setError(
            "Project structure could not be loaded. Task names remain available.",
          );
      });
    return () => controller.abort();
  }, [tasks, wbsGateway]);

  useEffect(() => {
    if (!candidates) return;
    candidateTitleRef.current?.scrollIntoView?.({ block: "nearest" });
    candidateTitleRef.current?.focus();
  }, [candidates, candidateFocusRequest]);

  async function generateSuggestion() {
    setLoading(true);
    setError("");
    try {
      const value = await gateway.suggest({
        startDate: detail.sprint.startDate,
        endDate: detail.sprint.endDate,
        memberIds: detail.members.map((member) => member.id),
      });
      setMembers(value.members);
      setTasks(value.tasks.map((item) => item.task));
      setCandidates(undefined);
      setCandidateFeedback("");
    } catch {
      setError(
        "Task suggestion could not be generated. The current review is unchanged.",
      );
    } finally {
      setLoading(false);
    }
  }

  async function loadCandidates() {
    setCandidateLoading(true);
    setCandidates(undefined);
    setCandidateFeedback("");
    setError("");
    try {
      const page = await gateway.candidates(detail.sprint.id, 1);
      setCandidates(page.items);
    } catch {
      setError("Task candidates could not be loaded. Please try again.");
    } finally {
      setCandidateLoading(false);
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
        taskIds: tasks.map((task) => task.id),
        version: detail.sprint.version,
      });
      onChanged(detail.sprint.id);
    } catch {
      setError("Task Review could not be saved. Your selection is preserved.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <section
      className="mt-6 rounded-panel border border-border-strong bg-surface p-5"
      aria-labelledby="sprint-task-review-title"
    >
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <p className="text-sm font-bold text-brand">Task Review</p>
          <h2 id="sprint-task-review-title" className="text-xl font-black">
            {detail.sprint.name}
          </h2>
          <p className="mt-1 text-sm text-muted">
            Manage selected Tasks in their Project structure.
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button onClick={onEdit}>Edit Details & Members</Button>
          <Button onClick={onClose}>Close Review</Button>
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
        <Button variant="primary" loading={loading} onClick={() => void save()}>
          Save Task Review
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
          selectedTasks={tasks}
          feedback={candidateFeedback}
          onAdd={(candidate) => {
            setTasks((current) => [...current, candidate]);
            setCandidateFeedback(
              `${candidate.name} from ${candidate.projectName} was added to the Sprint review.`,
            );
            setCandidateFocusRequest((current) => current + 1);
          }}
        />
      ) : null}
      {members.map((member) => (
        <MemberTaskTree
          key={member.id}
          member={member}
          tasks={tasks.filter(
            (task) =>
              task.assigneeId === member.id && !needsReview.includes(task),
          )}
          trees={trees}
          onRemove={(id) =>
            setTasks((current) => current.filter((task) => task.id !== id))
          }
        />
      ))}
      {needsReview.length ? (
        <MemberTaskTree
          member={{
            id: "needs-review",
            name: "Needs Review",
            roleName: "",
            capacityMinutes: 0,
            inSprintAllocationMinutes: needsReview.reduce(
              (total, task) => total + task.inSprintAllocationMinutes,
              0,
            ),
            remainingMinutes: 0,
            overcapacityMinutes: 0,
            dailyCapacity: [],
          }}
          tasks={needsReview}
          trees={trees}
          onRemove={(id) =>
            setTasks((current) => current.filter((task) => task.id !== id))
          }
        />
      ) : null}
    </section>
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
              <span>
                <strong>
                  {candidate.projectName} / {candidate.name}
                </strong>
                <span className="block text-sm text-muted">
                  {candidate.assigneeName ?? "Unassigned"}
                </span>
                <span className="block text-sm text-muted">
                  {candidate.executionStart ?? "Unscheduled"} –{" "}
                  {candidate.executionEnd ?? "Unscheduled"}
                </span>
                <span className="block text-sm text-muted">
                  {hours(candidate.inSprintAllocationMinutes)} in Sprint ·{" "}
                  {hours(candidate.totalAllocationMinutes)} total
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

function MemberTaskTree({
  member,
  tasks,
  trees,
  onRemove,
}: {
  member: SprintMemberProjection;
  tasks: SprintTaskProjection[];
  trees: Record<string, WBSNode[]>;
  onRemove(id: string): void;
}) {
  const projects = [
    ...new Map(
      tasks.map((task) => [task.projectId, task.projectName]),
    ).entries(),
  ];
  return (
    <section
      className="mt-5 rounded-panel border border-border-subtle p-4"
      aria-labelledby={`review-member-${member.id}`}
    >
      <h3 id={`review-member-${member.id}`} className="font-black">
        {member.name}
      </h3>
      <p className="text-sm text-muted">
        {hours(member.inSprintAllocationMinutes)} allocated ·{" "}
        {hours(member.capacityMinutes)} capacity
      </p>
      {projects.length === 0 ? (
        <p className="mt-3 text-sm text-muted">No selected Tasks.</p>
      ) : (
        <ul
          className="mt-3 space-y-3"
          aria-label={`${member.name} Project structure`}
        >
          {projects.map(([projectID, projectName]) => {
            const projectTasks = tasks.filter(
              (task) => task.projectId === projectID,
            );
            const selected = new Set(projectTasks.map((task) => task.id));
            const filtered = filterTree(trees[projectID] ?? [], selected);
            return (
              <li
                key={projectID}
                className="rounded-control bg-surface-muted p-3"
              >
                <div className="font-black">{projectName}</div>
                {filtered.length ? (
                  <ul className="mt-2 border-l border-border-strong pl-4">
                    {filtered.map((node) => (
                      <TreeNode
                        key={node.id}
                        node={node}
                        tasks={projectTasks}
                        onRemove={onRemove}
                      />
                    ))}
                  </ul>
                ) : (
                  <FallbackTasks tasks={projectTasks} onRemove={onRemove} />
                )}
              </li>
            );
          })}
        </ul>
      )}
    </section>
  );
}

function TreeNode({
  node,
  tasks,
  onRemove,
}: {
  node: WBSNode;
  tasks: SprintTaskProjection[];
  onRemove(id: string): void;
}) {
  const task = tasks.find((item) => item.id === node.id);
  return (
    <li className="mt-2">
      <div className="flex items-start justify-between gap-3">
        <span>
          <strong>{node.name}</strong>
          {task ? (
            <>
              <span className="block text-sm text-muted">
                {task.assigneeName ?? "Unassigned"} ·{" "}
                {task.executionStart ?? "Unscheduled"} –{" "}
                {task.executionEnd ?? "Unscheduled"}
              </span>
              <span className="block text-sm text-muted">
                {hours(task.inSprintAllocationMinutes)} in Sprint ·{" "}
                {hours(task.outsideAllocationMinutes)} outside ·{" "}
                {hours(task.totalAllocationMinutes)} total
              </span>
            </>
          ) : (
            <span className="ml-2 text-xs text-muted">Group</span>
          )}
        </span>
        {task ? (
          <Button
            compact
            aria-label={`Remove ${task.name} from Sprint`}
            onClick={() => onRemove(task.id)}
          >
            Remove
          </Button>
        ) : null}
      </div>
      {task?.warnings.map((warning) => (
        <Alert key={warning} tone="warning" className="mt-2">
          {warning}
        </Alert>
      ))}
      {task && task.allocations.length ? (
        <details className="mt-2 text-sm">
          <summary className="cursor-pointer font-bold">
            Daily Execution allocation
          </summary>
          <ul className="mt-1">
            {task.allocations.map((allocation) => (
              <li key={allocation.date}>
                {allocation.date}: {hours(allocation.minutes)}
              </li>
            ))}
          </ul>
        </details>
      ) : null}
      {node.children.length ? (
        <ul className="border-l border-border-subtle pl-4">
          {node.children.map((child) => (
            <TreeNode
              key={child.id}
              node={child}
              tasks={tasks}
              onRemove={onRemove}
            />
          ))}
        </ul>
      ) : null}
    </li>
  );
}

function FallbackTasks({
  tasks,
  onRemove,
}: {
  tasks: SprintTaskProjection[];
  onRemove(id: string): void;
}) {
  return (
    <ul className="mt-2 border-l border-border-strong pl-4">
      {tasks.map((task) => (
        <li key={task.id} className="mt-2 flex justify-between gap-3">
          <span>
            <strong>{task.name}</strong>
          </span>
          <Button
            compact
            aria-label={`Remove ${task.name} from Sprint`}
            onClick={() => onRemove(task.id)}
          >
            Remove
          </Button>
        </li>
      ))}
    </ul>
  );
}

function filterTree(nodes: WBSNode[], selected: Set<string>): WBSNode[] {
  return nodes.flatMap((node) => {
    const children = filterTree(node.children, selected);
    return selected.has(node.id) || children.length
      ? [{ ...node, children }]
      : [];
  });
}

function hours(minutes: number) {
  return `${(minutes / 60).toFixed(minutes % 60 === 0 ? 0 : 1)}h`;
}
