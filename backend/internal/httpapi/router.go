package httpapi

import (
	"net/http"

	rolehttp "github.com/banggok/sched_mind/backend/internal/roles/transport/http"
	systemhealthhttp "github.com/banggok/sched_mind/backend/internal/systemhealth/transport/http"
	teammemberhttp "github.com/banggok/sched_mind/backend/internal/teammembers/transport/http"
)

func NewRouter(
	roleService rolehttp.Service,
	teamMemberService teammemberhttp.Service,
) http.Handler {
	mux := http.NewServeMux()
	systemhealthhttp.Register(mux)
	rolehttp.New(roleService).Register(mux)
	teammemberhttp.New(teamMemberService).Register(mux)

	return mux
}
