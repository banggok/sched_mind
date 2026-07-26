import { useCallback, useEffect, useRef, useState } from "react";
import { ListSkeleton } from "../../../shared/presentation/ListSkeleton";
import { PaginationControls } from "../../../shared/presentation/PaginationControls";
import type { TeamMember } from "../../team-members/domain/teamMember";
import {
  createCapacityOverride,
  deleteCapacityOverride,
  listCapacityOverrides,
  updateCapacityOverride,
} from "../application/capacityOverrideManagement";
import type { CapacityOverridesGateway } from "../application/capacityOverridesGateway";
import {
  CapacityOverrideValidationError,
  type CapacityOverride,
} from "../domain/capacityOverride";
import { CapacityOverridesAPIError } from "../infrastructure/httpCapacityOverridesGateway";
import { CalendarPopover } from "../../../shared/presentation/CalendarPopover";
import { Alert } from "../../../shared/presentation/Alert";
import { Button } from "../../../shared/presentation/Button";
import { Dialog } from "../../../shared/presentation/Dialog";
import { EmptyState } from "../../../shared/presentation/EmptyState";

export function CapacityOverridesPanel({
  member,
  gateway,
  onClose,
}: {
  member: TeamMember;
  gateway: CapacityOverridesGateway;
  onClose(): void;
}) {
  const [items, setItems] = useState<CapacityOverride[]>([]),
    [page, setPage] = useState(1),
    [total, setTotal] = useState(0),
    [loading, setLoading] = useState(true),
    [refreshing, setRefreshing] = useState(false),
    [effectiveDate, setEffectiveDate] = useState(""),
    [listError, setListError] = useState(""),
    [editing, setEditing] = useState<CapacityOverride | null | undefined>(),
    [form, setForm] = useState({ startDate: "", endDate: "", capacity: "" }),
    [errors, setErrors] = useState<Record<string, string>>({}),
    [operationError, setOperationError] = useState(""),
    [saving, setSaving] = useState(false),
    [openingID, setOpeningID] = useState(""),
    [deleting, setDeleting] = useState<CapacityOverride>(),
    [success, setSuccess] = useState("");
  const hasLoadedRef = useRef(false);
  const load = useCallback(
    async (target = page, filter = effectiveDate) => {
      if (hasLoadedRef.current) setRefreshing(true);
      else setLoading(true);
      setListError("");
      try {
        const result = await listCapacityOverrides(
          gateway,
          member.id,
          target,
          filter,
        );
        if (result.items.length === 0 && target > 1) {
          setPage(target - 1);
          return;
        }
        setItems(result.items);
        setTotal(result.total);
        hasLoadedRef.current = true;
      } catch {
        setListError(
          "Capacity overrides could not be loaded. Please try again.",
        );
      } finally {
        setLoading(false);
        setRefreshing(false);
      }
    },
    [effectiveDate, gateway, member.id, page],
  );
  useEffect(() => {
    void load(page, effectiveDate);
  }, [effectiveDate, load, page]);
  async function open(value?: CapacityOverride) {
    setErrors({});
    setOperationError("");
    setSuccess("");
    if (!value) {
      setEditing(null);
      setForm({ startDate: "", endDate: "", capacity: "" });
      return;
    }
    setOpeningID(value.id);
    try {
      const detail = await gateway.get(member.id, value.id);
      setEditing(detail);
      setForm({
        startDate: detail.startDate,
        endDate: detail.endDate,
        capacity: String(detail.capacity),
      });
    } catch {
      setOperationError(
        "This capacity override could not be opened. Please try again.",
      );
    } finally {
      setOpeningID("");
    }
  }
  async function save(event: React.FormEvent) {
    event.preventDefault();
    if (saving) return;
    setErrors({});
    setOperationError("");
    const normalizedCapacity = roundToHalfDraft(form.capacity);
    setForm((current) => ({ ...current, capacity: normalizedCapacity }));
    const input = {
      startDate: form.startDate,
      endDate: form.endDate,
      capacity: parseDecimalDraft(normalizedCapacity),
    };
    try {
      setSaving(true);
      if (editing)
        await updateCapacityOverride(gateway, member.id, editing.id, input);
      else await createCapacityOverride(gateway, member.id, input);
      setEditing(undefined);
      setSuccess(
        editing ? "Capacity override updated." : "Capacity override added.",
      );
      await load();
    } catch (error) {
      if (error instanceof CapacityOverrideValidationError)
        setErrors({ [error.field]: error.message });
      else if (error instanceof CapacityOverridesAPIError && error.field)
        setErrors({ [error.field]: errorMessage(error) });
      else setOperationError(errorMessage(error));
    } finally {
      setSaving(false);
    }
  }
  async function confirmDelete() {
    if (!deleting || saving) return;
    setSaving(true);
    setOperationError("");
    try {
      await deleteCapacityOverride(gateway, member.id, deleting.id);
      setDeleting(undefined);
      setSuccess("Capacity override deleted.");
      await load();
    } catch {
      setOperationError(
        "Capacity override could not be deleted. Please try again.",
      );
    } finally {
      setSaving(false);
    }
  }
  return (
    <Dialog
      titleID="capacity-title"
      wide
      onClose={() => {
        if (!saving) onClose();
      }}
    >
      <div className="flex items-start justify-between gap-4">
        <div>
          <p className="text-sm font-bold text-brand">
            Members / {member.name}
          </p>
          <h2 id="capacity-title" className="mt-1 text-2xl font-black">
            Capacity overrides
          </h2>
          <p className="mt-1 text-sm text-muted">
            Temporary daily capacity changes for this member.
          </p>
        </div>
        <Button
          data-autofocus
          aria-label="Close capacity overrides"
          compact
          onClick={onClose}
        >
          Close
        </Button>
      </div>
      {success && (
        <Alert tone="success" className="mt-4">
          {success}
        </Alert>
      )}
      {operationError && (
        <Alert tone="danger" className="mt-4">
          {operationError}
        </Alert>
      )}
      {editing === undefined ? (
        <>
          <div className="mt-6 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
            <div className="flex flex-col gap-2 sm:flex-row sm:items-end">
              <div className="min-w-64">
                <CalendarPopover
                  label="Effective Date"
                  buttonLabel={
                    effectiveDate ? formatDate(effectiveDate) : "Select date"
                  }
                  initialDate={effectiveDate}
                  instruction="Select the date to check."
                  selectedDates={effectiveDate ? [effectiveDate] : []}
                  onSelect={(date) => {
                    setEffectiveDate(date);
                    setPage(1);
                    return true;
                  }}
                />
              </div>
              {effectiveDate ? (
                <Button
                  compact
                  onClick={() => {
                    setEffectiveDate("");
                    setPage(1);
                  }}
                >
                  Clear
                </Button>
              ) : null}
            </div>
            <Button variant="primary" onClick={() => void open()}>
              + Add Override
            </Button>
          </div>
          {refreshing ? (
            <p className="mt-3 text-sm font-bold text-brand" role="status">
              Refreshing capacity overrides…
            </p>
          ) : null}
          {listError && items.length > 0 ? (
            <Alert tone="danger" className="mt-3 text-label">
              {listError}{" "}
              <button className="underline" onClick={() => void load()}>
                Retry
              </button>
            </Alert>
          ) : null}
          <div className="mt-4 overflow-hidden rounded-panel border border-border-subtle">
            {loading ? (
              <ListSkeleton label="Loading capacity overrides" />
            ) : listError && items.length === 0 ? (
              <div className="p-10 text-center">
                <p role="alert">{listError}</p>
                <Button
                  variant="quiet"
                  className="mt-4"
                  onClick={() => void load()}
                >
                  Retry
                </Button>
              </div>
            ) : items.length === 0 && effectiveDate ? (
              <EmptyState
                title="No matching capacity override"
                description={`No capacity override applies on ${formatDate(effectiveDate)}.`}
                action={
                  <Button
                    variant="quiet"
                    onClick={() => {
                      setEffectiveDate("");
                      setPage(1);
                    }}
                  >
                    Clear Effective Date
                  </Button>
                }
              />
            ) : items.length === 0 ? (
              <EmptyState
                title="No capacity overrides yet"
                description="This member uses their base Daily Capacity until an override is added."
                action={
                  <Button variant="quiet" onClick={() => void open()}>
                    Add Override
                  </Button>
                }
              />
            ) : (
              <ul className="divide-y divide-border-subtle">
                {items.map((value) => (
                  <li
                    key={value.id}
                    className="flex flex-col gap-3 px-5 py-4 sm:flex-row sm:items-center sm:justify-between"
                  >
                    <div>
                      <strong>
                        {formatDate(value.startDate)} —{" "}
                        {formatDate(value.endDate)}
                      </strong>
                      <p className="text-sm text-muted">
                        {value.capacity} hours/day
                      </p>
                    </div>
                    <div className="flex gap-2">
                      <Button
                        compact
                        disabled={openingID === value.id}
                        loading={openingID === value.id}
                        onClick={() => void open(value)}
                      >
                        {openingID === value.id ? "Loading…" : "Edit"}
                      </Button>
                      <Button
                        compact
                        variant="danger"
                        disabled={openingID !== ""}
                        onClick={() => setDeleting(value)}
                      >
                        Delete
                      </Button>
                    </div>
                  </li>
                ))}
              </ul>
            )}
            {!loading && !listError && total > 0 ? (
              <PaginationControls
                page={page}
                pageSize={5}
                total={total}
                onPageChange={setPage}
              />
            ) : null}
          </div>
        </>
      ) : (
        <form
          className="mt-6 rounded-panel border border-border-subtle p-5"
          onSubmit={save}
          noValidate
        >
          <h3 className="text-lg font-black">
            {editing ? "Edit override" : "Add override"}
          </h3>
          <div className="mt-4 grid gap-4 sm:grid-cols-2">
            <DateRangePicker
              startDate={form.startDate}
              endDate={form.endDate}
              startError={errors.startDate}
              endError={errors.endDate}
              onChange={(startDate, endDate) =>
                setForm({ ...form, startDate, endDate })
              }
            />
            <Field label="Capacity (hours)" error={errors.capacity}>
              <input
                required
                type="text"
                inputMode="decimal"
                className="ui-input mt-2"
                value={form.capacity}
                onChange={(e) => {
                  if (isDecimalDraft(e.target.value))
                    setForm({ ...form, capacity: e.target.value });
                }}
                onBlur={() =>
                  setForm((current) => ({
                    ...current,
                    capacity: roundToHalfDraft(current.capacity),
                  }))
                }
              />
            </Field>
          </div>
          <div className="form-actions">
            <Button
              type="button"
              disabled={saving}
              onClick={() => setEditing(undefined)}
            >
              Cancel
            </Button>
            <Button variant="primary" loading={saving}>
              {saving ? "Saving…" : "Save"}
            </Button>
          </div>
        </form>
      )}
      {deleting && (
        <DeleteOverrideDialog
          value={deleting}
          saving={saving}
          onCancel={() => setDeleting(undefined)}
          onConfirm={() => void confirmDelete()}
        />
      )}
    </Dialog>
  );
}
function Field({
  label,
  error,
  children,
}: {
  label: string;
  error?: string;
  children: React.ReactNode;
}) {
  return (
    <label className="block text-sm font-bold">
      {label}
      {children}
      {error && (
        <span role="alert" className="mt-1 block text-xs text-danger">
          {error}
        </span>
      )}
    </label>
  );
}
function formatDate(value: string) {
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeZone: "UTC",
  }).format(new Date(`${value}T00:00:00Z`));
}
function errorMessage(error: unknown) {
  if (typeof error === "object" && error && "code" in error) {
    const code = String(error.code);
    if (code === "CAPACITY_OVERRIDE_OVERLAPS")
      return "This period overlaps an existing capacity override.";
    if (code === "TEAM_MEMBER_NOT_FOUND")
      return "This member is no longer available.";
    if (code === "CAPACITY_OVERRIDE_NOT_FOUND")
      return "This capacity override is no longer available.";
  }
  return "The capacity override could not be saved. Your input has been preserved.";
}
function DeleteOverrideDialog({
  value,
  saving,
  onCancel,
  onConfirm,
}: {
  value: CapacityOverride;
  saving: boolean;
  onCancel(): void;
  onConfirm(): void;
}) {
  return (
    <Dialog
      titleID="delete-override-title"
      kind="alertdialog"
      nested
      closeOnBackdrop={false}
      onClose={() => {
        if (!saving) onCancel();
      }}
    >
      <h3 id="delete-override-title" className="text-xl font-black">
        Delete capacity override?
      </h3>
      <p className="mt-3 text-muted">
        Delete {formatDate(value.startDate)} — {formatDate(value.endDate)} at{" "}
        {value.capacity} hours/day permanently.
      </p>
      <div className="form-actions">
        <Button data-autofocus disabled={saving} onClick={onCancel}>
          Cancel
        </Button>
        <Button variant="danger-solid" loading={saving} onClick={onConfirm}>
          {saving ? "Deleting…" : "Delete"}
        </Button>
      </div>
    </Dialog>
  );
}

