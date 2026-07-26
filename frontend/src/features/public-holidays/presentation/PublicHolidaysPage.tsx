import { useCallback, useEffect, useRef, useState } from "react";
import { Breadcrumb } from "../../../app/Breadcrumb";
import { PageContent } from "../../../app/PageContent";
import { Alert } from "../../../shared/presentation/Alert";
import { Button } from "../../../shared/presentation/Button";
import { CalendarPopover } from "../../../shared/presentation/CalendarPopover";
import { Dialog } from "../../../shared/presentation/Dialog";
import { EmptyState } from "../../../shared/presentation/EmptyState";
import { ListSkeleton } from "../../../shared/presentation/ListSkeleton";
import { ListSurface } from "../../../shared/presentation/ListSurface";
import { PaginationControls } from "../../../shared/presentation/PaginationControls";
import { Toast } from "../../../shared/presentation/Toast";
import {
  createPublicHoliday,
  deletePublicHoliday,
  listPublicHolidays,
  updatePublicHoliday,
} from "../application/publicHolidayManagement";
import type { PublicHolidaysGateway } from "../application/publicHolidaysGateway";
import {
  PublicHolidayValidationError,
  type PublicHoliday,
} from "../domain/publicHoliday";
import { PublicHolidaysAPIError } from "../infrastructure/httpPublicHolidaysGateway";

