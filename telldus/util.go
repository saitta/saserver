package telldus

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
)

type myRegexp struct {
	*regexp.Regexp
}

func (r *myRegexp) FindStringSubmatchMap(s string) map[string]string {
	captures := make(map[string]string)

	match := r.FindStringSubmatch(s)
	if match == nil {
		return captures
	}

	for i, name := range r.SubexpNames() {
		//
		if i == 0 {
			continue
		}
		captures[name] = match[i]

	}
	return captures
}

type MyPosInt struct {
	pos int
	val int
}

func getAllInts(indata string) (res []MyPosInt) {
	var intRegexp = regexp.MustCompile(`i(\d+)s`)
	//10:fineoffset11:temperaturei119si1s6:oregon4:EA4Ci204si1s
	allints := intRegexp.FindAllStringSubmatch(indata, -1)
	allIndexes := intRegexp.FindAllStringSubmatchIndex(indata, -1)
	res = make([]MyPosInt, len(allints))
	for i, tmp := range allints {
		tmpPos := allIndexes[i][1]
		tmpint, _ := strconv.Atoi(tmp[1])
		res[i].pos = tmpPos
		res[i].val = tmpint
	}

	return
}

func getTdInt(indata string) (res int) {
	var err error
	res = 0
	// "i10s\n"
	if indata[0] != 'i' {
		fmt.Printf("not i in pos 0 (%s)\n", indata)
	} else {
		pos := strings.Index(indata, "s")
		if pos < 0 {
			log.Println("getTdInt: cannot find end (s) in intstring")
			return
		}
		res, err = strconv.Atoi(indata[1:pos])
		if err != nil {
			fmt.Printf("getTdInt: cannot convert (%s) to int\n", indata[1:2])
			res = 0
		}
	}
	return
}

func getTdString(indata string) (res string) {
	res = ""
	lst := strings.SplitN(indata, ":", 2)
	if len(lst) != 2 {
		log.Printf("getTdString: wrong length of return value (%d) str(%s)\n", len(lst), indata)
	} else {
		res = strings.TrimSpace(lst[1])
	}
	return
}

func getFirstString(indata string) (res string) {
	res = "UNKNOWN"
	var strLenExp = regexp.MustCompile(`.*?(\d+)$`)
	twoParts := strings.SplitN(indata, ":", 2)
	if len(twoParts) != 2 {
		log.Printf("getFirstString: wrong length(%d) of splitted input string(%s)\n", len(twoParts), indata)
		return
	}

	match := strLenExp.FindStringSubmatch(twoParts[0])
	if match == nil {
		fmt.Printf("no result from sensor:(%s)\n", indata)
	} else {
		if len(match) > 1 {
			strlen, _ := strconv.Atoi(match[1])
			res = twoParts[1][0:strlen]
		}
	}
	return
}

func getSecondString(indata string) (res string) {
	res = "UNKNOWN"
	var strLenExp = regexp.MustCompile(`.*?(\d+)$`)
	threeParts := strings.SplitN(indata, ":", 3)
	if len(threeParts) != 3 {
		log.Printf("getSecondString: wrong length(%d) of splitted input string\n", len(threeParts))
		return
	}
	match := strLenExp.FindStringSubmatch(threeParts[1])
	if match == nil {
		fmt.Printf("no result from sensor:(%s)\n", indata)
	} else {
		if len(match) > 1 {
			strlen, _ := strconv.Atoi(match[1])
			res = threeParts[2][0:strlen]
		}
	}
	return
}

func getThirdString(indata string) (res string) {
	res = "UNKNOWN"
	var strLenExp = regexp.MustCompile(`.*?(\d+)$`)
	fourParts := strings.SplitN(indata, ":", 4)
	if len(fourParts) != 4 {
		log.Printf("getThirdString: wrong length(%d) of splitted input string\n", len(fourParts))
		return
	}
	match := strLenExp.FindStringSubmatch(fourParts[2])
	if match == nil {
		fmt.Printf("no result from sensor:(%s)\n", indata)
	} else {
		if len(match) > 1 {
			strlen, _ := strconv.Atoi(match[1])
			res = fourParts[3][0:strlen]
		}
	}
	return
}
