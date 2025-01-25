package main

import (
	_ "encoding/json"
	_ "io/ioutil"
	"net/http"
	"os"
	_ "time"

	log "github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"

	_ "github.com/thedevsaddam/gojsonq"
)

var httpRouterGin *gin.Engine
var APP_ENV string

type Data struct {
	Id     string  `json:"id"`
	Amount int64   `json:"amount"`
	Usd    int64   `json:"usd"`
	Change float64 `json:"change"`
}

type Res struct {
	Data []Data `json:"data"`
}

func main() {

	// Load Environemnt Variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
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

	// --- HLS STREAMING SERVER SETUP ---
	UPLOADS_LOCATION := os.Getenv("UPLOADS_LOCATION")

	APP_PORT := os.Getenv("APP_PORT")
	log.Info("Serving ", UPLOADS_LOCATION, " on HTTP port ", APP_PORT)

	http.Handle("/", addHeaders(http.FileServer(http.Dir(UPLOADS_LOCATION))))
	log.Fatal(http.ListenAndServe(APP_PORT, nil))
	// --- HLS STREAMING SERVER SETUP ---

}

// --- HLS STREAMING SERVER SETUP ---
// addHeaders will act as middleware to give us CORS support
func addHeaders(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		h.ServeHTTP(w, r)
	}
}
