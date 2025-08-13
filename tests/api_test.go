package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"generatio-pb/internal/auth"
	"generatio-pb/internal/crypto"
	"generatio-pb/internal/fal"
	localmodels "generatio-pb/internal/models"

	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testEmail    = "test@test.com"
	testPassword = "testpassword123"
	testFALToken = "test_fal_token_12345"
)

func TestMockFALClient(t *testing.T) {
	mockClient := fal.NewMockClient()

	t.Run("ValidateToken", func(t *testing.T) {
		err := mockClient.ValidateToken(context.Background(), "valid_token")
		assert.NoError(t, err)

		// Test custom validation function
		mockClient.SetValidateTokenFunc(func(ctx context.Context, token string) error {
			if token == "invalid" {
				return &fal.FALError{Code: "invalid_token", Message: "Invalid token"}
			}
			return nil
		})

		err = mockClient.ValidateToken(context.Background(), "invalid")
		assert.Error(t, err)
		
		var falErr *fal.FALError
		assert.ErrorAs(t, err, &falErr)
		assert.Equal(t, "invalid_token", falErr.Code)
	})

	t.Run("GenerateImage", func(t *testing.T) {
		req := fal.GenerationRequest{
			Model:  "flux/schnell",
			Prompt: "test prompt",
			Parameters: map[string]interface{}{
				"image_size": "square_hd",
			},
		}

		result, err := mockClient.GenerateImage(context.Background(), "valid_token", req)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Images)
		assert.Equal(t, 0.003, result.Cost)
		assert.Equal(t, "mock_request_123", result.RequestID)
	})

	t.Run("GetModels", func(t *testing.T) {
		models := mockClient.GetModels()
		assert.Contains(t, models, "flux/schnell")
		assert.Contains(t, models, "hidream/hidream-i1-dev")
	})

	t.Run("SubmitGeneration", func(t *testing.T) {
		req := fal.GenerationRequest{
			Model:  "flux/schnell",
			Prompt: "test prompt",
		}

		response, err := mockClient.SubmitGeneration(context.Background(), "valid_token", req)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, "mock_request_123", response.RequestID)
		assert.Equal(t, fal.StatusQueued, response.Status)
	})

	t.Run("CheckStatus", func(t *testing.T) {
		status, err := mockClient.CheckStatus(context.Background(), "valid_token", "test_request_id")
		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.Equal(t, fal.StatusCompleted, status.Status)
		assert.NotNil(t, status.Result)
	})

	t.Run("PollForCompletion", func(t *testing.T) {
		result, err := mockClient.PollForCompletion(context.Background(), "valid_token", "test_request_id")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Images)
	})

	t.Run("CancelGeneration", func(t *testing.T) {
		err := mockClient.CancelGeneration(context.Background(), "valid_token", "test_request_id")
		assert.NoError(t, err)

		err = mockClient.CancelGeneration(context.Background(), "invalid_token", "test_request_id")
		assert.Error(t, err)
	})
}

func TestAuthAndCrypto(t *testing.T) {
	t.Run("EncryptionService", func(t *testing.T) {
		encService := crypto.NewEncryptionService(1000)
		
		password := "testpassword"
		data := "sensitive_data"
		
		result, err := encService.Encrypt(data, password)
		require.NoError(t, err)
		assert.NotEmpty(t, result.Encrypted)
		assert.NotEmpty(t, result.Salt)
		
		decrypted, err := encService.Decrypt(result.Encrypted, result.Salt, password)
		require.NoError(t, err)
		assert.Equal(t, data, decrypted)
		
		// Test wrong password
		_, err = encService.Decrypt(result.Encrypted, result.Salt, "wrongpassword")
		assert.Error(t, err)
	})

	t.Run("SessionStore", func(t *testing.T) {
		sessionStore := auth.NewSessionStore(time.Hour) // 1 hour
		
		sessionID, err := sessionStore.Create("test_user_123", "decrypted_fal_token")
		require.NoError(t, err)
		assert.NotEmpty(t, sessionID)
		
		retrieved, err := sessionStore.Get(sessionID)
		require.NoError(t, err)
		assert.Equal(t, "test_user_123", retrieved.UserID)
		assert.Equal(t, "decrypted_fal_token", retrieved.FALToken)
		
		// Test invalid session ID
		_, err = sessionStore.Get("invalid_session_id")
		assert.Error(t, err)
		
		// Test delete
		err = sessionStore.Delete(sessionID)
		assert.NoError(t, err)
		
		_, err = sessionStore.Get(sessionID)
		assert.Error(t, err)
	})
}

func TestHandlerUtilities(t *testing.T) {
	// Test with minimal setup using test framework
	app, _ := tests.NewTestApp()
	defer app.ResetBootstrapState()

	encService := crypto.NewEncryptionService(1000)
	sessionStore := auth.NewSessionStore(time.Hour)
	mockClient := fal.NewMockClient()

	// Can't directly test handlers without proper PocketBase setup,
	// but we can test that they can be created
	assert.NotNil(t, encService)
	assert.NotNil(t, sessionStore)
	assert.NotNil(t, mockClient)
}

