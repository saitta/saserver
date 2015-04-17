package telldus

// #cgo CFLAGS:-I/home/leif/Hämtningar/telldus/telldus-core/client
// #cgo LDFLAGS:-ltelldus-core
// #include <stdio.h>
// #include <stdlib.h>
// #include <telldus-core.h>
// #include "cdefs.h"
import "C"

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

var my_callbacks = []C.int{}
var mtdtool *TDTool

func Random() int {
	return int(C.random())
}

func Seed(i int) {
	C.srandom(C.uint(i))
}

var TELLSTICK_TURNON int = int(C.TELLSTICK_TURNON)
var TELLSTICK_TURNOFF int = int(C.TELLSTICK_TURNOFF)
var TELLSTICK_DIM int = int(C.TELLSTICK_DIM)

//export sensorEvent_go
func sensorEvent_go(protocol *C.char, model *C.char, sensorId C.int, dataType C.int, value *C.char, ts C.int, callbackId C.int, context unsafe.Pointer) {
	/*
	   tmpstr := "123456789ABCDEFGHIJKLMNOPQRSTUVXYZ"
	*/
	g_protocol := C.GoString(protocol)
	//g_model := C.GoString(model)
	g_value := C.GoString(value)

	// C.free(unsafe.Pointer(c_protocol))
	// C.free(unsafe.Pointer(c_model))
	// C.free(unsafe.Pointer(c_value))

	//Retrieve the values the sensor supports
	if dataType == C.TELLSTICK_TEMPERATURE {
		//strftime(timeBuf, sizeof(timeBuf), "%Y-%m-%d %H:%M:%S", localtime(&timestamp));
		//printf("Temperature:\t%sº\t(%s)\n", value, timeBuf);
		tmpval, err := strconv.ParseFloat(g_value, 64)
		if err == nil {
			t2 := time.Now().Unix()
			if math.Abs(float64(t2-int64(ts))) > 600 {
				log.Printf("value is more than 10 min. off %d\n", t2-int64(ts))
			} else {
				dev, ok := (*mtdtool)[g_protocol]
				if !ok {
					thisDevice := Device{}
					thisDevice.Id = int(sensorId)
					thisDevice.Name = g_protocol
					thisDevice.Type = SENSOR
					thisDevice.Status = 0
					thisDevice.Dimlevel = 0
					thisDevice.Value = tmpval
					(*mtdtool)[thisDevice.Name] = &thisDevice
				} else {
					dev.Value = tmpval
				}
			}
		}
	}
}

//export deviceEvent_go
func deviceEvent_go(deviceId C.int, method C.int, data *C.char, callbackId C.int, context unsafe.Pointer) {
	g_data := C.GoString(data)
	name := C.GoString(C.tdGetName(deviceId))
	dev, ok := (*mtdtool)[name]
	if !ok {
		log.Printf("cannot find device(%s)\n", name)
	} else {
		log.Printf("device(%s) data(%s)\n", name, g_data)
		switch int(method) {
		case TELLSTICK_TURNON:
			dev.Status = 1
			dev.Dimlevel = 255
		case TELLSTICK_TURNOFF:
			dev.Status = 0
			dev.Dimlevel = 0
		case TELLSTICK_DIM:
			val, err := strconv.Atoi(g_data)
			if err == nil {
				dev.Dimlevel = val & 0xFF
				dev.Status = dev.Dimlevel
			}
		default:
			fmt.Printf("unknown event  from device %i  data(%s)\n", deviceId, g_data)
		}
		//(*mtdtool)[name] = dev
	}
}

//globals , trying to protect from garbage collection

var sensorEventFuncPointer = C.sensorEvent_cgo
var deviceEventFuncPointer = C.deviceEvent_cgo

//*C.TDSensorEvent
func init() {
	var dummy int
	var dummy2 int
	C.tdInit()
	mtdtool = NewTDTool()

	callbackId := C.tdRegisterSensorEvent((C.callback_fcn_sensor)(unsafe.Pointer(sensorEventFuncPointer)), unsafe.Pointer(&dummy))
	my_callbacks = append(my_callbacks, callbackId)
	callbackId = C.tdRegisterDeviceEvent((C.callback_fcn_device)(unsafe.Pointer(deviceEventFuncPointer)), unsafe.Pointer(&dummy2))
	my_callbacks = append(my_callbacks, callbackId)
}

func Cleanup() {
	for id := range my_callbacks {
		fmt.Printf("unregistering: %d\n", id)
		C.tdUnregisterCallback(C.int(id))
	}
	C.tdClose()
}

func GetTDTool() *TDTool {
	return mtdtool
}

