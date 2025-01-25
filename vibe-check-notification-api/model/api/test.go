package model

import "time"

type Test struct {
	Id          string    `json:"id"`
	DateCreated time.Time `json:"date_created"`
	Amount      int64     `json:"amount"`
	Usd         int64     `json:"usd"`
	Change      float64   `json:"change"`
}
