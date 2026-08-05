package httpapi

import (
	capacityoverridehttp "github.com/banggok/sched_mind/backend/internal/capacityoverrides/transport/http"
	dependencyhttp "github.com/banggok/sched_mind/backend/internal/dependencies/transport/http"
	portfoliohttp "github.com/banggok/sched_mind/backend/internal/portfolio/transport/http"
	projecthttp "github.com/banggok/sched_mind/backend/internal/projects/transport/http"
	publicholidayhttp "github.com/banggok/sched_mind/backend/internal/publicholidays/transport/http"
	sprinthttp "github.com/banggok/sched_mind/backend/internal/sprints/transport/http"
	wbshttp "github.com/banggok/sched_mind/backend/internal/wbs/transport/http"
	"net/http"

	rolehttp "github.com/banggok/sched_mind/backend/internal/roles/transport/http"
	"github.com/banggok/sched_mind/backend/internal/shared/schedulingimpact"
	systemhealthhttp "github.com/banggok/sched_mind/backend/internal/systemhealth/transport/http"
	teammemberhttp "github.com/banggok/sched_mind/backend/internal/teammembers/transport/http"
)

func NewRouter(
	roleService rolehttp.Service,
	teamMemberService teammemberhttp.Service,
	capacityOverrideService capacityoverridehttp.Service,
	publicHolidayService publicholidayhttp.Service,
	projectService projecthttp.Service,
	wbsService wbshttp.Service,
	dependencyService dependencyhttp.Service,
	portfolioService portfoliohttp.Service,
	sprintService sprinthttp.Service,
) http.Handler {
	mux := http.NewServeMux()
	systemhealthhttp.Register(mux)
	rolehttp.New(roleService).Register(mux)
	teammemberhttp.New(teamMemberService).Register(mux)
	capacityoverridehttp.New(capacityOverrideService).Register(mux)
	publicholidayhttp.New(publicHolidayService).Register(mux)
	projecthttp.New(projectService).Register(mux)
	wbshttp.New(wbsService).Register(mux)
	dependencyhttp.New(dependencyService).Register(mux)
	portfoliohttp.New(portfolioService).Register(mux)
	sprinthttp.New(sprintService).Register(mux)

	return schedulingimpact.CaptureToken(mux)
}
