package packet

import (
	"encoding/json"

	"github.com/techx/playground/db"
	"github.com/techx/playground/db/models"
	"github.com/techx/playground/utils"
)

type SponsorPacket struct {
	BasePacket
	Packet `json:",omitempty"`

	Sponsor *models.Sponsor `json:"sponsor"`
}

func NewSponsorPacket(sponsorID string) *SponsorPacket {
	var sponsor models.Sponsor
	sponsorRes, _ := db.GetInstance().HGetAll("sponsor:" + sponsorID).Result()
	utils.Bind(sponsorRes, &sponsor)

	return &SponsorPacket{
		BasePacket: BasePacket{
			Type: "sponsor",
		},
		Sponsor: &sponsor,
	}
}

func (p SponsorPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p SponsorPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
