package runtime

func (rt *runtime) FilterEndpoints(criteria []string) (err error) {
	rt.eIds, err = rt.models[0].FilterEndpoints(criteria)
	return
}
