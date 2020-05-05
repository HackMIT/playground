package main

import (
	"encoding/json"

	"github.com/google/uuid"
)

// Character is the digital representation of a client
type Character struct {
	id string `json:"id"`
	Name string `json:"name"`
	X float32 `json:"x"`
	Y float32 `json:"y"`
}

func newCharacter(id uuid.UUID, name string) *Character {
	return &Character{
		id: id.String(),
		Name: name,
		X: 0.5,
		Y: 0.5,
	}
}

// World maintains the characters present in the room
type World struct {
	characters map[uuid.UUID]*Character
}

func newWorld() *World {
	return &World{
		characters: make(map[uuid.UUID]*Character),
	}
}

func generateInitPacket(w *World) []byte {
	msg := &InitPacket{
		Type: "init",
		Characters: w.characters}
	raw, _ := json.Marshal(msg)
	return raw
}

func processMessage(w *World, m *SocketMessage) {
	res := BasePacket{}

	if err := json.Unmarshal(m.msg, &res); err != nil {
		// TODO: Better error handling
		panic(err)
	}

	switch res.Type {
	case "join":
		res := JoinPacket{}

		if err := json.Unmarshal(m.msg, &res); err != nil {
			panic(err)
		}

		// Save the character that just joined
		w.characters[m.sender.id] = newCharacter(m.sender.id, res.Name)
	case "move":
		res := MovePacket{}

		if err := json.Unmarshal(m.msg, &res); err != nil {
			panic(err)
		}

		// Update this character's position
		w.characters[m.sender.id].X = res.X
		w.characters[m.sender.id].Y = res.Y
	}
}
