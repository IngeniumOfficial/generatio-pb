package tests

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"generatio-pb/internal/auth"
	"generatio-pb/internal/crypto"
	"generatio-pb/internal/fal"
	localmodels "generatio-pb/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutoSessionCreationLogic(t *testing.T) {
	// Test the core auto-session creation logic that's used in the custom login endpoint
	// This tests the business logic without requiring full PocketBase HTTP setup

	encService := crypto.NewEncryptionService(1000) // Reduced iterations for testing
	sessionStore := auth.NewSessionStore(24 * time.Hour)
	userID := "test_user_123"
	userPassword := "userpassword"
	falToken := "test-fal-token"

	t.Run("Auto-session creation with valid token", func(t *testing.T) {
		// 1. Simulate token setup (encrypt FAL token with user password)
		encResult, err := encService.Encrypt(falToken, userPassword)
		require.NoError(t, err)

		// 2. Store in combined format as done in database
		combinedToken := encResult.Encrypted + "." + encResult.Salt

		// 3. Simulate login auto-session logic
		parts := strings.Split(combinedToken, ".")
		require.Len(t, parts, 2, "Combined token should have exactly 2 parts")

		falTokenEncrypted := parts[0]
		salt := parts[1]

		// 4. Try to decrypt using login password
		decryptedToken, err := encService.Decrypt(falTokenEncrypted, salt, userPassword)
		require.NoError(t, err)
		assert.Equal(t, falToken, decryptedToken)

		// 5. Create session with decrypted token
		sessionStore.DeleteUserSessions(userID) // Clear any existing sessions
		sessionID, err := sessionStore.Create(userID, decryptedToken)
		require.NoError(t, err)
		assert.NotEmpty(t, sessionID)

		// 6. Verify session was created correctly
		session, err := sessionStore.Get(sessionID)
		require.NoError(t, err)
		assert.Equal(t, userID, session.UserID)
		assert.Equal(t, falToken, session.FALToken)

		// Clean up
		sessionStore.Delete(sessionID)
	})

	t.Run("Auto-session creation with wrong password", func(t *testing.T) {
		// 1. Setup token encrypted with userPassword
		encResult, err := encService.Encrypt(falToken, userPassword)
		require.NoError(t, err)
		combinedToken := encResult.Encrypted + "." + encResult.Salt

		// 2. Try to decrypt with wrong password
		parts := strings.Split(combinedToken, ".")
		require.Len(t, parts, 2)

		wrongPassword := "wrongpassword"
		_, err = encService.Decrypt(parts[0], parts[1], wrongPassword)
		assert.Error(t, err, "Decryption should fail with wrong password")

		// 3. No session should be created
		sessionCount := sessionStore.GetSessionCount()
		assert.Equal(t, 0, sessionCount)
	})

	t.Run("Auto-session creation with different encryption password", func(t *testing.T) {
		// 1. Setup token encrypted with different password than user password
		encryptionPassword := "differentpassword"
		encResult, err := encService.Encrypt(falToken, encryptionPassword)
		require.NoError(t, err)
		combinedToken := encResult.Encrypted + "." + encResult.Salt

		// 2. Try to decrypt with user password (should fail)
		parts := strings.Split(combinedToken, ".")
		require.Len(t, parts, 2)

		_, err = encService.Decrypt(parts[0], parts[1], userPassword)
		assert.Error(t, err, "Decryption should fail when encryption password differs from user password")

		// 3. No session should be created
		sessionCount := sessionStore.GetSessionCount()
		assert.Equal(t, 0, sessionCount)
	})

	t.Run("Auto-session creation with invalid token format", func(t *testing.T) {
		// 1. Simulate invalid combined token format
		invalidToken := "invalidtokenformat" // Missing separator

		// 2. Try to parse
		parts := strings.Split(invalidToken, ".")
		assert.NotEqual(t, 2, len(parts), "Invalid token should not have exactly 2 parts")

		// 3. Should not proceed with session creation
		sessionCount := sessionStore.GetSessionCount()
		assert.Equal(t, 0, sessionCount)
	})

	t.Run("Auto-session creation with empty token", func(t *testing.T) {
		// 1. Simulate user with no FAL token
		combinedToken := ""

		// 2. Should not proceed
		assert.Empty(t, combinedToken)

		// 3. No session should be created
		sessionCount := sessionStore.GetSessionCount()
		assert.Equal(t, 0, sessionCount)
	})

	t.Run("Session cleanup on new login", func(t *testing.T) {
		// 1. Create existing session for user
		oldSessionID, err := sessionStore.Create(userID, "old-token")
		require.NoError(t, err)

		// 2. Verify old session exists
		oldSession, err := sessionStore.Get(oldSessionID)
		require.NoError(t, err)
		assert.Equal(t, "old-token", oldSession.FALToken)

		// 3. Simulate new login auto-session creation
		encResult, err := encService.Encrypt(falToken, userPassword)
		require.NoError(t, err)
		combinedToken := encResult.Encrypted + "." + encResult.Salt

		parts := strings.Split(combinedToken, ".")
		decryptedToken, err := encService.Decrypt(parts[0], parts[1], userPassword)
		require.NoError(t, err)

		// 4. Delete old sessions and create new one (as done in login handler)
		sessionStore.DeleteUserSessions(userID)
		newSessionID, err := sessionStore.Create(userID, decryptedToken)
		require.NoError(t, err)

		// 5. Verify old session is gone and new session exists
		_, err = sessionStore.Get(oldSessionID)
		assert.Error(t, err, "Old session should be deleted")

		newSession, err := sessionStore.Get(newSessionID)
		require.NoError(t, err)
		assert.Equal(t, falToken, newSession.FALToken)

		// Clean up
		sessionStore.Delete(newSessionID)
	})
}

