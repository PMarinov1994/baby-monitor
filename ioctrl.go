package main

import (
	"github.com/stianeikeland/go-rpio/v4"
	"githug.com/pmarinov1994/baby-monitor/util"
)

const (
	gpioCamIrFilter = 15
	gpioIrLED       = 14

	gpioGreenLED = 23
	gpioRedLED   = 24
)

var (
	pinCamIrFilter rpio.Pin
	pinIrLed       rpio.Pin

	pinGreenLED rpio.Pin
	pinRedLED   rpio.Pin

	isNightVisionOn = false
)

func initGpio() {
	if err := rpio.Open(); err != nil {
		util.CheckError(&err)
	}

	pinCamIrFilter = rpio.Pin(gpioCamIrFilter)
	pinCamIrFilter.Output()
	pinCamIrFilter.High()

	pinIrLed = rpio.Pin(gpioIrLED)
	pinIrLed.Output()
	pinIrLed.Low()

	pinGreenLED = rpio.Pin(gpioGreenLED)
	pinGreenLED.Output()
	pinGreenLED.Low()

	pinRedLED = rpio.Pin(gpioRedLED)
	pinRedLED.Output()
	pinRedLED.Low()
}

func closeGpio() {
	if err := rpio.Close(); err != nil {
		util.CheckError(&err)
	}
}

func toggleNightVision(toggle bool) {
	isNightVisionOn = toggle

	if toggle {
		pinCamIrFilter.Low()
		pinIrLed.High()
		pinRedLED.High()
	} else {
		pinCamIrFilter.High()
		pinIrLed.Low()
		pinRedLED.Low()
	}
}

func toggleGreenLED(toggle bool) {
	if toggle {
		pinGreenLED.High()
	} else {
		pinGreenLED.Low()
	}
}
