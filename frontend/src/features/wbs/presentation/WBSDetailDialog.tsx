import { useEffect, useState, type FormEvent } from "react";
import type { DependenciesGateway } from "../../dependencies/application/dependenciesGateway";
import { TaskDependencies } from "../../dependencies/presentation/TaskDependencies";
import type { Project } from "../../projects/domain/project";
import type { RolesGateway } from "../../roles/application/rolesGateway";
import type { Role } from "../../roles/domain/role";
import type { TeamMembersGateway } from "../../team-members/application/teamMembersGateway";
import type { TeamMember } from "../../team-members/domain/teamMember";
import type { WBSGateway } from "../application/wbsGateway";
import type { WBSNode } from "../domain/wbs";
import { Alert } from "../../../shared/presentation/Alert";
import { Button } from "../../../shared/presentation/Button";
import { CalendarPopover } from "../../../shared/presentation/CalendarPopover";
import { Dialog } from "../../../shared/presentation/Dialog";
import { formatDateOnly } from "../../../shared/presentation/formatDateOnly";
import { FormField } from "../../../shared/presentation/FormField";
import {
  isDecimalDraft,
  parseDecimalDraft,
  roundToHalfDraft,
} from "../../../shared/presentation/decimalDraft";

export function WBSDetailDialog({
  project,
  node,
  gateway,
  dependenciesGateway,
  rolesGateway,
  membersGateway,
  loadPublicHolidayDates,
  onClose,
  onChanged,
}: {
  project: Project;
  node: WBSNode;
  gateway: WBSGateway;
  dependenciesGateway?: DependenciesGateway;
  rolesGateway: RolesGateway;
  membersGateway: TeamMembersGateway;
  loadPublicHolidayDates?(
    startDate: string,
    endDate: string,
  ): Promise<string[]>;
  onClose(): void;
  onChanged(message: string): void;
}) {
  const [roles, setRoles] = useState<Role[]>([]);
  const [members, setMembers] = useState<TeamMember[]>([]);
  const [name, setName] = useState(node.name);
  const [role, setRole] = useState(node.executable.roleId ?? "");
  const [assignee, setAssignee] = useState(node.executable.assigneeId ?? "");
  const [effort, setEffort] = useState(
    node.executable.effortMinutes
      ? String(node.executable.effortMinutes / 60)
      : "",
  );
  const [executionStart, setExecutionStart] = useState(
    node.executable.executionTimeline.start ?? "",
  );
  const [executionEnd, setExecutionEnd] = useState(
    node.executable.executionTimeline.end ?? "",
  );
  const [commitmentStart, setCommitmentStart] = useState(
    node.executable.commitmentTimeline.start ?? "",
  );
  const [commitmentEnd, setCommitmentEnd] = useState(
    node.executable.commitmentTimeline.end ?? "",
  );
  const [actualEnd, setActualEnd] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  useEffect(() => {
    const controller = new AbortController();
    setError("");
    void Promise.all([
      rolesGateway.list(
        { search: "", page: 1, pageSize: 100 },
        controller.signal,
      ),
      membersGateway.list(
        { search: "", page: 1, pageSize: 100 },
        controller.signal,
      ),
    ])
      .then(([r, m]) => {
        setRoles(r.items);
        setMembers(m.items);
      })
      .catch((reason: unknown) => {
        if (!(reason instanceof DOMException && reason.name === "AbortError"))
          setError("Role and member options could not be loaded.");
      });
    return () => controller.abort();
  }, [rolesGateway, membersGateway]);
  const completed = Boolean(node.executable.actualEnd);
  const readOnly = completed || project.status === "closed";
  const manual =
    project.status === "open" && !project.automaticScheduling && !completed;
  async function save(event: FormEvent) {
    event.preventDefault();
    if (busy || readOnly) return;
    const normalizedEffort = roundToHalfDraft(effort);
    const effortHours = parseDecimalDraft(normalizedEffort);
    if (
      effortHours !== undefined &&
      (!Number.isFinite(effortHours) || effortHours < 0.5)
    ) {
      setError("Effort must be at least 0.5 hours in 0.5-hour increments.");
      return;
    }
    setEffort(normalizedEffort);
    setBusy(true);
    setError("");
    try {
      await gateway.updateExecutable(project.id, node.id, {
        name,
        roleId: role || undefined,
        assigneeId: assignee || undefined,
        effortHours,
        executionStart: executionStart || undefined,
        executionEnd: executionEnd || undefined,
        commitmentStart: commitmentStart || undefined,
        commitmentEnd: commitmentEnd || undefined,
      });
      onChanged("Task updated.");
    } catch (reason: unknown) {
      setError(
        reason instanceof Error ? reason.message : "Task could not be updated.",
      );
    } finally {
      setBusy(false);
    }
  }
  async function complete() {
    if (!actualEnd || busy) return;
    setBusy(true);
    setError("");
    try {
      await gateway.complete(project.id, node.id, actualEnd);
      onChanged("Task completed.");
    } catch (reason: unknown) {
      setError(
        reason instanceof Error
          ? reason.message
          : "Task could not be completed.",
      );
    } finally {
      setBusy(false);
    }
  }
  return (
    <Dialog
      nested
      titleID="wbs-detail-title"
      onClose={() => !busy && onClose()}
    >
      <h3 id="wbs-detail-title" className="text-xl font-extrabold">
        {node.hasChildren ? node.name : "Edit Task"}
      </h3>
      <p className="mt-1 text-sm text-muted">
        {node.hasChildren ? "Group" : "Task"}
      </p>
      {node.hasChildren ? (
        <>
          <Alert tone="success" className="mt-5">
            This group contains {node.children.length} direct item items. Task
            details are managed on tasks inside this group.
          </Alert>
          <div className="mt-6 flex justify-end">
            <Button onClick={onClose}>Close</Button>
          </div>
        </>
      ) : (
        <form className="mt-5 space-y-4" onSubmit={(e) => void save(e)}>
          <FormField
            id="detail-name"
            name="name"
            label="Name"
            value={name}
            disabled={readOnly}
            autoFocus
            onChange={(e) => setName(e.target.value)}
          />
          <label className="block text-label font-bold" htmlFor="detail-role">
            Role
          </label>
          <select
            id="detail-role"
            className="ui-input"
            value={role}
            disabled={readOnly}
            onChange={(e) => {
              setRole(e.target.value);
              if (
                members.find((m) => m.id === assignee)?.role.id !==
                e.target.value
              )
                setAssignee("");
            }}
          >
            <option value="">No role</option>
            {roles.map((r) => (
              <option key={r.id} value={r.id}>
                {r.name}
              </option>
            ))}
          </select>
          <label
            className="block text-label font-bold"
            htmlFor="detail-assignee"
          >
            Assignee
          </label>
          <select
            id="detail-assignee"
            className="ui-input"
            value={assignee}
            disabled={readOnly}
            onChange={(e) => {
              setAssignee(e.target.value);
              const m = members.find((v) => v.id === e.target.value);
              if (m) setRole(m.role.id);
            }}
          >
            <option value="">No assignee</option>
            {members
              .filter((m) => !role || m.role.id === role)
              .map((m) => (
                <option key={m.id} value={m.id}>
                  {m.name}
                </option>
              ))}
          </select>
          <FormField
            id="detail-effort"
            name="effortHours"
            label="Effort (hours)"
            type="text"
            inputMode="decimal"
            value={effort}
            disabled={readOnly}
            onChange={(e) => {
              if (isDecimalDraft(e.target.value)) setEffort(e.target.value);
            }}
            onBlur={() => setEffort(roundToHalfDraft(effort))}
          />
          <fieldset>
            <legend className="font-bold">Manual timelines</legend>
            <div className="mt-3 grid gap-4">
              <TaskTimelineCalendar
                label="Execution timeline"
                startDate={executionStart}
                endDate={executionEnd}
                disabled={!manual || busy}
                loadPublicHolidayDates={loadPublicHolidayDates}
                onChange={(start, end) => {
                  setExecutionStart(start);
                  setExecutionEnd(end);
                }}
              />
              <TaskTimelineCalendar
                label="Commitment timeline"
                startDate={commitmentStart}
                endDate={commitmentEnd}
                disabled={!manual || busy}
                loadPublicHolidayDates={loadPublicHolidayDates}
                onChange={(start, end) => {
                  setCommitmentStart(start);
                  setCommitmentEnd(end);
                }}
              />
            </div>
          </fieldset>
          {!manual && !completed ? (
            <p className="text-sm text-muted">
              Manual timelines require an Open project with Automatic Scheduling
              off.
            </p>
          ) : null}
          {error ? <Alert tone="danger">{error}</Alert> : null}
          {dependenciesGateway ? (
            <TaskDependencies
              taskId={node.id}
              gateway={dependenciesGateway}
              readOnly={readOnly}
            />
          ) : null}
          <div className="flex justify-end gap-3">
            <Button type="button" onClick={onClose}>
              Close
            </Button>
            <Button
              type="submit"
              variant="primary"
              loading={busy}
              disabled={readOnly}
            >
              Save
            </Button>
          </div>
          {!completed && project.status !== "closed" ? (
            <div className="border-t border-border-subtle pt-4">
              <CalendarPopover
                label="Actual End"
                buttonLabel={
                  actualEnd ? formatDateOnly(actualEnd) : "Select date"
                }
                initialDate={actualEnd}
                instruction="Select the task completion date."
                selectedDates={actualEnd ? [actualEnd] : []}
                disabled={busy}
                loadPublicHolidayDates={loadPublicHolidayDates}
                onSelect={(date) => {
                  setActualEnd(date);
                  return true;
                }}
              />
              <Button
                type="button"
                className="mt-3"
                loading={busy}
                disabled={!actualEnd}
                onClick={() => void complete()}
              >
                Mark completed
              </Button>
            </div>
          ) : completed ? (
            <Alert tone="success">
              Completed on {node.executable.actualEnd}. Completed work is
              read-only.
            </Alert>
          ) : null}
        </form>
      )}
    </Dialog>
  );
}