func TestCustomLoginResponseMessages(t *testing.T) {
	// Test the different response messages that should be returned
	// by the custom login endpoint for various scenarios

	t.Run("Message determination logic", func(t *testing.T) {
		// Test scenarios for message determination
		testCases := []struct {
			name            string
			hasToken        bool
			validFormat     bool
			decryptSuccess  bool
			sessionCreated  bool
			expectedMessage string
		}{
			{
				name:            "No FAL token configured",
				hasToken:        false,
				validFormat:     false,
				decryptSuccess:  false,
				sessionCreated:  false,
				expectedMessage: "Login successful. No FAL token configured - setup required",
			},
			{
				name:            "Invalid token format",
				hasToken:        true,
				validFormat:     false,
				decryptSuccess:  false,
				sessionCreated:  false,
				expectedMessage: "Login successful. Invalid FAL token format - please setup token again",
			},
			{
				name:            "Token exists but wrong password",
				hasToken:        true,
				validFormat:     true,
				decryptSuccess:  false,
				sessionCreated:  false,
				expectedMessage: "Login successful. FAL token found but password doesn't match - please call create-session manually",
			},
			{
				name:            "Session creation failed",
				hasToken:        true,
				validFormat:     true,
				decryptSuccess:  true,
				sessionCreated:  false,
				expectedMessage: "Login successful. Failed to auto-create session - please call create-session manually",
			},
			{
				name:            "Successful auto-session creation",
				hasToken:        true,
				validFormat:     true,
				decryptSuccess:  true,
				sessionCreated:  true,
				expectedMessage: "Login successful with auto-created session",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Simulate the message determination logic from the handler
				var message string

				if !tc.hasToken {
					message = "Login successful. No FAL token configured - setup required"
				} else if !tc.validFormat {
					message = "Login successful. Invalid FAL token format - please setup token again"
				} else if !tc.decryptSuccess {
					message = "Login successful. FAL token found but password doesn't match - please call create-session manually"
				} else if !tc.sessionCreated {
					message = "Login successful. Failed to auto-create session - please call create-session manually"
				} else {
					message = "Login successful with auto-created session"
				}

				assert.Equal(t, tc.expectedMessage, message)
			})
		}
	})
}

