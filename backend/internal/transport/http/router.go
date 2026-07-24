package transporthttp

import (
	"net/http"

	rolehttp "github.com/banggok/sched_mind/backend/internal/roles/transport/http"
	"github.com/banggok/sched_mind/backend/internal/transport/http/handler"
)

func NewRouter(roleService rolehttp.Service) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handler.Health)
	mux.HandleFunc("GET /api/health", handler.Health)
	rolehttp.New(roleService).Register(mux)

	return mux
}