function TaskTimelineCalendar({
  label,
  startDate,
  endDate,
  disabled,
  loadPublicHolidayDates,
  onChange,
}: {
  label: string;
  startDate: string;
  endDate: string;
  disabled: boolean;
  loadPublicHolidayDates?(
    startDate: string,
    endDate: string,
  ): Promise<string[]>;
  onChange(startDate: string, endDate: string): void;
}) {
  return (
    <CalendarPopover
      label={label}
      buttonLabel={
        startDate
          ? endDate
            ? `${formatDateOnly(startDate)} — ${formatDateOnly(endDate)}`
            : `${formatDateOnly(startDate)} — Select end date`
          : "Select start and end date"
      }
      initialDate={startDate}
      instruction={
        startDate && !endDate
          ? "Select an end date. Choose an earlier date to replace the start date."
          : "Select a start date."
      }
      selectedDates={[startDate, endDate].filter(Boolean)}
      disabled={disabled}
      loadPublicHolidayDates={loadPublicHolidayDates}
      isInRange={(date) =>
        Boolean(startDate && endDate && date > startDate && date < endDate)
      }
      onSelect={(date) => {
        if (!startDate || endDate || date < startDate) {
          onChange(date, "");
          return false;
        }
        onChange(startDate, date);
        return true;
      }}
    />
  );
}
