package client

import (
	"encoding/json"
	"log"
	"time"

	"collab-editor/internal/message"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

type Session interface {
	Register(client *Client)
	Unregister(client *Client)
	Broadcast(msg message.Message)
}

type Client struct {
	session Session
	conn    *websocket.Conn
	Send    chan message.Message
	UserID  string
	Color   string
}

func New(session Session, conn *websocket.Conn, userID, color string) *Client {
	return &Client{
		session: session,
		conn:    conn,
		Send:    make(chan message.Message, 256),
		UserID:  userID,
		Color:   color,
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.session.Unregister(c)
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		var msg message.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		msg.UserID = c.UserID
		msg.Color = c.Color
		c.session.Broadcast(msg)
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.Send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteJSON(msg); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
