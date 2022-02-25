package runtime

import (
	"fmt"
)

type userError string

var _ error = userError("")

func newUserError(f string, a ...interface{}) userError { return userError(fmt.Sprintf(f, a...)) }
func (e userError) Error() string                       { return string(e) }
