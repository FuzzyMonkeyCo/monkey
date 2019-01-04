package lib

import (
	"time"
)

type Monkey struct {
	Cfg      *UserCfg
	Vald     *Validator
	Name     string
	ws       *wsState
	eid      uint32
	progress *progress
}

func NewMonkey(cfg *UserCfg, vald *Validator, name string) *Monkey {
	return &Monkey{
		Cfg:  cfg,
		Vald: vald,
		Name: name,
	}
}

type progress struct {
	start         time.Time
	lastLane      lane
	shrinkingFrom lane
	totalR        uint32
	totalC        uint32
}

func newProgress() *progress {
	return &progress{
		start: time.Now(),
	}
}

type lane struct {
	t uint32
	r uint32
	c uint32
}

type Action interface {
	// TODO: isMsg_Msg() + split into req/rep interfaces
	exec(mnk *Monkey) (err error)
}

func (act *RepValidateProgress) exec(mnk *Monkey) (err error) {
	return
}

func (act *RepCallResult) exec(mnk *Monkey) (err error) {
	return
}

func (act *RepResetProgress) exec(mnk *Monkey) (err error) {
	return
}

func (act *RepCallDone) exec(mnk *Monkey) (err error) {
	return
}

func (act *SUTMetrics) exec(mnk *Monkey) (err error) {
	return
}
