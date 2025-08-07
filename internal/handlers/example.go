package handlers

import (
	"encoding/json"
	"net/http"

	"myapp/internal/auth"
	"myapp/internal/crypto"
	"myapp/internal/fal"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// ExampleHandler demonstrates working PocketBase integration
type ExampleHandler struct {
	app          *pocketbase.PocketBase
	sessionStore *auth.SessionStore
	encService   *crypto.EncryptionService
	falClient    *fal.Client
}

// NewExampleHandler creates a new example handler
func NewExampleHandler(app *pocketbase.PocketBase, sessionStore *auth.SessionStore, encService *crypto.EncryptionService, falClient *fal.Client) *ExampleHandler {
	return &ExampleHandler{
		app:          app,
		sessionStore: sessionStore,
		encService:   encService,
		falClient:    falClient,
	}
}

// GetStatus handles GET /api/custom/status
func (h *ExampleHandler) GetStatus(e *core.RequestEvent) error {
	// Get session stats
	sessionStats := h.sessionStore.Stats()
	
	// Get available models
	models := h.falClient.GetModels()
	
	// Create response
	response := map[string]interface{}{
		"status": "ok",
		"message": "Generatio PocketBase extension is running",
		"services": map[string]interface{}{
			"encryption": "AES-256-GCM with PBKDF2",
			"sessions": map[string]interface{}{
				"active": sessionStats.ActiveSessions,
				"total":  sessionStats.TotalSessions,
			},
			"fal_models": len(models),
		},
		"available_models": func() []string {
			var modelNames []string
			for name := range models {
				modelNames = append(modelNames, name)
			}
			return modelNames
		}(),
	}

	return e.JSON(http.StatusOK, response)
}

// TestEncryption handles POST /api/custom/test/encryption
func (h *ExampleHandler) TestEncryption(e *core.RequestEvent) error {
	var req struct {
		Text     string `json:"text"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
	}

	if req.Text == "" || req.Password == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "text and password are required",
		})
	}

	// Test encryption
	result, err := h.encService.Encrypt(req.Text, req.Password)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "encryption failed",
		})
	}

	// Test decryption
	decrypted, err := h.encService.Decrypt(result.Encrypted, result.Salt, req.Password)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "decryption failed",
		})
	}

	response := map[string]interface{}{
		"success": true,
		"original": req.Text,
		"encrypted": result.Encrypted,
		"salt": result.Salt,
		"decrypted": decrypted,
		"match": req.Text == decrypted,
	}

	return e.JSON(http.StatusOK, response)
}

// RegisterExampleRoutes registers example routes to demonstrate functionality
func RegisterExampleRoutes(se *core.ServeEvent, app *pocketbase.PocketBase, sessionStore *auth.SessionStore, encService *crypto.EncryptionService, falClient *fal.Client) {
	handler := NewExampleHandler(app, sessionStore, encService, falClient)

	// Example routes
	se.Router.GET("/api/custom/status", handler.GetStatus)
	se.Router.POST("/api/custom/test/encryption", handler.TestEncryption)
}