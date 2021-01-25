package runtime

import (
	"bytes"
	"io"

	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
)

var _ io.Writer = (*progressWriter)(nil)

// progressWriter redirects buffer lines through to a Progresser
type progressWriter struct {
	printf modeler.ShowFunc
}

func newProgressWriter(cb modeler.ShowFunc) *progressWriter {
	return &progressWriter{printf: cb}
}

func (pw *progressWriter) Write(p []byte) (n int, err error) {
	n = len(p)
	for _, pp := range bytes.Split(p, []byte{'\r', '\n'}) {
		for _, ppp := range bytes.Split(pp, []byte{'\n'}) {
			if len(ppp) != 0 {
				// TODO: mux stderr+stdout and fwd to server to track progress
				pw.printf("%s", ppp)
			}
		}
	}
	return
}
