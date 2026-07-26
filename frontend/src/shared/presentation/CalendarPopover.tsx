import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { Button } from "./Button";

export function CalendarPopover({
  label,
  buttonLabel,
  initialDate,
  instruction,
  selectedDates,
  isInRange,
  onSelect,
}: {
  label: string;
  buttonLabel: string;
  initialDate?: string;
  instruction: string;
  selectedDates: string[];
  isInRange?(date: string): boolean;
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
  const [maxHeight, setMaxHeight] = useState<number>();
  const rootRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const popoverRef = useRef<HTMLDivElement>(null);

  useLayoutEffect(() => {
    if (!open) return;
    const trigger = triggerRef.current?.getBoundingClientRect();
    const popover = popoverRef.current?.getBoundingClientRect();
    if (!trigger || !popover) return;
    const spaceBelow = window.innerHeight - trigger.bottom;
    const spaceAbove = trigger.top;
    const requiredSpace =
      Math.max(popoverRef.current?.scrollHeight ?? 0, popover.height) + 8;
    if (spaceBelow >= requiredSpace) return;
    if (spaceAbove >= requiredSpace) {
      setPlacement("above");
      return;
    }
    setMaxHeight(Math.max(spaceBelow - 8, 1));
  }, [open]);

  function setCalendarOpen(nextOpen: boolean) {
    if (nextOpen) {
      setPlacement("below");
      setMaxHeight(undefined);
    }
    setOpen(nextOpen);
  }

  useEffect(() => {
    if (!open) return;
    function pointerdown(event: PointerEvent) {
      if (
        event.target instanceof Node &&
        !rootRef.current?.contains(event.target)
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

  return (
    <div ref={rootRef} className="relative">
      <span className="block text-sm font-bold">{label}</span>
      <button
        ref={triggerRef}
        type="button"
        aria-label={`${label}: ${buttonLabel}`}
        aria-haspopup="dialog"
        aria-expanded={open}
        className="mt-2 w-full rounded-control border border-border-strong bg-surface p-3 text-left"
        onClick={() => setCalendarOpen(!open)}
      >
        {buttonLabel}
      </button>
      {open ? (
        <div
          ref={popoverRef}
          role="dialog"
          aria-label={`Choose ${label.toLocaleLowerCase()}`}
          data-placement={placement}
          data-scrollable={maxHeight === undefined ? undefined : "true"}
          style={
            maxHeight === undefined
              ? undefined
              : { maxHeight, overflowY: "auto" }
          }
          className={`calendar-popover layer-popover absolute rounded-surface border border-border-strong bg-surface p-4 shadow-floating ${
            placement === "above" ? "bottom-full mb-2" : "mt-2"
          }`}
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
                  aria-pressed={selectedDates.includes(day)}
                  className={`min-h-10 rounded-control text-sm ${
                    selectedDates.includes(day)
                      ? "bg-brand font-bold text-on-brand"
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
        </div>
      ) : null}
    </div>
  );
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
