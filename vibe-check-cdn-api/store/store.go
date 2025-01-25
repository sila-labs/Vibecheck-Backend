package store

import (
	"database/sql"
	_ "encoding/json"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gomodule/redigo/redis"
)

var DB *sql.DB
var Cache *redis.Pool

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
		panic(err.Error())
	}
	DB = db
}

// Initialize Cache
func InitCache() {
	APP_ENV := os.Getenv("APP_ENV")
	REDIS_HOST := "127.0.0.1"
	if APP_ENV == "prod" {
		REDIS_HOST = os.Getenv("REDIS_HOST")
	}
	Cache = &redis.Pool{
		MaxIdle:     100,
		IdleTimeout: 240 * time.Second,
		//MaxActive:   200,
		//Wait:        true,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", REDIS_HOST+":6379")
			if err != nil {
				return nil, err
			}
			return c, err
		},
	}
}

func ToString(reply interface{}, err error) (string, error) {
	return redis.String(reply, err)
}
