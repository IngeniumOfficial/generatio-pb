package fal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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
	// New HiDream models already have fal-ai prefix and no subpath
	if fullModelID == "fal-ai/hidream-i1-dev" {
		return "fal-ai/hidream-i1-dev"
	}
	if fullModelID == "fal-ai/hidream-i1-fast" {
		return "fal-ai/hidream-i1-fast"
	}
	
	// Handle already converted FAL model IDs
	if fullModelID == "fal-ai/flux/schnell" {
		return "fal-ai/flux"
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

	// Log incoming parameters for debugging
	fmt.Printf("🔍 FAL REQUEST PARAMETERS:\n")
	fmt.Printf("  Model: %s → %s\n", req.Model, falModelID)
	fmt.Printf("  Prompt: %s\n", req.Prompt)
	fmt.Printf("  Original Parameters: %+v\n", req.Parameters)

	// Get model info to show supported parameters
	if modelInfo, exists := GetModel(req.Model); exists {
		fmt.Printf("  Model Info: %s (%s)\n", modelInfo.DisplayName, modelInfo.Name)
		fmt.Printf("  Supported Parameters:\n")
		for paramName, paramDef := range modelInfo.Parameters {
			fmt.Printf("    - %s (%s): default=%v, required=%v\n", paramName, paramDef.Type, paramDef.Default, paramDef.Required)
		}
	}

	// Create request body - FAL expects different structure
	requestBody := map[string]interface{}{
		"prompt": req.Prompt,
	}

	// Add parameters directly to the request body (not under "input")
	// Only include parameters that are supported by the model
	if req.Parameters != nil {
		fmt.Printf("  Processing Parameters:\n")
		if modelInfo, exists := GetModel(req.Model); exists {
			for key, value := range req.Parameters {
				if paramDef, supported := modelInfo.Parameters[key]; supported {
					fmt.Printf("    ✅ %s = %v (supported: %s)\n", key, value, paramDef.Type)
					requestBody[key] = value
				} else {
					fmt.Printf("    ❌ %s = %v (FILTERED OUT - not supported by %s)\n", key, value, req.Model)
				}
			}
		} else {
			// If model info not found, include all parameters (fallback)
			fmt.Printf("  ⚠️  Model info not found, including all parameters\n")
			for key, value := range req.Parameters {
				requestBody[key] = value
			}
		}
	} else {
		fmt.Printf("  No parameters provided\n")
	}

	fmt.Printf("  Final Request Body: %+v\n", requestBody)
	
	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Log essential request info for debugging
	fmt.Printf("FAL API Request: %s %s (model: %s)\n", "POST", url, req.Model)
	fmt.Printf("📤 JSON PAYLOAD TO FAL: %s\n", string(body))

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

	// Log response status for actual errors (not 202 Accepted)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		fmt.Printf("FAL API Error: %d %s - %s\n", resp.StatusCode, resp.Status, string(respBody))
	}

	// Handle error responses - accept both 200 OK and 202 Accepted
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
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

	// Log response status for actual errors (not 202 Accepted)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		fmt.Printf("FAL Status Check Error: %d %s - %s\n", resp.StatusCode, resp.Status, string(respBody))
	}

	// Handle error responses - accept both 200 OK and 202 Accepted
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
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

	// Debug: Log the parsed response to understand the structure
	fmt.Printf("FAL Status Response Debug (Legacy):\n")
	fmt.Printf("  Status: %s\n", statusResp.Status)
	fmt.Printf("  RequestID: %s\n", statusResp.RequestID)
	fmt.Printf("  Result is nil: %t\n", statusResp.Result == nil)
	if statusResp.Result != nil {
		fmt.Printf("  Result.Status: %s\n", statusResp.Result.Status)
		fmt.Printf("  Result.Images count: %d\n", len(statusResp.Result.Images))
	}
	fmt.Printf("  Raw response: %s\n", string(respBody))

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

	// Log response status for actual errors (not 202 Accepted)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		fmt.Printf("FAL Status Check Error: %d %s - %s\n", resp.StatusCode, resp.Status, string(respBody))
	}

	// Handle error responses - accept both 200 OK and 202 Accepted
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
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

	// Debug: Log the parsed response to understand the structure
	fmt.Printf("FAL Status Response Debug:\n")
	fmt.Printf("  Status: %s\n", statusResp.Status)
	fmt.Printf("  RequestID: %s\n", statusResp.RequestID)
	fmt.Printf("  Result is nil: %t\n", statusResp.Result == nil)
	if statusResp.Result != nil {
		fmt.Printf("  Result.Status: %s\n", statusResp.Result.Status)
		fmt.Printf("  Result.Images count: %d\n", len(statusResp.Result.Images))
		fmt.Printf("  Result.RequestID: %s\n", statusResp.Result.RequestID)
		fmt.Printf("  ⚠️  WARNING: Result is not nil when status is '%s'!\n", statusResp.Status)
	} else {
		fmt.Printf("  ✅ Result is correctly nil for status '%s'\n", statusResp.Status)
	}
	fmt.Printf("  Raw response: %s\n", string(respBody))

	return &statusResp, nil
}

