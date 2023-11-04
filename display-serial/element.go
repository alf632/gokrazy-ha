package main

import (
	"fmt"
	"log"
	"strings"

	ExternalDevice "github.com/W-Floyd/ha-mqtt-iot/devices/externaldevice"
	InternalDevice "github.com/W-Floyd/ha-mqtt-iot/devices/internaldevice"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type NextionElement interface {
	GetMqttDevice() ExternalDevice.Device
	GetName() string
	GetState() string
	SendStateSerial()
	SetStateSerial(string)
	TouchEvent(byte)
}

type NextionCommon struct {
	ID    uint32
	short string
	State string
	page  int
	send  func(int, string)
}

func (nc *NextionCommon) GetName() string {
	return nc.short
}

func (nc *NextionCommon) GetState() string {
	return nc.State
}

// NextionButton represents a Button on the Display and translates into a binarySensor for HA
type NextionButton struct {
	NextionCommon
	BinarySensor *ExternalDevice.BinarySensor
}

func newNextionButton(name, short string, page int, send func(int, string)) *NextionButton {
	safeName := fmt.Sprintf("p%d%s", page, short)
	safeName = strings.ReplaceAll(safeName, " ", "-")
	externalDevice := InternalDevice.BinarySensor{
		Name:     &name,
		ObjectId: &safeName,
		UniqueId: &safeName,
	}.Translate()
	newButton := NextionButton{BinarySensor: &externalDevice}
	newButton.short = short
	newButton.page = page
	newButton.State = ""
	newButton.send = send
	newButton.BinarySensor.StateFunc = newButton.GetState

	newButton.BinarySensor.Initialize()
	return &newButton
}

func (nb *NextionButton) SendStateSerial() {
	log.Println("serial send state", nb.State, "for", nb.short)
	val := 0
	if nb.GetState() == "ON" {
		val = 1
	}
	nb.send(nb.page, fmt.Sprintf("%s.val=%d", nb.short, val))
}

func (nb *NextionButton) TouchEvent(state byte) {
	if state > 0 {
		nb.SetStateSerial("ON")
	} else {
		nb.SetStateSerial("OFF")
	}
}

func (nb *NextionButton) SetStateSerial(newState string) {
	nb.State = newState
	log.Println("serial set state", nb.State, "for", nb.short)
	nb.GetMqttDevice().UpdateState()
}

func (nb *NextionButton) GetMqttDevice() ExternalDevice.Device {
	return nb.BinarySensor
}

// NextionSwitch represents a Switch on the Display which is synced with a Switch on HA side
type NextionSwitch struct {
	NextionCommon
	Switch *ExternalDevice.Switch
}

func newNextionSwitch(name, short string, page int, send func(int, string)) *NextionSwitch {
	safeName := fmt.Sprintf("p%d%s", page, short)
	safeName = strings.ReplaceAll(safeName, " ", "-")
	externalDevice := InternalDevice.Switch{
		Name:     &name,
		ObjectId: &safeName,
		UniqueId: &safeName,
	}.Translate()
	newswitch := NextionSwitch{Switch: &externalDevice}
	newswitch.short = short
	newswitch.page = page
	newswitch.State = ""
	newswitch.Switch.CommandFunc = newswitch.SetStateMqtt
	newswitch.Switch.StateFunc = newswitch.GetState
	newswitch.send = send

	newswitch.Switch.Initialize()
	return &newswitch
}

func (nxt *NextionSwitch) SetStateMqtt(m mqtt.Message, c mqtt.Client) {
	nxt.State = string(m.Payload())
	log.Println("mqtt set state", nxt.State, "for", nxt.short)
	nxt.SendStateSerial()
}

func (nxt *NextionSwitch) SendStateSerial() {
	log.Println("serial send state", nxt.State, "for", nxt.short)
	val := 0
	if nxt.State == "ON" {
		val = 1
	}
	nxt.send(nxt.page, fmt.Sprintf("%s.val=%d", nxt.short, val))
}

func (nxt *NextionSwitch) TouchEvent(state byte) {
	if int(state) == 0 {
		if nxt.State == "OFF" {
			nxt.SetStateSerial("ON")
		} else {
			nxt.SetStateSerial("OFF")
		}
	}
}

func (nxt *NextionSwitch) SetStateSerial(state string) {
	nxt.State = state
	log.Println("serial set state", nxt.State, "for", nxt.short)
	nxt.GetMqttDevice().UpdateState()
}

func (nxt *NextionSwitch) GetMqttDevice() ExternalDevice.Device {
	return nxt.Switch
}

// NextionText represents a Text on the Display which is synced with a Text on HA side
type NextionText struct {
	NextionCommon
	Text *ExternalDevice.Text
}

func newNextionText(name, short string, page int, send func(int, string)) *NextionText {
	safeName := fmt.Sprintf("p%d%s", page, short)
	safeName = strings.ReplaceAll(safeName, " ", "-")
	externalDevice := InternalDevice.Text{
		Name:     &name,
		ObjectId: &safeName,
		UniqueId: &safeName,
	}.Translate()
	newText := NextionText{Text: &externalDevice}
	newText.short = short
	newText.page = page
	newText.State = ""
	newText.Text.CommandFunc = newText.SetStateMqtt
	newText.Text.StateFunc = newText.GetState
	newText.send = send

	newText.Text.Initialize()
	return &newText
}

func (nxt *NextionText) SetStateMqtt(m mqtt.Message, c mqtt.Client) {
	nxt.State = string(m.Payload())
	log.Println("mqtt set state", nxt.State, "for", nxt.short)
	nxt.SendStateSerial()
}

func (nxt *NextionText) SendStateSerial() {
	log.Println("serial send state", nxt.State, "for", nxt.short)
	nxt.send(nxt.page, fmt.Sprintf("%s.txt=%s", nxt.short, nxt.State))
}

func (nxt *NextionText) TouchEvent(state byte) {
	nxt.GetMqttDevice().UpdateState()
}

func (nxt *NextionText) SetStateSerial(state string) {
	nxt.State = state
	log.Println("serial set state", nxt.State, "for", nxt.short)
	nxt.GetMqttDevice().UpdateState()
}

func (nxt *NextionText) GetMqttDevice() ExternalDevice.Device {
	return nxt.Text
}
