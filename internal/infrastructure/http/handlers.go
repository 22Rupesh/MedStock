package http

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"medstock/internal/application"
	clientdomain "medstock/internal/domain"
	inventory "medstock/internal/domain/inventory"
	"medstock/internal/middleware"
	"medstock/pkg/logger"
	"medstock/pkg/validator"
)

type Handler struct {
	uploadService  *application.UploadService
	catalogService *application.CatalogService
	log            *logger.Logger
}

func NewHandler(upload *application.UploadService, catalog *application.CatalogService, log *logger.Logger) *Handler {
	return &Handler{
		uploadService:  upload,
		catalogService: catalog,
		log:            log,
	}
}

func (h *Handler) RegisterRoutes(r *mux.Router, auth *middleware.APIKeyAuth) {
	api := r.PathPrefix("/api/v1").Subrouter()
	api.Use(auth.Middleware)

	api.HandleFunc("/bulk-upload", h.HandleBulkUpload).Methods("POST")
	api.HandleFunc("/uploads/{session_id}", h.GetUploadStatus).Methods("GET")
	api.HandleFunc("/products", h.ListProducts).Methods("GET")
	api.HandleFunc("/products/{sku}", h.GetProduct).Methods("GET")

	r.HandleFunc("/healthz", h.HealthCheck).Methods("GET")
	r.HandleFunc("/healthz/ready", h.ReadinessCheck).Methods("GET")
}

func (h *Handler) HandleBulkUpload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqID := middleware.GetRequestIDFromContext(ctx)
	h.log = h.log.WithRequestID(ctx)

	var req struct {
		SessionID      string                 `json:"session_id"`
		IdempotencyKey string                `json:"idempotency_key" validate:"required,max=128"`
		Items          []map[string]interface{} `json:"items" validate:"required,min=1"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid json payload")
		return
	}

	if err := validator.Struct(req); err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	sessionID, err := uuid.Parse(req.SessionID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid session_id format")
		return
	}

	client := middleware.GetClientFromContext(ctx)
	clientID, ok := client.(*clientdomain.Client)
	if !ok || clientID == nil {
		h.writeError(w, http.StatusUnauthorized, "invalid client context")
		return
	}

	bulkReq := &inventory.BulkUploadRequest{
		SessionID:      sessionID,
		IdempotencyKey: req.IdempotencyKey,
		Items:          h.mapToBulkItems(req.Items),
	}

	resp, err := h.uploadService.ProcessBulkUpload(ctx, clientID.ID, bulkReq)
	if err != nil {
		switch err {
		case application.ErrDuplicateUpload:
			h.writeError(w, http.StatusConflict, "duplicate upload in progress")
		case application.ErrInvalidItemCount:
			h.writeError(w, http.StatusBadRequest, "item count exceeds maximum")
		default:
			h.log.Error().Err(err).Str("request_id", reqID).Msg("bulk_upload_failed")
			h.writeError(w, http.StatusInternalServerError, "processing failed")
		}
		return
	}

	h.writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetUploadStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	sessionID, err := uuid.Parse(vars["session_id"])
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid session_id format")
		return
	}

	upload, err := h.uploadService.GetUploadStatus(ctx, sessionID)
	if err != nil {
		if err == application.ErrUploadNotFound {
			h.writeError(w, http.StatusNotFound, "upload not found")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "failed to retrieve status")
		return
	}

	h.writeJSON(w, http.StatusOK, upload)
}

func (h *Handler) ListProducts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	client := middleware.GetClientFromContext(ctx)
	clientID, ok := client.(*clientdomain.Client)
	if !ok || clientID == nil {
		h.writeError(w, http.StatusUnauthorized, "invalid client context")
		return
	}

	page := 1
	pageSize := 20

	resp, err := h.catalogService.ListProducts(ctx, clientID.ID, page, pageSize)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to list products")
		return
	}

	h.writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetProduct(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	client := middleware.GetClientFromContext(ctx)
	clientID, ok := client.(*clientdomain.Client)
	if !ok || clientID == nil {
		h.writeError(w, http.StatusUnauthorized, "invalid client context")
		return
	}

	product, err := h.catalogService.GetProduct(ctx, clientID.ID, vars["sku"])
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to get product")
		return
	}
	if product == nil {
		h.writeError(w, http.StatusNotFound, "product not found")
		return
	}

	h.writeJSON(w, http.StatusOK, product)
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) ReadinessCheck(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]string{"error": message})
}

func (h *Handler) mapToBulkItems(items []map[string]interface{}) []application.BulkItemInput {
	result := make([]application.BulkItemInput, len(items))
	for i, item := range items {
		result[i] = application.BulkItemInput{
			SKU:              getString(item, "sku"),
			Name:             getString(item, "name"),
			Quantity:         getInt(item, "quantity"),
			ReservedQuantity: getInt(item, "reserved_quantity"),
			UnitPrice:        getFloat(item, "unit_price"),
			Currency:         getString(item, "currency"),
		}
	}
	return result
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}

func getFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return 0
}