package main

import (
	"log"
	"os"
	"time"

	"generatio-pb/internal/auth"
	"generatio-pb/internal/crypto"
	"generatio-pb/internal/fal"
	"generatio-pb/internal/handlers"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

var (
	// App version
	version = "0.1.01"
)

func main() {
	app := pocketbase.New()

	// Initialize services
	log.Println("Initializing Generatio PocketBase extension...")

	// Create encryption service
	encService := crypto.NewEncryptionService(100000) // 100k PBKDF2 iterations
	log.Println("✓ Encryption service initialized")

	// Create session store with 24-hour timeout
	sessionStore := auth.NewSessionStore(24 * time.Hour)
	log.Println("✓ Session store initialized")

	// Create FAL AI client
	falClient := fal.NewClient("https://queue.fal.run")
	falClient.SetTimeout(10 * time.Minute) // 10-minute generation timeout
	log.Println("✓ FAL AI client initialized")

	// Create cleanup service
	cleanupService := auth.NewCleanupService(sessionStore, 1*time.Hour)
	log.Println("✓ Cleanup service initialized")

	// Note: Session management uses standard PocketBase auth + token-status check
	// Clients can use token-status endpoint to determine if session creation is needed
	log.Println("✓ Session management configured with token-status endpoint")

	// Setup on serve
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		log.Println("Setting up Generatio services...")

		// Start cleanup service
		cleanupService.Start()
		log.Println("✓ Session cleanup service started")

		// Log available models
		models := falClient.GetModels()
		log.Printf("✓ FAL AI models available: %d", len(models))
		for modelName := range models {
			log.Printf("  - %s", modelName)
		}

		
		// Serve static files from the provided public dir (if exists) - register BEFORE custom routes
		se.Router.GET("/static/{path...}", apis.Static(os.DirFS("./pb_public"), false))
		
		// Register production API routes
		handlers.RegisterRoutes(se, app, sessionStore, encService, falClient)
		log.Println("✓ API routes registered")
		
		log.Println("Launching Generatio Pocketbase Version " + version)
		
		return se.Next()
	})

	log.Println("🚀 Starting Generatio PocketBase server...")
	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}