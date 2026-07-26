import { describe, expect, it } from "vitest";
import { getNavigationItem, navigationItems } from "./navigation";

describe("application navigation", () => {
  it("groups configuration pages under Team Configuration", () => {
    expect(getNavigationItem("public-holidays")).toMatchObject({
      label: "Public Holidays",
      href: "#public-holidays",
      groupLabel: "Team Configuration",
    });
    expect(navigationItems.some((item) => item.id === "public-holidays")).toBe(
      true,
    );
    expect(getNavigationItem("projects")).toMatchObject({
      label: "Projects",
      groupLabel: "Project",
    });
    expect(new Set(navigationItems.map((item) => item.groupLabel))).toEqual(
      new Set(["Team Configuration", "Project"]),
    );
  });
});