function DateRangePicker({
  startDate,
  endDate,
  startError,
  endError,
  onChange,
}: {
  startDate: string;
  endDate: string;
  startError?: string;
  endError?: string;
  onChange(startDate: string, endDate: string): void;
}) {
  const waitingForEnd = startDate !== "" && endDate === "";
  function select(value: string) {
    if (!startDate || endDate) {
      onChange(value, "");
      return false;
    }
    if (value < startDate) {
      onChange(value, "");
      return false;
    }
    onChange(startDate, value);
    return true;
  }
  const label = startDate
    ? endDate
      ? `${formatDate(startDate)} — ${formatDate(endDate)}`
      : `${formatDate(startDate)} — Select end date`
    : "Select start and end date";
  return (
    <div>
      <CalendarPopover
        label="Date range"
        buttonLabel={label}
        initialDate={startDate}
        instruction={
          waitingForEnd
            ? "Select an end date. Choose an earlier date to replace the start date."
            : "Select a start date."
        }
        selectedDates={[startDate, endDate].filter(Boolean)}
        isInRange={(date) =>
          Boolean(startDate && endDate && date > startDate && date < endDate)
        }
        onSelect={select}
      />
      {(startError || endError) && (
        <span role="alert" className="mt-1 block text-xs text-danger">
          {startError ?? endError}
        </span>
      )}
    </div>
  );
}
function parseDecimalDraft(value: string): number | undefined {
  const normalized = value.trim();
  if (!normalized) return undefined;
  const parsed = Number(normalized);
  return Number.isFinite(parsed) ? parsed : Number.NaN;
}
function roundToHalfDraft(value: string): string {
  const parsed = parseDecimalDraft(value);
  if (parsed === undefined || !Number.isFinite(parsed)) return value;
  return String(Math.round(parsed * 2) / 2);
}
function isDecimalDraft(value: string): boolean {
  return /^\d*(?:\.\d*)?$/.test(value);
}
