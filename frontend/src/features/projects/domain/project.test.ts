import { describe, expect, it } from "vitest";
import { normalizeProjectName, ProjectNameError } from "./project";

describe("project name", () => {
  it("trims a valid name", () =>
    expect(normalizeProjectName("  Alpha  ")).toBe("Alpha"));
  it("rejects empty and overly long names", () => {
    expect(() => normalizeProjectName(" ")).toThrow(ProjectNameError);
    expect(() => normalizeProjectName("x".repeat(101))).toThrow(
      "Project name must not exceed 100 characters",
    );
  });
});
