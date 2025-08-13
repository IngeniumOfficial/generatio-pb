package fal

import (
	"context"
	"time"
)

// MockClient implements the FAL client interface for testing
type MockClient struct {
	validateTokenFunc    func(ctx context.Context, token string) error
	generateImageFunc    func(ctx context.Context, token string, req GenerationRequest) (*GenerationResponse, error)
	getModelsFunc        func() map[string]ModelInfo
	submitGenerationFunc func(ctx context.Context, token string, req GenerationRequest) (*QueueResponse, error)
	checkStatusFunc      func(ctx context.Context, token, requestID string) (*StatusResponse, error)
	pollForCompletionFunc func(ctx context.Context, token, requestID string) (*GenerationResponse, error)
}

// NewMockClient creates a new mock FAL client
func NewMockClient() *MockClient {
	return &MockClient{
		validateTokenFunc: func(ctx context.Context, token string) error {
			if token == "invalid_token" {
				return &FALError{Code: "invalid_token", Message: "Invalid token"}
			}
			return nil
		},
		generateImageFunc: func(ctx context.Context, token string, req GenerationRequest) (*GenerationResponse, error) {
			if token == "invalid_token" {
				return nil, &FALError{Code: "invalid_token", Message: "Invalid token"}
			}
			
			// Return mock successful response
			return &GenerationResponse{
				RequestID: "mock_request_123",
				Status:    StatusCompleted,
				Images: []struct {
					URL         string `json:"url"`
					ThumbnailURL string `json:"thumbnail_url,omitempty"`
					Width       int    `json:"width,omitempty"`
					Height      int    `json:"height,omitempty"`
				}{
					{
						URL:          "https://mock-image-url.com/image.jpg",
						ThumbnailURL: "https://mock-image-url.com/thumb.jpg",
						Width:        1024,
						Height:       1024,
					},
				},
				Cost: 0.003,
			}, nil
		},
		getModelsFunc: func() map[string]ModelInfo {
			return GetAllModels() // Use real model definitions
		},
		submitGenerationFunc: func(ctx context.Context, token string, req GenerationRequest) (*QueueResponse, error) {
			if token == "invalid_token" {
				return nil, &FALError{Code: "invalid_token", Message: "Invalid token"}
			}
			return &QueueResponse{
				RequestID: "mock_request_123",
				Status:    StatusQueued,
			}, nil
		},
		checkStatusFunc: func(ctx context.Context, token, requestID string) (*StatusResponse, error) {
			if token == "invalid_token" {
				return nil, &FALError{Code: "invalid_token", Message: "Invalid token"}
			}
			return &StatusResponse{
				RequestID: requestID,
				Status:    StatusCompleted,
				Result: &GenerationResponse{
					RequestID: requestID,
					Status:    StatusCompleted,
					Images: []struct {
						URL         string `json:"url"`
						ThumbnailURL string `json:"thumbnail_url,omitempty"`
						Width       int    `json:"width,omitempty"`
						Height      int    `json:"height,omitempty"`
					}{
						{
							URL:          "https://mock-image-url.com/image.jpg",
							ThumbnailURL: "https://mock-image-url.com/thumb.jpg",
							Width:        1024,
							Height:       1024,
						},
					},
					Cost: 0.003,
				},
			}, nil
		},
		pollForCompletionFunc: func(ctx context.Context, token, requestID string) (*GenerationResponse, error) {
			if token == "invalid_token" {
				return nil, &FALError{Code: "invalid_token", Message: "Invalid token"}
			}
			return &GenerationResponse{
				RequestID: requestID,
				Status:    StatusCompleted,
				Images: []struct {
					URL         string `json:"url"`
					ThumbnailURL string `json:"thumbnail_url,omitempty"`
					Width       int    `json:"width,omitempty"`
					Height      int    `json:"height,omitempty"`
				}{
					{
						URL:          "https://mock-image-url.com/image.jpg",
						ThumbnailURL: "https://mock-image-url.com/thumb.jpg",
						Width:        1024,
						Height:       1024,
					},
				},
				Cost: 0.003,
			}, nil
		},
	}
}

// SetTimeout sets the timeout for generation requests (mock implementation)
func (c *MockClient) SetTimeout(timeout time.Duration) {
	// Mock implementation - no-op
}

// ValidateToken validates a FAL AI token (mock implementation)
func (c *MockClient) ValidateToken(ctx context.Context, token string) error {
	return c.validateTokenFunc(ctx, token)
}

// GenerateImage generates an image using the FAL AI service (mock implementation)
func (c *MockClient) GenerateImage(ctx context.Context, token string, req GenerationRequest) (*GenerationResponse, error) {
	return c.generateImageFunc(ctx, token, req)
}

// GetModels returns information about all supported models (mock implementation)
func (c *MockClient) GetModels() map[string]ModelInfo {
	return c.getModelsFunc()
}

// SubmitGeneration submits a generation request to the FAL AI queue (mock implementation)
func (c *MockClient) SubmitGeneration(ctx context.Context, token string, req GenerationRequest) (*QueueResponse, error) {
	return c.submitGenerationFunc(ctx, token, req)
}

// CheckStatus checks the status of a generation request (mock implementation)
func (c *MockClient) CheckStatus(ctx context.Context, token, requestID string) (*StatusResponse, error) {
	return c.checkStatusFunc(ctx, token, requestID)
}

// PollForCompletion polls for completion of a generation request (mock implementation)
func (c *MockClient) PollForCompletion(ctx context.Context, token, requestID string) (*GenerationResponse, error) {
	return c.pollForCompletionFunc(ctx, token, requestID)
}

// CancelGeneration cancels a generation request (mock implementation)
func (c *MockClient) CancelGeneration(ctx context.Context, token, requestID string) error {
	if token == "invalid_token" {
		return &FALError{Code: "invalid_token", Message: "Invalid token"}
	}
	return nil // Success
}

// Mock configuration methods

// SetValidateTokenFunc sets a custom validate token function for testing
func (c *MockClient) SetValidateTokenFunc(fn func(ctx context.Context, token string) error) {
	c.validateTokenFunc = fn
}

// SetGenerateImageFunc sets a custom generate image function for testing
func (c *MockClient) SetGenerateImageFunc(fn func(ctx context.Context, token string, req GenerationRequest) (*GenerationResponse, error)) {
	c.generateImageFunc = fn
}

// SetGetModelsFunc sets a custom get models function for testing
func (c *MockClient) SetGetModelsFunc(fn func() map[string]ModelInfo) {
	c.getModelsFunc = fn
}