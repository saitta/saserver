// Package telldus exposes parts of the original Telldus C API
// see READMEs and http://developer.telldus.com/doxygen for documentation
// @author Leif Kalla
package telldus

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	TELLSTICK_TURNON                      int = 1
	TELLSTICK_TURNOFF                         = 2
	TELLSTICK_BELL                            = 4
	TELLSTICK_TOGGLE                          = 8
	TELLSTICK_DIM                             = 16
	TELLSTICK_LEARN                           = 32
	TELLSTICK_EXECUTE                         = 64
	TELLSTICK_UP                              = 128
	TELLSTICK_DOWN                            = 256
	TELLSTICK_STOP                            = 512
	TELLSTICK_TEMPERATURE                     = 1
	TELLSTICK_HUMIDITY                        = 2
	TELLSTICK_RAINRATE                        = 4
	TELLSTICK_RAINTOTAL                       = 8
	TELLSTICK_WINDDIRECTION                   = 16
	TELLSTICK_WINDAVERAGE                     = 32
	TELLSTICK_WINDGUST                        = 64
	TELLSTICK_SUCCESS                         = 0
	TELLSTICK_ERROR_NOT_FOUND                 = -1
	TELLSTICK_ERROR_PERMISSION_DENIED         = -2
	TELLSTICK_ERROR_DEVICE_NOT_FOUND          = -3
	TELLSTICK_ERROR_METHOD_NOT_SUPPORTED      = -4
	TELLSTICK_ERROR_COMMUNICATION             = -5
	TELLSTICK_ERROR_CONNECTING_SERVICE        = -6
	TELLSTICK_ERROR_UNKNOWN_RESPONSE          = -7
	TELLSTICK_ERROR_SYNTAX                    = -8
	TELLSTICK_ERROR_BROKEN_PIPE               = -9
	TELLSTICK_ERROR_COMMUNICATING_SERVICE     = -10
	TELLSTICK_ERROR_UNKNOWN                   = -99
	TELLSTICK_BUFFER_MAX                      = 2000
)

const (
	SWITCH   int = 0
	DIMMER   int = 1
	SENSOR   int = 2
	DETECTOR int = 3
)

type TDTool map[string]*Device
type TDToolId map[int]*Device

//mutex is used to protect access to localTelldusClient
var lt_mutex = &sync.Mutex{}

var localTelldusClient *Client

func init() {
	fmt.Println("init")
	localTelldusClient = &Client{fname: "/tmp/TelldusClient", data: make(chan string)}
}

func Cleanup() {
	localTelldusClient.Cleanup()
}

/* telldus strings examples

8:tdTurnOni1s
5:tdDimi5si128s

send(4, "20:tdGetNumberOfDevices", 23, 0) = 23
send(4, "13:tdGetDeviceIdi0s", 19, 0)   = 19
send(4, "9:tdGetNamei1s", 14, 0)        = 14
send(4, "17:tdLastSentCommandi1si23s", 27, 0) = 27

send(4, "13:tdGetDeviceIdi1s", 19, 0)   = 19
send(4, "9:tdGetNamei2s", 14, 0)        = 14
send(4, "17:tdLastSentCommandi2si23s", 27, 0) = 27
send(4, "15:tdLastSentValuei2s", 21, 0) = 21

send(4, "8:tdSensor", 10, 0)            = 10
send(4, "13:tdSensorValue10:fineoffset11:"..., 51, 0) = 51

send(4, "8:tdSensor", 10, 0)            = 10
recv(4, "38:i1s10:fineoffset11:temperatur"..., 511, 0) = 42
send(4, "13:tdSensorValue10:fineoffset11:"..., 51, 0) = 51
send(3, "13:tdSensorValue6:oregon4:EA4Ci2"..., 38, 0)
recv(4, "17:3:2.4i1428072058s\n", 511, 0) = 21


EVENTS
=======

dimmer off:
13:TDDeviceEventi5si2s1:0
dimmer to level 170
13:TDDeviceEventi5si16s3:170

ClientEvent Read (25)
13:TDDeviceEventi8si2s1:0
ClientEvent Read (25)
13:TDDeviceEventi8si1s1:0
ClientEvent Read (92)
16:TDRawDeviceEvent67:class:sensor;protocol:fineoffset;id:119;model:temperature;temp:2.3;i2s
ClientEvent Read (68)
13:TDSensorEvent10:fineoffset11:temperaturei119si1s3:2.3i1428338995s
ClientEvent Read (28)
13:TDDeviceEventi5si16s3:178
ClientEvent Read (25)
13:TDDeviceEventi5si2s1:0

*/

