package model

import "time"

// customer data type
type Customer struct {
	Id          string    `json:"id" db:"id"`
	Email       string    `json:"email" db:"email"`
	Password    string    `json:"password" db:"password"`
	Username    string    `json:"username" db:"username"`
	Phone       string    `json:"phone" db:"phone"`
	DateCreated time.Time `json:"date_created" db:"date_created"`
	DateUpdated time.Time `json:"date_updated" db:"date_updated"`
}
