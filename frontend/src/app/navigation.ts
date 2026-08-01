export const navigationItems = [
  {
    id: "home",
    label: "Home",
    href: "#home",
    group: "home",
    groupLabel: "Home",
  },
  {
    id: "roles",
    label: "Roles",
    href: "#roles",
    group: "team-configuration",
    groupLabel: "Team Configuration",
  },
  {
    id: "team-members",
    label: "Members",
    href: "#team-members",
    group: "team-configuration",
    groupLabel: "Team Configuration",
  },
  {
    id: "public-holidays",
    label: "Public Holidays",
    href: "#public-holidays",
    group: "team-configuration",
    groupLabel: "Team Configuration",
  },
  {
    id: "projects",
    label: "Projects",
    href: "#projects",
    group: "project",
    groupLabel: "Project",
  },
] as const;

export type ApplicationPage = (typeof navigationItems)[number]["id"];
export type NavigationItem = (typeof navigationItems)[number];
export const applicationRootPath = "/";

export function getNavigationItem(page: ApplicationPage): NavigationItem {
  const item = navigationItems.find((candidate) => candidate.id === page);
  if (!item) {
    throw new Error(`Navigation metadata is missing for page: ${page}`);
  }
  return item;
}
