package shell

import (
	"context"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"go.starlark.net/starlark"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/progresser"
	"github.com/FuzzyMonkeyCo/monkey/pkg/resetter"
	"github.com/FuzzyMonkeyCo/monkey/pkg/tags"
)

// Name names the Starlark builtin
const Name = "shell"

// TODO:{start,reset,strop}_file a la Bazel

const (
	shell = "/bin/bash" // TODO: use mentioned shell

	scriptTimeout = 2 * time.Minute // TODO: tune through kwargs
)

// New instanciates a new resetter
func New(kwargs []starlark.Tuple) (resetter.Interface, error) {
	var lot struct {
		name, start, reset, stop starlark.String
		provides                 tags.UniqueStringsNonEmpty
	}
	if err := starlark.UnpackArgs(Name, nil, kwargs,
		"name", &lot.name,
		"provides", &lot.provides,
		// NOTE: all args following an optional? are implicitly optional.
		"start??", &lot.start,
		"reset??", &lot.reset,
		"stop??", &lot.stop,
		// TODO: waiton = "tcp/4000", various recipes => 1 rsttr per service
		//TODO: tags
	); err != nil {
		log.Println("[ERR]", err)
		return nil, err
	}
	log.Printf("[DBG] unpacked %+v", lot)

	// verify each

	name := lot.name.GoString()
	if err := tags.LegalName(name); err != nil { //TODO: newUserError
		log.Println("[ERR]", err)
		return nil, err
	}

	// verify all

	// assemble

	s := &Resetter{
		name:     name,
		provides: lot.provides.GoStrings(),
	}
	s.Start = strings.TrimSpace(lot.start.GoString())
	s.Rst = strings.TrimSpace(lot.reset.GoString())
	s.Stop = strings.TrimSpace(lot.stop.GoString())
	return s, nil
}

var _ resetter.Interface = (*Resetter)(nil)

// Resetter implements resetter.Interface
type Resetter struct {
	name     string
	provides []string
	fm.Clt_Fuzz_Resetter_Shell

	isNotFirstRun bool

	scriptsCreator sync.Once
	scriptsPaths   map[shellCmd]string
	stdin          io.WriteCloser
	sherr          chan error
	rcoms          chan uint8
}

// Name uniquely identifies this instance
func (s *Resetter) Name() string { return s.name }

// Provides lists the models a resetter resets
func (s *Resetter) Provides() []string { return s.provides }

// ToProto marshals a resetter.Interface implementation into a *fm.Clt_Fuzz_Resetter
func (s *Resetter) ToProto() *fm.Clt_Fuzz_Resetter {
	return &fm.Clt_Fuzz_Resetter{
		Name:     s.name,
		Provides: s.provides,
		Resetter: &fm.Clt_Fuzz_Resetter_Shell_{
			Shell: &s.Clt_Fuzz_Resetter_Shell,
		}}
}

// ExecStart executes the setup phase of the System Under Test
func (s *Resetter) ExecStart(ctx context.Context, shower progresser.Shower, only bool, envRead map[string]string) error {
	return s.exec(ctx, shower, envRead, cmdStart)
}

// ExecReset resets the System Under Test to a state similar to a post-ExecStart state
func (s *Resetter) ExecReset(ctx context.Context, shower progresser.Shower, only bool, envRead map[string]string) error {
	if only {
		// Makes `monkey exec reset` run as if in between tests
		s.isNotFirstRun = true
	}

	cmds, err := s.commands()
	if err != nil {
		return err
	}

	if !s.isNotFirstRun {
		s.isNotFirstRun = true
	}

	return s.exec(ctx, shower, envRead, cmds...)
}

// ExecStop executes the cleanup phase of the System Under Test
func (s *Resetter) ExecStop(ctx context.Context, shower progresser.Shower, only bool, envRead map[string]string) error {
	return s.exec(ctx, shower, envRead, cmdStop)
}

// Terminate cleans up after a resetter.Interface implementation instance
func (s *Resetter) Terminate(ctx context.Context, shower progresser.Shower, envRead map[string]string) (err error) {
	if hasStop := s.Stop != ""; hasStop {
		if err = s.ExecStop(ctx, shower, true, envRead); err != nil {
			log.Println("[ERR]", err)
			return
		}
	}
	log.Println("[NFO] exiting shell singleton")
	s.signal(comExit, "")
	return
}
