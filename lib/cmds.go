package lib

import (
	"time"
)

type Monkey struct {
	Cfg      *UserCfg
	Vald     *Validator
	Name     string
	EID      uint32
	Progress *Progress
}

type Progress struct {
	Start         time.Time
	LastLane      Lane
	ShrinkingFrom Lane
	TotalR        uint32
	TotalC        uint32
}

type Lane struct {
	T uint32
	R uint32
	C uint32
}

func NewMonkey(cfg *UserCfg, vald *Validator, name string) *Monkey {
	return &Monkey{
		Cfg:  cfg,
		Vald: vald,
		Name: name,
		Progress: &Progress{
			Start: time.Now(),
		},
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
