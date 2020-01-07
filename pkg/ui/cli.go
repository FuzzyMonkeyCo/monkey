package ui

import (
	"time"

	"github.com/FuzzyMonkeyCo/monkey/pkg/as"
	"github.com/superhawk610/bar"
	// See also: https://github.com/reconquest/barely
)

var _ Progresser = (*Cli)(nil)

type Cli struct {
	maxTestsCount                                      uint32
	totalTestsCount, totalCallsCount, totalChecksCount uint32
	testCallsCount, callChecksCount                    uint32
	bar                                                *bar.Bar
	failed                                             bool
	campaignSuccess                                    bool
	start                                              time.Time
	ticker                                             <-chan time.Time
	stateIdx                                           int
}

// NewCli TODO
func NewCli() *Cli {
	return &Cli{}
}

func (p *Cli) MaxTestsCount(v uint32) {
	p.maxTestsCount = v
	p.bar = bar.NewWithOpts(
		// bar.WithDebug(),
		bar.WithDimensions(int(v), 37),
		bar.WithDisplay("", "█", "█", " ", "|"),
		// bar.WithFormat(":state :percent :bar :rate ops/s :eta"),
		bar.WithFormat(":state :bar :rate ops/s :eta"),
	)
	p.start = time.Now()
	p.ticker = time.Tick(333 * time.Millisecond)
	go func() {
		for range p.ticker {
			p.state(0)
		}
	}()
}

func (p *Cli) Terminate() error {
	p.ticker = nil
	p.bar.Done()
	return nil
}

func (p *Cli) TotalTestsCount(v uint32) { p.totalTestsCount = v }
func (p *Cli) TotalCallsCount(v uint32) {
	if p.totalCallsCount != v {
		p.state(1)
	}
	p.totalCallsCount = v
}
func (p *Cli) TotalChecksCount(v uint32) { p.totalChecksCount = v }
func (p *Cli) TestCallsCount(v uint32)   { p.testCallsCount = v }
func (p *Cli) CallChecksCount(v uint32)  { p.callChecksCount = v }
func (p *Cli) CampaignSuccess(v bool)    { p.campaignSuccess = v }

func (p *Cli) state(inc int) {
	state := cliStates[p.stateIdx%len(cliStates)]
	p.stateIdx++
	advancement := inc + int(p.totalCallsCount)
	p.bar.Update(advancement, bar.Context{bar.Ctx("state", state)})
}

func (p *Cli) Printf(format string, s ...interface{}) { p.bar.Interruptf(format, s...) }
func (p *Cli) Errorf(format string, s ...interface{}) {
	p.bar.Interruptf("%s", as.ColorERR.Sprintf(format, s))
}

func (p *Cli) show(s string) { p.bar.Interrupt(s) }
func (p *Cli) nfo(s string)  { p.show(as.ColorNFO.Sprintf("%s", s)) }
func (p *Cli) wrn(s string)  { p.show(as.ColorWRN.Sprintf("%s", s)) }
func (p *Cli) err(s string) {
	p.show(as.ColorERR.Sprintf("%s", s))
	p.failed = true
}

func (p *Cli) ChecksPassed() { p.nfo(" All checks passed.\n") }
func (p *Cli) CheckPassed(s string) {
	p.show(" " + as.ColorOK.Sprintf(prefixSucceeded) + " " + as.ColorNFO.Sprintf(s))
}
func (p *Cli) CheckSkipped(s string) {
	p.show(" " + as.ColorWRN.Sprintf(prefixSkipped) + " " + as.ColorNFO.Sprintf(s) + " skipped")
}
func (p *Cli) CheckFailed(ss []string) {
	p.failed = true
	if len(ss) > 0 {
		p.show(" " + as.ColorERR.Sprintf(prefixFailed) + " " + as.ColorNFO.Sprintf(ss[0]))
	}
	if len(ss) > 1 {
		for _, s := range ss[1:] {
			p.show(as.ColorERR.Sprintf(s))
		}
	}
	p.nfo(" Found a bug!\n")
}
