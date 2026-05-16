package main

import (
	"context"
	"log"

	"dadv-project/internal/auth"
	"dadv-project/internal/db"
)

func main() {
	database, err := db.New("./data/dadv.db")
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer database.Close()

	ctx := context.Background()

	// Add dummy users
	users := []struct {
		Email    string
		Name     string
		Password string
	}{
		{"admin@example.com", "Admin User", "password123"},
		{"test@example.com", "Test User", "password123"},
		{"john.doe@example.com", "John Doe", "password123"},
	}

	for _, u := range users {
		exists, _ := database.EmailExists(ctx, u.Email)
		if !exists {
			hash, _ := auth.HashPassword(u.Password)
			userID, err := database.CreateUser(ctx, u.Email, hash, u.Name)
			if err != nil {
				log.Printf("Failed to create user %s: %v", u.Email, err)
				continue
			}
			log.Printf("Created user %s with ID %d", u.Email, userID)

			// Add some dummy files and metadata for this user
			fileID, err := database.CreateFile(ctx, userID, "dummy-job-"+u.Name, "test_data.csv", 1024, "text/csv")
			if err == nil {
				database.UpdateFileStatus(ctx, "dummy-job-"+u.Name, "completed")
				
				// Add dummy metadata
				database.InsertMetadataBatch(ctx, []map[string]interface{}{
					{
						"file_id": fileID,
						"name": "test_data.csv",
						"path": "/data/uploads/test_data.csv",
						"size": 1024,
						"extension": ".csv",
						"mime_type": "text/csv",
						"created_at": "2023-01-01T00:00:00Z",
						"modified_at": "2023-01-01T00:00:00Z",
						"accessed_at": "2023-01-01T00:00:00Z",
						"owner": "admin",
						"group": "admin",
						"permissions": "644",
					},
				})
				
				// Add dummy analysis result
				database.InsertAnalysisResult(ctx, fileID, "count", "total_files", 1.0, "")
			}
		} else {
			log.Printf("User %s already exists", u.Email)
		}
	}

	log.Println("Seeding complete.")
}
