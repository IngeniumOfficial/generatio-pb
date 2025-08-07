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

// Client represents a FAL AI client
type Client struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
}

// NewClient creates a new FAL AI client
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = "https://queue.fal.run/fal-ai"
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

	// Prepare the request
	url := fmt.Sprintf("%s/%s", c.baseURL, req.Model)
	
	// Create request body
	body, err := json.Marshal(map[string]interface{}{
		"prompt": req.Prompt,
		"input":  req.Parameters,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

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
	url := fmt.Sprintf("%s/requests/%s/status", c.baseURL, requestID)

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

// PollForCompletion polls for completion of a generation request
func (c *Client) PollForCompletion(ctx context.Context, token, requestID string) (*GenerationResponse, error) {
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
			status, err := c.CheckStatus(ctx, token, requestID)
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

	// Poll for completion
	result, err := c.PollForCompletion(ctx, token, queueResp.RequestID)
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
	url := fmt.Sprintf("%s/requests/%s/cancel", c.baseURL, requestID)

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, nil)
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
	// Make a simple request to validate the token
	url := fmt.Sprintf("%s/flux/schnell", c.baseURL)
	
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