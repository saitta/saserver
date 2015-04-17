package suncalc

import (
	"fmt"
	_ "math"
	"testing"
	"time"
)

func TestVariousCCSDS(t *testing.T) {
	//t.Error("an error")
	r, s, err := NextKirunaRiseSet(time.Now())
	fmt.Printf("rise:%s set:%s err:%v \n", r.Local().Format(time.RFC3339), s.Local().Format(time.RFC3339), err)

	polar_night := time.Date(2014, time.December, 24, 1, 0, 0, 0, time.Local)
	r, s, err = NextKirunaRiseSet(polar_night)
	if err == nil {
		t.Error("should return error for polar night")
	}

	midnight_sun := time.Date(2014, time.June, 24, 1, 0, 0, 0, time.Local)
	r, s, err = NextKirunaRiseSet(midnight_sun)
	if err == nil {
		t.Error("should return error for midnightsun")
	}

}
