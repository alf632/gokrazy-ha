package main

import (
	"strings"

	ExternalDevice "github.com/W-Floyd/ha-mqtt-iot/devices/externaldevice"
	InternalDevice "github.com/W-Floyd/ha-mqtt-iot/devices/internaldevice"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type ElementType int

const (
	TypeButton ElementType = iota
	TypeSwitch
)

type NextionElement interface {
	GetMqttDevice() ExternalDevice.Device
	GetType() ElementType
}

// NextionButton represents a Button on the Display and translates into a binarySensor for HA
type NextionButton struct {
	BinarySensor *ExternalDevice.BinarySensor
	State        string
	Type         ElementType
}

func newNextionButton(name, short string) *NextionButton {
	safeName := short
	safeName = strings.ReplaceAll(safeName, " ", "-")
	externalDevice := InternalDevice.BinarySensor{
		Name:     &name,
		ObjectId: &safeName,
		UniqueId: &safeName,
	}.Translate()
	newButton := NextionButton{
		Type:         TypeButton,
		BinarySensor: &externalDevice,
	}
	newButton.BinarySensor.StateFunc = newButton.GetState

	newButton.BinarySensor.Initialize()
	return &newButton
}

func (nxt *NextionButton) SetStateSerial(newState string) {
	nxt.State = newState
	nxt.GetMqttDevice().UpdateState()
}

func (nxt *NextionButton) GetState() string {
	return nxt.State
}

func (nxt *NextionButton) GetMqttDevice() ExternalDevice.Device {
	return nxt.BinarySensor
}

func (nxt *NextionButton) GetType() ElementType {
	return nxt.Type
}

// NextionSwitch represents a Switch on the Display which is synced with a Switch on HA side
type NextionSwitch struct {
	Switch *ExternalDevice.Switch
	State  string
	Sender func(string)
	Type   ElementType
}

func newNextionSwitch(name, short string) *NextionSwitch {
	safeName := short
	safeName = strings.ReplaceAll(safeName, " ", "-")
	externalDevice := InternalDevice.Switch{
		Name:     &name,
		ObjectId: &safeName,
		UniqueId: &safeName,
	}.Translate()
	newButton := NextionSwitch{
		Type:   TypeSwitch,
		Switch: &externalDevice}
	newButton.Switch.CommandFunc = newButton.SetStateMqtt
	newButton.Switch.StateFunc = newButton.GetState

	newButton.Switch.Initialize()
	return &newButton
}

func (nxt *NextionSwitch) RegisterSender(sender func(string)) {
	nxt.Sender = sender
}

func (nxt *NextionSwitch) GetState() string {
	return nxt.State
}

func (nxt *NextionSwitch) SetStateMqtt(m mqtt.Message, c mqtt.Client) {
	nxt.State = string(m.Payload())
	nxt.Sender(nxt.GetMqttDevice().GetUniqueId() + nxt.State)
}

func (nxt *NextionSwitch) SetStateSerial(newState string) {
	nxt.State = newState
	nxt.GetMqttDevice().UpdateState()
}

func (nxt *NextionSwitch) GetMqttDevice() ExternalDevice.Device {
	return nxt.Switch
}

func (nxt *NextionSwitch) GetType() ElementType {
	return nxt.Type
}
