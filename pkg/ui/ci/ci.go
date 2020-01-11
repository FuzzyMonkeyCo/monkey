package ci

import (
	"fmt"

	"github.com/FuzzyMonkeyCo/monkey/pkg/as"
	"github.com/FuzzyMonkeyCo/monkey/pkg/ui"
)

const (
	cTest  = "●"
	cCall  = "•"
	cCheck = "."
)

var _ ui.Progresser = (*Progresser)(nil)

type Progresser struct {
	totalTestsCount, totalCallsCount, totalChecksCount uint32
}

func dot(s string, n uint32, o *uint32) {
	if *o != n {
		fmt.Printf(s)
		*o = n
	}
}

func (p *Progresser) MaxTestsCount(v uint32)    {}
func (p *Progresser) Terminate() error          { return nil }
func (p *Progresser) TotalTestsCount(v uint32)  { dot(cTest, v, &p.totalTestsCount) }
func (p *Progresser) TotalCallsCount(v uint32)  { dot(cCall, v, &p.totalCallsCount) }
func (p *Progresser) TotalChecksCount(v uint32) { dot(cCheck, v, &p.totalChecksCount) }
func (p *Progresser) TestCallsCount(v uint32)   {}
func (p *Progresser) CallChecksCount(v uint32)  {}

func (p *Progresser) Printf(format string, s ...interface{}) { fmt.Printf(format, s...) }
func (p *Progresser) Errorf(format string, s ...interface{}) { as.ColorERR.Printf(format, s...) }

func (p *Progresser) ChecksPassed() {
	as.ColorNFO.Println(" Checks passed.")
}

func (p *Progresser) CheckPassed(s string) {
	fmt.Println(" " + as.ColorOK.Sprintf("PASSED") + " " + as.ColorNFO.Sprintf(s))
}

func (p *Progresser) CheckSkipped(s string) {
	fmt.Println(" " + as.ColorWRN.Sprintf("SKIPPED") + " " + as.ColorNFO.Sprintf(s) + " skipped")
}

func (p *Progresser) CheckFailed(ss []string) {
	if len(ss) > 0 {
		fmt.Println(" " + as.ColorERR.Sprintf("FAILED") + " " + as.ColorNFO.Sprintf(ss[0]))
	}
	if len(ss) > 1 {
		for _, s := range ss[1:] {
			as.ColorERR.Println(s)
		}
	}
	as.ColorNFO.Println(" Found a bug!")
}
