package main

import (
	_ "encoding/json"
	"errors"
	"fmt"
	_ "io/ioutil"
	"net/http"
	"os"
	_ "time"

	"log"
	"vibe/api"
	"vibe/api/subscriber"
	"vibe/api/twilio"
	"vibe/api/user"
	"vibe/api/video"
	"vibe/auth"
	"vibe/config"
	"vibe/store"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
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

func Test(w http.ResponseWriter, r *http.Request) {
	obj1 := Data{"TEST", 1, 2, 3}
	var res Res
	res.Data = append(res.Data, obj1)
	api.Respond(w, res, http.StatusOK)
}

// Requests
func handleAuthRequests(r *mux.Router) {
	r.HandleFunc("/isauth", auth.IsAuthenticated).Methods("GET")
	r.HandleFunc("/signup", auth.Signup).Methods("POST")
	r.HandleFunc("/update-password", auth.UpdatePassword).Methods("POST")
	r.HandleFunc("/login", auth.Signin).Methods("POST")
	r.HandleFunc("/verify-phone-num", twilio.VerifyPhoneNumber).Methods("POST")
	r.HandleFunc("/pass-rec-verify-phone-num", twilio.PasswordRecoveryVerifyPhoneNumber).Methods("POST")
	r.HandleFunc("/verify-phone-code", twilio.VerifyCode).Methods("POST")
	r.HandleFunc("/username-check", user.UsernameAvailablityCheck).Methods("POST")
	r.Handle("/signout", auth.RequireAuth(auth.RemoveSession)).Methods("POST")
	r.HandleFunc("/test-no-auth", Test).Methods("GET")
	r.Handle("/test-auth", auth.RequireAuth(Test)).Methods("GET")
	// possibly should go through RequireAuth route
	r.HandleFunc("/user-info", user.GetUserInfo).Methods("GET")
	r.HandleFunc("/chunk-upload", video.ChunkUploadHandler).Methods("POST")
	r.HandleFunc("/videos/{latitude}/{longitude}", video.GetLatestVideo).Methods("GET")
	r.HandleFunc("/set-delete-status", user.SetDeleteStatus).Methods("POST")
	r.HandleFunc("/set-user-following", user.SetUserFollowing).Methods("POST")
	r.HandleFunc("/set-user-unfollowing", user.SetUserUnfollowing).Methods("POST")
	r.HandleFunc("/get-following-data", user.GetFollowingData).Methods("POST")
	r.HandleFunc("/get-follower-data", user.GetFollowerData).Methods("POST")
	r.HandleFunc("/get-follower-following-count", user.GetFollowingAndFollowerCount).Methods("POST")
	r.HandleFunc("/subscribe", subscriber.Subscribe).Methods("POST")
}

func main() {

	// Load Environemnt Variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal("error loading .env file")
	}

	// overwrite APP_ENV with environment variable if it exists, otherwise it should defualt to "test"
	APP_ENV := os.Getenv("APP_ENV")
	config.InitConfig(APP_ENV)
	config.PrintConfig()

	// if we get here and APP_ENV is still an empty string, kill program.
	if APP_ENV == "" {
		log.Fatal("APP_ENV not found!")
	}

	// Build log file
	file, err := os.OpenFile("log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err.Error())
	}
	log.SetOutput(file)
	log.Println("STARTING LOG")
	log.Println("APP_ENV: " + APP_ENV)

	// Requests
	r := mux.NewRouter()
	handleAuthRequests(r)
	log.Println("past handleAuthRequests")

	// Initialize DB Connection
	store.InitDB()
	log.Println("past InitDB")

	// Initialize Cache Connection
	store.InitCache()
	log.Println("past InitCache")

	var VIBE_PORT = config.CONFIGURATION.VIBE_PORT
	fmt.Printf("Starting server on %v\n", VIBE_PORT)

	// Serve
	if APP_ENV == "prod" {
		err = http.ListenAndServeTLS(VIBE_PORT, "/certs/fullchain.pem", "/certs/key.pem", r) // Blocking function
	} else {
		log.Println("in listen and serve")
		err = http.ListenAndServe(VIBE_PORT, r) // Blocking function
	}

	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}

func GetConfig() {
	panic("unimplemented")
}

// addHeaders will act as middleware to give us CORS support
func addHeaders(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		h.ServeHTTP(w, r)
	}
}
