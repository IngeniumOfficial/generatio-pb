package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	localmodels "generatio-pb/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	baseURL           = "http://localhost:8090"
	integrationEmail    = "test@test.com"
	integrationPassword = "testpassword123"
	integrationFALToken = "test_fal_token_12345"
)

// HTTPClient wraps http.Client with additional functionality
type HTTPClient struct {
	client   *http.Client
	baseURL  string
	authToken string
}

// NewHTTPClient creates a new HTTP client for integration testing
func NewHTTPClient(baseURL string) *HTTPClient {
	return &HTTPClient{
		client:  &http.Client{Timeout: 30 * time.Second},
		baseURL: baseURL,
	}
}

// SetAuthToken sets the authentication token for subsequent requests
func (c *HTTPClient) SetAuthToken(token string) {
	c.authToken = token
}

// Request makes an HTTP request with optional authentication
func (c *HTTPClient) Request(method, path string, body interface{}, headers map[string]string) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, err
	}

	// Set default headers
	req.Header.Set("Content-Type", "application/json")
	
	// Add auth token if available
	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}

	// Add custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return c.client.Do(req)
}

// AuthenticateUser authenticates with PocketBase and returns the JWT token
func (c *HTTPClient) AuthenticateUser(email, password, collection string) (string, error) {
	authData := map[string]interface{}{
		"identity": email,
		"password": password,
	}

	resp, err := c.Request("POST", fmt.Sprintf("/api/collections/%s/auth-with-password", collection), authData, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("authentication failed: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	var authResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return "", err
	}

	token, ok := authResp["token"].(string)
	if !ok {
		return "", fmt.Errorf("no token in auth response")
	}

	c.SetAuthToken(token)
	return token, nil
}

// TestIntegrationModelsEndpoint tests the models endpoint (no auth required)
func TestIntegrationModelsEndpoint(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := NewHTTPClient(baseURL)
	
	resp, err := client.Request("GET", "/api/custom/generate/models", nil, nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	t.Logf("Models endpoint response: %d - %s", resp.StatusCode, string(bodyBytes))
	
	// Models endpoint may require auth, so we just log the response
	if resp.StatusCode == http.StatusOK {
		var response map[string]interface{}
		err = json.Unmarshal(bodyBytes, &response)
		if err == nil {
			t.Logf("Available models: %+v", response)
		}
	}
}

// TestIntegrationAuthFlow tests the complete authentication flow
func TestIntegrationAuthFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := NewHTTPClient(baseURL)

	t.Run("AuthenticateWithDefaultUsers", func(t *testing.T) {
		// Try authenticating with default users collection
		_, err := client.AuthenticateUser(integrationEmail, integrationPassword, "users")
		if err != nil {
			t.Logf("Authentication with 'users' collection failed: %v", err)
		} else {
			t.Log("Successfully authenticated with 'users' collection")
		}
	})

	t.Run("AuthenticateWithGeneratioUsers", func(t *testing.T) {
		// Try authenticating with generatio_users collection
		token, err := client.AuthenticateUser(integrationEmail, integrationPassword, "generatio_users")
		if err != nil {
			t.Logf("Authentication with 'generatio_users' collection failed: %v", err)
			t.Skip("Cannot test token setup without proper authentication")
		}
		
		t.Logf("Successfully authenticated with 'generatio_users' collection, token: %s...", token[:20])

		// Now test token setup
		t.Run("TokenSetup", func(t *testing.T) {
			setupReq := localmodels.SetupTokenRequest{
				FALToken: integrationFALToken,
				Password: integrationPassword,
			}

			resp, err := client.Request("POST", "/api/custom/tokens/setup", setupReq, nil)
			require.NoError(t, err)
			defer resp.Body.Close()

			bodyBytes, _ := io.ReadAll(resp.Body)
			t.Logf("Token setup response: %d - %s", resp.StatusCode, string(bodyBytes))

			if resp.StatusCode == http.StatusOK {
				var response map[string]interface{}
				err = json.Unmarshal(bodyBytes, &response)
				require.NoError(t, err)
				assert.True(t, response["success"].(bool))
			}
		})
	})
}

// TestIntegrationDebugRequest tests the exact request format you're using
func TestIntegrationDebugRequest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := NewHTTPClient(baseURL)

	// First try to authenticate with various collections
	collections := []string{"users", "generatio_users"}
	
	for _, collection := range collections {
		t.Run(fmt.Sprintf("TestWith%sCollection", collection), func(t *testing.T) {
			_, err := client.AuthenticateUser(integrationEmail, integrationPassword, collection)
			if err != nil {
				t.Logf("Authentication with '%s' failed: %v", collection, err)
				return
			}

			t.Logf("Authenticated successfully with '%s' collection", collection)

			// Test the exact request format from your frontend
			requestBody := map[string]interface{}{
				"fal_token": integrationFALToken,
				"password":  integrationPassword,
			}

			resp, err := client.Request("POST", "/api/custom/tokens/setup", requestBody, nil)
			require.NoError(t, err)
			defer resp.Body.Close()

			bodyBytes, _ := io.ReadAll(resp.Body)
			t.Logf("Response for %s collection: %d - %s", collection, resp.StatusCode, string(bodyBytes))

			// Check server logs for debugging info
			if resp.StatusCode != http.StatusOK {
				t.Logf("Check server logs for detailed debugging information")
			}
		})
	}
}

// TestManualDebugSteps provides manual testing steps
func TestManualDebugSteps(t *testing.T) {
	t.Log("Manual debugging steps:")
	t.Log("1. Ensure PocketBase server is running on http://localhost:8090")
	t.Log("2. Check server logs when making the request")
	t.Log("3. Verify you have 'generatio_users' auth collection set up")
	t.Log("4. Create a test user in the 'generatio_users' collection")
	t.Log("5. Use that user's credentials to authenticate")
	t.Log("6. Try calling the endpoint with proper Authorization header")
	t.Log("")
	t.Log("Example curl commands:")
	t.Log("# 1. Authenticate")
	t.Log(`curl -X POST http://localhost:8090/api/collections/generatio_users/auth-with-password \`)
	t.Log(`  -H "Content-Type: application/json" \`)
	t.Log(`  -d '{"identity":"test@test.com","password":"testpassword123"}'`)
	t.Log("")
	t.Log("# 2. Use the token from step 1 in Authorization header")
	t.Log(`curl -X POST http://localhost:8090/api/custom/tokens/setup \`)
	t.Log(`  -H "Content-Type: application/json" \`)
	t.Log(`  -H "Authorization: Bearer YOUR_TOKEN_HERE" \`)
	t.Log(`  -d '{"fal_token":"test_token","password":"testpassword123"}'`)
}