package lib

import (
	"log"
	"time"
)

func (mnk *Monkey) showReseting() {
	// TODO: use [..][..][..] instead of |..|..|..|
	// with: mnk.progress.lastLane.GetTotalTestsCount() == 0
	ColorERR.Printf("|")
}

func (act *FuzzProgress) exec(mnk *Monkey) (err error) {
	log.Printf("[ERR] >>> FuzzProgress %+v", act)
	var str string

	switch {
	case act.GetLastCallSuccess():
		str += ColorNFO.Sprint("✓")
	case act.GetLastCallFailure():
		str += ColorERR.Sprint("✗")
	case act.GetSuccess() || act.GetFailure():
		str += "\n"
	}

	switch {
	case act.GetShrinking() && mnk.progress.shrinkingFrom == nil:
		mnk.progress.shrinkingFrom = &mnk.progress.lastLane
		str += "Shrinking: "
	case act.GetTotalTestsCount() == 0:
		str = "Testing: " + str
	case act.GetTotalTestsCount() != mnk.progress.lastLane.GetTotalTestsCount():
		str += "]"
	}

	mnk.progress.lastLane = *act
	ColorNFO.Print(str)
	return
}

func (mnk *Monkey) TestsSucceeded() (success bool) {
	p := mnk.progress

	totalTests := p.lastLane.GetTotalTestsCount()
	totalCalls := p.lastLane.GetTotalCallsCount()
	totalChecks := p.lastLane.GetTotalChecksCount()
	ColorWRN.Println(
		"Ran", totalTests, plural("test", totalTests),
		"totalling", totalCalls, plural("request", totalCalls),
		"and", totalChecks, plural("check", totalChecks),
		"in", time.Since(p.start))

	switch {
	case p.lastLane.GetSuccess():
		success = true
		ColorNFO.Println("No bugs found... yet.")

	case p.lastLane.GetFailure():
		success = false
		var d, m uint32
		if p.shrinkingFrom == nil {
			d = p.lastLane.GetTotalTestsCount()
		} else {
			d = p.shrinkingFrom.GetTotalTestsCount()
			m = p.lastLane.GetTotalTestsCount() - d
		}
		testCalls := p.lastLane.GetTestCallsCount()
		ColorERR.Printf("A bug reproducible in %d HTTP %s was detected after %d",
			testCalls, plural("request", testCalls), d)
		switch {
		case testCalls == 1:
			ColorERR.Println(" test.")
		case m == 0:
			ColorERR.Println(" tests and not yet shrunk.")
		case m == 1:
			ColorERR.Println(" tests then shrunk", "once.")
		default:
			ColorERR.Println(" tests then shrunk", m, "times.")
		}

	default:
		panic(`either xor have to be true`)
	}
	return
}

func plural(s string, n uint32) string {
	if n == 1 {
		return s
	}
	return s + "s"
}
