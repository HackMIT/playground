package packet

import (
	"encoding/json"
	"strings"

	"github.com/techx/playground/db"
	"github.com/techx/playground/db/models"
	"github.com/techx/playground/utils"
)

// Sent by clients after receiving the init packet. Identifies them to the
// server, and in turn other clients
type JoinPacket struct {
	BasePacket
	Packet `json:",omitempty"`

	// Client attributes
	Name       string `json:"name,omitempty"`
	QuillToken string `json:"quillToken,omitempty"`
	Token      string `json:"token,omitempty"`

	Email string `json:"email,omitempty"`
	Code  int    `json:"code,omitempty"`

	// Server attributes
	Character *models.Character `json:"character"`
	ClientID  string            `json:"clientId,omitempty"`
	Project   *models.Project   `json:"project"`
	Room      string            `json:"room"`
}

func NewJoinPacket(character *models.Character, room string) *JoinPacket {
	p := new(JoinPacket)
	p.BasePacket = BasePacket{Type: "join"}
	p.Character = character
	p.Character.Email = ""

	if strings.HasPrefix(p.Room, "arena:") {
		p.SetProject()
	}

	return p
}

func (p *JoinPacket) SetProject() {
	projectID, err := db.GetInstance().Get("character:" + p.Character.ID + ":project").Result()

	if err != nil || len(projectID) == 0 {
		return
	}

	p.Project = new(models.Project)
	projectRes, _ := db.GetInstance().HGetAll("project:" + projectID).Result()
	utils.Bind(projectRes, p.Project)
}

func (p JoinPacket) PermissionCheck(characterID string, role models.Role) bool {
	return true
}

func (p JoinPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p JoinPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
