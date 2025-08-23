package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	localmodels "generatio-pb/internal/models"

	"github.com/pocketbase/pocketbase/core"
)

// TokenSetup handles POST /api/custom/tokens/setup
func (h *Handler) TokenSetup(e *core.RequestEvent) error {
	log.Printf("TokenSetup: Received request from %s", e.Request.RemoteAddr)
	log.Printf("TokenSetup: Request headers: %+v", e.Request.Header)
	
	var req localmodels.SetupTokenRequest
	if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
		log.Printf("TokenSetup: Failed to decode request body: %v", err)
		return h.errorResponse(e, http.StatusBadRequest, localmodels.ErrCodeValidation, "Invalid request body")
	}

	log.Printf("TokenSetup: Request decoded successfully, FAL token length: %d", len(req.FALToken))

	if req.FALToken == "" || req.Password == "" {
		log.Printf("TokenSetup: Missing required fields - FAL token empty: %t, Password empty: %t", req.FALToken == "", req.Password == "")
		return h.errorResponse(e, http.StatusBadRequest, localmodels.ErrCodeValidation, "FAL token and password are required")
	}

	// Get authenticated user
	log.Printf("TokenSetup: Attempting to get authenticated user")
	log.Printf("TokenSetup: Auth record present: %t", e.Auth != nil)
	if e.Auth != nil {
		log.Printf("TokenSetup: Auth record collection: %s, ID: %s", e.Auth.Collection().Name, e.Auth.Id)
	}
	
	user, err := h.getAuthenticatedUser(e)
	if err != nil {
		log.Printf("TokenSetup: Authentication failed: %v", err)
		return h.errorResponse(e, http.StatusUnauthorized, localmodels.ErrCodeAuth, "Authentication required")
	}

	log.Printf("TokenSetup: User authenticated successfully - ID: %s, Collection: %s", user.Id, user.Collection().Name)

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

	// Store encrypted data and salt together, separated by period
	combinedToken := encResult.Encrypted + "." + encResult.Salt
	user.Set("fal_token", combinedToken)
	
	// Save to database
	if err := h.app.Save(user); err != nil {
		return h.errorResponse(e, http.StatusInternalServerError, localmodels.ErrCodeInternal, "Failed to save user data")
	}

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

	combinedToken := user.GetString("fal_token")
	
	// Parse encrypted data and salt from combined token (format: "encrypted.salt")
	parts := strings.Split(combinedToken, ".")
	if len(parts) != 2 {
		return h.errorResponse(e, http.StatusBadRequest, localmodels.ErrCodeValidation, "Invalid token format")
	}
	falToken := parts[0]
	salt := parts[1]
	
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

	combinedToken := user.GetString("fal_token")
	
	// Parse encrypted data and salt from combined token (format: "encrypted.salt")
	parts := strings.Split(combinedToken, ".")
	if len(parts) != 2 {
		return h.errorResponse(e, http.StatusBadRequest, localmodels.ErrCodeValidation, "Invalid token format")
	}
	falToken := parts[0]
	salt := parts[1]

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

// TokenStatus handles GET /api/custom/auth/token-status
func (h *Handler) TokenStatus(e *core.RequestEvent) error {
	// Get authenticated user
	user, err := h.getAuthenticatedUser(e)
	if err != nil {
		return h.errorResponse(e, http.StatusUnauthorized, localmodels.ErrCodeAuth, "Authentication required")
	}

	log.Printf("TokenStatus: Checking token status for user %s", user.Id)

	// Check if user has stored encrypted token
	combinedToken := user.GetString("fal_token")
	hasToken := combinedToken != ""

	// Check if user has active session
	hasActiveSession := false
	if hasToken {
		// Check if user has any active sessions
		_, err := h.sessionStore.GetUserSession(user.Id)
		hasActiveSession = err == nil
	}

	// Determine if login is required
	requiresLogin := hasToken && !hasActiveSession

	response := localmodels.TokenStatusResponse{
		HasToken:         hasToken,
		HasActiveSession: hasActiveSession,
		RequiresLogin:    requiresLogin,
	}

	log.Printf("TokenStatus: User %s - HasToken: %t, HasActiveSession: %t, RequiresLogin: %t",
		user.Id, hasToken, hasActiveSession, requiresLogin)

	return e.JSON(http.StatusOK, response)
}