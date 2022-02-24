package fm

// code from gogo-generated Equal methods
// FIXME: whence https://github.com/planetscale/vtprotobuf/pull/28

import (
	"google.golang.org/protobuf/types/known/structpb"
)

func (this *Schema_JSON) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*Schema_JSON)
	if !ok {
		that2, ok := that.(Schema_JSON)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		return this == nil
	} else if this == nil {
		return false
	}
	if len(this.Types) != len(that1.Types) {
		return false
	}
	for i := range this.Types {
		if this.Types[i] != that1.Types[i] {
			return false
		}
	}
	if len(this.Enum) != len(that1.Enum) {
		return false
	}
	for i := range this.Enum {
		// if !this.Enum[i].Equal(that1.Enum[i]) {
		if !equalValue(this.Enum[i], that1.Enum[i]) {
			return false
		}
	}
	if this.Format != that1.Format {
		return false
	}
	if this.MinLength != that1.MinLength {
		return false
	}
	if this.MaxLength != that1.MaxLength {
		return false
	}
	if this.HasMaxLength != that1.HasMaxLength {
		return false
	}
	if this.Pattern != that1.Pattern {
		return false
	}
	if this.Minimum != that1.Minimum {
		return false
	}
	if this.Maximum != that1.Maximum {
		return false
	}
	if this.HasMinimum != that1.HasMinimum {
		return false
	}
	if this.HasMaximum != that1.HasMaximum {
		return false
	}
	if this.TranslatedMultipleOf != that1.TranslatedMultipleOf {
		return false
	}
	if this.ExclusiveMinimum != that1.ExclusiveMinimum {
		return false
	}
	if this.ExclusiveMaximum != that1.ExclusiveMaximum {
		return false
	}
	if len(this.Items) != len(that1.Items) {
		return false
	}
	for i := range this.Items {
		if this.Items[i] != that1.Items[i] {
			return false
		}
	}
	if this.UniqueItems != that1.UniqueItems {
		return false
	}
	if this.MinItems != that1.MinItems {
		return false
	}
	if this.MaxItems != that1.MaxItems {
		return false
	}
	if this.HasMaxItems != that1.HasMaxItems {
		return false
	}
	if len(this.Properties) != len(that1.Properties) {
		return false
	}
	for i := range this.Properties {
		if this.Properties[i] != that1.Properties[i] {
			return false
		}
	}
	if len(this.Required) != len(that1.Required) {
		return false
	}
	for i := range this.Required {
		if this.Required[i] != that1.Required[i] {
			return false
		}
	}
	if this.MinProperties != that1.MinProperties {
		return false
	}
	if this.MaxProperties != that1.MaxProperties {
		return false
	}
	if this.HasMaxProperties != that1.HasMaxProperties {
		return false
	}
	if !this.AdditionalProperties.Equal(that1.AdditionalProperties) {
		return false
	}
	if this.HasAdditionalProperties != that1.HasAdditionalProperties {
		return false
	}
	if len(this.AllOf) != len(that1.AllOf) {
		return false
	}
	for i := range this.AllOf {
		if this.AllOf[i] != that1.AllOf[i] {
			return false
		}
	}
	if len(this.AnyOf) != len(that1.AnyOf) {
		return false
	}
	for i := range this.AnyOf {
		if this.AnyOf[i] != that1.AnyOf[i] {
			return false
		}
	}
	if len(this.OneOf) != len(that1.OneOf) {
		return false
	}
	for i := range this.OneOf {
		if this.OneOf[i] != that1.OneOf[i] {
			return false
		}
	}
	if this.Not != that1.Not {
		return false
	}
	// if !bytes.Equal(this.XXX_unrecognized, that1.XXX_unrecognized) {
	// 	return false
	// }
	return true
}
func (this *Schema_JSON_AdditionalProperties) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*Schema_JSON_AdditionalProperties)
	if !ok {
		that2, ok := that.(Schema_JSON_AdditionalProperties)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		return this == nil
	} else if this == nil {
		return false
	}
	if that1.AddProps == nil {
		if this.AddProps != nil {
			return false
		}
	} else if this.AddProps == nil {
		return false
		// } else if !this.AddProps.Equal(that1.AddProps) {
		// 	return false
	} else {
		if x, ok := this.AddProps.(*Schema_JSON_AdditionalProperties_AlwaysSucceed); ok {
			if !x.Equal(that1.AddProps) {
				return false
			}
		}
		if x, ok := this.AddProps.(*Schema_JSON_AdditionalProperties_SID); ok {
			if !x.Equal(that1.AddProps) {
				return false
			}
		}
	}
	// if !bytes.Equal(this.XXX_unrecognized, that1.XXX_unrecognized) {
	// 	return false
	// }
	return true
}
func (this *Schema_JSON_AdditionalProperties_AlwaysSucceed) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*Schema_JSON_AdditionalProperties_AlwaysSucceed)
	if !ok {
		that2, ok := that.(Schema_JSON_AdditionalProperties_AlwaysSucceed)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		return this == nil
	} else if this == nil {
		return false
	}
	if this.AlwaysSucceed != that1.AlwaysSucceed {
		return false
	}
	return true
}
func (this *Schema_JSON_AdditionalProperties_SID) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*Schema_JSON_AdditionalProperties_SID)
	if !ok {
		that2, ok := that.(Schema_JSON_AdditionalProperties_SID)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		return this == nil
	} else if this == nil {
		return false
	}
	if this.SID != that1.SID {
		return false
	}
	return true
}

func equalValue(this, that *structpb.Value) bool {
	switch x := this.GetKind().(type) {
	case *structpb.Value_NullValue:
		_, ok := that.GetKind().(*structpb.Value_NullValue)
		return ok
	case *structpb.Value_BoolValue:
		y, ok := that.GetKind().(*structpb.Value_BoolValue)
		if !ok {
			return false
		}
		return x == y
	case *structpb.Value_NumberValue:
		y, ok := that.GetKind().(*structpb.Value_NumberValue)
		if !ok {
			return false
		}
		return x == y
	case *structpb.Value_StringValue:
		y, ok := that.GetKind().(*structpb.Value_StringValue)
		if !ok {
			return false
		}
		return x == y
	case *structpb.Value_ListValue:
		if _, ok := that.GetKind().(*structpb.Value_ListValue); !ok {
			return false
		}
		xs := x.ListValue
		ys := that.GetListValue()
		if len(xs.Values) != len(ys.Values) {
			return false
		}
		for i := range make([]struct{}, len(xs.Values)) {
			if !equalValue(xs.Values[i], ys.Values[i]) {
				return false
			}
		}
		return true
	case *structpb.Value_StructValue:
		if _, ok := that.GetKind().(*structpb.Value_StructValue); !ok {
			return false
		}
		xs := x.StructValue
		ys := that.GetStructValue()
		if len(xs.Fields) != len(ys.Fields) {
			return false
		}
		for i := range xs.Fields {
			if !equalValue(xs.Fields[i], ys.Fields[i]) {
				return false
			}
		}
		return true
	default:
		panic("unreachable")
	}
}
