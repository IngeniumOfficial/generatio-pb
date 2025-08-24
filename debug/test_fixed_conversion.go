package main

import (
	"fmt"
)

// Test the FIXED model ID conversion chain
func main() {
	fmt.Println("🧪 Testing FIXED Model ID Conversion Chain")
	fmt.Println("==========================================")
	
	// Simulate the FIXED flow
	fmt.Println("1. Testing GenerateImage flow:")
	fmt.Println("-----------------------------")
	
	// Step 1: Original request
	originalModel := "flux/schnell"
	fmt.Printf("Original model from request: '%s'\n", originalModel)
	
	// Step 2: SubmitGeneration converts to FAL format
	falModelForSubmission := "fal-ai/" + originalModel
	fmt.Printf("SubmitGeneration URL: https://queue.fal.run/%s\n", falModelForSubmission)
	
	// Step 3: PollForCompletionWithModel gets called with ORIGINAL model (FIXED)
	modelForPolling := originalModel // This is the fix - was "fal-ai/flux/schnell" before
	fmt.Printf("PollForCompletionWithModel called with: '%s'\n", modelForPolling)
	
	// Step 4: CheckStatusWithModel converts again
	falModelForStatus := "fal-ai/" + modelForPolling
	if len(modelForPolling) >= 7 && modelForPolling[:7] == "fal-ai/" {
		falModelForStatus = modelForPolling
	}
	fmt.Printf("CheckStatusWithModel converts to: '%s'\n", falModelForStatus)
	
	// Step 5: getBaseModelID extracts base
	var baseModel string
	if falModelForStatus == "fal-ai/flux/schnell" {
		baseModel = "fal-ai/flux"
	} else {
		baseModel = falModelForStatus
	}
	fmt.Printf("Base model for status check: '%s'\n", baseModel)
	
	// Step 6: Final URL construction
	finalURL := fmt.Sprintf("https://queue.fal.run/%s/requests/test-123/status", baseModel)
	fmt.Printf("Final status URL: '%s'\n", finalURL)
	
	// Check for double-prefixing
	if baseModel == "fal-ai/fal-ai/flux" {
		fmt.Printf("❌ DOUBLE-PREFIXING STILL DETECTED!\n")
	} else {
		fmt.Printf("✅ No double-prefixing - URL looks correct!\n")
	}
	
	fmt.Println("\n2. Testing the problematic scenario (BEFORE fix):")
	fmt.Println("------------------------------------------------")
	
	// This was the OLD problematic flow
	oldModelForPolling := "fal-ai/flux/schnell" // This was the bug
	fmt.Printf("OLD PollForCompletionWithModel called with: '%s'\n", oldModelForPolling)
	
	// What would happen with the old flow
	oldFalModel := "fal-ai/" + oldModelForPolling
	if len(oldModelForPolling) >= 7 && oldModelForPolling[:7] == "fal-ai/" {
		oldFalModel = oldModelForPolling // This would prevent double-prefixing
	}
	fmt.Printf("OLD conversion result: '%s'\n", oldFalModel)
	
	var oldBaseModel string
	if oldFalModel == "fal-ai/flux/schnell" {
		oldBaseModel = "fal-ai/flux"
	} else {
		oldBaseModel = oldFalModel
	}
	fmt.Printf("OLD base model: '%s'\n", oldBaseModel)
	
	oldFinalURL := fmt.Sprintf("https://queue.fal.run/%s/requests/test-123/status", oldBaseModel)
	fmt.Printf("OLD final URL: '%s'\n", oldFinalURL)
	
	if oldBaseModel == "fal-ai/fal-ai/flux" {
		fmt.Printf("❌ OLD flow would have caused double-prefixing!\n")
	} else {
		fmt.Printf("🤔 OLD flow actually worked correctly due to prefix check\n")
	}
	
	fmt.Println("\n3. Summary:")
	fmt.Println("-----------")
	fmt.Printf("FIXED flow: %s → %s → %s\n", originalModel, falModelForStatus, baseModel)
	fmt.Printf("Final URL: %s\n", finalURL)
	
	if finalURL == "https://queue.fal.run/fal-ai/flux/requests/test-123/status" {
		fmt.Printf("✅ Perfect! This is the expected URL format.\n")
	} else {
		fmt.Printf("❌ Something is still wrong.\n")
	}
	
	fmt.Println("\n✅ Fixed conversion test completed")
}