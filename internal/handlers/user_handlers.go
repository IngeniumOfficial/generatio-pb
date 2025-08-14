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

	// Calculate recent spending (last 30 days)
	recentSpending, err := h.calculateRecentSpending(user.Id, 30)
	if err != nil {
		recentSpending = 0 // Default to 0 on error
	}

	// Calculate average cost
	var averageCost float64
	if financialData.TotalImages > 0 {
		averageCost = financialData.TotalSpent / float64(financialData.TotalImages)
	}

	resp := localmodels.FinancialStatsResponse{
		TotalSpent:     financialData.TotalSpent,
		TotalImages:    financialData.TotalImages,
		RecentSpending: recentSpending,
		AverageCost:    averageCost,
	}

	return e.JSON(http.StatusOK, resp)
}

// GetPreferences handles POST /api/custom/preferences/get
func (h *Handler) GetPreferences(e *core.RequestEvent) error {
	var req localmodels.GetPreferencesRequest
	if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
		return h.errorResponse(e, http.StatusBadRequest, localmodels.ErrCodeValidation, "Invalid request body")
	}

	if req.ModelName == "" {
		return h.errorResponse(e, http.StatusBadRequest, localmodels.ErrCodeValidation, "Model name is required")
	}

	// Get authenticated user
	user, err := h.getAuthenticatedUser(e)
	if err != nil {
		return h.errorResponse(e, http.StatusUnauthorized, localmodels.ErrCodeAuth, "Authentication required")
	}

	// Find user preferences for this model
	record, err := h.app.FindFirstRecordByFilter(
		"model_preferences",
		"model_name = {:model_name}",
		map[string]any{
			"model_name": req.ModelName,
		},
	)

	resp := localmodels.PreferencesResponse{
		ModelName:      req.ModelName,
		HasPreferences: false,
		Preferences:    make(map[string]interface{}),
	}

	if err == nil && record != nil {
		// Check if this preference record is linked to the current user
		userPrefs := user.Get("model_preferences")
		if userPrefs != nil {
			if prefsList, ok := userPrefs.([]interface{}); ok {
				for _, prefID := range prefsList {
					if prefID == record.Id {
						if prefs := record.Get("preferences"); prefs != nil {
							if prefsMap, ok := prefs.(map[string]interface{}); ok {
								resp.Preferences = prefsMap
								resp.HasPreferences = true
								break
							}
						}
					}
				}
			}
		}
	}

	return e.JSON(http.StatusOK, resp)
}

// SavePreferences handles POST /api/custom/preferences/save
func (h *Handler) SavePreferences(e *core.RequestEvent) error {
	var req localmodels.SavePreferencesRequest
	if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
		return h.errorResponse(e, http.StatusBadRequest, localmodels.ErrCodeValidation, "Invalid request body")
	}

	if req.ModelName == "" {
		return h.errorResponse(e, http.StatusBadRequest, localmodels.ErrCodeValidation, "Model name is required")
	}

	// Get authenticated user
	user, err := h.getAuthenticatedUser(e)
	if err != nil {
		return h.errorResponse(e, http.StatusUnauthorized, localmodels.ErrCodeAuth, "Authentication required")
	}

	// Find existing preferences record for this model
	record, err := h.app.FindFirstRecordByFilter(
		"model_preferences",
		"model_name = {:model_name}",
		map[string]any{
			"model_name": req.ModelName,
		},
	)

	var isNewRecord bool
	if err != nil {
		// Create new record
		collection, err := h.app.FindCollectionByNameOrId("model_preferences")
		if err != nil {
			return h.errorResponse(e, http.StatusInternalServerError, localmodels.ErrCodeInternal, "Failed to find preferences collection")
		}
		record = core.NewRecord(collection)
		record.Set("model_name", req.ModelName)
		isNewRecord = true
	}

	record.Set("preferences", req.Preferences)

	if err := h.app.Save(record); err != nil {
		return h.errorResponse(e, http.StatusInternalServerError, localmodels.ErrCodeInternal, "Failed to save preferences")
	}

	// If new record, link it to the user
	if isNewRecord {
		userPrefs := user.Get("model_preferences")
		var prefsList []interface{}
		if userPrefs != nil {
			if existing, ok := userPrefs.([]interface{}); ok {
				prefsList = existing
			}
		}
		prefsList = append(prefsList, record.Id)
		user.Set("model_preferences", prefsList)
		h.app.Save(user) // Update user with new preference link
	}

	return e.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Preferences saved successfully",
	})
}