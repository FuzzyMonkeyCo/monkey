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
	totalTestsCount, totalCallsCount uint32
	dotting                          bool
}

func (p *Progresser) dot(n uint32, o *uint32, f, c string) {
	if *o != n {
		p.dotting = true
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
func (p *Progresser) Terminate() error {
	p.dotting = false
	fmt.Printf(">")
	return nil
}

// TotalTestsCount may be called many times during testing
func (p *Progresser) TotalTestsCount(v uint32) { p.dot(v, &p.totalTestsCount, "<", "> <") }

// TotalCallsCount may be called many times during testing
func (p *Progresser) TotalCallsCount(v uint32) { p.dot(v, &p.totalCallsCount, ".", ".") }

// TotalChecksCount may be called many times during testing
func (p *Progresser) TotalChecksCount(v uint32) {}

// TestCallsCount may be called many times during testing
func (p *Progresser) TestCallsCount(v uint32) {}

// CallChecksCount may be called many times during testing
func (p *Progresser) CallChecksCount(v uint32) {}

// Printf formats informational data
func (p *Progresser) Printf(format string, s ...interface{}) {
	switch format {
	case "  --seed=%s":
		fmt.Printf("\nseed: %s\n", s...)
	}
}

// Errorf formats error messages
func (p *Progresser) Errorf(format string, s ...interface{}) {
	if p.dotting {
		fmt.Println()
		p.dotting = false
	}
	if format[len(format)-1] != '\n' {
		format = format + "\n"
	}
	fmt.Printf(format, s...)
}

// ChecksPassed may be called many times during testing
func (p *Progresser) ChecksPassed() {}

// CheckPassed may be called many times during testing
func (p *Progresser) CheckPassed(name, msg string) {
	p.dotting = true
	as.ColorOK.Printf("|")
}

// CheckSkipped may be called many times during testing
func (p *Progresser) CheckSkipped(name, msg string) {}

// CheckFailed may be called many times during testing
func (p *Progresser) CheckFailed(name string, ss []string) {
	as.ColorERR.Println("x")
	as.ColorNFO.Printf("Check failed: ")
	fmt.Println(name)
	for i, s := range ss {
		if i != 0 {
			as.ColorERR.Println(s)
		}
	}
}
