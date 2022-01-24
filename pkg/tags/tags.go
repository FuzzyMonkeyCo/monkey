package tags

import (
	"errors"
	"fmt"
	"strings"
)

// Alphabet lists the only characters allowed in LegalName
const Alphabet = "abcdefghijklmnopqrstuvwxyz_1234567890"

// Tags represent a set of legal (per LegalName) tags.
type Tags = map[string]struct{}

// Filter is used to activate or deactivate a monkey.check during Fuzz.
type Filter struct {
	excludeAll, includeAll bool
	include, exclude       Tags
}

// NewFilter attempts to build a tags filter from CLI-ish arguments.
func NewFilter(includeSetButZero, excludeSetButZero bool, i, o []string) (r *Filter, err error) {
	if includeSetButZero && excludeSetButZero ||
		len(i) != 0 && len(o) != 0 ||
		includeSetButZero && len(o) != 0 ||
		excludeSetButZero && len(i) != 0 {
		err = errors.New("filtering tags with both inclusion and exclusion lists is unsupported")
		return
	}
	f := &Filter{excludeAll: includeSetButZero, includeAll: excludeSetButZero}
	if f.include, err = fromSlice(i); err != nil {
		return
	}
	if f.exclude, err = fromSlice(o); err != nil {
		return
	}
	r = f
	return
}

// Excludes applies the filter to a monkey.check's tags.
func (f *Filter) Excludes(checking Tags) bool {
	if f.includeAll {
		return false
	}
	if f.excludeAll {
		return true
	}
	for tag := range checking {
		if _, ok := f.include[tag]; ok {
			return !ok
		}
		if _, ok := f.exclude[tag]; ok {
			return ok
		}
	}
	return false
}

func fromSlice(xs []string) (r Tags, err error) {
	m := make(Tags, len(xs))
	for _, x := range xs {
		if err = LegalName(x); err != nil {
			return
		}
		if _, ok := m[x]; ok {
			err = fmt.Errorf("tag %q appears more than once in filter list", x)
			return
		}
		m[x] = struct{}{}
	}
	r = m
	return
}

// LegalName fails when string isn't the right format.
func LegalName(name string) error {
	if len(name) == 0 {
		return errors.New("string is empty")
	}
	if len(name) > 255 {
		return fmt.Errorf("string is too long: %q", name)
	}
	for _, c := range name {
		if !strings.ContainsRune(Alphabet, c) {
			return fmt.Errorf("string should only contain characters from %s: %q", Alphabet, name)
		}
	}
	return nil
}
