package runtime

import (
	"bytes"
	"io"
)

var _ io.Writer = (*progressWriter)(nil)

// progressWriter redirects buffer lines through to a Progresser
type progressWriter struct {
	printf func(string, ...interface{})
}

func newProgressWriter(cb func(string, ...interface{})) *progressWriter {
	return &progressWriter{printf: cb}
}

func (pw *progressWriter) Write(p []byte) (n int, err error) {
	n = len(p)
	pps := bytes.Split(p, []byte{'\r', '\n'})
	for _, pp := range pps {
		ppps := bytes.Split(pp, []byte{'\n'})
		for _, ppp := range ppps {
			// TODO: mux stderr+stdout and fwd to server to track progress
			pw.printf("%s", ppp)
		}
	}
	return
}
