package pkg

import (
	"log"
	"time"
)

func (act *FuzzProgress) exec(mnk *Monkey) (err error) {
	log.Printf("[ERR] >>> FuzzProgress %+v", act)
	last := mnk.progress.lastLane
	mnk.progress.lastLane = *act
	var str string

	// if last.GetTotalTestsCount() == 0 {
	// 	str = ColorNFO.Sprint("[")
	// }

	diff := &FuzzProgress{
		TotalTestsCount:  act.GetTotalTestsCount() - last.GetTotalTestsCount(),
		TotalCallsCount:  act.GetTotalCallsCount() - last.GetTotalCallsCount(),
		TotalChecksCount: act.GetTotalChecksCount() - last.GetTotalChecksCount(),
		TestCallsCount:   act.GetTestCallsCount() - last.GetTestCallsCount(),
		CallChecksCount:  act.GetCallChecksCount() - last.GetCallChecksCount(),
	}
	log.Printf("[ERR] diff %+v", diff)

	if act.GetLastCheckSuccess() {
		str += "•"
	} else if act.GetLastCheckFailure() {
		str += "!"
	}
	if act.GetLastCallSuccess() {
		str += ColorWRN.Sprint("✓")
	} else if act.GetLastCallFailure() {
		str += ColorERR.Sprint("⨯")
	}
	// if act.GetSuccess() {
	// 	str += ColorWRN.Sprint("PASSED") + ColorNFO.Sprint("]") + "\n"
	// } else if act.GetFailure() {
	// 	str += ColorERR.Sprint("FAILED") + ColorNFO.Sprint("]") + "\n"
	// }

	// case act.GetTotalTestsCount() == 0:
	// 	// Avoids getting in below case
	// case act.GetTotalTestsCount() != mnk.progress.lastLane.GetTotalTestsCount():
	// 	str += "]["

	// fmt.Print(str)
	return
}

// TestsSucceeded TODO
func (mnk *Monkey) TestsSucceeded() (success bool) {
	p := mnk.progress

	totalTests := p.lastLane.GetTotalTestsCount()
	totalCalls := p.lastLane.GetTotalCallsCount()
	totalChecks := p.lastLane.GetTotalChecksCount()
	tests := plural("test", totalTests)
	ColorWRN.Println(
		"Ran", totalTests, tests,
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
		d = p.lastLane.GetTotalTestsCount()
		m = 0
		testCalls := p.lastLane.GetTestCallsCount()
		ColorERR.Printf("A bug reproducible in %d HTTP %s was detected after %d",
			testCalls, plural("request", testCalls), d)
		switch {
		case m == 0:
			ColorERR.Printf(" %s and not yet shrunk.\n", tests)
			//TODO: suggest shrinking invocation
			// A task that tries to minimize a testcase to its smallest possible size, such that it still triggers the same underlying bug on the target program.
		case m == 1:
			ColorERR.Printf(" %s then shrunk once.\n", tests)
		default:
			ColorERR.Printf(" %s then shrunk %d %s.\n", tests, m, plural("time", m))
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
