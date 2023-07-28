package runtime

import (
	"context"
	"errors"
	"sort"

	"go.starlark.net/starlark"
	"golang.org/x/sync/errgroup"

	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
	"github.com/FuzzyMonkeyCo/monkey/pkg/resetter"
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

func (rt *Runtime) forEachSelectedResetter(ctx context.Context, f func(string, resetter.Interface) error) error {
	if rt.selectedResetters == nil {
		rt.selectedResetters = make(map[string]struct{}, len(rt.resetters))
		for name, rsttr := range rt.resetters {
			for _, modelName := range rsttr.Provides() {
				if _, ok := rt.selectedEIDs[modelName]; ok {
					rt.selectedResetters[name] = struct{}{}
					break
				}
			}
		}
	}
	if len(rt.selectedResetters) == 0 {
		return errors.New("no resetter selected")
	}

	g, _ := errgroup.WithContext(ctx)
	_ = rt.forEachResetter(func(name string, rsttr resetter.Interface) error {
		if _, ok := rt.selectedResetters[name]; ok {
			g.Go(func() error {
				return f(name, rsttr)
			})
		}
		return nil
	})
	return g.Wait()
}

func (rt *Runtime) forEachGlobal(f func(name string, value starlark.Value) error) error {
	names := make([]string, 0, len(rt.globals))
	for name := range rt.globals {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		name, value := name, rt.globals[name]
		if err := f(name, value); err != nil {
			return err
		}
	}
	return nil
}
