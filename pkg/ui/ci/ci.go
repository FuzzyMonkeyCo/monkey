package ci

import (
	"fmt"

	"github.com/FuzzyMonkeyCo/monkey/pkg/as"
	"github.com/FuzzyMonkeyCo/monkey/pkg/ui"
)

var _ ui.Progresser = (*Progresser)(nil)

type Progresser struct {
	totalTestsCount, totalCallsCount, totalChecksCount uint32
}

func dot(n uint32, o *uint32) {
	if *o != n {
		*o = n
	}
}

func (p *Progresser) MaxTestsCount(v uint32)    {}
func (p *Progresser) Terminate() error          { return nil }
func (p *Progresser) TotalTestsCount(v uint32)  { dot(v, &p.totalTestsCount) }
func (p *Progresser) TotalCallsCount(v uint32)  { dot(v, &p.totalCallsCount) }
func (p *Progresser) TotalChecksCount(v uint32) { dot(v, &p.totalChecksCount) }
func (p *Progresser) TestCallsCount(v uint32)   {}
func (p *Progresser) CallChecksCount(v uint32)  {}

func (p *Progresser) Printf(format string, s ...interface{}) { fmt.Printf(format, s...) }
func (p *Progresser) Errorf(format string, s ...interface{}) { as.ColorERR.Printf(format, s...) }

func (p *Progresser) ChecksPassed() {
	as.ColorNFO.Println(" Checks passed.")
}

func (p *Progresser) CheckPassed(s string) {
	as.ColorOK.Printf("PASSED ")
	as.ColorNFO.Println(s)
}

func (p *Progresser) CheckSkipped(s string) {
	as.ColorWRN.Printf("SKIPPED ")
	as.ColorNFO.Println(s)
}

func (p *Progresser) CheckFailed(ss []string) {
	if len(ss) > 0 {
		as.ColorERR.Printf("FAILED ")
		as.ColorNFO.Println(ss[0])
	}
	if len(ss) > 1 {
		for _, s := range ss[1:] {
			as.ColorERR.Println(s)
		}
	}
	as.ColorNFO.Println(" Found a bug!")
}
