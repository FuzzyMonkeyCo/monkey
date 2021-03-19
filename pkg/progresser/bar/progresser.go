package bar

import (
	"context"
	"time"

	"github.com/FuzzyMonkeyCo/monkey/pkg/as"
	"github.com/FuzzyMonkeyCo/monkey/pkg/progresser"
	pbar "github.com/superhawk610/bar"
	// See also: https://github.com/reconquest/barely
	// See also: https://github.com/snapcore/snapd/tree/3178a5499f2605329ebd25c7293ae1a0fb9fbd3b/progress
)

const tickEvery = 333 * time.Millisecond

var _ progresser.Interface = (*Progresser)(nil)

// Progresser implements progresser.Interface
type Progresser struct {
	ctx                                                context.Context
	maxTestsCount                                      uint32
	totalTestsCount, totalCallsCount, totalChecksCount uint32
	testCallsCount, callChecksCount                    uint32
	bar                                                *pbar.Bar
	ticker                                             *time.Ticker
	ticks, stateIdx                                    int
}

// WithContext sets ctx of a progresser.Interface implementation
func (p *Progresser) WithContext(ctx context.Context) { p.ctx = ctx }

// MaxTestsCount sets an upper bound before testing starts
func (p *Progresser) MaxTestsCount(v uint32) {
	p.maxTestsCount = v

	p.bar = pbar.NewWithOpts(
		// pbar.WithDebug(),
		pbar.WithDimensions(int(v), 37),
		pbar.WithDisplay("", "█", "█", " ", "|"),
		// pbar.WithFormat(":state :percent :bar :rate ops/s :eta"),
		pbar.WithFormat(":state :bar :rate calls/s :eta"),
	)

	p.ticker = time.NewTicker(tickEvery)
	go func() {
		defer p.ticker.Stop()
		for {
			select {
			case <-p.ctx.Done():
				return
			case <-p.ticker.C:
				p.tick(0)
			}
		}
	}()
}

// Terminate cleans up after a progresser.Interface implementation instance
func (p *Progresser) Terminate() error {
	p.ticker.Stop()
	p.bar.Done()
	return nil
}

// TotalTestsCount may be called many times during testing
func (p *Progresser) TotalTestsCount(v uint32) { p.totalTestsCount = v }

// TotalCallsCount may be called many times during testing
func (p *Progresser) TotalCallsCount(v uint32) {
	if p.totalCallsCount != v {
		p.tick(1)
	}
	p.totalCallsCount = v
}

// TotalChecksCount may be called many times during testing
func (p *Progresser) TotalChecksCount(v uint32) { p.totalChecksCount = v }

// TestCallsCount may be called many times during testing
func (p *Progresser) TestCallsCount(v uint32) { p.testCallsCount = v }

// CallChecksCount may be called many times during testing
func (p *Progresser) CallChecksCount(v uint32) { p.callChecksCount = v }

func (p *Progresser) tick(offset int) {
	state := states[p.stateIdx%len(states)]
	p.stateIdx++
	p.ticks += offset
	p.bar.Update(p.ticks, pbar.Context{pbar.Ctx("state", state)})
}

// Printf formats informational data
func (p *Progresser) Printf(format string, s ...interface{}) {
	p.bar.Interruptf(format, s...)
}

// Errorf formats error messages
func (p *Progresser) Errorf(format string, s ...interface{}) {
	p.bar.Interruptf("%s", as.ColorERR.Sprintf(format, s...))
}

func (p *Progresser) show(s string) { p.bar.Interrupt(s) }
func (p *Progresser) nfo(s string)  { p.show(as.ColorNFO.Sprintf("%s", s)) }
func (p *Progresser) wrn(s string)  { p.show(as.ColorWRN.Sprintf("%s", s)) }
func (p *Progresser) err(s string)  { p.show(as.ColorERR.Sprintf("%s", s)) }

// ChecksPassed may be called many times during testing
func (p *Progresser) ChecksPassed() {
	p.nfo(" Checks passed.\n")
}

// CheckPassed may be called many times during testing
func (p *Progresser) CheckPassed(name, msg string) {
	if msg != "" {
		if msg == "<function lambda>" { // name of user check
			msg = ""
		} else {
			msg = ": " + msg
		}
	}
	p.bar.Interruptf(" %s %s%s",
		as.ColorOK.Sprintf(prefixSucceeded),
		as.ColorNFO.Sprintf(name),
		msg,
	)
}

// CheckSkipped may be called many times during testing
func (p *Progresser) CheckSkipped(name, msg string) {
	if msg != "" {
		if msg == "<function lambda>" { // name of user check
			msg = ""
		} else {
			msg = ": " + msg
		}
	}
	p.bar.Interruptf(" %s %s SKIPPED%s",
		as.ColorWRN.Sprintf(prefixSkipped),
		name,
		msg,
	)
}

// CheckFailed may be called many times during testing
func (p *Progresser) CheckFailed(name string, ss []string) {
	if len(ss) > 0 {
		p.show(" " + as.ColorERR.Sprintf(prefixFailed) + " " + as.ColorNFO.Sprintf(ss[0]))
	}
	if len(ss) > 1 {
		for _, s := range ss[1:] {
			p.err(s)
		}
	}
	p.nfo(" Found a bug!\n")
}
