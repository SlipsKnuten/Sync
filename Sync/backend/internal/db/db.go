package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type Database struct {
	conn *sql.DB
}

type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type Session struct {
	ID           int       `json:"id"`
	SessionCode  string    `json:"session_code"`
	Content      string    `json:"content"`
	LastModified time.Time `json:"last_modified"`
}

func New() (*Database, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://collabuser:collabpass@localhost:5432/collabdb?sslmode=disable"
	}

	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Connected to PostgreSQL database")
	return &Database{conn: conn}, nil
}

func (db *Database) Close() error {
	return db.conn.Close()
}

// User operations
func (db *Database) CreateUser(username, email, password string) (*User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	var user User
	err = db.conn.QueryRow(`
        INSERT INTO users (username, email, password_hash)
        VALUES ($1, $2, $3)
        RETURNING id, username, email, created_at
    `, username, email, string(hashedPassword)).Scan(
		&user.ID, &user.Username, &user.Email, &user.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (db *Database) GetUserByUsername(username string) (*User, error) {
	var user User
	// var passwordHash string // REMOVED

	err := db.conn.QueryRow(`
        SELECT id, username, email, created_at
        FROM users WHERE username = $1
    `, username).Scan(&user.ID, &user.Username, &user.Email, &user.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows { // Optional: More specific error for not found
			return nil, fmt.Errorf("user %s not found", username)
		}
		return nil, err
	}

	return &user, nil
}

func (db *Database) VerifyUserPassword(username, password string) (*User, error) {
	var user User
	var passwordHash string

	err := db.conn.QueryRow(`
        SELECT id, username, email, password_hash, created_at
        FROM users WHERE username = $1
    `, username).Scan(&user.ID, &user.Username, &user.Email, &passwordHash, &user.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows { // Optional: More specific error for not found
			return nil, fmt.Errorf("user %s not found", username)
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid password") // Don't reveal if username or password was wrong for security
	}

	return &user, nil
}

// Session operations
func (db *Database) GetOrCreateSession(sessionCode string) (*Session, error) {
	var session Session

	// Try to get existing session
	err := db.conn.QueryRow(`
        SELECT es.id, es.session_code, COALESCE(d.content, ''), es.last_modified
        FROM editing_sessions es
        LEFT JOIN documents d ON es.id = d.session_id AND d.version = (SELECT MAX(version) FROM documents WHERE session_id = es.id)
        WHERE es.session_code = $1
    `, sessionCode).Scan(&session.ID, &session.SessionCode, &session.Content, &session.LastModified)
	// Note: I modified the JOIN condition for documents to ensure you get the LATEST document version
	// The original ORDER BY ... LIMIT 1 outside a subquery for this join might not always give the latest document if there are multiple documents for one session and you're also joining other tables.
	// A more robust way to get the latest document for a session is often a subquery or a window function, but the above is a common pattern.
	// If you only ever expect one document row per session in the documents table (e.g., always updating, not versioning in that table), your original was fine.
	// Given your SaveDocument creates versions, ensuring you get the latest here is important.

	if err == sql.ErrNoRows {
		// Create new session
		err = db.conn.QueryRow(`
            INSERT INTO editing_sessions (session_code)
            VALUES ($1)
            RETURNING id, session_code, last_modified
        `, sessionCode).Scan(&session.ID, &session.SessionCode, &session.LastModified)

		if err != nil {
			return nil, fmt.Errorf("failed to create session: %w", err)
		}

		session.Content = "" // Default content for new session
	} else if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return &session, nil
}

func (db *Database) SaveDocument(sessionCode, content string, userID *int) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // Ensure rollback on panic or error before commit

	// Get session ID
	var sessionID int
	err = tx.QueryRow(`
        SELECT id FROM editing_sessions WHERE session_code = $1
    `, sessionCode).Scan(&sessionID)

	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("session with code %s not found", sessionCode)
		}
		return err
	}

	// Update last_modified
	_, err = tx.Exec(`
        UPDATE editing_sessions SET last_modified = CURRENT_TIMESTAMP WHERE id = $1
    `, sessionID)

	if err != nil {
		return err
	}

	// Insert new document version
	// This subquery for version needs to be correct.
	// If there are no existing documents for the session_id, MAX(version) will be NULL, COALESCE(NULL, 0) + 1 = 1. Correct.
	// If there are existing documents, it takes the max and adds 1. Correct.
	_, err = tx.Exec(`
        INSERT INTO documents (session_id, content, version)
        VALUES ($1, $2, (SELECT COALESCE(MAX(version), 0) + 1 FROM documents WHERE session_id = $1))
    `, sessionID, content)

	if err != nil {
		return err
	}

	// If user is logged in, update user_sessions
	if userID != nil {
		_, err = tx.Exec(`
            INSERT INTO user_sessions (user_id, session_id, last_seen)
            VALUES ($1, $2, CURRENT_TIMESTAMP)
            ON CONFLICT (user_id, session_id) 
            DO UPDATE SET last_seen = CURRENT_TIMESTAMP
        `, *userID, sessionID)

		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (db *Database) GetUserSessions(userID int) ([]Session, error) {
	rows, err := db.conn.Query(`
		WITH LatestDocuments AS (
			SELECT 
				session_id, 
				content,
				ROW_NUMBER() OVER(PARTITION BY session_id ORDER BY version DESC) as rn
			FROM documents
		)
        SELECT es.id, es.session_code, COALESCE(ld.content, ''), es.last_modified
        FROM user_sessions us
        JOIN editing_sessions es ON us.session_id = es.id
        LEFT JOIN LatestDocuments ld ON es.id = ld.session_id AND ld.rn = 1
        WHERE us.user_id = $1
        ORDER BY us.last_seen DESC
    `, userID)
	// Note: Modified this query to more reliably get the LATEST document content using a CTE with ROW_NUMBER().
	// Your original LEFT JOIN documents d ON es.id = d.session_id could potentially return multiple rows per session if a session had multiple document versions,
	// which would then lead to Scan errors or incorrect data.

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var session Session
		if err := rows.Scan(&session.ID, &session.SessionCode, &session.Content, &session.LastModified); err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}
	if err = rows.Err(); err != nil { // Check for errors during iteration
		return nil, err
	}

	return sessions, nil
}
