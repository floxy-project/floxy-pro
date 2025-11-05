package cleanup

import (
	"net/http"

	floxy "github.com/rom8726/floxy-pro"
	"github.com/rom8726/floxy-pro/api"
)

var _ api.Plugin = (*Plugin)(nil)

type Plugin struct {
	store floxy.Store
}

func New(store floxy.Store) *Plugin {
	return &Plugin{
		store: store,
	}
}

func (p *Plugin) Name() string { return "cleanup" }

func (p *Plugin) Description() string { return "Clean up old completed workflow instances" }

func (p *Plugin) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/cleanup", HandleCleanupWorkflows(p.store))
}

func HandleCleanupWorkflows(
	store floxy.Store,
) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Call the cleanup function
		err := store.CleanupOldWorkflows(ctx)
		if err != nil {
			api.WriteErrorResponse(w, err, http.StatusInternalServerError)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}"))
	}
}
