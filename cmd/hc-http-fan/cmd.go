package main

import (
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/brutella/hc/characteristic"

	"github.com/brutella/hc/service"

	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
	"github.com/duncanleo/hc-http-fan/config"
)

func createFanAccessory(cfgFan config.Fan) *accessory.Accessory {
	var (
		power = cfgFan.IsDefaultPowerOn
		speed = cfgFan.DefaultSpeed
	)

	sort.SliceStable(cfgFan.Speeds, func(i, j int) bool {
		return cfgFan.Speeds[i].Speed < cfgFan.Speeds[j].Speed
	})

	info := accessory.Info{
		Name:         cfgFan.Name,
		Manufacturer: cfgFan.Manufacturer,
		Model:        cfgFan.Model,
		SerialNumber: cfgFan.Serial,
	}

	ac := accessory.New(info, accessory.TypeFan)

	fan := service.NewFan()

	fan.On.OnValueGet(func() interface{} { return power })
	fan.On.OnValueRemoteUpdate(func(p bool) {
		power = p

		var url = cfgFan.Power.OffURL

		if p {
			url = cfgFan.Power.OnURL
		}

		if len(url) > 0 {
			resp, err := http.Get(url)
			if err != nil {
				log.Println(err)
			}

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Println(err)
			}

			log.Println(string(body))
		}
	})
	rotationSpeed := characteristic.NewRotationSpeed()
	rotationSpeed.OnValueGet(func() interface{} {
		return speed
	})
	rotationSpeed.OnValueRemoteUpdate(func(v float64) {
		speed = int(v)

		closestSpeedIndex := 0
		for i, s := range cfgFan.Speeds {
			if s.Speed > speed {
				closestSpeedIndex = i
				break
			}
		}

		if closestSpeedIndex > 0 &&
			closestSpeedIndex+1 < len(cfgFan.Speeds) {
			lowerSpeed := cfgFan.Speeds[closestSpeedIndex].Speed
			upperSpeed := cfgFan.Speeds[closestSpeedIndex+1].Speed
			if upperSpeed-speed < speed-lowerSpeed {
				closestSpeedIndex++
			}
		}

		closestSpeed := cfgFan.Speeds[closestSpeedIndex]

		log.Printf("Requested speed %d, mapped to %d", speed, closestSpeed.Speed)

		resp, err := http.Get(closestSpeed.URL)
		if err != nil {
			log.Println(err)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println(err)
		}

		log.Println(string(body))
	})
	fan.AddCharacteristic(rotationSpeed.Characteristic)

	ac.AddService(fan.Service)
	return ac
}

