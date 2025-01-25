package main

import (
	_ "encoding/json"
	"fmt"
	_ "io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	_ "time"

	notifications "vibe/api/notifications"
	test "vibe/api/test"
	"vibe/store"

	log "github.com/sirupsen/logrus"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/thedevsaddam/gojsonq"
)

var APP_ENV string
var METHOD_LOGGING string
var ENV_PROD = "prod"

// Requests
func handleAuthRequests(r *mux.Router) {
	r.HandleFunc("/send-notification-to-device", notifications.SendNotificationToDevice).Methods("POST")
	r.HandleFunc("/send-notification-to-topic", notifications.SendNotificationToTopic).Methods("POST")
	r.HandleFunc("/subscribe-devices-to-topic", notifications.SubscribeDevicesToTopic).Methods("POST")
	r.HandleFunc("/unsubscribe-devices-from-topic", notifications.UnsubscribeDevicesToChannel).Methods("POST")
}

func main() {
	fmt.Println("Starting template-api microservice...")
	fmt.Println("No logs will be generated here. Please see log.txt file for logging")

	// Build log output file
	os.Remove("log.txt") // remove old log
	file, err := os.OpenFile("log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Println("Cannot create log file: ", err)
	}
	log.SetOutput(file)

	// Load environment variables
	err = godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file, program will terminate: ", err)
	}

	// Get environment type, prod, dev, or local
	APP_ENV := os.Getenv("APP_ENV")
	if APP_ENV == "" {
		log.Fatal("APP_ENV not found, program will terminate.")
	}

	// Check if we should be logging methods along log messages
	methodLogging, ok := os.LookupEnv("METHOD_LOGGING")
	if !ok {

		log.Warning("METHOD_LOGGING not specified in .env, defaulting to false")
		methodLogging = "false"
	}

	if methodLogging == "true" {
		log.SetReportCaller(true)
	}

	// Trace, Debug, Info, Warn, Error, Fatal, and Panic (oridnal 6 - 0)
	logLevel, ok := os.LookupEnv("LOG_LEVEL")

	// LOG_LEVEL not set, let's default to info
	if !ok {
		logLevel = "info"
		log.Warning("LOG_LEVEL not specified in .env, defaulting to info")
	}

	// Parse string to log level
	parsedLevel, err := log.ParseLevel(logLevel)
	if err != nil {
		log.Error("Invalid log level, defaulting to debug: ", err)
		parsedLevel = log.DebugLevel
	}

	// Set global log level
	log.SetLevel(parsedLevel)

	log.Info("STARTING LOG...")
	log.Info("APP_ENV: " + APP_ENV)
	log.Info("LOG_LEVEL: " + logLevel)
	log.Info("METHOD_LOGGING: " + methodLogging)

	// Requests
	r := mux.NewRouter()
	handleAuthRequests(r)

	// Initialize DB Connection
	store.InitDB()

	APP_PORT := os.Getenv("APP_PORT")
	log.Info("Listening and serving on HTTP port", APP_PORT)
	notifications.Setup()

	// Adds waitgroup to wait for os signal or http server failure
	var wg sync.WaitGroup
	wg.Add(1)

	// Listen and serve
	// need to implement certs for security
	go SetupHttp(APP_PORT, r, &wg)

	// OS signal handler
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	go WaitForOSSignal(sig, &wg)

	wg.Wait() // Wait for all routines to finish to finish (Only happens if interrupted or exit or error)
	close(sig)
}

// addHeaders will act as middleware to give us CORS support
func addHeaders(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		h.ServeHTTP(w, r)
	}
}

// Setup http as a go routine
func SetupHttp(APP_PORT string, r *mux.Router, wg *sync.WaitGroup) {
	if APP_ENV == ENV_PROD {
		fmt.Println("Prod initialization complete")
		log.Error(http.ListenAndServe(APP_PORT, r))
		Cleanup()
		wg.Done()
	} else {
		fmt.Println("Dev initialization complete")
		log.Error(http.ListenAndServe(APP_PORT, r))
		Cleanup()
		wg.Done()
	}
}

// Waits for os signal as a go routine
func WaitForOSSignal(sig chan os.Signal, wg *sync.WaitGroup) {
	select {
	case <-sig:
		fmt.Println("Received os signal, shutting down")
		Cleanup()
		wg.Done()
	}
}

// Performs cleanup of service to make sure no leaks of resources
func Cleanup() {
	fmt.Println("Cleaning up!")
	test.Cleanup()
}
