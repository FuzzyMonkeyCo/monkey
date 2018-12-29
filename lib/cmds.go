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
	exec(mnk *Monkey) (act Action, err error)
}

func (act *RepValidateProgress) exec(mnk *Monkey) (nxt Action, err error) {
	return
}

func (act *RepCallResult) exec(mnk *Monkey) (nxt Action, err error) {
	return
}