func TestAPIModels(t *testing.T) {
	t.Run("SetupTokenRequest", func(t *testing.T) {
		req := localmodels.SetupTokenRequest{
			FALToken: "test_token",
			Password: "test_password",
		}
		
		data, err := json.Marshal(req)
		require.NoError(t, err)
		
		var unmarshaled localmodels.SetupTokenRequest
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)
		
		assert.Equal(t, req.FALToken, unmarshaled.FALToken)
		assert.Equal(t, req.Password, unmarshaled.Password)
	})

	t.Run("GenerateImageRequest", func(t *testing.T) {
		req := localmodels.GenerateImageRequest{
			Model:  "flux/schnell",
			Prompt: "A beautiful sunset",
			Parameters: map[string]interface{}{
				"image_size":     "square_hd",
				"guidance_scale": 7.5,
			},
		}
		
		data, err := json.Marshal(req)
		require.NoError(t, err)
		
		var unmarshaled localmodels.GenerateImageRequest
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)
		
		assert.Equal(t, req.Model, unmarshaled.Model)
		assert.Equal(t, req.Prompt, unmarshaled.Prompt)
		assert.Contains(t, unmarshaled.Parameters, "image_size")
	})

	t.Run("APIError", func(t *testing.T) {
		apiErr := localmodels.APIError{
			Code:    localmodels.ErrCodeValidation,
			Message: "Validation failed",
		}
		
		data, err := json.Marshal(apiErr)
		require.NoError(t, err)
		
		var unmarshaled localmodels.APIError
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)
		
		assert.Equal(t, apiErr.Code, unmarshaled.Code)
		assert.Equal(t, apiErr.Message, unmarshaled.Message)
	})
}

func TestEndToEndFlow(t *testing.T) {
	// This tests the core business logic flow without requiring a full PocketBase setup
	
	t.Run("TokenEncryptionFlow", func(t *testing.T) {
		encService := crypto.NewEncryptionService(1000)
		sessionStore := auth.NewSessionStore(time.Hour)
		mockClient := fal.NewMockClient()
		
		// Test token validation
		err := mockClient.ValidateToken(context.Background(), testFALToken)
		assert.NoError(t, err)
		
		// Test encryption
		result, err := encService.Encrypt(testFALToken, testPassword)
		require.NoError(t, err)
		
		// Test combined storage format
		combined := result.Encrypted + "." + result.Salt
		assert.Contains(t, combined, ".")
		
		// Test decryption
		decrypted, err := encService.Decrypt(result.Encrypted, result.Salt, testPassword)
		require.NoError(t, err)
		assert.Equal(t, testFALToken, decrypted)
		
		// Test session creation
		sessionID, err := sessionStore.Create("test_user_123", testFALToken)
		require.NoError(t, err)
		assert.NotEmpty(t, sessionID)
		
		// Test session retrieval
		session, err := sessionStore.Get(sessionID)
		require.NoError(t, err)
		assert.Equal(t, "test_user_123", session.UserID)
		assert.Equal(t, testFALToken, session.FALToken)
		
		// Test image generation flow
		req := fal.GenerationRequest{
			Model:  "flux/schnell",
			Prompt: "A beautiful sunset over mountains",
			Parameters: map[string]interface{}{
				"image_size": "square_hd",
				"num_images": 1,
			},
		}
		
		result2, err := mockClient.GenerateImage(context.Background(), testFALToken, req)
		require.NoError(t, err)
		assert.NotEmpty(t, result2.Images)
		assert.Equal(t, 0.003, result2.Cost)
		
		// Test cleanup
		err = sessionStore.Delete(sessionID)
		assert.NoError(t, err)
	})
}

func TestAPIRequestResponseCycle(t *testing.T) {
	t.Run("HTTPRequestFormatting", func(t *testing.T) {
		// Test request body formatting
		requestBody := localmodels.GenerateImageRequest{
			Model:  "flux/schnell",
			Prompt: "Test prompt",
			Parameters: map[string]interface{}{
				"image_size": "square_hd",
			},
		}
		
		var reqBody bytes.Buffer
		err := json.NewEncoder(&reqBody).Encode(requestBody)
		require.NoError(t, err)
		
		req := httptest.NewRequest("POST", "/api/custom/generate/image", &reqBody)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test_token")
		req.Header.Set("X-Session-ID", "test_session_123")
		
		assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test_token", req.Header.Get("Authorization"))
		assert.Equal(t, "test_session_123", req.Header.Get("X-Session-ID"))
		
		// Test response formatting
		responseData := localmodels.GenerateImageResponse{
			Images: []localmodels.GeneratedImageInfo{
				{
					ID:           "test_image_123",
					URL:          "https://example.com/image.jpg",
					ThumbnailURL: "https://example.com/thumb.jpg",
				},
			},
			Cost:  0.003,
			Model: "flux/schnell",
		}
		
		respBody, err := json.Marshal(responseData)
		require.NoError(t, err)
		assert.Contains(t, string(respBody), "test_image_123")
		assert.Contains(t, string(respBody), "0.003")
		assert.Contains(t, string(respBody), "flux/schnell")
	})
}

