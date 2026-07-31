import { createHTTPRolesGateway } from "../features/roles/infrastructure/httpRolesGateway";
import { RolesDashboardPage } from "../features/roles/presentation/RolesDashboardPage";
import { createHTTPTeamMembersGateway } from "../features/team-members/infrastructure/httpTeamMembersGateway";
import { TeamMembersDashboardPage } from "../features/team-members/presentation/TeamMembersDashboardPage";
import { useEffect, useState } from "react";
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
const rolesGateway = createHTTPRolesGateway(
  apiBaseURL,
  teamMembersGateway.invalidateListCache,
);

export function App() {
  const [route, setRoute] = useState(window.location.hash);
  const [capacityMember, setCapacityMember] = useState<TeamMember>();
  const [wbsProject, setWBSProject] = useState<Project>();
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
          : "roles";

  useEffect(() => {
    window.scrollTo({ top: 0, left: 0, behavior: "auto" });
  }, [activePage]);

  return (
    <AppShell activePage={activePage}>
      {activePage === "projects" ? (
        <ProjectsPage
          gateway={projectsGateway}
          loadPublicHolidayDates={publicHolidaysGateway.calendar}
          onManageWBS={setWBSProject}
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
      {wbsProject ? (
        <WBSPanel
          project={wbsProject}
          gateway={wbsGateway}
          dependenciesGateway={dependenciesGateway}
          rolesGateway={rolesGateway}
          membersGateway={teamMembersGateway}
          loadPublicHolidayDates={publicHolidaysGateway.calendar}
          onClose={() => setWBSProject(undefined)}
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
