package model

import "time"

// customer data type
type Video struct {
	Id          string    `json:"id" db:"id"`
	Latitude    string    `json:"latitude" db:"latitude"`
	Longitude   string    `json:"longitude" db:"longitude"`
	User_Id     string    `json:"user_id" db:"user_id"`
	DateCreated time.Time `json:"date_created" db:"date_created"`
	Vibe_Points string    `json:"vibe_points" db:"vibe_points"`
}
