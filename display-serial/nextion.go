package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"sync"

	"github.com/alf632/gokrazy-ha/mqttComponent"
)

type NextionController struct {
	sc             *SerialController
	mc             *mqttComponent.MqttController
	pages          map[int]*NextionPage
	currentPage    int
	getTxtReceiver func(string)
	getLock        sync.Mutex
	getValReceiver func(uint32)
}

type NextionPage struct {
	ID           int
	Elements     map[string]NextionElement
	ElementIndex map[uint32]string
}

func NewNextionController(sc *SerialController, mc *mqttComponent.MqttController) *NextionController {
	newNextionController := NextionController{sc: sc, mc: mc, pages: map[int]*NextionPage{}}
	newNextionController.RegisterReceiver()
	return &newNextionController
}

func (nc *NextionController) SendState(page int, msg string) {
	if page == nc.currentPage {
		nc.sc.QueueMessage(msg)
	}
}

func (nc *NextionController) BroadcastPage(pageIdx int) {
	page, exists := nc.pages[pageIdx]
	if !exists {
		nc.CreatePage(pageIdx)
	} else {
		for _, element := range page.Elements {
			element.SendStateSerial()
		}
	}
}

func (nc *NextionController) CreatePage(pageIdx int) {
	newPage := NextionPage{
		ID:           pageIdx,
		Elements:     map[string]NextionElement{},
		ElementIndex: map[uint32]string{},
	}
	nc.pages[pageIdx] = &newPage
	nc.QueryPage(pageIdx)
}

func (nc *NextionController) QueryPage(page int) {
	nc.discoverButtons(page, 0)
	nc.discoverDualStateButtons(page, 0)
	nc.discoverSwitches(page, 0)
	nc.discoverTexts(page, 0)
}

// discoverButtons tries to get the text of button b[idx].
// if data is returned from the display and the callback is fired we
// recurively call this method for the next button.
func (nc *NextionController) discoverButtons(page, idx int) {
	short := fmt.Sprintf("b%d", idx)
	nc.GetTxt(short, func(txt string) {
		log.Println("discovered Button", short)
		newButton := newNextionButton(txt, short, page, nc.SendState)
		nc.mc.AddDevice(newButton.GetMqttDevice())
		nc.pages[page].Elements[short] = newButton
		nc.GetID(short, func(ID uint32) {
			newButton.ID = ID
			nc.pages[page].ElementIndex[ID] = short
		})
		nc.GetVal(short, func(val uint32) {
			if val > 0 {
				newButton.SetStateSerial("ON")
			} else {
				newButton.SetStateSerial("OFF")
			}
		})

		idx++
		nc.discoverButtons(page, idx)
	})
}

func (nc *NextionController) discoverDualStateButtons(page, idx int) {
	short := fmt.Sprintf("bt%d", idx)
	nc.GetTxt(short, func(txt string) {
		log.Println("discovered DualStateButton", short)
		newSwitch := newNextionSwitch(txt, short, page, nc.SendState)
		nc.mc.AddDevice(newSwitch.GetMqttDevice())
		nc.pages[page].Elements[short] = newSwitch
		nc.GetID(short, func(ID uint32) {
			newSwitch.ID = ID
			nc.pages[page].ElementIndex[ID] = short
		})
		nc.GetVal(short, func(val uint32) {
			if val > 0 {
				newSwitch.SetStateSerial("ON")
			} else {
				newSwitch.SetStateSerial("OFF")
			}
		})

		idx++
		nc.discoverDualStateButtons(page, idx)
	})
}

func (nc *NextionController) discoverSwitches(page, idx int) {
	short := fmt.Sprintf("s%d", idx)
	nc.GetTxt(short, func(txt string) {
		log.Println("discovered Switch", short)
		newSwitch := newNextionSwitch(txt, short, page, nc.SendState)
		nc.mc.AddDevice(newSwitch.GetMqttDevice())
		nc.pages[page].Elements[short] = newSwitch
		nc.GetID(short, func(ID uint32) {
			newSwitch.ID = ID
			nc.pages[page].ElementIndex[ID] = short
		})
		nc.GetVal(short, func(val uint32) {
			if val > 0 {
				newSwitch.SetStateSerial("ON")
			} else {
				newSwitch.SetStateSerial("OFF")
			}
		})

		idx++
		nc.discoverSwitches(page, idx)
	})
}

