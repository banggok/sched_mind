import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type CSSProperties,
  type MouseEvent,
  type ReactNode,
} from "react";
import { createPortal } from "react-dom";
import { useDialogFocus } from "./useDialogFocus";

type DialogStack = {
  depth: number;
  registerChild(): () => void;
};

const DialogStackContext = createContext<DialogStack>({
  depth: -1,
  registerChild: () => () => undefined,
});

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
  const parentStack = useContext(DialogStackContext);
  const depth = nested ? parentStack.depth + 1 : 0;
  const [openChildren, setOpenChildren] = useState(0);
  const registerChild = useCallback(() => {
    setOpenChildren((value) => value + 1);
    let registered = true;
    return () => {
      if (!registered) return;
      registered = false;
      setOpenChildren((value) => Math.max(0, value - 1));
    };
  }, []);
  const stack = useMemo(
    () => ({ depth, registerChild }),
    [depth, registerChild],
  );
  useEffect(() => {
    if (!nested) return undefined;
    return parentStack.registerChild();
  }, [nested, parentStack]);
  const active = openChildren === 0;
  const overlayStyle = nested
    ? ({
        "--dialog-layer-offset": Math.max(depth - 1, 0),
      } as CSSProperties)
    : undefined;
  const dialogRef = useDialogFocus(onClose);
  function backdrop(event: MouseEvent<HTMLDivElement>) {
    if (closeOnBackdrop && event.currentTarget === event.target) onClose();
  }
  const content = (
    <div
      className={`dialog-overlay ${nested ? "dialog-overlay-nested" : ""}`}
      data-dialog-depth={depth}
      style={overlayStyle}
      role="presentation"
      onMouseDown={backdrop}
    >
      <section
        ref={dialogRef}
        className={`dialog-panel ${wide ? "dialog-panel-wide" : ""}`}
        role={kind}
        aria-modal={active ? "true" : undefined}
        aria-hidden={active ? undefined : true}
        data-dialog-active={active ? "true" : "false"}
        aria-labelledby={titleID}
        aria-describedby={descriptionID}
      >
        {children}
      </section>
    </div>
  );

  const rendered =
    nested && typeof document !== "undefined"
      ? createPortal(content, document.body)
      : content;

  return (
    <DialogStackContext.Provider value={stack}>
      {rendered}
    </DialogStackContext.Provider>
  );
}
