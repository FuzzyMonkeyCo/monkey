package cli

import (
	"fmt"
	"time"

	"github.com/FuzzyMonkeyCo/monkey/pkg/as"
)

// CampaignSummary concludes the testing campaing and reports to the user.
func (p *Cli) CampaignSummary() (success bool) {
	fmt.Println()
	fmt.Println()
	as.ColorWRN.Println(
		"Ran", p.totalTestsCount, plural("test", p.totalTestsCount),
		"totalling", p.totalCallsCount, plural("request", p.totalCallsCount),
		"and", p.totalChecksCount, plural("check", p.totalChecksCount),
		"in", time.Since(p.start))

	if p.campaignSuccess {
		success = true
		as.ColorNFO.Println("No bugs found... yet.")
		return
	}

	as.ColorERR.Printf("A bug reproducible in %d HTTP %s was detected after %d",
		p.testCallsCount, plural("request", p.testCallsCount), p.totalTestsCount)
	var m uint32 // FIXME: handle shrinking report
	switch {
	case p.totalTestsCount == 1:
		as.ColorERR.Printf(" %s.\n", plural("test", p.totalTestsCount))
	case m == 0:
		as.ColorERR.Printf(" %s and not yet shrunk.\n", plural("test", p.totalTestsCount))
		//TODO: suggest shrinking invocation
		// A task that tries to minimize a testcase to its smallest possible size, such that it still triggers the same underlying bug on the target program.
	case m == 1:
		as.ColorERR.Printf(" %s then shrunk once.\n", plural("test", p.totalTestsCount))
	default:
		as.ColorERR.Printf(" %s then shrunk %d %s.\n", plural("test", p.totalTestsCount), m, plural("time", m))
	}
	return
}

func plural(s string, n uint32) string {
	if n == 1 {
		return s
	}
	return s + "s"
}
