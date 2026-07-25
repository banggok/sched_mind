export const navigationItems = [
  {
    id: 'roles',
    label: 'Roles',
    href: '#roles',
    group: 'team-management',
    groupLabel: 'Team',
  },
  {
    id: 'team-members',
    label: 'Members',
    href: '#team-members',
    group: 'team-management',
    groupLabel: 'Team',
  },
] as const

export type ApplicationPage = (typeof navigationItems)[number]['id']
export type NavigationItem = (typeof navigationItems)[number]
export const applicationRootPath = '/'

export function getNavigationItem(page: ApplicationPage): NavigationItem {
  const item = navigationItems.find((candidate) => candidate.id === page)
  if (!item) {
    throw new Error(`Navigation metadata is missing for page: ${page}`)
  }
  return item
}
