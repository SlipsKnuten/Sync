package hub

import (
	"log"
	"net/http"
	"sync"
	"time"

	"collab-editor/internal/client"
	"collab-editor/internal/db"
	"collab-editor/internal/message"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var colors = []string{"#FF6B6B", "#4ECDC4", "#45B7D1", "#96CEB4", "#DDA0DD", "#F4A460"}

type Session struct {
	clients     map[*client.Client]bool
	broadcast   chan message.Message
	register    chan *client.Client
	unregister  chan *client.Client
	document    string
	sessionCode string
	mutex       sync.RWMutex
	colorIndex  int
	colorMutex  sync.Mutex
	lastSave    time.Time
	saveTimer   *time.Timer
	db          *db.Database
}

type Hub struct {
	sessions map[string]*Session
	mutex    sync.RWMutex
	db       *db.Database
}

func New(database *db.Database) *Hub {
	return &Hub{
		sessions: make(map[string]*Session),
		db:       database,
	}
}

func (h *Hub) GetOrCreateSession(sessionCode string) *Session {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if session, exists := h.sessions[sessionCode]; exists {
		return session
	}

	// Load session from database
	dbSession, err := h.db.GetOrCreateSession(sessionCode)
	if err != nil {
		log.Printf("Failed to load session from database: %v", err)
	}

	// Create new session
	session := &Session{
		broadcast:   make(chan message.Message),
		register:    make(chan *client.Client),
		unregister:  make(chan *client.Client),
		clients:     make(map[*client.Client]bool),
		document:    dbSession.Content,
		sessionCode: sessionCode,
		colorIndex:  0,
		db:          h.db,
		lastSave:    time.Now(),
	}

	h.sessions[sessionCode] = session
	go session.run()

	return session
}

func (s *Session) getNextColor() string {
	s.colorMutex.Lock()
	defer s.colorMutex.Unlock()
	color := colors[s.colorIndex%len(colors)]
	s.colorIndex++
	return color
}

func (s *Session) scheduleSave() {
	if s.saveTimer != nil {
		s.saveTimer.Stop()
	}

	s.saveTimer = time.AfterFunc(5*time.Second, func() {
		s.mutex.RLock()
		content := s.document
		s.mutex.RUnlock()

		if err := s.db.SaveDocument(s.sessionCode, content, nil); err != nil {
			log.Printf("Failed to save document: %v", err)
		} else {
			log.Printf("Document saved for session %s", s.sessionCode)
		}
	})
}

func (s *Session) run() {
	for {
		select {
		case c := <-s.register:
			s.clients[c] = true

			// Send current document state to new client
			select {
			case c.Send <- message.Message{
				Type:    "init",
				Content: s.document,
				UserID:  c.UserID,
				Color:   c.Color,
			}:
			default:
			}

			// Send existing users to new client
			for existingClient := range s.clients {
				if existingClient != c {
					select {
					case c.Send <- message.Message{
						Type:   "userJoined",
						UserID: existingClient.UserID,
						Color:  existingClient.Color,
					}:
					default:
					}
				}
			}

			// Notify others about new user
			for existingClient := range s.clients {
				if existingClient != c {
					select {
					case existingClient.Send <- message.Message{
						Type:   "userJoined",
						UserID: c.UserID,
						Color:  c.Color,
					}:
					default:
					}
				}
			}

		case c := <-s.unregister:
			if _, ok := s.clients[c]; ok {
				delete(s.clients, c)
				close(c.Send)

				// Notify others about user leaving
				for existingClient := range s.clients {
					select {
					case existingClient.Send <- message.Message{
						Type:   "userLeft",
						UserID: c.UserID,
					}:
					default:
					}
				}
			}

		case msg := <-s.broadcast:
			if msg.Type == "update" {
				s.mutex.Lock()
				s.document = msg.Content
				s.mutex.Unlock()

				// Schedule save after document update
				s.scheduleSave()
			}

			for c := range s.clients {
				select {
				case c.Send <- msg:
				default:
					close(c.Send)
					delete(s.clients, c)
				}
			}
		}
	}
}

func (s *Session) Register(c *client.Client) {
	s.register <- c
}

func (s *Session) Unregister(c *client.Client) {
	s.unregister <- c
}

func (s *Session) Broadcast(msg message.Message) {
	s.broadcast <- msg
}

func ServeWS(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	userID := r.URL.Query().Get("userId")
	if userID == "" {
		userID = "user" + string(rune(len(hub.sessions)+1))
	}

	sessionCode := r.URL.Query().Get("session")
	if sessionCode == "" {
		log.Println("No session code provided")
		conn.Close()
		return
	}

	session := hub.GetOrCreateSession(sessionCode)
	c := client.New(session, conn, userID, session.getNextColor())
	session.Register(c)

	go c.WritePump()
	go c.ReadPump()
}
