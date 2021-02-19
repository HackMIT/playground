package db

import (
	"encoding/json"
	"io/ioutil"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/techx/playground/config"
)

// RoomType is an enum representing all possible room templates
type RoomType string

const (
	// Home is the room that everyone spawns in, otherwise known as town square
	Home RoomType = "home"

	// Plaza is the room where you can get to the coffee shop, arcade, and stadium
	Plaza = "plaza"

	// Nightclub is the club, accessible from town square
	Nightclub = "nightclub"

	// CoffeeShop is the coffee shop, accessible from plaza
	CoffeeShop = "coffee_shop"

	// Nonprofits is the campground with all of the nonprofit tents
	Nonprofits = "nonprofits"

	// Personal is a template for someone's personal room
	Personal = "personal"

	// PlatArea is the area accessible from town square with the two plat sponsor buildings
	PlatArea = "plat_area"

	// LeftField is the left sponsor area
	LeftField = "left_field"

	// RightField is the right sponsor area
	RightField = "right_field"

	// Plat is a plat-tier sponsor's room
	Plat = "plat"

	// Gold is a gold-tier sponsor's room
	Gold = "gold"

	// Silver is a silver-tier sponsor's room
	Silver = "silver"

	// Bronze is a bronze-tier sponsor's room
	Bronze = "bronze"

	// Arena is the hacking arena, accessible from town square
	Arena = "arena"

	// Mall is the clothing store, accessible from town square
	Mall = "mall"

	// MISTI is the room for MISTI, accessible from the plaza
	MISTI = "misti"

	// Auditorium is the room for the stadium, accessible from the plaza
	Auditorium = "auditorium"
)

// CreateRoom builds a room with the given ID from a template file
func createRoomWithData(id string, roomType RoomType, data map[string]interface{}) {
	dat, err := ioutil.ReadFile("config/rooms/" + string(roomType) + ".json")

	if err != nil {
		return
	}

	var roomData map[string]interface{}
	json.Unmarshal(dat, &roomData)
	data["background"] = roomData["background"]

	if val, ok := roomData["corners"]; ok {
		data["corners"] = val
	}

	if sponsorID, ok := data["id"].(string); ok {
		data["background"] = strings.ReplaceAll(data["background"].(string), "<id>", sponsorID)

		if val, ok := roomData["sponsor"].(bool); ok && val {
			data["sponsorId"] = sponsorID
		}
	}

	instance.HSet("room:"+id, data)

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
			elementData["changingRandomly"] = true
		}

		if _, ok := elementData["campfire"]; ok {
			// If this is a campfire, animate it
			delete(elementData, "campfire")
			elementData["width"] = 0.0253
			elementData["path"] = "campfire/campfire1.svg"
			elementData["changingImagePath"] = true
			elementData["changingPaths"] = "campfire/campfire1.svg,campfire/campfire2.svg,campfire/campfire3.svg,campfire/campfire4.svg,campfire/campfire5.svg"
			elementData["changingInterval"] = 250
			elementData["changingRandomly"] = false
		}

		if _, ok := elementData["fountain"]; ok {
			// If this is a fountain, animate it
			delete(elementData, "fountain")
			elementData["path"] = "fountain1.svg"
			elementData["changingImagePath"] = true
			elementData["changingPaths"] = "fountain1.svg,fountain2.svg,fountain3.svg"
			elementData["changingInterval"] = 1000
			elementData["changingRandomly"] = false
		}

		if _, ok := elementData["toggleable"]; ok {
			switch elementData["path"] {
			case "street_lamp.svg":
				elementData["path"] = "street_lamp.svg,street_lamp_off.svg"
			case "bar_closed.svg":
				elementData["path"] = "bar_closed.svg,bar_open.svg"
			case "flashlight_off.svg":
				elementData["path"] = "flashlight_off.svg,flashlight_on.svg"
			default:
				break
			}

			elementData["state"] = 0
		}

		if id, ok := data["id"].(string); ok {
			elementData["path"] = strings.ReplaceAll(elementData["path"].(string), "<id>", id)
		}

		instance.HSet("element:"+elementID, elementData)
		instance.RPush("room:"+id+":elements", elementID)
	}

	for _, val := range roomData["hallways"].([]interface{}) {
		hallwayData := val.(map[string]interface{})

		if roomType == Bronze || roomType == Silver || roomType == Gold || roomType == Plat {
			hallwayData["toX"] = data["toX"].(float64)
			hallwayData["toY"] = data["toY"].(float64)

			if val, ok := data["to"].(string); ok {
				hallwayData["to"] = val
			}
		}

		hallwayID := uuid.New().String()
		instance.HSet("hallway:"+hallwayID, val)
		instance.SAdd("room:"+id+":hallways", hallwayID)
	}

	instance.SAdd("rooms", id)
}

