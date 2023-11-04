package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/alf632/gokrazy-ha/mqttComponent"
	"github.com/plus3it/gorecurcopy"
	"github.com/racerxdl/go-mcp23017"
)

func main() {
	configFile := flag.String("config", "/perm/goMqttGpio/config.json", "path to config file")
	secretsFile := flag.String("secrets", "/perm/goMqttGpio/secrets.json", "path to secrets file")
	flag.Parse()
	config := mqttComponent.MQTTConfig{
		ConfigFile:  configFile,
		SecretsFile: secretsFile,
	}

	if _, err := os.Stat("/perm/goMqttGpio/"); os.IsNotExist(err) {
		if err := gorecurcopy.CopyDirectory("/opt/goMqttGpio/", "/perm/goMqttGpio/"); err != nil {
			log.Fatal(err)
		}
	}

	d, err := mcp23017.Open(1, 0)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
	defer d.Close()

	log.Println("initializing mqtt controller")
	mqttc := mqttComponent.NewMqttController(config)
	defer mqttc.Stop()
	log.Println("mqtt controller initialized")

	relais, err := setupRelais(d)
	if err != nil {
		log.Fatal(err)
	}
	for _, r := range relais {
		mqttc.AddDevice(r.GetMqttDevice())
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Everything is set up")

	signal := <-done
	log.Println("Exiting with signal", signal.String())

}

func setupRelais(d *mcp23017.Device) ([]*Relais, error) {
	log.Println("setting up relais")
	relais := []*Relais{}
	for i := 0; i < 9; i++ {
		if err := d.PinMode(uint8(i), mcp23017.OUTPUT); err != nil {
			return relais, err
		}
		relais = append(relais, NewRelay(uint8(i), d))
	}
	return relais, nil

}
