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

	// Save generated images to database and create response
	var imageInfos []localmodels.GeneratedImageInfo
	for i, img := range result.Images {
		// Create generated image record
		collection, err := h.app.FindCollectionByNameOrId("images")
		if err == nil && collection != nil {
			imageRecord := core.NewRecord(collection)
			imageRecord.Set("title", req.Prompt) // Use prompt as title
			imageRecord.Set("url", img.URL)
			imageRecord.Set("user_id", user.Id)
			imageRecord.Set("prompt", req.Prompt)
			imageRecord.Set("request_id", result.RequestID)
			imageRecord.Set("model", req.Model)
			imageRecord.Set("batch_number", float64(i+1)) // Batch number for this image
			
			// Set image size from parameters or default
			imageSize := map[string]interface{}{
				"width":  1024, // Default
				"height": 1024, // Default
			}
			if req.Parameters != nil {
				if size, exists := req.Parameters["image_size"]; exists {
					if sizeObj, ok := size.(map[string]interface{}); ok {
						imageSize = sizeObj
					}
				}
			}
			imageRecord.Set("image_size", imageSize)
			
			// Store generation info in other_info
			otherInfo := map[string]interface{}{
				"cost_usd":           result.Cost / float64(len(result.Images)),
				"generation_time_ms": generationTime.Milliseconds(),
				"parameters":         req.Parameters,
			}
			imageRecord.Set("other_info", otherInfo)
			
			// Set folder if provided (renamed from collection)
			if req.CollectionID != "" {
				imageRecord.Set("folder_id", req.CollectionID)
			}

			if err := h.app.Save(imageRecord); err != nil {
				// Log error but don't fail the request
				h.app.Logger().Error("Failed to save image record", "error", err)
			}

			imageInfos = append(imageInfos, localmodels.GeneratedImageInfo{
				ID:           imageRecord.Id,
				URL:          img.URL,
				ThumbnailURL: img.ThumbnailURL,
			})
		} else {
			// Fallback if collection doesn't exist
			imageInfos = append(imageInfos, localmodels.GeneratedImageInfo{
				ID:           result.RequestID + "_" + string(rune(i)),
				URL:          img.URL,
				ThumbnailURL: img.ThumbnailURL,
			})
		}
	}

	// Update user financial data
	h.updateUserFinancialData(user, result.Cost, len(result.Images))

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