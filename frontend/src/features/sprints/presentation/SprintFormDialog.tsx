import { useEffect, useState } from "react";
import type { TeamMember } from "../../team-members/domain/teamMember";
import type { TeamMembersGateway } from "../../team-members/application/teamMembersGateway";
import { Alert } from "../../../shared/presentation/Alert";
import { Button } from "../../../shared/presentation/Button";
import { CalendarPopover } from "../../../shared/presentation/CalendarPopover";
import { Dialog } from "../../../shared/presentation/Dialog";
import { formatDateOnly } from "../../../shared/presentation/formatDateOnly";
import {
  SprintAPIError,
  type SprintsGateway,
} from "../application/sprintsGateway";
import type { Sprint, SprintDetail } from "../domain/sprint";

export function SprintFormDialog({
  gateway,
  membersGateway,
  initialDetail,
  loadPublicHolidayDates,
  onClose,
  onSaved,
}: {
  gateway: Pick<SprintsGateway, "create" | "update">;
  membersGateway: TeamMembersGateway;
  initialDetail?: SprintDetail;
  loadPublicHolidayDates(startDate: string, endDate: string): Promise<string[]>;
  onClose(): void;
  onSaved(sprint: Sprint): void;
}) {
  const [step, setStep] = useState<1 | 2>(1);
  const [name, setName] = useState(initialDetail?.sprint.name ?? "");
  const [startDate, setStartDate] = useState(
    initialDetail?.sprint.startDate ?? "",
  );
  const [endDate, setEndDate] = useState(initialDetail?.sprint.endDate ?? "");
  const [members, setMembers] = useState<TeamMember[]>([]);
  const [memberIds, setMemberIds] = useState<string[]>(
    initialDetail?.members.map((member) => member.id) ?? [],
  );
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  useEffect(() => {
    const controller = new AbortController();
    void membersGateway
      .list({ search: "", page: 1, pageSize: 100 }, controller.signal)
      .then((page) => setMembers(page.items))
      .catch((reason: unknown) => {
        if (!(reason instanceof DOMException && reason.name === "AbortError"))
          setError("Members could not be loaded. Please try again.");
      });
    return () => controller.abort();
  }, [membersGateway]);

  function validateDetails() {
    const errors: Record<string, string> = {};
    if (!name.trim()) errors.name = "Sprint Name is required.";
    else if ([...name.trim()].length > 200)
      errors.name = "Sprint Name must not exceed 200 characters.";
    if (!startDate) errors.startDate = "Start Date is required.";
    if (!endDate) errors.endDate = "End Date is required.";
    else if (startDate && endDate < startDate)
      errors.endDate = "End Date cannot be before Start Date.";
    setFieldErrors(errors);
    return Object.keys(errors).length === 0;
  }

  async function save() {
    if (loading) return;
    if (!validateDetails()) {
      setStep(1);
      return;
    }
    if (memberIds.length === 0) {
      setFieldErrors({ memberIds: "Select at least one Member." });
      return;
    }
    setLoading(true);
    setError("");
    try {
      const input = {
        name,
        startDate,
        endDate,
        memberIds,
        taskIds: initialDetail?.tasks.map((task) => task.id) ?? [],
      };
      const saved = initialDetail
        ? await gateway.update(initialDetail.sprint.id, {
            ...input,
            version: initialDetail.sprint.version,
          })
        : await gateway.create(input);
      onSaved(saved);
    } catch (reason: unknown) {
      if (
        reason instanceof SprintAPIError &&
        reason.code === "SPRINT_MEMBER_OVERLAP" &&
        reason.details
      ) {
        const conflict = reason.details;
        setError(
          `Conflicts with ${conflict.sprintName} (${conflict.startDate}–${conflict.endDate}) for ${conflict.members.map((member) => member.name).join(", ")}. Your draft has been preserved.`,
        );
      } else {
        setError(
          "The Sprint could not be saved. Your input has been preserved.",
        );
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <Dialog
      titleID="sprint-form-title"
      closeOnBackdrop={false}
      onClose={onClose}
    >
      <h2 id="sprint-form-title" className="text-xl font-black">
        {initialDetail ? `Edit ${initialDetail.sprint.name}` : "Create Sprint"}
      </h2>
      <ol
        className="mt-4 grid grid-cols-2 gap-2"
        aria-label="Sprint creation steps"
      >
        {["Details", "Members"].map((label, index) => (
          <li
            key={label}
            className={`rounded-control border p-2 text-center text-sm font-bold ${step === index + 1 ? "border-brand bg-brand-soft text-brand-strong" : "border-border-subtle text-muted"}`}
            aria-current={step === index + 1 ? "step" : undefined}
          >
            {index + 1}. {label}
          </li>
        ))}
      </ol>
      {error ? (
        <Alert tone="danger" className="mt-4">
          {error}
        </Alert>
      ) : null}
      {step === 1 ? (
        <div className="mt-5 space-y-4">
          <label className="block text-sm font-bold">
            Sprint Name
            <input
              data-autofocus
              className="ui-input mt-2"
              maxLength={200}
              value={name}
              onChange={(event) => setName(event.target.value)}
            />
            {fieldErrors.name ? (
              <span className="mt-1 block text-xs text-danger" role="alert">
                {fieldErrors.name}
              </span>
            ) : null}
          </label>
          <div className="grid gap-4 sm:grid-cols-2">
            <DateField
              label="Start Date"
              value={startDate}
              error={fieldErrors.startDate}
              loadPublicHolidayDates={loadPublicHolidayDates}
              onChange={setStartDate}
            />
            <DateField
              label="End Date"
              value={endDate}
              error={fieldErrors.endDate}
              loadPublicHolidayDates={loadPublicHolidayDates}
              onChange={setEndDate}
            />
          </div>
        </div>
      ) : (
        <fieldset className="mt-5">
          <legend className="text-sm font-bold">Select Members</legend>
          {fieldErrors.memberIds ? (
            <p className="mt-1 text-xs text-danger" role="alert">
              {fieldErrors.memberIds}
            </p>
          ) : null}
          <div className="mt-3 max-h-72 space-y-2 overflow-y-auto">
            {members.map((member) => (
              <label
                key={member.id}
                className="flex items-center gap-3 rounded-control border border-border-subtle p-3"
              >
                <input
                  type="checkbox"
                  checked={memberIds.includes(member.id)}
                  onChange={(event) =>
                    setMemberIds((current) =>
                      event.target.checked
                        ? [...current, member.id]
                        : current.filter((id) => id !== member.id),
                    )
                  }
                />
                <span>
                  <strong>{member.name}</strong>
                  <span className="ml-2 text-sm text-muted">
                    {member.role.name}
                  </span>
                </span>
              </label>
            ))}
          </div>
        </fieldset>
      )}
      <div className="form-actions">
        <Button disabled={loading} onClick={onClose}>
          Cancel
        </Button>
        {step === 2 ? (
          <Button disabled={loading} onClick={() => setStep(1)}>
            Back
          </Button>
        ) : null}
        {step === 1 ? (
          <Button
            variant="primary"
            onClick={() => {
              if (validateDetails()) setStep(2);
            }}
          >
            Next: Members
          </Button>
        ) : (
          <Button
            variant="primary"
            loading={loading}
            onClick={() => void save()}
          >
            {initialDetail ? "Save Sprint" : "Create Sprint"}
          </Button>
        )}
      </div>
    </Dialog>
  );
}

function DateField({
  label,
  value,
  error,
  loadPublicHolidayDates,
  onChange,
}: {
  label: string;
  value: string;
  error?: string;
  loadPublicHolidayDates(startDate: string, endDate: string): Promise<string[]>;
  onChange(value: string): void;
}) {
  return (
    <div>
      <CalendarPopover
        label={label}
        buttonLabel={value ? formatDateOnly(value) : "Select date"}
        initialDate={value}
        instruction={`Select the Sprint ${label.toLowerCase()}.`}
        selectedDates={value ? [value] : []}
        loadPublicHolidayDates={loadPublicHolidayDates}
        onSelect={(date) => {
          onChange(date);
          return true;
        }}
      />
      {error ? (
        <span className="mt-1 block text-xs text-danger" role="alert">
          {error}
        </span>
      ) : null}
    </div>
  );
}
