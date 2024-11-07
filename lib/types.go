package lib

import "time"

type NexusPool struct {
	Id    string `json:"id"`
	Alive bool   `json:"alive"`
	Url   string `json:"url"`
}

type ArenaSession struct {
	ConnexionExpirationDate time.Time `json:"connexion_expiration_date"`
	UserID                  string    `json:"user_id"`
	Alive                   bool      `json:"alive"`
	Started                 bool      `json:"started"`
	UserIP                  string    `json:"user_ip"`
}
