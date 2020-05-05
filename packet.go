package main

import (
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
	Type string `json:"type"`

	// Map of characters that are already in the room
	Characters map[uuid.UUID]*Character `json:"characters"`
}

// Sent by clients after receiving the init packet. Identifies them to the
// server, and in turn other clients
type JoinPacket struct {
	Type string `json:"type"`

	// The id of the client who's joining
	Id string `json:"id"`

	// The client's username
	Name string `json:"name"`
}

// Sent by clients when they move around
type MovePacket struct {
	Type string `json:"type"`

	// The id of the  client who is moving
	Id string `json:"id"`

	// The client's x position (0-1)
	X float32 `json:"x"`

	// The client's y position (0-1)
	Y float32 `json:"y"`
}

type LeavePacket struct {
	Type string `json:"type"`

	// The id of the client who's leaving
	Id string `json:"id"`
}
