package store

import (
	"database/sql"
	_ "encoding/json"
	"os"

	_ "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
)

var DB *sql.DB

// Initialize DB
func InitDB() {
	APP_ENV := os.Getenv("APP_ENV")
	DB_HOST := "127.0.0.1"
	DB_USER := "root"
	DB_PASS := "build"
	if APP_ENV == "prod" {
		DB_HOST = os.Getenv("MARIA_DB_HOST")
		DB_PASS = os.Getenv("MARIA_DB_PASSWORD")
	}
	db, err := sql.Open("mysql", DB_USER+":"+DB_PASS+"@tcp("+DB_HOST+":3306)/vibe_db?parseTime=true")
	if err != nil {
		log.Fatal("Unable to create connection to DB:", err)
	}
	DB = db
}
