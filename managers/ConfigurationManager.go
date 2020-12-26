package managers

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

func GetConfiguration() Configuration {
	return config
}
