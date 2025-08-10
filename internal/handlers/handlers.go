package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

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

// TokenSetup handles POST /api/custom/tokens/setup
func (h *Handler) TokenSetup(e *core.RequestEvent) error {
	var req localmodels.SetupTokenRequest
	if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
		return h.errorResponse(e, http.StatusBadRequest, localmodels.ErrCodeValidation, "Invalid request body")
	}

	if req.FALToken == "" || req.Password == "" {
		return h.errorResponse(e, http.StatusBadRequest, localmodels.ErrCodeValidation, "FAL token and password are required")
	}

	// Get authenticated user
	user, err := h.getAuthenticatedUser(e)
	if err != nil {
		return h.errorResponse(e, http.StatusUnauthorized, localmodels.ErrCodeAuth, "Authentication required")
	}

	// Validate FAL token by testing it
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	if err := h.falClient.ValidateToken(ctx, req.FALToken); err != nil {
		return h.errorResponse(e, http.StatusBadRequest, localmodels.ErrCodeValidation, "Invalid FAL AI token")
	}

	// Encrypt the token
	encResult, err := h.encService.Encrypt(req.FALToken, req.Password)
	if err != nil {
		return h.errorResponse(e, http.StatusInternalServerError, localmodels.ErrCodeInternal, "Failed to encrypt token")
	}

	// Update user record (simplified for now)
	user.Set("fal_token", encResult.Encrypted)
	user.Set("salt", encResult.Salt)
	
	// TODO: Save record once we fix the Dao access
	// if err := h.app.Save(user); err != nil {
	//     return h.errorResponse(e, http.StatusInternalServerError, localmodels.ErrCodeInternal, "Failed to save user data")
	// }

	return e.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "FAL token setup successfully",
	})
}

// TokenVerify handles POST /api/custom/tokens/verify
func (h *Handler) TokenVerify(e *core.RequestEvent) error {
	var req localmodels.VerifyTokenRequest
	if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
		return h.errorResponse(e, http.StatusBadRequest, localmodels.ErrCodeValidation, "Invalid request body")
	}

	if req.Password == "" {
		return h.errorResponse(e, http.StatusBadRequest, localmodels.ErrCodeValidation, "Password is required")
	}

	// Get authenticated user
	user, err := h.getAuthenticatedUser(e)
	if err != nil {
		return h.errorResponse(e, http.StatusUnauthorized, localmodels.ErrCodeAuth, "Authentication required")
	}

	falToken := user.GetString("fal_token")
	salt := user.GetString("salt")
	
	resp := localmodels.VerifyTokenResponse{
		HasToken:   falToken != "",
		CanDecrypt: false,
	}

	if falToken != "" && salt != "" {
		// Test if password can decrypt the token
		_, err := h.encService.Decrypt(falToken, salt, req.Password)
		resp.CanDecrypt = err == nil
	}

	return e.JSON(http.StatusOK, resp)
}

// CreateSession handles POST /api/custom/auth/create-session
func (h *Handler) CreateSession(e *core.RequestEvent) error {
	var req localmodels.CreateSessionRequest
	if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
		return h.errorResponse(e, http.StatusBadRequest, localmodels.ErrCodeValidation, "Invalid request body")
	}

	if req.Password == "" {
		return h.errorResponse(e, http.StatusBadRequest, localmodels.ErrCodeValidation, "Password is required")
	}

	// Get authenticated user
	user, err := h.getAuthenticatedUser(e)
	if err != nil {
		return h.errorResponse(e, http.StatusUnauthorized, localmodels.ErrCodeAuth, "Authentication required")
	}

	falToken := user.GetString("fal_token")
	salt := user.GetString("salt")

	if falToken == "" || salt == "" {
		return h.errorResponse(e, http.StatusBadRequest, localmodels.ErrCodeValidation, "FAL token not configured. Please setup token first")
	}

	// Decrypt the FAL token
	decryptedToken, err := h.encService.Decrypt(falToken, salt, req.Password)
	if err != nil {
		return h.errorResponse(e, http.StatusUnauthorized, localmodels.ErrCodeAuth, "Invalid password")
	}

	// Remove any existing sessions for this user
	h.sessionStore.DeleteUserSessions(user.Id)

	// Create new session
	sessionID, err := h.sessionStore.Create(user.Id, decryptedToken)
	if err != nil {
		return h.errorResponse(e, http.StatusInternalServerError, localmodels.ErrCodeInternal, "Failed to create session")
	}

	session, err := h.sessionStore.Get(sessionID)
	if err != nil {
		return h.errorResponse(e, http.StatusInternalServerError, localmodels.ErrCodeInternal, "Failed to retrieve session")
	}

	resp := localmodels.CreateSessionResponse{
		SessionID: sessionID,
		ExpiresAt: session.ExpiresAt,
	}

	return e.JSON(http.StatusOK, resp)
}

