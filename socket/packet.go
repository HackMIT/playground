package socket

import (
	"encoding/json"

	"github.com/techx/playground/db"
	"github.com/techx/playground/models"

	"github.com/google/uuid"
)

// The base packet that can be sent between clients and server. These are all
// of the required attributes of any packet
type BasePacket struct {
	// Identifies the type of packet
	Type string `json:"type"`
}

// Sent by server to clients upon connecting. Contains information about the
// world that they load into
type InitPacket struct {
	BasePacket

	// Map of characters that are already in the room
	Characters map[uuid.UUID]*models.Character `json:"characters"`
}

func (p *InitPacket) Init(roomSlug string) *InitPacket {
	// Fetch characters from redis
	data, _ := db.GetRejsonHandler().JSONGet("room:" + roomSlug, "characters")

	var characters map[uuid.UUID]*models.Character
	json.Unmarshal(data.([]byte), &characters)

	// Set data and return
	p.BasePacket = BasePacket{Type: "init"}
	p.Characters = characters
	return p
}

func (p InitPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p InitPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}

// Sent by clients after receiving the init packet. Identifies them to the
// server, and in turn other clients
type JoinPacket struct {
	BasePacket

	// The id of the client who's joining
	Id string `json:"id"`

	// The client's username
	Name string `json:"name"`
}

func (p JoinPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p JoinPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}

// Sent by clients when they move around
type MovePacket struct {
	BasePacket

	// The id of the  client who is moving
	Id string `json:"id"`

	// The client's x position (0-1)
	X float32 `json:"x"`

	// The client's y position (0-1)
	Y float32 `json:"y"`
}

func (p MovePacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p MovePacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}

// Sent by ingests when a client disconnects
type LeavePacket struct {
	BasePacket

	// The id of the client who's leaving
	Id string `json:"id"`
}

func (p *LeavePacket) Init(id uuid.UUID) *LeavePacket {
	p.BasePacket = BasePacket{Type: "leave"}
	p.Id = id.String()
	return p
}
