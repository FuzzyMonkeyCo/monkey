package modeler_openapiv3

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"regexp"
	"sort"
	"strings"

	"github.com/FuzzyMonkeyCo/monkey/pkg/as"
	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
	"github.com/gogo/protobuf/types"
	"github.com/xeipuuv/gojsonschema"
)

type validator struct {
	Spec *fm.SpecIR
	Refs map[string]sid
	Refd *gojsonschema.SchemaLoader
}

func newValidator(capaEndpoints, capaSchemas int) *validator {
	return &validator{
		Refs: make(map[string]sid, capaSchemas),
		Spec: &fm.SpecIR{
			Endpoints: make(map[eid]*fm.Endpoint, capaEndpoints),
			Schemas:   &fm.Schemas{Json: make(map[sid]*fm.RefOrSchemaJSON, capaSchemas)},
		},
		Refd: gojsonschema.NewSchemaLoader(),
	}
}

func (vald *validator) newSID() sid {
	return sid(1 + len(vald.Spec.Schemas.Json))
}

func (vald *validator) seed(base string, schemas schemasJSON) (err error) {
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
		vald.Spec.Schemas.Json[refSID] = &fm.RefOrSchemaJSON{
			PtrOrSchema: &fm.RefOrSchemaJSON_Ptr{&fm.SchemaPtr{Ref: absRef, SID: 0}},
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
		vald.Spec.Schemas.Json[refSID] = &fm.RefOrSchemaJSON{
			PtrOrSchema: &fm.RefOrSchemaJSON_Ptr{&fm.SchemaPtr{Ref: absRef, SID: sid}},
		}
	}
	return
}

func (vald *validator) ensureMapped(ref string, goSchema schemaJSON) sid {
	if ref == "" {
		schema := vald.fromGo(goSchema)
		for SID, schemaPtr := range vald.Spec.Schemas.Json {
			if s := schemaPtr.GetSchema(); s != nil && schema.Equal(s) {
				return SID
			}
		}
		SID := vald.newSID()
		vald.Spec.Schemas.Json[SID] = &fm.RefOrSchemaJSON{
			PtrOrSchema: &fm.RefOrSchemaJSON_Schema{&schema},
		}
		return SID
	}

	mappedSID, ok := vald.Refs[ref]
	if !ok {
		// Every $ref should already be in here
		panic(ref)
	}
	schemaPtr := &fm.SchemaPtr{Ref: ref, SID: mappedSID}
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
	vald.Spec.Schemas.Json[SID] = &fm.RefOrSchemaJSON{
		PtrOrSchema: &fm.RefOrSchemaJSON_Ptr{schemaPtr},
	}
	return SID
}

