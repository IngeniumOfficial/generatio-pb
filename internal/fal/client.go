package fal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// convertToFALModelID converts our internal model ID to FAL API format
func convertToFALModelID(modelID string) string {
	// If the model ID already has the fal-ai prefix, return as-is
	if len(modelID) >= 7 && modelID[:7] == "fal-ai/" {
		return modelID
	}
	
	// Add the fal-ai prefix for FAL API endpoints
	return "fal-ai/" + modelID
}

// getBaseModelID extracts the base model ID for status/result operations
// For models with subpaths like "fal-ai/flux/schnell", returns "fal-ai/flux"
// For models without subpaths, returns the full model ID
func getBaseModelID(fullModelID string) string {
	// Handle our internal model names first
	if fullModelID == "flux/schnell" {
		return "fal-ai/flux"
	}
	if fullModelID == "hidream/hidream-i1-dev" || fullModelID == "hidream/hidream-i1-fast" {
		return "fal-ai/hidream"
	}
	
	// Handle already converted FAL model IDs
	if fullModelID == "fal-ai/flux/schnell" {
		return "fal-ai/flux"
	}
	if fullModelID == "fal-ai/hidream/hidream-i1-dev" || fullModelID == "fal-ai/hidream/hidream-i1-fast" {
		return "fal-ai/hidream"
	}
	
	// For other models, return as-is (no subpath)
	return fullModelID
}

// Client represents a FAL AI client
type Client struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
}

// NewClient creates a new FAL AI client
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		// Official FAL AI queue endpoint
		baseURL = "https://queue.fal.run"
	}

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		timeout: 5 * time.Minute, // Default timeout for generation
	}
}

// SetTimeout sets the timeout for generation requests
func (c *Client) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
}

// SubmitGeneration submits a generation request to the FAL AI queue
func (c *Client) SubmitGeneration(ctx context.Context, token string, req GenerationRequest) (*QueueResponse, error) {
	// Validate the model
	model, exists := GetModel(req.Model)
	if !exists {
		return nil, &FALError{
			Code:    "invalid_model",
			Message: "unsupported model: " + req.Model,
		}
	}

	// Validate parameters
	if err := model.ValidateParameters(req.Parameters); err != nil {
		return nil, err
	}

	// Prepare the request - updated URL structure for FAL API
	falModelID := convertToFALModelID(req.Model)
	url := fmt.Sprintf("%s/%s", c.baseURL, falModelID)
	
	// Create request body - FAL expects different structure
	requestBody := map[string]interface{}{
		"prompt": req.Prompt,
	}
	
	// Add parameters directly to the request body (not under "input")
	if req.Parameters != nil {
		for key, value := range req.Parameters {
			requestBody[key] = value
		}
	}
	
	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Log essential request info for debugging
	fmt.Printf("FAL API Request: %s %s (model: %s)\n", "POST", url, req.Model)

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Key "+token)

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Log response status
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("FAL API Error: %d %s - %s\n", resp.StatusCode, resp.Status, string(respBody))
	}

	// Handle error responses
	if resp.StatusCode != http.StatusOK {
		var falErr FALError
		if err := json.Unmarshal(respBody, &falErr); err != nil {
			return nil, &FALError{
				Code:    "http_error",
				Message: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody)),
			}
		}
		return nil, &falErr
	}

	// Parse response
	var queueResp QueueResponse
	if err := json.Unmarshal(respBody, &queueResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &queueResp, nil
}

// CheckStatus checks the status of a generation request
func (c *Client) CheckStatus(ctx context.Context, token, requestID string) (*StatusResponse, error) {
	// Extract model ID from request ID context - we need to pass it properly
	// For now, we'll need to store the model ID with the request
	// This is a design issue - we need the model ID for status checks
	
	// TEMPORARY: We'll try to find the model ID from common models
	// This should be fixed by storing model ID with the request
	modelID := "flux/schnell" // Default for now - use ORIGINAL model ID
	falModelID := convertToFALModelID(modelID)
	baseModelID := getBaseModelID(falModelID)
	
	// Official FAL queue status endpoint format
	url := fmt.Sprintf("%s/%s/requests/%s/status", c.baseURL, baseModelID, requestID)

	// Log status check request
	fmt.Printf("FAL Status Check: %s (model: %s, request: %s)\n", url, modelID, requestID)

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Authorization", "Key "+token)

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Log response status for errors
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("FAL Status Check Error: %d %s - %s\n", resp.StatusCode, resp.Status, string(respBody))
	}

	// Handle error responses
	if resp.StatusCode != http.StatusOK {
		var falErr FALError
		if err := json.Unmarshal(respBody, &falErr); err != nil {
			return nil, &FALError{
				Code:    "http_error",
				Message: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody)),
			}
		}
		return nil, &falErr
	}

	// Parse response
	var statusResp StatusResponse
	if err := json.Unmarshal(respBody, &statusResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &statusResp, nil
}

