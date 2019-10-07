package fuzz

import (
	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
)

// CheckerFunc TODO
type CheckerFunc func(*fm.monkey) (string, []string)

var (
	_ fm.cltDoer = (*msgCall)(nil)
	_ fm.cltDoer = (*msgReset)(nil)
)