func (vald *validator) fromGo(s schemaJSON) (schema fm.Schema_JSON) {
	// "enum"
	if v, ok := s["enum"]; ok {
		enum := v.([]interface{})
		schema.Enum = make([]*types.Value, len(enum))
		for i, vv := range enum {
			schema.Enum[i] = enumFromGo(vv)
		}
	}

	// "type"
	if v, ok := s["type"]; ok {
		types := v.([]string)
		schema.Types = make([]fm.Schema_JSON_Type, len(types))
		for i, vv := range types {
			schema.Types[i] = fm.Schema_JSON_Type(fm.Schema_JSON_Type_value[vv])
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

type schemap map[sid]*fm.RefOrSchemaJSON

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
	if schemaFormat := schema.GetFormat(); schemaFormat != fm.Schema_JSON_NONE {
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

func formatFromGo(format string) fm.Schema_JSON_Format {
	switch format {
	case "date-time":
		return fm.Schema_JSON_date_time
	case "uriref", "uri-reference":
		return fm.Schema_JSON_uri_reference
	default:
		v, ok := fm.Schema_JSON_Format_value[format]
		if ok {
			return fm.Schema_JSON_Format(v)
		}
		return fm.Schema_JSON_NONE
	}
}

func formatToGo(format fm.Schema_JSON_Format) string {
	switch format {
	case fm.Schema_JSON_NONE:
		return ""
	case fm.Schema_JSON_date_time:
		return "date-time"
	case fm.Schema_JSON_uri_reference:
		return "uri-reference"
	default:
		return format.String()
	}
}

func enumFromGo(value interface{}) *types.Value {
	if value == nil {
		return &types.Value{Kind: &types.Value_NullValue{types.NullValue_NULL_VALUE}}
	}
	switch val := value.(type) {
	case bool:
		return &types.Value{Kind: &types.Value_BoolValue{val}}
	case uint32:
		return &types.Value{Kind: &types.Value_NumberValue{float64(val)}}
	case float64:
		return &types.Value{Kind: &types.Value_NumberValue{val}}
	case string:
		return &types.Value{Kind: &types.Value_StringValue{val}}
	case []interface{}:
		vs := make([]*types.Value, len(val))
		for i, v := range val {
			vs[i] = enumFromGo(v)
		}
		return &types.Value{Kind: &types.Value_ListValue{ListValue: &types.ListValue{Values: vs}}}
	case map[string]interface{}:
		vs := make(map[string]*types.Value, len(val))
		for n, v := range val {
			vs[n] = enumFromGo(v)
		}
		return &types.Value{Kind: &types.Value_StructValue{StructValue: &types.Struct{Fields: vs}}}
	default:
		panic(fmt.Errorf("cannot convert to value type: %T", value))
	}
}

func EnumToGo(value *types.Value) interface{} {
	switch value.GetKind().(type) {
	case *types.Value_NullValue:
		return nil
	case *types.Value_BoolValue:
		return value.GetBoolValue()
	case *types.Value_NumberValue:
		return value.GetNumberValue()
	case *types.Value_StringValue:
		return value.GetStringValue()
	case *types.Value_ListValue:
		val := value.GetListValue().GetValues()
		vs := make([]interface{}, len(val))
		for i, v := range val {
			vs[i] = EnumToGo(v)
		}
		return vs
	case *types.Value_StructValue:
		val := value.GetStructValue().GetFields()
		vs := make(map[string]interface{}, len(val))
		for n, v := range val {
			vs[n] = EnumToGo(v)
		}
		return vs
	default:
		panic("unreachable")
	}
}

func (vald *validator) FilterEndpoints(args []string) (eids []eid, err error) {
	// TODO? filter on 2nd, 3rd, ... -level schemas
	// instead of just first level (ref A references B & C)

	// TODO: use Go templates to filter on very specific fields. See:
	// https://github.com/kubernetes/kubernetes/blob/c0d9a0728ce5920f97fecab977be15636e57126b/staging/src/k8s.io/cli-runtime/pkg/genericclioptions/printers/jsonpath.go#L143
	// https://github.com/kubernetes/kubernetes/blob/103813057c5ef6cc416e6fdb71515e90d98cd3a9/staging/src/k8s.io/cli-runtime/pkg/genericclioptions/printers/template.go#L85

	const fmtMPIO = "%s\t%s\t%s ➜ %s"
	total := len(vald.Spec.Endpoints)
	all := make(map[eid]string, total)
	for eid := range vald.Spec.Endpoints {
		e := vald.Spec.Endpoints[eid].GetJson()
		path := pathToOA3(e.PathPartials)
		inputs := make([]sid, 0, len(e.Inputs))
		for _, param := range e.Inputs {
			inputs = append(inputs, param.SID)
		}
		ins := strings.Join(vald.refsFromSIDs(inputs), " | ")
		outputs := make([]sid, 0, len(e.Outputs))
		for _, SID := range e.Outputs {
			outputs = append(outputs, SID)
		}
		outs := strings.Join(vald.refsFromSIDs(outputs), " | ")
		all[eid] = fmt.Sprintf(fmtMPIO, e.Method, path, ins, outs)
	}

	{
		argz := make([]string, 0, len(args))
	outter:
		for i := 0; i < len(args); i++ {
			arg := args[i]
			for _, p := range []string{"--only=", "--except=",
				"--calls-with-input=", "--calls-without-input=",
				"--calls-with-output=", "--calls-without-output=",
			} {
				l := len(p)
				if len(arg) > l && p == arg[0:l] {
					argz = append(argz, []string{p[0 : l-1], arg[l:]}...)
					break outter
				}
			}
			argz = append(argz, arg)
		}
		args = argz
	}

	for i := 0; i < len(args); i++ {
		cmd := args[i]
		i++
		switch cmd {
		case "--only":
			err = filterEndpoints(all, true, args[i])
		case "--except":
			err = filterEndpoints(all, false, args[i])
		case "--calls-with-input":
			err = filterEndpoints(all, true, "^[^\t]+\t[^\t]+\t([^\t]*"+args[i]+"[^\t]*) ➜ [^$]*$")
		case "--calls-without-input":
			err = filterEndpoints(all, false, "^[^\t]+\t[^\t]+\t([^\t]*"+args[i]+"[^\t]*) ➜ [^$]*$")
		case "--calls-with-output":
			err = filterEndpoints(all, true, "^[^\t]+\t[^\t]+\t[^\t]* ➜ ([^\t]*"+args[i]+"[^\t]*)$")
		case "--calls-without-output":
			err = filterEndpoints(all, false, "^[^\t]+\t[^\t]+\t[^\t]* ➜ ([^\t]*"+args[i]+"[^\t]*)$")
		default:
			i--
		}
		if err != nil {
			return
		}
	}

	selected := uint32(len(all))
	e := fmt.Sprintf("%d of %d %s selected for testing", selected, total, plural("endpoint", selected))
	if selected == 0 {
		err = errors.New(e)
		log.Println("[ERR]", err)
		// Error printed in main.go
		return
	}

	log.Println("[NFO]", e)
	as.ColorNFO.Println(e)
	eids = make([]eid, 0, selected)
	for eid := range all {
		eids = append(eids, eid)
	}
	sort.Slice(eids, func(i, j int) bool { return eids[i] < eids[j] })
	for _, eid := range eids {
		fmt.Println(all[eid])
	}
	return
}

func plural(s string, n uint32) string {
	if n == 1 {
		return s
	}
	return s + "s"
}

func filterEndpoints(all map[eid]string, only bool, pattern string) (err error) {
	var re *regexp.Regexp
	if re, err = regexp.Compile(pattern); err != nil {
		log.Println("[ERR]", err)
		return
	}

	onlyMatched := false
	for eid, e := range all {
		if re.MatchString(e) {
			log.Println("[DBG]", pattern, "matched", e)
			onlyMatched = true
			if !only {
				delete(all, eid)
			}
		} else if only {
			delete(all, eid)
		}
	}
	if only && !onlyMatched {
		// Fail if any `only` is not there
		err = fmt.Errorf("%s did not match any endpoints", pattern)
		log.Println("[ERR]", err)
	}
	return
}

func (vald *validator) InputsCount() int {
	return len(vald.Refs)
}

func (vald *validator) WriteAbsoluteReferences(w io.Writer) {
	if vald.InputsCount() != 0 {
		as.ColorNFO.Fprintln(w, "Available types:")
	}

	all := make([]string, 0, vald.InputsCount())
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

func (vald *validator) ValidateAgainstSchema(absRef string, data []byte) (err error) {
	if _, ok := vald.Refs[absRef]; !ok {
		err = modeler.ErrNoSuchRef
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
		as.ColorERR.Println(e)
	}
	if len(errs) > 0 {
		err = modeler.ErrUnparsablePayload
	}
	return
}

func (vald *validator) Validate(SID sid, json_data interface{}) []string {
	var sm schemap
	sm = vald.Spec.Schemas.GetJson()
	s := sm.toGo(SID)

	data, ok := json_data.(*types.Value)
	if !ok {
		panic(fmt.Sprintf("%T", json_data))
	}
	toValidate := EnumToGo(data)
	log.Printf("[DBG] SID:%d -> %+v against %+v", SID, s, toValidate)

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
	res, err := schema.Validate(gojsonschema.NewGoLoader(toValidate))
	if err != nil {
		log.Println("[ERR]", err)
		return []string{err.Error()}
	}

	errors := res.Errors()
	errs := make([]string, 0, len(errors))
	for _, e := range errors {
		errs = append(errs, e.String())
		log.Printf("[ERR] value: %s", e.Value())
	}
	return errs
}