func NewTDTool() *TDTool {

	my_devices := make(TDTool)
	var intNumberOfDevices int
	var id C.int

	intNumberOfDevices = int(C.tdGetNumberOfDevices())
	//Correct
	for i := 0; i < intNumberOfDevices; i++ {
		id = C.tdGetDeviceId(C.int(i))
		name := C.tdGetName(id)

		thisDevice := Device{}
		thisDevice.Id = int(id)
		thisDevice.Name = C.GoString(name)
		thisDevice.Type = SWITCH
		thisDevice.Status = 0
		thisDevice.Dimlevel = 0

		if strings.Contains(thisDevice.Name, "detector") {
			thisDevice.Type = DETECTOR
		}

		methods := int(C.tdMethods(id, C.TELLSTICK_TURNON|C.TELLSTICK_TURNOFF|C.TELLSTICK_DIM))
		if (methods & TELLSTICK_DIM) > 0 {
			//res , err := C.tdDim(id,C.uchar(128))
			tmp_char := C.tdLastSentValue(id)
			thisDevice.Dimlevel, _ = strconv.Atoi(C.GoString(tmp_char))
			C.tdReleaseString(tmp_char)
			thisDevice.Type = DIMMER
		} else {
			//assume switch
			val := int(C.tdLastSentCommand(id, C.TELLSTICK_TURNON|C.TELLSTICK_TURNOFF))
			if (val & TELLSTICK_TURNON) > 0 {
				thisDevice.Status = 1
			}
		}
		C.tdReleaseString(name)
		my_devices[thisDevice.Name] = &thisDevice
	}

	fmt.Printf("NDEV(%d)\n", intNumberOfDevices)
	/*
	   tdSensor	(	char * 	protocol,
	   int 	protocolLen,
	   char * 	model,
	   int 	modelLen,
	   int * 	id,
	   int * 	dataTypes
	   )
	*/
	tmpstr := "123456789ABCDEFGHIJKLMNOPQRSTUVXYZ"
	c_protocol := C.CString(tmpstr)
	c_model := C.CString(tmpstr)
	c_value := C.CString(tmpstr)
	var dataTypes C.int
	var t1 C.int

	for {
		res := C.tdSensor(c_protocol, C.int(len(tmpstr)), c_model, C.int(len(tmpstr)), &id, &dataTypes)
		if res == C.TELLSTICK_SUCCESS {
			fmt.Printf("Got sensor %s \n", C.GoString(c_protocol))
			thisDevice := Device{}
			thisDevice.Id = int(id)
			thisDevice.Name = C.GoString(c_protocol)
			thisDevice.Type = SENSOR
			thisDevice.Status = 0
			thisDevice.Dimlevel = 0
			thisDevice.Value = 0.0

			if (dataTypes & C.TELLSTICK_TEMPERATURE) > 0 {
				/*
				   int WINAPI tdSensorValue	(	const char * 	protocol,
				   const char * 	model,
				   int 	id,
				   int 	dataType,
				   char * 	value,
				   int 	len,
				   int * 	timestamp
				   )
				*/
				res = C.tdSensorValue(c_protocol, c_model, id, C.TELLSTICK_TEMPERATURE, c_value, C.int(len(tmpstr)), &t1)
				if res == C.TELLSTICK_SUCCESS {
					tmpval, err := strconv.ParseFloat(C.GoString(c_value), 64)
					if err == nil {
						t2 := time.Now().Unix()
						if math.Abs(float64(t2-int64(t1))) > 600 {
							fmt.Printf("value is more than 10 min. off %d\n", t2-int64(t1))
						} else {
							thisDevice.Value = tmpval
						}
					}
				}
			}

			my_devices[thisDevice.Name] = &thisDevice
		} else {
			break
		}
	}
	C.free(unsafe.Pointer(c_protocol))
	C.free(unsafe.Pointer(c_model))
	C.free(unsafe.Pointer(c_value))

	return &my_devices
}

const (
	SWITCH   int = 0
	DIMMER   int = 1
	SENSOR   int = 2
	DETECTOR int = 3
	TURN_OFF int = 0
	TURN_ON  int = 1
	DIM      int = 2
)

type Device struct {
	Id          int
	Name        string
	Description string
	Type        int
	Status      int
	Action      int
	Dimlevel    int
	Value       float64
	Unit        string
}

func (d *Device) exec(action []string) string {
	return "hello"
}

func (d *Device) String() string {
	return fmt.Sprintf("%s id(%d) type(%d) action(%d) status(%d) ", d.Name, d.Id, d.Type, d.Action, d.Status)
}

func (d *Device) Dim(level int) string {
	d.Status = level
	d.Dimlevel = level
	C.tdDim(C.int(d.Id), C.uchar(level))
	return d.Name
}

func (d *Device) On() string {
	d.Status = 1
	C.tdTurnOn(C.int(d.Id))
	return d.Name
}

func (d *Device) Off() string {
	d.Status = 0
	C.tdTurnOff(C.int(d.Id))
	return d.Name
}

func (d *Device) PerformAction() {
	var res string

	switch d.Action {
	case TURN_ON:
		res = d.On()
	case TURN_OFF:
		res = d.Off()
	case DIM:
		res = d.Dim(d.Dimlevel)

	default:
		fmt.Println("Unknown action, should return error")
	}
	fmt.Println(res)
}

type TDTool map[string]*Device