func (nc *NextionController) discoverTexts(page, idx int) {
	short := fmt.Sprintf("t%d", idx)
	nc.GetTxt(short, func(txt string) {
		log.Println("discovered Text", short)
		newText := newNextionText("Text "+short, short, page, nc.SendState)
		nc.mc.AddDevice(newText.GetMqttDevice())
		nc.pages[page].Elements[short] = newText
		nc.GetID(short, func(ID uint32) {
			newText.ID = ID
			nc.pages[page].ElementIndex[ID] = short
		})
		newText.SetStateSerial(txt)

		idx++
		nc.discoverTexts(page, idx)
	})
}

func (nc *NextionController) GetTxt(component string, callback func(string)) {
	nc.getLock.Lock()
	nc.getTxtReceiver = callback
	nc.sc.QueueMessage(fmt.Sprintf("get %s.txt", component))
}

func (nc *NextionController) GetVal(component string, callback func(uint32)) {
	nc.getLock.Lock()
	nc.getValReceiver = callback
	nc.sc.QueueMessage(fmt.Sprintf("get %s.val", component))
}

func (nc *NextionController) GetID(component string, callback func(uint32)) {
	nc.getLock.Lock()
	nc.getValReceiver = callback
	nc.sc.QueueMessage(fmt.Sprintf("get %s.id", component))
}

func (nc *NextionController) SetTxt(component, txt string) {
	nc.sc.QueueMessage(fmt.Sprintf("%s.txt=%s", component, txt))
}

func (nc *NextionController) SetVal(component string, val int) {
	nc.sc.QueueMessage(fmt.Sprintf("%s.val=%d", component, val))
}

func (nc *NextionController) RegisterReceiver() {
	nc.sc.AddReceiver(fmt.Sprintf("%X", 0x00), nc.ReceiveInvalidInstruction)
	nc.sc.AddReceiver(fmt.Sprintf("%X", 0x01), nc.ReceiveInstructionSuccessful)
	nc.sc.AddReceiver(fmt.Sprintf("%X", 0x02), nc.ReceiveInvalidComponentID)
	nc.sc.AddReceiver(fmt.Sprintf("%X", 0x03), nc.ReceiveInvalidPageID)
	nc.sc.AddReceiver(fmt.Sprintf("%X", 0x04), nc.ReceiveInvalidPictureID)
	nc.sc.AddReceiver(fmt.Sprintf("%X", 0x05), nc.ReceiveInvalidFontID)
	nc.sc.AddReceiver(fmt.Sprintf("%X", 0x06), nc.ReceiveInvalidFileOperation)
	nc.sc.AddReceiver(fmt.Sprintf("%X", 0x09), nc.ReceiveInvalidCRC)
	nc.sc.AddReceiver(fmt.Sprintf("%X", 0x11), nc.ReceiveInvalidBaudrateSetting)
	nc.sc.AddReceiver(fmt.Sprintf("%X", 0x12), nc.ReceiveInvalidWaveformIDOrChannel)
	nc.sc.AddReceiver(fmt.Sprintf("%X", 0x1A), nc.ReceiveInvalidVariableNameOrAttribute)
	nc.sc.AddReceiver(fmt.Sprintf("%X", 0x1B), nc.ReceiveInvalidVariableOperation)
	nc.sc.AddReceiver(fmt.Sprintf("%X", 0x1C), nc.ReceiveAssignmentFailedToAssign)
	nc.sc.AddReceiver(fmt.Sprintf("%X", 0x1D), nc.ReceiveEEPROMOperationFailed)
	nc.sc.AddReceiver(fmt.Sprintf("%X", 0x1E), nc.ReceiveInvalidQuantityOfParameters)
	nc.sc.AddReceiver(fmt.Sprintf("%X", 0x1F), nc.ReceiveIOOperationFailed)
	nc.sc.AddReceiver(fmt.Sprintf("%X", 0x20), nc.ReceiveEscapeCharacterInvalid)
	nc.sc.AddReceiver(fmt.Sprintf("%X", 0x23), nc.ReceiveVariableNameTooLong)
	nc.sc.AddReceiver(fmt.Sprintf("%X", 0x00), nc.ReceiveNextionStartup)
	nc.sc.AddReceiver(fmt.Sprintf("%X", 0x24), nc.ReceiveSerialBufferOverflow)
	nc.sc.AddReceiver(fmt.Sprintf("%X", 0x65), nc.ReceiveTouchEvent)
	nc.sc.AddReceiver(fmt.Sprintf("%X", 0x66), nc.ReceiveCurrentPageNumber)
	nc.sc.AddReceiver(fmt.Sprintf("%X", 0x67), nc.ReceiveTouchCoordinateAwake)
	nc.sc.AddReceiver(fmt.Sprintf("%X", 0x68), nc.ReceiveTouchCoordinateSleep)
	nc.sc.AddReceiver(fmt.Sprintf("%X", 0x70), nc.ReceiveStringDataEnclosed)
	nc.sc.AddReceiver(fmt.Sprintf("%X", 0x71), nc.ReceiveNumericDataEnclosed)
	nc.sc.AddReceiver(fmt.Sprintf("%X", 0x86), nc.ReceiveAutoEnteredSleepMode)
	nc.sc.AddReceiver(fmt.Sprintf("%X", 0x87), nc.ReceiveAutoWakeFromSleep)
	nc.sc.AddReceiver(fmt.Sprintf("%X", 0x88), nc.ReceiveNextionReady)
	nc.sc.AddReceiver(fmt.Sprintf("%X", 0x89), nc.ReceiveStartMicroSDUpgrade)
	nc.sc.AddReceiver(fmt.Sprintf("%X", 0xFD), nc.ReceiveTransparentDataFinished)
	nc.sc.AddReceiver(fmt.Sprintf("%X", 0xFE), nc.ReceiveTransparentDataReady)
}

