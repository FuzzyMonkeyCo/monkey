package modeler

import (
	"io"

	"github.com/FuzzyMonkeyCo/monkey/pkg/do/fuzz/reset"
	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
)

// Modeler describes checkable models
type Modeler interface {
	ToProto() *fm.Clt_Msg_Fuzz_Model

	SetSUTResetter(reset.SUTResetter)
	GetSUTResetter() reset.SUTResetter

	Pretty(w io.Writer) (n int, err error)
}
