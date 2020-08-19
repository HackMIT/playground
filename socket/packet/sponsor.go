package packet

import (
	"encoding/json"
)

// Sent by clients when they move around
type SponsorPacket struct {
	BasePacket

	// The id of the client
	ID string `json:"id"`

	// The room that the client is in
	Room string `json:"room"`

	// New color to set
	Color float64 `json:"color"`

}

func (p SponsorPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p SponsorPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
