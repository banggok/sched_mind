import { useEffect, useRef, useState, type FormEvent } from "react";
import type { DependenciesGateway } from "../../dependencies/application/dependenciesGateway";
import { TaskDependencies } from "../../dependencies/presentation/TaskDependencies";
import type { Project } from "../../projects/domain/project";
import type { RolesGateway } from "../../roles/application/rolesGateway";
import type { Role } from "../../roles/domain/role";
import type { TeamMembersGateway } from "../../team-members/application/teamMembersGateway";
import {
  sortTeamMembers,
  type TeamMember,
} from "../../team-members/domain/teamMember";
import type {
  AllocationGroups,
  AllocationRow,
  AssigneeRecommendationInput,
  AssigneeRecommendationItem,
  AssigneeRecommendationResult,
  SchedulePreviewInput,
  WBSGateway,
} from "../application/wbsGateway";
import type { WBSNode } from "../domain/wbs";
import { summarizeWBS } from "../domain/wbsSummary";
import { Alert } from "../../../shared/presentation/Alert";
import { Button } from "../../../shared/presentation/Button";
import { CalendarPopover } from "../../../shared/presentation/CalendarPopover";
import { Dialog } from "../../../shared/presentation/Dialog";
import { formatDateOnly } from "../../../shared/presentation/formatDateOnly";
import { FormField } from "../../../shared/presentation/FormField";
import { WBSSummary } from "./WBSSummary";
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
  onReopened,
  nested = true,
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
  onReopened(node: WBSNode, message: string): void;
  nested?: boolean;
}) {
  const [roles, setRoles] = useState<Role[]>([]);
  const [members, setMembers] = useState<TeamMember[]>([]);
  const [optionsLoadError, setOptionsLoadError] = useState("");
  const [optionsReloadVersion, setOptionsReloadVersion] = useState(0);
  const [name, setName] = useState(node.name);
  const [role, setRole] = useState(node.executable.roleId ?? "");
  const [assignee, setAssignee] = useState(node.executable.assigneeId ?? "");
  const [effort, setEffort] = useState(
    node.executable.effortMinutes
      ? String(node.executable.effortMinutes / 60)
      : "",
  );
  const [lag, setLag] = useState(String(node.executable.lagDays));
  const [capacityAllocationPercentage, setCapacityAllocationPercentage] =
    useState(String(node.executable.capacityAllocationPercentage ?? 100));
  const [capacityAllocationError, setCapacityAllocationError] = useState("");
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
  const [executionUnscheduledReason, setExecutionUnscheduledReason] = useState(
    node.executable.executionUnscheduledReason ?? "",
  );
  const [commitmentUnscheduledReason, setCommitmentUnscheduledReason] =
    useState(node.executable.commitmentUnscheduledReason ?? "");
  const [recommendation, setRecommendation] =
    useState<AssigneeRecommendationResult>();
  const [heldRecommendation, setHeldRecommendation] =
    useState<AssigneeRecommendationResult>();
  const [recommendationBusy, setRecommendationBusy] = useState(false);
  const [recommendationInteractionStale, setRecommendationInteractionStale] =
    useState(false);
  const [recommendationError, setRecommendationError] = useState("");
  const [previewBusy, setPreviewBusy] = useState(false);
  const [previewError, setPreviewError] = useState("");
  const [hasCurrentSchedulePreview, setHasCurrentSchedulePreview] =
    useState(false);
  const [actualStart, setActualStart] = useState("");
  const [actualEnd, setActualEnd] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [reopenOpen, setReopenOpen] = useState(false);
  const [reopenBusy, setReopenBusy] = useState(false);
  const [reopenError, setReopenError] = useState("");
  const reopenLock = useRef(false);
  const recommendationController = useRef<AbortController | undefined>(
    undefined,
  );
  const recommendationRequestSequence = useRef(0);
  const recommendationDraftVersion = useRef(0);
  const assigneeInteractionOpenRef = useRef(false);
  const recommendationInteractionStaleRef = useRef(false);
  const recommendationRef = useRef<AssigneeRecommendationResult | undefined>(
    undefined,
  );
  const previewController = useRef<AbortController | undefined>(undefined);
  const scheduleDraftVersion = useRef(0);
  const previewingVersion = useRef<number | undefined>(undefined);
  const lastPreviewedVersion = useRef(0);
  useEffect(() => {
    if (node.hasChildren) return;
    const controller = new AbortController();
    setOptionsLoadError("");
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
          setOptionsLoadError("Role and member options could not be loaded.");
      });
    return () => controller.abort();
  }, [membersGateway, node.hasChildren, optionsReloadVersion, rolesGateway]);
  useEffect(
    () => () => {
      recommendationController.current?.abort();
      recommendationController.current = undefined;
      previewController.current?.abort();
      previewController.current = undefined;
      previewingVersion.current = undefined;
    },
    [],
  );
  const completed = Boolean(
    node.executable.actualStart && node.executable.actualEnd,
  );
  const readOnly = completed || project.status !== "open";
  const canReopen = completed && project.status === "open";
  const manual =
    project.status === "open" && !project.automaticScheduling && !completed;
  function clearRecommendation() {
    recommendationController.current?.abort();
    recommendationController.current = undefined;
    recommendationRequestSequence.current += 1;
    recommendationDraftVersion.current += 1;
    recommendationRef.current = undefined;
    setRecommendation(undefined);
    setHeldRecommendation(undefined);
    recommendationInteractionStaleRef.current = false;
    setRecommendationInteractionStale(false);
    setRecommendationBusy(false);
    setRecommendationError("");
  }

  function markScheduleDraftChanged(options?: {
    preserveRecommendation?: boolean;
  }) {
    scheduleDraftVersion.current += 1;
    previewController.current?.abort();
    previewController.current = undefined;
    previewingVersion.current = undefined;
    setPreviewBusy(false);
    setPreviewError("");
    setHasCurrentSchedulePreview(false);
    if (!options?.preserveRecommendation) clearRecommendation();
  }

  function recommendationInput(): AssigneeRecommendationInput | undefined {
    const effortHours = parseDecimalDraft(effort);
    if (
      !role ||
      effortHours === undefined ||
      !Number.isFinite(effortHours) ||
      effortHours < 0.5 ||
      !Number.isInteger(effortHours * 2) ||
      !/^\d+$/.test(lag) ||
      !/^\d+$/.test(capacityAllocationPercentage) ||
      Number(capacityAllocationPercentage) < 1 ||
      Number(capacityAllocationPercentage) > 100
    ) {
      return undefined;
    }
    return {
      roleId: role,
      effortHours,
      lagDays: Number(lag),
      capacityAllocationPercentage: Number(capacityAllocationPercentage),
      executionStart: manual && executionStart ? executionStart : undefined,
    };
  }

  async function loadRecommendations() {
    if (readOnly || node.hasChildren) return;
    const input = recommendationInput();
    if (!input) {
      clearRecommendation();
      return;
    }
    if (!gateway.recommendAssignees) {
      recommendationRef.current = undefined;
      setRecommendation(undefined);
      setHeldRecommendation(undefined);
      setRecommendationBusy(false);
      setRecommendationError(
        "Assignee recommendation is temporarily unavailable.",
      );
      return;
    }

    const draftVersion = recommendationDraftVersion.current;
    const requestSequence = recommendationRequestSequence.current + 1;
    recommendationRequestSequence.current = requestSequence;
    const controller = new AbortController();
    recommendationController.current?.abort();
    recommendationController.current = controller;
    setRecommendationBusy(true);
    setRecommendationError("");
    try {
      const result = await gateway.recommendAssignees(
        project.id,
        node.id,
        input,
        controller.signal,
      );
      if (
        controller.signal.aborted ||
        recommendationRequestSequence.current !== requestSequence ||
        recommendationDraftVersion.current !== draftVersion
      ) {
        return;
      }
      if (
        assigneeInteractionOpenRef.current &&
        recommendationRef.current !== undefined
      ) {
        setHeldRecommendation(result);
      } else {
        recommendationRef.current = result;
        setRecommendation(result);
        setHeldRecommendation(undefined);
      }
    } catch (reason: unknown) {
      if (
        controller.signal.aborted ||
        recommendationRequestSequence.current !== requestSequence ||
        recommendationDraftVersion.current !== draftVersion
      ) {
        return;
      }
      recommendationRef.current = undefined;
      setRecommendation(undefined);
      setHeldRecommendation(undefined);
      if (!(reason instanceof DOMException && reason.name === "AbortError")) {
        setRecommendationError(
          "Assignee recommendation is temporarily unavailable.",
        );
      }
    } finally {
      if (
        recommendationRequestSequence.current === requestSequence &&
        recommendationDraftVersion.current === draftVersion
      ) {
        setRecommendationBusy(false);
      }
    }
  }

  useEffect(() => {
    return gateway.subscribeToConfirmedChanges?.(() => {
      scheduleDraftVersion.current += 1;
      previewController.current?.abort();
      previewController.current = undefined;
      previewingVersion.current = undefined;
      setPreviewBusy(false);
      setPreviewError("");
      setHasCurrentSchedulePreview(false);
      recommendationController.current?.abort();
      recommendationController.current = undefined;
      recommendationRequestSequence.current += 1;
      recommendationDraftVersion.current += 1;
      setHeldRecommendation(undefined);
      setRecommendationBusy(false);
      setRecommendationError("");
      if (
        assigneeInteractionOpenRef.current &&
        recommendationRef.current !== undefined
      ) {
        recommendationInteractionStaleRef.current = true;
        setRecommendationInteractionStale(true);
        return;
      }
      recommendationRef.current = undefined;
      setRecommendation(undefined);
      recommendationInteractionStaleRef.current = false;
      setRecommendationInteractionStale(false);
    });
  }, [gateway]);

  function clearDraftSchedule(reason: string) {
    setHasCurrentSchedulePreview(false);
    setExecutionStart("");
    setExecutionEnd("");
    setCommitmentStart("");
    setCommitmentEnd("");
    setExecutionUnscheduledReason(reason);
    setCommitmentUnscheduledReason(reason);
  }

  async function previewSchedule(overrides?: { effort?: string }) {
    if (
      project.status !== "open" ||
      !project.automaticScheduling ||
      completed
    ) {
      return;
    }
    const version = scheduleDraftVersion.current;
    if (
      version === lastPreviewedVersion.current ||
      version === previewingVersion.current
    ) {
      return;
    }
    const effortDraft = overrides?.effort ?? effort;
    const effortHours = parseDecimalDraft(effortDraft);
    const lagValid = /^\d+$/.test(lag);
    if (
      !role ||
      effortHours === undefined ||
      !Number.isFinite(effortHours) ||
      effortHours < 0.5 ||
      !Number.isInteger(effortHours * 2) ||
      !lagValid ||
      !/^\d+$/.test(capacityAllocationPercentage) ||
      Number(capacityAllocationPercentage) < 1 ||
      Number(capacityAllocationPercentage) > 100
    ) {
      clearDraftSchedule(
        "Complete Role, Effort, and valid Lag to preview the schedule.",
      );
      return;
    }

    const input: SchedulePreviewInput = {
      roleId: role,
      assigneeId: assignee || undefined,
      effortHours,
      lagDays: Number(lag),
      capacityAllocationPercentage: Number(capacityAllocationPercentage),
    };
    const controller = new AbortController();
    previewController.current?.abort();
    previewController.current = controller;
    previewingVersion.current = version;
    setPreviewBusy(true);
    setPreviewError("");
    try {
      const preview = await gateway.previewExecutableSchedule(
        project.id,
        node.id,
        input,
        controller.signal,
      );
      if (
        controller.signal.aborted ||
        scheduleDraftVersion.current !== version
      ) {
        return;
      }
      setExecutionStart(preview.task.executable.executionTimeline.start ?? "");
      setExecutionEnd(preview.task.executable.executionTimeline.end ?? "");
      setCommitmentStart(
        preview.task.executable.commitmentTimeline.start ?? "",
      );
      setCommitmentEnd(preview.task.executable.commitmentTimeline.end ?? "");
      setExecutionUnscheduledReason(
        preview.task.executable.executionUnscheduledReason ?? "",
      );
      setCommitmentUnscheduledReason(
        preview.task.executable.commitmentUnscheduledReason ?? "",
      );
      lastPreviewedVersion.current = version;
      setHasCurrentSchedulePreview(true);
    } catch (reason: unknown) {
      if (
        controller.signal.aborted ||
        scheduleDraftVersion.current !== version
      ) {
        return;
      }
      if (!(reason instanceof DOMException && reason.name === "AbortError")) {
        setPreviewError(
          reason instanceof Error
            ? reason.message
            : "Schedule preview could not be generated. Try again.",
        );
      }
    } finally {
      if (previewingVersion.current === version) {
        previewingVersion.current = undefined;
        setPreviewBusy(false);
      }
    }
  }

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
    if (!/^\d+$/.test(lag)) {
      setError("Lag must be a non-negative whole number of days.");
      return;
    }
    if (
      !/^\d+$/.test(capacityAllocationPercentage) ||
      Number(capacityAllocationPercentage) < 1 ||
      Number(capacityAllocationPercentage) > 100
    ) {
      setCapacityAllocationError(
        "Capacity Allocation (%) must be a whole number from 1 to 100.",
      );
      return;
    }
    setCapacityAllocationError("");
    const lagDays = Number(lag);
    setEffort(normalizedEffort);
    previewController.current?.abort();
    previewController.current = undefined;
    previewingVersion.current = undefined;
    setPreviewBusy(false);
    setBusy(true);
    setError("");
    try {
      await gateway.updateExecutable(project.id, node.id, {
        name,
        roleId: role || undefined,
        assigneeId: assignee || undefined,
        effortHours,
        lagDays,
        capacityAllocationPercentage: Number(capacityAllocationPercentage),
        executionStart: manual ? executionStart || undefined : undefined,
        executionEnd: manual ? executionEnd || undefined : undefined,
        commitmentStart: manual ? commitmentStart || undefined : undefined,
        commitmentEnd: manual ? commitmentEnd || undefined : undefined,
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
    if (!actualStart || !actualEnd || busy) return;
    setBusy(true);
    setError("");
    try {
      await gateway.complete(project.id, node.id, actualStart, actualEnd);
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
  async function reopen() {
    if (reopenLock.current || !canReopen) return;
    reopenLock.current = true;
    setReopenBusy(true);
    setReopenError("");
    try {
      const confirmed = await gateway.reopen(project.id, node.id);
      reopenLock.current = false;
      setReopenBusy(false);
      setReopenOpen(false);
      onReopened(confirmed, "Task reopened.");
    } catch (reason: unknown) {
      setReopenError(
        reason instanceof Error
          ? reason.message
          : "The Task could not be reopened. Try again.",
      );
      reopenLock.current = false;
      setReopenBusy(false);
    }
  }
  async function saveGroup(event: FormEvent) {
    event.preventDefault();
    if (project.status !== "open" || busy) return;
    setBusy(true);
    setError("");
    try {
      await gateway.rename(project.id, node.id, name);
      onChanged("Group updated.");
    } catch (reason: unknown) {
      setError(
        reason instanceof Error
          ? reason.message
          : "Group could not be updated.",
      );
    } finally {
      setBusy(false);
    }
  }
  const currentAssignee = members.find((member) => member.id === assignee);
  const currentAssigneeMismatchesRole = Boolean(
    currentAssignee && role && currentAssignee.role.id !== role,
  );
  const eligibleMembers = sortTeamMembers(
    members.filter((member) => !role || member.role.id === role),
  );
  const automaticRecommendationAnchorMissing = Boolean(
    recommendation?.mode === "automatic" &&
    recommendation.items.length > 0 &&
    recommendation.items.every(
      (item) => item.reasonCode === "AUTOMATIC_ANCHOR_MISSING",
    ),
  );
  const recommendationItems = automaticRecommendationAnchorMissing
    ? undefined
    : recommendation?.items.filter((item) => item.roleId === role);
  const optionIdentities = recommendationItems
    ? recommendationItems.map((item) => ({
        id: item.memberId,
        name: item.memberName,
        roleId: item.roleId,
      }))
    : eligibleMembers.map((member) => ({
        id: member.id,
        name: member.name,
        roleId: member.role.id,
      }));
  const duplicateNames = new Set(
    optionIdentities
      .filter(
        (candidate, index, all) =>
          all.findIndex(
            (other) =>
              other.name.localeCompare(candidate.name, undefined, {
                sensitivity: "base",
              }) === 0,
          ) !== index,
      )
      .map((candidate) => candidate.name.toLocaleLowerCase()),
  );
  const assigneeOptions =
    recommendationItems && recommendation
      ? recommendationItems.map((item) => ({
          id: item.memberId,
          roleId: item.roleId,
          label: recommendationOptionLabel(
            item,
            recommendation.mode,
            recommendationMemberName(item, roles, duplicateNames),
          ),
        }))
      : eligibleMembers.map((member) => ({
          id: member.id,
          roleId: member.role.id,
          label: recommendationMemberName(
            {
              memberId: member.id,
              memberName: member.name,
              roleId: member.role.id,
            },
            roles,
            duplicateNames,
          ),
        }));
  if (
    currentAssignee &&
    !currentAssigneeMismatchesRole &&
    !assigneeOptions.some((option) => option.id === currentAssignee.id)
  ) {
    assigneeOptions.push({
      id: currentAssignee.id,
      roleId: currentAssignee.role.id,
      label: currentAssignee.name,
    });
  }
  if (currentAssignee && currentAssigneeMismatchesRole) {
    assigneeOptions.push({
      id: currentAssignee.id,
      roleId: currentAssignee.role.id,
      label: `${currentAssignee.name} — does not match selected Role`,
    });
  }
  const recommendationPrerequisitesComplete =
    recommendationInput() !== undefined;
  return (
    <Dialog
      nested={nested}
      wide
      titleID="wbs-detail-title"
      onClose={() => !busy && !reopenBusy && !reopenLock.current && onClose()}
    >
      <h3 id="wbs-detail-title" className="text-xl font-extrabold">
        {node.hasChildren ? node.name : "Edit Task"}
      </h3>
      <p className="mt-1 text-sm text-muted">
        {node.hasChildren ? "Group" : "Task"}
      </p>
      {node.hasChildren ? (
        <form className="mt-5" onSubmit={(event) => void saveGroup(event)}>
          <FormField
            id="group-detail-name"
            name="name"
            maxLength={200}
            label="Name"
            value={name}
            disabled={project.status !== "open" || busy}
            autoFocus={project.status === "open"}
            onChange={(event) => setName(event.target.value)}
          />
          {project.status !== "open" ? (
            <p className="mt-2 text-sm text-muted">
              Group details are read-only while this Project is {project.status}
              .
            </p>
          ) : null}
          <div className="mt-5">
            <WBSSummary
              summary={summarizeWBS(node.children)}
              subject="group"
              idPrefix={`group-${node.id}-summary`}
            />
          </div>
          {error ? (
            <Alert tone="danger" className="mt-4">
              {error}
            </Alert>
          ) : null}
          <div className="mt-6 flex justify-end gap-3">
            <Button type="button" disabled={busy} onClick={onClose}>
              {project.status === "open" ? "Cancel" : "Close"}
            </Button>
            {project.status === "open" ? (
              <Button type="submit" variant="primary" loading={busy}>
                Save
              </Button>
            ) : null}
          </div>
        </form>
      ) : (
        <>
          <form
            className="mt-5 grid gap-6 md:grid-cols-2"
            onSubmit={(e) => void save(e)}
          >
            <div className="min-w-0 space-y-4">
              <FormField
                id="detail-name"
                name="name"
                maxLength={200}
                label="Name"
                value={name}
                disabled={readOnly}
                autoFocus
                onChange={(e) => setName(e.target.value)}
              />
              <FormField
                id="detail-effort"
                name="effortHours"
                label="Effort (hours)"
                type="text"
                inputMode="decimal"
                value={effort}
                disabled={readOnly}
                onChange={(e) => {
                  if (isDecimalDraft(e.target.value)) {
                    markScheduleDraftChanged();
                    setEffort(e.target.value);
                  }
                }}
                onBlur={() => {
                  const normalized = roundToHalfDraft(effort);
                  setEffort(normalized);
                  void previewSchedule({ effort: normalized });
                }}
              />
              <div>
                <FormField
                  id="detail-capacity-allocation"
                  name="capacityAllocationPercentage"
                  label="Capacity Allocation (%)"
                  type="number"
                  min={1}
                  max={100}
                  step={1}
                  value={capacityAllocationPercentage}
                  disabled={readOnly}
                  error={capacityAllocationError}
                  help="Maximum planned capacity per day. Remaining capacity may be used by other tasks."
                  onChange={(event) => {
                    if (/^\d*$/.test(event.target.value)) {
                      markScheduleDraftChanged();
                      setCapacityAllocationError("");
                      setCapacityAllocationPercentage(event.target.value);
                    }
                  }}
                  onBlur={() => void previewSchedule()}
                />
              </div>
              <div>
                <FormField
                  id="detail-lag"
                  name="lagDays"
                  label="Lag (days)"
                  type="text"
                  inputMode="numeric"
                  value={lag}
                  disabled={readOnly}
                  onChange={(e) => {
                    if (/^\d*$/.test(e.target.value)) {
                      markScheduleDraftChanged();
                      setLag(e.target.value);
                    }
                  }}
                  onBlur={() => void previewSchedule()}
                  aria-describedby="detail-lag-help"
                />
                <p id="detail-lag-help" className="mt-2 text-sm text-muted">
                  Calendar-day offset applied once after blockers or the project
                  scheduling start date.
                </p>
              </div>
              <div>
                <label
                  className="block text-label font-bold"
                  htmlFor="detail-role"
                >
                  Role
                </label>
                <select
                  id="detail-role"
                  className="ui-input mt-2"
                  value={role}
                  disabled={readOnly}
                  onChange={(event) => {
                    markScheduleDraftChanged();
                    setRole(event.target.value);
                  }}
                  onBlur={() => void previewSchedule()}
                >
                  <option value="">No role</option>
                  {roles.map((candidateRole) => (
                    <option key={candidateRole.id} value={candidateRole.id}>
                      {candidateRole.name}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label
                  className="block text-label font-bold"
                  htmlFor="detail-assignee"
                >
                  Assignee
                </label>
                <select
                  id="detail-assignee"
                  className="ui-input mt-2"
                  value={assignee}
                  disabled={readOnly}
                  aria-describedby="detail-assignee-help"
                  aria-busy={recommendationBusy}
                  onFocus={() => {
                    assigneeInteractionOpenRef.current = true;
                    recommendationInteractionStaleRef.current = false;
                    setRecommendationInteractionStale(false);
                    if (recommendationRef.current?.mode === "manual-advisory") {
                      recommendationRef.current = undefined;
                      setRecommendation(undefined);
                      setHeldRecommendation(undefined);
                    }
                    if (!recommendationRef.current && !recommendationBusy) {
                      void loadRecommendations();
                    }
                  }}
                  onChange={(event) => {
                    markScheduleDraftChanged({ preserveRecommendation: true });
                    setAssignee(event.target.value);
                    const member = members.find(
                      (candidate) => candidate.id === event.target.value,
                    );
                    if (member) setRole(member.role.id);
                  }}
                  onBlur={() => {
                    assigneeInteractionOpenRef.current = false;
                    if (recommendationInteractionStaleRef.current) {
                      clearRecommendation();
                    } else if (heldRecommendation) {
                      recommendationRef.current = heldRecommendation;
                      setRecommendation(heldRecommendation);
                      setHeldRecommendation(undefined);
                    }
                    void previewSchedule();
                  }}
                >
                  <option value="">No assignee</option>
                  {assigneeOptions.map((option) => (
                    <option
                      key={option.id}
                      value={option.id}
                      disabled={
                        recommendationBusy || recommendationInteractionStale
                      }
                    >
                      {option.label}
                    </option>
                  ))}
                </select>
                <div
                  id="detail-assignee-help"
                  className="mt-2 space-y-2 text-sm text-muted"
                  role="status"
                  aria-live="polite"
                >
                  {!recommendationPrerequisitesComplete ? (
                    <p>
                      Select Role and enter valid Effort, Capacity Allocation,
                      and Lag to calculate recommendations. Candidates remain
                      alphabetical until then.
                    </p>
                  ) : recommendationBusy ? (
                    <p>Calculating recommendations…</p>
                  ) : recommendationInteractionStale ? (
                    <p>
                      Scheduling state changed while Assignee was open. Close
                      and reopen it to load the latest ranking.
                    </p>
                  ) : recommendationError ? (
                    <div>
                      <p>Assignee recommendation is temporarily unavailable.</p>
                      <Button
                        type="button"
                        className="mt-2"
                        onClick={() => void loadRecommendations()}
                      >
                        Retry recommendations
                      </Button>
                    </div>
                  ) : automaticRecommendationAnchorMissing ? (
                    <p>
                      Automatic Scheduling needs a Project Scheduling Start Date
                      before recommendations can be calculated. Candidates
                      remain alphabetical.
                    </p>
                  ) : recommendation ? (
                    <p>
                      {recommendation.mode === "automatic"
                        ? "Projected finishes"
                        : "Advisory estimates"}{" "}
                      use confirmed scheduling state calculated on{" "}
                      {formatDateOnly(recommendation.calculatedOnDate)}. Save
                      remains authoritative.
                    </p>
                  ) : (
                    <p>
                      Open Assignee to calculate one ranked batch. Selection
                      does not reserve capacity.
                    </p>
                  )}
                </div>
                {currentAssigneeMismatchesRole ? (
                  <Alert tone="warning" className="mt-3">
                    Current Assignee does not match selected Role. Replace or
                    clear it before Save.
                  </Alert>
                ) : null}
                {optionsLoadError ? (
                  <Alert tone="danger" className="mt-3">
                    <p>{optionsLoadError}</p>
                    <Button
                      type="button"
                      className="mt-2"
                      onClick={() =>
                        setOptionsReloadVersion((version) => version + 1)
                      }
                    >
                      Retry member options
                    </Button>
                  </Alert>
                ) : null}
              </div>
            </div>

            <div className="min-w-0 space-y-4">
              <fieldset>
                <legend className="font-bold">
                  {manual ? "Manual timelines" : "Generated schedule"}
                </legend>
                <div className="mt-3 grid gap-4">
                  <TaskTimelineCalendar
                    label="Execution timeline"
                    startDate={executionStart}
                    endDate={executionEnd}
                    disabled={!manual || busy}
                    loadPublicHolidayDates={loadPublicHolidayDates}
                    onChange={(start, end) => {
                      if (start !== executionStart) markScheduleDraftChanged();
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
                <>
                  <p className="text-sm text-muted">
                    Generated dates are controlled by Automatic Scheduling.
                  </p>
                  <div
                    className="rounded-control border border-border-subtle p-3 text-sm"
                    role="status"
                    aria-live="polite"
                    aria-label="Automatic schedule status"
                  >
                    <p className="mb-2 text-muted">
                      {previewBusy
                        ? "Updating schedule preview…"
                        : hasCurrentSchedulePreview
                          ? "Unconfirmed schedule preview. Save confirms the draft."
                          : "Preview updates after leaving Role, Assignee, Effort, Capacity Allocation, or Lag. Save confirms the draft."}
                    </p>
                    <p>
                      <strong>Execution:</strong>{" "}
                      {scheduleStatus(
                        executionStart || undefined,
                        executionEnd || undefined,
                        executionUnscheduledReason || undefined,
                      )}
                    </p>
                    <p className="mt-1">
                      <strong>Commitment:</strong>{" "}
                      {scheduleStatus(
                        commitmentStart || undefined,
                        commitmentEnd || undefined,
                        commitmentUnscheduledReason || undefined,
                      )}
                    </p>
                  </div>
                  {previewError ? (
                    <Alert tone="danger">{previewError}</Alert>
                  ) : null}
                </>
              ) : null}
            </div>
            {error ? (
              <div className="md:col-span-2">
                <Alert tone="danger">{error}</Alert>
              </div>
            ) : null}
            {dependenciesGateway ? (
              <div className="min-w-0 md:col-span-2">
                <TaskDependencies
                  taskId={node.id}
                  gateway={dependenciesGateway}
                  readOnly={readOnly}
                />
              </div>
            ) : null}
            <div className="flex justify-end gap-3 md:col-span-2">
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
              <div className="border-t border-border-subtle pt-4 md:col-span-2">
                <TaskTimelineCalendar
                  label="Actual Date"
                  startDate={actualStart}
                  endDate={actualEnd}
                  disabled={busy}
                  loadPublicHolidayDates={loadPublicHolidayDates}
                  onChange={(start, end) => {
                    setActualStart(start);
                    setActualEnd(end);
                  }}
                />
                {project.status === "locked" ? (
                  <p className="mt-2 text-sm text-muted">
                    Actual Date is the only Task field that remains writable
                    while the Project is Locked. The locked planning baseline
                    will not change.
                  </p>
                ) : null}
                <Button
                  type="button"
                  className="mt-3"
                  loading={busy}
                  disabled={!actualStart || !actualEnd}
                  onClick={() => void complete()}
                >
                  Mark completed
                </Button>
              </div>
            ) : completed ? (
              <div className="space-y-3 md:col-span-2">
                <Alert tone="success">
                  Actual Date: {formatDateOnly(node.executable.actualStart!)} —{" "}
                  {formatDateOnly(node.executable.actualEnd!)}. Completed work
                  is read-only for normal changes.
                </Alert>
                {canReopen ? (
                  <Button
                    type="button"
                    variant="danger"
                    onClick={() => {
                      setReopenError("");
                      setReopenOpen(true);
                    }}
                  >
                    Reopen Task
                  </Button>
                ) : null}
              </div>
            ) : null}
          </form>
          <CapacityAllocationSection
            projectId={project.id}
            taskId={node.id}
            completed={completed}
            manual={!project.automaticScheduling}
            gateway={gateway}
          />
          {reopenOpen ? (
            <Dialog
              nested
              kind="alertdialog"
              titleID="reopen-task-title"
              descriptionID="reopen-task-description"
              closeOnBackdrop={!reopenBusy}
              onClose={() => {
                if (!reopenBusy && !reopenLock.current) setReopenOpen(false);
              }}
            >
              <h4 id="reopen-task-title" className="text-xl font-extrabold">
                Reopen Task?
              </h4>
              <div
                id="reopen-task-description"
                className="mt-4 min-w-0 space-y-3 break-words"
              >
                <p>
                  <strong className="break-words">{node.name}</strong> was
                  completed for {formatDateOnly(node.executable.actualStart!)} —{" "}
                  {formatDateOnly(node.executable.actualEnd!)}.
                </p>
                <p className="text-sm text-muted">
                  Reopening removes both Actual Start and Actual End and returns
                  the Task to unfinished. It is available only while the Project
                  is Open.
                </p>
              </div>
              {reopenError ? (
                <Alert tone="danger" className="mt-4 break-words">
                  {reopenError}
                </Alert>
              ) : null}
              <div className="mt-6 flex flex-wrap justify-end gap-3">
                <Button
                  type="button"
                  disabled={reopenBusy}
                  onClick={() => {
                    if (!reopenLock.current) setReopenOpen(false);
                  }}
                >
                  Cancel
                </Button>
                <Button
                  type="button"
                  variant="danger-solid"
                  loading={reopenBusy}
                  onClick={() => void reopen()}
                >
                  Reopen Task
                </Button>
              </div>
            </Dialog>
          ) : null}
        </>
      )}
    </Dialog>
  );
}

type RecommendationMemberReference = Pick<
  AssigneeRecommendationItem,
  "memberId" | "memberName" | "roleId"
>;

function recommendationMemberName(
  member: RecommendationMemberReference,
  roles: Role[],
  duplicateNames: Set<string>,
): string {
  if (!duplicateNames.has(member.memberName.toLocaleLowerCase())) {
    return member.memberName;
  }
  const roleName =
    roles.find((candidateRole) => candidateRole.id === member.roleId)?.name ??
    "Unknown role";
  const shortID = member.memberId.slice(-6) || member.memberId;
  return `${member.memberName} (${roleName}, ${shortID})`;
}

function recommendationOptionLabel(
  item: AssigneeRecommendationItem,
  mode: AssigneeRecommendationResult["mode"],
  memberName: string,
): string {
  if (item.rankGroup === "no-completion" || !item.executionEnd) {
    return `${memberName} — no projected completion`;
  }
  const finishVerb = mode === "automatic" ? "finishes" : "estimated";
  if (item.rankGroup === "overcapacity") {
    return `${memberName} — ${finishVerb} ${formatDateOnly(
      item.executionEnd,
    )}, adds ${formatRecommendationHours(
      item.incrementalOvercapacityHours,
    )} overcapacity`;
  }
  return `${memberName} — ${finishVerb} ${formatDateOnly(
    item.executionEnd,
  )}, ${formatRecommendationHours(
    item.remainingExecutionCapacityHours,
  )} remaining`;
}

function formatRecommendationHours(hours: number): string {
  const rounded = Math.round(hours * 100) / 100;
  return `${Number.isInteger(rounded) ? rounded : rounded.toFixed(2)}h`;
}

function CapacityAllocationSection({
  projectId,
  taskId,
  completed,
  manual,
  gateway,
}: {
  projectId: string;
  taskId: string;
  completed: boolean;
  manual: boolean;
  gateway: WBSGateway;
}) {
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [groups, setGroups] = useState<AllocationGroups>();

  useEffect(() => {
    if (!open) return;
    const controller = new AbortController();
    setBusy(true);
    setError("");
    void gateway
      .allocations(projectId, taskId, controller.signal)
      .then(setGroups)
      .catch((reason: unknown) => {
        if (!(reason instanceof DOMException && reason.name === "AbortError")) {
          setError(
            reason instanceof Error
              ? reason.message
              : "Capacity allocation could not be loaded.",
          );
        }
      })
      .finally(() => {
        if (!controller.signal.aborted) setBusy(false);
      });
    return () => controller.abort();
  }, [gateway, open, projectId, taskId]);

  return (
    <section className="mt-6 border-t border-border-subtle pt-5">
      <button
        type="button"
        className="flex w-full items-center justify-between gap-3 text-left font-bold"
        aria-expanded={open}
        aria-controls={`capacity-allocation-${taskId}`}
        onClick={() => setOpen((value) => !value)}
      >
        <span>Capacity Allocation</span>
        <span aria-hidden="true">{open ? "−" : "+"}</span>
      </button>
      {open ? (
        <div id={`capacity-allocation-${taskId}`} className="mt-4 space-y-5">
          <p className="text-sm text-muted">
            Allocation dates are analytical capacity rows, not literal work
            timestamps.
          </p>
          {busy ? <p role="status">Loading capacity allocation…</p> : null}
          {error ? <Alert tone="danger">{error}</Alert> : null}
          {groups && !busy ? (
            <>
              <AllocationTable
                title="Execution Allocation"
                rows={groups.execution}
                plannedOvercapacity={manual}
              />
              <AllocationTable
                title="Commitment Allocation"
                rows={groups.commitment}
                plannedOvercapacity={manual}
              />
              {completed ? (
                <AllocationTable
                  title="Actual Allocation"
                  rows={groups.actual}
                  actual
                />
              ) : null}
            </>
          ) : null}
        </div>
      ) : null}
    </section>
  );
}

function AllocationTable({
  title,
  rows,
  actual = false,
  plannedOvercapacity = false,
}: {
  title: string;
  rows: AllocationRow[];
  actual?: boolean;
  plannedOvercapacity?: boolean;
}) {
  return (
    <section
      aria-labelledby={`${title.replace(/\s+/g, "-").toLowerCase()}-title`}
    >
      <h4
        id={`${title.replace(/\s+/g, "-").toLowerCase()}-title`}
        className="font-bold"
      >
        {title}
      </h4>
      {rows.length === 0 ? (
        <p className="mt-2 text-sm text-muted">No allocation rows.</p>
      ) : (
        <div className="mt-2 overflow-x-auto">
          <table className="w-full min-w-[36rem] text-left text-sm">
            <thead>
              <tr className="border-b border-border-subtle">
                <th className="py-2 pr-4">Date</th>
                <th className="py-2 pr-4">Allocated</th>
                <th className="py-2 pr-4">Capacity</th>
                {!actual ? (
                  <th className="py-2 pr-4">Percentage / daily limit</th>
                ) : null}
                <th className="py-2">
                  {actual
                    ? "Remaining / historical overcapacity"
                    : plannedOvercapacity
                      ? "Remaining / planned overcapacity"
                      : "Remaining"}
                </th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => (
                <tr
                  key={`${title}-${row.date}`}
                  className="border-b border-border-subtle"
                >
                  <td className="py-2 pr-4">{formatDateOnly(row.date)}</td>
                  <td className="py-2 pr-4">
                    {formatMinutes(row.allocatedMinutes)}
                  </td>
                  <td className="py-2 pr-4">
                    {formatMinutes(row.capacityMinutes)}
                  </td>
                  {!actual ? (
                    <td className="py-2 pr-4">
                      {row.capacityAllocationPercentage}% /{" "}
                      {formatMinutes(row.taskDailyLimitMinutes)}
                    </td>
                  ) : null}
                  <td className="py-2">
                    {row.overcapacityMinutes > 0
                      ? `${formatMinutes(row.overcapacityMinutes)} ${actual ? "historical" : "planned"} overcapacity`
                      : `${formatMinutes(row.remainingMinutes)} remaining`}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

function formatMinutes(minutes: number): string {
  const hours = minutes / 60;
  return `${Number.isInteger(hours) ? hours : hours.toFixed(2)}h`;
}

function scheduleStatus(
  startDate: string | undefined,
  endDate: string | undefined,
  unscheduledReason: string | undefined,
): string {
  if (unscheduledReason) return unscheduledReason;
  if (startDate && endDate) {
    return `${formatDateOnly(startDate)} — ${formatDateOnly(endDate)}`;
  }
  return "Schedule is pending recalculation.";
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
