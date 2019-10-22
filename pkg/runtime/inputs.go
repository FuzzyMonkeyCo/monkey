package runtime

import (
	"io"

	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
)

func (rt *runtime) InputsCount() int {
	var mdl modeler.Interface
	for _, mdl = range rt.models {
		break
	}

	return mdl.InputsCount()
}

func (rt *runtime) WriteAbsoluteReferences(w io.Writer) {
	var mdl modeler.Interface
	for _, mdl = range rt.models {
		break
	}

	mdl.WriteAbsoluteReferences(w)
}

func (rt *runtime) ValidateAgainstSchema(absRef string, data []byte) (err error) {
	var mdl modeler.Interface
	for _, mdl = range rt.models {
		break
	}

	return mdl.ValidateAgainstSchema(absRef, data)
}
