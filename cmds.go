package main

type Req1 struct {
	UID     interface{} `json:"uid"`
	Method  string      `json:"method"`
	Url     string      `json:"url"`
	Headers []string    `json:"headers"`
	Payload *string     `json:"payload"`
}

type RepOK1 struct {
	UID     interface{} `json:"uid"`
	V       uint        `json:"v"`
	Us      uint64      `json:"us"`
	Code    int         `json:"code"`
	Headers []string    `json:"headers"`
	Payload string      `json:"payload"`
}

type RepKO1 struct {
	UID    interface{} `json:"uid"`
	V      uint        `json:"v"`
	Us     uint64      `json:"us"`
	Reason string      `json:"reason"`
}
