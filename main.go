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
	falClient := fal.NewClient("https://queue.fal.run/fal-ai")
	falClient.SetTimeout(10 * time.Minute) // 10-minute generation timeout
	log.Println("✓ FAL AI client initialized")

	// Create cleanup service
	cleanupService := auth.NewCleanupService(sessionStore, 1*time.Hour)
	log.Println("✓ Cleanup service initialized")

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

		log.Println("✓ Generatio PocketBase extension ready")
		log.Println("")
		log.Println("📋 Required Schema:")
		log.Println("1. Main collections expected:")
		log.Println("   - generatio_users (auth collection)")
		log.Println("   - images (for generated images)")
		log.Println("   - folders (for collections/organization)")
		log.Println("   - model_preferences (for user preferences)")
		log.Println("2. generatio_users collection should have:")
		log.Println("   - fal_token (text) - for encrypted FAL AI token")
		log.Println("   - financial_data (json) - for spending tracking & salt storage")
		log.Println("")
		log.Println("🔧 API Endpoints will be available at:")
		log.Println("   POST /api/custom/tokens/setup")
		log.Println("   POST /api/custom/tokens/verify")
		log.Println("   POST /api/custom/auth/create-session")
		log.Println("   DELETE /api/custom/auth/session")
		log.Println("   POST /api/custom/generate/image")
		log.Println("   GET /api/custom/generate/models")
		log.Println("   GET /api/custom/financial/stats")
		log.Println("   GET /api/custom/preferences/{model_name}")
		log.Println("   POST /api/custom/preferences/{model_name}")
		log.Println("   POST /api/custom/collections/create")
		log.Println("   GET /api/custom/collections")
		log.Println("")

		// Register production API routes
		handlers.RegisterRoutes(se, app, sessionStore, encService, falClient)
		log.Println("✓ API routes registered")

		// Serve static files from the provided public dir (if exists)
		se.Router.GET("/{path...}", apis.Static(os.DirFS("./pb_public"), false))

		return se.Next()
	})

	log.Println("🚀 Starting Generatio PocketBase server...")
	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}