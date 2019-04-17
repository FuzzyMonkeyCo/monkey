package lib

import (
	"time"
)

type Monkey struct {
	Cfg      *UserCfg
	Vald     *Validator
	Name     string
	ws       *wsState
	eid      eid
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
	lastLane      FuzzProgress
	shrinkingFrom *FuzzProgress
}

func newProgress() *progress {
	return &progress{
		start: time.Now(),
	}
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
