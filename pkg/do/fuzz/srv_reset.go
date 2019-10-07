package fuzz

import (
	"github.com/FuzzyMonkeyCo/monkey/pkg"
	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
)

type msgReset struct{ fm.Srv_Msg_Reset_ }

func (msg *msgReset) do(mnk *pkg.monkey) (err error) { return }
