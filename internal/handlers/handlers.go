package handlers

import (
	"generatio-pb/internal/auth"
	"generatio-pb/internal/crypto"
	"generatio-pb/internal/fal"
	localmodels "generatio-pb/internal/models"
	"time"

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

// updateUserFinancialData updates user's financial tracking data
func (h *Handler) updateUserFinancialData(user *core.Record, cost float64, imageCount int) {
	financialDataRaw := user.Get("financial_data")
	var financialData localmodels.FinancialData
	
	if financialDataRaw != nil {
		if data, ok := financialDataRaw.(map[string]interface{}); ok {
			if totalSpent, ok := data["total_spent"].(float64); ok {
				financialData.TotalSpent = totalSpent
			}
			if totalImages, ok := data["total_images"].(float64); ok {
				financialData.TotalImages = int(totalImages)
			}
		}
	}

	// Update with new spending
	financialData.TotalSpent += cost
	financialData.TotalImages += imageCount

	// Save back to user record
	user.Set("financial_data", map[string]interface{}{
		"total_spent":  financialData.TotalSpent,
		"total_images": financialData.TotalImages,
	})

	// Save user record (ignore errors for financial data updates)
	h.app.Save(user)
}

// calculateRecentSpending calculates spending in the last N days
func (h *Handler) calculateRecentSpending(userID string, days int) (float64, error) {
	// Calculate date threshold
	threshold := time.Now().AddDate(0, 0, -days)
	
	records, err := h.app.FindRecordsByFilter(
		"images",
		"user_id = {:user_id} && created >= {:threshold} && deleted_at = null",
		"",
		-1,
		0,
		map[string]any{
			"user_id":   userID,
			"threshold": threshold.Format("2006-01-02 15:04:05"),
		},
	)

	if err != nil {
		return 0, err
	}

	var total float64
	for _, record := range records {
		// Cost is stored in other_info JSON field
		if otherInfo := record.Get("other_info"); otherInfo != nil {
			if data, ok := otherInfo.(map[string]interface{}); ok {
				if cost, exists := data["cost_usd"]; exists {
					if costFloat, ok := cost.(float64); ok {
						total += costFloat
					}
				}
			}
		}
	}

	return total, nil
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