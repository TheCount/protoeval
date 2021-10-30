package protoeval

import (
	"fmt"
	"reflect"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/anypb"
)

// anypbName is the full name of the well-known Any protobuf message.
var anypbName = (&anypb.Any{}).ProtoReflect().Descriptor().FullName()

// getProtoType returns the Go type for the protobuf type specified by the
// protobuf kind and protobuf type name.
// typ must be the empty string for protobuf kinds which determine the type
// automatically.
//
// If kind is Value_INVALID and typ is empty, the type of interface{} is
// returned.
//
// If kind is Value_MESSAGE, type must be the full message type name,
// and the message type must be linked in the binary.
//
// If kind is Value_ENUM, type must be the full enum type name. In this case,
// the returned type is the type of protoreflect.EnumNumber.
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
			return reflect.TypeOf(msgType.Zero().Interface()), nil
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
			return reflect.TypeOf(protoreflect.EnumNumber(0)), nil
		default:
			return nil, fmt.Errorf("non-empty type '%s' with protobuf kind '%s'",
				typeName, kind)
		}
	}
	switch kind {
	case Value_INVALID:
		return reflect.TypeOf((*interface{})(nil)).Elem(), nil
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
			"kind '%s' invalid or requires a non-empty type name", kind)
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

// go2protofd converts the given go value to a value assignable to the given
// field in the given protobuf message.
func go2protofd(
	goval interface{}, msg protoreflect.Message, fd protoreflect.FieldDescriptor,
) (protoreflect.Value, error) {
	value := reflect.ValueOf(goval)
	switch {
	case fd.IsList():
		if value.Kind() != reflect.Slice && value.Kind() != reflect.Array {
			return protoreflect.Value{}, fmt.Errorf("expected list, got %T", goval)
		}
		typeName := ""
		switch fd.Kind() {
		case protoreflect.EnumKind:
			typeName = string(fd.Enum().FullName())
		case protoreflect.MessageKind:
			typeName = string(fd.Message().FullName())
		}
		elemType, err := getProtoType(Value_Kind(fd.Kind()), typeName)
		if err != nil {
			return protoreflect.Value{}, err
		}
		result := msg.NewField(fd).List()
		for i := 0; i != value.Len(); i++ {
			elemValue := value.Index(i)
			if !elemValue.Type().ConvertibleTo(elemType) {
				return protoreflect.Value{}, fmt.Errorf(
					"unable to convert list element %d type %s to %s",
					i, elemValue.Type(), elemType)
			}
			converted := elemValue.Convert(elemType).Interface()
			if x, ok := converted.(proto.Message); ok {
				converted = x.ProtoReflect()
			}
			result.Append(protoreflect.ValueOf(converted))
		}
		return protoreflect.ValueOfList(result), nil
	case fd.IsMap():
		if value.Kind() != reflect.Map {
			return protoreflect.Value{}, fmt.Errorf("expected map, got %T", goval)
		}
		keyType, err := getProtoMapKeyType(Value_Kind(fd.MapKey().Kind()))
		if err != nil {
			return protoreflect.Value{}, err
		}
		typeName := ""
		switch fd.MapValue().Kind() {
		case protoreflect.EnumKind:
			typeName = string(fd.MapValue().Enum().FullName())
		case protoreflect.MessageKind:
			typeName = string(fd.MapValue().Message().FullName())
		}
		valueType, err := getProtoType(Value_Kind(fd.MapValue().Kind()), typeName)
		if err != nil {
			return protoreflect.Value{}, err
		}
		result := msg.NewField(fd).Map()
		for iter := value.MapRange(); iter.Next(); {
			k, v := iter.Key(), iter.Value()
			if !k.Type().ConvertibleTo(keyType) {
				return protoreflect.Value{}, fmt.Errorf(
					"unable to convert map key %v of type %s to %s",
					k.Interface(), k.Type(), keyType)
			}
			if !v.Type().ConvertibleTo(valueType) {
				return protoreflect.Value{}, fmt.Errorf(
					"unable to convert map value for key %v of type %s to %s",
					k.Interface(), v.Type(), valueType)
			}
			keyValue := protoreflect.ValueOf(k.Convert(keyType).Interface()).MapKey()
			convertedValue := v.Convert(valueType).Interface()
			if x, ok := convertedValue.(proto.Message); ok {
				convertedValue = x.ProtoReflect()
			}
			result.Set(keyValue, protoreflect.ValueOf(convertedValue))
		}
		return protoreflect.ValueOfMap(result), nil
	// the following case must come after the list/map check
	case fd.Kind() == protoreflect.MessageKind:
		if x, ok := goval.(proto.Message); ok {
			return protoreflect.ValueOf(x.ProtoReflect()), nil
		}
		return protoreflect.Value{},
			fmt.Errorf("expected proto.Message, got %T", goval)
	case fd.Kind() == protoreflect.EnumKind:
		enumType, err := getProtoType(Value_ENUM, string(fd.Enum().FullName()))
		if err != nil {
			return protoreflect.Value{}, err
		}
		if !value.Type().ConvertibleTo(enumType) {
			return protoreflect.Value{}, fmt.Errorf(
				"value type %T not convertible to enum number %s", goval, enumType)
		}
		return protoreflect.ValueOf(value.Convert(enumType).Interface()), nil
	default:
		targetType := reflect.TypeOf(msg.NewField(fd).Interface())
		if !value.Type().ConvertibleTo(targetType) {
			return protoreflect.Value{},
				fmt.Errorf("value type %T not convertible to %s", goval, targetType)
		}
		return protoreflect.ValueOf(value.Convert(targetType).Interface()), nil
	}
}
