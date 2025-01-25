package model

type Subscriber struct {
	Email string `json:"email" db:"email"`
}