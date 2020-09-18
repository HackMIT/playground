package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

// Sent by clients when the window gains or loses focus
type StatusPacket struct {
	BasePacket
	Packet `json:",omitempty"`

	// True if the user is online and has the tab open -- false if the window doesn't have focus
	Active bool `json:"active"`

	// True if the user has the tab open, regardless of focus
	Online bool `json:"online"`

	// The ID of the character who this is a status update for
	ID string `json:"id"`

	// Server attributes
	FriendIDs   []string `json:"friendIds"`
	TeammateIDs []string `json:"teammateIds"`
}

func NewStatusPacket(characterID string, online bool) *StatusPacket {
	return &StatusPacket{
		BasePacket: BasePacket{
			Type: "status",
		},
		ID:     characterID,
		Active: online,
		Online: online,
	}
}

func (p StatusPacket) PermissionCheck(characterID string, role models.Role) bool {
	return len(characterID) > 0
}

func (p StatusPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p StatusPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
