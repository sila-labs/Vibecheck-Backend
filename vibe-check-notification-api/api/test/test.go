package test

import (
	"net/http"
	"time"

	"vibe/api"
	mAPI "vibe/model/api"

	log "github.com/sirupsen/logrus"
)

func GetTest(w http.ResponseWriter, r *http.Request) {
	log.Info("In test handler -------------------------")
	data := mAPI.Test{
		Id:          "TEST",
		DateCreated: time.Now(),
		Amount:      1,
		Usd:         2,
		Change:      3.0,
	}

	api.RespondOK(w, data)
}

func Cleanup() {}
