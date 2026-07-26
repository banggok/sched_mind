import type { MouseEvent, ReactNode } from "react";
import { useDialogFocus } from "./useDialogFocus";

export function Dialog({
  titleID,
  onClose,
  children,
  kind = "dialog",
  wide = false,
  nested = false,
  closeOnBackdrop = true,
}: {
  titleID: string;
  onClose(): void;
  children: ReactNode;
  kind?: "dialog" | "alertdialog";
  wide?: boolean;
  nested?: boolean;
  closeOnBackdrop?: boolean;
}) {
  const dialogRef = useDialogFocus(onClose);
  function backdrop(event: MouseEvent<HTMLDivElement>) {
    if (closeOnBackdrop && event.currentTarget === event.target) onClose();
  }
  return (
    <div
      className={`dialog-overlay ${nested ? "dialog-overlay-nested" : ""}`}
      role="presentation"
      onMouseDown={backdrop}
    >
      <section
        ref={dialogRef}
        className={`dialog-panel ${wide ? "dialog-panel-wide" : ""}`}
        role={kind}
        aria-modal="true"
        aria-labelledby={titleID}
      >
        {children}
      </section>
    </div>
  );
}
