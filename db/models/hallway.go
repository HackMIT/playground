package models

type Hallway struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Radius float64 `json:"radius"`
	To string `json:"to"`
}