type Sensor struct {
	Protocol  string
	Model     string
	Id        int
	Datatype  int
	StrEndPos int
	Value     float64
	T         time.Time
}

type ClientEvent struct {
	con           net.Conn
	Fname         string
	Device_events chan<- *Device
	Sensor_events chan<- *Sensor
}

func (cli *ClientEvent) Read() {
	var eventExp = regexp.MustCompile(`\d+:(TD\w+Event).*`)
	cli.reconnect()
	connbuf := bufio.NewReader(cli.con)
	data := make([]byte, TELLSTICK_BUFFER_MAX)
	for {
	retry:
		//str, err := connbuf.ReadString('\n'); if err != nil {
		nr, err := connbuf.Read(data)
		if err != nil {
			cli.reconnect()
			time.Sleep(time.Second)
			goto retry
		}
		str := string(data[0:nr])
		res := eventExp.FindStringSubmatch(str)
		if len(res) > 1 {
			switch res[1] {
			case "TDDeviceEvent":
				//13:TDDeviceEventi8si2s1:0
				//rest := strings.Split(str, res[1])[1]
				allInts := getAllInts(str)
				//[{19 7} {22 2}]
				lt_mutex.Lock()
				dev := Device{Name: localTelldusClient.TdGetName(allInts[0].val), Id: allInts[0].val, Status: allInts[1].val, Action: allInts[1].val}
				lt_mutex.Unlock()
				if dev.Action == TELLSTICK_DIM {
					if dimlevel, err := strconv.Atoi(getSecondString(str)); err == nil {
						dev.Dimlevel = dimlevel
					}
				}
				cli.Device_events <- &dev
				fmt.Printf("Device: %v\n", dev)
			case "TDSensorEvent":
				//13:TDSensorEvent10:fineoffset11:temperaturei119si1s3:2.3i1428338995s
				sen := Sensor{}
				rest := strings.Split(str, res[1])[1]
				sen.Protocol = getFirstString(rest)
				sen.Model = getSecondString(rest)
				allInts := getAllInts(rest)
				sen.Id = allInts[0].val
				sen.Datatype = allInts[1].val
				sen.StrEndPos = allInts[2].pos
				sen.T = time.Unix(int64(allInts[2].val), 0)
				sen.Value, err = strconv.ParseFloat(getThirdString(rest), 64)
				//Sensor: TDSensorEvent {fineoffset temperature 0 0 0} [{32 119} {35 1} {52 1428425781}]
				fmt.Printf("Sensor: %s %v\n", res[1], sen)
				cli.Sensor_events <- &sen
			default:
				fmt.Printf("Default: %s\n", res[1])
			}
		}
	}
	cli.con.Close()
}

func (cli *ClientEvent) Cleanup() {
	cli.con.Close()
}

func (cli *ClientEvent) reconnect() {
	var err error
	cli.con, err = net.Dial("unix", cli.Fname)
	if err != nil {
		panic(err)
	}
}

type Client struct {
	con   net.Conn
	fname string
	data  chan string
}

func (cli *Client) exec(cmd string) (res string) {
	res = "NONE"
	cli.reconnect()
restart:
	_, err := cli.con.Write([]byte(cmd))
	if err != nil {
		log.Println("write error:", err)
		cli.reconnect()
		goto restart
	} else {
		res = cli.read()
	}
	return res
}

func (cli *Client) read() (res string) {
	connbuf := bufio.NewReader(cli.con)
	str, err := connbuf.ReadString('\n')
	if len(str) > 0 {
		res = str
	}
	if err != nil {
		log.Println(err)
	}
	cli.con.Close()
	return
}

func (cli *Client) reconnect() {
	var err error
	cli.con, err = net.Dial("unix", cli.fname)
	if err != nil {
		panic(err)
	}
}

