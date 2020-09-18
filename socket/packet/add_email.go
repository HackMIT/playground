package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

// Sent by clients when they need to add a valid email to our system
type AddEmailPacket struct {
	BasePacket
	Packet `json:",omitempty"`

	// The email address to check and send the code to
	Email string `json:"email"`

	// The role this user is signing up for (see models.Role)
	Role int `json:"role"`

	// The id of the sponsor, if Role == SponsorRep
	SponsorID string `json:"sponsorId"`
}

func (p AddEmailPacket) PermissionCheck(characterID string, role models.Role) bool {
	return len(characterID) > 0 && role == models.Organizer
}

func (p AddEmailPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p AddEmailPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
