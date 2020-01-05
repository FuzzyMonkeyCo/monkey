package house

import (
	"io"

	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
)

func (rt *Runtime) InputsCount() int {
	var mdl modeler.Interface
	for _, mdl = range rt.models {
		break
	}

	return mdl.InputsCount()
}

func (rt *Runtime) WriteAbsoluteReferences(w io.Writer) {
	var mdl modeler.Interface
	for _, mdl = range rt.models {
		break
	}

	mdl.WriteAbsoluteReferences(w)
}

func (rt *Runtime) ValidateAgainstSchema(absRef string, data []byte) (err error) {
	var mdl modeler.Interface
	for _, mdl = range rt.models {
		break
	}

	return mdl.ValidateAgainstSchema(absRef, data)
}
