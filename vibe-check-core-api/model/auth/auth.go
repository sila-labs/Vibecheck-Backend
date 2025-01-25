package model

import mAPI "vibe/model/api"

type Auth struct {
	IsAuth bool      `json:"is_auth"`
	User   mAPI.User `json:"user"`
}

type Session struct {
	Id string `json:"session_id"`
}
