package reset

import (
	"context"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
)

var _ SUTResetter = (*SUTShell)(nil)

// SUTShell TODO
type SUTShell struct {
	start, reset, stop string
}

// ToProto TODO
func (s *SUTShell) ToProto() *fm.Clt_Msg_Fuzz_Resetter {
	return &fm.Clt_Msg_Fuzz_Resetter{
		Resetter: &fm.Clt_Msg_Fuzz_Resetter_SutShell{&fm.Clt_Msg_Fuzz_Resetter_SUTShell{
			Start: s.start,
			Rst:   s.reset,
			Stop:  s.stop,
		}}}
}

// Start TODO
func (s *SUTShell) Start(ctx context.Context) error { return nil }

// Reset TODO
func (s *SUTShell) Reset(ctx context.Context) error { return nil }

// Stop TODO
func (s *SUTShell) Stop(ctx context.Context) error { return nil }
