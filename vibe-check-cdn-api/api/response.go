package api

import (
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
)

func Respond(w http.ResponseWriter, data interface{}, statusCode int) {
	log.Trace("entered Response")
	w.Header().Set("Content-Type", "application/json")

	// 200 is implicitly called - will cause "superfluous response.WriteHeader call error"
	if statusCode != 200 {
		w.WriteHeader(statusCode)
	} else {
		log.Warn("use RespondOK() function instead for status http.StatusOK")
	}

	res, err := json.Marshal(data)
	if err != nil {
		log.Info("unable to encode data to JSON")
		w.WriteHeader(http.StatusBadRequest)
	}
	log.Trace(string(res))
	w.Write(res)
}

// Used for http.StatusOK write responses as it is default
func RespondOK(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")

	log.Trace("entered ResponseOK")
	res, err := json.Marshal(data)
	if err != nil {
		log.Info("unable to encode data to JSON")
		w.WriteHeader(http.StatusBadRequest)
	}
	log.Trace(string(res))
	w.Write(res)
}

func RespondRaw(w http.ResponseWriter, data []byte, statusCode int) {
	log.Trace("entered ResponseRaw")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(data)
}
