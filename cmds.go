package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
)

var (
	lastLane      lane
	shrinkingFrom lane
	totalR        uint
)

type lane struct {
	T uint `json:"t"`
	R uint `json:"r"`
}

type someCmd interface {
	Kind() cmdKind
	Exec(cfg *ymlCfg) (rep []byte, err error)
}

type cmdKind int

const (
	kindStart cmdKind = iota
	kindReq
	kindReset
	kindStop
	kindDone
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

func unmarshalCmd(cmdJSON []byte) (cmd someCmd, err error) {
	var ok bool

	if ok, err = isValidForSchemaREQv1(cmdJSON); err != nil {
		return
	}
	if ok {
		var cmd reqCmd
		if err = json.Unmarshal(cmdJSON, &cmd); err != nil {
			log.Println("[ERR]", err)
			return nil, err
		}
		return &cmd, nil
	}

	if ok, err = isValidForSchemaCMDv1(cmdJSON); err != nil {
		return
	}
	if ok {
		var cmd rstCmd
		if err = json.Unmarshal(cmdJSON, &cmd); err != nil {
			log.Println("[ERR]", err)
			return nil, err
		}
		return &cmd, nil
	}

	if ok, err = isValidForSchemaCMDDonev1(cmdJSON); err != nil {
		return
	}
	if ok {
		var cmd doneCmd
		if err = json.Unmarshal(cmdJSON, &cmd); err != nil {
			log.Println("[ERR]", err)
			return nil, err
		}
		return &cmd, nil
	}

	err = errors.New("invalid JSON data received")
	log.Println("[ERR]", err)
	return
}
