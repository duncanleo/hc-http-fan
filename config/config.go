package config

// Config represents a configuration file
type Config struct {
	Name             string `json:"-"`
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
