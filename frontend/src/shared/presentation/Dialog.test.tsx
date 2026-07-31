import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import axe from "axe-core";
import { useState } from "react";
import { describe, expect, it } from "vitest";
import { Dialog } from "./Dialog";

function DialogFixture() {
  const [open, setOpen] = useState(false);
  return (
    <>
      <button type="button" onClick={() => setOpen(true)}>
        Open dialog
      </button>
      {open ? (
        <Dialog titleID="dialog-title" onClose={() => setOpen(false)}>
          <h2 id="dialog-title">Accessible dialog</h2>
          <button type="button" data-autofocus>
            First action
          </button>
          <button type="button">Last action</button>
        </Dialog>
      ) : null}
    </>
  );
}

function NestedDialogFixture() {
  const [nestedOpen, setNestedOpen] = useState(false);
  const [confirmationOpen, setConfirmationOpen] = useState(false);
  return (
    <Dialog titleID="parent-dialog-title" onClose={() => undefined}>
      <h2 id="parent-dialog-title">Parent dialog</h2>
      <button type="button" onClick={() => setNestedOpen(true)}>
        Open nested dialog
      </button>
      {nestedOpen ? (
        <Dialog
          nested
          titleID="nested-dialog-title"
          onClose={() => setNestedOpen(false)}
        >
          <h3 id="nested-dialog-title">Nested dialog</h3>
          <button type="button" onClick={() => setConfirmationOpen(true)}>
            Open confirmation
          </button>
          {confirmationOpen ? (
            <Dialog
              nested
              kind="alertdialog"
              titleID="confirmation-title"
              onClose={() => setConfirmationOpen(false)}
            >
              <h4 id="confirmation-title">Confirmation</h4>
              <button type="button">Confirm action</button>
            </Dialog>
          ) : null}
        </Dialog>
      ) : null}
    </Dialog>
  );
}

describe("Dialog", () => {
  it("has no detectable accessibility violations", async () => {
    const { container } = render(<DialogFixture />);
    await userEvent
      .setup()
      .click(screen.getByRole("button", { name: "Open dialog" }));

    const results = await axe.run(container, {
      rules: {
        // jsdom has no layout engine, so contrast remains a manual/browser gate.
        "color-contrast": { enabled: false },
      },
    });
    expect(results.violations).toEqual([]);
  });

  it("portals nested dialogs outside the scrollable parent overlay", async () => {
    const user = userEvent.setup();
    render(<NestedDialogFixture />);

    const parent = screen.getByRole("dialog", { name: "Parent dialog" });
    await user.click(
      within(parent).getByRole("button", { name: "Open nested dialog" }),
    );

    const nested = screen.getByRole("dialog", { name: "Nested dialog" });
    const nestedOverlay = nested.parentElement;
    expect(parent.getAttribute("aria-hidden")).toBe("true");
    expect(parent.getAttribute("aria-modal")).toBeNull();
    expect(nested.dataset.dialogActive).toBe("true");
    expect(parent.contains(nested)).toBe(false);
    expect(nestedOverlay?.className).toContain("dialog-overlay-nested");
    expect(nestedOverlay?.dataset.dialogDepth).toBe("1");
    expect(nestedOverlay?.style.getPropertyValue("--dialog-layer-offset")).toBe(
      "0",
    );
    expect(nestedOverlay?.parentElement).toBe(document.body);
  });

  it("MVF-01 assigns a higher semantic layer to a dialog nested inside another nested dialog", async () => {
    const user = userEvent.setup();
    render(<NestedDialogFixture />);

    const parent = screen.getByRole("dialog", { name: "Parent dialog" });
    expect(parent.parentElement?.dataset.dialogDepth).toBe("0");
    await user.click(
      within(parent).getByRole("button", { name: "Open nested dialog" }),
    );

    const nested = screen.getByRole("dialog", { name: "Nested dialog" });
    await user.click(
      within(nested).getByRole("button", { name: "Open confirmation" }),
    );

    const confirmation = screen.getByRole("alertdialog", {
      name: "Confirmation",
    });
    const confirmationOverlay = confirmation.parentElement;
    expect(nested.contains(confirmation)).toBe(false);
    expect(confirmationOverlay?.parentElement).toBe(document.body);
    expect(confirmationOverlay?.dataset.dialogDepth).toBe("2");
    expect(
      confirmationOverlay?.style.getPropertyValue("--dialog-layer-offset"),
    ).toBe("1");
    expect(nested.getAttribute("aria-hidden")).toBe("true");
    expect(nested.getAttribute("aria-modal")).toBeNull();
    expect(confirmation.dataset.dialogActive).toBe("true");
    expect(document.activeElement?.closest('[role="alertdialog"]')).toBe(
      confirmation,
    );

    await user.keyboard("{Escape}");
    expect(screen.queryByRole("alertdialog")).toBeNull();
    expect(nested.getAttribute("aria-hidden")).toBeNull();
    expect(nested.getAttribute("aria-modal")).toBe("true");
    expect(document.activeElement).toBe(
      within(nested).getByRole("button", { name: "Open confirmation" }),
    );
  });

  it("manages initial focus, traps tab navigation, and restores focus", async () => {
    const user = userEvent.setup();
    render(<DialogFixture />);

    const trigger = screen.getByRole("button", { name: "Open dialog" });
    await user.click(trigger);

    const first = screen.getByRole("button", { name: "First action" });
    const last = screen.getByRole("button", { name: "Last action" });
    expect(document.activeElement).toBe(first);

    last.focus();
    await user.tab();
    expect(document.activeElement).toBe(first);

    await user.keyboard("{Escape}");
    expect(screen.queryByRole("dialog")).toBeNull();
    expect(document.activeElement).toBe(trigger);
  });
});
