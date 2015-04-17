package telldus

import (
	"fmt"
	_ "math"
	"testing"
	"time"
)

// #cgo CFLAGS:-I/home/leif/Hämtningar/telldus/telldus-core/client
// #cgo LDFLAGS:-ltelldus-core

func TestVariousTelldus(t *testing.T) {
	defer Cleanup()

	d1 := Device{Name: "vrum", Id: 1, Type: 1}
	fmt.Println("hello:", d1.String())

	//t.Error("an error")
	thedevs := NewTDTool()
	for _, dev := range *thedevs {
		fmt.Printf("%s\n", dev.String())
	}
	t1 := time.Now()
	if 1 == 1 {
		fmt.Printf("test:%d %v\n", 1, t1)
	}
}
