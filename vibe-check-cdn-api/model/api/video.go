package model

import "time"

// customer data type
type Video struct {
	Id          string    `json:"id"`
	Latitude    string    `json:"latitude"`
	Longitude   string    `json:"longitude"`
	User_Id     string    `json:"user_id"`
	DateCreated time.Time `json:"date_created"`
	Vibe_Points string    `json:"vibe_points"`
}
