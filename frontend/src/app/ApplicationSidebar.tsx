import type { ReactNode } from "react";

import {
  navigationItems,
  type ApplicationPage,
  type NavigationItem,
} from "./navigation";

export function ApplicationSidebar({
  activePage,
  open,
  onOpenChange,
}: {
  activePage: ApplicationPage;
  open: boolean;
  onOpenChange(open: boolean): void;
}) {
  return (
    <>
      {open ? (
        <button
          type="button"
          className="layer-navigation-backdrop fixed inset-0 bg-overlay/35 lg:hidden"
          aria-label="Close navigation"
          onClick={() => onOpenChange(false)}
        />
      ) : null}
      <button
        type="button"
        className="navigation-toggle layer-navigation-control fixed grid size-9 place-items-center rounded-control border border-border-strong bg-surface font-bold shadow-surface hover:border-brand hover:text-brand"
        aria-label={open ? "Collapse navigation" : "Expand navigation"}
        aria-expanded={open}
        aria-controls="application-sidebar"
        onClick={() => onOpenChange(!open)}
      >
        ☰
      </button>
      <aside
        id="application-sidebar"
        className={`layer-navigation fixed inset-y-0 left-0 flex w-64 flex-col border-r border-border-subtle bg-surface shadow-floating transition-transform duration-interface ease-interface motion-reduce:transition-none ${
          open ? "translate-x-0" : "-translate-x-full"
        }`}
        aria-label="Application sidebar"
      >
        <div className="application-sidebar-spacer shrink-0 border-b border-border-subtle" />
        <nav className="flex-1 p-3" aria-label="Application navigation">
          {groupNavigationItems().map((group) => (
            <section
              key={group.id}
              aria-labelledby={
                group.id === "home" ? undefined : `nav-${group.id}`
              }
            >
              {group.id === "home" ? null : (
                <p
                  id={`nav-${group.id}`}
                  className="overflow-hidden px-3 pt-2 pb-2 text-xs font-extrabold tracking-widest whitespace-nowrap text-subtle uppercase"
                >
                  {group.label}
                </p>
              )}
              {group.items.map((item) => (
                <NavigationLink
                  key={item.id}
                  item={item}
                  activePage={activePage}
                >
                  {icons[item.id]}
                </NavigationLink>
              ))}
            </section>
          ))}
        </nav>
      </aside>
    </>
  );
}

function NavigationLink({
  item,
  activePage,
  children,
}: {
  item: NavigationItem;
  activePage: ApplicationPage;
  children: ReactNode;
}) {
  const active = item.id === activePage;
  return (
    <a
      className={`mt-2 flex items-center gap-3 rounded-control px-3 py-3 text-sm font-extrabold ${
        active
          ? "bg-brand-soft text-brand-strong"
          : "text-text-secondary hover:bg-surface-muted"
      }`}
      href={item.href}
      aria-current={active ? "page" : undefined}
    >
      <span
        aria-hidden="true"
        className={`grid size-8 shrink-0 place-items-center rounded-action ${
          active ? "bg-surface text-brand" : "bg-surface-muted"
        }`}
      >
        {children}
      </span>
      <span>{item.label}</span>
    </a>
  );
}

const icons: Record<ApplicationPage, ReactNode> = {
  home: <HomeIcon />,
  roles: <RoleIcon />,
  "team-members": <TeamMembersIcon />,
  "public-holidays": <CalendarIcon />,
  projects: <ProjectIcon />,
  sprints: <SprintIcon />,
};

function HomeIcon() {
  return (
    <svg
      className="size-5"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="M3 11.5 12 4l9 7.5" />
      <path d="M5.5 10.5V20h13v-9.5M9 20v-5h6v5" />
    </svg>
  );
}

function ProjectIcon() {
  return (
    <svg
      className="size-5"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="M4 6.5h6l2 2h8v10H4z" />
      <path d="M4 9h16" />
    </svg>
  );
}

function SprintIcon() {
  return (
    <svg
      className="size-5"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <rect x="4" y="4" width="16" height="16" rx="3" />
      <path d="M8 2v4M16 2v4M4 9h16M8 13h3M8 17h6" />
    </svg>
  );
}

function groupNavigationItems() {
  return navigationItems.reduce<
    Array<{
      id: string;
      label: string;
      items: NavigationItem[];
    }>
  >((groups, item) => {
    const existing = groups.find((group) => group.id === item.group);
    if (existing) existing.items.push(item);
    else
      groups.push({
        id: item.group,
        label: item.groupLabel,
        items: [item],
      });
    return groups;
  }, []);
}

function RoleIcon() {
  return (
    <svg
      className="size-5"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <rect x="3" y="5" width="18" height="14" rx="3" />
      <circle cx="9" cy="11" r="2.25" />
      <path d="M5.75 16c.7-1.6 1.8-2.4 3.25-2.4s2.55.8 3.25 2.4M15 10h3M15 14h2" />
    </svg>
  );
}

function TeamMembersIcon() {
  return (
    <svg
      className="size-5"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <circle cx="9" cy="8" r="3" />
      <path d="M3.5 19c.5-3.5 2.3-5.2 5.5-5.2s5 1.7 5.5 5.2M16 7.5a2.5 2.5 0 0 1 0 5M16.5 14.5c2.4.4 3.7 1.9 4 4.5" />
    </svg>
  );
}

function CalendarIcon() {
  return (
    <svg
      className="size-5"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <rect x="3" y="5" width="18" height="16" rx="3" />
      <path d="M7 3v4M17 3v4M3 10h18M8 14h.01M12 14h.01M16 14h.01M8 18h.01M12 18h.01" />
    </svg>
  );
}
