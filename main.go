package main

import (
	"log"
	"os"
	"time"

	"myapp/internal/auth"
	"myapp/internal/crypto"
	"myapp/internal/fal"
	"myapp/internal/handlers"

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
		log.Println("📋 Required Setup:")
		log.Println("1. Create collections via PocketBase admin:")
		log.Println("   - generated_images")
		log.Println("   - model_preferences") 
		log.Println("   - collections")
		log.Println("2. Add fields to users collection:")
		log.Println("   - fal_token (text)")
		log.Println("   - salt (text)")
		log.Println("   - financial_data (json)")
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

		// Register example routes for testing
		handlers.RegisterExampleRoutes(se, app, sessionStore, encService, falClient)
		log.Println("✓ Example API routes registered")

		// Serve static files from the provided public dir (if exists)
		se.Router.GET("/{path...}", apis.Static(os.DirFS("./pb_public"), false))

		return se.Next()
	})

	log.Println("🚀 Starting Generatio PocketBase server...")
	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}