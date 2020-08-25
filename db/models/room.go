package models

import (
	"encoding/json"

	"github.com/techx/playground/utils"

	"github.com/go-redis/redis/v7"
	"github.com/google/uuid"
)

type Room struct {
	Characters map[string]*Character `json:"characters" redis:"-"`
	Elements   []*Element            `json:"elements" redis:"-"`
	Hallways   map[string]*Hallway   `json:"hallways" redis:"-"`

	Background string `json:"background" redis:"background"`
	ID         string `json:"id" redis:"id"`
	Sponsor    bool   `json:"sponsor" redis:"sponsor"`
}

func NewRoom(id, background string, sponsor bool) *Room {
	return &Room{
		Characters: map[string]*Character{},
		Elements:   []*Element{},
		Hallways:   map[string]*Hallway{},
		Background: background,
		ID:         id,
		Sponsor:    sponsor,
	}
}

func (r *Room) Init() *Room {
	r.Characters = map[string]*Character{}
	r.Elements = []*Element{}
	r.Hallways = map[string]*Hallway{}
	return r
}

func (r Room) MarshalBinary() ([]byte, error) {
	return json.Marshal(r)
}

func (r Room) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, r)
}

func CreateHomeRoom(pip redis.Pipeliner, characterID string) {
	room := &Room{
		Background: "personal_room.svg",
		ID:         "home:" + characterID,
	}

	pip.HSet("room:home:"+characterID, utils.StructToMap(room))
	pip.SAdd("rooms", "home:"+characterID)

	bedElemID := uuid.New().String()
	bedElem := &Element{
		X:     0.341,
		Y:     0.65,
		Width: 0.2514,
		Path:  "bed.svg",
	}

	deskElemID := uuid.New().String()
	deskElem := &Element{
		X:     0.7089,
		Y:     0.5792,
		Width: 0.1709,
		Path:  "desk.svg",
	}

	plantElemID := uuid.New().String()
	plantElem := &Element{
		X:     0.3979,
		Y:     0.4486,
		Width: 0.0259,
		Path:  "plant.svg",
	}

	chairElemID := uuid.New().String()
	chairElem := &Element{
		X:     0.6821,
		Y:     0.6696,
		Width: 0.0559,
		Path:  "chair.svg",
	}

	shelfElemID := uuid.New().String()
	shelfElem := &Element{
		X:     0.5183,
		Y:     0.9012,
		Width: 0.0907,
		Path:  "shelf.svg",
	}

	pip.HSet("element:"+bedElemID, utils.StructToMap(bedElem))
	pip.HSet("element:"+deskElemID, utils.StructToMap(deskElem))
	pip.HSet("element:"+plantElemID, utils.StructToMap(plantElem))
	pip.HSet("element:"+chairElemID, utils.StructToMap(chairElem))
	pip.HSet("element:"+shelfElemID, utils.StructToMap(shelfElem))

	pip.RPush("room:home:"+characterID+":elements", bedElemID)
	pip.RPush("room:home:"+characterID+":elements", deskElemID)
	pip.RPush("room:home:"+characterID+":elements", plantElemID)
	pip.RPush("room:home:"+characterID+":elements", chairElemID)
	pip.RPush("room:home:"+characterID+":elements", shelfElemID)
}
