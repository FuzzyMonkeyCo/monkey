package runtime

import (
	"io"

	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
)

// InputsCount sums the amount of named schemas or types APIs define
func (rt *Runtime) InputsCount() int {
	count := 0

	_ = rt.forEachModel(func(name string, mdl modeler.Interface) error {
		count += mdl.InputsCount()
		return nil
	})

	return count
}

// WriteAbsoluteReferences pretty-prints the API's named types
func (rt *Runtime) WriteAbsoluteReferences(w io.Writer) {
	_ = rt.forEachModel(func(name string, mdl modeler.Interface) error {
		mdl.WriteAbsoluteReferences(w)
		return nil
	})
}

// ValidateAgainstSchema tries to smash the data through the given keyhole
func (rt *Runtime) ValidateAgainstSchema(absRef string, data []byte) (err error) {
	count := 0

	err = rt.forEachModel(func(name string, mdl modeler.Interface) error {
		err := mdl.ValidateAgainstSchema(absRef, data)
		// TODO: support >1 models (MAY validate against schema of wrong mdl)
		if _, ok := err.(*modeler.NoSuchRefError); ok {
			count++
			err = nil
		}
		return err
	})

	if count == len(rt.modelsNames) {
		err = modeler.NewNoSuchRefError(absRef)
	}
	return
}
