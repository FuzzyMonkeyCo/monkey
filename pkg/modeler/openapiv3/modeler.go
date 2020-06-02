package openapiv3

import (
	"fmt"
	"io"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
	"github.com/FuzzyMonkeyCo/monkey/pkg/resetter"
	"github.com/gogo/protobuf/types"
	"go.starlark.net/starlark"
)

var _ modeler.Interface = (*oa3)(nil)

type T = oa3

type oa3 struct {
	fm.Clt_Fuzz_Model_OpenAPIv3

	resetter resetter.Interface

	vald *validator

	tcap *tCapHTTP
}

// ToProto TODO
func (m *oa3) ToProto() *fm.Clt_Fuzz_Model {
	m.Spec = m.vald.Spec
	return &fm.Clt_Fuzz_Model{
		Model: &fm.Clt_Fuzz_Model_Openapiv3{
			Openapiv3: &m.Clt_Fuzz_Model_OpenAPIv3,
		},
	}
}

// FromProto TODO
func (m *oa3) FromProto(p *fm.Clt_Fuzz_Model) error {
	if mm := p.GetOpenapiv3(); mm != nil {
		m.Clt_Fuzz_Model_OpenAPIv3 = *mm
		m.vald = &validator{Spec: mm.Spec} // TODO? merge vald with oa3
		return nil
	}
	return fmt.Errorf("unexpected model type: %T", p.GetModel())
}

// SetResetter TODO
func (m *oa3) SetResetter(sr resetter.Interface) { m.resetter = sr }

// GetResetter TODO
func (m *oa3) GetResetter() resetter.Interface { return m.resetter }

func (m *oa3) NewFromKwargs(d starlark.StringDict) (modeler.Interface, *modeler.Error) {
	m = &oa3{}
	var err *modeler.Error

	if m.File, err = slGetString(d, "file"); err != nil {
		return nil, err
	}
	if m.Host, err = slGetString(d, "host"); err != nil {
		return nil, err
	}
	if m.HeaderAuthorization, err = slGetString(d, "header_authorization"); err != nil {
		return nil, err
	}

	return m, nil
}

func slGetString(d starlark.StringDict, field string) (str string, err *modeler.Error) {
	var (
		found bool
		val   starlark.Value
	)
	if val, found = d[field]; found && val.Type() != "string" {
		err = modeler.NewError(field, "a string", val.Type())
		return
	}
	if found {
		str = val.(starlark.String).GoString()
	}
	return
}

func (m *oa3) InputsCount() int {
	return m.vald.InputsCount()
}

func (m *oa3) FilterEndpoints(args []string) ([]eid, error) {
	return m.vald.FilterEndpoints(args)
}

func (m *oa3) Validate(SID sid, data *types.Value) []string {
	return m.vald.Validate(SID, data)
}

func (m *oa3) ValidateAgainstSchema(absRef string, data []byte) error {
	return m.vald.ValidateAgainstSchema(absRef, data)
}

func (m *oa3) WriteAbsoluteReferences(w io.Writer) {
	m.vald.WriteAbsoluteReferences(w)
}
