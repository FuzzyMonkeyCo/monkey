package dots

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

func dot(n uint32, o *uint32, f, c string) {
	if *o != n {
		if *o == 0 {
			fmt.Printf(f)
		} else {
			fmt.Printf(c)
		}
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
func (p *Progresser) TotalTestsCount(v uint32) { dot(v, &p.totalTestsCount, "[", "][") }

// TotalCallsCount may be called many times during testing
func (p *Progresser) TotalCallsCount(v uint32) { dot(v, &p.totalCallsCount, ".", ".") }

// TotalChecksCount may be called many times during testing
func (p *Progresser) TotalChecksCount(v uint32) {}

// TestCallsCount may be called many times during testing
func (p *Progresser) TestCallsCount(v uint32) {}

// CallChecksCount may be called many times during testing
func (p *Progresser) CallChecksCount(v uint32) {}

// Printf formats informational data
func (p *Progresser) Printf(format string, s ...interface{}) {
	if format == "  --seed='%s'" {
		fmt.Printf("\nseed: '%s'\n", s...)
	}
}

// Errorf formats error messages
func (p *Progresser) Errorf(format string, s ...interface{}) {}

// ChecksPassed may be called many times during testing
func (p *Progresser) ChecksPassed() {}

// CheckPassed may be called many times during testing
func (p *Progresser) CheckPassed(name, msg string) { as.ColorOK.Printf("-") }

// CheckSkipped may be called many times during testing
func (p *Progresser) CheckSkipped(name, msg string) { fmt.Printf("-") }

// CheckFailed may be called many times during testing
func (p *Progresser) CheckFailed(name string, ss []string) { as.ColorERR.Println("!") }
