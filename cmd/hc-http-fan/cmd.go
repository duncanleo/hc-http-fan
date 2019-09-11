package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"

	"github.com/duncanleo/hc-http-fan/config"
)

func main() {
	configFile, err := os.Open("config.json")
	if err != nil {
		log.Panic(err)
	}
	configFileBytes, _ := ioutil.ReadAll(configFile)
	var cfg config.Config
	err = json.Unmarshal(configFileBytes, &cfg)
	if err != nil {
		log.Panic(err)
	}
	log.Println(cfg)
}