func (cli *Client) TdTurnOn(id int) int {
	tmp := cli.exec(fmt.Sprintf("8:tdTurnOni%ds", id))
	return getTdInt(tmp)
}

func (cli *Client) TdTurnOff(id int) int {
	tmp := cli.exec(fmt.Sprintf("9:tdTurnOffi%ds", id))
	return getTdInt(tmp)
}

func (cli *Client) TdDim(id int, level int) int {
	tmp := cli.exec(fmt.Sprintf("5:tdDimi%dsi%ds", id, level))
	return getTdInt(tmp)
}

func (cli *Client) TdGetNumberOfDevices() int {
	tmp := cli.exec("20:tdGetNumberOfDevices")
	return getTdInt(tmp)
}

// TdGetName returns the name of the device with id id
// OBS! The string length in telldus return is not correct for
// characters like åäö
func (cli *Client) TdGetName(id int) string {
	tmp := cli.exec(fmt.Sprintf("9:tdGetNamei%ds", id))
	twoParts := strings.SplitN(tmp, ":", 2)
	if len(twoParts) != 2 {
		log.Printf("TdGetName: wrong length(%d) of splitted input string(%s)\n", len(twoParts), tmp)
		return "UNKNOWN"
	}
	return strings.TrimSpace(twoParts[1])
}

func (cli *Client) TdGetDeviceId(nr int) int {
	tmp := cli.exec(fmt.Sprintf("13:tdGetDeviceIdi%ds", nr))
	return getTdInt(tmp)
}

func (cli *Client) TdMethods(id int, selector int) int {
	tmp := cli.exec(fmt.Sprintf("9:tdMethodsi%dsi%ds", id, selector))
	return getTdInt(tmp)
}

func (cli *Client) TdLastSentCommand(id int, selector int) int {
	tmp := cli.exec(fmt.Sprintf("17:tdLastSentCommandi%dsi%ds", id, selector))
	return getTdInt(tmp)
}

func (cli *Client) TdLastSentValue(id int) (res int) {
	tmp := cli.exec(fmt.Sprintf("15:tdLastSentValuei%ds", id))
	if getTdInt(tmp) != 0 {
		res, _ = strconv.Atoi(getFirstString(tmp))
	}

	return res
}

/** indata is the string which represents _one_ sensor
* it is returned from the Telldus daemon
*    [out]	protocol	A by ref string where the protocol of the sensor will be placed.
*    [in]	protocolLen	The length of the protocol parameter.
*    [out]	model	A by ref string where the model of the sensor will be placed.
*    [in]	modelLen	The length of the model parameter.
*    [out]	id	A by ref int where the id of the sensor will be placed.
*    [out]	dataTypes	A by ref int with flags for the supported sensor values.
 */
func getSensor(indata string) (ret Sensor, err error) {
	err = nil
	//we have sensor
	ret.Protocol = getFirstString(indata)
	ret.Model = getSecondString(indata)
	allInts := getAllInts(indata)
	if len(allInts) < 2 {
		err = errors.New("TdSensor: cannot get sensor id and dataType" + indata)
	} else {
		ret.Id = allInts[0].val
		ret.Datatype = allInts[1].val
		ret.StrEndPos = allInts[1].pos
	}
	//fmt.Printf("TdSensor: protocol(%s) model(%s) id(%d) dataTypes(%d)\n",ret.protocol,ret.model,ret.id,ret.datatype)

	return
}

func (cli *Client) TdSensor() (ret []Sensor, err error) {
	//var sensorRegexp = myRegexp{regexp.MustCompile(`i(?P<OK>\d+)s\d+:(?P<protocol>[a-z,A-Z]+)\d+:(?P<model>[a-z,A-Z]+)`)}
	var intRegexp = regexp.MustCompile(`i(\d+)s`)

	err = nil
	tmp := strings.SplitN(cli.exec("8:tdSensor"), ":", 2)[1] //cut away the first length

	//tmp := "i2s10:fineoffset11:temperaturei119si1s6:oregon4:EA4Ci204si1s"
	nSensors := getTdInt(tmp)
	x := intRegexp.Split(tmp, 2) //split after first telldus int ixxs
	tmp = strings.TrimSpace(x[1])
	var slicepos int = 0

	if nSensors > 0 {
		ret = make([]Sensor, nSensors)
		for i := 0; i < nSensors; i++ {
			//this will cut the string after each sensor is processed prom the line
			tmp = tmp[slicepos:]
			aSensor, err2 := getSensor(tmp)
			if err2 != nil {
				err = err2
			} else {
				ret[i] = aSensor
				slicepos = aSensor.StrEndPos
			}
		}
	} else {
		err = errors.New("TdSensor: no sensors " + tmp)
	}
	return
}

