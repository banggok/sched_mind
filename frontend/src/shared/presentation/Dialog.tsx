import type { MouseEvent, ReactNode } from "react";
import { createPortal } from "react-dom";
import { useDialogFocus } from "./useDialogFocus";

export function Dialog({
  titleID,
  descriptionID,
  onClose,
  children,
  kind = "dialog",
  wide = false,
  nested = false,
  closeOnBackdrop = true,
}: {
  titleID: string;
  descriptionID?: string;
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
  const content = (
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
        aria-describedby={descriptionID}
      >
        {children}
      </section>
    </div>
  );

  return nested && typeof document !== "undefined"
    ? createPortal(content, document.body)
    : content;
}
