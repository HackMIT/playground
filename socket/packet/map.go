package packet

import (
	"encoding/json"

	"github.com/go-redis/redis/v7"
	"github.com/techx/playground/db"
	"github.com/techx/playground/db/models"
	"github.com/techx/playground/utils"
)

type MapPacket struct {
	BasePacket
	Packet `json:",omitempty"`

	Locations []*models.Location `json:"locations"`
}

func NewMapPacket() *MapPacket {
	// Load locations from Redis
	locationIDs, _ := db.GetInstance().SMembers("locations").Result()

	pip := db.GetInstance().Pipeline()
	locationCmds := make([]*redis.StringStringMapCmd, len(locationIDs))

	for i, locationID := range locationIDs {
		locationCmds[i] = pip.HGetAll("location:" + locationID)
	}

	pip.Exec()
	locations := make([]*models.Location, len(locationCmds))

	for i, locationCmd := range locationCmds {
		locationRes, _ := locationCmd.Result()
		locations[i] = new(models.Location)
		utils.Bind(locationRes, locations[i])
	}

	// Send locations back to client
	return &MapPacket{
		BasePacket: BasePacket{
			Type: "map",
		},
		Locations: locations,
	}
}

func (p MapPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p MapPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
