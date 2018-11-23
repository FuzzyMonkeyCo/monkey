package main

import (
	"encoding/json"
	// "errors"
	"fmt"
	// "log"
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

type action interface {
	exec(mnk *monkey) (act action, err error)
}

type someCmd interface {
	kind() cmdKind
	exec(cfg *UserCfg) (rep []byte, err error)
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
