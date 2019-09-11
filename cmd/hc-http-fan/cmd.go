package main

import (
	"log"

	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
	"github.com/duncanleo/hc-http-fan/config"
)

func main() {
	cfg, err := config.GetConfig()
	if err != nil {
		log.Panic(err)
	}
	if err != nil {
		log.Panic(err)
	}
	log.Println(cfg)
}
