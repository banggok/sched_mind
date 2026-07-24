package transporthttp

import (
	"net/http"

	"github.com/banggok/sched_mind/backend/internal/transport/http/handler"
)

func NewRouter() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handler.Health)

	return mux
}
