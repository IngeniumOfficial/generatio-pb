package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	localmodels "generatio-pb/internal/models"

	"github.com/pocketbase/pocketbase/core"
)

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