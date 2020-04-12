package starlarktruth

import (
	"fmt"
	"io"
)

var _ io.Closer = (*T)(nil)

func (t *T) Close() (err error) {
	if !t.closed {
		err = fmt.Errorf("well %+v", t)
	}
	return
}
