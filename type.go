package protoeval

import (
	"bytes"
	"fmt"
	"math"
	"reflect"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// getProtoType returns the Go type for the protobuf type specified by the
// protobuf kind and protobuf type name.
// typ must be the empty string for protobuf kinds which determine the type
// automatically.
//
// If kind is Value_INVALID and typ is empty, (nil, nil) is returned.
//
// If kind is Value_MESSAGE, type must be the full message type name,
// and the message type must be linked in the binary.
//
// If kind is Value_ENUM, type must be the full enum type name. In this case,
// the returned type is the type of protoreflect.EnumValueDescriptor.
func getProtoType(kind Value_Kind, typeName string) (reflect.Type, error) {
	protoName := protoreflect.FullName(typeName)
	if protoName != "" {
		if !protoName.IsValid() {
			return nil, fmt.Errorf("invalid protobuf type name: %s", protoName)
		}
		switch kind {
		case Value_MESSAGE:
			msgType, err := protoregistry.GlobalTypes.FindMessageByName(protoName)
			if err != nil {
				return nil, fmt.Errorf("find protobuf message type '%s': %w",
					protoName, err)
			}
			return reflect.TypeOf(msgType.New().Interface()), nil
		case Value_ENUM:
			enumType, err := protoregistry.GlobalTypes.FindEnumByName(protoName)
			if err != nil {
				return nil, fmt.Errorf("find protobuf enum type '%s': %w",
					protoName, err)
			}
			values := enumType.Descriptor().Values()
			if values.Len() == 0 { // can only happen with proto2 enums
				return nil, fmt.Errorf("enum '%s' is empty", typeName)
			}
			return reflect.TypeOf(values.Get(0)), nil
		default:
			return nil, fmt.Errorf("non-empty type '%s' with protobuf kind '%s'",
				typeName, kind)
		}
	}
	switch kind {
	case Value_INVALID:
		return nil, nil
	case Value_DOUBLE:
		return reflect.TypeOf(float64(0)), nil
	case Value_FLOAT:
		return reflect.TypeOf(float32(0)), nil
	case Value_INT64, Value_SFIXED64, Value_SINT64:
		return reflect.TypeOf(int64(0)), nil
	case Value_UINT64, Value_FIXED64:
		return reflect.TypeOf(uint64(0)), nil
	case Value_INT32, Value_SFIXED32, Value_SINT32:
		return reflect.TypeOf(int32(0)), nil
	case Value_BOOL:
		return reflect.TypeOf(false), nil
	case Value_STRING:
		return reflect.TypeOf(""), nil
	case Value_BYTES:
		return reflect.TypeOf([]byte{}), nil
	case Value_UINT32, Value_FIXED32:
		return reflect.TypeOf(uint32(0)), nil
	default:
		return nil, fmt.Errorf(
			"kind '%s' invalid or requires a non-zero type name", kind)
	}
}

// getProtoMapKeyType is like getProtoType, except that only valid protobuf
// map key types are returned.
//
// If kind is Value_INVALID, (nil, nil) is returned.
func getProtoMapKeyType(kind Value_Kind) (reflect.Type, error) {
	switch kind {
	case Value_INVALID, Value_INT64, Value_UINT64, Value_INT32, Value_FIXED64,
		Value_FIXED32, Value_BOOL, Value_STRING, Value_UINT32, Value_SFIXED32,
		Value_SFIXED64, Value_SINT32, Value_SINT64:
		return getProtoType(kind, "")
	default:
		return nil, fmt.Errorf("invalid protobuf map key kind '%s'", kind)
	}
}

// isValidMapKeyType determines whether typ is a valid protobuf map key type.
func isValidMapKeyType(typ reflect.Type) bool {
	switch typ.Kind() {
	case reflect.Int64, reflect.Uint64, reflect.Int32, reflect.Uint32,
		reflect.Bool, reflect.String:
		return true
	default:
		return false
	}
}

// protoEqual generalises proto.Equal to all possible protobuf values, not
// just messages.
func protoEqual(a, b interface{}) bool {
	switch x := a.(type) {
	case nil, int64, uint64, int32, bool, string:
		return a == b
	case float32: // special NaN treatment
		y, ok := b.(float32)
		if math.IsNaN(float64(x)) {
			return ok && math.IsNaN(float64(y))
		}
		return ok && x == y
	case float64: // special NaN treatment
		y, ok := b.(float64)
		if math.IsNaN(x) {
			return ok && math.IsNaN(y)
		}
		return ok && x == y
	case proto.Message:
		y, ok := b.(proto.Message)
		return ok && proto.Equal(x, y)
	case []byte:
		y, ok := b.([]byte)
		return ok && bytes.Equal(x, y)
	case protoreflect.EnumValueDescriptor:
		y, ok := b.(protoreflect.EnumValueDescriptor)
		return ok &&
			x.Parent().FullName() == y.Parent().FullName() &&
			x.Number() == y.Number()
	}
	x := reflect.ValueOf(a)
	y := reflect.ValueOf(b)
	xType := x.Type()
	yType := y.Type()
	switch xType.Kind() {
	case reflect.Slice:
		if yType.Kind() != reflect.Slice {
			return false
		}
		if x.Len() != y.Len() {
			return false
		}
		for i := 0; i < x.Len(); i++ {
			if !protoEqual(x.Index(i).Interface(), y.Index(i).Interface()) {
				return false
			}
		}
		return true
	case reflect.Map:
		if yType.Kind() != reflect.Map || xType.Key() != yType.Key() {
			return false
		}
		if x.Len() != y.Len() {
			return false
		}
		keys := x.MapKeys()
		for _, key := range keys {
			xVal := x.MapIndex(key)
			yVal := y.MapIndex(key)
			if !yVal.IsValid() {
				return false
			}
			if !protoEqual(xVal.Interface(), yVal.Interface()) {
				return false
			}
		}
		return true
	default:
		panic(fmt.Sprintf("BUG: unsupported protoEqual types %T/%T", a, b))
	}
}

// isComparable reports whether the specified type is comparable.
// Floating point types count as comparable, even though the NaN value is not
// comparable. Enum types are not comparable. String and []byte are comparable.
func isComparable(typ reflect.Type) bool {
	switch typ {
	case reflect.TypeOf(float64(0)), reflect.TypeOf(float32(0)),
		reflect.TypeOf(int64(0)), reflect.TypeOf(uint64(0)),
		reflect.TypeOf(int32(0)), reflect.TypeOf(uint32(0)),
		reflect.TypeOf(""), reflect.TypeOf([]byte{}):
		return true
	default:
		return false
	}
}

// isNaN returns true if and only if value is a float32 or float64 NaN value.
func isNaN(value interface{}) bool {
	switch x := value.(type) {
	case float32:
		return math.IsNaN(float64(x))
	case float64:
		return math.IsNaN(x)
	default:
		return false
	}
}

// protoCompare returns -1, 0, or 1 if a is smaller than, equal to, or
// greater than b, respectively. If a and b have distinct protoCompare panics.
// If either a or b is NaN, the result is unspecified.
func protoCompare(a, b interface{}) int {
	s := func(smaller, equal bool) int {
		switch {
		case smaller:
			return -1
		case equal:
			return 0
		default:
			return 1
		}
	}
	switch x := a.(type) {
	case float64:
		y := b.(float64)
		return s(x < y, x == y)
	case float32:
		y := b.(float32)
		return s(x < y, x == y)
	case int64:
		y := b.(int64)
		return s(x < y, x == y)
	case uint64:
		y := b.(uint64)
		return s(x < y, x == y)
	case int32:
		y := b.(int32)
		return s(x < y, x == y)
	case uint32:
		y := b.(uint32)
		return s(x < y, x == y)
	case string:
		y := b.(string)
		return s(x < y, x == y)
	case []byte:
		y := b.([]byte)
		return bytes.Compare(x, y)
	default:
		panic(fmt.Sprintf("BUG: unsupported comparison types %T/%T", a, b))
	}
}

// protoUnpackValue unpacks a protobuf reflection value into unreflected
// Go values. The given descriptor must be the field descriptor for the value,
// except if the value is a message, in which case desc may be nil.
func protoUnpackValue(
	pDesc protoreflect.FieldDescriptor, pValue protoreflect.Value,
) interface{} {
	switch x := pValue.Interface().(type) {
	case protoreflect.Message:
		return x.Interface()
	case protoreflect.List:
		typ := reflect.TypeOf(protoUnpackValue(pDesc, x.NewElement()))
		result := reflect.MakeSlice(reflect.SliceOf(typ), 0, x.Len())
		for i := 0; i < x.Len(); i++ {
			result = reflect.Append(result,
				reflect.ValueOf(protoUnpackValue(pDesc, x.Get(i))))
		}
		return result.Interface()
	case protoreflect.Map:
		keyDesc := pDesc.MapKey()
		valueDesc := pDesc.MapValue()
		keyType, _ := getProtoMapKeyType(Value_Kind(keyDesc.Kind()))
		valueType := reflect.TypeOf(protoUnpackValue(valueDesc, x.NewValue()))
		result := reflect.MakeMapWithSize(
			reflect.MapOf(keyType, valueType), x.Len())
		x.Range(func(key protoreflect.MapKey, value protoreflect.Value) bool {
			result.SetMapIndex(
				reflect.ValueOf(protoUnpackValue(keyDesc, key.Value())),
				reflect.ValueOf(protoUnpackValue(valueDesc, value)),
			)
			return true
		})
		return result.Interface()
	case protoreflect.EnumNumber:
		return pDesc.Enum().Values().ByNumber(x)
	default:
		return pValue.Interface()
	}
}
