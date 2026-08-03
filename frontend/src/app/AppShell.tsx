import { useEffect, useState, type ReactNode } from "react";

import { ApplicationSidebar } from "./ApplicationSidebar";
import type { ApplicationPage } from "./navigation";
import { TopBar } from "./TopBar";

export function AppShell({
  activePage,
  children,
}: {
  activePage: ApplicationPage;
  children: ReactNode;
}) {
  const [sidebarOpen, setSidebarOpen] = useState(readSidebarState);

  useEffect(() => {
    try {
      window.localStorage.setItem(
        sidebarStateStorageKey,
        sidebarOpen ? "expanded" : "collapsed",
      );
    } catch {
      // Storage may be unavailable; the in-memory state remains usable.
    }
  }, [sidebarOpen]);

  useEffect(() => {
    if (activePage !== "home") {
      return;
    }

    document.documentElement.classList.add("home-viewport-locked");

    return () => {
      document.documentElement.classList.remove("home-viewport-locked");
    };
  }, [activePage]);

  return (
    <div
      className={`${activePage === "home" ? "h-screen overflow-hidden" : "min-h-screen"} bg-canvas text-text-primary`}
    >
      <ApplicationSidebar
        activePage={activePage}
        open={sidebarOpen}
        onOpenChange={setSidebarOpen}
      />
      <main
        className={`shell-content ${
          activePage === "home"
            ? "flex h-screen min-h-0 flex-col overflow-hidden"
            : "min-h-screen"
        } ${sidebarOpen ? "lg:ml-64" : "ml-0"}`}
      >
        <TopBar />
        {children}
      </main>
    </div>
  );
}

const sidebarStateStorageKey = "schedmind.application-sidebar";

function readSidebarState(): boolean {
  try {
    return window.localStorage.getItem(sidebarStateStorageKey) === "expanded";
  } catch {
    return false;
  }
}
