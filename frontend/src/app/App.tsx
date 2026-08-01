import { SchedulingImpactDialog } from "../shared/presentation/SchedulingImpactDialog";
import { useEffect, useRef, useState } from "react";
import { Toast } from "../shared/presentation/Toast";
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
import {
  ProjectsPage,
  type ProjectCommandState,
} from "../features/projects/presentation/ProjectsPage";
import type { Project } from "../features/projects/domain/project";
import { createHTTPWBSGateway } from "../features/wbs/infrastructure/httpWBSGateway";
import { WBSPanel } from "../features/wbs/presentation/WBSPanel";
import { ProjectWBSSummary } from "../features/wbs/presentation/ProjectWBSSummary";
import { createHTTPDependenciesGateway } from "../features/dependencies/infrastructure/httpDependenciesGateway";
import { createHTTPPortfolioGateway } from "../features/portfolio/infrastructure/httpPortfolioGateway";
import {
  PortfolioHomePage,
  type HomeProjectCommandKind,
} from "../features/portfolio/presentation/PortfolioHomePage";

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
  moveNodeId?: string;
};

export function App() {
  const [route, setRoute] = useState(window.location.hash);
  const [capacityMember, setCapacityMember] = useState<TeamMember>();
  const [wbsRequest, setWBSRequest] = useState<WBSRequest>();
  const [projectToEdit, setProjectToEdit] = useState<Project>();
  const [projectCommand, setProjectCommand] = useState<ProjectCommandState>();
  const [homeOverlayError, setHomeOverlayError] = useState("");
  const overlayRequestVersion = useRef(0);
  const activePageRef = useRef<ApplicationPage>("home");
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
    activePageRef.current = activePage;
    if (activePage !== "home") {
      overlayRequestVersion.current += 1;
      setProjectToEdit(undefined);
      setProjectCommand(undefined);
      setWBSRequest(undefined);
      setHomeOverlayError("");
    }
  }, [activePage]);

  async function openProject(projectID: string) {
    const requestVersion = beginHomeOverlayRequest();
    try {
      const project = await projectsGateway.get(projectID);
      if (!isCurrentHomeOverlayRequest(requestVersion)) return;
      setProjectToEdit(project);
    } catch {
      showHomeOverlayLoadFailure(requestVersion);
    }
  }
  async function openProjectCommand(
    kind: HomeProjectCommandKind,
    projectID: string,
  ) {
    const requestVersion = beginHomeOverlayRequest();
    try {
      const project = await projectsGateway.get(projectID);
      if (!isCurrentHomeOverlayRequest(requestVersion)) return;
      setProjectCommand({ kind, project });
    } catch {
      showHomeOverlayLoadFailure(requestVersion);
    }
  }
  async function openWBS(request: {
    projectId: string;
    nodeId?: string;
    createParentId?: string | null;
    moveNodeId?: string;
  }) {
    const requestVersion = beginHomeOverlayRequest();
    try {
      const project = await projectsGateway.get(request.projectId);
      if (!isCurrentHomeOverlayRequest(requestVersion)) return;
      setWBSRequest({
        key: requestVersion,
        project,
        nodeId: request.nodeId,
        createParentId: request.createParentId,
        moveNodeId: request.moveNodeId,
      });
    } catch {
      showHomeOverlayLoadFailure(requestVersion);
    }
  }
  function beginHomeOverlayRequest(): number {
    const requestVersion = overlayRequestVersion.current + 1;
    overlayRequestVersion.current = requestVersion;
    setHomeOverlayError("");
    setProjectToEdit(undefined);
    setProjectCommand(undefined);
    setWBSRequest(undefined);
    return requestVersion;
  }
  function isCurrentHomeOverlayRequest(requestVersion: number): boolean {
    return (
      requestVersion === overlayRequestVersion.current &&
      activePageRef.current === "home"
    );
  }
  function showHomeOverlayLoadFailure(requestVersion: number) {
    if (!isCurrentHomeOverlayRequest(requestVersion)) return;
    setHomeOverlayError("The selected Project could not be loaded. Try again.");
  }

  return (
    <AppShell activePage={activePage}>
      {activePage === "home" ? (
        <PortfolioHomePage
          gateway={portfolioGateway}
          wbsGateway={wbsGateway}
          onOpenProject={(projectID) => void openProject(projectID)}
          onProjectCommand={(kind, projectID) =>
            void openProjectCommand(kind, projectID)
          }
          onOpenWBS={(request) => void openWBS(request)}
        />
      ) : activePage === "projects" ? (
        <ProjectsPage
          gateway={projectsGateway}
          loadPublicHolidayDates={publicHolidaysGateway.calendar}
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
      {activePage === "home" && wbsRequest ? (
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
          initialMoveNodeId={wbsRequest.moveNodeId}
          onClose={() => setWBSRequest(undefined)}
        />
      ) : null}
      {activePage === "home" && projectToEdit ? (
        <ProjectsPage
          gateway={projectsGateway}
          dialogOnly
          initialEditProject={projectToEdit}
          returnPageOnClose="home"
          onReturnToPage={() => setProjectToEdit(undefined)}
          onOverlayComplete={() => setProjectToEdit(undefined)}
          loadPublicHolidayDates={publicHolidaysGateway.calendar}
          renderProjectSummary={(project) => (
            <ProjectWBSSummary
              key={project.id}
              projectId={project.id}
              gateway={wbsGateway}
            />
          )}
        />
      ) : null}
      {activePage === "home" && projectCommand ? (
        <ProjectsPage
          gateway={projectsGateway}
          dialogOnly
          initialCommand={projectCommand}
          onReturnToPage={() => setProjectCommand(undefined)}
          onOverlayComplete={() => setProjectCommand(undefined)}
        />
      ) : null}
      {homeOverlayError ? (
        <Toast
          message={homeOverlayError}
          onDismiss={() => setHomeOverlayError("")}
        />
      ) : null}
      <SchedulingImpactDialog />
    </AppShell>
  );
}

function requiredEnvironment(name: string, value: string | undefined): string {
  if (!value) {
    throw new Error(`${name} environment variable is required`);
  }
  return value.replace(/\/+$/, "");
}
