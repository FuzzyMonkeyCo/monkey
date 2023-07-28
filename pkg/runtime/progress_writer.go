package runtime

// fixme: move to reset.go

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

func (pw *progressWriter) Write(p []byte) (int, error) {
	print := func(data []byte) {
		if n := len(data); n > 0 {
			if bytes.HasPrefix(data, []byte("+ ")) {
				return
			}
			if x := bytes.TrimPrefix(data, []byte("++ ")); n != len(x) {
				if string(x) != "set +o xtrace" {
					pw.printf("%s", x)
				}
			}
		}
	}

	for i := 0; ; {
		n := bytes.IndexAny(p[i:], "\n\r")
		if n < 0 {
			print(p[i:])
			break
		}
		print(p[i : i+n])
		i += n + 1
	}
	return len(p), nil
}
