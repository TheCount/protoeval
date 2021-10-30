package protoeval

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// celTypeRegistry is the CEL registry of types we make available to all CEL
// programs.
// These are currently all protobuf message types linked into the binary, plus
// some extra types.
var celTypeRegistry ref.TypeRegistry

// initCelTypeRegistryOnce ensures the celTypeRegistry variable is initialised
// only once, via initCelTypeRegistry().
// We don't initialise the celTypeRegistry via this package's init function
// because it might be called before all protobuf types are registered.
var initCelTypeRegistryOnce sync.Once

// initCelTypeRegistry ensures the celTypeRegistry variable is initialised.
func initCelTypeRegistry() {
	initCelTypeRegistryOnce.Do(func() {
		messages := make([]proto.Message, 0)
		protoregistry.GlobalTypes.
			RangeMessages(func(mt protoreflect.MessageType) bool {
				if msg := mt.Zero(); msg != nil {
					messages = append(messages, msg.Interface())
				}
				return true
			})
		reg, err := types.NewRegistry(messages...)
		if err != nil {
			panic(err)
		}
		celTypeRegistry = reg
	})
}

// scope2cel converts a scope to a CEL scope.
func scope2cel(s *scope) (*Scope, error) {
	if s == nil {
		return nil, nil
	}
	parent, err := scope2cel(s.parent)
	if err != nil {
		return nil, fmt.Errorf("parent scope: %w", err)
	}
	result := &Scope{
		Parent: parent,
	}
	if s.desc != nil {
		result.FieldDescriptor = protodesc.ToFieldDescriptorProto(s.desc)
	}
	switch x := s.value.Interface().(type) {
	case protoreflect.List:
		for i := 0; i < x.Len(); i++ {
			elt, err := value2any(x.Get(i).Interface())
			if err != nil {
				return nil, fmt.Errorf("convert list element %d to Any: %w", i, err)
			}
			result.List = append(result.List, elt)
		}
	case protoreflect.Map:
		result.Map = make(map[string]*anypb.Any)
		x.Range(func(key protoreflect.MapKey, value protoreflect.Value) bool {
			var stringKey string
			switch y := key.Interface().(type) {
			case string:
				stringKey = y
			case int32:
				stringKey = strconv.FormatInt(int64(y), 10)
			case int64:
				stringKey = strconv.FormatInt(y, 10)
			case uint32:
				stringKey = strconv.FormatUint(uint64(y), 10)
			case uint64:
				stringKey = strconv.FormatUint(y, 10)
			case bool:
				if y {
					stringKey = "True"
				} else {
					stringKey = "False"
				}
			default:
				panic(fmt.Sprintf("BUG: missing MapKey case for %T", key.Interface()))
			}
			var anyValue *anypb.Any
			anyValue, err = value2any(value.Interface())
			if err != nil {
				err = fmt.Errorf("convert map value for key '%s': %w", stringKey, err)
				return false
			}
			result.Map[stringKey] = anyValue
			return true
		})
		if err != nil {
			return nil, err
		}
	default:
		anyValue, err := value2any(s.value.Interface())
		if err != nil {
			return nil, err
		}
		result.Value = anyValue
	}
	return result, nil
}

// value2any converts the given value to an anypb.
func value2any(value interface{}) (result *anypb.Any, err error) {
	switch x := value.(type) {
	case protoreflect.Message:
		result, err = anypb.New(x.Interface())
	case nil:
		result, err = anypb.New(structpb.NewNullValue())
	case bool:
		result, err = anypb.New(&wrapperspb.BoolValue{Value: x})
	case int32:
		result, err = anypb.New(&wrapperspb.Int32Value{Value: x})
	case int64:
		result, err = anypb.New(&wrapperspb.Int64Value{Value: x})
	case uint32:
		result, err = anypb.New(&wrapperspb.UInt32Value{Value: x})
	case uint64:
		result, err = anypb.New(&wrapperspb.UInt64Value{Value: x})
	case float32:
		result, err = anypb.New(&wrapperspb.FloatValue{Value: x})
	case float64:
		result, err = anypb.New(&wrapperspb.DoubleValue{Value: x})
	case protoreflect.EnumNumber:
		result, err = anypb.New(&wrapperspb.Int32Value{Value: int32(x)})
	case string:
		result, err = anypb.New(&wrapperspb.StringValue{Value: x})
	case []byte:
		result, err = anypb.New(&wrapperspb.BytesValue{Value: x})
	default:
		return nil, fmt.Errorf("cannot convert %T to anypb.Any", value)
	}
	if err != nil {
		return nil, fmt.Errorf("convert %T to anypb.Any: %w", value, err)
	}
	return result, nil
}
