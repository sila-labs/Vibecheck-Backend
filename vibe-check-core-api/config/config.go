package config

import (
	"fmt"

	"github.com/tkanos/gonfig"
)

type Configuration struct {
	APP_ENV           string
	VIBE_PORT         string
	REDIS_HOST        string
	REDIS_PORT        string
	MARIA_DB_USERNAME string
	MARIA_DB_PASSWORD string
	MARIA_DB_PORT     string
	MARIA_DB_HOST     string
	MONGO_USER        string
	MONGO_PASS        string
	MONGO_ARGS        string
	MONGO_HOST        string
	MONGO_PORT        string
	UPLOADS_LOCATION  string
}

var ENV string
var CONFIGURATION Configuration

func InitConfig(params ...string) Configuration {
	configuration := Configuration{}
	ENV = "dev"
	if len(params) > 0 {
		ENV = params[0]
	}
	fileName := fmt.Sprintf("./config/%s_config.json", ENV)
	gonfig.GetConf(fileName, &configuration)
	CONFIGURATION = configuration
	return configuration
}

func PrintConfig() {
	fmt.Println("Using " + ENV + " config... using " + "./config/" + ENV + "_config.json")
}
