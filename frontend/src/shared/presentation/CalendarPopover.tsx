import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { Button } from "./Button";

export function CalendarPopover({
  label,
  buttonLabel,
  disabled = false,
  initialDate,
  instruction,
  selectedDates,
  isInRange,
  loadPublicHolidayDates,
  onSelect,
}: {
  label: string;
  buttonLabel: string;
  disabled?: boolean;
  initialDate?: string;
  instruction: string;
  selectedDates: string[];
  isInRange?(date: string): boolean;
  loadPublicHolidayDates?(
    startDate: string,
    endDate: string,
  ): Promise<string[]>;
  onSelect(date: string): boolean;
}) {
  const initial = initialDate
    ? new Date(`${initialDate}T00:00:00Z`)
    : new Date();
  const [open, setOpen] = useState(false);
  const [month, setMonth] = useState(
    new Date(Date.UTC(initial.getUTCFullYear(), initial.getUTCMonth(), 1)),
  );
  const [placement, setPlacement] = useState<"above" | "below">("below");
  const [alignment, setAlignment] = useState<"left" | "right">("left");
  const [maxHeight, setMaxHeight] = useState<number>();
  const [coordinates, setCoordinates] = useState({ top: 0, left: 0 });
  const [publicHolidayDates, setPublicHolidayDates] = useState<string[]>([]);
  const rootRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const popoverRef = useRef<HTMLDivElement>(null);

  useLayoutEffect(() => {
    if (!open) return;
    const trigger = triggerRef.current?.getBoundingClientRect();
    const popover = popoverRef.current?.getBoundingClientRect();
    if (!trigger || !popover) return;
    const margin = 8;
    const popoverWidth = popover.width;
    const left = Math.min(
      Math.max(margin, trigger.left),
      window.innerWidth - popoverWidth - margin,
    );
    setAlignment(left < trigger.left ? "right" : "left");
    const spaceBelow = window.innerHeight - trigger.bottom;
    const spaceAbove = trigger.top;
    const requiredSpace =
      Math.max(popoverRef.current?.scrollHeight ?? 0, popover.height) + margin;
    if (spaceBelow >= requiredSpace) {
      setCoordinates({ top: trigger.bottom + margin, left });
      return;
    }
    if (spaceAbove >= requiredSpace) {
      setPlacement("above");
      setCoordinates({ top: trigger.top - requiredSpace, left });
      return;
    }
    const availableHeight = Math.max(spaceBelow - margin, 1);
    setMaxHeight(availableHeight);
    setCoordinates({ top: trigger.bottom + margin, left });
  }, [open]);

  function setCalendarOpen(nextOpen: boolean) {
    if (nextOpen) {
      setPlacement("below");
      setAlignment("left");
      setMaxHeight(undefined);
      setCoordinates({ top: 0, left: 0 });
    }
    setOpen(nextOpen);
  }

  useEffect(() => {
    if (!open) return;
    function pointerdown(event: PointerEvent) {
      if (
        event.target instanceof Node &&
        !rootRef.current?.contains(event.target) &&
        !popoverRef.current?.contains(event.target)
      ) {
        setCalendarOpen(false);
      }
    }
    function keydown(event: KeyboardEvent) {
      if (event.key !== "Escape") return;
      event.preventDefault();
      setCalendarOpen(false);
      triggerRef.current?.focus();
    }
    document.addEventListener("pointerdown", pointerdown);
    document.addEventListener("keydown", keydown);
    return () => {
      document.removeEventListener("pointerdown", pointerdown);
      document.removeEventListener("keydown", keydown);
    };
  }, [open]);

  useEffect(() => {
    if (!open || !loadPublicHolidayDates) return;
    let active = true;
    const year = month.getUTCFullYear();
    const monthIndex = month.getUTCMonth();
    const start = `${year}-${String(monthIndex + 1).padStart(2, "0")}-01`;
    const end = `${year}-${String(monthIndex + 1).padStart(2, "0")}-${String(new Date(Date.UTC(year, monthIndex + 1, 0)).getUTCDate()).padStart(2, "0")}`;
    void loadPublicHolidayDates(start, end)
      .then((dates) => {
        if (active) setPublicHolidayDates(dates);
      })
      .catch(() => {
        if (active) setPublicHolidayDates([]);
      });
    return () => {
      active = false;
    };
  }, [loadPublicHolidayDates, month, open]);

  return (
    <div ref={rootRef} className="relative">
      <span className="block text-sm font-bold">{label}</span>
      <button
        ref={triggerRef}
        type="button"
        aria-label={`${label}: ${buttonLabel}`}
        aria-haspopup="dialog"
        aria-expanded={open}
        disabled={disabled}
        className="mt-2 w-full rounded-control border border-border-strong bg-surface p-3 text-left"
        onClick={() => !disabled && setCalendarOpen(!open)}
      >
        {buttonLabel}
      </button>
      {open
        ? createPortal(
            <div
              ref={popoverRef}
              role="dialog"
              aria-label={`Choose ${label.toLocaleLowerCase()}`}
              data-placement={placement}
              data-alignment={alignment}
              data-scrollable={maxHeight === undefined ? undefined : "true"}
              style={{
                top: coordinates.top,
                left: coordinates.left,
                maxHeight,
                overflowY: maxHeight === undefined ? undefined : "auto",
              }}
              className="calendar-popover layer-dialog-popover fixed rounded-surface border border-border-strong bg-surface p-4 shadow-floating"
            >
              <div className="flex items-center justify-between">
                <Button
                  compact
                  aria-label="Previous month"
                  onClick={() => setMonth(addMonths(month, -1))}
                >
                  ‹
                </Button>
                <strong>
                  {new Intl.DateTimeFormat(undefined, {
                    month: "long",
                    year: "numeric",
                    timeZone: "UTC",
                  }).format(month)}
                </strong>
                <Button
                  compact
                  aria-label="Next month"
                  onClick={() => setMonth(addMonths(month, 1))}
                >
                  ›
                </Button>
              </div>
              <p className="mt-3 text-sm text-muted" role="status">
                {instruction}
              </p>
              <div className="mt-3 grid grid-cols-7 text-center text-xs font-bold text-muted">
                {["Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"].map((day) => (
                  <span key={day}>{day}</span>
                ))}
              </div>
              <div className="mt-2 grid grid-cols-7 gap-1">
                {calendarDays(month).map((day, index) =>
                  day ? (
                    <button
                      type="button"
                      key={day}
                      aria-label={day}
                      title={dateStatus(day, publicHolidayDates)}
                      aria-pressed={selectedDates.includes(day)}
                      className={`min-h-10 rounded-control text-sm ${
                        selectedDates.includes(day)
                          ? "bg-brand font-bold text-on-brand"
                          : publicHolidayDates.includes(day)
                            ? "bg-danger-soft text-danger"
                            : isWeekend(day)
                              ? "bg-danger-soft text-danger"
                              : isInRange?.(day)
                                ? "bg-brand-soft"
                                : "hover:bg-brand-soft"
                      }`}
                      onClick={() => {
                        if (onSelect(day)) setCalendarOpen(false);
                      }}
                    >
                      {Number(day.slice(-2))}
                    </button>
                  ) : (
                    <span key={`empty-${index}`} />
                  ),
                )}
              </div>
            </div>,
            document.body,
          )
        : null}
    </div>
  );
}

function isWeekend(value: string) {
  const day = new Date(`${value}T00:00:00Z`).getUTCDay();
  return day === 0 || day === 6;
}
function dateStatus(value: string, publicHolidays: string[]) {
  if (publicHolidays.includes(value) || isWeekend(value)) return "Holiday";
  return undefined;
}

function calendarDays(month: Date): (string | null)[] {
  const year = month.getUTCFullYear();
  const monthIndex = month.getUTCMonth();
  const firstDay = new Date(Date.UTC(year, monthIndex, 1)).getUTCDay();
  const count = new Date(Date.UTC(year, monthIndex + 1, 0)).getUTCDate();
  return [
    ...Array(firstDay).fill(null),
    ...Array.from(
      { length: count },
      (_, index) =>
        `${year}-${String(monthIndex + 1).padStart(2, "0")}-${String(index + 1).padStart(2, "0")}`,
    ),
  ];
}

function addMonths(month: Date, amount: number) {
  return new Date(
    Date.UTC(month.getUTCFullYear(), month.getUTCMonth() + amount, 1),
  );
}
