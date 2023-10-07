package main

import (
	"fmt"
	"log"

	ExternalDevice "github.com/W-Floyd/ha-mqtt-iot/devices/externaldevice"
	InternalDevice "github.com/W-Floyd/ha-mqtt-iot/devices/internaldevice"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/racerxdl/go-mcp23017"
)

type Relais struct {
	Switch    *ExternalDevice.Switch
	pin       uint8
	pinDevice *mcp23017.Device
}

func NewRelay(pin uint8, pinDevice *mcp23017.Device) *Relais {
	name := fmt.Sprintf("Relay %v", pin)
	safeName := fmt.Sprintf("relay_%v", pin)
	externalDevice := InternalDevice.Switch{
		Name:     &name,
		ObjectId: &safeName,
		UniqueId: &safeName,
	}.Translate()
	newRelay := &Relais{
		pin:       pin,
		Switch:    &externalDevice,
		pinDevice: pinDevice,
	}
	topicPrefix := ExternalDevice.GetTopicPrefix(newRelay.Switch)
	commandTopic := topicPrefix + "cmd"
	stateTopic := topicPrefix + "state"
	newRelay.Switch.CommandFunc = newRelay.Command
	newRelay.Switch.CommandTopic = &commandTopic
	newRelay.Switch.StateFunc = newRelay.getState
	newRelay.Switch.StateTopic = &stateTopic

	newRelay.Switch.Initialize()
	return newRelay
}

func (r *Relais) getState() string {
	level, err := r.pinDevice.DigitalRead(r.pin)
	log.Println("Read Pin", r.pin, level)
	if err != nil {
		log.Println(err)
		return "NA"
	}
	if level {
		return "ON"
	} else {
		return "OFF"
	}

}

func (r *Relais) Command(msg mqtt.Message, c mqtt.Client) {
	state := string(msg.Payload())
	log.Print("Setting pin", r.pin)
	if state == "ON" {
		log.Println(" high")
		r.pinDevice.DigitalWrite(r.pin, false)
	} else {
		log.Println(" low")
		r.pinDevice.DigitalWrite(r.pin, true)
	}
}

func (r *Relais) GetMqttDevice() ExternalDevice.Device {
	return r.Switch
}
