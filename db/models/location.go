package models

type Location struct {
    Lat float64 `json:"lat"`
    Lng float64 `json:"lng"`
    Name string `json:"name"`
}
