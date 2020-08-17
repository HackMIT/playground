package models

type Element struct {
    X float64 `json:"x" redis:"x"`
    Y float64 `json:"y" redis:"y"`
    Width float64 `json:"width" redis:"width"`
    Path string `json:"path" redis:"path"`
}
