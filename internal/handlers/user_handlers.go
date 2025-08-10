package handlers

import (
	"encoding/json"
	"net/http"

	localmodels "generatio-pb/internal/models"

	"github.com/pocketbase/pocketbase/core"
)

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