import { describe, expect, it } from "vitest";
import {
  normalizeProjectName,
  ProjectNameError,
  ProjectSettingsError,
  validateProjectBuffer,
} from "./project";

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

describe("project settings", () => {
  it("accepts inclusive integer boundaries", () => {
    expect(validateProjectBuffer(0)).toBe(0);
    expect(validateProjectBuffer(100)).toBe(100);
  });
  it("rejects out of range and decimal values", () => {
    expect(() => validateProjectBuffer(-1)).toThrow(ProjectSettingsError);
    expect(() => validateProjectBuffer(101)).toThrow(ProjectSettingsError);
    expect(() => validateProjectBuffer(20.5)).toThrow(ProjectSettingsError);
  });
});
