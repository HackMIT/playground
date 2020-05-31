package world

import (
	"encoding/json"

	"github.com/techx/playground/db"
	"github.com/techx/playground/models"
)


func processMessage(m *SocketMessage) {
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

		res.Id = m.sender.id.String()

		character := models.NewCharacter(m.sender.id, res.Name)

		_, err := db.Rh.JSONSet("rooms:home", "characters[\"" + m.sender.id.String() + "\"]", character)

		if err != nil {
			panic(err)
		}

		_, publishErr := db.Instance.Publish("room", res).Result()

		if publishErr != nil {
			panic(publishErr)
		}
	case "move":
		res := MovePacket{}

		if err := json.Unmarshal(m.msg, &res); err != nil {
			panic(err)
		}

		res.Id = m.sender.id.String()

		// TODO: go-rejson doesn't currently support transactions, but
		// these should really be done together
		db.Rh.JSONSet("rooms:home", "characters[\"" + m.sender.id.String() + "\"][\"x\"]", res.X)
		db.Rh.JSONSet("rooms:home", "characters[\"" + m.sender.id.String() + "\"][\"y\"]", res.Y)

		_, publishErr := db.Instance.Publish("room", res).Result()

		if publishErr != nil {
			panic(publishErr)
		}
	}
}
