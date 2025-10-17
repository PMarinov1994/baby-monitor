package main

import (
	"github.com/stianeikeland/go-rpio/v4"
	"githug.com/pmarinov1994/baby-monitor/util"
)

const (
	gpioCamIrFilter = 15
	gpioIrLED       = 14
)

var (
	pinCamIrFilter rpio.Pin
	pinIrLed       rpio.Pin
)

func initGpio() {
	if err := rpio.Open(); err != nil {
		util.CheckError(&err)
	}

	pinCamIrFilter = rpio.Pin(gpioCamIrFilter)
	pinCamIrFilter.High()

	pinIrLed = rpio.Pin(gpioIrLED)
	pinIrLed.Low()
}

func closeGpio() {
	if err := rpio.Close(); err != nil {
		util.CheckError(&err)
	}
}

func toggleNightVision(toggle bool) {
	if toggle {
		pinCamIrFilter.Low()
		pinIrLed.High()
	} else {
		pinCamIrFilter.High()
		pinIrLed.Low()
	}
}
