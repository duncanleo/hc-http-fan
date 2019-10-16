package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strconv"

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

func main() {
	cfg, err := config.GetConfig()
	if err != nil {
		log.Panic(err)
	}

	var accessories []*accessory.Accessory

	for _, cfgFan := range cfg.Fans {
		accessories = append(accessories, createFanAccessory(cfgFan))
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
