package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

// Sent by clients when they move around
type MovePacket struct {
	BasePacket
	Packet `json:",omitempty"`

	// The id of the client who is moving
	ID string `json:"id"`

	// The room that the client is moving in
	Room string `json:"room"`

	// The client's new x position (0-1)
	X float64 `json:"x"`

	// The client's new y position (0-1)
	Y float64 `json:"y"`
}

func NewMovePacket(id, room string, x, y float64) *MovePacket {
	return &MovePacket{
		BasePacket: BasePacket{
			Type: "move",
		},
		ID:   id,
		Room: room,
		X:    x,
		Y:    y,
	}
}

func (p MovePacket) PermissionCheck(characterID string, role models.Role) bool {
	return len(characterID) > 0
}

func (p MovePacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p MovePacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
