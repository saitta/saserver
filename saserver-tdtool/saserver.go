package saserver

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const (
	SWITCH   int = 0
	DIMMER   int = 1
	SENSOR   int = 2
	DETECTOR int = 3
	TURN_OFF int = 0
	TURN_ON  int = 1
	DIM      int = 2
)

const TDTOOL string = "/usr/bin/tdtool"

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
	// Create an *exec.Cmd
	cmd := exec.Command(TDTOOL, action...)
	// Stdout buffer
	cmdOutput := &bytes.Buffer{}
	cmd.Stdout = cmdOutput
	if err := cmd.Run(); err != nil {
		return fmt.Sprintf("%v %v", err, cmdOutput.String())
	} else {
		return cmdOutput.String()
	}
}

func (d *Device) String() string {
	return fmt.Sprintf("%s %d %d action: %d ", d.Name, d.Id, d.Type, d.Action)
}

func (d *Device) Dim(level int) string {
	d.Status = level
	d.Dimlevel = level
	d.exec([]string{"--dimlevel", strconv.Itoa(level), "--dim", d.Name})
	return d.exec([]string{"--dimlevel", strconv.Itoa(level), "--dim", d.Name})
}

func (d *Device) On() string {
	d.Status = 1
	d.exec([]string{"--on", d.Name})
	return d.exec([]string{"--on", d.Name})
}

func (d *Device) Off() string {
	d.Status = 0
	d.exec([]string{"--off", d.Name})
	return d.exec([]string{"--off", d.Name})
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

func main() {
	d1 := Device{Name: "vrum", Id: 1, Type: 1}
	fmt.Println("hello:", d1.String())
}

type TDTool map[string]*Device

func NewTDTool() *TDTool {

	var nDevices int

	my_devices := make(TDTool)

	// Create an *exec.Cmd
	cmd := exec.Command(TDTOOL, []string{"-l"}...)
	// Stdout buffer
	cmdOutput := &bytes.Buffer{}
	cmd.Stdout = cmdOutput
	if err := cmd.Run(); err != nil {
		fmt.Println(err)
	} else {

		//number of devices
		//1       switch_uthus    OFF
		ndev_patt, _ := regexp.Compile(`Number of devices:\s+(\d+)`)
		devices := strings.Split(cmdOutput.String(), "\n")
		for _, dev := range devices {
			matched := ndev_patt.FindAllStringSubmatch(dev, -1)
			if len(matched) > 0 {
				//as soon as length is > 0 we have a match on number of devices
				nDevices, _ = strconv.Atoi((matched[0][1]))
				fmt.Printf("nDevices:%d\n", nDevices)
				break
			}
		}

		//the devices
		dev_patt, _ := regexp.Compile(`^(\d+)\s+(\w+)\s+([ONF]+|DIMMED)`)
		for _, dev := range devices {
			fmt.Println(dev)
			matched := dev_patt.FindAllStringSubmatch(dev, -1)
			if len(matched) > 0 {
				thisDevice := Device{}
				thisDevice.Id, _ = strconv.Atoi(matched[0][1])
				thisDevice.Name = matched[0][2]
				thisDevice.Type = SWITCH
				thisDevice.Status = 0

				if strings.Contains(thisDevice.Name, "dimmer") {
					thisDevice.Type = DIMMER
				}
				if strings.Contains(thisDevice.Name, "detector") {
					thisDevice.Type = DETECTOR
				}
				switch matched[0][3] {
				case "ON":
					thisDevice.Status = 1
				case "OFF":
					thisDevice.Status = 0
				default:
					if strings.Contains(matched[0][3], "DIMMED:") {
						res := strings.Split(matched[0][3], ":")
						thisDevice.Status, _ = strconv.Atoi(res[1])
						thisDevice.Dimlevel, _ = strconv.Atoi(res[1])
						thisDevice.Type = DIMMER
					} else {
						fmt.Println("Cannot determine status!")
					}
				}
				my_devices[thisDevice.Name] = &thisDevice
			}
		}

		//the sensors
		//PROTOCOL                MODEL                   ID      TEMP    HUMIDITY        RAIN                    WIND                    LAST UPDATED
		//fineoffset              temperature             119     -7.9°                                                                   2015-01-25 18:57:41
		//oregon                  1A2D                    177     21.7°   32%                                                             2015-01-12 20:25:30
		patt, _ := regexp.Compile(`^(\w+)\s+(\w+)\s+(\d+)\s+([-\.\d]+)°\s+(\d+%)?\s+(\d{4}-\d{2}-\d{2}) (\d{2}:\d{2}:\d{2})(?:\s*)$`)
		found := false
		for _, dev := range devices {

			if found {
				res := patt.FindAllStringSubmatch(dev, -1)
				if len(res) > 0 {
					fmt.Printf("%s : %s : %s : %s : %s : %s : %v \n", res[0][1], res[0][2], res[0][3], res[0][4], res[0][5], res[0][6], res[0][7])
					thisDevice := Device{}
					thisDevice.Id, _ = strconv.Atoi(res[0][3])
					thisDevice.Name = res[0][1]
					thisDevice.Type = SENSOR
					thisDevice.Value, _ = strconv.ParseFloat(res[0][4], 32)
					thisDevice.Unit = "degC"
					my_devices[thisDevice.Name] = &thisDevice
				}
			} else if strings.HasPrefix(dev, "PROTOCOL") {
				found = true
			}
		}
	}

	return &my_devices
}
