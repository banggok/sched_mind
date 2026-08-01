import { SchedulingImpactDialog } from "../shared/presentation/SchedulingImpactDialog";
import { useEffect, useState } from "react";
import { createHTTPRolesGateway } from "../features/roles/infrastructure/httpRolesGateway";
import { RolesDashboardPage } from "../features/roles/presentation/RolesDashboardPage";
import { createHTTPTeamMembersGateway } from "../features/team-members/infrastructure/httpTeamMembersGateway";
import { TeamMembersDashboardPage } from "../features/team-members/presentation/TeamMembersDashboardPage";
import { AppShell } from "./AppShell";
import type { ApplicationPage } from "./navigation";
import { createHTTPCapacityOverridesGateway } from "../features/capacity-overrides/infrastructure/httpCapacityOverridesGateway";
import { CapacityOverridesPanel } from "../features/capacity-overrides/presentation/CapacityOverridesPanel";
import type { TeamMember } from "../features/team-members/domain/teamMember";
import { createHTTPPublicHolidaysGateway } from "../features/public-holidays/infrastructure/httpPublicHolidaysGateway";
import { PublicHolidaysPage } from "../features/public-holidays/presentation/PublicHolidaysPage";
import { createHTTPProjectsGateway } from "../features/projects/infrastructure/httpProjectsGateway";
import { ProjectsPage } from "../features/projects/presentation/ProjectsPage";
import type { Project } from "../features/projects/domain/project";
import { createHTTPWBSGateway } from "../features/wbs/infrastructure/httpWBSGateway";
import { WBSPanel } from "../features/wbs/presentation/WBSPanel";
import { ProjectWBSSummary } from "../features/wbs/presentation/ProjectWBSSummary";
import { createHTTPDependenciesGateway } from "../features/dependencies/infrastructure/httpDependenciesGateway";
import { createHTTPPortfolioGateway } from "../features/portfolio/infrastructure/httpPortfolioGateway";
import { PortfolioHomePage } from "../features/portfolio/presentation/PortfolioHomePage";

const apiBaseURL = requiredEnvironment(
  "VITE_API_BASE_URL",
  import.meta.env.VITE_API_BASE_URL,
);
const teamMembersGateway = createHTTPTeamMembersGateway(apiBaseURL);
const capacityOverridesGateway = createHTTPCapacityOverridesGateway(apiBaseURL);
const publicHolidaysGateway = createHTTPPublicHolidaysGateway(apiBaseURL);
const projectsGateway = createHTTPProjectsGateway(apiBaseURL);
const wbsGateway = createHTTPWBSGateway(apiBaseURL);
const dependenciesGateway = createHTTPDependenciesGateway(apiBaseURL);
const portfolioGateway = createHTTPPortfolioGateway(apiBaseURL);
const rolesGateway = createHTTPRolesGateway(
  apiBaseURL,
  teamMembersGateway.invalidateListCache,
);

type WBSRequest = {
  key: number;
  project: Project;
  nodeId?: string;
  createParentId?: string | null;
  returnToHome?: boolean;
};

export function App() {
  const [route, setRoute] = useState(window.location.hash);
  const [capacityMember, setCapacityMember] = useState<TeamMember>();
  const [wbsRequest, setWBSRequest] = useState<WBSRequest>();
  const [projectToEdit, setProjectToEdit] = useState<Project>();
  const [projectReturnPage, setProjectReturnPage] = useState<ApplicationPage>();
  useEffect(() => {
    const updateRoute = () => setRoute(window.location.hash);
    window.addEventListener("hashchange", updateRoute);
    window.addEventListener("popstate", updateRoute);
    return () => {
      window.removeEventListener("hashchange", updateRoute);
      window.removeEventListener("popstate", updateRoute);
    };
  }, []);
  const activePage: ApplicationPage =
    route === "#projects"
      ? "projects"
      : route === "#team-members"
        ? "team-members"
        : route === "#public-holidays"
          ? "public-holidays"
          : route === "#roles"
            ? "roles"
            : "home";

  useEffect(() => {
    window.scrollTo({ top: 0, left: 0, behavior: "auto" });
  }, [activePage]);

  async function openProject(projectID: string, returnPage?: ApplicationPage) {
    const project = await projectsGateway.get(projectID);
    setProjectToEdit(project);
    setProjectReturnPage(returnPage);
    window.location.hash = "#projects";
  }
  async function openWBS(request: {
    projectId: string;
    nodeId?: string;
    createParentId?: string | null;
    returnToHome?: boolean;
  }) {
    const project = await projectsGateway.get(request.projectId);
    setWBSRequest({
      key: Date.now(),
      project,
      nodeId: request.nodeId,
      createParentId: request.createParentId,
      returnToHome: request.returnToHome,
    });
  }

  return (
    <AppShell activePage={activePage}>
      {activePage === "home" ? (
        <PortfolioHomePage
          gateway={portfolioGateway}
          onOpenProject={(projectID) => void openProject(projectID, "home")}
          onOpenWBS={(request) =>
            void openWBS({ ...request, returnToHome: true })
          }
        />
      ) : activePage === "projects" ? (
        <ProjectsPage
          gateway={projectsGateway}
          loadPublicHolidayDates={publicHolidaysGateway.calendar}
          onManageWBS={(project) => setWBSRequest({ key: Date.now(), project })}
          initialEditProject={projectToEdit}
          onInitialEditConsumed={() => setProjectToEdit(undefined)}
          returnPageOnClose={projectReturnPage}
          onReturnToPage={() => {
            if (!projectReturnPage) return;
            setProjectReturnPage(undefined);
            window.location.hash = `#${projectReturnPage}`;
          }}
          renderProjectSummary={(project) => (
            <ProjectWBSSummary
              key={project.id}
              projectId={project.id}
              gateway={wbsGateway}
            />
          )}
        />
      ) : activePage === "public-holidays" ? (
        <PublicHolidaysPage gateway={publicHolidaysGateway} />
      ) : activePage === "team-members" ? (
        <TeamMembersDashboardPage
          gateway={teamMembersGateway}
          roleOptionsGateway={rolesGateway}
          onManageCapacity={setCapacityMember}
        />
      ) : (
        <RolesDashboardPage gateway={rolesGateway} />
      )}
      {capacityMember ? (
        <CapacityOverridesPanel
          member={capacityMember}
          gateway={capacityOverridesGateway}
          loadPublicHolidayDates={publicHolidaysGateway.calendar}
          onClose={() => setCapacityMember(undefined)}
        />
      ) : null}
      <SchedulingImpactDialog />
      {wbsRequest ? (
        <WBSPanel
          key={wbsRequest.key}
          project={wbsRequest.project}
          gateway={wbsGateway}
          dependenciesGateway={dependenciesGateway}
          rolesGateway={rolesGateway}
          membersGateway={teamMembersGateway}
          loadPublicHolidayDates={publicHolidaysGateway.calendar}
          initialNodeId={wbsRequest.nodeId}
          initialCreateParentId={wbsRequest.createParentId}
          returnToCallerOnComplete={wbsRequest.returnToHome}
          onClose={() => setWBSRequest(undefined)}
        />
      ) : null}
    </AppShell>
  );
}

function requiredEnvironment(name: string, value: string | undefined): string {
  if (!value) {
    throw new Error(`${name} environment variable is required`);
  }
  return value.replace(/\/+$/, "");
}
