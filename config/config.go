package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

// Config represents a configuration file
type Config struct {
	Pin  string `json:"pin"`
	Fans []Fan  `json:"fans"`
}

// Fan represents a fan accessory
type Fan struct {
	Name             string `json:"name"`
	Manufacturer     string `json:"manufacturer"`
	Model            string `json:"model"`
	Serial           string `json:"serial"`
	IsDefaultPowerOn bool   `json:"default_power_on"`
	DefaultSpeed     int    `json:"default_speed"`
	Speeds           []struct {
		URL   string `json:"url"`
		Speed int    `json:"speed"`
	} `json:"speeds"`
}

// GetConfig read and parse the config file
func GetConfig() (Config, error) {
	var cfg Config
	configFile, err := os.Open("config.json")
	if err != nil {
		return cfg, err
	}
	configFileBytes, _ := ioutil.ReadAll(configFile)

	err = json.Unmarshal(configFileBytes, &cfg)
	if err != nil {
		return cfg, err
	}
	return cfg, err
}
