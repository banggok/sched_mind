import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { Toast } from "./Toast";

afterEach(() => vi.useRealTimers());

describe("Toast", () => {
  it("supports manual dismissal", async () => {
    const dismiss = vi.fn();
    render(<Toast message="Saved" onDismiss={dismiss} />);
    await userEvent
      .setup()
      .click(screen.getByRole("button", { name: "Dismiss notification" }));
    expect(dismiss).toHaveBeenCalledTimes(1);
  });

  it("dismisses automatically after the shared delay", () => {
    vi.useFakeTimers();
    const dismiss = vi.fn();
    render(<Toast message="Saved" onDismiss={dismiss} />);
    act(() => vi.advanceTimersByTime(4999));
    expect(dismiss).not.toHaveBeenCalled();
    act(() => vi.advanceTimersByTime(1));
    expect(dismiss).toHaveBeenCalledTimes(1);
  });
});
