import { checkBackendHealth } from '../features/system-health/infrastructure/checkBackendHealth'
import { useSystemHealth } from '../features/system-health/presentation/useSystemHealth'
import { DesignReviewPage } from './DesignReviewPage'

export function App() {
  const healthStatus = useSystemHealth(checkBackendHealth)

  return <DesignReviewPage healthStatus={healthStatus} />
}

