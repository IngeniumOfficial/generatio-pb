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
	user, err := h.getAuthenticatedUser(e)
	if err != nil {
		return h.errorResponse(e, http.StatusUnauthorized, localmodels.ErrCodeAuth, "Authentication required")
	}

	// Create folder record (collections are called folders in the schema)
	collection, err := h.app.FindCollectionByNameOrId("folders")
	if err != nil {
		return h.errorResponse(e, http.StatusInternalServerError, localmodels.ErrCodeInternal, "Failed to find folders collection")
	}
	
	record := core.NewRecord(collection)
	record.Set("user_id", user.Id)
	record.Set("name", req.Name)
	record.Set("private", false) // Default to public
	
	if req.ParentID != "" {
		record.Set("parent_id", req.ParentID)
	}

	if err := h.app.Save(record); err != nil {
		return h.errorResponse(e, http.StatusInternalServerError, localmodels.ErrCodeInternal, "Failed to create folder")
	}

	resp := localmodels.CreateCollectionResponse{
		ID:       record.Id,
		Name:     req.Name,
		ParentID: req.ParentID,
		Created:  time.Now(), // Fallback until we fix timestamp access
	}

	return e.JSON(http.StatusOK, resp)
}

// GetCollections handles GET /api/custom/collections
func (h *Handler) GetCollections(e *core.RequestEvent) error {
	// Get authenticated user
	user, err := h.getAuthenticatedUser(e)
	if err != nil {
		return h.errorResponse(e, http.StatusUnauthorized, localmodels.ErrCodeAuth, "Authentication required")
	}

	// Get all folders for user (collections are called folders in the schema)
	records, err := h.app.FindRecordsByFilter(
		"folders",
		"user_id = {:user_id} && deleted_at = null",
		"-created",
		100,
		0,
		map[string]any{
			"user_id": user.Id,
		},
	)

	if err != nil {
		return h.errorResponse(e, http.StatusInternalServerError, localmodels.ErrCodeInternal, "Failed to fetch folders")
	}

	var collections []localmodels.Collection
	for _, record := range records {
		collection := localmodels.Collection{
			ID:       record.Id,
			UserID:   record.GetString("user_id"),
			Name:     record.GetString("name"),
			ParentID: record.GetString("parent_id"),
			Created:  time.Now(), // Fallback until we fix timestamp access
			Updated:  time.Now(), // Fallback until we fix timestamp access
		}
		collections = append(collections, collection)
	}

	return e.JSON(http.StatusOK, map[string]interface{}{
		"collections": collections,
	})
}