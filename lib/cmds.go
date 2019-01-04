package lib

var (
	lastLane      lane
	shrinkingFrom lane
	totalR        uint32
)

type lane struct {
	T uint32
	R uint32
}

type Monkey struct {
	Cfg  *UserCfg
	Vald *Validator
	Name string
	EID  uint32
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
