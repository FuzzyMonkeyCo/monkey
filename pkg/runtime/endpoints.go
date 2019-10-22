package runtime

import (
	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
)

func (rt *runtime) FilterEndpoints(criteria []string) (err error) {
	var mdl modeler.Interface
	for _, mdl = range rt.models {
		break
	}

	rt.eIds, err = mdl.FilterEndpoints(criteria)
	return
}