func (nc *NextionController) ReceiveInvalidInstruction(msg []byte) {
	// Example 0x00 0xFF 0xFF 0xFF
	log.Println("instruction sent by user has failed")
	nc.getLock.Unlock()
}
func (nc *NextionController) ReceiveInstructionSuccessful(msg []byte) {
	// Example 0x01 0xFF 0xFF 0xFF
	log.Println("instruction sent by user was successful")
	nc.getLock.Unlock()
}
func (nc *NextionController) ReceiveInvalidComponentID(msg []byte) {
	// Example 0x02 0xFF 0xFF 0xFF
	log.Println("invalid Component ID or name was used")
	nc.getLock.Unlock()
}
func (nc *NextionController) ReceiveInvalidPageID(msg []byte) {
	// Example 0x03 0xFF 0xFF 0xFF
	log.Println("invalid Page ID or name was used")
	nc.getLock.Unlock()
}
func (nc *NextionController) ReceiveInvalidPictureID(msg []byte) {
	// Example 0x04 0xFF 0xFF 0xFF
	log.Println("invalid Picture ID was used")
	nc.getLock.Unlock()
}
func (nc *NextionController) ReceiveInvalidFontID(msg []byte) {
	// Example 0x05 0xFF 0xFF 0xFF
	log.Println("invalid Font ID was used")
	nc.getLock.Unlock()
}
func (nc *NextionController) ReceiveInvalidFileOperation(msg []byte) {
	// Example 0x06 0xFF 0xFF 0xFF
	log.Println("File operation fails")
	nc.getLock.Unlock()
}
func (nc *NextionController) ReceiveInvalidCRC(msg []byte) {
	// Example 0x09 0xFF 0xFF 0xFF
	log.Println("Instructions with CRC validation fails their CRC check")
	nc.getLock.Unlock()
}
func (nc *NextionController) ReceiveInvalidBaudrateSetting(msg []byte) {
	// Example 0x11 0xFF 0xFF 0xFF
	log.Println("invalid Baud rate was used")
	nc.getLock.Unlock()
}
func (nc *NextionController) ReceiveInvalidWaveformIDOrChannel(msg []byte) {
	// Example 0x12 0xFF 0xFF 0xFF
	log.Println("invalid Waveform ID or Channel # was used")
	nc.getLock.Unlock()
}
func (nc *NextionController) ReceiveInvalidVariableNameOrAttribute(msg []byte) {
	// Example 0x1A 0xFF 0xFF 0xFF
	log.Println("invalid Variable name or invalid attribute was used")
	nc.getLock.Unlock()
}
func (nc *NextionController) ReceiveInvalidVariableOperation(msg []byte) {
	// Example 0x1B 0xFF 0xFF 0xFF
	log.Println("Operation of Variable is invalid. ie: Text assignment t0.txt=abc or t0.txt=23, Numeric assignment j0.val=”50″ or j0.val=abc")
	nc.getLock.Unlock()
}
func (nc *NextionController) ReceiveAssignmentFailedToAssign(msg []byte) {
	// Example 0x1C 0xFF 0xFF 0xFF
	log.Println("attribute assignment failed to assign")
	nc.getLock.Unlock()
}
func (nc *NextionController) ReceiveEEPROMOperationFailed(msg []byte) {
	// Example 0x1D 0xFF 0xFF 0xFF
	log.Println("an EEPROM Operation has failed")
	nc.getLock.Unlock()
}
func (nc *NextionController) ReceiveInvalidQuantityOfParameters(msg []byte) {
	// Example 0x1E 0xFF 0xFF 0xFF
	log.Println("the number of instruction parameters is invalid")
	nc.getLock.Unlock()
}
func (nc *NextionController) ReceiveIOOperationFailed(msg []byte) {
	// Example 0x1F 0xFF 0xFF 0xFF
	log.Println("an IO operation has failed")
	nc.getLock.Unlock()
}
func (nc *NextionController) ReceiveEscapeCharacterInvalid(msg []byte) {
	// Example 0x20 0xFF 0xFF 0xFF
	log.Println("an unsupported escape character is used")
	nc.getLock.Unlock()
}
func (nc *NextionController) ReceiveVariableNameTooLong(msg []byte) {
	// Example 0x23 0xFF 0xFF 0xFF
	log.Println("variable name is too long. Max length is 29 characters: 14 for page + “.” + 14 for component.")
	nc.getLock.Unlock()
}

