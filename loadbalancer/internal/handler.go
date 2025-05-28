package internal

import (
	"encoding/json"
	"fmt"
	myType "github.com/mar-coding/go-queue/loadbalancer/types"
	"log"
	"net/http"
)

type Handler struct {
	service *Service
}

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

	// Queue operations
	mux.HandleFunc("/createQueue", h.HandleCreateQueue)
	mux.HandleFunc("/appendData", h.HandleAppendData)
	mux.HandleFunc("/readData", h.HandleReadData)

	return mux
}

func (h *Handler) RegisterNode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req myType.RegisterNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.service.RegisterNode(req.ID, r.RemoteAddr, req.Port, req.RPCPort); err != nil {
		h.respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]string{
		"status":  "registered",
		"node_id": req.ID,
	})
}

func (h *Handler) HandleCreateQueue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var reqBody struct {
		Name     string `json:"name"`
		ClientID string `json:"clientId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	node, err := h.service.GetHealthyNode()
	if err != nil {
		h.respondWithError(w, http.StatusServiceUnavailable, "No healthy nodes available")
		return
	}
	defer h.service.ReleaseNode(node.ID)

	rpcReq := &myType.CreateQueueRequest{
		Name:     reqBody.Name,
		ClientID: reqBody.ClientID,
	}

	resp, err := h.service.rpcClient.CreateQueue(r.Context(), fmt.Sprintf("%s:%s", node.Address, node.RPCPort), rpcReq)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]string{
		"queueId": resp.QueueID,
	})
}

func (h *Handler) HandleAppendData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var reqBody struct {
		QueueID  string      `json:"queueId"`
		ClientID string      `json:"clientId"`
		Data     interface{} `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	node, err := h.service.GetHealthyNode()
	if err != nil {
		h.respondWithError(w, http.StatusServiceUnavailable, "No healthy nodes available")
		return
	}
	defer h.service.ReleaseNode(node.ID)

	// Convert data to bytes
	dataBytes, err := json.Marshal(reqBody.Data)
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid data format")
		return
	}

	rpcReq := &myType.AppendDataRequest{
		QueueID:  reqBody.QueueID,
		ClientID: reqBody.ClientID,
		Data:     dataBytes,
	}

	resp, err := h.service.rpcClient.AppendData(r.Context(), fmt.Sprintf("%s:%s", node.Address, node.RPCPort), rpcReq)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]string{
		"messageId": resp.MessageID,
	})
}

func (h *Handler) HandleReadData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	queueID := r.URL.Query().Get("queueId")
	clientID := r.URL.Query().Get("clientId")

	if queueID == "" || clientID == "" {
		h.respondWithError(w, http.StatusBadRequest, "Missing required parameters")
		return
	}

	node, err := h.service.GetHealthyNode()
	if err != nil {
		h.respondWithError(w, http.StatusServiceUnavailable, "No healthy nodes available")
		return
	}
	defer h.service.ReleaseNode(node.ID)

	rpcReq := &myType.ReadDataRequest{
		QueueID:  queueID,
		ClientID: clientID,
	}

	resp, err := h.service.rpcClient.ReadData(r.Context(), fmt.Sprintf("%s:%s", node.Address, node.RPCPort), rpcReq)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"messageId": resp.MessageID,
		"data":      resp.Data,
	})
}

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

func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	stats := h.service.GetNodeStats()
	h.respondWithJSON(w, http.StatusOK, stats)
}

// Helper methods for responding
func (h *Handler) respondWithError(w http.ResponseWriter, code int, message string) {
	log.Printf("Error response: %d - %s", code, message)
	h.respondWithJSON(w, code, map[string]string{"error": message})
}

func (h *Handler) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshaling JSON response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if _, err := w.Write(response); err != nil {
		log.Printf("Error writing response: %v", err)
	}
}
