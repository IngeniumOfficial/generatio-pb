package handlers

import (
	"generatio-pb/internal/auth"
	"generatio-pb/internal/crypto"
	"generatio-pb/internal/fal"
	localmodels "generatio-pb/internal/models"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// Handler provides all API endpoints for Generatio
type Handler struct {
	app          *pocketbase.PocketBase
	sessionStore *auth.SessionStore
	encService   *crypto.EncryptionService
	falClient    *fal.Client
}

// NewHandler creates a new handler instance
func NewHandler(app *pocketbase.PocketBase, sessionStore *auth.SessionStore, encService *crypto.EncryptionService, falClient *fal.Client) *Handler {
	return &Handler{
		app:          app,
		sessionStore: sessionStore,
		encService:   encService,
		falClient:    falClient,
	}
}

// Helper methods

// getAuthenticatedUser extracts and validates the authenticated user from the request
func (h *Handler) getAuthenticatedUser(e *core.RequestEvent) (*core.Record, error) {
	authRecord := e.Auth
	if authRecord == nil {
		return nil, &localmodels.APIError{Code: localmodels.ErrCodeAuth, Message: "Authentication required"}
	}
	return authRecord, nil
}

// getAuthenticatedUserAndSession extracts user and validates session
func (h *Handler) getAuthenticatedUserAndSession(e *core.RequestEvent) (*core.Record, *localmodels.Session, error) {
	user, err := h.getAuthenticatedUser(e)
	if err != nil {
		return nil, nil, err
	}

	sessionID := e.Request.Header.Get("X-Session-ID")
	if sessionID == "" {
		return nil, nil, &localmodels.APIError{Code: localmodels.ErrCodeAuth, Message: "Session ID required in X-Session-ID header"}
	}

	session, err := h.sessionStore.Get(sessionID)
	if err != nil {
		return nil, nil, &localmodels.APIError{Code: localmodels.ErrCodeAuth, Message: "Invalid or expired session"}
	}

	if session.UserID != user.Id {
		return nil, nil, &localmodels.APIError{Code: localmodels.ErrCodeAuthorization, Message: "Session does not belong to authenticated user"}
	}

	return user, session, nil
}

// errorResponse sends a standardized error response
func (h *Handler) errorResponse(e *core.RequestEvent, status int, code, message string) error {
	apiErr := localmodels.APIError{
		Code:    code,
		Message: message,
	}
	return e.JSON(status, apiErr)
}

// RegisterRoutes registers all the API routes
func RegisterRoutes(se *core.ServeEvent, app *pocketbase.PocketBase, sessionStore *auth.SessionStore, encService *crypto.EncryptionService, falClient *fal.Client) {
	handler := NewHandler(app, sessionStore, encService, falClient)

	// Token management
	se.Router.POST("/api/custom/tokens/setup", handler.TokenSetup)
	se.Router.POST("/api/custom/tokens/verify", handler.TokenVerify)

	// Session management
	se.Router.POST("/api/custom/auth/create-session", handler.CreateSession)
	se.Router.DELETE("/api/custom/auth/session", handler.DeleteSession)

	// Image generation
	se.Router.POST("/api/custom/generate/image", handler.GenerateImage)
	se.Router.GET("/api/custom/generate/models", handler.GetModels)

	// Financial tracking
	se.Router.GET("/api/custom/financial/stats", handler.GetFinancialStats)

	// User preferences
	se.Router.GET("/api/custom/preferences/{model_name}", handler.GetPreferences)
	se.Router.POST("/api/custom/preferences/{model_name}", handler.SavePreferences)

	// Collections management
	se.Router.POST("/api/custom/collections/create", handler.CreateCollection)
	se.Router.GET("/api/custom/collections", handler.GetCollections)
}