func (nc *NextionController) ReceiveNextionStartup(msg []byte) {
	// Example 0x00 0x00 0x00 0xFF 0xFF 0xFF
	// Since Nextion Editor v1.65.0, the Startup preamble is not at the firmware level but has been moved to a printh statement in Program.s allowing a user to keep, modify or remove as they choose.
	log.Println("Nextion has started or reset")
}
func (nc *NextionController) ReceiveSerialBufferOverflow(msg []byte) {
	// Example 0x24 0xFF 0xFF 0xFF
	// Buffer will continue to receive the current instruction, all previous instructions are lost.
	log.Println("a Serial Buffer overflow occured")
}
func (nc *NextionController) ReceiveTouchEvent(msg []byte) {
	// Example 0x65 0x00 0x01 0x01 0xFF 0xFF 0xFF
	// Touch occurs and component’s
	// corresponding Send Component ID is checked
	// in the users HMI design.
	// 0x00 is page number, 0x01 is component ID,
	// 0x01 is event (0x01 Press and 0x00 Release)
	// data: Page 0, Component 1, Pressed
	log.Printf("touch event. page:%b component:%b event:%d", msg[1], msg[2], msg[3])
	page, exists := nc.pages[int(msg[1])]
	if !exists {
		log.Println("cannot find page")
		return
	}

	short, exists := page.ElementIndex[uint32(msg[2])]
	if !exists {
		log.Println("cannot find element")
		return
	}

	element, exists := page.Elements[short]
	if !exists {
		log.Println("cannot find element with this ID")
		return
	}

	element.TouchEvent(msg[3])
}
func (nc *NextionController) ReceiveCurrentPageNumber(msg []byte) {
	// Example 0x66 0x01 0xFF 0xFF 0xFF
	// the sendme command is used.
	// 0x01 is current page number
	// data: page 1
	nc.currentPage = int(msg[1])
	log.Println("CurrentPageNumber", msg[1])
	nc.BroadcastPage(nc.currentPage)
}
func (nc *NextionController) ReceiveTouchCoordinateAwake(msg []byte) {
	// Example 0x67 0x00 0x7A 0x00 0x1E 0x01 0xFF 0xFF 0xFF
	// sendxy=1 and not in sleep mode
	// 0x00 0x7A is x coordinate in big endian order,
	// 0x00 0x1E is y coordinate in big endian order,
	// 0x01 is event (0x01 Press and 0x00 Release)
	// (0x00*256+0x71,0x00*256+0x1E)
	// data: (122,30) Pressed
	log.Println("TouchCoordinateAwake", msg[1:3], msg[3:5], msg[5])
}
func (nc *NextionController) ReceiveTouchCoordinateSleep(msg []byte) {
	// Example 0x68 0x00 0x7A 0x00 0x1E 0x01 0xFF 0xFF 0xFF
	// sendxy=1 and exiting sleep
	// 0x00 0x7A is x coordinate in big endian order,
	// 0x00 0x1E is y coordinate in big endian order,
	// 0x01 is event (0x01 Press and 0x00 Release)
	// (0x00*256+0x71,0x00*256+0x1E)
	// data: (122,30) Pressed
	log.Println("TouchCoordinateSleep", msg[1:3], msg[3:5], msg[5])
}
func (nc *NextionController) ReceiveStringDataEnclosed(msg []byte) {
	// Example 0x70 0x61 0x62 0x31 0x32 0x33 0xFF 0xFF 0xFF
	// using get command for a string.
	// Each byte is converted to char.
	// data: ab123
	str := string(msg[1:])
	log.Println("StringDataEnclosed", str)
	go nc.getTxtReceiver(str)
	nc.getLock.Unlock()
}
func (nc *NextionController) ReceiveNumericDataEnclosed(msg []byte) {
	// Example 0x71 0x01 0x02 0x03 0x04 0xFF 0xFF 0xFF
	// get command to return a number
	// 4 byte 32-bit value in little endian order.
	// (0x01+0x02*256+0x03*65536+0x04*16777216)
	// data: 67305985
	val := binary.LittleEndian.Uint32(msg[1:5])
	log.Println("NumericDataEnclosed", val)
	go nc.getValReceiver(val)
	nc.getLock.Unlock()
}
func (nc *NextionController) ReceiveAutoEnteredSleepMode(msg []byte) {
	// Example 0x86 0xFF 0xFF 0xFF
	// Nextion enters sleep automatically
	// Using sleep=1 will not return an 0x86
	log.Println("AutoEnteredSleepMode")
}
func (nc *NextionController) ReceiveAutoWakeFromSleep(msg []byte) {
	// Example 0x87 0xFF 0xFF 0xFF
	// Nextion leaves sleep automatically
	// Using sleep=0 will not return an 0x87
	log.Println("AutoWakeFromSleep")
}
func (nc *NextionController) ReceiveNextionReady(msg []byte) {
	// Example 0x88 0xFF 0xFF 0xFF
	// Nextion has powered up and is now initialized successfully. Since Nextion Editor v1.65.0, the Nextion Ready is not at the firmware level but has been moved to a printh statement in Program.s allowing a user to keep, modify or remove as they choose.
	log.Println("NextionReady")
}
func (nc *NextionController) ReceiveStartMicroSDUpgrade(msg []byte) {
	// Example 0x89 0xFF 0xFF 0xFF
	// power on detects inserted microSD
	// and begins Upgrade by microSD process
	log.Println("StartMicroSDUpgrade")
}
func (nc *NextionController) ReceiveTransparentDataFinished(msg []byte) {
	// Example 0xFD 0xFF 0xFF 0xFF
	// all requested bytes of Transparent
	// Data mode have been received, and is now leaving transparent data mode (see 1.16)
	log.Println("TransparentDataFinished")
}
func (nc *NextionController) ReceiveTransparentDataReady(msg []byte) {
	// Example 0xFE 0xFF 0xFF 0xFF
	// requesting Transparent Data
	// mode, and device is now ready to begin receiving
	// the specified quantity of data (see 1.16)
	log.Println("TransparentDataReady")
}
