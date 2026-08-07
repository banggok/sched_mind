import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { WBSSummary } from "./WBSSummary";

describe("US-4.3 shared WBS summary presentation", () => {
  it("AC-25 keeps Group summary horizontal on large screens while preserving wrapping", () => {
    render(
      <WBSSummary
        summary={{
          taskCount: 123_456,
          execution: {
            start: "2026-08-01",
            end: "2026-12-31",
            scheduledTaskCount: 123_455,
          },
          commitment: {
            start: "2026-08-01",
            end: "2027-01-31",
            scheduledTaskCount: 123_454,
          },
          completedKnownEffortMinutes: 3_703_680,
          totalKnownEffortMinutes: 7_407_390,
          taskWithoutEffortCount: 12_345,
          completionPercentage: 50,
        }}
        subject="group"
        idPrefix="large-group-summary"
      />,
    );

    const summary = screen.getByLabelText("Group summary details");
    expect(summary.className).toContain("min-w-0");
    expect(summary.className).toContain("lg:grid-cols-3");

    const coverage = within(summary).getByText(
      "123455 of 123456 tasks scheduled",
    );
    const effort = within(summary).getByText(
      "61728 of 123456.5 hours completed (50%)",
    );
    const disclosure = within(summary).getByText(
      "12345 tasks without effort are excluded from this calculation.",
    );
    expect(coverage.className).toContain("break-words");
    expect(effort.className).toContain("break-words");
    expect(disclosure.className).toContain("break-words");
  });
});
