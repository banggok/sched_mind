import { createHTTPRolesGateway } from '../features/roles/infrastructure/httpRolesGateway'
import { RolesDashboardPage } from '../features/roles/presentation/RolesDashboardPage'

const apiBaseURL = requiredEnvironment(
  'VITE_API_BASE_URL',
  import.meta.env.VITE_API_BASE_URL,
)
const rolesGateway = createHTTPRolesGateway(apiBaseURL)

export function App() {
  return <RolesDashboardPage gateway={rolesGateway} />
}

function requiredEnvironment(name: string, value: string | undefined): string {
  if (!value) {
    throw new Error(`${name} environment variable is required`)
  }
  return value.replace(/\/+$/, '')
}
