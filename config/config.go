package config

import (
	"encoding/json"
	"io/ioutil"
	"math"
	"os"
	"sort"
)

// Config represents a configuration file
type Config struct {
	Bridge      Accessory `json:"bridge"`
	Pin         string    `json:"pin"`
	StoragePath string    `json:"storage_path"`
	Port        int       `json:"port"`
	BrokerURI   string    `json:"broker_uri"`
	ClientID    string    `json:"client_id"`
	Fans        []Fan     `json:"fans"`
	Lights      []Light   `json:"lights"`
}

type MQTTPublish struct {
	Topic   string `json:"topic"`
	Payload string `json:"payload"`
}

type Accessory struct {
	Name         string `json:"name"`
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
	Serial       string `json:"serial"`
}

type FanSpeed struct {
	MQTTPublish
	Speed int `json:"speed"`
}

// Fan represents a fan accessory
type Fan struct {
	Accessory
	IsDefaultPowerOn bool `json:"default_power_on"`
	DefaultSpeed     int  `json:"default_speed"`
	Power            struct {
		On  MQTTPublish `json:"on"`
		Off MQTTPublish `json:"off"`
	} `json:"power"`
	Speeds []FanSpeed `json:"speeds"`
}

func (f Fan) sortSpeeds() {
	sort.SliceStable(f.Speeds, func(i, j int) bool {
		return f.Speeds[i].Speed < f.Speeds[j].Speed
	})
}

func (f Fan) GetClosestSpeedIndex(speed int) int {
	f.sortSpeeds()
	closestIndex := 0
	for i, s := range f.Speeds {
		if s.Speed >= speed {
			closestIndex = i
			break
		}
	}

	if closestIndex > 0 &&
		closestIndex+1 < len(f.Speeds) {
		lower := f.Speeds[closestIndex].Speed
		upper := f.Speeds[closestIndex+1].Speed
		if upper-speed < speed-lower {
			closestIndex++
		}
	}
	return closestIndex
}

// LightType an enum type to represent type of light
type LightType string

const (
	// LightTypeToggle represents lights that have a single toggle button to toggle between speeds & OFF
	LightTypeToggle LightType = "toggle"
	// LightTypeBasic represents lights that have specific URLs to call for OFF, ON, and brightness levels
	LightTypeBasic LightType = "basic"
	// LightTypeSwitch represents a light that only has ON and OFF.
	LightTypeSwitch LightType = "switch"
)

// Light represents a light accessory
type Light struct {
	Accessory
	IsDefaultPowerOn  bool      `json:"default_power_on"`
	Type              LightType `json:"type"`
	DefaultBrightness int       `json:"default_brightness"`

	Basic struct {
		BrightnessLevels []struct {
			MQTTPublish
			Brightness int `json:"brightness"`
		} `json:"brightness_levels"`

		Power struct {
			On  MQTTPublish `json:"on"`
			Off MQTTPublish `json:"off"`
		} `json:"power"`
	} `json:"basic"` // Used only for BASIC light types

	Toggle struct {
		MQTTPublish
		Ascending  bool `json:"ascending"`
		LevelCount int  `json:"level_count"` // Number of toggle brightness levels, INCLUDING an OFF mode.
	} `json:"toggle"` // Used only for TOGGLE light types

	Switch struct {
		On  MQTTPublish `json:"on"`
		Off MQTTPublish `json:"off"`
	} `json:"switch"`
}

func (l Light) GetClosestBrightnessIndex(brightness int) int {
	closestIndex := 0
	for i, b := range l.Basic.BrightnessLevels {
		if b.Brightness >= brightness {
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
