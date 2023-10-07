package main

import (
	"fmt"
	"log"
	"strings"

	"go.bug.st/serial"
)

type SerialHandler interface {
	send(string)
	receive(string)
}

type SerialController struct {
	port      serial.Port
	sendQueue chan string
	receiver  map[string]func(string)
}

func NewSerialController(elements map[string]NextionElement) *SerialController {
	// Retrieve the port list
	ports, err := serial.GetPortsList()
	if err != nil {
		log.Fatal(err)
	}
	if len(ports) == 0 {
		log.Fatal("No serial ports found!")
	}

	// Print the list of detected ports
	for _, port := range ports {
		fmt.Printf("Found port: %v\n", port)
	}

	// Open the first serial port detected at 9600bps N81
	mode := &serial.Mode{
		BaudRate: 9600,
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	}
	port, err := serial.Open(ports[0], mode)
	if err != nil {
		log.Fatal(err)
	}

	sc := SerialController{port: port, sendQueue: make(chan string, 10)}

	receiver := map[string]func(string){}
	for key, element := range elements {
		switch element.GetType() {
		case TypeButton:
			receiver[key] = element.(*NextionButton).SetStateSerial
		case TypeSwitch:
			receiver[key] = element.(*NextionSwitch).SetStateSerial
			element.(*NextionSwitch).RegisterSender(sc.QueueMessage)
		}
	}
	sc.receiver = receiver

	log.Println("Serial is set up")
	go sc.receive()
	go sc.send()
	return &sc

	// Read and print the response
}

func (sc *SerialController) stop() {
	sc.port.Close()
}

func (sc *SerialController) receive() {
	for {
		buff := make([]byte, 100)
		for {
			// Reads up to 100 bytes
			n, err := sc.port.Read(buff)
			if err != nil {
				log.Fatal(err)
			}
			if n == 0 {
				fmt.Println("\nEOF")
				break
			}

			fmt.Printf("%s", string(buff[:n]))

			// If we receive a newline stop reading
			if strings.Contains(string(buff[:n]), "\n") {
				break
			}
		}
		go sc.brokerMessage(string(buff))
	}
}

func (sc *SerialController) brokerMessage(msg string) {
	log.Println("brokering", msg)
	header := msg[:2]
	receiver, exists := sc.receiver[header]
	if exists {
		receiver(msg[2:])
	} else {
		log.Println("receiver not found", header)
	}
}

func (sc *SerialController) send() {
	for {
		data := <-sc.sendQueue
		n, err := sc.port.Write([]byte(data + "\n\r"))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Sent %v bytes\n", n)
	}
}

func (sc *SerialController) QueueMessage(msg string) {
	sc.sendQueue <- msg
}
