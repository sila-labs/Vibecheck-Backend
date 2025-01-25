package model

import "time"

// customer data type
type Customer struct {
	Id          string    `json:"id"`
	Email       string    `json:"email"`
	Username    string    `json:"username"`
	Phone       string    `json:"phone"`
	DateCreated time.Time `json:"date_created"`
	DateUpdated time.Time `json:"date_updated"`
}
