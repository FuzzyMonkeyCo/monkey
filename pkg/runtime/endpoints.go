package runtime

import (
	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
)

// FilterEndpoints restricts which API endpoints are considered
func (rt *Runtime) FilterEndpoints(criteria []string) error {
	return rt.forEachModel(func(name string, mdl modeler.Interface) (err error) {
		var eids []uint32
		if eids, err = mdl.FilterEndpoints(criteria); err != nil || len(eids) == 0 {
			return
		}

		if rt.selectedEIDs == nil {
			rt.selectedEIDs = make(map[string]*fm.Uint32S)
		}
		rt.selectedEIDs[name] = &fm.Uint32S{Values: eids}
		return
	})
}
