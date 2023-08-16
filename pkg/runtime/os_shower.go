package runtime

import (
	"fmt"
	"os"

	"github.com/FuzzyMonkeyCo/monkey/pkg/progresser"
)

var _ progresser.Shower = &osShower{}

type osShower struct{}

// Printf formats informational data
func (p *osShower) Printf(format string, s ...interface{}) {
	_, _ = fmt.Fprintf(os.Stdout, format+"\n", s...)
}

// Errorf formats error messages
func (p *osShower) Errorf(format string, s ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", s...)
}
