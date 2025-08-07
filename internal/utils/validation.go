package utils

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"myapp/internal/fal"
	"myapp/internal/models"
)

// ValidatePrompt validates and sanitizes a generation prompt
func ValidatePrompt(prompt string) error {
	if prompt == "" {
		return NewValidationError("prompt cannot be empty")
	}

	// Check length
	if utf8.RuneCountInString(prompt) > 1000 {
		return NewValidationError("prompt cannot exceed 1000 characters")
	}

	// Check for potentially harmful content
	if containsHarmfulContent(prompt) {
		return NewValidationError("prompt contains inappropriate content")
	}

	return nil
}

// SanitizePrompt sanitizes a prompt by removing potentially harmful content
func SanitizePrompt(prompt string) string {
	// Remove excessive whitespace
	prompt = strings.TrimSpace(prompt)
	prompt = regexp.MustCompile(`\s+`).ReplaceAllString(prompt, " ")

	// Remove potential injection attempts
	prompt = strings.ReplaceAll(prompt, "<script", "")
	prompt = strings.ReplaceAll(prompt, "</script>", "")
	prompt = strings.ReplaceAll(prompt, "javascript:", "")
	prompt = strings.ReplaceAll(prompt, "data:", "")

	return prompt
}

// ValidateGenerationRequest validates a complete generation request
func ValidateGenerationRequest(req *models.GenerateImageRequest) error {
	// Validate prompt
	if err := ValidatePrompt(req.Prompt); err != nil {
		return err
	}

	// Validate model
	model, exists := fal.GetModel(req.Model)
	if !exists {
		return NewValidationError("unsupported model: " + req.Model)
	}

	// Validate parameters
	if req.Parameters != nil {
		if err := model.ValidateParameters(req.Parameters); err != nil {
			return NewValidationError("invalid parameters: " + err.Error())
		}
	}

	// Sanitize prompt
	req.Prompt = SanitizePrompt(req.Prompt)

	return nil
}

// ValidateCollectionName validates a collection name
func ValidateCollectionName(name string) error {
	if name == "" {
		return NewValidationError("collection name cannot be empty")
	}

	if utf8.RuneCountInString(name) > 100 {
		return NewValidationError("collection name cannot exceed 100 characters")
	}

	// Check for invalid characters
	if strings.ContainsAny(name, "<>:\"/\\|?*") {
		return NewValidationError("collection name contains invalid characters")
	}

	return nil
}

// ValidateUUID validates a UUID string
func ValidateUUID(id string) error {
	if id == "" {
		return NewValidationError("ID cannot be empty")
	}

	// Simple UUID validation (basic format check)
	uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	if !uuidRegex.MatchString(id) {
		return NewValidationError("invalid ID format")
	}

	return nil
}

// ValidateEmail validates an email address
func ValidateEmail(email string) error {
	if email == "" {
		return NewValidationError("email cannot be empty")
	}

	// Basic email validation
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return NewValidationError("invalid email format")
	}

	return nil
}

// ValidatePassword validates a password
func ValidatePassword(password string) error {
	if password == "" {
		return NewValidationError("password cannot be empty")
	}

	if len(password) < 8 {
		return NewValidationError("password must be at least 8 characters long")
	}

	if len(password) > 128 {
		return NewValidationError("password cannot exceed 128 characters")
	}

	return nil
}

// ValidateFALToken validates a FAL AI token format
func ValidateFALToken(token string) error {
	if token == "" {
		return NewValidationError("FAL token cannot be empty")
	}

	// Basic token format validation
	if len(token) < 10 {
		return NewValidationError("FAL token appears to be too short")
	}

	if len(token) > 200 {
		return NewValidationError("FAL token appears to be too long")
	}

	// Check for obvious invalid characters
	if strings.ContainsAny(token, " \t\n\r") {
		return NewValidationError("FAL token contains invalid characters")
	}

	return nil
}

