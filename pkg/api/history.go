package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mockzilla/connexions/v2/pkg/db"
)

// CreateHistoryRoutes adds history routes to the router.
func CreateHistoryRoutes(router *Router) error {
	if router.Config().DisableUI || router.Config().History.URL == "" {
		return nil
	}

	handler := &HistoryHandler{
		router: router,
	}

	url := router.Config().History.URL
	url = "/" + strings.Trim(url, "/")

	router.Route(url, func(r chi.Router) {
		r.Get("/*", handler.list)
		r.Delete("/*", handler.clear)
	})

	return nil
}

// HistoryHandler handles history routes.
type HistoryHandler struct {
	router *Router
}

// HistoryListResponse is the response for history list endpoint.
type HistoryListResponse struct {
	Items []*db.HistoryEntry `json:"items"`
}

// getService looks up the service by name and checks that history is enabled for it.
// Returns the DB or writes an error response and returns nil.
func (h *HistoryHandler) getService(w http.ResponseWriter, r *http.Request) (string, db.DB) {
	serviceName := h.getServiceName(r)

	svc := h.router.GetServices()[serviceName]
	if svc == nil {
		http.Error(w, "Service not found", http.StatusNotFound)
		return serviceName, nil
	}

	if svc.Config != nil && !svc.Config.HistoryEnabled() {
		http.Error(w, "History disabled for this service", http.StatusNotFound)
		return serviceName, nil
	}

	database := h.router.GetDB(serviceName)
	if database == nil {
		http.Error(w, "Service not found", http.StatusNotFound)
		return serviceName, nil
	}

	return serviceName, database
}

func (h *HistoryHandler) list(w http.ResponseWriter, r *http.Request) {
	serviceName, database := h.getService(w, r)
	if database == nil {
		return
	}

	// Check for ID path segment after service name
	id := h.getEntryID(r, serviceName)
	if id != "" {
		entry, ok := database.History().GetByID(r.Context(), id)
		if !ok {
			http.Error(w, "Entry not found", http.StatusNotFound)
			return
		}
		NewJSONResponse(w).Send(entry)
		return
	}

	items := database.History().Data(r.Context())
	if items == nil {
		items = make([]*db.HistoryEntry, 0)
	}
	NewJSONResponse(w).Send(&HistoryListResponse{Items: items})
}

func (h *HistoryHandler) clear(w http.ResponseWriter, r *http.Request) {
	_, database := h.getService(w, r)
	if database == nil {
		return
	}

	database.History().Clear(r.Context())
	NewJSONResponse(w).Send(&HistoryListResponse{Items: make([]*db.HistoryEntry, 0)})
}

// getServiceName extracts the service name from the request path.
func (h *HistoryHandler) getServiceName(r *http.Request) string {
	historyURL := h.router.Config().History.URL
	prefix := "/" + strings.Trim(historyURL, "/") + "/"
	name := strings.TrimPrefix(r.URL.Path, prefix)

	// Strip any trailing path segments (entry ID)
	if idx := strings.Index(name, "/"); idx != -1 {
		name = name[:idx]
	}

	// Handle root service
	if name == RootServiceName {
		name = ""
	}

	return name
}

// getEntryID extracts the entry ID from the request path (segment after service name).
func (h *HistoryHandler) getEntryID(r *http.Request, serviceName string) string {
	historyURL := h.router.Config().History.URL
	prefix := "/" + strings.Trim(historyURL, "/") + "/"
	path := strings.TrimPrefix(r.URL.Path, prefix)

	// For root service, the URL name is RootServiceName
	urlName := serviceName
	if serviceName == "" {
		urlName = RootServiceName
	}

	rest := strings.TrimPrefix(path, urlName)
	rest = strings.TrimPrefix(rest, "/")
	if rest == "" {
		return ""
	}
	return rest
}
