package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

// Sent by clients when they dance
type DancePacket struct {
	BasePacket
	Packet `json:",omitempty"`

	// The id of the client who is dancing
	ID string `json:"id"`

	// The room that the client is dancing in
	Room string `json:"room"`

	// The client's new x position (0-1)
	Dance int `json:"dance"`
}

func (p DancePacket) PermissionCheck(characterID string, role models.Role) bool {
	return len(characterID) > 0
}

func (p DancePacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p DancePacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
