package main

/*
#include <stdlib.h>
*/

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/saitta/saserver/cron"
	"github.com/saitta/saserver/suncalc"
	"github.com/saitta/saserver/telldus"
	"log"
	"math"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

//mutex is used to protect access to mtdtool (map of devices)
var mutex = &sync.Mutex{}
var mtdtool *telldus.TDTool
var mtdtoolId *telldus.TDToolId

var myCron *cron.Cron
var schedules map[string]string = make(map[string]string)

const LATITUDE = float64(67.8545) * math.Pi / 180
const LONGITUDE = float64(20.2151) * math.Pi / 180
const AMSTERDAM_LAT = float64(52.366667) * math.Pi / 180
const AMSTERDAM_LON = float64(4.9) * math.Pi / 180

var cron_id map[cron.EntryID]cron.EntryID = make(map[cron.EntryID]cron.EntryID)

type connection struct {
	// The websocket connection.
	ws *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte
}

type hub struct {
	// Registered connections.
	connections map[*connection]bool

	// Register requests from the connections.
	register chan *connection

	// Unregister requests from connections.
	unregister chan *connection

	event chan *telldus.Device
}

var h = hub{
	register:    make(chan *connection),
	unregister:  make(chan *connection),
	connections: make(map[*connection]bool),
	event:       make(chan *telldus.Device),
}

func (h *hub) run() {
	for {
		select {
		case con := <-h.register:
			h.connections[con] = true
		case con := <-h.unregister:
			if _, ok := h.connections[con]; ok {
				delete(h.connections, con)
				close(con.send)
			}
		case evt := <-h.event:
			if tosend, err := json.Marshal(*evt); err == nil {
				for con := range h.connections {
					select {
					case con.send <- []byte(tosend):
					default:
						delete(h.connections, con)
						close(con.send)
					}
				}
			}
		}
	}
}

//reader will read telecommand from websocket client and
//put it into the HUBs common telecommand channel
func (con *connection) reader() {
	for {
		_, message, err := con.ws.ReadMessage()
		if err != nil {
			break
		}
		fmt.Println(message)
	}
	con.ws.Close()
}

func (con *connection) writer() {
	for message := range con.send {
		err := con.ws.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			break
		}
	}
	con.ws.Close()
}

var upgrader = &websocket.Upgrader{ReadBufferSize: 4096, WriteBufferSize: 1024}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	con := &connection{send: make(chan []byte, 256), ws: ws}
	h.register <- con
	defer func() { h.unregister <- con }()
	go con.writer()
	con.reader()
}

type CronSchedule struct {
	CronDays []int
	CronTime []int
	Devices  []string
	Action   string
	Value    int
	Id       string
	Group    string
}

func (s *CronSchedule) CronString() string {
	// "sec min hour DoM MoY DoW", newDev.PerformAction)
	var tmp string = ""
	for _, day := range s.CronDays {
		tmp += (strconv.Itoa(day) + ",")
	}
	tmp = tmp[:len(tmp)-1]
	return fmt.Sprintf("0 %d %d * * %s", s.CronTime[1], s.CronTime[0], tmp)
}

func get_devices(w http.ResponseWriter, r *http.Request) {
	// no need to update here
	//mtdtool = telldus.GetTDTool()
	mutex.Lock()
	if b, err := json.Marshal(*mtdtool); err != nil {
		mutex.Unlock()
		w.Write(make([]byte, 5))
	} else {
		mutex.Unlock()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(b)
	}
}

func get_cron_entries(w http.ResponseWriter, r *http.Request) {
	res := make(map[string]string)
	for _, entry := range myCron.Entries() {
		if value, ok := schedules[strconv.Itoa(int(entry.ID))]; ok {
			res[strconv.Itoa(int(entry.ID))] = fmt.Sprintf("%s %s", entry.Next, value)
		} else {
			res[strconv.Itoa(int(entry.ID))] = fmt.Sprintf("%s %s", entry.Next, runtime.FuncForPC(reflect.ValueOf(entry.Job).Pointer()).Name())
		}

	}

	if b, err := json.Marshal(res); err != nil {
		fmt.Println(err)
		w.Write(make([]byte, 5))
	} else {
		w.Write(b)
	}
}

func parseDevicePostOn(w http.ResponseWriter, request *http.Request) {

	fmt.Println("pdevice:", request.Body)
	decoder := json.NewDecoder(request.Body)

	var t telldus.Device
	err := decoder.Decode(&t)

	if err != nil {
		panic(err)
	}
	w.Write([]byte(t.On()))
	//fmt.Println(t.Name)
}

func parseDevicePostOff(w http.ResponseWriter, request *http.Request) {

	decoder := json.NewDecoder(request.Body)
	var t telldus.Device
	err := decoder.Decode(&t)

	if err != nil {
		panic(err)
	}
	w.Write([]byte(t.Off()))
}

