package telldus

import "fmt"

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

//{2 dimmer_vrum  1 1 0 2 0 }

func (d *Device) exec(action []string) string {
	return "hello"
}

func (d *Device) String() string {
	return fmt.Sprintf("%s id(%d) type(%d) action(%d) status(%d) dimlevel(%d)", d.Name, d.Id, d.Type, d.Action, d.Status, d.Dimlevel)
}

func (d *Device) Dim(level int) string {
	d.Status = level
	d.Dimlevel = level
	localTelldusClient.TdDim(d.Id, d.Dimlevel)
	return d.Name
}

func (d *Device) On() string {
	d.Status = 1
	localTelldusClient.TdTurnOn(d.Id)
	return d.Name
}

func (d *Device) Off() string {
	d.Status = 2
	localTelldusClient.TdTurnOff(d.Id)
	return d.Name
}

func (d *Device) PerformAction() {
	var res string

	switch d.Action {
	case TELLSTICK_TURNON:
		res = d.On()
	case TELLSTICK_TURNOFF:
		res = d.Off()
	case TELLSTICK_DIM:
		res = d.Dim(d.Dimlevel)

	default:
		fmt.Println("Unknown action, should return error")
	}
	fmt.Println(res)
}
