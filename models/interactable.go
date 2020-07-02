package models

type Interactable struct {
	Action     string  `json:"action"`
	Appearance string  `json:"appearance"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
}

func (i *Interactable) Init(action string, appearance string, x float64, y float64) *Interactable {
	i.Action = action
	i.Appearance = appearance
	i.X = x
	i.Y = y
	return i
}
