package models

type Message struct {
	From string `json:"from" redis:"from"`
	Text string `json:"text" redis:"text"`
	To   string `json:"to" redis:"to"`
}