func (cli *Client) TdSensorValue(sen *Sensor) (value float64, t1 time.Time) {
	//18:4:20.4i1428159391s
	//"6:oregon4:EA4Ci204si1s"
	//13:tdSensorValue10:fineoffset:11:temperaturei119si1s
	toSend := fmt.Sprintf("13:tdSensorValue%d:%s%d:%si%dsi%ds", len(sen.Protocol), sen.Protocol, len(sen.Model), sen.Model, sen.Id, sen.Datatype)
	tmp := cli.exec(toSend)
	if len(tmp) > 5 {
		tvalue, err := strconv.ParseFloat(getFirstString(strings.SplitN(tmp, ":", 2)[1]), 64)
		if err != nil {
			log.Printf("TdSensorValue: cannot convert sensor value to float")
		} else {
			value = tvalue
		}

		allints := getAllInts(tmp)
		t1 = time.Unix(int64(allints[0].val), 0)
		// fmt.Printf(" %f %d %v\n", value, allints[0].val, time.Since(t1))
		sen.Value = value
		sen.T = t1
	}
	return
}

func (cli *Client) Cleanup() {
	fmt.Printf("Cleanup :%d\n", 1)
	cli.con.Close()
}

func NewTDTool() *TDTool {
	my_devices := make(TDTool)
	lt_mutex.Lock()
	intNumberOfDevices := localTelldusClient.TdGetNumberOfDevices()

	for i := 0; i < intNumberOfDevices; i++ {
		thisDevice := Device{}
		thisDevice.Id = localTelldusClient.TdGetDeviceId(i)
		thisDevice.Name = localTelldusClient.TdGetName(thisDevice.Id)
		thisDevice.Type = SWITCH
		if strings.Contains(thisDevice.Name, "detector") {
			thisDevice.Type = DETECTOR
		}
		thisDevice.Status = 0
		thisDevice.Dimlevel = 0

		methods := int(localTelldusClient.TdMethods(thisDevice.Id, TELLSTICK_TURNON|TELLSTICK_TURNOFF|TELLSTICK_DIM))
		if (methods & TELLSTICK_DIM) > 0 {
			thisDevice.Type = DIMMER
			if localTelldusClient.TdLastSentCommand(thisDevice.Id, TELLSTICK_DIM) > 0 {
				thisDevice.Dimlevel = localTelldusClient.TdLastSentValue(thisDevice.Id)
			}

			thisDevice.Status = TELLSTICK_TURNOFF
			if thisDevice.Dimlevel > 0 {
				thisDevice.Status = TELLSTICK_TURNON
			}
		} else {
			if (methods & (TELLSTICK_TURNON | TELLSTICK_TURNOFF)) > 0 {
				thisDevice.Status = localTelldusClient.TdLastSentCommand(thisDevice.Id, TELLSTICK_TURNON|TELLSTICK_TURNOFF)
			}
		}
		my_devices[thisDevice.Name] = &thisDevice
		fmt.Printf("%s\n", thisDevice.String())
	}
	theSensors, err := localTelldusClient.TdSensor()
	if err != nil {
		log.Printf("no sensors\n")
	} else {
		for _, sen := range theSensors {
			value, t1 := localTelldusClient.TdSensorValue(&sen)
			thisDevice := Device{}
			thisDevice.Id = sen.Id
			thisDevice.Name = sen.Protocol
			thisDevice.Type = SENSOR
			thisDevice.Status = 0
			thisDevice.Dimlevel = 0
			thisDevice.Value = value

			my_devices[thisDevice.Name] = &thisDevice
			fmt.Printf("%v %f %d %s\n", sen, value, t1, thisDevice.String())
		}
	}
	lt_mutex.Unlock()
	return &my_devices
}
