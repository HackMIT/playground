package world

import (
	"encoding/json"

	"github.com/techx/playground/models"

	"github.com/google/uuid"
)


// World maintains the characters present in the room
type World struct {
	characters map[uuid.UUID]*models.Character
}

func NewWorld() *World {
	return &World{
		characters: make(map[uuid.UUID]*models.Character),
	}
}

func removeCharacter(w *World, id uuid.UUID) {
	delete(w.characters, id)
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
		w.characters[m.sender.id] = models.NewCharacter(m.sender.id, res.Name)

		res.Id = m.sender.id.String()
		raw, _ := json.Marshal(res)
		m.msg = raw
	case "move":
		res := MovePacket{}

		if err := json.Unmarshal(m.msg, &res); err != nil {
			panic(err)
		}

		// Update this character's position
		w.characters[m.sender.id].X = res.X
		w.characters[m.sender.id].Y = res.Y

		res.Id = m.sender.id.String()
		raw, _ := json.Marshal(res)
		m.msg = raw
	}
}
