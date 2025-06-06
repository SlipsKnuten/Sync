package main

import (
	"log"
	"net/http"
	"os"

	"collab-editor/internal/auth"
	"collab-editor/internal/db"
	"collab-editor/internal/document"
	"collab-editor/internal/export"
	"collab-editor/internal/hub"
)

func enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func main() {
	// Initialize database
	database, err := db.New()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Close()

	// Initialize hub with database
	h := hub.New(database)

	// Initialize auth handler
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "your-secret-key-change-this-in-production"
	}
	authHandler := auth.NewAuthHandler(database, jwtSecret)

	// Set auth handler in hub
	h.SetAuthHandler(authHandler)

	// Initialize export handler
	exportHandler := export.NewExportHandler(database)

	// Initialize document handler
	documentHandler := document.NewDocumentHandler(database, authHandler)

	// Routes
	http.HandleFunc("/ws", enableCORS(func(w http.ResponseWriter, r *http.Request) {
		hub.ServeWS(h, w, r)
	}))

	http.HandleFunc("/api/register", enableCORS(authHandler.Register))
	http.HandleFunc("/api/login", enableCORS(authHandler.Login))
	http.HandleFunc("/api/sessions", enableCORS(authHandler.GetUserSessions))
	http.HandleFunc("/api/export", enableCORS(exportHandler.ExportDocument))
	http.HandleFunc("/api/document/save", enableCORS(documentHandler.SaveDocument))

	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}