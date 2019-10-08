package pkg

type Monkey struct {
	Cfg      *UserCfg
	Vald     *Validator
	ws       *wsState
	eid      eid
	progress *progress
}

type Action interface {
	// TODO: split into req/rep interfaces
	// FIXME: embed isMsg_Msg
	exec(mnk *Monkey) (err error)
}

func (act *RepValidateProgress) exec(mnk *Monkey) (err error) { return }
func (act *RepCallResult) exec(mnk *Monkey) (err error)       { return }
func (act *RepResetProgress) exec(mnk *Monkey) (err error)    { return }
func (act *RepCallDone) exec(mnk *Monkey) (err error)         { return }
func (act *SUTMetrics) exec(mnk *Monkey) (err error)          { return }
