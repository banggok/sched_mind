import { render, screen } from "@testing-library/react";
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
