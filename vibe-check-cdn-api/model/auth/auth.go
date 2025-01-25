package model

import mAPI "vibe/model/api"

type Auth struct {
	IsAuth   bool          `json:"is_auth"`
	Customer mAPI.Customer `json:"customer"`
}

type Session struct {
	Id string `json:"session_id"`
}
