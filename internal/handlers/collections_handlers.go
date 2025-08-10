package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	localmodels "generatio-pb/internal/models"

	"github.com/pocketbase/pocketbase/core"
)

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