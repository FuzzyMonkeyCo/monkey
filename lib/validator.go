package lib

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"sort"

	"github.com/golang/protobuf/proto"
	"github.com/xeipuuv/gojsonschema"
)

var ErrInvalidPayload = errors.New("invalid JSON payload")
var ErrNoSuchRef = errors.New("no such $ref")

type sid = uint32
type schemaJSON = map[string]interface{}
type schemasJSON = map[string]schemaJSON

type Validator struct {
	Spec *SpecIR
	Refs map[string]sid
	Refd *gojsonschema.SchemaLoader
	Anon map[sid]schemaJSON
}

func newValidator(capaEndpoints, capaSchemas int) *Validator {
	return &Validator{
		Refs: make(map[string]sid, capaSchemas),
		Anon: make(map[sid]schemaJSON, capaEndpoints),
		Spec: &SpecIR{
			Endpoints: make([]*Endpoint, 0, capaEndpoints),
			Schemas:   &Schemas{Json: make(map[sid]*RefOrSchemaJSON, capaSchemas)},
		},
		Refd: gojsonschema.NewSchemaLoader(),
	}
}

func (vald *Validator) newSID() sid {
	return sid(1 + len(vald.Spec.Schemas.Json))
}

func (vald *Validator) seed(base string, schemas schemasJSON) (err error) {
	i, names := 0, make([]string, len(schemas))
	for name := range schemas {
		names[i] = name
		i++
	}
	sort.Strings(names)

	for j := 0; j != i; j++ {
		name := names[j]
		absRef := base + name
		log.Printf("[DBG] pre-seeding ref '%s'", absRef)
		refSID := vald.newSID()
		vald.Spec.Schemas.Json[refSID] = &RefOrSchemaJSON{
			PtrOrSchema: &RefOrSchemaJSON_Ptr{&SchemaPtr{Ref: absRef, SID: 0}},
		}
		vald.Refs[absRef] = refSID
	}

	for j := 0; j != i; j++ {
		name := names[j]
		absRef := base + name
		schema := schemas[name]
		log.Printf("[DBG] seeding schema '%s'", absRef)

		sl := gojsonschema.NewGoLoader(schema)
		if err = vald.Refd.AddSchema(absRef, sl); err != nil {
			log.Println("[ERR]", err)
			return
		}

		sid := vald.ensureMapped("", schema)
		if sid == 0 {
			// Impossible
			panic(absRef)
		}
		refSID := vald.Refs[absRef]
		vald.Refs[absRef] = sid
		vald.Spec.Schemas.Json[refSID] = &RefOrSchemaJSON{
			PtrOrSchema: &RefOrSchemaJSON_Ptr{&SchemaPtr{Ref: absRef, SID: sid}},
		}
	}
	return
}

func (vald *Validator) ensureMapped(ref string, goSchema schemaJSON) sid {
	if ref == "" {
		schema := vald.fromGo(goSchema)
		for SID, schemaPtr := range vald.Spec.Schemas.Json {
			if s := schemaPtr.GetSchema(); s != nil && proto.Equal(&schema, s) {
				return SID
			}
		}
		SID := vald.newSID()
		vald.Spec.Schemas.Json[SID] = &RefOrSchemaJSON{
			PtrOrSchema: &RefOrSchemaJSON_Schema{&schema},
		}
		return SID
	}

	mappedSID, ok := vald.Refs[ref]
	if !ok {
		// Every $ref should already be in here
		panic(ref)
	}
	schemaPtr := &SchemaPtr{Ref: ref, SID: mappedSID}
	SID := sid(0)
	for refSID, schemaPtr := range vald.Spec.Schemas.Json {
		if ptr := schemaPtr.GetPtr(); ptr != nil && ptr.GetRef() == ref {
			SID = refSID
		}
	}
	if SID == 0 {
		// Impossible not to find that ref
		panic(ref)
	}
	vald.Spec.Schemas.Json[SID] = &RefOrSchemaJSON{
		PtrOrSchema: &RefOrSchemaJSON_Ptr{schemaPtr},
	}
	return SID
}