func TestServiceIntegration(t *testing.T) {
	t.Run("SessionStoreWithEncryption", func(t *testing.T) {
		encService := crypto.NewEncryptionService(1000)
		sessionStore := auth.NewSessionStore(time.Hour)
		
		// Encrypt a token
		encResult, err := encService.Encrypt(testFALToken, testPassword)
		require.NoError(t, err)
		
		// Verify we can decrypt it
		decrypted, err := encService.Decrypt(encResult.Encrypted, encResult.Salt, testPassword)
		require.NoError(t, err)
		assert.Equal(t, testFALToken, decrypted)
		
		// Create session with decrypted token
		sessionID, err := sessionStore.Create("user123", decrypted)
		require.NoError(t, err)
		
		// Retrieve session and verify
		session, err := sessionStore.Get(sessionID)
		require.NoError(t, err)
		assert.Equal(t, testFALToken, session.FALToken)
		
		// Clean up
		err = sessionStore.Delete(sessionID)
		assert.NoError(t, err)
	})

	t.Run("SessionStoreStats", func(t *testing.T) {
		sessionStore := auth.NewSessionStore(time.Hour)
		
		// Initial stats should be empty
		stats := sessionStore.Stats()
		assert.Equal(t, 0, stats.TotalSessions)
		assert.Equal(t, 0, stats.ActiveSessions)
		
		// Create a session
		sessionID, err := sessionStore.Create("user123", "token123")
		require.NoError(t, err)
		
		// Stats should reflect one active session
		stats = sessionStore.Stats()
		assert.Equal(t, 1, stats.TotalSessions)
		assert.Equal(t, 1, stats.ActiveSessions)
		
		// Delete session
		err = sessionStore.Delete(sessionID)
		assert.NoError(t, err)
		
		// Stats should be empty again
		stats = sessionStore.Stats()
		assert.Equal(t, 0, stats.TotalSessions)
		assert.Equal(t, 0, stats.ActiveSessions)
	})

	t.Run("SessionExpiration", func(t *testing.T) {
		// Create store with very short timeout for testing
		sessionStore := auth.NewSessionStore(1 * time.Millisecond)
		
		sessionID, err := sessionStore.Create("user123", "token123")
		require.NoError(t, err)
		
		// Session should exist initially
		session, err := sessionStore.Get(sessionID)
		require.NoError(t, err)
		assert.False(t, session.IsExpired())
		
		// Wait for expiration
		time.Sleep(10 * time.Millisecond)
		
		// Session should now be expired and get should fail
		_, err = sessionStore.Get(sessionID)
		assert.Error(t, err)
	})
}

func TestCompleteWorkflow(t *testing.T) {
	t.Run("UserRegistrationToImageGeneration", func(t *testing.T) {
		// Setup services
		encService := crypto.NewEncryptionService(1000)
		sessionStore := auth.NewSessionStore(time.Hour)
		mockClient := fal.NewMockClient()
		
		// 1. User sets up FAL token (encrypt and store)
		encResult, err := encService.Encrypt(testFALToken, testPassword)
		require.NoError(t, err)
		
		// This would normally be stored in PocketBase database
		storedToken := encResult.Encrypted + "." + encResult.Salt
		assert.NotEmpty(t, storedToken)
		
		// 2. User creates session (decrypt token and create session)
		parts := []string{encResult.Encrypted, encResult.Salt}
		decryptedToken, err := encService.Decrypt(parts[0], parts[1], testPassword)
		require.NoError(t, err)
		
		sessionID, err := sessionStore.Create("user123", decryptedToken)
		require.NoError(t, err)
		
		// 3. User generates image (using session token)
		session, err := sessionStore.Get(sessionID)
		require.NoError(t, err)
		
		req := fal.GenerationRequest{
			Model:  "flux/schnell",
			Prompt: "A beautiful landscape",
			Parameters: map[string]interface{}{
				"image_size": "landscape_4_3",
				"num_images": 1,
			},
		}
		
		result, err := mockClient.GenerateImage(context.Background(), session.FALToken, req)
		require.NoError(t, err)
		assert.NotEmpty(t, result.Images)
		assert.Equal(t, 0.003, result.Cost)
		
		// 4. Cleanup session
		err = sessionStore.Delete(sessionID)
		assert.NoError(t, err)
		
		// Verify session is gone
		_, err = sessionStore.Get(sessionID)
		assert.Error(t, err)
	})
}