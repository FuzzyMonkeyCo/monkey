package lib

import (
	"encoding/json"
	"fmt"
)

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

type cmdKind int

const (
	kindDone cmdKind = iota
	kindReq
	kindReset
	kindStart
	kindStop
)

func (k cmdKind) String() string {
	return map[cmdKind]string{
		kindReq:   "req",
		kindReset: "reset",
		kindStart: "start",
		kindStop:  "stop",
		kindDone:  "done",
	}[k]
}

func (k *cmdKind) UnmarshalJSON(data []byte) (err error) {
	var cmd string
	if err = json.Unmarshal(data, &cmd); err != nil {
		return
	}

	if kind, ok := map[string]cmdKind{
		"req":   kindReq,
		"reset": kindReset,
		"done":  kindDone,
	}[cmd]; ok {
		*k = kind
		return
	}
	err = fmt.Errorf("expected one of req reset done, not %s", cmd)
	return
}

func (k cmdKind) MarshalJSON() (data []byte, err error) {
	var ok bool
	if data, ok = map[cmdKind][]byte{
		kindReq:   []byte(`"req"`),
		kindReset: []byte(`"reset"`),
		kindStart: []byte(`"start"`),
		kindStop:  []byte(`"stop"`),
		kindDone:  []byte(`"done"`),
	}[k]; ok {
		return
	}
	err = fmt.Errorf("impossibru %#v", k)
	return
}

func (act *RepValidateProgress) exec(mnk *Monkey) (nxt Action, err error) {
	return
}

func (act *ReqDoValidate) exec(mnk *Monkey) (nxt Action, err error) {
	// FIXME: use .Anon?
	nxt = &RepValidateProgress{Failure: false, Success: true}
	return
}
