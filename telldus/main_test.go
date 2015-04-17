package telldus

import (
	"fmt"
	"testing"
)

func otherFunc(devE <-chan *Device, senE <-chan *Sensor) {
	for {
		select {
		case d := <-devE:
			fmt.Println(d.String())
		case s := <-senE:
			fmt.Println(s)
		}
	}
}

func TestMainTelldus2(t *testing.T) {
	dev_events := make(chan *Device)
	sen_events := make(chan *Sensor)
	evr := &ClientEvent{Fname: "/tmp/TelldusEvents", Device_events: dev_events, Sensor_events: sen_events}
	go evr.Read()
	go otherFunc(dev_events, sen_events)

	c := &Client{fname: "/tmp/TelldusClient", data: make(chan string)}
	defer c.Cleanup()
	intNumberOfDevices := c.TdGetNumberOfDevices()

	for i := 0; i < intNumberOfDevices; i++ {
		thisDevice := Device{}
		thisDevice.Id = c.TdGetDeviceId(i)
		thisDevice.Name = c.TdGetName(thisDevice.Id)
		thisDevice.Type = SWITCH
		thisDevice.Status = 0
		thisDevice.Dimlevel = 0

		methods := int(c.TdMethods(thisDevice.Id, TELLSTICK_TURNON|TELLSTICK_TURNOFF|TELLSTICK_DIM))
		if (methods & TELLSTICK_DIM) > 0 {
			thisDevice.Type = DIMMER
			if c.TdLastSentCommand(thisDevice.Id, TELLSTICK_DIM) > 0 {
				thisDevice.Dimlevel = c.TdLastSentValue(thisDevice.Id)
			}

			thisDevice.Status = TELLSTICK_TURNOFF
			if thisDevice.Dimlevel > 0 {
				thisDevice.Status = TELLSTICK_TURNON
			}
		} else {
			if (methods & (TELLSTICK_TURNON | TELLSTICK_TURNOFF)) > 0 {
				thisDevice.Status = c.TdLastSentCommand(thisDevice.Id, TELLSTICK_TURNON|TELLSTICK_TURNOFF)
			}
		}
		fmt.Printf("%s\n", thisDevice.String())
	}
	theSensors, err := c.TdSensor()
	if err != nil {
		t.Error(err)
	} else {
		fmt.Printf("TdSensor:%v\n", theSensors)
	}
	for _, sen := range theSensors {
		value, t1 := c.TdSensorValue(&sen)
		thisDevice := Device{}
		thisDevice.Id = sen.Id
		thisDevice.Name = sen.Protocol
		thisDevice.Type = SENSOR
		thisDevice.Status = 0
		thisDevice.Dimlevel = 0
		thisDevice.Value = value

		fmt.Printf("%v %f %d %s\n", sen, value, t1, thisDevice.String())
	}

	ch := make(chan int)
	_ = <-ch

	//t.Error("an error")
}