func (vald *Validator) fromGo(s schemaJSON) (schema Schema_JSON) {
	// "enum"
	if v, ok := s["enum"]; ok {
		enum := v.([]interface{})
		schema.Enum = make([]*ValueJSON, len(enum))
		for i, vv := range enum {
			schema.Enum[i] = enumFromGo(vv)
		}
	}

	// "type"
	if v, ok := s["type"]; ok {
		types := v.([]string)
		schema.Types = make([]Schema_JSON_Type, len(types))
		for i, vv := range types {
			schema.Types[i] = Schema_JSON_Type(Schema_JSON_Type_value[vv])
		}
	}

	// "format"
	if v, ok := s["format"]; ok {
		schema.Format = formatFromGo(v.(string))
	}
	// "minLength"
	if v, ok := s["minLength"]; ok {
		schema.MinLength = v.(uint64)
	}
	// "maxLength"
	if v, ok := s["maxLength"]; ok {
		schema.MaxLength = v.(uint64)
		schema.HasMaxLength = true
	}
	// "pattern"
	if v, ok := s["pattern"]; ok {
		schema.Pattern = v.(string)
	}

	// "minimum"
	if v, ok := s["minimum"]; ok {
		schema.Minimum = v.(float64)
		schema.HasMinimum = true
	}
	// "maximum"
	if v, ok := s["maximum"]; ok {
		schema.Maximum = v.(float64)
		schema.HasMaximum = true
	}
	// "exclusiveMinimum"
	if v, ok := s["exclusiveMinimum"]; ok {
		schema.ExclusiveMinimum = v.(bool)
	}
	// "exclusiveMaximum"
	if v, ok := s["exclusiveMaximum"]; ok {
		schema.ExclusiveMaximum = v.(bool)
	}
	// "multipleOf"
	if v, ok := s["multipleOf"]; ok {
		schema.TranslatedMultipleOf = v.(float64) - 1.0
	}

	// "uniqueItems"
	if v, ok := s["uniqueItems"]; ok {
		schema.UniqueItems = v.(bool)
	}
	// "minItems"
	if v, ok := s["minItems"]; ok {
		schema.MinItems = v.(uint64)
	}
	// "maxItems"
	if v, ok := s["maxItems"]; ok {
		schema.MaxItems = v.(uint64)
		schema.HasMaxItems = true
	}
	// "items"
	if v, ok := s["items"]; ok {
		items := v.([]schemaJSON)
		schema.Items = make([]sid, len(items))
		for i, ss := range items {
			var ref string
			if v, ok := ss["$ref"]; ok {
				ref = v.(string)
			}
			schema.Items[i] = vald.ensureMapped(ref, ss)
		}
	}

	// "minProperties"
	if v, ok := s["minProperties"]; ok {
		schema.MinProperties = v.(uint64)
	}
	// "maxProperties"
	if v, ok := s["maxProperties"]; ok {
		schema.MaxProperties = v.(uint64)
		schema.HasMaxProperties = true
	}
	// "required"
	if v, ok := s["required"]; ok {
		schema.Required = v.([]string)
	}
	// "properties"
	if v, ok := s["properties"]; ok {
		properties := v.(schemasJSON)
		if count := len(properties); count != 0 {
			schema.Properties = make(map[string]sid, count)
			i, props := 0, make([]string, count)
			for propName := range properties {
				props[i] = propName
				i++
			}
			sort.Strings(props)

			for j := 0; j != i; j++ {
				propName := props[j]
				ss := properties[propName]
				var ref string
				if v, ok := ss["$ref"]; ok {
					ref = v.(string)
				}
				schema.Properties[propName] = vald.ensureMapped(ref, ss)
			}
		}
	}
	//FIXME: "additionalProperties"

	// "allOf"
	if v, ok := s["allOf"]; ok {
		of := v.([]schemaJSON)
		schema.AllOf = make([]sid, len(of))
		for i, ss := range of {
			var ref string
			if v, ok := ss["$ref"]; ok {
				ref = v.(string)
			}
			schema.AllOf[i] = vald.ensureMapped(ref, ss)
		}
	}

	// "anyOf"
	if v, ok := s["anyOf"]; ok {
		of := v.([]schemaJSON)
		schema.AnyOf = make([]sid, len(of))
		for i, ss := range of {
			var ref string
			if v, ok := ss["$ref"]; ok {
				ref = v.(string)
			}
			schema.AnyOf[i] = vald.ensureMapped(ref, ss)
		}
	}

	// "oneOf"
	if v, ok := s["oneOf"]; ok {
		of := v.([]schemaJSON)
		schema.OneOf = make([]sid, len(of))
		for i, ss := range of {
			var ref string
			if v, ok := ss["$ref"]; ok {
				ref = v.(string)
			}
			schema.OneOf[i] = vald.ensureMapped(ref, ss)
		}
	}

	// "not"
	if v, ok := s["not"]; ok {
		ss := v.(schemaJSON)
		var ref string
		if v, ok := ss["$ref"]; ok {
			ref = v.(string)
		}
		schema.Not = vald.ensureMapped(ref, ss)
	}

	return
}

