package fal

import (
	"context"
	"time"
)

// FALClient defines the interface for FAL AI operations
type FALClient interface {
	SetTimeout(timeout time.Duration)
	ValidateToken(ctx context.Context, token string) error
	GenerateImage(ctx context.Context, token string, req GenerationRequest) (*GenerationResponse, error)
	GetModels() map[string]ModelInfo
	SubmitGeneration(ctx context.Context, token string, req GenerationRequest) (*QueueResponse, error)
	CheckStatus(ctx context.Context, token, requestID string) (*StatusResponse, error)
	PollForCompletion(ctx context.Context, token, requestID string) (*GenerationResponse, error)
	CancelGeneration(ctx context.Context, token, requestID string) error
}

// Ensure both implementations satisfy the interface
var _ FALClient = (*Client)(nil)
var _ FALClient = (*MockClient)(nil)