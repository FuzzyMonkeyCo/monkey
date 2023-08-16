package modeler

import (
	"context"
	"errors"
	"io"

	"go.starlark.net/starlark"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
)

var (
	ErrUnparsablePayload = errors.New("unparsable piped payload")
	ErrNoSuchSchema      = errors.New("no such schema")
)

// Maker types the New func that instanciates new models
type Maker func(kwargs []starlark.Tuple) (Interface, error)

// Interface describes checkable models.
// A package defining a type that implements Interface also has to define:
// * a non-empty const Name that names the Starlark builtin
// * a func of type Maker named New that instanciates a new model
type Interface interface { // TODO models.Modeler
	// Name uniquely identifies this instance
	Name() string

	// ToProto marshals a modeler.Interface implementation into a *fm.Clt_Fuzz_Model
	ToProto() *fm.Clt_Fuzz_Model

	// Lint goes through specs and unsures they are valid
	Lint(ctx context.Context, showSpec bool) error

	// InputsCount sums the amount of named schemas or types APIs define
	InputsCount() int
	// WriteAbsoluteReferences pretty-prints the API's named types
	WriteAbsoluteReferences(w io.Writer)
	// FilterEndpoints restricts which API endpoints are considered
	FilterEndpoints(criteria []string) ([]uint32, error)

	ValidateAgainstSchema(ref string, data []byte) error
	Validate(uint32, *structpb.Value) []string

	// NewCaller is called before making each call
	NewCaller(ctx context.Context, call *fm.Srv_Call, showf ShowFunc) Caller // FIXME: use Shower iface?
}

// ShowFunc can be used to display informational messages to the tester
type ShowFunc func(string, ...interface{})
