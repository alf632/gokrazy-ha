package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/alf632/gokrazy-ha/mqttComponent"
)

type Config struct {
	MQTT       mqttComponent.MQTTConfig
	SerialPort *string
}

func main() {
	configFile := flag.String("config", "/perm/nextion/config.json", "path to config file")
	secretsFile := flag.String("secrets", "/perm/nextion/secrets.json", "path to secrets file")
	serialPort := flag.String("port", "/dev/ttyS0", "path to tty interface")
	flag.Parse()
	config := Config{
		MQTT: mqttComponent.MQTTConfig{
			ConfigFile:  configFile,
			SecretsFile: secretsFile,
		},
		SerialPort: serialPort,
	}

	/*
		if _, err := os.Stat("/perm/nextion/"); os.IsNotExist(err) {
			if err := gorecurcopy.CopyDirectory("/opt/nextion/", "/perm/nextion/"); err != nil {
				log.Fatal(err)
			}
		}*/
	nc := NewNextionController(NewSerialController(config), mqttComponent.NewMqttController(config.MQTT))
	defer nc.mc.Stop()
	defer nc.sc.stop()

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Everything is set up")

	signal := <-done
	log.Println("Exiting with signal", signal.String())
}