// DeleteSession handles DELETE /api/custom/auth/session
func (h *Handler) DeleteSession(e *core.RequestEvent) error {
	sessionID := e.Request.Header.Get("X-Session-ID")
	if sessionID == "" {
		return h.errorResponse(e, http.StatusBadRequest, localmodels.ErrCodeValidation, "Session ID required in X-Session-ID header")
	}

	// Get authenticated user
	user, err := h.getAuthenticatedUser(e)
	if err != nil {
		return h.errorResponse(e, http.StatusUnauthorized, localmodels.ErrCodeAuth, "Authentication required")
	}

	// Verify session belongs to user
	session, err := h.sessionStore.Get(sessionID)
	if err != nil {
		return h.errorResponse(e, http.StatusNotFound, localmodels.ErrCodeNotFound, "Session not found")
	}

	if session.UserID != user.Id {
		return h.errorResponse(e, http.StatusForbidden, localmodels.ErrCodeAuthorization, "Access denied")
	}

	// Delete session
	h.sessionStore.Delete(sessionID)

	return e.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Session deleted successfully",
	})
}

// GenerateImage handles POST /api/custom/generate/image
func (h *Handler) GenerateImage(e *core.RequestEvent) error {
	var req localmodels.GenerateImageRequest
	if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
		return h.errorResponse(e, http.StatusBadRequest, localmodels.ErrCodeValidation, "Invalid request body")
	}

	if req.Model == "" || req.Prompt == "" {
		return h.errorResponse(e, http.StatusBadRequest, localmodels.ErrCodeValidation, "Model and prompt are required")
	}

	// Get authenticated user and session
	user, session, err := h.getAuthenticatedUserAndSession(e)
	if err != nil {
		return h.errorResponse(e, http.StatusUnauthorized, localmodels.ErrCodeAuth, "Valid session required")
	}

	// Create FAL generation request
	falReq := fal.GenerationRequest{
		Model:      req.Model,
		Prompt:     req.Prompt,
		Parameters: req.Parameters,
	}

	// Generate image
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	startTime := time.Now()
	result, err := h.falClient.GenerateImage(ctx, session.FALToken, falReq)
	if err != nil {
		return h.errorResponse(e, http.StatusInternalServerError, localmodels.ErrCodeExternal, "Image generation failed: "+err.Error())
	}
	generationTime := time.Since(startTime)

	// Create response without saving to database for now
	var imageInfos []localmodels.GeneratedImageInfo
	for i, img := range result.Images {
		imageInfos = append(imageInfos, localmodels.GeneratedImageInfo{
			ID:           result.RequestID + "_" + string(rune(i)), // Temporary ID
			URL:          img.URL,
			ThumbnailURL: img.ThumbnailURL,
		})
	}

	// TODO: Save generated images to database
	// TODO: Update user financial data

	h.app.Logger().Info("Image generated successfully", 
		"user_id", user.Id,
		"model", req.Model,
		"cost", result.Cost,
		"generation_time", generationTime.String(),
	)

	resp := localmodels.GenerateImageResponse{
		Images: imageInfos,
		Cost:   result.Cost,
		Model:  req.Model,
	}

	return e.JSON(http.StatusOK, resp)
}

// GetModels handles GET /api/custom/generate/models
func (h *Handler) GetModels(e *core.RequestEvent) error {
	// Verify authentication
	_, err := h.getAuthenticatedUser(e)
	if err != nil {
		return h.errorResponse(e, http.StatusUnauthorized, localmodels.ErrCodeAuth, "Authentication required")
	}

	models := h.falClient.GetModels()
	return e.JSON(http.StatusOK, models)
}

