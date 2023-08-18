package shell

import (
	"bytes"
	"io"
)

var _ io.Writer = (*linesWriter)(nil)

type linesWriter struct {
	cb lineWriter
}

type lineWriter func([]byte)

func newlinesWriter(cb lineWriter) *linesWriter {
	return &linesWriter{cb: cb}
}

func (lw *linesWriter) Write(p []byte) (int, error) {
	for i := 0; ; {
		n := bytes.IndexAny(p[i:], "\n\r")
		if n < 0 {
			lw.cb(p[i:])
			break
		}
		lw.cb(p[i : i+n])
		i += n + 1
	}
	return len(p), nil
}
