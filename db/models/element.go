package models

type ElementAction int

const (
	OpenJukebox ElementAction = iota + 1
)

type Element struct {
	ID string `json:"id,omitempty"`

	X     float64 `json:"x" redis:"x"`
	Y     float64 `json:"y" redis:"y"`
	Width float64 `json:"width" redis:"width"`
	Path  string  `json:"path" redis:"path"`

	ChangingImagePath bool   `json:"changingImagePath" redis:"changingImagePath"`
	ChangingPaths     string `json:"changingPaths" redis:"changingPaths"`

	// How often to update the image, in milliseconds
	ChangingInterval int `json:"changingInterval" redis:"changingInterval"`

	ChangingRandomly bool `json:"changingRandomly" redis:"changingRandomly"`

	Action int `json:"action" redis:"action"`

	Hoverable  bool `json:"hoverable" redis:"hoverable"`
	Toggleable bool `json:"toggleable" redis:"toggleable"`
	State      int  `json:"state" redis:"state"`
}
