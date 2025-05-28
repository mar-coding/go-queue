package internal

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// Handler handles HTTP requests for the load balancer
type Handler struct {
	service *Service
}

// NewHandler creates a new HTTP handler
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// SetupRoutes configures the HTTP routes
func (h *Handler) SetupRoutes() http.Handler {
	mux := http.NewServeMux()

	// Node registration
	mux.HandleFunc("/lb/register", h.RegisterNode)

	// Node status and stats
	mux.HandleFunc("/lb/status", h.GetStatus)
	mux.HandleFunc("/lb/stats", h.GetStats)

	// Proxy all other requests to nodes
	mux.HandleFunc("/", h.ProxyHandler)

	return mux
}

// RegisterNodeRequest represents the request body for node registration
type RegisterNodeRequest struct {
	ID      string `json:"id"`
	Address string `json:"address"`
	Port    string `json:"port"`
}

// RegisterNode handles node registration requests
func (h *Handler) RegisterNode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req RegisterNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.service.RegisterNode(req.ID, req.Address, req.Port); err != nil {
		h.respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]string{
		"status":  "registered",
		"node_id": req.ID,
	})
}

// GetStatus handles load balancer status requests
func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	stats := map[string]interface{}{
		"healthy_nodes": len(h.service.GetHealthyNodes()),
		"total_nodes":   len(h.service.GetAllNodes()),
	}

	h.respondWithJSON(w, http.StatusOK, stats)
}

// GetStats returns detailed statistics about nodes
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	stats := h.service.GetNodeStats()
	h.respondWithJSON(w, http.StatusOK, stats)
}

// ProxyHandler forwards requests to the selected node using least connections strategy
func (h *Handler) ProxyHandler(w http.ResponseWriter, r *http.Request) {
	node, err := h.service.GetHealthyNode()
	if err != nil {
		h.respondWithError(w, http.StatusServiceUnavailable, "No healthy nodes available")
		return
	}

	// Create the target URL
	target, err := url.Parse("http://" + node.Address + ":" + node.Port)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Invalid node address")
		return
	}

	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(target)

	// Add custom error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Error proxying request to node %s: %v", node.ID, err)
		h.service.ReleaseNode(node.ID)
		h.respondWithError(w, http.StatusBadGateway, "Error forwarding request")
	}

	// Forward the request
	proxy.ServeHTTP(w, r)

	// Release the connection after the request is done
	h.service.ReleaseNode(node.ID)
}

// respondWithError sends an error response
func (h *Handler) respondWithError(w http.ResponseWriter, code int, message string) {
	h.respondWithJSON(w, code, map[string]string{"error": message})
}

// respondWithJSON sends a JSON response
func (h *Handler) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshaling JSON response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, err = w.Write(response)
	if err != nil {
		return
	}
}
