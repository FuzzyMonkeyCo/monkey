package modelers

import (
	"context"

	"github.com/FuzzyMonkeyCo/monkey/pkg/do/fuzz/reset"
	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
)

var _ reset.SUTResetter = (*SUTShell)(nil)

// SUTShell TODO
type SUTShell struct {
	start, reset, stop string
}

func (s *SUTShell) ToProto() fm.isClt_Msg_Fuzz_Resetter_Resetter {
	return &fm.Clt_Msg_Fuzz_Resetter_SutShell{&fm.Clt_Msg_Fuzz_Resetter_SUTShell{
		Start: s.start,
		Rst:   s.reset,
		Stop:  s.stop,
	}}
}

// Start TODO
func (s *SUTShell) Start(ctx context.Context) error { return nil }

// Reset TODO
func (s *SUTShell) Reset(ctx context.Context) error { return nil }

// Stop TODO
func (s *SUTShell) Stop(ctx context.Context) error { return nil }
