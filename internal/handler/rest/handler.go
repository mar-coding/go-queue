package rest

import (
	"encoding/json"
	"github.com/mar-coding/go-queue/internal/service"
	"log"
	"net/http"
)

// Handler handles HTTP requests for the queue service
type Handler struct {
	queueService *service.QueueService
}

// NewHandler creates a new HTTP handler
func NewHandler(queueService *service.QueueService) *Handler {
	return &Handler{
		queueService: queueService,
	}
}

// SetupRoutes configures the HTTP routes
func (h *Handler) SetupRoutes() http.Handler {
	mux := http.NewServeMux()

	// Queue operations
	mux.HandleFunc("/createQueue", h.CreateQueue)
	mux.HandleFunc("/appendData", h.AppendData)
	mux.HandleFunc("/readData", h.ReadData)

	// Node status
	mux.HandleFunc("/status", h.GetStatus)

	return mux
}

// CreateQueue handles queue creation requests
func (h *Handler) CreateQueue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req CreateQueueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	queueID, err := h.queueService.CreateQueue(r.Context(), req.Name)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondWithJSON(w, http.StatusCreated, CreateQueueResponse{
		QueueID: queueID,
		Message: "Queue created successfully",
	})
}

// AppendData handles data append requests
func (h *Handler) AppendData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req AppendDataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	messageID, err := h.queueService.AppendMessage(r.Context(), req.QueueID, req.ClientID, []byte(req.Data))
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondWithJSON(w, http.StatusOK, AppendDataResponse{
		MessageID: messageID,
		Message:   "Data appended successfully",
	})
}

// ReadData handles data read requests
func (h *Handler) ReadData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	queueID := r.URL.Query().Get("queueId")
	if queueID == "" {
		h.respondWithError(w, http.StatusBadRequest, "queueId parameter is required")
		return
	}

	clientID := r.URL.Query().Get("clientId")
	if clientID == "" {
		h.respondWithError(w, http.StatusBadRequest, "clientId parameter is required")
		return
	}

	message, err := h.queueService.ReadMessage(r.Context(), queueID, clientID)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondWithJSON(w, http.StatusOK, ReadDataResponse{
		MessageID: message.ID,
		Data:      string(message.Data),
	})
}

// GetStatus handles node status requests
func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	status := h.queueService.GetNodeStatus()
	h.respondWithJSON(w, http.StatusOK, status)
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
	w.Write(response)
}
