import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { CalendarPopover } from "./CalendarPopover";

describe("CalendarPopover", () => {
  it("closes when the user clicks outside without moving focus back", async () => {
    const user = userEvent.setup();
    render(
      <>
        <CalendarPopover
          label="Effective Date"
          buttonLabel="Select date"
          instruction="Select a date."
          selectedDates={[]}
          onSelect={vi.fn(() => true)}
        />
        <button type="button">Outside action</button>
      </>,
    );

    await user.click(
      screen.getByRole("button", {
        name: "Effective Date: Select date",
      }),
    );
    expect(screen.getByRole("dialog")).toBeTruthy();

    const outsideAction = screen.getByRole("button", {
      name: "Outside action",
    });
    await user.click(outsideAction);

    expect(screen.queryByRole("dialog")).toBeNull();
    expect(document.activeElement).toBe(outsideAction);
  });
});
