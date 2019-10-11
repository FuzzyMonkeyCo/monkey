package ui

import (
	"time"

	"github.com/FuzzyMonkeyCo/monkey/pkg/as"
	"github.com/superhawk610/bar"
)

var _ Progresser = (*Cli)(nil)

type Cli struct {
	maxTestsCount                                      uint32
	totalTestsCount, totalCallsCount, totalChecksCount uint32
	testCallsCount, callChecksCount                    uint32
	bar                                                *bar.Bar
	failed                                             bool
	start                                              time.Time
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
}

func (p *Cli) TotalTestsCount(v uint32)  { p.totalTestsCount = v }
func (p *Cli) TotalCallsCount(v uint32)  { p.totalCallsCount = v }
func (p *Cli) TotalChecksCount(v uint32) { p.totalChecksCount = v }
func (p *Cli) TestCallsCount(v uint32)   { p.testCallsCount = v }
func (p *Cli) CallChecksCount(v uint32)  { p.callChecksCount = v }

func (p *Cli) state(s string) {
	advancement := 1 + p.totalCallsCount
	p.bar.Update(int(advancement), bar.Context{bar.Ctx("state", s)})
}
func (p *Cli) show(s string)                         { p.bar.Interrupt(s) }
func (p *Cli) Showf(format string, s ...interface{}) { p.bar.Interruptf(format, s...) }
func (p *Cli) nfo(s string)                          { p.show(as.ColorNFO.Sprintf("%s", s)) }
func (p *Cli) wrn(s string)                          { p.show(as.ColorWRN.Sprintf("%s", s)) }
func (p *Cli) err(s string) {
	p.show(as.ColorERR.Sprintf("%s", s))
	p.failed = true
}

func (p *Cli) ChecksPassed() { p.nfo(" All checks passed.\n") }
func (p *Cli) CheckPassed(s string) {
	const prefixSucceeded = "●" // ✔ ✓ 🆗 👌 ☑ ✅
	p.show(" " + as.ColorOK.Sprintf(prefixSucceeded) + " " + as.ColorNFO.Sprintf(s))
}
func (p *Cli) CheckSkipped(s string) {
	const prefixSkipped = "○" // ● • ‣ ◦ ⁃ ○ ◯ ⭕ 💮
	p.show(" " + as.ColorWRN.Sprintf(prefixSkipped) + " " + as.ColorNFO.Sprintf(s) + " skipped")
}
func (p *Cli) CheckFailed(ss []string) {
	p.failed = true
	if len(ss) > 0 {
		const prefixFailed = "✖" // ⨯ × ✗ x X ☓ ✘
		p.show(" " + as.ColorERR.Sprintf(prefixFailed) + " " + as.ColorNFO.Sprintf(ss[0]))
	}
	if len(ss) > 1 {
		for _, s := range ss[1:] {
			p.show(as.ColorERR.Sprintf(s))
		}
	}
	p.nfo(" Found a bug!\n")
}

func (p *Cli) Terminate() error {
	p.bar.Done()
	return nil
}
