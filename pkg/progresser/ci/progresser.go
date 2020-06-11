package ci

import (
	"context"
	"fmt"

	"github.com/FuzzyMonkeyCo/monkey/pkg/as"
	"github.com/FuzzyMonkeyCo/monkey/pkg/progresser"
)

var _ progresser.Interface = (*Progresser)(nil)

// Progresser implements progresser.Interface
type Progresser struct {
	totalTestsCount, totalCallsCount, totalChecksCount uint32
}

func dot(n uint32, o *uint32) {
	if *o != n {
		*o = n
	}
}

// WithContext sets ctx of a progresser.Interface implementation
func (p *Progresser) WithContext(ctx context.Context) {}

// MaxTestsCount sets an upper bound before testing starts
func (p *Progresser) MaxTestsCount(v uint32) {}

// Terminate cleans up after a progresser.Interface implementation instance
func (p *Progresser) Terminate() error { return nil }

// TotalTestsCount may be called many times during testing
func (p *Progresser) TotalTestsCount(v uint32) { dot(v, &p.totalTestsCount) }

// TotalCallsCount may be called many times during testing
func (p *Progresser) TotalCallsCount(v uint32) { dot(v, &p.totalCallsCount) }

// TotalChecksCount may be called many times during testing
func (p *Progresser) TotalChecksCount(v uint32) { dot(v, &p.totalChecksCount) }

// TestCallsCount may be called many times during testing
func (p *Progresser) TestCallsCount(v uint32) {}

// CallChecksCount may be called many times during testing
func (p *Progresser) CallChecksCount(v uint32) {}

// Printf formats informational data
func (p *Progresser) Printf(format string, s ...interface{}) { fmt.Printf(format+"\n", s...) }

// Errorf formats error messages
func (p *Progresser) Errorf(format string, s ...interface{}) { as.ColorERR.Printf(format+"\n", s...) }

// ChecksPassed may be called many times during testing
func (p *Progresser) ChecksPassed() {
	as.ColorOK.Println("PASSED CHECKS")
}

// CheckPassed may be called many times during testing
func (p *Progresser) CheckPassed(name, msg string) {
	as.ColorOK.Printf("PASSED ")
	as.ColorNFO.Printf("%s", name)
	if msg != "" {
		fmt.Printf(" (%s)", msg)
	}
	fmt.Println()
}

// CheckSkipped may be called many times during testing
func (p *Progresser) CheckSkipped(name, msg string) {
	as.ColorWRN.Printf("SKIPPED ")
	as.ColorNFO.Printf("%s", name)
	if msg != "" {
		fmt.Printf(" (%s)", msg)
	}
	fmt.Println()
}

// CheckFailed may be called many times during testing
func (p *Progresser) CheckFailed(name string, ss []string) {
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
