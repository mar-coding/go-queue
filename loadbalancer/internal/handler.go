package internal

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
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
	Port    string `json:"port"`
	RPCPort string `json:"rpcPort"`
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

	// Extract the client's IP address
	address := r.RemoteAddr
	if clientIP := r.Header.Get("X-Real-IP"); clientIP != "" {
		address = clientIP
	} else if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		address = strings.Split(forwardedFor, ",")[0]
	}

	// Remove port from the address if present and handle IPv6
	if host, _, err := net.SplitHostPort(address); err == nil {
		address = host
	}

	// Convert localhost IPv6 to IPv4
	if address == "::1" {
		address = "127.0.0.1"
	}

	// If it's still an IPv6 address, try to get IPv4 equivalent
	if strings.Contains(address, ":") {
		ip := net.ParseIP(address)
		if ip != nil {
			if ip4 := ip.To4(); ip4 != nil {
				address = ip4.String()
			}
		}
	}

	if err := h.service.RegisterNode(req.ID, address, req.Port, req.RPCPort); err != nil {
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
	log.Printf("Received request: %s %s", r.Method, r.URL.Path)

	node, err := h.service.GetHealthyNode()
	if err != nil {
		log.Printf("No healthy nodes available: %v", err)
		h.respondWithError(w, http.StatusServiceUnavailable, "No healthy nodes available")
		return
	}

	log.Printf("Selected node %s at %s:%s", node.ID, node.Address, node.Port)

	// Create the target URL
	targetURL := fmt.Sprintf("http://%s:%s%s", node.Address, node.Port, r.URL.Path)
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}
	target, err := url.Parse(targetURL)
	if err != nil {
		log.Printf("Failed to parse target URL %s: %v", targetURL, err)
		h.respondWithError(w, http.StatusInternalServerError, "Invalid node address")
		return
	}

	// Create director function to modify the request
	director := func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = target.Path
		req.URL.RawQuery = target.RawQuery

		// Preserve the original headers
		if _, ok := req.Header["User-Agent"]; !ok {
			req.Header.Set("User-Agent", "")
		}

		// Add X-Forwarded headers
		req.Header.Set("X-Forwarded-Host", req.Host)
		req.Header.Set("X-Forwarded-Proto", "http")
		req.Header.Set("X-Forwarded-For", req.RemoteAddr)

		// Update the Host header to match the target
		req.Host = target.Host
	}

	// Create reverse proxy with modified transport
	proxy := &httputil.ReverseProxy{
		Director: director,
		Transport: &http.Transport{
			ResponseHeaderTimeout: 30 * time.Second,
			DisableKeepAlives:     false,
			MaxIdleConnsPerHost:   100,
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("Proxy error for node %s: %v", node.ID, err)
			h.service.ReleaseNode(node.ID)
			h.respondWithError(w, http.StatusBadGateway, "Error forwarding request")
		},
	}

	log.Printf("Forwarding request to %s", targetURL)

	// Forward the request
	proxy.ServeHTTP(w, r)

	// Release the connection after the request is done
	h.service.ReleaseNode(node.ID)
	log.Printf("Request completed for node %s", node.ID)
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
