package main

// ClientAuthOptions has priviledges for ClientAuth
type ClientAuthOptions struct {
	Tests uint32 `json:"t"`
}

// ClientAuth is sent to verify rights
type ClientAuth struct {
	VSN     uint32            `json:"v"`
	Client  string            `json:"c"`
	Options ClientAuthOptions `json:"o"`
}
