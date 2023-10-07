package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	ExternalDevice "github.com/W-Floyd/ha-mqtt-iot/devices/externaldevice"
	"github.com/racerxdl/go-mcp23017"
)

func main() {
	d, err := mcp23017.Open(1, 0)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
	defer d.Close()

	relais, err := setupRelais(d)
	if err != nil {
		log.Fatal(err)
	}
	devices := []ExternalDevice.Device{}
	for _, r := range relais {
		devices = append(devices, r.GetMqttDevice())
	}

	mqttc := NewMqttController(devices)
	defer mqttc.Stop()

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Everything is set up")

	signal := <-done
	log.Println("Exiting with signal", signal.String())

}

func setupRelais(d *mcp23017.Device) ([]*Relais, error) {
	relais := []*Relais{}
	for i := 0; i < 9; i++ {
		if err := d.PinMode(uint8(i), mcp23017.OUTPUT); err != nil {
			return relais, err
		}
		relais = append(relais, NewRelay(uint8(i), d))
	}
	return relais, nil

}
