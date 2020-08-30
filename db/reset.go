package db

import (
	"encoding/json"
	"io/ioutil"

	"github.com/google/uuid"
)

func CreateRoom(id, filename string) {
	dat, err := ioutil.ReadFile("config/rooms/" + filename + ".json")

	if err != nil {
		return
	}

	var roomData map[string]interface{}
	json.Unmarshal(dat, &roomData)

	instance.HSet("room:"+id, map[string]interface{}{
		"background": roomData["background"],
		"sponsor":    roomData["sponsor"],
	})

	elements := roomData["elements"].([]interface{})

	// If this is the nightclub, add floor tiles
	if id == "nightclub" {
		tileStartX := 0.374
		tileStartY := 0.552
		tileSeparator := 0.0305
		numTilesX := 7
		numTilesY := 4

		newTiles := make([]interface{}, numTilesX*numTilesY)

		for i := 0; i < numTilesY; i++ {
			for j := 0; j < numTilesX; j++ {
				newTiles[i*numTilesX+j] = map[string]interface{}{
					"x":    tileStartX + float64(i+j)*tileSeparator,
					"y":    tileStartY + float64((numTilesY-i)+j)*tileSeparator,
					"tile": true,
				}
			}
		}

		elements = append(newTiles, elements...)
	}

	for _, val := range elements {
		elementID := uuid.New().String()
		elementData := val.(map[string]interface{})

		if _, ok := elementData["tile"]; ok {
			// If this is a nightclub floor tile, autofill some attributes
			delete(elementData, "tile")
			elementData["width"] = 0.052
			elementData["path"] = "tiles/blue1.svg"
			elementData["changingImagePath"] = true
			elementData["changingPaths"] = "tiles/blue1.svg,tiles/blue2.svg,tiles/blue3.svg,tiles/blue4.svg,tiles/green1.svg,tiles/green2.svg,tiles/pink1.svg,tiles/pink2.svg,tiles/pink3.svg,tiles/pink4.svg,tiles/yellow1.svg"
			elementData["changingInterval"] = 2000
		}

		instance.HSet("element:"+elementID, val)
		instance.RPush("room:"+id+":elements", elementID)
	}

	for _, val := range roomData["hallways"].([]interface{}) {
		hallwayID := uuid.New().String()
		instance.HSet("hallway:"+hallwayID, val)
		instance.SAdd("room:"+id+":hallways", hallwayID)
	}

	instance.SAdd("rooms", id)
}

func reset() {
	instance.FlushDB()
	CreateRoom("home", "home")
	CreateRoom("nightclub", "nightclub")
	CreateRoom("nonprofits", "nonprofits")
	CreateRoom("plat_area", "plat_area")
}