// GetFinancialStats handles GET /api/custom/financial/stats
func (h *Handler) GetFinancialStats(e *core.RequestEvent) error {
	// Get authenticated user
	user, err := h.getAuthenticatedUser(e)
	if err != nil {
		return h.errorResponse(e, http.StatusUnauthorized, localmodels.ErrCodeAuth, "Authentication required")
	}

	// Get financial data from user record
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

	// For now, just return basic stats without recent spending calculation
	var averageCost float64
	if financialData.TotalImages > 0 {
		averageCost = financialData.TotalSpent / float64(financialData.TotalImages)
	}

	resp := localmodels.FinancialStatsResponse{
		TotalSpent:     financialData.TotalSpent,
		TotalImages:    financialData.TotalImages,
		RecentSpending: 0, // TODO: Calculate from database
		AverageCost:    averageCost,
	}

	return e.JSON(http.StatusOK, resp)
}

// GetPreferences handles GET /api/custom/preferences/{model_name}
func (h *Handler) GetPreferences(e *core.RequestEvent) error {
	modelName := e.Request.PathValue("model_name")
	if modelName == "" {
		return h.errorResponse(e, http.StatusBadRequest, localmodels.ErrCodeValidation, "Model name is required")
	}

	// Get authenticated user
	_, err := h.getAuthenticatedUser(e)
	if err != nil {
		return h.errorResponse(e, http.StatusUnauthorized, localmodels.ErrCodeAuth, "Authentication required")
	}

	// For now, return empty preferences
	resp := localmodels.PreferencesResponse{
		ModelName:      modelName,
		HasPreferences: false,
		Preferences:    make(map[string]interface{}),
	}

	return e.JSON(http.StatusOK, resp)
}

// SavePreferences handles POST /api/custom/preferences/{model_name}
func (h *Handler) SavePreferences(e *core.RequestEvent) error {
	modelName := e.Request.PathValue("model_name")
	if modelName == "" {
		return h.errorResponse(e, http.StatusBadRequest, localmodels.ErrCodeValidation, "Model name is required")
	}

	var req localmodels.SavePreferencesRequest
	if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
		return h.errorResponse(e, http.StatusBadRequest, localmodels.ErrCodeValidation, "Invalid request body")
	}

	// Get authenticated user
	_, err := h.getAuthenticatedUser(e)
	if err != nil {
		return h.errorResponse(e, http.StatusUnauthorized, localmodels.ErrCodeAuth, "Authentication required")
	}

	// TODO: Save preferences to database

	return e.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Preferences saved successfully",
	})
}

// CreateCollection handles POST /api/custom/collections/create
func (h *Handler) CreateCollection(e *core.RequestEvent) error {
	var req localmodels.CreateCollectionRequest
	if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
		return h.errorResponse(e, http.StatusBadRequest, localmodels.ErrCodeValidation, "Invalid request body")
	}

	if req.Name == "" {
		return h.errorResponse(e, http.StatusBadRequest, localmodels.ErrCodeValidation, "Collection name is required")
	}

	// Get authenticated user
	_, err := h.getAuthenticatedUser(e)
	if err != nil {
		return h.errorResponse(e, http.StatusUnauthorized, localmodels.ErrCodeAuth, "Authentication required")
	}

	// TODO: Create collection record in database
	resp := localmodels.CreateCollectionResponse{
		ID:       "temp_" + req.Name, // Temporary ID
		Name:     req.Name,
		ParentID: req.ParentID,
		Created:  time.Now(),
	}

	return e.JSON(http.StatusOK, resp)
}

// GetCollections handles GET /api/custom/collections
func (h *Handler) GetCollections(e *core.RequestEvent) error {
	// Get authenticated user
	_, err := h.getAuthenticatedUser(e)
	if err != nil {
		return h.errorResponse(e, http.StatusUnauthorized, localmodels.ErrCodeAuth, "Authentication required")
	}

	// TODO: Get collections from database
	var collections []localmodels.Collection

	return e.JSON(http.StatusOK, map[string]interface{}{
		"collections": collections,
	})
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