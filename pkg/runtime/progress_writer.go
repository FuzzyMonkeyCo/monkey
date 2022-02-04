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

func (pw *progressWriter) Write(p []byte) (int, error) {
	print := func(data []byte) {
		if len(data) > 0 {
			pw.printf("%s", data)
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