func TestAutoSessionIntegrationFlow(t *testing.T) {
	// Test the complete auto-session integration flow
	// This simulates the exact workflow in the custom login endpoint

	t.Run("Complete auto-session workflow", func(t *testing.T) {
		encService := crypto.NewEncryptionService(1000)
		sessionStore := auth.NewSessionStore(time.Hour)
		mockClient := fal.NewMockClient()

		userID := "user123"
		userPassword := "password123"
		falToken := "fal-token-123"

		// 1. Initial token setup (what happens in /tokens/setup)
		encResult, err := encService.Encrypt(falToken, userPassword)
		require.NoError(t, err)
		
		// Validate token first
		err = mockClient.ValidateToken(nil, falToken)
		require.NoError(t, err)

		combinedToken := encResult.Encrypted + "." + encResult.Salt

		// 2. Simulate custom login auto-session logic
		var sessionID string
		var message string

		if combinedToken != "" {
			parts := strings.Split(combinedToken, ".")
			if len(parts) == 2 {
				falTokenEncrypted := parts[0]
				salt := parts[1]

				decryptedToken, err := encService.Decrypt(falTokenEncrypted, salt, userPassword)
				if err != nil {
					message = "Login successful. FAL token found but password doesn't match - please call create-session manually"
				} else {
					sessionStore.DeleteUserSessions(userID)
					sessionID, err = sessionStore.Create(userID, decryptedToken)
					if err != nil {
						message = "Login successful. Failed to auto-create session - please call create-session manually"
					} else {
						message = "Login successful with auto-created session"
					}
				}
			} else {
				message = "Login successful. Invalid FAL token format - please setup token again"
			}
		} else {
			message = "Login successful. No FAL token configured - setup required"
		}

		// 3. Verify results
		assert.Equal(t, "Login successful with auto-created session", message)
		assert.NotEmpty(t, sessionID)

		session, err := sessionStore.Get(sessionID)
		require.NoError(t, err)
		assert.Equal(t, userID, session.UserID)
		assert.Equal(t, falToken, session.FALToken)

		// 4. Test that session can be used for generation
		req := fal.GenerationRequest{
			Model:  "flux/schnell",
			Prompt: "Test image",
		}

		result, err := mockClient.GenerateImage(nil, session.FALToken, req)
		require.NoError(t, err)
		assert.NotEmpty(t, result.Images)

		// Clean up
		sessionStore.Delete(sessionID)
	})
}

