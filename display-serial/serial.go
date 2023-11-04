package main

import (
	"bytes"
	"fmt"
	"log"

	"go.bug.st/serial"
)

// SerialController takes commands from the display to broker them to corresponding mqtt endpoints and vise versa
type SerialController struct {
	port      serial.Port
	sendQueue chan string
	receiver  map[string]func([]byte)
}

func NewSerialController(config Config) *SerialController {
	// Retrieve the port list
	ports, err := serial.GetPortsList()
	if err != nil {
		log.Fatal(err)
	}
	if len(ports) == 0 {
		log.Fatal("No serial ports found!")
	}

	// Print the list of detected ports
	usePort := ports[0]
	for _, port := range ports {
		if port == *config.SerialPort {
			usePort = port
			fmt.Printf("Found port: *%v\n", port)
		} else {
			fmt.Printf("Found port: %v\n", port)
		}
	}

	// Open the first serial port detected at 9600bps N81
	mode := &serial.Mode{
		BaudRate: 38400,
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	}
	port, err := serial.Open(usePort, mode)
	if err != nil {
		log.Fatal(err)
	}

	sc := SerialController{port: port, sendQueue: make(chan string, 10), receiver: map[string]func([]byte){}}

	log.Println("Serial is set up")
	go sc.receive()
	go sc.send()
	return &sc

	// Read and print the response
}

func (sc *SerialController) stop() {
	sc.port.Close()
}

func (sc *SerialController) AddReceiver(header string, callback func([]byte)) {
	sc.receiver[header] = callback
}

func (sc *SerialController) receive() {
	EOF := []byte{0xff, 0xff, 0xff}
	nextBuffer := []byte{}
	for {
		var buff bytes.Buffer
		buff.Write(nextBuffer)
		for {
			buff2 := make([]byte, 100)
			// Reads up to 100 bytes
			n, err := sc.port.Read(buff2)
			if err != nil {
				log.Fatal(err)
			}
			if n == 0 {
				fmt.Println("\nEOF")
				break
			}

			fmt.Printf("%X", buff2[:n])

			buff.Write(buff2[:n])
			// If we receive a nextion delimiter stop reading
			if bytes.Contains(buff.Bytes(), EOF) {
				break
			}
		}

		splitBuff := bytes.Split(buff.Bytes(), EOF)
		// sometimes an incomplete command is left after EOF. prime the next buffer with it
		// if no incomplete command is left, the last element of splitBuff will be empty
		nextBuffer = splitBuff[len(splitBuff)-1]
		for _, msg := range splitBuff[:len(splitBuff)-1] {
			go sc.brokerMessage(msg)
		}
	}
}

func (sc *SerialController) brokerMessage(msg []byte) {
	log.Printf("brokering %X\n", msg)
	if len(msg) == 0 {
		return
	}

	returnCode := fmt.Sprintf("%X", msg[0:1])
	receiver, exists := sc.receiver[returnCode]
	if exists {
		receiver(msg)
	} else {
		log.Println("no receiver found")
	}
}

func (sc *SerialController) send() {
	for {
		data := <-sc.sendQueue
		byteData := []byte(data)
		byteData = append(byteData, []byte{0xff, 0xff, 0xff}...)
		fmt.Println("data:", data)
		fmt.Printf("sending %X\n", byteData)
		n, err := sc.port.Write(byteData)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Sent %v bytes\n", n)
	}
}

func (sc *SerialController) QueueMessage(msg string) {
	sc.sendQueue <- msg
}