// CheckStatusWithModel checks the status of a generation request with model ID
func (c *Client) CheckStatusWithModel(ctx context.Context, token, modelID, requestID string) (*StatusResponse, error) {
	// First convert to FAL format, then get base model ID for status checks
	falModelID := convertToFALModelID(modelID)
	baseModelID := getBaseModelID(falModelID)
	
	// Official FAL queue status endpoint format
	url := fmt.Sprintf("%s/%s/requests/%s/status", c.baseURL, baseModelID, requestID)

	// Log status check request with model
	fmt.Printf("FAL Status Check: %s (model: %s → %s, request: %s)\n", url, modelID, baseModelID, requestID)

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Authorization", "Key "+token)

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		fmt.Printf("❌ FAL Status Check Request failed: %v\n", err)
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Log response status for errors
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("FAL Status Check Error: %d %s - %s\n", resp.StatusCode, resp.Status, string(respBody))
	}

	// Handle error responses
	if resp.StatusCode != http.StatusOK {
		var falErr FALError
		if err := json.Unmarshal(respBody, &falErr); err != nil {
			return nil, &FALError{
				Code:    "http_error",
				Message: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody)),
			}
		}
		return nil, &falErr
	}

	// Parse response
	var statusResp StatusResponse
	if err := json.Unmarshal(respBody, &statusResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &statusResp, nil
}

// PollForCompletion polls for completion of a generation request (legacy interface method)
func (c *Client) PollForCompletion(ctx context.Context, token, requestID string) (*GenerationResponse, error) {
	// Use default model ID for backward compatibility - use ORIGINAL model ID, not converted
	return c.PollForCompletionWithModel(ctx, token, "flux/schnell", requestID)
}

// PollForCompletionWithModel polls for completion of a generation request with model ID
func (c *Client) PollForCompletionWithModel(ctx context.Context, token, modelID, requestID string) (*GenerationResponse, error) {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second) // Poll every 2 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, &FALError{
				Code:    "timeout",
				Message: "generation request timed out",
			}
		case <-ticker.C:
			status, err := c.CheckStatusWithModel(ctx, token, modelID, requestID)
			if err != nil {
				return nil, err
			}

			switch status.Status {
			case StatusCompleted:
				if status.Result == nil {
					return nil, &FALError{
						Code:    "missing_result",
						Message: "generation completed but no result provided",
					}
				}
				return status.Result, nil
			case StatusFailed:
				if status.Error != nil {
					return nil, status.Error
				}
				return nil, &FALError{
					Code:    "generation_failed",
					Message: "generation failed with unknown error",
				}
			case StatusCancelled:
				return nil, &FALError{
					Code:    "generation_cancelled",
					Message: "generation was cancelled",
				}
			case StatusQueued, StatusProcessing:
				// Continue polling
				continue
			default:
				return nil, &FALError{
					Code:    "unknown_status",
					Message: "unknown generation status: " + status.Status,
				}
			}
		}
	}
}

// GenerateImage generates an image using the FAL AI service
func (c *Client) GenerateImage(ctx context.Context, token string, req GenerationRequest) (*GenerationResponse, error) {
	// Submit the generation request
	queueResp, err := c.SubmitGeneration(ctx, token, req)
	if err != nil {
		return nil, err
	}

	// Poll for completion - pass the original model ID, let CheckStatusWithModel handle conversion
	result, err := c.PollForCompletionWithModel(ctx, token, req.Model, queueResp.RequestID)
	if err != nil {
		return nil, err
	}

	// Calculate cost based on model and number of images
	model, _ := GetModel(req.Model)
	numImages := 1
	if req.Parameters != nil {
		if num, ok := req.Parameters["num_images"]; ok {
			if numInt, ok := num.(int); ok {
				numImages = numInt
			} else if numFloat, ok := num.(float64); ok {
				numImages = int(numFloat)
			}
		}
	}
	
	result.Cost = model.CostPerImage * float64(numImages)
	result.RequestID = queueResp.RequestID

	return result, nil
}

// CancelGeneration cancels a generation request
func (c *Client) CancelGeneration(ctx context.Context, token, requestID string) error {
	// Extract model ID (same issue as status check)
	modelID := "flux/schnell" // Default for now - use ORIGINAL model ID
	falModelID := convertToFALModelID(modelID)
	baseModelID := getBaseModelID(falModelID)
	
	// Official FAL queue cancel endpoint with correct method (PUT)
	url := fmt.Sprintf("%s/%s/requests/%s/cancel", c.baseURL, baseModelID, requestID)

	// Create HTTP request with PUT method (not POST)
	httpReq, err := http.NewRequestWithContext(ctx, "PUT", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Authorization", "Key "+token)

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Handle error responses
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		var falErr FALError
		if err := json.Unmarshal(respBody, &falErr); err != nil {
			return &FALError{
				Code:    "http_error",
				Message: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody)),
			}
		}
		return &falErr
	}

	return nil
}

// ValidateToken validates a FAL AI token by making a test request
func (c *Client) ValidateToken(ctx context.Context, token string) error {
	// Make a simple request to validate the token using correct endpoint
	// FIX: Use proper model ID conversion instead of hardcoded path
	testModelID := "flux/schnell"
	falModelID := convertToFALModelID(testModelID)
	url := fmt.Sprintf("%s/%s", c.baseURL, falModelID)
	
	// Log token validation request
	fmt.Printf("FAL Token Validation: %s\n", url)
	
	testReq := map[string]interface{}{
		"prompt": "test",
		"input": map[string]interface{}{
			"num_images":  1,
			"image_size": "square",
		},
	}

	body, err := json.Marshal(testReq)
	if err != nil {
		return fmt.Errorf("failed to marshal test request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Key "+token)

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode == http.StatusUnauthorized {
		return &FALError{
			Code:    "invalid_token",
			Message: "invalid or expired FAL AI token",
		}
	}

	// Any other response (including success) means the token is valid
	return nil
}

// GetModels returns information about all supported models
func (c *Client) GetModels() map[string]ModelInfo {
	return GetAllModels()
}