package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"

	"golang.org/x/crypto/pbkdf2"
)

const (
	// PBKDF2 parameters
	DefaultIterations = 100000
	SaltSize         = 32
	KeySize          = 32 // AES-256
	NonceSize        = 12 // GCM standard nonce size
)

// EncryptionService provides AES-256-GCM encryption with PBKDF2 key derivation
type EncryptionService struct {
	iterations int
}

// NewEncryptionService creates a new encryption service with specified PBKDF2 iterations
func NewEncryptionService(iterations int) *EncryptionService {
	if iterations <= 0 {
		iterations = DefaultIterations
	}
	return &EncryptionService{
		iterations: iterations,
	}
}

// EncryptResult contains the encrypted data and salt
type EncryptResult struct {
	Encrypted string `json:"encrypted"`
	Salt      string `json:"salt"`
}

// Encrypt encrypts plaintext using AES-256-GCM with a key derived from password and salt
func (e *EncryptionService) Encrypt(plaintext, password string) (*EncryptResult, error) {
	if plaintext == "" {
		return nil, errors.New("plaintext cannot be empty")
	}
	if password == "" {
		return nil, errors.New("password cannot be empty")
	}

	// Generate random salt
	salt, err := e.generateSalt()
	if err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive key from password and salt
	key := e.deriveKey([]byte(password), salt)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt the plaintext
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Encode to base64 for storage
	encrypted := base64.StdEncoding.EncodeToString(ciphertext)
	saltB64 := base64.StdEncoding.EncodeToString(salt)

	return &EncryptResult{
		Encrypted: encrypted,
		Salt:      saltB64,
	}, nil
}

// Decrypt decrypts ciphertext using AES-256-GCM with a key derived from password and salt
func (e *EncryptionService) Decrypt(encrypted, salt, password string) (string, error) {
	if encrypted == "" {
		return "", errors.New("encrypted data cannot be empty")
	}
	if salt == "" {
		return "", errors.New("salt cannot be empty")
	}
	if password == "" {
		return "", errors.New("password cannot be empty")
	}

	// Decode from base64
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", fmt.Errorf("failed to decode encrypted data: %w", err)
	}

	saltBytes, err := base64.StdEncoding.DecodeString(salt)
	if err != nil {
		return "", fmt.Errorf("failed to decode salt: %w", err)
	}

	// Validate minimum length (nonce + at least some data + tag)
	if len(ciphertext) < NonceSize+16 { // 16 is GCM tag size
		return "", errors.New("ciphertext too short")
	}

	// Derive key from password and salt
	key := e.deriveKey([]byte(password), saltBytes)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Extract nonce and ciphertext
	nonce := ciphertext[:NonceSize]
	ciphertextData := ciphertext[NonceSize:]

	// Decrypt the ciphertext
	plaintext, err := gcm.Open(nil, nonce, ciphertextData, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// deriveKey derives a key from password and salt using PBKDF2-SHA256
func (e *EncryptionService) deriveKey(password, salt []byte) []byte {
	return pbkdf2.Key(password, salt, e.iterations, KeySize, sha256.New)
}

// generateSalt generates a cryptographically secure random salt
func (e *EncryptionService) generateSalt() ([]byte, error) {
	salt := make([]byte, SaltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	return salt, nil
}

// VerifyPassword verifies if a password can decrypt the given encrypted data
func (e *EncryptionService) VerifyPassword(encrypted, salt, password string) bool {
	_, err := e.Decrypt(encrypted, salt, password)
	return err == nil
}

// ClearMemory attempts to clear sensitive data from memory
func ClearMemory(data []byte) {
	for i := range data {
		data[i] = 0
	}
}

// ClearString attempts to clear sensitive string data from memory
func ClearString(s *string) {
	if s != nil {
		*s = ""
	}
}