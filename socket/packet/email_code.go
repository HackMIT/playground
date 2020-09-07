package packet

import (
	"encoding/json"
)

// Sent by clients when they need a login code
type EmailCodePacket struct {
	BasePacket

	// The email address to check and send the code to
	Email string `json:"email"`
}

func (p EmailCodePacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p EmailCodePacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
