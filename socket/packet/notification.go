package packet

import (
	"encoding/json"
)

type NotificationLevel int

const (
	// Low-level/info notification -- we'll only display the notification pop-up in Hack Penguin
	Low NotificationLevel = iota

	// Medium-level notification -- we'll also contact them through their preferred methods of notification
	Medium

	// Urgent notification -- we will reach out through every method we have available
	High
)

type NotificationType string

const (
	Achievement NotificationType = "achievement"
	Message                      = "message"
)

type NotificationPacket struct {
	BasePacket
	Packet `json:",omitempty"`

	NotificationType NotificationType       `json:"notificationType"`
	Data             map[string]interface{} `json:"data"`
	Level            NotificationLevel      `json:"level"`
}

func newNotificationPacket(notificationType NotificationType, level NotificationLevel, data map[string]interface{}) *NotificationPacket {
	return &NotificationPacket{
		BasePacket: BasePacket{
			Type: "notification",
		},
		NotificationType: notificationType,
		Data:             data,
		Level:            level,
	}
}

func NewAchievementNotificationPacket(id string) *NotificationPacket {
	return newNotificationPacket(Achievement, Low, map[string]interface{}{
		"id": id,
	})
}

func NewMessageNotificationPacket(text string) *NotificationPacket {
	return newNotificationPacket(Message, Low, map[string]interface{}{
		"text": text,
	})
}

func (p NotificationPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p NotificationPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
