package fm

import (
	"fmt"
	"strings"
)

// CLIString is used to display quick data on a CounterexampleItem
func (ceI *Srv_FuzzingResult_CounterexampleItem) CLIString() (s string) {
	switch x := ceI.GetCallRequest().GetInput().(type) {
	case *Clt_CallRequestRaw_Input_HttpRequest_:
		req := ceI.GetCallRequest().GetHttpRequest()
		rep := ceI.GetCallResponse().GetHttpResponse()

		var b strings.Builder
		indent := func() { b.WriteString(" \\\n     ") }
		b.WriteString("curl -#fsSL -X ")
		b.WriteString(req.GetMethod())
		indent()
		for _, kvs := range req.GetHeaders() {
			values := strings.Join(kvs.GetValues(), ",")
			switch key := kvs.GetKey(); key {
			case "User-Agent":
				b.WriteString("-A ")
				b.WriteString(shellEscape(values))
			default:
				b.WriteString("-H ")
				b.WriteString(shellEscape(fmt.Sprintf("%s: %s", key, values)))
			}
			indent()
		}
		if body := req.GetBody(); len(body) != 0 {
			b.WriteString("-d ")
			b.WriteString(shellEscape(string(body)))
			indent()
		}
		b.WriteString(shellEscape(req.GetUrl()))
		b.WriteString("\n")
		b.WriteString("# ")
		b.WriteString(rep.GetReason())
		s = b.String()
	default:
		panic(fmt.Sprintf("unhandled CounterexampleItem %T %+v", x, ceI))
	}
	return
}

func shellEscape(s string) string {
	return `'` + strings.ReplaceAll(s, `'`, `'\''`) + `'`
}