func parseDevicePostDim(w http.ResponseWriter, request *http.Request) {

	decoder := json.NewDecoder(request.Body)

	var t telldus.Device
	err := decoder.Decode(&t)

	if err != nil {
		panic(err)
	}
	w.Write([]byte(t.Dim(t.Dimlevel)))
}

func addCronJob(crnstr string, devicename string, action string, value string) {
	mutex.Lock()
	dev := (*mtdtool)[devicename]
	newDev := *dev
	mutex.Unlock()
	switch action {
	case "on":
		newDev.Action = telldus.TELLSTICK_TURNON
	case "off":
		newDev.Action = telldus.TELLSTICK_TURNOFF
	case "dim":
		newDev.Action = telldus.TELLSTICK_DIM
		newDev.Dimlevel, _ = strconv.Atoi(value)
	}
	if ID, addErr := myCron.AddFunc(crnstr, newDev.PerformAction); addErr != nil {
		fmt.Println("addCronJob failed")
	} else {
		schedules[strconv.Itoa(int(ID))] = fmt.Sprintf("%s:%s:%s:%s", crnstr, devicename, action, value)
	}
}

func parseSchedulePost(w http.ResponseWriter, request *http.Request) {

	decoder := json.NewDecoder(request.Body)
	fmt.Println(request.Body)
	var t CronSchedule
	err := decoder.Decode(&t)
	for _, devName := range t.Devices {
		mutex.Lock()
		dev := (*mtdtool)[devName]
		newDev := *dev
		mutex.Unlock()
		switch t.Action {
		case "on":
			newDev.Action = telldus.TELLSTICK_TURNON
		case "off":
			newDev.Action = telldus.TELLSTICK_TURNOFF
		case "dim":
			newDev.Action = telldus.TELLSTICK_DIM
			newDev.Dimlevel = t.Value
		}
		if ID, addErr := myCron.AddFunc(t.CronString(), newDev.PerformAction); addErr != nil {
			fmt.Println("ParseSchedulePost: faild to add cronjob")
		} else {
			schedules[strconv.Itoa(int(ID))] = fmt.Sprintf("%s:%s:%s:%d", t.CronString(), devName, t.Action, t.Value)
		}
	}
	res, _ := json.Marshal(schedules)
	fmt.Println(string(res))
	fd, err := os.Create("schedule.json")
	if err != nil {
		fmt.Println("Error:", err)
	}
	defer fd.Close()
	fd.Write(res)
	if err != nil {
		panic(err)
	}
	w.Write([]byte("OK"))

}

func Log(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL)
		handler.ServeHTTP(w, r)
	})
}

func loadSchedule() {
	fd, err := os.Open("schedule.json")
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		defer fd.Close()

		res := make([]byte, 200000)
		n, _ := fd.Read(res)
		fd.Close()

		var mymap map[string]string
		err = json.Unmarshal(res[:n], &mymap)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(mymap)
		for key, value := range mymap {
			strs := strings.Split(value, ":")
			if len(strs) == 4 {
				crnstr := strs[0]
				dev := strs[1]
				action := strs[2]
				value := strs[3]
				addCronJob(crnstr, dev, action, value)
			}
			fmt.Printf("%s %v \n", key, value)
		}
		if mjson, errJson := json.Marshal(schedules); errJson != nil {
			fmt.Println(errJson)
		} else {
			fmt.Println(mjson)
			fd, err = os.Create("schedule.json")
			if err != nil {
				fmt.Println("Error:", err)
			}
			defer fd.Close()
			fd.Write(mjson)
			if err != nil {
				panic(err)
			}
		}

	}

}

func sunJobCreator() {
	//put device newDev.PerformAction
	// to switch off at sunrise
	// to switch on at sunset

	//remove all old sunjobs !
	for _, ID := range cron_id {
		fmt.Println("removed:" + strconv.Itoa(int(myCron.Entry(ID).ID)))
		myCron.Remove(ID)
		delete(cron_id, ID)
		// not added delete(schedules,strconv.Itoa(int(ID)))
	}

	mutex.Lock()
	if dev, ok := (*mtdtool)["uthus"]; ok {
		newDev := *dev
		newDevS := *dev
		mutex.Unlock()
		t1 := time.Now()
	restart:
		r, s, e := suncalc.NextKirunaRiseSet(t1)
		if e != nil {
			fmt.Println(e)
		} else {
			if s.Before(time.Now()) {
				t1 = t1.Add(time.Hour * 24)
				goto restart
			}
			newDev.Action = telldus.TELLSTICK_TURNOFF
			rL := r.Add(time.Minute * -30).Local()
			sL := s.Add(time.Minute * 15).Local()

			cronString := fmt.Sprintf("0 %d %d %d %d *", rL.Minute(), rL.Hour(), rL.Day(), int(rL.Month()))
			ID, _ := myCron.AddFunc(cronString, newDev.PerformAction)
			cron_id[ID] = ID
			//schedules[strconv.Itoa(int(ID))]=fmt.Sprintf("%s:%s:%s:%d",cronString,newDev.Name,newDev.Action,newDev.Value)

			newDevS.Action = telldus.TELLSTICK_TURNON
			cronString = fmt.Sprintf("0 %d %d %d %d *", sL.Minute(), sL.Hour(), sL.Day(), int(sL.Month()))
			ID, _ = myCron.AddFunc(cronString, newDevS.PerformAction)
			//schedules[strconv.Itoa(int(ID))]=fmt.Sprintf("%s:%s:%s:%d",cronString,newDevS.Name,newDevS.Action,newDevS.Value)
			cron_id[ID] = ID
		}
	} else {
		mutex.Unlock()
	}
}