func createLightAccessory(cfgLight config.Light) *accessory.Accessory {
	var (
		power      = cfgLight.IsDefaultPowerOn
		brightness = cfgLight.DefaultBrightness
	)

	info := accessory.Info{
		Name:         cfgLight.Name,
		Manufacturer: cfgLight.Manufacturer,
		Model:        cfgLight.Model,
		SerialNumber: cfgLight.Serial,
	}

	switch cfgLight.Type {
	case config.LightTypeToggle:
		brightness = cfgLight.GetToggleBrightnessLevels()[cfgLight.GetClosestToggleBrightnessIndex(cfgLight.DefaultBrightness)]
		break
	default:
		cfgLight.Type = config.LightTypeBasic
		log.Printf("The light '%s' has unknown type '%s' specified, assuming BASIC.\n", cfgLight.Name, cfgLight.Type)
	}

	ac := accessory.New(info, accessory.TypeLightbulb)

	light := service.NewLightbulb()

	var updateLightToggleBrightness = func(v int) {
		currentIndex := cfgLight.GetClosestToggleBrightnessIndex(brightness)
		targetIndex := cfgLight.GetClosestToggleBrightnessIndex(v)

		numSteps := 0

		if targetIndex == currentIndex {
			numSteps = 0
		} else if !cfgLight.Toggle.Ascending && targetIndex > currentIndex {
			numSteps = targetIndex - currentIndex
		} else if !cfgLight.Toggle.Ascending {
			numSteps = cfgLight.Toggle.LevelCount - int(math.Abs(float64(targetIndex)-float64(currentIndex)))
		} else if cfgLight.Toggle.Ascending && targetIndex < currentIndex {
			numSteps = currentIndex - targetIndex
		} else if cfgLight.Toggle.Ascending {
			numSteps = cfgLight.Toggle.LevelCount - int(math.Abs(float64(currentIndex)-float64(targetIndex)))
		}

		log.Printf("%d (i=%d) => %d(i=%d), We're gonna toggle %d times\n", brightness, currentIndex, v, targetIndex, numSteps)

		brightness = cfgLight.GetToggleBrightnessLevels()[targetIndex]

		for i := 0; i < numSteps; i++ {
			resp, err := http.Get(cfgLight.Toggle.URL)
			if err != nil {
				log.Println(err)
			}

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Println(err)
			}

			log.Println(string(body))
			time.Sleep(1 * time.Second)
		}
	}

	light.On.OnValueGet(func() interface{} { return power })
	light.On.OnValueRemoteUpdate(func(p bool) {
		power = p

		switch cfgLight.Type {
		case config.LightTypeToggle:
			log.Printf("Set power of light '%s' to %v\n", cfgLight.Name, p)
			if !p {
				updateLightToggleBrightness(0)
			} else {
				levels := cfgLight.GetToggleBrightnessLevels()
				targetBrightness := levels[0]
				if cfgLight.Toggle.Ascending {
					targetBrightness = levels[len(levels)-1]
				}
				updateLightToggleBrightness(targetBrightness)
			}
			break
		case config.LightTypeBasic:
			var url = cfgLight.Basic.Power.OffURL

			if p {
				url = cfgLight.Basic.Power.OnURL
			}

			if len(url) > 0 {
				resp, err := http.Get(url)
				if err != nil {
					log.Println(err)
				}

				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					log.Println(err)
				}

				log.Println(string(body))
			}
			break
		}
	})

	light.Brightness.OnValueGet(func() interface{} {
		return brightness
	})
	light.Brightness.OnValueRemoteUpdate(func(v int) {
		switch cfgLight.Type {
		case config.LightTypeToggle:
			updateLightToggleBrightness(v)
			return
		case config.LightTypeBasic:
			targetIndex := cfgLight.GetClosestBrightnessIndex(v)

			resp, err := http.Get(cfgLight.Basic.BrightnessLevels[targetIndex].URL)
			if err != nil {
				log.Println(err)
			}

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Println(err)
			}

			log.Println(string(body))
			break
		default:
			log.Printf("Unsupported type %s\n", cfgLight.Type)
			break
		}

		brightness = v
	})

	ac.AddService(light.Service)
	return ac
}

func main() {
	cfg, err := config.GetConfig()
	if err != nil {
		log.Panic(err)
	}

	var accessories []*accessory.Accessory

	for _, cfgFan := range cfg.Fans {
		accessories = append(accessories, createFanAccessory(cfgFan))
	}

	for _, cfgLight := range cfg.Lights {
		accessories = append(accessories, createLightAccessory(cfgLight))
	}

	var portStr = strconv.Itoa(cfg.Port)
	if cfg.Port == 0 {
		portStr = ""
	}

	hcConfig := hc.Config{
		Pin:         cfg.Pin,
		StoragePath: cfg.StoragePath,
		Port:        portStr,
	}

	var subAccs []*accessory.Accessory
	if len(accessories) > 1 {
		subAccs = accessories[1:]
	}

	t, err := hc.NewIPTransport(hcConfig, accessories[0], subAccs...)
	if err != nil {
		log.Panic(err)
	}

	hc.OnTermination(func() {
		<-t.Stop()
	})

	t.Start()
}
