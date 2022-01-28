package openapiv3

import (
	"fmt"
	"io"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
	"github.com/gogo/protobuf/types"
	"go.starlark.net/starlark"
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
	m := &OA3{name: name.GoString()}
	m.File = file.GoString()
	m.Host = host.GoString()
	m.HeaderAuthorization = headerAuthorization.GoString()
	return m, nil
}

var _ modeler.Interface = (*OA3)(nil) // TODO: *oa3

// OA3 implements a modeler.Interface for use by `monkey`.
type OA3 struct {
	name string
	fm.Clt_Fuzz_Model_OpenAPIv3

	vald *validator

	tcap *tCapHTTP
}

// Name uniquely identifies this instance
func (m *OA3) Name() string { return m.name }

// ToProto marshals a modeler.Interface implementation into a *fm.Clt_Fuzz_Model
func (m *OA3) ToProto() *fm.Clt_Fuzz_Model {
	m.Spec = m.vald.Spec
	return &fm.Clt_Fuzz_Model{
		Name: m.name,
		Model: &fm.Clt_Fuzz_Model_Openapiv3{
			Openapiv3: &m.Clt_Fuzz_Model_OpenAPIv3,
		},
	}
}

////////////// FromProto unmarshals a modeler.Interface implementation into a *fm.Clt_Fuzz_Model
func (m *OA3) fromProto(p *fm.Clt_Fuzz_Model) error {
	if mm := p.GetOpenapiv3(); mm != nil {
		m.Clt_Fuzz_Model_OpenAPIv3 = *mm
		m.vald = &validator{Spec: mm.Spec}
		return nil
	}
	return fmt.Errorf("unexpected model type: %T", p.GetModel())
}

// InputsCount sums the amount of named schemas or types APIs define
func (m *OA3) InputsCount() int {
	return m.vald.inputsCount()
}

// FilterEndpoints restricts which API endpoints are considered
func (m *OA3) FilterEndpoints(args []string) ([]eid, error) {
	return m.vald.filterEndpoints(args)
}

func (m *OA3) Validate(SID sid, data *types.Value) []string {
	return m.vald.Validate(SID, data)
}

// ValidateAgainstSchema tries to smash the data through the given keyhole
func (m *OA3) ValidateAgainstSchema(absRef string, data []byte) error {
	return m.vald.validateAgainstSchema(absRef, data)
}

// WriteAbsoluteReferences pretty-prints the API's named types
func (m *OA3) WriteAbsoluteReferences(w io.Writer) {
	m.vald.writeAbsoluteReferences(w)
}
