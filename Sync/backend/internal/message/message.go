package message

type Message struct {
	Type      string `json:"type"`
	Content   string `json:"content"`
	UserID    string `json:"userId"`
	CursorPos int    `json:"cursorPos,omitempty"`
	Color     string `json:"color,omitempty"`
}
