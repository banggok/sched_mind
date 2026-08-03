import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { AppShell } from "./AppShell";

describe("AppShell viewport ownership", () => {
  it("keeps Home within the viewport so its Gantt owns vertical scrolling", () => {
    const { container, unmount } = render(
      <AppShell activePage="home">
        <div>Home</div>
      </AppShell>,
    );

    expect(container.firstElementChild?.className).toContain("h-screen");
    expect(container.firstElementChild?.className).toContain("overflow-hidden");
    expect(container.querySelector("main")?.className).toContain("min-h-0");
    expect(container.querySelector("main")?.className).toContain(
      "overflow-hidden",
    );
    expect(
      document.documentElement.classList.contains("home-viewport-locked"),
    ).toBe(true);

    unmount();

    expect(
      document.documentElement.classList.contains("home-viewport-locked"),
    ).toBe(false);
  });

  it("preserves document scrolling for non-Home pages", () => {
    const { container } = render(
      <AppShell activePage="roles">
        <div>Roles</div>
      </AppShell>,
    );

    expect(container.firstElementChild?.className).toContain("min-h-screen");
    expect(container.querySelector("main")?.className).toContain(
      "min-h-screen",
    );
    expect(
      document.documentElement.classList.contains("home-viewport-locked"),
    ).toBe(false);
  });
});
