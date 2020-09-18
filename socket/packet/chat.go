package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

type ChatPacket struct {
	BasePacket
	Packet `json:",omitempty"`

	// The message being sent
	Message string `json:"mssg"`

	// The id of the client who's joining
	ID string `json:"id"`

	// The client's room
	Room string `json:"room"`
}

func (p ChatPacket) PermissionCheck(characterID string, role models.Role) bool {
	return len(characterID) > 0
}

func (p ChatPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p ChatPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
