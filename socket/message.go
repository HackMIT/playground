package socket

import (
	"encoding/json"
)

// SocketMessage stores messages sent over WS with the client who sent it
type SocketMessage struct {
	msg    []byte
	sender *Client
}

func (m SocketMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}

func (m SocketMessage) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, m)
}
