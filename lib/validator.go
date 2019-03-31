package lib

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/xeipuuv/gojsonschema"
)

var ErrInvalidPayload = errors.New("invalid JSON payload")
var ErrNoSuchRef = errors.New("no such $ref")

type eid = uint32
type sid = uint32
type schemaJSON = map[string]interface{}
type schemasJSON = map[string]schemaJSON

type Validator struct {
	Spec *SpecIR
	Refs map[string]sid
	Refd *gojsonschema.SchemaLoader
}

func newValidator(capaEndpoints, capaSchemas int) *Validator {
	return &Validator{
		Refs: make(map[string]sid, capaSchemas),
		Spec: &SpecIR{
			Endpoints: make(map[eid]*Endpoint, capaEndpoints),
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
			if s := schemaPtr.GetSchema(); s != nil && schema.Equal(s) {
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

type schemap map[sid]*RefOrSchemaJSON

func (sm schemap) toGo(SID sid) (s schemaJSON) {
	schemaOrRef, ok := sm[SID]
	if !ok {
		log.Fatalf("unknown SID %d", SID)
	}
	if sp := schemaOrRef.GetPtr(); sp != nil {
		return schemaJSON{"$ref": sp.GetRef()}
	}
	schema := schemaOrRef.GetSchema()
	s = make(schemaJSON)

	// "enum"
	if schemaEnum := schema.GetEnum(); len(schemaEnum) != 0 {
		enum := make([]interface{}, len(schemaEnum))
		for i, v := range schemaEnum {
			enum[i] = EnumToGo(v)
		}
		s["enum"] = enum
	}

	// "type"
	if schemaTypes := schema.GetTypes(); len(schemaTypes) != 0 {
		types := make([]string, len(schemaTypes))
		for i, v := range schemaTypes {
			types[i] = v.String()
		}
		s["type"] = types
	}

	// "format"
	if schemaFormat := schema.GetFormat(); schemaFormat != Schema_JSON_NONE {
		s["format"] = formatToGo(schemaFormat)
	}
	// "minLength"
	if schemaMinLength := schema.GetMinLength(); schemaMinLength != 0 {
		s["minLength"] = schemaMinLength
	}
	// "maxLength"
	if schema.GetHasMaxLength() {
		s["maxLength"] = schema.GetMaxLength()
	}
	// "pattern"
	if schemaPattern := schema.GetPattern(); schemaPattern != "" {
		s["pattern"] = schemaPattern
	}

	// "minimum"
	if schema.GetHasMinimum() {
		s["minimum"] = schema.GetMinimum()
	}
	// "maximum"
	if schema.GetHasMaximum() {
		s["maximum"] = schema.GetMaximum()
	}
	// "exclusiveMinimum"
	if schemaExclusiveMinimum := schema.GetExclusiveMinimum(); schemaExclusiveMinimum {
		s["exclusiveMin"] = schemaExclusiveMinimum
	}
	// "exclusiveMaximum"
	if schemaExclusiveMaximum := schema.GetExclusiveMaximum(); schemaExclusiveMaximum {
		s["exclusiveMax"] = schemaExclusiveMaximum
	}
	// "multipleOf"
	if mulOf := schema.GetTranslatedMultipleOf(); mulOf != 0.0 {
		s["multipleOf"] = mulOf + 1.0
	}

	// "uniqueItems"
	if schemaUniqueItems := schema.GetUniqueItems(); schemaUniqueItems {
		s["uniqueItems"] = schemaUniqueItems
	}
	// "minItems"
	if schemaMinItems := schema.GetMinItems(); schemaMinItems != 0 {
		s["minItems"] = schemaMinItems
	}
	// "maxItems"
	if schema.GetHasMaxItems() {
		s["maxItems"] = schema.GetMaxItems()
	}
	// "items"
	if schemaItems := schema.GetItems(); len(schemaItems) > 0 {
		items := make([]schemaJSON, len(schemaItems))
		for i, itemSchema := range schemaItems {
			items[i] = sm.toGo(itemSchema)
		}
		s["items"] = items
	}

	// "minProperties"
	if schemaMinProps := schema.GetMinProperties(); schemaMinProps != 0 {
		s["minProps"] = schemaMinProps
	}
	// "maxProperties"
	if schema.GetHasMaxProperties() {
		s["maxProperties"] = schema.GetMaxProperties()
	}
	// "required"
	if schemaRequired := schema.GetRequired(); len(schemaRequired) != 0 {
		s["required"] = schemaRequired
	}
	// "properties"
	if schemaProps := schema.GetProperties(); len(schemaProps) != 0 {
		props := make(schemaJSON, len(schemaProps))
		for propName, propSchema := range schemaProps {
			props[propName] = sm.toGo(propSchema)
		}
		s["properties"] = props
	}

	// "allOf"
	if schemaAllOf := schema.GetAllOf(); len(schemaAllOf) != 0 {
		allOf := make([]schemaJSON, len(schemaAllOf))
		for i, schemaOf := range schemaAllOf {
			allOf[i] = sm.toGo(schemaOf)
		}
		s["allOf"] = allOf
	}

	// "anyOf"
	if schemaAnyOf := schema.GetAnyOf(); len(schemaAnyOf) != 0 {
		anyOf := make([]schemaJSON, len(schemaAnyOf))
		for i, schemaOf := range schemaAnyOf {
			anyOf[i] = sm.toGo(schemaOf)
		}
		s["anyOf"] = anyOf
	}

	// "oneOf"
	if schemaOneOf := schema.GetOneOf(); len(schemaOneOf) != 0 {
		oneOf := make([]schemaJSON, len(schemaOneOf))
		for i, schemaOf := range schemaOneOf {
			oneOf[i] = sm.toGo(schemaOf)
		}
		s["oneOf"] = oneOf
	}

	// "not"
	if schemaNot := schema.GetNot(); 0 != schemaNot {
		s["not"] = sm.toGo(schemaNot)
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

func formatToGo(format Schema_JSON_Format) string {
	switch format {
	case Schema_JSON_NONE:
		return ""
	case Schema_JSON_date_time:
		return "date-time"
	case Schema_JSON_uri_reference:
		return "uri-reference"
	default:
		return format.String()
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

func EnumToGo(value *ValueJSON) interface{} {
	if value.GetIsNull() {
		return nil
	}
	switch value.GetValue().(type) {
	case *ValueJSON_Boolean:
		return value.GetBoolean()
	case *ValueJSON_Number:
		return value.GetNumber()
	case *ValueJSON_Text:
		return value.GetText()
	case *ValueJSON_Array:
		val := value.GetArray().GetValues()
		vs := make([]interface{}, len(val))
		for i, v := range val {
			vs[i] = EnumToGo(v)
		}
		return vs
	case *ValueJSON_Object:
		val := value.GetObject().GetValues()
		vs := make(map[string]interface{}, len(val))
		for n, v := range val {
			vs[n] = EnumToGo(v)
		}
		return vs
	default:
		panic("unreachable")
	}
}

func compileSlice(strs []string) (res map[string]*regexp.Regexp, err error) {
	res = make(map[string]*regexp.Regexp, len(strs))
	for _, str := range strs {
		var re *regexp.Regexp
		if re, err = regexp.Compile(str); err != nil {
			return
		}
		res[str] = re
	}
	return
}

func (vald *Validator) FilterEndpoints(only, except []string) (eids []eid, err error) {
	// Ensure user input is valid
	onlys, err := compileSlice(only)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}
	if _, err = compileSlice(except); err != nil {
		log.Println("[ERR]", err)
		return
	}

	total := len(vald.Spec.Endpoints)
	all := make(map[eid]string, total)
	for eid := range vald.Spec.Endpoints {
		e := vald.Spec.Endpoints[eid].GetJson()
		path := pathToOA3(e.PathPartials)
		all[eid] = fmt.Sprintf("%s\t%s", e.Method, path)
	}

	var es map[eid]string
	if len(only) == 0 {
		es = all
	} else {
		es = make(map[eid]string, total)
		// OR over `only`
		// Fail if any `only` is not there
		for str, re := range onlys {
			matched := false
			for eid, e := range all {
				if re.MatchString(e) {
					log.Println("[DBG]", str, "matched", e)
					matched = true
					if _, ok := es[eid]; !ok {
						es[eid] = e
					}
				}
			}
			if !matched {
				err = fmt.Errorf("%s did not match any endpoints", str)
				log.Println("[ERR]", err)
				return
			}
		}
	}

	// OR over `except`
	// Do not error if any `except` is not there
	if len(except) != 0 {
		var excepts *regexp.Regexp
		if excepts, err = regexp.Compile(strings.Join(except, "|")); err != nil {
			log.Println("[ERR]", err)
			return
		}
		for eid, e := range es {
			if excepts.MatchString(e) {
				delete(es, eid)
			}
		}
	}

	// TODO? fallback from regexp to JSON Pointer matching
	// TODO: ensure PTRs are valid & exist
	// TODO: filter on endpoints
	// TODO: filter on schemas (i.e.: on endpoints which I/O given schemas)
	// TODO: golang (Docker style) filtering in general (tags, operationIDs, ...)

	// Fail on empty EIDs
	if len(es) == 0 {
		err = errors.New("0 endpoints selected for testing")
		log.Println("[ERR]", err)
		return
	}

	selected := uint32(len(es))
	eids = make([]eid, 0, selected)
	for eid := range es {
		eids = append(eids, eid)
	}
	sort.Slice(eids, func(i, j int) bool { return eids[i] < eids[j] })

	e := fmt.Sprintf("%d %s selected for testing", selected, plural("endpoint", selected))
	log.Println("[NFO]", e)
	ColorNFO.Println(e)
	for _, eid := range eids {
		fmt.Println(es[eid])
	}
	return
}

func (vald *Validator) WriteAbsoluteReferences(w io.Writer) {
	all := make([]string, 0, len(vald.Refs))
	for absRef := range vald.Refs {
		all = append(all, absRef)
	}
	sort.Slice(all, func(i, j int) bool {
		return strings.ToLower(all[i]) < strings.ToLower(all[j])
	})
	for _, absRef := range all {
		fmt.Fprintln(w, absRef)
	}
}

func (vald *Validator) ValidateAgainstSchema(absRef string) (err error) {
	if _, ok := vald.Refs[absRef]; !ok {
		err = ErrNoSuchRef
		return
	}

	var data []byte
	if data, err = ioutil.ReadAll(os.Stdin); err != nil {
		log.Println("[ERR]", err)
		return
	}
	var value interface{}
	if err = json.Unmarshal(data, &value); err != nil {
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

func (ss *Schemas) Validate(SID sid, json_data interface{}) []string {
	var sm schemap
	sm = ss.GetJson()
	s := sm.toGo(SID)
	log.Printf("[???] SID:%d %+v", SID, s)
	// FIXME? turns out Compile does not need an $id set?
	// id := fmt.Sprintf("file:///schema_%d.json", SID)
	// s["$id"] = id
	loader := gojsonschema.NewGoLoader(s)

	log.Println("[NFO] compiling schema refs")
	refd := gojsonschema.NewSchemaLoader()
	for _, refOrSchema := range sm {
		if ptr := refOrSchema.GetPtr(); ptr != nil {
			SID, ref := ptr.GetSID(), ptr.GetRef()
			s := sm.toGo(SID)
			// log.Printf("[???] SID:%d %s %+v", SID, ref, s)
			sl := gojsonschema.NewGoLoader(s)
			if err := refd.AddSchema(ref, sl); err != nil {
				panic(err)
			}
		}
	}
	schema, err := refd.Compile(loader)
	if err != nil {
		log.Println("[ERR]", err)
		return []string{err.Error()}
	}

	log.Println("[NFO] validating payload against refs")
	res, err := schema.Validate(gojsonschema.NewGoLoader(json_data))
	if err != nil {
		log.Println("[ERR]", err)
		return []string{err.Error()}
	}

	errors := res.Errors()
	errs := make([]string, 0, len(errors))
	for _, e := range errors {
		errs = append(errs, e.String())
	}
	return errs
}
