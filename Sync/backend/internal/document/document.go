package document

import (
	"encoding/json"
	"net/http"
	"strings"

	"collab-editor/internal/auth"
	"collab-editor/internal/db"
)

type DocumentHandler struct {
	db   *db.Database
	auth *auth.AuthHandler
}

type SaveDocumentRequest struct {
	SessionCode string `json:"session_code"`
	Content     string `json:"content"`
}

func NewDocumentHandler(database *db.Database, authHandler *auth.AuthHandler) *DocumentHandler {
	return &DocumentHandler{
		db:   database,
		auth: authHandler,
	}
}

func (h *DocumentHandler) SaveDocument(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodOptions {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Handle OPTIONS for CORS
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Get token from Authorization header
	authHeader := r.Header.Get("Authorization")
	
	var req SaveDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if authHeader == "" {
		// Allow anonymous saves
		if err := h.db.SaveDocument(req.SessionCode, req.Content, nil); err != nil {
			http.Error(w, "Failed to save document", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "saved", "type": "anonymous"})
		return
	}

	// Authenticated save
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	userID, err := h.auth.ValidateToken(tokenString)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Save with user ID
	if err := h.db.SaveDocument(req.SessionCode, req.Content, &userID); err != nil {
		http.Error(w, "Failed to save document", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "saved", 
		"user_id": userID,
		"type": "authenticated",
	})
}