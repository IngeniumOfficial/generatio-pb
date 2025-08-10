package auth

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"generatio-pb/internal/models"

	"github.com/google/uuid"
)

// SessionStore manages in-memory user sessions
type SessionStore struct {
	sessions map[string]*models.Session
	mutex    sync.RWMutex
	timeout  time.Duration
}

// NewSessionStore creates a new session store with the specified timeout
func NewSessionStore(timeout time.Duration) *SessionStore {
	return &SessionStore{
		sessions: make(map[string]*models.Session),
		timeout:  timeout,
	}
}

// Create creates a new session for the user with their decrypted FAL token
func (s *SessionStore) Create(userID, falToken string) (string, error) {
	if userID == "" {
		return "", fmt.Errorf("user ID cannot be empty")
	}
	if falToken == "" {
		return "", fmt.Errorf("FAL token cannot be empty")
	}

	// Generate a cryptographically secure session ID
	sessionID := uuid.New().String()

	// Create session
	session := &models.Session{
		ID:        sessionID,
		UserID:    userID,
		FALToken:  falToken,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(s.timeout),
	}

	// Store session
	s.mutex.Lock()
	s.sessions[sessionID] = session
	s.mutex.Unlock()

	return sessionID, nil
}

// Get retrieves a session by ID
func (s *SessionStore) Get(sessionID string) (*models.Session, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID cannot be empty")
	}

	s.mutex.RLock()
	session, exists := s.sessions[sessionID]
	s.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	// Check if session has expired
	if session.IsExpired() {
		// Remove expired session
		s.Delete(sessionID)
		return nil, fmt.Errorf("session expired")
	}

	return session, nil
}

// Delete removes a session by ID
func (s *SessionStore) Delete(sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	session, exists := s.sessions[sessionID]
	if exists {
		// Clear sensitive data before deletion
		session.Clear()
		delete(s.sessions, sessionID)
	}

	return nil
}

// GetUserSession retrieves the active session for a user (if any)
func (s *SessionStore) GetUserSession(userID string) (*models.Session, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, session := range s.sessions {
		if session.UserID == userID && !session.IsExpired() {
			return session, nil
		}
	}

	return nil, fmt.Errorf("no active session found for user")
}

// DeleteUserSessions removes all sessions for a specific user
func (s *SessionStore) DeleteUserSessions(userID string) error {
	if userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	var toDelete []string
	for sessionID, session := range s.sessions {
		if session.UserID == userID {
			session.Clear()
			toDelete = append(toDelete, sessionID)
		}
	}

	for _, sessionID := range toDelete {
		delete(s.sessions, sessionID)
	}

	return nil
}

// Cleanup removes expired sessions from memory
func (s *SessionStore) Cleanup() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now()
	var toDelete []string

	for sessionID, session := range s.sessions {
		if now.After(session.ExpiresAt) {
			session.Clear()
			toDelete = append(toDelete, sessionID)
		}
	}

	for _, sessionID := range toDelete {
		delete(s.sessions, sessionID)
	}
}

// StartCleanup starts a background goroutine that periodically cleans up expired sessions
func (s *SessionStore) StartCleanup(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			s.Cleanup()
		}
	}()
}

// Stats returns statistics about the session store
func (s *SessionStore) Stats() SessionStats {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	stats := SessionStats{
		TotalSessions: len(s.sessions),
		ActiveSessions: 0,
		ExpiredSessions: 0,
	}

	now := time.Now()
	for _, session := range s.sessions {
		if now.After(session.ExpiresAt) {
			stats.ExpiredSessions++
		} else {
			stats.ActiveSessions++
		}
	}

	return stats
}

// SessionStats represents session store statistics
type SessionStats struct {
	TotalSessions   int `json:"total_sessions"`
	ActiveSessions  int `json:"active_sessions"`
	ExpiredSessions int `json:"expired_sessions"`
}

// ExtendSession extends the expiration time of a session
func (s *SessionStore) ExtendSession(sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found")
	}

	if session.IsExpired() {
		return fmt.Errorf("session already expired")
	}

	// Extend the session by the configured timeout
	session.ExpiresAt = time.Now().Add(s.timeout)
	return nil
}

// Clear removes all sessions from the store
func (s *SessionStore) Clear() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Clear sensitive data from all sessions
	for _, session := range s.sessions {
		session.Clear()
	}

	// Clear the map
	s.sessions = make(map[string]*models.Session)
}

// GetSessionCount returns the current number of sessions
func (s *SessionStore) GetSessionCount() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return len(s.sessions)
}

// ValidateSession checks if a session exists and is valid
func (s *SessionStore) ValidateSession(sessionID string) bool {
	session, err := s.Get(sessionID)
	return err == nil && session != nil && !session.IsExpired()
}

// GetFALToken retrieves the FAL token for a session
func (s *SessionStore) GetFALToken(sessionID string) (string, error) {
	session, err := s.Get(sessionID)
	if err != nil {
		return "", err
	}

	if session.FALToken == "" {
		return "", fmt.Errorf("no FAL token in session")
	}

	return session.FALToken, nil
}

// generateSecureID generates a cryptographically secure random ID
func generateSecureID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", bytes), nil
}