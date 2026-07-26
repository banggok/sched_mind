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

  it("opens below with internal scrolling when neither side can fit it", async () => {
    const user = userEvent.setup();
    const rectangle = vi
      .spyOn(HTMLElement.prototype, "getBoundingClientRect")
      .mockImplementation(function (this: HTMLElement) {
        if (this.getAttribute("role") === "dialog")
          return rectangleValue(0, 0, 352, 600);
        if (this.getAttribute("aria-haspopup") === "dialog")
          return rectangleValue(0, 300, 352, 50);
        return rectangleValue(0, 0, 0, 0);
      });
    Object.defineProperty(window, "innerHeight", {
      configurable: true,
      value: 500,
    });
    render(
      <CalendarPopover
        label="Effective Date"
        buttonLabel="Select date"
        instruction="Select a date."
        selectedDates={[]}
        onSelect={vi.fn(() => true)}
      />,
    );

    await user.click(
      screen.getByRole("button", { name: "Effective Date: Select date" }),
    );
    const calendar = screen.getByRole("dialog");
    expect(calendar.getAttribute("data-placement")).toBe("below");
    expect(calendar.getAttribute("data-scrollable")).toBe("true");
    expect(calendar.style.overflowY).toBe("auto");

    Object.defineProperty(window, "innerHeight", {
      configurable: true,
      value: 1200,
    });
    window.dispatchEvent(new Event("scroll"));
    expect(calendar.getAttribute("data-placement")).toBe("below");
    expect(calendar.getAttribute("data-scrollable")).toBe("true");
    rectangle.mockRestore();
  });
});

function rectangleValue(
  left: number,
  top: number,
  width: number,
  height: number,
) {
  return {
    top,
    bottom: top + height,
    left,
    right: left + width,
    width,
    height,
    x: left,
    y: top,
    toJSON() {},
  };
}
