package models

type Hallway struct {
	X      float64 `json:"x" redis:"x"`
	Y      float64 `json:"y" redis:"y"`
	ToX    float64 `json:"toX" redis:"toX"`
	ToY    float64 `json:"toY" redis:"toY"`
	Radius float64 `json:"radius" redis:"radius"`
	To     string  `json:"to" redis:"to"`
}
