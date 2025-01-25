package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
)

func Respond(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	//w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
	//w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	//w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	//w.Header().Set("Access-Control-Allow-Credentials", "true")

	// 200 is implicitly called - will cause "superfluous response.WriteHeader call error"
	if statusCode != 200 {
		w.WriteHeader(statusCode)
	} else {
		fmt.Println("Use RespondOK() function instead for status http.StatusOK")
	}

	log.Info("In response -------------------------")
	res, err := json.Marshal(data)
	if err != nil {
		log.Error("Unable to encode JSON: ", err)
		w.WriteHeader(http.StatusBadRequest)
	}
	w.Write(res)
}

// Used for http.StatusOK write responses as it is default
func RespondOK(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")

	log.Info("In response -------------------------")
	res, err := json.Marshal(data)
	if err != nil {
		log.Error("Unable to encode JSON: ", err)
		w.WriteHeader(http.StatusBadRequest)
	}
	w.Write(res)
}

func RespondRaw(w http.ResponseWriter, data []byte, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	//w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
	//w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	//w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	//w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.WriteHeader(statusCode)
	log.Info("In response -------------------------")
	w.Write(data)
}
