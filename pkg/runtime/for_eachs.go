package runtime

import (
	"context"

	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
	"github.com/FuzzyMonkeyCo/monkey/pkg/resetter"
	"golang.org/x/sync/errgroup"
)

func (rt *Runtime) forEachCheck(f func(name string, chk *check) error) error {
	for _, name := range rt.checksNames {
		name, chk := name, rt.checks[name]
		if err := f(name, chk); err != nil {
			return err
		}
	}
	return nil
}

func (rt *Runtime) forEachModel(f func(name string, mdl modeler.Interface) error) error {
	for _, name := range rt.modelsNames {
		name, mdl := name, rt.models[name]
		if err := f(name, mdl); err != nil {
			return err
		}
	}
	return nil
}

func (rt *Runtime) forEachSelectedModel(f func(string, modeler.Interface) error) error {
	return rt.forEachModel(func(name string, mdl modeler.Interface) error {
		if _, ok := rt.selectedEIDs[name]; ok {
			return f(name, mdl)
		}
		return nil
	})
}

func (rt *Runtime) forEachResetter(f func(name string, rsttr resetter.Interface) error) error {
	for _, name := range rt.resettersNames {
		name, rsttr := name, rt.resetters[name]
		if err := f(name, rsttr); err != nil {
			return err
		}
	}
	return nil
}

var selectedResetters map[string]struct{}

func (rt *Runtime) forEachSelectedResetter(ctx context.Context, f func(string, resetter.Interface) error) error {
	if selectedResetters == nil {
		selectedResetters = make(map[string]struct{}, len(rt.resetters))
		for name, rsttr := range rt.resetters {
			for _, modelName := range rsttr.Provides() {
				if _, ok := rt.selectedEIDs[modelName]; ok {
					selectedResetters[name] = struct{}{}
					break
				}
			}
		}
	}

	g, _ := errgroup.WithContext(ctx)
	_ = rt.forEachResetter(func(name string, rsttr resetter.Interface) error {
		if _, ok := selectedResetters[name]; ok {
			g.Go(func() error {
				return f(name, rsttr)
			})
		}
		return nil
	})
	return g.Wait()
}