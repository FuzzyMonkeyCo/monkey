package fuzz

import (
	"github.com/FuzzyMonkeyCo/monkey/pkg"
	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
)

// CheckerFunc TODO
type CheckerFunc func(*pkg.monkey) (string, []string)

var (
	_ fm.cltDoer = (*msgCall)(nil)
	_ fm.cltDoer = (*msgReset)(nil)
)