var prevTimeOn = time.Now().Add(-time.Minute)
var prevTimeOff = time.Now().Add(-time.Minute)

func updateDevice(devE <-chan *telldus.Device, senE <-chan *telldus.Sensor) {

	for {
		select {
		case d := <-devE:
			fmt.Println("willUpdate" + d.String())
			mutex.Lock()
			if dev, ok := (*mtdtoolId)[d.Id]; ok {
				dev.Status = d.Status
				dev.Action = d.Action
				switch d.Action {
				case telldus.TELLSTICK_DIM:
					dev.Dimlevel = d.Dimlevel
				case telldus.TELLSTICK_TURNON:
					dev.Dimlevel = 255
				case telldus.TELLSTICK_TURNOFF:
					dev.Dimlevel = 0
				}
				h.event <- dev
			}
			mutex.Unlock()
			if d.Id == 10 {
				switch d.Action {
				case telldus.TELLSTICK_TURNON:
					fmt.Printf("ON timesince:%v sec10:%v\n", time.Since(prevTimeOn), time.Second*10)
					if time.Since(prevTimeOn) > time.Second*10 {
						fmt.Println("swich on")
						prevTimeOn = time.Now()
						mutex.Lock()
						dev, ok := (*mtdtool)["källare"]
						mutex.Unlock()
						if ok {
							go func() {
								time.Sleep(time.Millisecond * 1500)
								dev.On()
							}()
						}
					}
				case telldus.TELLSTICK_TURNOFF:
					fmt.Printf("OFF timesince:%v sec10:%v\n", time.Since(prevTimeOff), time.Second*10)
					if time.Since(prevTimeOff) > time.Second*10 {
						fmt.Println("swich off")
						prevTimeOff = time.Now()
						mutex.Lock()
						dev, ok := (*mtdtool)["källare"]
						mutex.Unlock()
						if ok {
							go func() {
								time.Sleep(time.Millisecond * 1500)
								dev.Off()
							}()
						}
					}
				}
			}

		case s := <-senE:
			mutex.Lock()
			if sen, ok := (*mtdtoolId)[s.Id]; ok {
				sen.Value = s.Value
				h.event <- sen
			}
			mutex.Unlock()
		}
	}
}

func main() {

	//no need to mutex lock here
	//mtdtool, mtdtoolId = telldus.GetTDTool()
	mtdtool = telldus.NewTDTool()
	tmp := make(telldus.TDToolId)
	mtdtoolId = &tmp
	for _, val := range *mtdtool {
		(*mtdtoolId)[val.Id] = val
	}

	dev_events := make(chan *telldus.Device)
	sen_events := make(chan *telldus.Sensor)
	evr := &telldus.ClientEvent{Fname: "/tmp/TelldusEvents", Device_events: dev_events, Sensor_events: sen_events}
	go evr.Read()
	defer evr.Cleanup()
	go updateDevice(dev_events, sen_events)

	go h.run()
	http.HandleFunc("/ws", wsHandler)

	defer telldus.Cleanup()
	myCron = cron.New()
	myCron.AddFunc("0 1 0 * * *", sunJobCreator)

	myCron.Start()
	sunJobCreator()
	loadSchedule()

	for _, entry := range myCron.Entries() {
		fmt.Printf("%s %s\n", entry.Next, runtime.FuncForPC(reflect.ValueOf(entry.Job).Pointer()).Name())
	}

	http.HandleFunc("/get_devices", get_devices)
	http.HandleFunc("/get_cron_entries", get_cron_entries)
	http.HandleFunc("/pdevice/on", parseDevicePostOn)
	http.HandleFunc("/pdevice/off", parseDevicePostOff)
	http.HandleFunc("/pdevice/dim", parseDevicePostDim)
	http.HandleFunc("/add_schedule", parseSchedulePost)

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", http.StripPrefix("/", fs))

	// Mandatory root-based resources
	//serveSingle("/favicon.ico", "./favicon.ico")
	http.ListenAndServe(":8081", Log(http.DefaultServeMux))
}
