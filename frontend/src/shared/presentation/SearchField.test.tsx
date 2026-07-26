import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { SearchField } from "./SearchField";

describe("SearchField", () => {
  it("reserves semantic input spacing for the leading search icon", () => {
    render(<SearchField label="Search projects" value="" onChange={vi.fn()} />);
    expect(
      screen
        .getByLabelText("Search projects")
        .classList.contains("ui-search-input"),
    ).toBe(true);
  });
});
