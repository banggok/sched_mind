import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { CalendarPopover } from "./CalendarPopover";

describe("CalendarPopover", () => {
  it("does not open when the date control is read-only", async () => {
    const user = userEvent.setup();
    render(
      <CalendarPopover
        label="Scheduling Start Date"
        buttonLabel="Select date"
        disabled
        instruction="Select a date."
        selectedDates={[]}
        onSelect={vi.fn(() => true)}
      />,
    );

    const trigger = screen.getByRole("button", {
      name: "Scheduling Start Date: Select date",
    });
    expect(trigger.hasAttribute("disabled")).toBe(true);
    await user.click(trigger);
    expect(screen.queryByRole("dialog")).toBeNull();
  });

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

  it("aligns to the right when left alignment would cross the viewport", async () => {
    const user = userEvent.setup();
    const rectangle = vi
      .spyOn(HTMLElement.prototype, "getBoundingClientRect")
      .mockImplementation(function (this: HTMLElement) {
        if (this.getAttribute("role") === "dialog")
          return rectangleValue(800, 200, 352, 400);
        if (this.getAttribute("aria-haspopup") === "dialog")
          return rectangleValue(800, 140, 200, 50);
        return rectangleValue(0, 0, 0, 0);
      });
    Object.defineProperty(window, "innerWidth", {
      configurable: true,
      value: 1000,
    });
    Object.defineProperty(window, "innerHeight", {
      configurable: true,
      value: 900,
    });
    render(
      <CalendarPopover
        label="Holiday Date"
        buttonLabel="Select date"
        instruction="Select a date."
        selectedDates={[]}
        onSelect={vi.fn(() => true)}
      />,
    );
    await user.click(
      screen.getByRole("button", { name: "Holiday Date: Select date" }),
    );
    const calendar = screen.getByRole("dialog");
    expect(calendar.getAttribute("data-alignment")).toBe("right");
    expect(calendar.classList.contains("layer-dialog-popover")).toBe(true);
    rectangle.mockRestore();
  });

  it("marks weekends and configured holidays without disabling selection", async () => {
    const user = userEvent.setup();
    const select = vi.fn(() => true);
    render(
      <CalendarPopover
        label="Date"
        buttonLabel="Select date"
        initialDate="2026-07-01"
        instruction="Select a date."
        selectedDates={[]}
        loadPublicHolidayDates={vi.fn().mockResolvedValue(["2026-07-03"])}
        onSelect={select}
      />,
    );
    await user.click(screen.getByRole("button", { name: "Date: Select date" }));
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "2026-07-03" }).title).toBe(
        "Holiday",
      ),
    );
    expect(screen.getByRole("button", { name: "2026-07-04" }).title).toBe(
      "Holiday",
    );
    expect(screen.queryByLabelText("Calendar legend")).toBeNull();
    await user.click(screen.getByRole("button", { name: "2026-07-03" }));
    expect(select).toHaveBeenCalledWith("2026-07-03");
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
