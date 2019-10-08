package ui

import (
	"time"

	"github.com/FuzzyMonkeyCo/monkey/pkg/as"
	"github.com/superhawk610/bar"
)

const (
	prefixSucceeded = "●" // ✔ ✓ 🆗 👌 ☑ ✅
	prefixSkipped   = "○" // ● • ‣ ◦ ⁃ ○ ◯ ⭕ 💮
	prefixFailed    = "✖" // ⨯ × ✗ x X ☓ ✘
)

var _ Progresser = (*cliProgress)(nil)

type cliProgress struct {
	maxTestsCount                                      uint32
	totalTestsCount, totalCallsCount, totalChecksCount uint32
	testCallsCount, callChecksCount                    uint32
	bar                                                *bar.Bar
	failed                                             bool
	start                                              time.Time
}

// NewCLIProgress TODO
func NewCLIProgress() *cliProgress {
	return &cliProgress{}
}

func (p *cliProgress) MaxTestsCount(v uint32) {
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

func (p *cliProgress) TotalTestsCount(v uint32)  { p.totalTestsCount = v }
func (p *cliProgress) TotalCallsCount(v uint32)  { p.totalCallsCount = v }
func (p *cliProgress) TotalChecksCount(v uint32) { p.totalChecksCount = v }
func (p *cliProgress) TestCallsCount(v uint32)   { p.testCallsCount = v }
func (p *cliProgress) CallChecksCount(v uint32)  { p.callChecksCount = v }

func (p *cliProgress) state(s string) {
	advancement := 1 + p.totalCallsCount
	p.bar.Update(int(advancement), bar.Context{bar.Ctx("state", s)})
}
func (p *cliProgress) show(s string)                         { p.bar.Interrupt(s) }
func (p *cliProgress) Showf(format string, s ...interface{}) { p.bar.Interruptf(format, s...) }
func (p *cliProgress) nfo(s string)                          { p.show(as.ColorNFO.Sprintf("%s", s)) }
func (p *cliProgress) wrn(s string)                          { p.show(as.ColorWRN.Sprintf("%s", s)) }
func (p *cliProgress) err(s string) {
	p.show(as.ColorERR.Sprintf("%s", s))
	p.failed = true
}

func (p *cliProgress) ChecksPassed() { p.nfo(" All checks passed.\n") }
func (p *cliProgress) CheckPassed(s string) {
	p.show(" " + as.ColorOK.Sprintf(prefixSucceeded) + " " + as.ColorNFO.Sprintf(s))
}
func (p *cliProgress) CheckSkipped(s string) {
	p.show(" " + as.ColorWRN.Sprintf(prefixSkipped) + " " + as.ColorNFO.Sprintf(s) + " skipped")
}
func (p *cliProgress) CheckFailed(ss []string) {
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
