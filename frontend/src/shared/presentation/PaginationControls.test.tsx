import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { PaginationControls } from "./PaginationControls";

describe("PaginationControls", () => {
  it("shows the current range and navigates between available pages", () => {
    const onPageChange = vi.fn();

    render(
      <PaginationControls
        page={2}
        pageSize={20}
        total={45}
        onPageChange={onPageChange}
      />,
    );

    expect(screen.getByText("21–40 of 45")).toBeTruthy();
    expect(screen.getByText("Page 2 of 3")).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Previous" }));
    fireEvent.click(screen.getByRole("button", { name: "Next" }));

    expect(onPageChange).toHaveBeenNthCalledWith(1, 1);
    expect(onPageChange).toHaveBeenNthCalledWith(2, 3);
  });

  it("disables navigation when only one page exists", () => {
    render(
      <PaginationControls
        page={1}
        pageSize={20}
        total={4}
        onPageChange={vi.fn()}
      />,
    );

    expect(
      (screen.getByRole("button", { name: "Previous" }) as HTMLButtonElement)
        .disabled,
    ).toBe(true);
    expect(
      (screen.getByRole("button", { name: "Next" }) as HTMLButtonElement)
        .disabled,
    ).toBe(true);
  });
});
