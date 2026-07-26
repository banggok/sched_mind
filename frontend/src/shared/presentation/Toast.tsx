import { useEffect, useRef } from "react";

const DEFAULT_DISMISS_DELAY = 5000;

export function Toast({
  message,
  onDismiss,
  dismissDelay = DEFAULT_DISMISS_DELAY,
}: {
  message: string;
  onDismiss(): void;
  dismissDelay?: number;
}) {
  const onDismissRef = useRef(onDismiss);
  useEffect(() => {
    onDismissRef.current = onDismiss;
  }, [onDismiss]);
  useEffect(() => {
    if (!message) return;
    const timer = window.setTimeout(() => onDismissRef.current(), dismissDelay);
    return () => window.clearTimeout(timer);
  }, [dismissDelay, message]);

  if (!message) return null;
  return (
    <div
      className="layer-notification fixed right-5 bottom-5 flex max-w-sm items-center gap-4 rounded-panel bg-overlay px-5 py-4 text-sm font-bold text-on-brand shadow-floating"
      role="status"
    >
      <span>{message}</span>
      <button
        type="button"
        className="grid size-8 shrink-0 place-items-center rounded-action text-lg hover:bg-surface/10 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white"
        aria-label="Dismiss notification"
        onClick={onDismiss}
      >
        ×
      </button>
    </div>
  );
}
