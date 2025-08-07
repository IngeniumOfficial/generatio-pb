package auth

import (
	"log"
	"time"
)

// CleanupService manages background cleanup tasks for sessions
type CleanupService struct {
	sessionStore *SessionStore
	interval     time.Duration
	stopChan     chan struct{}
}

// NewCleanupService creates a new cleanup service
func NewCleanupService(sessionStore *SessionStore, interval time.Duration) *CleanupService {
	if interval <= 0 {
		interval = 1 * time.Hour // Default cleanup interval
	}

	return &CleanupService{
		sessionStore: sessionStore,
		interval:     interval,
		stopChan:     make(chan struct{}),
	}
}

// Start begins the background cleanup process
func (c *CleanupService) Start() {
	go c.run()
	log.Printf("Session cleanup service started with interval: %v", c.interval)
}

// Stop stops the background cleanup process
func (c *CleanupService) Stop() {
	close(c.stopChan)
	log.Println("Session cleanup service stopped")
}

// run is the main cleanup loop
func (c *CleanupService) run() {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.performCleanup()
		case <-c.stopChan:
			return
		}
	}
}

// performCleanup performs the actual cleanup of expired sessions
func (c *CleanupService) performCleanup() {
	startTime := time.Now()
	
	// Get stats before cleanup
	statsBefore := c.sessionStore.Stats()
	
	// Perform cleanup
	c.sessionStore.Cleanup()
	
	// Get stats after cleanup
	statsAfter := c.sessionStore.Stats()
	
	// Calculate cleanup metrics
	cleanedSessions := statsBefore.TotalSessions - statsAfter.TotalSessions
	duration := time.Since(startTime)
	
	if cleanedSessions > 0 {
		log.Printf("Session cleanup completed: removed %d expired sessions in %v", cleanedSessions, duration)
	}
	
	// Log stats if there are active sessions
	if statsAfter.ActiveSessions > 0 {
		log.Printf("Session stats: %d active, %d total", statsAfter.ActiveSessions, statsAfter.TotalSessions)
	}
}

// ForceCleanup performs an immediate cleanup
func (c *CleanupService) ForceCleanup() {
	c.performCleanup()
}

// GetStats returns current session statistics
func (c *CleanupService) GetStats() SessionStats {
	return c.sessionStore.Stats()
}