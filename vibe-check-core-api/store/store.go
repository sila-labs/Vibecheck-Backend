package store

import (
	"vibe/config"

	"go.mongodb.org/mongo-driver/mongo"

	"database/sql"
	_ "encoding/json"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gomodule/redigo/redis"
)

var DB *sql.DB
var MONGO_DB_CLIENT *mongo.Client
var Cache *redis.Pool
var UPLOADS_LOCATION string

// UNSURE IF WE ARE STILL USING MARIA DB???
// Initialize DB
func InitDB() {

	config := config.CONFIGURATION

	MARIA_DB_HOST := config.MARIA_DB_HOST
	MARIA_DB_PORT := config.MARIA_DB_PORT
	MARIA_DB_USER := config.MARIA_DB_USERNAME
	MARIA_DB_PASS := config.MARIA_DB_PASSWORD

	// temp local testing on ruky's machine
	// MARIA_DB_HOST := "127.0.0.1"
	// MARIA_DB_PORT := "3306"
	// MARIA_DB_USER := "root"
	// MARIA_DB_PASS := "build"

	// MONGO_URI := ""
	// // MONGO_USER := ""
	// // MONGO_PASS := ""
	// MONGO_ARGS := config.MONGO_ARGS
	// MONGO_HOST := config.MONGO_HOST
	// MONGO_PORT := config.MONGO_PORT

	// UPLOADS_LOCATION = config.UPLOADS_LOCATION

	// if config.APP_ENV == "prod" {
	// 	// Use this until we set up username and password in database, then uncomment USER, PASSWORD and URI below
	// 	MONGO_URI = "mongodb://" + MONGO_HOST + ":" + MONGO_PORT + MONGO_ARGS
	// 	// MONGO_URI = "mongodb://" + MONGO_USER + ":" + MONGO_PASS + "@" + MONGO_HOST + MONGO_PORT + MONGO_ARGS
	// 	fmt.Printf("MONGO URI: %s\n", MONGO_URI)
	// } else {

	// 	MONGO_URI = "mongodb://" + MONGO_HOST + ":" + MONGO_PORT + MONGO_ARGS
	// 	fmt.Printf("MONGO URI: %s\n", MONGO_URI)
	// }

	// Maria DB
	db, err := sql.Open("mysql", MARIA_DB_USER+":"+MARIA_DB_PASS+"@tcp("+MARIA_DB_HOST+":"+MARIA_DB_PORT+")/vibe_db?parseTime=true")
	DB = db

	if err != nil {
		panic(err)
	}

	// // Mongo DB
	// // Create a new client and connect to the server
	// client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(MONGO_URI))
	// if err != nil {
	// 	panic(err)
	// }
	// defer func() {
	// 	if err = client.Disconnect(context.TODO()); err != nil {
	// 		panic(err)
	// 	}
	// }()
	// // Ping the primary
	// if err := client.Ping(context.TODO(), readpref.Primary()); err != nil {
	// 	panic(err)
	// }
	// fmt.Println("Successfully connected to and pinged Mongodb at " + MONGO_URI)

	// if err != nil {
	// 	panic(err.Error())
	// }
	// MONGO_DB_CLIENT = client

}

// Initialize Cache
func InitCache() {

	REDIS_HOST := config.CONFIGURATION.REDIS_HOST
	REDIS_PORT := config.CONFIGURATION.REDIS_PORT

	Cache = &redis.Pool{
		MaxIdle:     100,
		IdleTimeout: 240 * time.Second,
		//MaxActive:   200,
		//Wait:        true,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", REDIS_HOST+":"+REDIS_PORT)
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
