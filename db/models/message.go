package models

type Message struct {
    From string `json:"from"`
    Text string `json:"text"`
    To string `json:"to"`
}
