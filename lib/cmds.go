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

type Action interface {
	exec(mnk *Monkey) (act Action, err error)
}

func (act *RepValidateProgress) exec(mnk *Monkey) (nxt Action, err error) {
	return
}

func (act *ReqDoValidate) exec(mnk *Monkey) (nxt Action, err error) {
	// FIXME: use .Anon?
	nxt = &RepValidateProgress{Failure: false, Success: true}
	return
}