// ValidatePreferences validates user preferences for a model
func ValidatePreferences(modelName string, preferences map[string]interface{}) error {
	// Get model info
	model, exists := fal.GetModel(modelName)
	if !exists {
		return NewValidationError("unsupported model: " + modelName)
	}

	// Validate preferences against model parameters
	if err := model.ValidateParameters(preferences); err != nil {
		return NewValidationError("invalid preferences: " + err.Error())
	}

	return nil
}

// ValidateImageIDs validates a list of image IDs
func ValidateImageIDs(imageIDs []string) error {
	if len(imageIDs) == 0 {
		return NewValidationError("at least one image ID is required")
	}

	if len(imageIDs) > 100 {
		return NewValidationError("cannot process more than 100 images at once")
	}

	for i, id := range imageIDs {
		if err := ValidateUUID(id); err != nil {
			return NewValidationError(fmt.Sprintf("invalid image ID at index %d: %s", i, err.Error()))
		}
	}

	return nil
}

// ValidateSessionID validates a session ID
func ValidateSessionID(sessionID string) error {
	if sessionID == "" {
		return NewValidationError("session ID cannot be empty")
	}

	// Session IDs should be UUIDs
	return ValidateUUID(sessionID)
}

// ValidatePagination validates pagination parameters
func ValidatePagination(page, limit int) error {
	if page < 1 {
		return NewValidationError("page must be at least 1")
	}

	if limit < 1 {
		return NewValidationError("limit must be at least 1")
	}

	if limit > 100 {
		return NewValidationError("limit cannot exceed 100")
	}

	return nil
}

// containsHarmfulContent checks for potentially harmful content in prompts
func containsHarmfulContent(prompt string) bool {
	// Convert to lowercase for case-insensitive matching
	lower := strings.ToLower(prompt)

	// List of potentially harmful keywords/patterns
	harmfulPatterns := []string{
		// Violence and harm
		"violence", "kill", "murder", "death", "suicide", "self-harm",
		"torture", "abuse", "assault", "weapon", "bomb", "explosive",
		
		// Adult content
		"nude", "naked", "sex", "porn", "erotic", "adult", "nsfw",
		
		// Hate speech
		"hate", "racist", "nazi", "terrorist", "extremist",
		
		// Personal information
		"ssn", "social security", "credit card", "password", "private key",
		
		// Illegal activities
		"drug", "illegal", "piracy", "fraud", "scam", "hack",
	}

	for _, pattern := range harmfulPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}

	return false
}

// SanitizeInput sanitizes general string input
func SanitizeInput(input string) string {
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")
	
	// Trim whitespace
	input = strings.TrimSpace(input)
	
	// Remove control characters except newlines and tabs
	var result strings.Builder
	for _, r := range input {
		if r >= 32 || r == '\n' || r == '\t' {
			result.WriteRune(r)
		}
	}
	
	return result.String()
}

// ValidateJSONField validates that a field contains valid JSON
func ValidateJSONField(field string, data interface{}) error {
	if data == nil {
		return nil // Allow nil values
	}

	// Try to marshal and unmarshal to validate JSON structure
	_, err := json.Marshal(data)
	if err != nil {
		return NewValidationError(fmt.Sprintf("invalid JSON in field %s: %s", field, err.Error()))
	}

	return nil
}

// ValidateNumericRange validates that a numeric value is within a specified range
func ValidateNumericRange(field string, value, min, max float64) error {
	if value < min {
		return NewValidationError(fmt.Sprintf("%s must be at least %.2f", field, min))
	}
	
	if value > max {
		return NewValidationError(fmt.Sprintf("%s must be at most %.2f", field, max))
	}
	
	return nil
}

// ValidateStringLength validates string length
func ValidateStringLength(field, value string, minLen, maxLen int) error {
	length := utf8.RuneCountInString(value)
	
	if length < minLen {
		return NewValidationError(fmt.Sprintf("%s must be at least %d characters", field, minLen))
	}
	
	if length > maxLen {
		return NewValidationError(fmt.Sprintf("%s cannot exceed %d characters", field, maxLen))
	}
	
	return nil
}