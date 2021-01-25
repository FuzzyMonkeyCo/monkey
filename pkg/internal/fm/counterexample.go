package fm

import (
	"fmt"
)

// ShortString is used to display quick data on a CounterexampleItem
func (ceI *Srv_FuzzingResult_CounterexampleItem) ShortString() (s string) {
	switch x := ceI.GetCallRequest().GetInput().(type) {
	case *Clt_CallRequestRaw_Input_HttpRequest_:
		req := ceI.GetCallRequest().GetHttpRequest()
		rep := ceI.GetCallResponse().GetHttpResponse()
		body := req.GetBody()
		if limit := 97; len(body) > limit {
			body = body[:limit]
			body = append(body, []byte("...")...)
		}
		s = fmt.Sprintf("%d %s \t%s \t%s",
			rep.GetStatusCode(),
			req.GetMethod(),
			req.GetUrl(),
			body,
		)
	default:
		panic(fmt.Sprintf("unhandled CounterexampleItem %T %+v", x, ceI))
	}
	return
}
