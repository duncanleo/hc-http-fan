package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"sort"

	"github.com/brutella/hc/characteristic"

	"github.com/brutella/hc/service"

	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
	hcLog "github.com/brutella/hc/log"
	"github.com/duncanleo/hc-http-fan/config"
)

var (
	power bool
	speed int
)

func main() {
	hcLog.Debug.Enable()

	cfg, err := config.GetConfig()
	if err != nil {
		log.Panic(err)
	}
	power = cfg.IsDefaultPowerOn
	speed = cfg.DefaultSpeed

	sort.SliceStable(cfg.Speeds, func(i, j int) bool {
		return cfg.Speeds[i].Speed < cfg.Speeds[j].Speed
	})

	info := accessory.Info{
		Name:         cfg.Name,
		Manufacturer: cfg.Manufacturer,
		Model:        cfg.Model,
		SerialNumber: cfg.Serial,
	}

	ac := accessory.New(info, accessory.TypeFan)

	fan := service.NewFan()

	fan.On.OnValueGet(func() interface{} { return power })
	fan.On.OnValueRemoteUpdate(func(p bool) {
		power = p
	})
	rotationSpeed := characteristic.NewRotationSpeed()
	rotationSpeed.OnValueGet(func() interface{} {
		return speed
	})
	rotationSpeed.OnValueRemoteUpdate(func(v float64) {
		speed = int(v)

		closestSpeedIndex := 0
		for i, s := range cfg.Speeds {
			if s.Speed > speed {
				closestSpeedIndex = i
				break
			}
		}

		log.Printf("Fishcake %d", cfg.Speeds[closestSpeedIndex].Speed)

		if closestSpeedIndex > 0 &&
			closestSpeedIndex+1 < len(cfg.Speeds) {
			lowerSpeed := cfg.Speeds[closestSpeedIndex].Speed
			upperSpeed := cfg.Speeds[closestSpeedIndex+1].Speed
			if upperSpeed-speed < speed-lowerSpeed {
				closestSpeedIndex++
			}
		}

		closestSpeed := cfg.Speeds[closestSpeedIndex]

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

	hcConfig := hc.Config{
		Pin:         "00102003",
		StoragePath: "storage",
	}

	t, err := hc.NewIPTransport(hcConfig, ac)
	if err != nil {
		log.Panic(err)
	}

	hc.OnTermination(func() {
		<-t.Stop()
	})

	t.Start()
}
