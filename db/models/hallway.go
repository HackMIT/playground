package models

type Hallway struct {
	X      float64 `json:"x" redis:"x"`
	Y      float64 `json:"y" redis:"y"`
	Radius float64 `json:"radius" redis:"radius"`
	To     string  `json:"to" redis:"to"`
}
