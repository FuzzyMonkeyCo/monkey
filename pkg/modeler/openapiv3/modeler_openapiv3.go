package modeler_openapiv3

import (
	"github.com/FuzzyMonkeyCo/monkey/pkg/resetter"
	"github.com/FuzzyMonkeyCo/monkey/pkg/runtime"
)

func init() {
	runtime.RegisterModeler("OpenAPIv3", modelerOpenAPIv3)
}

var _ modeler.Modeler = (*ModelOpenAPIv3)(nil)

// ModelOpenAPIv3 describes OpenAPIv3 models
type ModelOpenAPIv3 struct {
	resetter resetter.Resetter

	/// Fields editable on initial run
	// File is a path within current directory pointing to a YAML spec
	File string
	// Host superseeds the spec's base URL
	Host string
	// HeaderAuthorization if non-empty is added to requests as bearer token
	HeaderAuthorization string

	// FIXME? tcap *tCapHTTP
}

// ToProto TODO
func (m *ModelOpenAPIv3) ToProto() *fm.Clt_Msg_Fuzz_Model {
	return &Clt_Msg_Fuzz_Model_Openapiv3{&Clt_Msg_Fuzz_Model_OpenAPIv3{
		File:                m.File,
		Host:                m.Host,
		HeaderAuthorization: m.HeaderAuthorization,
	}}
}

// SetResetter TODO
func (m *ModelOpenAPIv3) SetResetter(sr resetter.Resetter) { m.resetter = sr }

// GetResetter TODO
func (m *ModelOpenAPIv3) GetResetter() resetter.Resetter { return m.resetter }

// Pretty TODO
func (m *ModelOpenAPIv3) Pretty(w io.Writer) (int, error) { return fmt.Fprintf(w, "%+v\n", m) }

func modelerOpenAPIv3(d starlark.StringDict) (modeler.Modeler, *modeler.Error) {
	mo := &ModelOpenAPIv3{}
	var (
		found              bool
		field              string
		file, host, hAuthz starlark.Value
	)

	field = "file"
	if file, found = d[field]; !found || file.Type() != "string" {
		e := &modeler.Error{FieldRead: field, Want: "a string", Got: file.Type()}
		return nil, e
	}
	mo.File = file.(starlark.String).GoString()

	field = "host"
	if host, found = d[field]; found && host.Type() != "string" {
		e := &modeler.Error{FieldRead: field, Want: "a string", Got: host.Type()}
		return nil, e
	}
	if found {
		h := host.(starlark.String).GoString()
		mo.Host = h
		addHost = &h
	}

	field = "header_authorization"
	if hAuthz, found = d[field]; found && hAuthz.Type() != "string" {
		e := &modeler.Error{FieldRead: field, Want: "a string", Got: hAuthz.Type()}
		return nil, e
	}
	if found {
		authz := hAuthz.(starlark.String).GoString()
		mo.HeaderAuthorization = authz
		addHeaderAuthorization = &authz
	}

	return mo, nil
}
