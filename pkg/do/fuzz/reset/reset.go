package reset

import (
	"context"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
)

// SUTResetter describes ways to reset the system under test to a known initial state
type SUTResetter interface {
	ToProto() fm.Clt_Msg_Fuzz_Resetter

	Start(context.Context) error
	Reset(context.Context) error
	Stop(context.Context) error
}
