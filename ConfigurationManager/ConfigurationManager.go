package ConfigurationManager

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
)

type Configuration struct {
	DocumentRoot string
	Port string
}

type JWTConfig struct {
	Key string `json:"key"`
}

var version = "1.0.0"
var config Configuration

func OpenConfiguration() Configuration {
	if _, err := os.Stat("config.json"); err != nil {
		log.Fatalf("[ERROR] The configuration file is not found ::> config.json\n%s" +
			"\nPlease insure that the file is in the same directory as the executable", err)
	}
	configJson, err:= ioutil.ReadFile("config.json")
	if err != nil {
		log.Fatalf("[ERROR] Unable to read the configuration file ::> config.json\n%s" +
			"\nPlease insure that the file has the reading right", err)
	}
	err = json.Unmarshal(configJson, &config)

	if err != nil {
		log.Fatalf("[ERROR] Configuration file incorrect ::> config.json\n%s", err)
	}

	return config
}

func LoadJWTKey() JWTConfig {
	if _, err := os.Stat("jwt.json"); err != nil {
		log.Fatalf("[ERROR] The JWT Key file is not found ::> jwt.json\n%s" +
			"\nPlease insure that the file is in the same directory as the executable", err)
	}
	configJson, err:= ioutil.ReadFile("jwt.json")
	if err != nil {
		log.Fatalf("[ERROR] Unable to read the JWT Key file ::> jwt.json\n%s" +
			"\nPlease insure that the file has the reading right", err)
	}
	var jwt JWTConfig
	err = json.Unmarshal(configJson, &jwt)

	if err != nil {
		log.Fatalf("[ERROR] JWT Key file incorrect ::> config.json\n%s", err)
	}

	return jwt
}

func GetConfiguration() Configuration {
	return config
}

func GetVersion() string {
	return version
}