func createSponsors() {
	dat, err := ioutil.ReadFile("config/sponsors.json")

	if err != nil {
		return
	}

	var sponsorsData []map[string]interface{}
	json.Unmarshal(dat, &sponsorsData)

	for _, sponsor := range sponsorsData {
		sponsorID := sponsor["id"].(string)
		delete(sponsor, "id")

		instance.HSet("sponsor:"+sponsorID, sponsor)
		instance.SAdd("sponsors", sponsorID)
	}
}

func CreateRoom(id string, roomType RoomType) {
	createRoomWithData(id, roomType, map[string]interface{}{})
}

func createEvents() {
	dat, err := ioutil.ReadFile("config/events.json")

	if err != nil {
		return
	}

	var eventsData []map[string]interface{}
	json.Unmarshal(dat, &eventsData)

	for _, event := range eventsData {
		startTime, err := time.Parse("2006-01-02T15:04:05-0700", event["start_time"].(string))

		if err != nil {
			panic(err)
		}

		event["startTime"] = int(startTime.Unix())

		eventID := uuid.New().String()[:4]
		instance.HSet("event:"+eventID, event)
		instance.SAdd("events", eventID)
	}
}

func reset() {
	instance.FlushDB()
	CreateRoom("home", Home)
	CreateRoom("nightclub", Nightclub)
	CreateRoom("nonprofits", Nonprofits)
	CreateRoom("plat_area", PlatArea)
	CreateRoom("left_field", LeftField)
	CreateRoom("right_field", RightField)
	CreateRoom("plaza", Plaza)
	CreateRoom("coffee_shop", CoffeeShop)
	CreateRoom("mall", Mall)
	CreateRoom("auditorium", Auditorium)

	createRoomWithData("arena:connectivity", Arena, map[string]interface{}{
		"id": "connectivity",
	})

	createRoomWithData("arena:education", Arena, map[string]interface{}{
		"id": "education",
	})

	createRoomWithData("arena:health", Arena, map[string]interface{}{
		"id": "health",
	})

	createRoomWithData("arena:urban", Arena, map[string]interface{}{
		"id": "urban",
	})

	createRoomWithData("sponsor:beaverworks", Gold, map[string]interface{}{
		"id":  "beaverworks",
		"to": "left_field",
		"toX": 0.3042,
		"toY": 0.6834,
	})

	createRoomWithData("sponsor:ieee", Gold, map[string]interface{}{
		"id":  "ieee",
		"to": "left_field",
		"toX": 0.1137,
		"toY": 0.4796,
	})

	createRoomWithData("sponsor:kodewithklossy", Gold, map[string]interface{}{
		"id":  "kodewithklossy",
		"to":  "left_field",
		"toX": 0.6958,
		"toY": 0.6834,
	})

	createRoomWithData("sponsor:ktbyte", Gold, map[string]interface{}{
		"id":  "ktbyte",
		"to":  "left_field",
		"toX": 0.8969,
		"toY": 0.4657,
	})

	createRoomWithData("sponsor:leah", Gold, map[string]interface{}{
		"id":  "leah",
		"to":  "right_field",
		"toX": 0.1,
		"toY": 0.4598,
	})

	createRoomWithData("sponsor:lsa", Gold, map[string]interface{}{
		"id":  "lsa",
		"to":  "right_field",
		"toX": 0.3108,
		"toY": 0.6903,
	})

	createRoomWithData("sponsor:medscience", Gold, map[string]interface{}{
		"id":  "medscience",
		"to":  "right_field",
		"toX": 0.7037,
		"toY": 0.6647,
	})

	createRoomWithData("sponsor:lincoln", Gold, map[string]interface{}{
		"id":  "lincoln",
		"to":  "right_field",
		"toX": 0.9152,
		"toY": 0.4563,
	})

	createRoomWithData("sponsor:misti", MISTI, map[string]interface{}{
		"id": "misti",
	})

	createEvents()
	createSponsors()

	if len(config.GetSecret("EMAIL")) > 0 {
		instance.SAdd("organizer_emails", config.GetSecret("EMAIL"))
	}
}
