package config

import (
	"encoding/json"
	"io/ioutil"
	"math"
	"os"
)

// Config represents a configuration file
type Config struct {
	Pin         string  `json:"pin"`
	StoragePath string  `json:"storage_path"`
	Port        int     `json:"port"`
	Fans        []Fan   `json:"fans"`
	Lights      []Light `json:"lights"`
}

// Fan represents a fan accessory
type Fan struct {
	Name             string `json:"name"`
	Manufacturer     string `json:"manufacturer"`
	Model            string `json:"model"`
	Serial           string `json:"serial"`
	IsDefaultPowerOn bool   `json:"default_power_on"`
	DefaultSpeed     int    `json:"default_speed"`
	Power            struct {
		OnURL  string `json:"on_url"`
		OffURL string `json:"off_url"`
	} `json:"power"`
	Speeds []struct {
		URL   string `json:"url"`
		Speed int    `json:"speed"`
	} `json:"speeds"`
}

// LightType an enum type to represent type of light
type LightType string

const (
	// LightTypeToggle represents lights that have a single toggle button to toggle between speeds & OFF
	LightTypeToggle LightType = "toggle"
	// LightTypeBasic represents lights that have specific URLs to call for OFF, ON, and brightness levels
	LightTypeBasic LightType = "basic"
)

// Light represents a light accessory
type Light struct {
	Name              string    `json:"name"`
	Manufacturer      string    `json:"manufacturer"`
	Model             string    `json:"model"`
	Serial            string    `json:"serial"`
	IsDefaultPowerOn  bool      `json:"default_power_on"`
	Type              LightType `json:"type"`
	DefaultBrightness int       `json:"default_brightness"`

	Basic struct {
		BrightnessLevels []struct {
			URL        string `json:"url"`
			Brightness int    `json:"brightness"`
		} `json:"brightness_levels"`

		Power struct {
			OnURL  string `json:"on_url"`
			OffURL string `json:"off_url"`
		} `json:"power"`
	} `json:"basic"` // Used only for BASIC light types

	Toggle struct {
		Ascending  bool   `json:"ascending"`
		URL        string `json:"url"`
		LevelCount int    `json:"level_count"` // Number of toggle brightness levels, INCLUDING an OFF mode.
	} `json:"toggle"` // Used only for TOGGLE light types
}

func (l Light) GetClosestBrightnessIndex(brightness int) int {
	closestIndex := 0
	for i, b := range l.Basic.BrightnessLevels {
		if b.Brightness > brightness {
			closestIndex = i
			break
		}
	}

	if closestIndex > 0 &&
		closestIndex+1 < len(l.Basic.BrightnessLevels) {
		lower := l.Basic.BrightnessLevels[closestIndex].Brightness
		upper := l.Basic.BrightnessLevels[closestIndex+1].Brightness
		if upper-brightness < brightness-lower {
			closestIndex++
		}
	}
	return closestIndex
}

func (l Light) GetClosestToggleBrightnessIndex(brightness int) int {
	levels := l.GetToggleBrightnessLevels()
	closestIndex := 0
	curr := 9999
	for i, b := range levels {
		if math.Abs(float64(brightness)-float64(b)) < math.Abs(float64(brightness)-float64(curr)) {
			curr = b
			closestIndex = i
		}
	}
	return closestIndex
}

func (l Light) GetToggleBrightnessStep() int {
	return int(math.Floor(100.0 / float64(l.Toggle.LevelCount)))
}

func (l Light) GetToggleBrightnessLevels() []int {
	var brightnessLevels []int

	i := l.Toggle.LevelCount - 1
	if l.Toggle.Ascending {
		i = 0
	}
	for {
		brightnessLevels = append(brightnessLevels, i*l.GetToggleBrightnessStep())
		if l.Toggle.Ascending {
			i++
			if i == l.Toggle.LevelCount {
				break
			}
		} else {
			i--
			if i < 0 {
				break
			}
		}
	}
	return brightnessLevels
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
