package models

import (
	"time"
)

// User represents the extended user data
type User struct {
	ID           string                 `json:"id"`
	Email        string                 `json:"email"`
	FALToken     string                 `json:"fal_token,omitempty"`     // Encrypted FAL token
	Salt         string                 `json:"salt,omitempty"`          // Salt for encryption
	FinancialData *FinancialData        `json:"financial_data,omitempty"` // Financial tracking
	Created      time.Time              `json:"created"`
	Updated      time.Time              `json:"updated"`
}

// FinancialData tracks user spending and usage
type FinancialData struct {
	TotalSpent   float64 `json:"total_spent"`   // Total amount spent in USD
	TotalImages  int     `json:"total_images"`  // Total images generated
}

// GeneratedImage represents a generated AI image
type GeneratedImage struct {
	ID             string                 `json:"id"`
	UserID         string                 `json:"user_id"`
	Prompt         string                 `json:"prompt"`
	Model          string                 `json:"model"`
	ImageURL       string                 `json:"image_url"`
	ThumbnailURL   string                 `json:"thumbnail_url,omitempty"`
	GenerationCost float64                `json:"generation_cost"`
	GenerationTime float64                `json:"generation_time"` // Time in seconds
	Parameters     map[string]interface{} `json:"parameters"`
	FALRequestID   string                 `json:"fal_request_id"`
	CollectionID   string                 `json:"collection_id,omitempty"`
	Created        time.Time              `json:"created"`
	Updated        time.Time              `json:"updated"`
}

// ModelPreferences represents user preferences for a specific model
type ModelPreferences struct {
	ID          string                 `json:"id"`
	UserID      string                 `json:"user_id"`
	ModelName   string                 `json:"model_name"`
	Preferences map[string]interface{} `json:"preferences"`
	Created     time.Time              `json:"created"`
	Updated     time.Time              `json:"updated"`
}

// Collection represents an image collection/folder
type Collection struct {
	ID       string    `json:"id"`
	UserID   string    `json:"user_id"`
	Name     string    `json:"name"`
	ParentID string    `json:"parent_id,omitempty"` // Optional parent collection
	Created  time.Time `json:"created"`
	Updated  time.Time `json:"updated"`
}

// Session represents an in-memory user session
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	FALToken  string    `json:"-"`        // Never serialize - keep in memory only
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// IsExpired checks if the session has expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// Clear clears sensitive data from the session
func (s *Session) Clear() {
	s.FALToken = ""
}

// API Request/Response Types

// SetupTokenRequest represents the request to setup a FAL token
type SetupTokenRequest struct {
	FALToken string `json:"fal_token" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// VerifyTokenRequest represents the request to verify token accessibility
type VerifyTokenRequest struct {
	Password string `json:"password" validate:"required"`
}

// VerifyTokenResponse represents the response for token verification
type VerifyTokenResponse struct {
	HasToken   bool `json:"has_token"`
	CanDecrypt bool `json:"can_decrypt"`
}

// CreateSessionRequest represents the request to create a session
type CreateSessionRequest struct {
	Password string `json:"password" validate:"required"`
}

// CreateSessionResponse represents the response for session creation
type CreateSessionResponse struct {
	SessionID string    `json:"session_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

// GenerateImageRequest represents the request to generate an image
type GenerateImageRequest struct {
	Model        string                 `json:"model" validate:"required"`
	Prompt       string                 `json:"prompt" validate:"required,max=1000"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
	CollectionID string                 `json:"collection_id,omitempty"`
}

// GenerateImageResponse represents the response for image generation
type GenerateImageResponse struct {
	Images []GeneratedImageInfo `json:"images"`
	Cost   float64              `json:"cost"`
	Model  string               `json:"model"`
}

// GeneratedImageInfo represents basic info about a generated image
type GeneratedImageInfo struct {
	ID           string `json:"id"`
	URL          string `json:"url"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
}

// FinancialStatsResponse represents financial statistics
type FinancialStatsResponse struct {
	TotalSpent      float64 `json:"total_spent"`
	TotalImages     int     `json:"total_images"`
	RecentSpending  float64 `json:"recent_spending"`  // Last 30 days
	AverageCost     float64 `json:"average_cost"`     // Per image
}

// PreferencesResponse represents user preferences for a model
type PreferencesResponse struct {
	ModelName   string                 `json:"model_name"`
	Preferences map[string]interface{} `json:"preferences"`
	HasPreferences bool                `json:"has_preferences"`
}

// SavePreferencesRequest represents the request to save preferences
type SavePreferencesRequest struct {
	ModelName   string                 `json:"model_name" validate:"required"`
	Preferences map[string]interface{} `json:"preferences" validate:"required"`
}

// GetPreferencesRequest represents the request to get preferences
type GetPreferencesRequest struct {
	ModelName string `json:"model_name" validate:"required"`
}

// CreateCollectionRequest represents the request to create a collection
type CreateCollectionRequest struct {
	Name     string `json:"name" validate:"required,max=100"`
	ParentID string `json:"parent_id,omitempty"`
}

// CreateCollectionResponse represents the response for collection creation
type CreateCollectionResponse struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	ParentID string    `json:"parent_id,omitempty"`
	Created  time.Time `json:"created"`
}

// MoveCollectionRequest represents the request to move a collection
type MoveCollectionRequest struct {
	ParentID string `json:"parent_id,omitempty"` // Empty string for root level
}

// AddImagesToCollectionRequest represents the request to add images to a collection
type AddImagesToCollectionRequest struct {
	ImageIDs []string `json:"image_ids" validate:"required,min=1"`
}

// APIError represents a standardized API error response
type APIError struct {
	Code    string      `json:"error"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// Error implements the error interface
func (e *APIError) Error() string {
	return e.Message
}

// Common error codes
const (
	ErrCodeValidation    = "validation_error"
	ErrCodeAuth          = "authentication_error"
	ErrCodeAuthorization = "authorization_error"
	ErrCodeNotFound      = "not_found"
	ErrCodeInternal      = "internal_error"
	ErrCodeExternal      = "external_error"
	ErrCodeRateLimit     = "rate_limit_error"
)