func formatFromGo(format string) Schema_JSON_Format {
	switch format {
	case "date-time":
		return Schema_JSON_date_time
	case "uriref", "uri-reference":
		return Schema_JSON_uri_reference
	default:
		v, ok := Schema_JSON_Format_value[format]
		if ok {
			return Schema_JSON_Format(v)
		}
		return Schema_JSON_NONE
	}
}

func enumFromGo(value interface{}) *ValueJSON {
	if value == nil {
		return &ValueJSON{Value: &ValueJSON_IsNull{true}}
	}
	switch value.(type) {
	case bool:
		return &ValueJSON{Value: &ValueJSON_Boolean{value.(bool)}}
	case float64:
		return &ValueJSON{Value: &ValueJSON_Number{value.(float64)}}
	case string:
		return &ValueJSON{Value: &ValueJSON_Text{value.(string)}}
	case []interface{}:
		val := value.([]interface{})
		vs := make([]*ValueJSON, len(val))
		for i, v := range val {
			vs[i] = enumFromGo(v)
		}
		return &ValueJSON{Value: &ValueJSON_Array{&ArrayJSON{Values: vs}}}
	case map[string]interface{}:
		val := value.(map[string]interface{})
		vs := make(map[string]*ValueJSON, len(val))
		for n, v := range val {
			vs[n] = enumFromGo(v)
		}
		return &ValueJSON{Value: &ValueJSON_Object{&ObjectJSON{Values: vs}}}
	default:
		panic("unreachable")
	}
}

func (vald *Validator) ValidateAgainstSchema(absRef string) (err error) {
	if _, ok := vald.Refs[absRef]; !ok {
		err = ErrNoSuchRef
		return
	}

	var value interface{}
	if err = json.NewDecoder(os.Stdin).Decode(&value); err != nil {
		log.Println("[ERR]", err)
		return
	}

	// TODO: Compile errs on bad refs only, MUST do this step in `lint`
	log.Println("[NFO] compiling schema refs")
	schema, err := vald.Refd.Compile(
		gojsonschema.NewGoLoader(schemaJSON{"$ref": absRef}))
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	log.Println("[NFO] validating payload against refs")
	res, err := schema.Validate(gojsonschema.NewGoLoader(value))
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	errs := res.Errors()
	for _, e := range errs {
		// ResultError interface
		ColorERR.Println(e)
	}
	if len(errs) > 0 {
		err = ErrInvalidPayload
	}
	return
}
