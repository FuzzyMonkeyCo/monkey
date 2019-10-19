package runtime

// Lint TODO
func (rt *runtime) Lint(showSpec bool) error {
	return rt.models[0].Lint(showSpec)
}
