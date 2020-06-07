package packet

// The base packet that can be sent between clients and server. These are all
// of the required attributes of any packet
type BasePacket struct {
	// Identifies the type of packet
	Type string `json:"type"`
}
