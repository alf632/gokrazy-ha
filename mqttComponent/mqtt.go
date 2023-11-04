package mqttComponent

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/url"
	"os"
	"strings"
	"time"

	"dario.cat/mergo"
	"github.com/W-Floyd/ha-mqtt-iot/common"
	"github.com/W-Floyd/ha-mqtt-iot/config"
	ExternalDevice "github.com/W-Floyd/ha-mqtt-iot/devices/externaldevice"
	"github.com/denisbrodbeck/machineid"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const float64EqualityThreshold = 1e-9

type MQTTConfig struct {
	ConfigFile  *string
	SecretsFile *string
}

type MqttController struct {
	client  mqtt.Client
	tickers []*time.Ticker
	devices []ExternalDevice.Device
}

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) <= float64EqualityThreshold
}

func NewMqttController(mqttConfig MQTTConfig) *MqttController {
	id, err := machineid.ID()
	if err != nil {
		log.Fatal(err)
	}

	common.MachineID = id

	configFiles := [...]string{*mqttConfig.ConfigFile, *mqttConfig.SecretsFile}

	var sconfig config.Config

	for _, configFile := range configFiles {
		var tConfig config.Config

		// read file
		data, err := os.ReadFile(configFile)
		if err != nil {
			common.LogError("Error reading "+configFile, err)
		}

		d := json.NewDecoder(strings.NewReader(string(data)))
		d.DisallowUnknownFields()

		// unmarshall it
		err = d.Decode(&tConfig)
		if err != nil {
			common.LogError("Error parsing config", err)
		}

		mergo.Merge(&sconfig, tConfig)

	}

	devices, opts := sconfig.Convert()
	//devices = append(devices, myDevices...)

	if sconfig.Logging.Debug && sconfig.Logging.Mqtt {
		mqtt.DEBUG = common.DebugLog
	}
	if sconfig.Logging.Warn {
		mqtt.WARN = common.WarnLog
	}
	if sconfig.Logging.Error {
		mqtt.ERROR = common.ErrorLog
	}
	if sconfig.Logging.Critical {
		mqtt.CRITICAL = common.CriticalLog
	}

	common.LogState.Debug = sconfig.Logging.Debug
	common.LogState.Warn = sconfig.Logging.Warn
	common.LogState.Error = sconfig.Logging.Error
	common.LogState.Critical = sconfig.Logging.Critical

	newMqttController := &MqttController{tickers: []*time.Ticker{}}

	opts.SetOnConnectHandler(
		func(c mqtt.Client) {
			log.Println("connected")
			for _, d := range newMqttController.GetDevices() {
				common.LogDebug("Subscribing " + d.GetRawId() + "." + d.GetUniqueId())
				go d.Subscribe()
			}
		},
	)
	opts.SetConnectionAttemptHandler(
		func(broker *url.URL, tlsCfg *tls.Config) *tls.Config {
			log.Printf("attemting connection to %s...\n", broker)
			return tlsCfg
		},
	)

	log.Println("initializing mqtt client with ops")
	log.Printf("%v+", opts)
	log.Println("keepalive", opts.KeepAlive)
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		common.LogError(token.Error())
	}
	newMqttController.client = client
	// wait for client to connect
	for !client.IsConnectionOpen() {
		log.Println("waiting for client to connect")
		time.Sleep(time.Second)
	}
	log.Println("mqtt client initialized")

	for _, d := range devices {
		newMqttController.AddDevice(d)
	}

	client.Subscribe("homeassistant/status", 0, func(c mqtt.Client, m mqtt.Message) {
		log.Println("homeassistant status", string(m.Payload()))
		if string(m.Payload()) == "online" {
			log.Println("homeassistant started")
			for _, d := range devices {
				switch dev := d.(type) {
				case *ExternalDevice.Switch:
					dev.AnnounceAvailable()
				case *ExternalDevice.BinarySensor:
					dev.AnnounceAvailable()
				default:
					fmt.Printf("I don't know about type %T!\n", dev)
				}
			}
		}
	})

	common.LogDebug("MQTT is set up")

	return newMqttController

}

func (mc *MqttController) GetDevices() []ExternalDevice.Device {
	return mc.devices
}

func (mc *MqttController) AddDevice(device ExternalDevice.Device) {
	f := device.GetMQTTFields()
	f.Client = &mc.client
	device.SetMQTTFields(f)

	if device.GetMQTTFields().UpdateInterval != nil && !almostEqual(*device.GetMQTTFields().UpdateInterval, 0) {
		newTicker := time.NewTicker(time.Duration(*device.GetMQTTFields().UpdateInterval) * time.Second)
		mc.tickers = append(mc.tickers, newTicker)
		go func(t *time.Ticker, device ExternalDevice.Device) {
			for range t.C {
				go device.UpdateState()
			}
		}(newTicker, device)
	}
	mc.devices = append(mc.devices, device)
	common.LogDebug("Connecting " + device.GetRawId() + "." + device.GetUniqueId())
	go device.Subscribe()
	common.LogDebug(fmt.Sprintf("Added Device %v+", device))
}

func (mc *MqttController) Stop() {
	for _, t := range mc.tickers {
		t.Stop()
	}
	common.LogDebug("Server Stopped")

	for _, d := range mc.devices {
		d.UnSubscribe()
		common.LogDebug(d.GetRawId() + " Unsubscribed")
	}
	common.LogDebug("All Devices Unsubscribed")

	mc.client.Disconnect(250)

	time.Sleep(1 * time.Second)
}
