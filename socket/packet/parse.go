package packet

import (
	"encoding/json"
	"errors"
)

func ParsePacket(data []byte) (Packet, error) {
	res := BasePacket{}

	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}

	switch res.Type {
	case "add_email":
		p := AddEmailPacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "auth", "join":
		p := JoinPacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "chat":
		p := ChatPacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "dance":
		p := DancePacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "element_toggle":
		p := ElementTogglePacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "element_update":
		p := ElementUpdatePacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "email_code":
		p := EmailCodePacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "event":
		p := EventPacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "friend_request":
		p := FriendRequestPacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "friend_update":
		p := FriendUpdatePacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "get_achievements":
		p := GetAchievementsPacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "get_map":
		p := GetMapPacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "get_messages":
		p := GetMessagesPacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "get_current_song":
		p := GetCurrentSongPacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "get_songs":
		p := GetSongsPacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "get_sponsor":
		p := GetSponsorPacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "hallway_add":
		p := HallwayAddPacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "hallway_delete":
		p := HallwayDeletePacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "hallway_update":
		p := HallwayUpdatePacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "jukebox_warning":
		p := JukeboxWarningPacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "leave":
		p := LeavePacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "message":
		p := MessagePacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "move":
		p := MovePacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "play_song":
		p := PlaySongPacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "project_form":
		p := ProjectFormPacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "queue_join":
		p := QueueJoinPacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "queue_remove":
		p := QueueRemovePacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "queue_subscribe":
		p := QueueSubscribePacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "queue_unsubscribe":
		p := QueueUnsubscribePacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "queue_update_hacker":
		p := QueueUpdateHackerPacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "queue_update_sponsor":
		p := QueueUpdateSponsorPacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "register":
		p := RegisterPacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "report":
		p := ReportPacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "room_add":
		p := RoomAddPacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "settings":
		p := SettingsPacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "song":
		p := SongPacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "status":
		p := StatusPacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "teleport", "teleport_home":
		p := TeleportPacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "update_map":
		p := UpdateMapPacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "update_sponsor":
		p := UpdateSponsorPacket{}
		json.Unmarshal(data, &p)
		return p, nil
	case "wardrobe_change":
		p := WardrobeChangePacket{}
		json.Unmarshal(data, &p)
		return p, nil
	default:
		return nil, errors.New("Invalid packet type: " + res.Type)
	}
}
