module github.com/alf632/gokrazy-ha/display-serial

go 1.21.1

replace github.com/alf632/gokrazy-ha/mqttComponent => /home/malte/Projects/gokrazy-ha/mqttComponent

require (
	github.com/denisbrodbeck/machineid v1.0.1
	github.com/eclipse/paho.mqtt.golang v1.4.3
	github.com/imdario/mergo v0.3.15
)

require (
	dario.cat/mergo v1.0.0 // indirect
	github.com/creack/goselect v0.1.2 // indirect
	github.com/iancoleman/strcase v0.2.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
)

require (
	github.com/W-Floyd/ha-mqtt-iot v0.0.0-20230406181311-8b8c6bf30434
	github.com/alf632/gokrazy-ha/mqttComponent v0.0.0-00010101000000-000000000000
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/plus3it/gorecurcopy v0.0.1
	go.bug.st/serial v1.6.1
	golang.org/x/net v0.9.0 // indirect
	golang.org/x/sync v0.1.0 // indirect
)
