package suncalc

import (
	"errors"
	"github.com/soniakeys/meeus/julian"
	"math"
	"time"
)

const (
	J1970   float64 = 2440588
	J2000   float64 = 2451545
	deg2rad         = math.Pi / 180
	M0              = 357.5291 * deg2rad
	M1              = 0.98560028 * deg2rad
	J0              = 0.0009
	J1              = 0.0053
	J2              = -0.0069
	C1              = 1.9148 * deg2rad
	C2              = 0.0200 * deg2rad
	C3              = 0.0003 * deg2rad
	P               = 102.9372 * deg2rad
	e               = 23.45 * deg2rad
	th0             = 280.1600 * deg2rad
	th1             = 360.9856235 * deg2rad
	h0              = -0.83 * deg2rad //sunset angle
	d0              = 0.53 * deg2rad  //sun diameter
	h1              = -6 * deg2rad    //nautical twilight angle
	h2              = -12 * deg2rad   //astronomical twilight angle
	h3              = -18 * deg2rad   //darkness angle
	msInDay         = 1000 * 60 * 60 * 24

	KIRUNA_LAT = float64(67.8545)
	KIRUNA_LON = float64(20.2151)
)

func dateToJulianDate(date time.Time) float64 {
	return julian.TimeToJD(date)
}

func julianDateToDate(j float64) time.Time {
	return julian.JDToTime(j)
}

func getJulianCycle(J float64, lw float64) float64 {
	return math.Floor(J - J2000 - J0 - lw/(2*math.Pi) + 0.5)
}

func getApproxSolarTransit(Ht float64, lw float64, n float64) float64 {
	return J2000 + J0 + (Ht+lw)/(2*math.Pi) + n
}

func getSolarMeanAnomaly(Js float64) float64 {
	return M0 + M1*(Js-J2000)
}

func getEquationOfCenter(M float64) float64 {
	return C1*math.Sin(M) + C2*math.Sin(2*M) + C3*math.Sin(3*M)
}

func getEclipticLongitude(M float64, C float64) float64 {
	return M + P + C + math.Pi
}

func getSolarTransit(Js float64, M float64, Lsun float64) float64 {
	return Js + (J1 * math.Sin(M)) + (J2 * math.Sin(2*Lsun))
}

func getSunDeclination(Lsun float64) float64 {
	return math.Asin(math.Sin(Lsun) * math.Sin(e))
}

func getRightAscension(Lsun float64) float64 {
	return math.Atan2(math.Sin(Lsun)*math.Cos(e), math.Cos(Lsun))
}

func getSiderealTime(J float64, lw float64) float64 {
	return th0 + th1*(J-J2000) - lw
}

func getAzimuth(th float64, a float64, phi float64, d float64) float64 {
	H := th - a
	return math.Atan2(math.Sin(H), math.Cos(H)*math.Sin(phi)-math.Tan(d)*math.Cos(phi))
}

func getAltitude(th float64, a float64, phi float64, d float64) float64 {
	H := th - a
	return math.Asin(math.Sin(phi)*math.Sin(d) + math.Cos(phi)*math.Cos(d)*math.Cos(H))
}

func getHourAngle(h float64, phi float64, d float64) float64 {
	return math.Acos((math.Sin(h) - math.Sin(phi)*math.Sin(d)) / (math.Cos(phi) * math.Cos(d)))
}

func getSunsetJulianDate(w0 float64, M float64, Lsun float64, lw float64, n float64) float64 {
	return getSolarTransit(getApproxSolarTransit(w0, lw, n), M, Lsun)
}

func getSunriseJulianDate(Jtransit float64, Jset float64) float64 {
	return Jtransit - (Jset - Jtransit)
}

func getSunPosition(J float64, lw float64, phi float64) (azimuth float64, altitude float64) {
	M := getSolarMeanAnomaly(J)
	C := getEquationOfCenter(M)
	Lsun := getEclipticLongitude(M, C)
	d := getSunDeclination(Lsun)
	a := getRightAscension(Lsun)
	th := getSiderealTime(J, lw)

	return getAzimuth(th, a, phi, d), getAltitude(th, a, phi, d)
}

func NextKirunaRiseSet(date time.Time) (rise time.Time, set time.Time, err error) {

	rise = time.Now()
	set = rise
	err = nil

	lw := -KIRUNA_LON * deg2rad
	phi := KIRUNA_LAT * deg2rad
	J := dateToJulianDate(date)

	n := getJulianCycle(J, lw)
	Js := getApproxSolarTransit(0, lw, n)
	M := getSolarMeanAnomaly(Js)
	C := getEquationOfCenter(M)
	Lsun := getEclipticLongitude(M, C)
	d := getSunDeclination(Lsun)
	Jtransit := getSolarTransit(Js, M, Lsun)
	w0 := getHourAngle(h0, phi, d)
	if math.IsNaN(w0) {
		if date.Month() == time.December || date.Month() == time.January {
			err = errors.New("No rise/set can be calculated! sun below horizon !")
			return
		} else {
			err = errors.New("No rise/set can be calculated! sun above horizon !")
			return
		}

	}
	w1 := getHourAngle(h0+d0, phi, d)
	Jset := getSunsetJulianDate(w0, M, Lsun, lw, n)
	Jsetstart := getSunsetJulianDate(w1, M, Lsun, lw, n)
	Jrise := getSunriseJulianDate(Jtransit, Jset)
	Jriseend := getSunriseJulianDate(Jtransit, Jsetstart)
	w2 := getHourAngle(h1, phi, d)
	Jnau := getSunsetJulianDate(w2, M, Lsun, lw, n)
	Jciv2 := getSunriseJulianDate(Jtransit, Jnau)
	_ = Jriseend
	_ = Jciv2
	//dawn := julianDateToDate(Jciv2)
	rise = julianDateToDate(Jrise)
	//rise_end := julianDateToDate(Jriseend)
	//transit := julianDateToDate(Jtransit)
	//set_start := julianDateToDate(Jsetstart)
	set = julianDateToDate(Jset)
	//dusk := julianDateToDate(Jnau)
	return
}
