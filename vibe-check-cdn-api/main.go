package main

import (
	_ "encoding/json"
	_ "io/ioutil"
	"net/http"
	"os"
	_ "time"

	"vibe/api/video"
	"vibe/store"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"

	_ "github.com/thedevsaddam/gojsonq"
)

// Requests
func handleAuthRequests(r *mux.Router) {
	r.HandleFunc("/chunk-upload", video.ChunkUploadHandler).Methods("POST")
	r.HandleFunc("/user-pic-upload", video.UserPicUploadHandler).Methods("POST")
	r.HandleFunc("/getLocationLatestData", video.GetLocationLatestData).Methods("POST")
	r.HandleFunc("/data/user", video.GetDataByUser).Methods("POST")
	r.HandleFunc("/videos/location", video.GetVideosByLocation).Methods("POST")
	r.HandleFunc("/setFavoriteStatus/{locationName}/{lat}/{lon}/{user_name}/{liked_status}", video.SetFavoriteStatus).Methods("POST")
	r.HandleFunc("/getUserFavoriteLocationData/{user_name}", video.GetUserFavoriteLocationData).Methods("GET")
	r.HandleFunc("/chat-message-upload", video.ChatMessageUpload).Methods("POST")
	r.HandleFunc("/getLocationChat", video.GetLocationChat).Methods("POST")
	r.HandleFunc("/setVideoLikedStatus", video.SetVideoLikedStatus).Methods("POST")
	r.HandleFunc("/setIsVideoDeletedStatus", video.SetIsVideoDeletedStatus).Methods("POST")
	r.HandleFunc("/get-user-latest-data", video.GetUserLatestData).Methods("POST")

}

func main() {

	// Load environemnt variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal("error loading .env file, program will terminate.")
	}

	// Get environment type, prod, dev, or local
	APP_ENV := os.Getenv("APP_ENV")

	// If we get here and APP_ENV is still an empty string, kill program.
	if APP_ENV == "" {
		log.Fatal("APP_ENV not found, program will terminate.")
	}

	// Build log output file
	os.Remove("log.txt") // remove old log
	file, err := os.OpenFile("log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}
	log.SetOutput(file)

	// Trace, Debug, Info, Warn, Error, Fatal, and Panic (oridnal 6 - 0)
	logLevel, ok := os.LookupEnv("LOG_LEVEL")

	// LOG_LEVEL not set, let's default to info
	if !ok {
		logLevel = "info"
	}

	// Parse string to log level
	parsedLevel, err := log.ParseLevel(logLevel)
	if err != nil {
		parsedLevel = log.DebugLevel
	}
	// set global log level
	log.SetLevel(parsedLevel)

	log.Info("STARTING LOG...")
	log.Info("APP_ENV: " + APP_ENV)
	log.Info("LOG_LEVEL: " + logLevel)

	// Requests
	r := mux.NewRouter()
	handleAuthRequests(r)

	// Initialize DB Connection
	store.InitDB()

	// Initialize Cache Connection
	store.InitCache()

	APP_PORT := os.Getenv("APP_PORT")
	log.Info("listening and serving on HTTP port" + APP_PORT)

	// Listen and serve
	// need to implement certs for security
	if APP_ENV == "prod" {
		log.Fatal(http.ListenAndServe(APP_PORT, r))
		// log.Fatal(http.ListenAndServeTLS(":8082", "/certs/fullchain.pem", "/certs/key.pem", r))
	} else {
		log.Fatal(http.ListenAndServe(APP_PORT, r))
	}

}
