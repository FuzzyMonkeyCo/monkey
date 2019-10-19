package runtime

import (
	"io"
)

func (rt *runtime) InputsCount() int {
	return rt.models[0].InputsCount()
}

func (rt *runtime) WriteAbsoluteReferences(w io.Writer) {
	rt.models[0].WriteAbsoluteReferences(w)
}

func (rt *runtime) ValidateAgainstSchema(absRef string, data []byte) (err error) {
	return rt.models[0].ValidateAgainstSchema(absRef, data)
}
