package models

type Location struct {
	Lat  float64 `json:"lat" redis:"lat"`
	Lng  float64 `json:"lng" redis:"lng"`
	Name string  `json:"name" redis:"name"`
}
