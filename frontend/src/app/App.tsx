import { createHTTPRolesGateway } from '../features/roles/infrastructure/httpRolesGateway'
import { RolesDashboardPage } from '../features/roles/presentation/RolesDashboardPage'
import { createHTTPTeamMembersGateway } from '../features/team-members/infrastructure/httpTeamMembersGateway'
import { TeamMembersDashboardPage } from '../features/team-members/presentation/TeamMembersDashboardPage'
import { useEffect, useState } from 'react'
import { AppShell } from './AppShell'
import type { ApplicationPage } from './navigation'

const apiBaseURL = requiredEnvironment(
  'VITE_API_BASE_URL',
  import.meta.env.VITE_API_BASE_URL,
)
const teamMembersGateway = createHTTPTeamMembersGateway(apiBaseURL)
const rolesGateway = createHTTPRolesGateway(
  apiBaseURL,
  teamMembersGateway.invalidateListCache,
)

export function App() {
  const [route, setRoute] = useState(window.location.hash)
  useEffect(() => {
    const updateRoute = () => setRoute(window.location.hash)
    window.addEventListener('hashchange', updateRoute)
    window.addEventListener('popstate', updateRoute)
    return () => {
      window.removeEventListener('hashchange', updateRoute)
      window.removeEventListener('popstate', updateRoute)
    }
  }, [])
  const activePage: ApplicationPage =
    route === '#team-members' ? 'team-members' : 'roles'

  useEffect(() => {
    window.scrollTo({ top: 0, left: 0, behavior: 'auto' })
  }, [activePage])

  return (
    <AppShell activePage={activePage}>
      {activePage === 'team-members' ? (
        <TeamMembersDashboardPage
          gateway={teamMembersGateway}
          rolesGateway={rolesGateway}
        />
      ) : (
        <RolesDashboardPage gateway={rolesGateway} />
      )}
    </AppShell>
  )
}

function requiredEnvironment(name: string, value: string | undefined): string {
  if (!value) {
    throw new Error(`${name} environment variable is required`)
  }
  return value.replace(/\/+$/, '')
}
