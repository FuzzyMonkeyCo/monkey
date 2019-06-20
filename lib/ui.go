package lib

import (
	"log"
	"time"

	"github.com/superhawk610/bar"
)

type progress struct {
	bar           *bar.Bar
	failed        bool
	start         time.Time
	lastLane      FuzzProgress
	shrinkingFrom *FuzzProgress
}

func newProgress(n uint32) *progress {
	return &progress{
		bar: bar.NewWithOpts(
			// bar.WithDebug(),
			bar.WithDimensions(int(n), 37),
			bar.WithDisplay("", "█", "█", " ", "|"),
			// bar.WithFormat(":state :percent :bar :rate ops/s :eta"),
			bar.WithFormat(":state :bar :rate ops/s :eta"),
		),
		start: time.Now(),
	}
}

func (p *progress) dbg(s string) { p.bar.Interrupt(s) }
func (p *progress) nfo(s string) { p.bar.Interrupt(ColorNFO.Sprintf("%s", s)) }
func (p *progress) wrn(s string) { p.bar.Interrupt(ColorWRN.Sprintf("%s", s)) }
func (p *progress) err(s string) {
	p.bar.Interrupt(ColorERR.Sprintf("%s", s))
	p.failed = true
}
func (p *progress) state(s string) {
	p.bar.Update(int(p.lastLane.GetTotalTestsCount()),
		bar.Context{bar.Ctx("state", s)})
}

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

	///////SO let's compute a diff of act - lastLane and display in accordance
	/////// be smart on diffing bools!
	if act.GetShrinking() && mnk.progress.shrinkingFrom == nil {
		mnk.progress.shrinkingFrom = &mnk.progress.lastLane
		str += "Shrinking: "
	}
	// case act.GetTotalTestsCount() == 0:
	// 	// Avoids getting in below case
	// case act.GetTotalTestsCount() != mnk.progress.lastLane.GetTotalTestsCount():
	// 	str += "]["

	// fmt.Print(str)
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
			//TODO: suggest shrinking invocation
			// A task that tries to minimize a testcase to its smallest possible size, such that it still triggers the same underlying bug on the target program.
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
