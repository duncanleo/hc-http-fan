package main

import (
	"fmt"
	"log"
	"math"
	"net/url"
	"sort"
	"strconv"
	"time"

	"github.com/brutella/hc/characteristic"
	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/brutella/hc/service"

	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
	"github.com/duncanleo/hc-mqtt-fan/config"
)

var (
	mqttClient mqtt.Client
)

func connect(clientID string, uri *url.URL) (mqtt.Client, error) {
	var opts = mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s", uri.Host))
	opts.SetUsername(uri.User.Username())
	password, _ := uri.User.Password()
	opts.SetPassword(password)
	opts.SetClientID(clientID)

	var client = mqtt.NewClient(opts)
	var token = client.Connect()
	for !token.WaitTimeout(3 * time.Second) {
	}
	return client, token.Error()
}

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

		var target = cfgFan.Power.Off

		if p {
			target = cfgFan.Power.On
		}

		token := mqttClient.Publish(target.Topic, 0, false, target.Payload)
		if token.Error() != nil {
			log.Println(token.Error())
		}

		if p {
			// Restore previous speed
			closestSpeedIndex := cfgFan.GetClosestSpeedIndex(speed)
			closestSpeed := cfgFan.Speeds[closestSpeedIndex]

			log.Printf("Requested speed %d, mapped to %d", speed, closestSpeed.Speed)

			token := mqttClient.Publish(closestSpeed.Topic, 0, false, closestSpeed.Payload)
			if token.Error() != nil {
				log.Println(token.Error())
			}
		}
	})
	rotationSpeed := characteristic.NewRotationSpeed()
	rotationSpeed.OnValueGet(func() interface{} {
		return speed
	})
	rotationSpeed.OnValueRemoteUpdate(func(v float64) {
		speed = int(v)

		closestSpeedIndex := cfgFan.GetClosestSpeedIndex(speed)

		closestSpeed := cfgFan.Speeds[closestSpeedIndex]

		log.Printf("Requested speed %d, mapped to %d", speed, closestSpeed.Speed)

		token := mqttClient.Publish(closestSpeed.Topic, 0, false, closestSpeed.Payload)
		if token.Error() != nil {
			log.Println(token.Error())
		}
	})
	fan.AddCharacteristic(rotationSpeed.Characteristic)

	currentFanState := characteristic.NewCurrentFanState()
	currentFanState.OnValueGet(func() interface{} {
		if !power {
			return characteristic.CurrentFanStateIdle
		}
		return characteristic.CurrentFanStateBlowingAir
	})
	fan.AddCharacteristic(currentFanState.Characteristic)

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
	case config.LightTypeSwitch:
		break
	default:
		cfgLight.Type = config.LightTypeBasic
		log.Printf("The light '%s' has unknown type '%s' specified, assuming BASIC.\n", cfgLight.Name, cfgLight.Type)
	}

	ac := accessory.New(info, accessory.TypeLightbulb)

	light := service.Lightbulb{}
	light.Service = service.New(service.TypeLightbulb)
	light.On = characteristic.NewOn()
	light.AddCharacteristic(light.On.Characteristic)
	light.Brightness = characteristic.NewBrightness()

	switch cfgLight.Type {
	case config.LightTypeBasic:
	case config.LightTypeToggle:
		light.AddCharacteristic(light.Brightness.Characteristic)
		break
	case config.LightTypeSwitch:
		break
	}

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
			token := mqttClient.Publish(cfgLight.Toggle.Topic, 0, false, cfgLight.Toggle.Payload)
			if token.Error() != nil {
				log.Println(token.Error())
			}

			time.Sleep(1 * time.Second)
		}
	}

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
			target := cfgLight.Basic.BrightnessLevels[targetIndex]

			token := mqttClient.Publish(target.Topic, 0, false, target.Payload)
			if token.Error() != nil {
				log.Println(token.Error())
			}

			break
		default:
			log.Printf("Unsupported type %s\n", cfgLight.Type)
			break
		}
	})

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
			var target = cfgLight.Basic.Power.Off

			if p {
				target = cfgLight.Basic.Power.On
			}

			token := mqttClient.Publish(target.Topic, 0, false, target.Payload)
			if token.Error() != nil {
				log.Println(token.Error())
			}
			break
		case config.LightTypeSwitch:
			target := cfgLight.Switch.Off
			if p {
				target = cfgLight.Switch.On
			}
			token := mqttClient.Publish(target.Topic, 0, false, target.Payload)
			if token.Error() != nil {
				log.Println(token.Error())
			}
			break
		}
	})

	ac.AddService(light.Service)
	return ac
}

func main() {
	cfg, err := config.GetConfig()
	if err != nil {
		log.Panic(err)
	}

	mqttURI, err := url.Parse(cfg.BrokerURI)
	if err != nil {
		log.Fatal(err)
	}

	mqttClient, err = connect(cfg.ClientID, mqttURI)
	if err != nil {
		log.Fatal(err)
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
