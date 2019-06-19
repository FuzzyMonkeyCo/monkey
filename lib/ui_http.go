package lib

import (
	"net/http"
	"net/http/httputil"
)

func (mnk *Monkey) showRequest(r *http.Request) error {
	// TODO: output `curl` requests when showing counterexample
	//   https://github.com/sethgrid/gencurl
	//   https://github.com/moul/http2curl
	dump, err := httputil.DumpRequestOut(r, false)
	if err != nil {
		// MUST log in caller
		return err
	}
	// ColorNFO.Printf("%s", dump)
	mnk.progress.bar.Interrupt(ColorNFO.Sprintf("%s", dump))
	return nil
}

func (mnk *Monkey) showResponse(r *http.Response, e string) error {
	if r == nil {
		ColorERR.Println(e)
		return nil
	}

	dump, err := httputil.DumpResponse(r, false)
	if err != nil {
		// MUST log in caller
		return err
	}
	// ColorWRN.Printf("%s", dump)
	mnk.progress.bar.Interrupt(ColorWRN.Sprintf("%s", dump))
	return nil
}
