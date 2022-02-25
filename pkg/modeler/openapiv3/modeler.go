package openapiv3

import (
	"io"

	"go.starlark.net/starlark"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
)

// Name names the Starlark builtin
const Name = "openapi3"

// New instanciates a new model
func New(kwargs []starlark.Tuple) (modeler.Interface, error) {
	var name, file, host, headerAuthorization starlark.String
	if err := starlark.UnpackArgs(Name, nil, kwargs,
		"name", &name,
		"file", &file,
		"host??", &host,
		"header_authorization??", &headerAuthorization,
	); err != nil {
		return nil, err
	}
	m := &oa3{name: name.GoString()}
	m.pb = &fm.Clt_Fuzz_Model_OpenAPIv3{
		File:                file.GoString(),
		Host:                host.GoString(),
		HeaderAuthorization: headerAuthorization.GoString(),
	}
	return m, nil
}

var _ modeler.Interface = (*oa3)(nil)

// oa3 implements a modeler.Interface for use by `monkey`.
type oa3 struct {
	name string

	pb *fm.Clt_Fuzz_Model_OpenAPIv3

	vald *validator

	tcap *tCapHTTP
}

// Name uniquely identifies this instance
func (m *oa3) Name() string { return m.name }

// ToProto marshals a modeler.Interface implementation into a *fm.Clt_Fuzz_Model
func (m *oa3) ToProto() *fm.Clt_Fuzz_Model {
	m.pb.Spec = m.vald.Spec
	return &fm.Clt_Fuzz_Model{
		Name:  m.name,
		Model: &fm.Clt_Fuzz_Model_Openapiv3{Openapiv3: m.pb},
	}
}

// InputsCount sums the amount of named schemas or types APIs define
func (m *oa3) InputsCount() int {
	return m.vald.inputsCount()
}

// FilterEndpoints restricts which API endpoints are considered
func (m *oa3) FilterEndpoints(args []string) ([]eid, error) {
	return m.vald.filterEndpoints(args)
}

func (m *oa3) Validate(SID sid, data *structpb.Value) []string {
	return m.vald.Validate(SID, data)
}

// ValidateAgainstSchema tries to smash the data through the given keyhole
func (m *oa3) ValidateAgainstSchema(absRef string, data []byte) error {
	return m.vald.validateAgainstSchema(absRef, data)
}

// WriteAbsoluteReferences pretty-prints the API's named types
func (m *oa3) WriteAbsoluteReferences(w io.Writer) {
	m.vald.writeAbsoluteReferences(w)
}
