package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"generatio-pb/internal/fal"
	localmodels "generatio-pb/internal/models"

	"github.com/pocketbase/pocketbase/core"
)

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