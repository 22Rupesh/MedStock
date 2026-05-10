package main

import (
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/google/uuid"
	"medstock/pkg/logger"
)

type MockClient struct {
	ID        uuid.UUID
	Name      string
	APIKey    string
	RateLimit int
}

type MockRepository struct {
	mu       sync.RWMutex
	clients  map[string]*MockClient
	uploads  map[string]*UploadStatus
	products map[string]*MockProduct
}

type MockProduct struct {
	ID        uuid.UUID `json:"id"`
	SKU       string    `json:"sku"`
	ClientID  uuid.UUID `json:"client_id"`
	Name      string    `json:"name"`
	Quantity  int       `json:"quantity"`
	UnitPrice float64   `json:"unit_price"`
	Category  string    `json:"category"`
	CreatedAt time.Time `json:"created_at"`
}

type UploadStatus struct {
	ID             uuid.UUID  `json:"id"`
	SessionID      uuid.UUID  `json:"session_id"`
	ClientID       uuid.UUID  `json:"client_id"`
	IdempotencyKey string     `json:"idempotency_key"`
	TotalItems     int        `json:"total_items"`
	ProcessedItems  int        `json:"processed_items"`
	FailedItems     int        `json:"failed_items"`
	Status         string     `json:"status"`
	ErrorSummary   *string     `json:"error_summary,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

func NewMockRepository() *MockRepository {
	repo := &MockRepository{
		clients:  make(map[string]*MockClient),
		uploads:  make(map[string]*UploadStatus),
		products: make(map[string]*MockProduct),
	}
	repo.clients["test-api-key-123"] = &MockClient{
		ID:        uuid.New(),
		Name:      "Test Client",
		APIKey:    "test-api-key-123",
		RateLimit: 100,
	}
	repo.clients["demo-api-key"] = &MockClient{
		ID:        uuid.New(),
		Name:      "Demo Client",
		APIKey:    "demo-api-key",
		RateLimit: 100,
	}

	// Seed some initial products
	seedProducts := []MockProduct{
		{ID: uuid.New(), SKU: "MED-001", Name: "Surgical Gloves", Quantity: 5000, UnitPrice: 0.50, Category: "PPE"},
		{ID: uuid.New(), SKU: "MED-002", Name: "Face Shield", Quantity: 200, UnitPrice: 5.99, Category: "PPE"},
		{ID: uuid.New(), SKU: "MED-003", Name: "N95 Mask", Quantity: 1000, UnitPrice: 2.50, Category: "PPE"},
		{ID: uuid.New(), SKU: "MED-004", Name: "Hand Sanitizer 500ml", Quantity: 800, UnitPrice: 3.99, Category: "Sanitization"},
		{ID: uuid.New(), SKU: "MED-005", Name: "Thermometer Digital", Quantity: 150, UnitPrice: 15.99, Category: "Diagnostic"},
	}
	for _, p := range seedProducts {
		p.CreatedAt = time.Now()
		repo.products[p.SKU] = &p
	}

	return repo
}

var globalRepo = NewMockRepository()

type BulkItemInput struct {
	SKU       string  `json:"sku" validate:"required"`
	Name      string  `json:"name" validate:"required"`
	Quantity  int     `json:"quantity" validate:"gte=0"`
	UnitPrice float64 `json:"unit_price" validate:"gte=0"`
}

type BulkUploadResponse struct {
	SessionID    uuid.UUID `json:"session_id"`
	Status       string    `json:"status"`
	TotalItems   int       `json:"total_items"`
	Processed    int       `json:"processed"`
	Failed       int       `json:"failed"`
	Progress     float64   `json:"progress"`
	DurationMs   int64     `json:"duration_ms"`
	IsIdempotent bool      `json:"is_idempotent"`
}

type ProductListResponse struct {
	Products   []MockProduct    `json:"products"`
	Pagination Pagination       `json:"pagination"`
}

type Pagination struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

func main() {
	log := logger.New()
	log.Info().Msg("starting medstock api (mock mode)")
	log.Info().Str("products", "5 seeded").Msg("database initialized")
	log.Info().Msg("connected to redis (mock)")

	router := mux.NewRouter()
	router.Use(corsMiddleware)
	router.Use(requestLogger(log))

	router.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		sendJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	router.HandleFunc("/healthz/ready", func(w http.ResponseWriter, r *http.Request) {
		sendJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})

	api := router.PathPrefix("/api/v1").Subrouter()
	api.Use(apiKeyAuth)

	api.HandleFunc("/bulk-upload", handleBulkUpload).Methods("POST", "OPTIONS")
	api.HandleFunc("/uploads/{session_id}", handleGetUploadStatus).Methods("GET", "OPTIONS")
	api.HandleFunc("/products", handleListProducts).Methods("GET", "OPTIONS")
	api.HandleFunc("/products/{sku}", handleGetProduct).Methods("GET", "OPTIONS")
	api.HandleFunc("/products/{sku}", handleUpdateProduct).Methods("PUT", "OPTIONS")
	api.HandleFunc("/products/{sku}", handleDeleteProduct).Methods("DELETE", "OPTIONS")
	api.HandleFunc("/stats", handleGetStats).Methods("GET", "OPTIONS")

	server := &http.Server{
		Addr:         "0.0.0.0:8080",
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Info().Str("addr", server.Addr).Msg("server starting")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down server...")
	log.Info().Msg("server stopped")
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func apiKeyAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusOK)
			return
		}

		// Allow requests without auth for testing (for real production, remove this)
		if r.URL.Path == "/api/v1/products" || r.URL.Path == "/api/v1/bulk-upload" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error": "missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		parts := splitAuth(authHeader)
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, `{"error": "invalid authorization format"}`, http.StatusUnauthorized)
			return
		}

		key := parts[1]
		globalRepo.mu.RLock()
		_, ok := globalRepo.clients[key]
		globalRepo.mu.RUnlock()

		if !ok {
			http.Error(w, `{"error": "invalid api key"}`, http.StatusUnauthorized)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", "*")
		next.ServeHTTP(w, r)
	})
}

func splitAuth(header string) []string {
	for i := 0; i < len(header); i++ {
		if header[i] == ' ' {
			return []string{header[:i], header[i+1:]}
		}
	}
	return nil
}

func requestLogger(log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rw, r)
			log.Info().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", rw.statusCode).
				Int64("duration_ms", time.Since(start).Milliseconds()).
				Msg("http_request")
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func handleBulkUpload(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionID      string        `json:"session_id"`
		IdempotencyKey string        `json:"idempotency_key"`
		Items          []BulkItemInput `json:"items"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "invalid json payload")
		return
	}

	if len(req.Items) == 0 {
		sendError(w, http.StatusBadRequest, "no items to upload")
		return
	}

	sessionID, err := uuid.Parse(req.SessionID)
	if err != nil {
		sendError(w, http.StatusBadRequest, "invalid session_id")
		return
	}

	startTime := time.Now()

	// Check for duplicate idempotency key
	globalRepo.mu.RLock()
	existingUpload, exists := globalRepo.uploads[req.SessionID]
	globalRepo.mu.RUnlock()

	if exists {
		resp := BulkUploadResponse{
			SessionID:    sessionID,
			Status:       existingUpload.Status,
			TotalItems:   existingUpload.TotalItems,
			Processed:   existingUpload.ProcessedItems,
			Failed:      existingUpload.FailedItems,
			Progress:    100,
			DurationMs:  time.Since(startTime).Milliseconds(),
			IsIdempotent: true,
		}
		sendJSON(w, http.StatusOK, resp)
		return
	}

	var processed, failed int
	now := time.Now()

	for i, item := range req.Items {
		// Simulate processing
		time.Sleep(10 * time.Millisecond)

		// 95% success rate
		if i%20 != 0 {
			processed++
			globalRepo.mu.Lock()
			globalRepo.products[item.SKU] = &MockProduct{
				ID:        uuid.New(),
				SKU:       item.SKU,
				ClientID:  uuid.New(),
				Name:      item.Name,
				Quantity:  item.Quantity,
				UnitPrice: item.UnitPrice,
				CreatedAt: now,
			}
			globalRepo.mu.Unlock()
		} else {
			failed++
		}
	}

	status := "completed"
	if failed > 0 && processed == 0 {
		status = "failed"
	} else if failed > 0 {
		status = "partial_failure"
	}

	completedAt := time.Now()
	upload := &UploadStatus{
		ID:             uuid.New(),
		SessionID:      sessionID,
		IdempotencyKey: req.IdempotencyKey,
		TotalItems:     len(req.Items),
		ProcessedItems: processed,
		FailedItems:    failed,
		Status:         status,
		CreatedAt:      now,
		CompletedAt:    &completedAt,
	}

	globalRepo.mu.Lock()
	globalRepo.uploads[req.SessionID] = upload
	globalRepo.mu.Unlock()

	resp := BulkUploadResponse{
		SessionID:    sessionID,
		Status:       status,
		TotalItems:   len(req.Items),
		Processed:    processed,
		Failed:       failed,
		Progress:     100,
		DurationMs:   time.Since(startTime).Milliseconds(),
		IsIdempotent: false,
	}
	sendJSON(w, http.StatusOK, resp)
}

func handleGetUploadStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["session_id"]

	globalRepo.mu.RLock()
	upload, ok := globalRepo.uploads[sessionID]
	globalRepo.mu.RUnlock()

	if !ok {
		sendError(w, http.StatusNotFound, "upload not found")
		return
	}
	sendJSON(w, http.StatusOK, upload)
}

func handleListProducts(w http.ResponseWriter, r *http.Request) {
	globalRepo.mu.RLock()
	products := make([]MockProduct, 0, len(globalRepo.products))
	for _, p := range globalRepo.products {
		products = append(products, *p)
	}
	globalRepo.mu.RUnlock()

	totalItems := len(products)
	totalPages := 1
	if totalItems > 20 {
		totalPages = (totalItems + 19) / 20
	}

	resp := ProductListResponse{
		Products: products,
		Pagination: Pagination{
			Page:       1,
			PageSize:   20,
			TotalItems: totalItems,
			TotalPages: totalPages,
		},
	}
	sendJSON(w, http.StatusOK, resp)
}

func handleGetProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sku := vars["sku"]

	globalRepo.mu.RLock()
	product, ok := globalRepo.products[sku]
	globalRepo.mu.RUnlock()

	if !ok {
		sendError(w, http.StatusNotFound, "product not found")
		return
	}
	sendJSON(w, http.StatusOK, product)
}

func handleUpdateProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sku := vars["sku"]

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		sendError(w, http.StatusBadRequest, "invalid json")
		return
	}

	globalRepo.mu.Lock()
	defer globalRepo.mu.Unlock()

	product, ok := globalRepo.products[sku]
	if !ok {
		sendError(w, http.StatusNotFound, "product not found")
		return
	}

	if qty, ok := updates["quantity"].(float64); ok {
		product.Quantity = int(qty)
	}
	if price, ok := updates["unit_price"].(float64); ok {
		product.UnitPrice = price
	}
	if name, ok := updates["name"].(string); ok {
		product.Name = name
	}

	globalRepo.products[sku] = product
	sendJSON(w, http.StatusOK, product)
}

func handleDeleteProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sku := vars["sku"]

	globalRepo.mu.Lock()
	defer globalRepo.mu.Unlock()

	if _, ok := globalRepo.products[sku]; !ok {
		sendError(w, http.StatusNotFound, "product not found")
		return
	}

	delete(globalRepo.products, sku)
	sendJSON(w, http.StatusOK, map[string]string{"message": "product deleted"})
}

func handleGetStats(w http.ResponseWriter, r *http.Request) {
	globalRepo.mu.RLock()
	products := globalRepo.products
	globalRepo.mu.RUnlock()

	var totalValue float64
	var totalQuantity int
	lowStockCount := 0

	for _, p := range products {
		totalValue += p.UnitPrice * float64(p.Quantity)
		totalQuantity += p.Quantity
		if p.Quantity < 50 {
			lowStockCount++
		}
	}

	stats := map[string]interface{}{
		"total_products":   len(products),
		"total_quantity":  totalQuantity,
		"total_value":     totalValue,
		"low_stock_count": lowStockCount,
		"categories": map[string]int{
			"PPE": 3,
			"Sanitization": 1,
			"Diagnostic": 1,
		},
	}
	sendJSON(w, http.StatusOK, stats)
}

func sendJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func sendError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}