export function PublicHolidaysPage({
  gateway,
}: {
  gateway: PublicHolidaysGateway;
}) {
  const [items, setItems] = useState<PublicHoliday[]>([]),
    [page, setPage] = useState(1),
    [total, setTotal] = useState(0),
    [holidayDate, setHolidayDate] = useState(""),
    [loading, setLoading] = useState(true),
    [refreshing, setRefreshing] = useState(false),
    [listError, setListError] = useState(""),
    [form, setForm] = useState<{
      mode: "create" | "edit";
      holiday?: PublicHoliday;
      startDate: string;
      endDate: string;
      description: string;
    }>(),
    [deleting, setDeleting] = useState<PublicHoliday>(),
    [errors, setErrors] = useState<Record<string, string>>({}),
    [operationError, setOperationError] = useState(""),
    [submitting, setSubmitting] = useState(false),
    [notification, setNotification] = useState("");
  const loaded = useRef(false),
    requestID = useRef(0);
  const load = useCallback(
    async (target = page, filter = holidayDate) => {
      const current = ++requestID.current;
      if (loaded.current) setRefreshing(true);
      else setLoading(true);
      setListError("");
      try {
        const result = await listPublicHolidays(gateway, target, filter);
        if (current !== requestID.current) return;
        if (result.items.length === 0 && target > 1) {
          setPage(Math.max(1, Math.ceil(result.total / 5)));
          return;
        }
        setItems(result.items);
        setTotal(result.total);
        loaded.current = true;
      } catch {
        if (current === requestID.current)
          setListError(
            "Public holidays could not be loaded. Please try again.",
          );
      } finally {
        if (current === requestID.current) {
          setLoading(false);
          setRefreshing(false);
        }
      }
    },
    [gateway, holidayDate, page],
  );
  useEffect(() => {
    void load(page, holidayDate);
  }, [holidayDate, load, page]);
  function openCreate() {
    setErrors({});
    setOperationError("");
    setForm({ mode: "create", startDate: "", endDate: "", description: "" });
  }
  async function openEdit(value: PublicHoliday) {
    setErrors({});
    setOperationError("");
    try {
      const detail = await gateway.get(value.id);
      setForm({
        mode: "edit",
        holiday: detail,
        startDate: detail.startDate,
        endDate: detail.endDate,
        description: detail.description,
      });
    } catch {
      setOperationError(
        "This public holiday could not be opened. Please try again.",
      );
    }
  }
  async function save() {
    if (!form || submitting) return;
    setErrors({});
    setOperationError("");
    try {
      setSubmitting(true);
      if (form.mode === "edit" && form.holiday)
        await updatePublicHoliday(gateway, form.holiday.id, {
          startDate: form.startDate,
          endDate: form.endDate,
          description: form.description,
        });
      else
        await createPublicHoliday(gateway, {
          startDate: form.startDate,
          endDate: form.endDate,
          description: form.description,
        });
      setNotification(
        form.mode === "edit"
          ? "Public holiday updated."
          : "Public holiday added.",
      );
      setForm(undefined);
      await load();
    } catch (error) {
      if (error instanceof PublicHolidayValidationError)
        setErrors({ [error.field]: error.message });
      else if (error instanceof PublicHolidaysAPIError && error.field)
        setErrors({ [error.field]: errorMessage(error) });
      else setOperationError(errorMessage(error));
    } finally {
      setSubmitting(false);
    }
  }
  async function confirmDelete() {
    if (!deleting || submitting) return;
    setSubmitting(true);
    setOperationError("");
    try {
      await deletePublicHoliday(gateway, deleting.id);
      setDeleting(undefined);
      setNotification("Public holiday deleted.");
      await load();
    } catch (error) {
      setOperationError(
        error instanceof PublicHolidaysAPIError &&
          error.code === "PUBLIC_HOLIDAY_NOT_FOUND"
          ? "This public holiday is no longer available."
          : "The public holiday could not be deleted. Please try again.",
      );
    } finally {
      setSubmitting(false);
    }
  }
  return (
    <>
      <PageContent>
        <section className="min-w-0">
          <div className="page-header">
            <Breadcrumb activePage="public-holidays" />
            <Button variant="primary" className="shrink-0" onClick={openCreate}>
              + Add Public Holiday
            </Button>
          </div>
          <ListSurface
            title="Public Holidays"
            controls={
              <div className="flex flex-col gap-2 sm:flex-row sm:items-end">
                <div className="min-w-64">
                  <CalendarPopover
                    label="Holiday Date"
                    buttonLabel={
                      holidayDate ? formatDate(holidayDate) : "Select date"
                    }
                    initialDate={holidayDate}
                    instruction="Select the holiday date to check."
                    selectedDates={holidayDate ? [holidayDate] : []}
                    loadPublicHolidayDates={gateway.calendar}
                    onSelect={(date) => {
                      setHolidayDate(date);
                      setPage(1);
                      return true;
                    }}
                  />
                </div>
                {holidayDate ? (
                  <Button
                    compact
                    onClick={() => {
                      setHolidayDate("");
                      setPage(1);
                    }}
                  >
                    Clear
                  </Button>
                ) : null}
              </div>
            }
          >
            {refreshing ? (
              <p
                className="px-6 pt-3 text-sm font-bold text-brand"
                role="status"
              >
                Refreshing public holidays…
              </p>
            ) : null}
            {listError && items.length > 0 ? (
              <Alert tone="danger" className="m-4">
                {listError}{" "}
                <button className="underline" onClick={() => void load()}>
                  Retry
                </button>
              </Alert>
            ) : null}
            {operationError ? (
              <Alert tone="danger" className="m-4">
                {operationError}
              </Alert>
            ) : null}
            {loading ? (
              <ListSkeleton label="Loading public holidays" />
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
            ) : items.length === 0 && holidayDate ? (
              <EmptyState
                title="No public holiday is configured"
                description={`No public holiday is configured for ${formatDate(holidayDate)}.`}
                action={
                  <Button
                    variant="quiet"
                    onClick={() => {
                      setHolidayDate("");
                      setPage(1);
                    }}
                  >
                    Clear Holiday Date
                  </Button>
                }
              />
            ) : items.length === 0 ? (
              <EmptyState
                title="No public holidays yet"
                description="The scheduler currently has no configured global holidays."
                action={
                  <Button variant="quiet" onClick={openCreate}>
                    Add Public Holiday
                  </Button>
                }
              />
            ) : (
              <ul className="divide-y divide-border-subtle">
                {items.map((value) => (
                  <li
                    key={value.id}
                    className="flex flex-col gap-4 px-6 py-5 sm:flex-row sm:items-center sm:justify-between"
                  >
                    <div>
                      <h3 className="font-extrabold">{value.description}</h3>
                      <p className="mt-1 text-sm text-muted">
                        {formatRange(value.startDate, value.endDate)}
                      </p>
                    </div>
                    <div className="flex gap-2 self-end sm:self-auto">
                      <Button
                        compact
                        aria-label={`Edit ${value.description}`}
                        onClick={() => void openEdit(value)}
                      >
                        Edit
                      </Button>
                      <Button
                        compact
                        variant="danger"
                        aria-label={`Delete ${value.description}`}
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
          </ListSurface>
        </section>
      </PageContent>
      {form ? (
        <HolidayForm
          form={form}
          loadPublicHolidayDates={gateway.calendar}
          errors={errors}
          operationError={operationError}
          submitting={submitting}
          onChange={(field, value) => {
            setErrors((currentErrors) => {
              const nextErrors = { ...currentErrors };
              if (field === "startDate" || field === "endDate") {
                delete nextErrors.startDate;
                delete nextErrors.endDate;
              } else {
                delete nextErrors[field];
              }
              return nextErrors;
            });
            setForm((current) =>
              current ? { ...current, [field]: value } : current,
            );
          }}
          onClose={() => {
            if (!submitting) setForm(undefined);
          }}
          onSubmit={() => void save()}
        />
      ) : null}
      {deleting ? (
        <DeleteDialog
          holiday={deleting}
          error={operationError}
          submitting={submitting}
          onCancel={() => {
            if (!submitting) setDeleting(undefined);
          }}
          onConfirm={() => void confirmDelete()}
        />
      ) : null}
      <Toast message={notification} onDismiss={() => setNotification("")} />
    </>
  );
}

function HolidayForm({
  form,
  loadPublicHolidayDates,
  errors,
  operationError,
  submitting,
  onChange,
  onClose,
  onSubmit,
}: {
  form: {
    mode: "create" | "edit";
    startDate: string;
    endDate: string;
    description: string;
  };
  loadPublicHolidayDates(startDate: string, endDate: string): Promise<string[]>;
  errors: Record<string, string>;
  operationError: string;
  submitting: boolean;
  onChange(field: "startDate" | "endDate" | "description", value: string): void;
  onClose(): void;
  onSubmit(): void;
}) {
  return (
    <Dialog titleID="holiday-form-title" onClose={onClose}>
      <form
        onSubmit={(event) => {
          event.preventDefault();
          onSubmit();
        }}
        noValidate
      >
        <h2 id="holiday-form-title" className="text-xl font-black">
          {form.mode === "edit" ? "Edit public holiday" : "Add public holiday"}
        </h2>
        {operationError ? (
          <Alert tone="danger" className="mt-4">
            {operationError}
          </Alert>
        ) : null}
        <div className="mt-5">
          <CalendarPopover
            label="Date range"
            buttonLabel={
              form.startDate
                ? formatRange(form.startDate, form.endDate)
                : "Select start and end date"
            }
            initialDate={form.startDate}
            instruction={
              form.startDate && !form.endDate
                ? "Select an end date."
                : "Select a start date."
            }
            selectedDates={[form.startDate, form.endDate].filter(Boolean)}
            loadPublicHolidayDates={loadPublicHolidayDates}
            isInRange={(date) =>
              Boolean(
                form.startDate &&
                form.endDate &&
                date >= form.startDate &&
                date <= form.endDate,
              )
            }
            onSelect={(date) => {
              if (!form.startDate || form.endDate || date < form.startDate) {
                onChange("startDate", date);
                onChange("endDate", "");
                return false;
              }
              onChange("endDate", date);
              return true;
            }}
          />
          {errors.startDate || errors.endDate ? (
            <p className="mt-1 text-xs text-danger" role="alert">
              {errors.startDate || errors.endDate}
            </p>
          ) : null}
        </div>
        <label className="mt-4 block text-sm font-bold">
          Description
          <input
            data-autofocus
            className="ui-input mt-2"
            maxLength={100}
            value={form.description}
            onChange={(event) => onChange("description", event.target.value)}
          />
          {errors.description ? (
            <span className="mt-1 block text-xs text-danger" role="alert">
              {errors.description}
            </span>
          ) : null}
        </label>
        <div className="form-actions">
          <Button type="button" disabled={submitting} onClick={onClose}>
            Cancel
          </Button>
          <Button variant="primary" loading={submitting}>
            {submitting ? "Saving…" : "Save"}
          </Button>
        </div>
      </form>
    </Dialog>
  );
}
function DeleteDialog({
  holiday,
  error,
  submitting,
  onCancel,
  onConfirm,
}: {
  holiday: PublicHoliday;
  error: string;
  submitting: boolean;
  onCancel(): void;
  onConfirm(): void;
}) {
  return (
    <Dialog
      titleID="delete-holiday-title"
      kind="alertdialog"
      nested
      closeOnBackdrop={false}
      onClose={onCancel}
    >
      <h2 id="delete-holiday-title" className="text-xl font-black">
        Delete public holiday?
      </h2>
      <p className="mt-3 text-muted">
        Delete {holiday.description} (
        {formatRange(holiday.startDate, holiday.endDate)}) permanently.
      </p>
      {error ? (
        <Alert tone="danger" className="mt-4">
          {error}
        </Alert>
      ) : null}
      <div className="form-actions">
        <Button data-autofocus disabled={submitting} onClick={onCancel}>
          Cancel
        </Button>
        <Button variant="danger-solid" loading={submitting} onClick={onConfirm}>
          {submitting ? "Deleting…" : "Delete"}
        </Button>
      </div>
    </Dialog>
  );
}
function formatDate(value: string) {
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeZone: "UTC",
  }).format(new Date(`${value}T00:00:00Z`));
}
function formatRange(start: string, end: string) {
  if (!end || start === end) return formatDate(start);
  return `${formatDate(start)} – ${formatDate(end)}`;
}
function errorMessage(error: unknown) {
  if (error instanceof PublicHolidaysAPIError) {
    if (error.code === "PUBLIC_HOLIDAY_DATE_ALREADY_EXISTS")
      return "One or more weekdays in this range already have a public holiday.";
    if (error.code === "PUBLIC_HOLIDAY_NOT_FOUND")
      return "This public holiday is no longer available.";
  }
  return "The public holiday could not be saved. Your input has been preserved.";
}