// GetResult retrieves the result of a completed generation request
func (c *Client) GetResult(ctx context.Context, token, modelID, requestID string) (*GenerationResponse, error) {
	// First convert to FAL format, then get base model ID for result retrieval
	falModelID := convertToFALModelID(modelID)
	baseModelID := getBaseModelID(falModelID)
	
	// FAL API result endpoint format (without /status)
	url := fmt.Sprintf("%s/%s/requests/%s", c.baseURL, baseModelID, requestID)

	// Log result retrieval request
	fmt.Printf("FAL Get Result: %s (model: %s → %s, request: %s)\n", url, modelID, baseModelID, requestID)

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

	// Log response status for actual errors (not 202 Accepted)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		fmt.Printf("FAL Get Result Error: %d %s - %s\n", resp.StatusCode, resp.Status, string(respBody))
	}

	// Handle error responses - accept both 200 OK and 202 Accepted
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		var falErr FALError
		if err := json.Unmarshal(respBody, &falErr); err != nil {
			return nil, &FALError{
				Code:    "http_error",
				Message: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody)),
			}
		}
		return nil, &falErr
	}

	// Parse response directly as GenerationResponse
	var result GenerationResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse result response: %w", err)
	}

	// Debug: Log the parsed result
	fmt.Printf("FAL Result Response Debug:\n")
	fmt.Printf("  RequestID: %s\n", result.RequestID)
	fmt.Printf("  Status: %s\n", result.Status)
	fmt.Printf("  Images count: %d\n", len(result.Images))
	fmt.Printf("  Raw response: %s\n", string(respBody))

	return &result, nil
}

// PollForCompletion polls for completion of a generation request (legacy interface method)
func (c *Client) PollForCompletion(ctx context.Context, token, requestID string) (*GenerationResponse, error) {
	// Use default model ID for backward compatibility - use ORIGINAL model ID, not converted
	return c.PollForCompletionWithModel(ctx, token, "flux/schnell", requestID)
}

// PollForCompletionWithModel polls for completion of a generation request with model ID
func (c *Client) PollForCompletionWithModel(ctx context.Context, token, modelID, requestID string) (*GenerationResponse, error) {
	fmt.Printf("⏰ POLLING START: Model=%s, RequestID=%s, Timeout=%v\n", modelID, requestID, c.timeout)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	startTime := time.Now()
	ticker := time.NewTicker(2 * time.Second) // Poll every 2 seconds
	defer ticker.Stop()

	pollCount := 0
	for {
		select {
		case <-ctx.Done():
			duration := time.Since(startTime)
			fmt.Printf("⏰ POLLING TIMEOUT: After %v (limit: %v)\n", duration, c.timeout)
			return nil, &FALError{
				Code:    "timeout",
				Message: "generation request timed out",
			}
		case <-ticker.C:
			pollCount++
			fmt.Printf("🔄 POLLING #%d: Checking status for request %s (model: %s)\n", pollCount, requestID, modelID)

			status, err := c.CheckStatusWithModel(ctx, token, modelID, requestID)
			if err != nil {
				fmt.Printf("❌ POLLING ERROR: Failed to check status: %v\n", err)
				return nil, err
			}

			// Normalize status to lowercase for comparison
			normalizedStatus := strings.ToLower(status.Status)
			fmt.Printf("📊 POLLING STATUS: Raw='%s', Normalized='%s'\n", status.Status, normalizedStatus)
			fmt.Printf("🔍 POLLING DEBUG: status.Status type: %T, value: %q\n", status.Status, status.Status)
			fmt.Printf("🔍 POLLING DEBUG: normalizedStatus type: %T, value: %q\n", normalizedStatus, normalizedStatus)

			switch normalizedStatus {
			case StatusCompleted:
				fmt.Printf("✅ POLLING COMPLETE: Status is '%s', fetching result...\n", status.Status)
				// When status is completed, fetch the actual result from the result endpoint
				result, err := c.GetResult(ctx, token, modelID, requestID)
				if err != nil {
					fmt.Printf("❌ POLLING RESULT ERROR: Failed to get result: %v\n", err)
					return nil, fmt.Errorf("failed to get completed result: %w", err)
				}
				fmt.Printf("✅ POLLING SUCCESS: Result retrieved with %d images after %d polls\n", len(result.Images), pollCount)
				return result, nil
			case StatusFailed:
				fmt.Printf("❌ POLLING FAILED: Status is '%s' after %d polls\n", status.Status, pollCount)
				if status.Error != nil {
					return nil, status.Error
				}
				return nil, &FALError{
					Code:    "generation_failed",
					Message: "generation failed with unknown error",
				}
			case StatusCancelled:
				fmt.Printf("❌ POLLING CANCELLED: Status is '%s' after %d polls\n", status.Status, pollCount)
				return nil, &FALError{
					Code:    "generation_cancelled",
					Message: "generation was cancelled",
				}
			case StatusQueued, StatusProcessing:
				fmt.Printf("⏳ POLLING CONTINUE #%d: Status is '%s', continuing to poll...\n", pollCount, status.Status)
				// Continue polling
				continue
			case "in_progress":
				fmt.Printf("⏳ POLLING CONTINUE #%d: Status is '%s' (HiDream format), continuing to poll...\n", pollCount, status.Status)
				// Continue polling - HiDream models use "IN_PROGRESS" instead of "processing"
				continue
			default:
				fmt.Printf("❓ POLLING UNKNOWN #%d: Unknown status '%s', expected: %s, %s, %s, %s, %s, %s\n",
					pollCount, status.Status, StatusQueued, StatusProcessing, StatusCompleted, StatusFailed, StatusCancelled, "in_progress")
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

	// Handle error responses - accept both 200 OK and 202 Accepted
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
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

	// Check response - accept both 200 OK and 202 Accepted as valid
	if resp.StatusCode == http.StatusUnauthorized {
		return &FALError{
			Code:    "invalid_token",
			Message: "invalid or expired FAL AI token",
		}
	}

	// Any other response (including success and 202 Accepted) means the token is valid
	return nil
}

// GetModels returns information about all supported models
func (c *Client) GetModels() map[string]ModelInfo {
	return GetAllModels()
}