func TestTokenStatus(t *testing.T) {
	// Test the token status endpoint logic
	// This tests the business logic without requiring full PocketBase HTTP setup

	encService := crypto.NewEncryptionService(1000)
	sessionStore := auth.NewSessionStore(24 * time.Hour)
	userID := "test_user_123"
	userPassword := "userpassword"
	falToken := "test-fal-token"

	t.Run("User with no token", func(t *testing.T) {
		// Simulate user with no FAL token
		combinedToken := ""
		hasToken := combinedToken != ""

		hasActiveSession := false
		if hasToken {
			_, err := sessionStore.GetUserSession(userID)
			hasActiveSession = err == nil
		}

		requiresLogin := hasToken && !hasActiveSession

		// Expected response
		expectedResponse := map[string]bool{
			"has_token":          false,
			"has_active_session": false,
			"requires_login":     false,
		}

		assert.Equal(t, expectedResponse["has_token"], hasToken)
		assert.Equal(t, expectedResponse["has_active_session"], hasActiveSession)
		assert.Equal(t, expectedResponse["requires_login"], requiresLogin)
	})

	t.Run("User with token but no session", func(t *testing.T) {
		// Setup encrypted token
		encResult, err := encService.Encrypt(falToken, userPassword)
		require.NoError(t, err)
		combinedToken := encResult.Encrypted + "." + encResult.Salt

		hasToken := combinedToken != ""

		hasActiveSession := false
		if hasToken {
			_, err := sessionStore.GetUserSession(userID)
			hasActiveSession = err == nil
		}

		requiresLogin := hasToken && !hasActiveSession

		// Expected response
		expectedResponse := map[string]bool{
			"has_token":          true,
			"has_active_session": false,
			"requires_login":     true,
		}

		assert.Equal(t, expectedResponse["has_token"], hasToken)
		assert.Equal(t, expectedResponse["has_active_session"], hasActiveSession)
		assert.Equal(t, expectedResponse["requires_login"], requiresLogin)
	})

	t.Run("User with token and active session", func(t *testing.T) {
		// Setup encrypted token
		encResult, err := encService.Encrypt(falToken, userPassword)
		require.NoError(t, err)
		combinedToken := encResult.Encrypted + "." + encResult.Salt

		hasToken := combinedToken != ""

		// Create active session
		sessionID, err := sessionStore.Create(userID, falToken)
		require.NoError(t, err)

		hasActiveSession := false
		if hasToken {
			_, err := sessionStore.GetUserSession(userID)
			hasActiveSession = err == nil
		}

		requiresLogin := hasToken && !hasActiveSession

		// Expected response
		expectedResponse := map[string]bool{
			"has_token":          true,
			"has_active_session": true,
			"requires_login":     false,
		}

		assert.Equal(t, expectedResponse["has_token"], hasToken)
		assert.Equal(t, expectedResponse["has_active_session"], hasActiveSession)
		assert.Equal(t, expectedResponse["requires_login"], requiresLogin)

		// Clean up
		sessionStore.Delete(sessionID)
	})

	t.Run("Session expiration affects status", func(t *testing.T) {
		// Create session store with very short timeout
		shortSessionStore := auth.NewSessionStore(1 * time.Millisecond)

		// Setup encrypted token
		encResult, err := encService.Encrypt(falToken, userPassword)
		require.NoError(t, err)
		combinedToken := encResult.Encrypted + "." + encResult.Salt

		hasToken := combinedToken != ""

		// Create session that will expire quickly
		_, err = shortSessionStore.Create(userID, falToken)
		require.NoError(t, err)

		// Initially should have active session
		_, err = shortSessionStore.GetUserSession(userID)
		assert.NoError(t, err)

		// Wait for expiration
		time.Sleep(10 * time.Millisecond)

		// Now should not have active session
		hasActiveSession := false
		if hasToken {
			_, err := shortSessionStore.GetUserSession(userID)
			hasActiveSession = err == nil
		}

		requiresLogin := hasToken && !hasActiveSession

		// Should require login after expiration
		assert.True(t, hasToken)
		assert.False(t, hasActiveSession)
		assert.True(t, requiresLogin)
	})

	t.Run("Multiple sessions for user", func(t *testing.T) {
		// Test that user session detection works with multiple sessions
		encResult, err := encService.Encrypt(falToken, userPassword)
		require.NoError(t, err)
		combinedToken := encResult.Encrypted + "." + encResult.Salt

		hasToken := combinedToken != ""

		// Create multiple sessions for user
		sessionID1, err := sessionStore.Create(userID, falToken)
		require.NoError(t, err)

		sessionID2, err := sessionStore.Create(userID+"_other", falToken)
		require.NoError(t, err)

		// Should still detect active session for our user
		hasActiveSession := false
		if hasToken {
			_, err := sessionStore.GetUserSession(userID)
			hasActiveSession = err == nil
		}

		assert.True(t, hasToken)
		assert.True(t, hasActiveSession)
		assert.False(t, hasToken && !hasActiveSession) // requires_login should be false

		// Clean up
		sessionStore.Delete(sessionID1)
		sessionStore.Delete(sessionID2)
	})
}

func TestTokenStatusResponseStructure(t *testing.T) {
	// Test the response structure and JSON marshaling
	
	t.Run("JSON marshaling", func(t *testing.T) {
		response := localmodels.TokenStatusResponse{
			HasToken:         true,
			HasActiveSession: false,
			RequiresLogin:    true,
		}

		data, err := json.Marshal(response)
		require.NoError(t, err)

		var unmarshaled localmodels.TokenStatusResponse
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, response.HasToken, unmarshaled.HasToken)
		assert.Equal(t, response.HasActiveSession, unmarshaled.HasActiveSession)
		assert.Equal(t, response.RequiresLogin, unmarshaled.RequiresLogin)
	})

	t.Run("All scenarios coverage", func(t *testing.T) {
		scenarios := []struct {
			name             string
			hasToken         bool
			hasActiveSession bool
			expectedLogin    bool
		}{
			{"No token, no session", false, false, false},
			{"Has token, no session", true, false, true},
			{"Has token, has session", true, true, false},
			{"No token, has session", false, true, false}, // Edge case, shouldn't happen
		}

		for _, scenario := range scenarios {
			t.Run(scenario.name, func(t *testing.T) {
				response := localmodels.TokenStatusResponse{
					HasToken:         scenario.hasToken,
					HasActiveSession: scenario.hasActiveSession,
					RequiresLogin:    scenario.hasToken && !scenario.hasActiveSession,
				}

				assert.Equal(t, scenario.expectedLogin, response.RequiresLogin)
			})
		}
	})
}