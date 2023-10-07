package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	ExternalDevice "github.com/W-Floyd/ha-mqtt-iot/devices/externaldevice"
)

var nextionElements map[string]NextionElement

func main() {

	nextionElements = map[string]NextionElement{}

	newButton := newNextionButton("test button", "b0")
	nextionElements["b0"] = newButton

	newSwitch := newNextionSwitch("test switch", "s0")
	nextionElements["s0"] = newSwitch

	mqttDevices := []ExternalDevice.Device{}
	for _, element := range nextionElements {
		mqttDevices = append(mqttDevices, element.GetMqttDevice())
	}

	mqttc := NewMqttController(mqttDevices)
	defer mqttc.Stop()
	serialc := NewSerialController(nextionElements)
	defer serialc.stop()

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Everything is set up")

	signal := <-done
	log.Println("Exiting with signal", signal.String())
}
