package model

import "time"

// user data type
// originally customer data type
type User struct {
	UserId      string    `json:"user_id" db:"user_id"`
	UserName    string    `json:"user_name" db:"user_name"`
	Password    string    `json:"password" db:"password"`
	Email       string    `json:"email" db:"email"`
	DateCreated time.Time `json:"date_created" db:"date_created"`
	DateUpdated time.Time `json:"date_updated" db:"date_updated"`
	Phone       string    `json:"phone" db:"phone"`
	Photo       bool      `json:"photo" db:"photo"`
	FirstName   string    `json:"first_name" db:"first_name"`
	LastName    string    `json:"last_name" db:"last_name"`
}

type FollowRequest struct {
	UserId      string    `json:"user_id" db:"user_id"`
	UserName    string    `json:"user_name" db:"user_name"`
	Password    string    `json:"password" db:"password"`
	Email       string    `json:"email" db:"email"`
	DateCreated time.Time `json:"date_created" db:"date_created"`
	DateUpdated time.Time `json:"date_updated" db:"date_updated"`
	Phone       string    `json:"phone" db:"phone"`
	Photo       bool      `json:"photo" db:"photo"`
	FirstName   string    `json:"first_name" db:"first_name"`
	LastName    string    `json:"last_name" db:"last_name"`
}
