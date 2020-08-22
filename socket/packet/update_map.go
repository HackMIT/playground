package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

type UpdateMapPacket struct {
	BasePacket
	*models.Location
}

func (p UpdateMapPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p UpdateMapPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
