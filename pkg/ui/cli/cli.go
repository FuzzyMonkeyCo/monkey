package cli

import (
	"time"

	"github.com/FuzzyMonkeyCo/monkey/pkg/as"
	"github.com/FuzzyMonkeyCo/monkey/pkg/ui"
	"github.com/superhawk610/bar"
	// See also: https://github.com/reconquest/barely
)

var _ ui.Progresser = (*Progresser)(nil)

type Progresser struct {
	maxTestsCount                                      uint32
	totalTestsCount, totalCallsCount, totalChecksCount uint32
	testCallsCount, callChecksCount                    uint32
	bar                                                *bar.Bar
	ticker                                             <-chan time.Time
	stateIdx                                           int
}

func (p *Progresser) MaxTestsCount(v uint32) {
	p.maxTestsCount = v

	p.bar = bar.NewWithOpts(
		// bar.WithDebug(),
		bar.WithDimensions(int(v), 37),
		bar.WithDisplay("", "█", "█", " ", "|"),
		// bar.WithFormat(":state :percent :bar :rate ops/s :eta"),
		bar.WithFormat(":state :bar :rate ops/s :eta"),
	)

	p.ticker = time.Tick(333 * time.Millisecond)
	go func() {
		for range p.ticker {
			p.state(0)
		}
	}()
}

func (p *Progresser) Terminate() error {
	p.ticker = nil
	p.bar.Done()
	return nil
}

func (p *Progresser) TotalTestsCount(v uint32) { p.totalTestsCount = v }
func (p *Progresser) TotalCallsCount(v uint32) {
	if p.totalCallsCount != v {
		p.state(1)
	}
	p.totalCallsCount = v
}
func (p *Progresser) TotalChecksCount(v uint32) { p.totalChecksCount = v }
func (p *Progresser) TestCallsCount(v uint32)   { p.testCallsCount = v }
func (p *Progresser) CallChecksCount(v uint32)  { p.callChecksCount = v }

func (p *Progresser) state(inc int) {
	state := cliStates[p.stateIdx%len(cliStates)]
	p.stateIdx++
	advancement := inc + int(p.totalCallsCount)
	p.bar.Update(advancement, bar.Context{bar.Ctx("state", state)})
}

func (p *Progresser) Printf(format string, s ...interface{}) { p.bar.Interruptf(format, s...) }
func (p *Progresser) Errorf(format string, s ...interface{}) {
	p.bar.Interruptf("%s", as.ColorERR.Sprintf(format, s...))
}

func (p *Progresser) show(s string) { p.bar.Interrupt(s) }
func (p *Progresser) nfo(s string)  { p.show(as.ColorNFO.Sprintf("%s", s)) }
func (p *Progresser) wrn(s string)  { p.show(as.ColorWRN.Sprintf("%s", s)) }
func (p *Progresser) err(s string)  { p.show(as.ColorERR.Sprintf("%s", s)) }

func (p *Progresser) ChecksPassed() { p.nfo(" Checks passed.\n") }
func (p *Progresser) CheckPassed(s string) {
	p.show(" " + as.ColorOK.Sprintf(prefixSucceeded) + " " + as.ColorNFO.Sprintf(s))
}
func (p *Progresser) CheckSkipped(s string) {
	p.show(" " + as.ColorWRN.Sprintf(prefixSkipped) + " " + as.ColorNFO.Sprintf(s) + " skipped")
}
func (p *Progresser) CheckFailed(ss []string) {
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
