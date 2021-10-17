package protoeval

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/pb"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// celTypes is the list of types we make available to all CEL programs.
// These are all protobuf message types linked into the binary, plus special
// types implementing cel's ref.Type interface.
var celTypes []interface{}

// initCelTypesOnce ensures the celTypes variable is initialised only once,
// via initCelTypes().
// We don't initialise celTypes via this package's init function because
// it might be called before all protobuf types are registered.
var initCelTypesOnce sync.Once

// initCelTypes ensures the celTypes variable is initialised.
func initCelTypes() {
	initCelTypesOnce.Do(func() {
		celTypes = []interface{}{(*celScope)(nil)}
		protoregistry.GlobalTypes.
			RangeMessages(func(mt protoreflect.MessageType) bool {
				if msg := mt.Zero(); msg != nil {
					celTypes = append(celTypes, msg.Interface())
				}
				return true
			})
	})
}

// celScope is the cel version of scope. It has all the methods needed to
// implement various cel interfaces.
type celScope scope

// HasTrait implements cel's ref.Type.HasTrait API.
func (*celScope) HasTrait(trait int) bool {
	return trait&traits.ReceiverType != 0
}

// TypeName implements cel's ref.Type.TypeName API.
func (*celScope) TypeName() string {
	return "com.github.thecount.protoeval.Scope"
}

// ConvertToNative implements cel's ref.Val.ConvertToNative API.
func (s *celScope) ConvertToNative(typeDesc reflect.Type) (interface{}, error) {
	thisType := reflect.TypeOf(celScope{})
	switch typeDesc {
	case thisType:
		return *s, nil
	case reflect.PtrTo(thisType):
		ns := *s
		return &ns, nil
	default:
		return nil, fmt.Errorf("cannot convert scope to %s", typeDesc)
	}
}

// ConvertToType implements cel's ref.Val.ConvertToType API.
func (s *celScope) ConvertToType(typeValue ref.Type) ref.Val {
	// FIXME: evaluate if type conversion from a scope can be useful in practice.
	return types.NewErr("scope conversion not supported")
}

// Equal implements cel's ref.Val.Equal API.
// Two scopes are equal if their descriptors and values match.
func (s *celScope) Equal(other ref.Val) ref.Val {
	otherScope, ok := other.Value().(*celScope)
	if !ok {
		return types.False
	}
	msg1, ok1 := s.value.Interface().(protoreflect.Message)
	msg2, ok2 := otherScope.value.Interface().(protoreflect.Message)
	if (ok1 && !ok2) || (!ok1 && ok2) {
		return types.False
	}
	if ok1 {
		if !ok2 {
			return types.False
		}
		if msg1.Descriptor().FullName() != msg2.Descriptor().FullName() {
			return types.False
		}
		return types.Bool(proto.Equal(msg1.Interface(), msg2.Interface()))
	}
	if ok2 {
		return types.False
	}
	if s.desc.FullName() != otherScope.desc.FullName() {
		return types.False
	}
	return types.Bool(protoEqual(s.value.Interface(),
		otherScope.value.Interface()))
}

// Type implements cel's ref.Val.Type API.
func (*celScope) Type() ref.Type {
	return (*celScope)(nil)
}

// Value implements cel's ref.Val.Value API.
func (s *celScope) Value() interface{} {
	return (*scope)(s)
}

// Receive implements cel's traits.Receiver.Receive API.
func (s *celScope) Receive(name string, op string, args []ref.Val) ref.Val {
	if op != "" {
		return types.NewErr("operator '%s' not supported", op)
	}
	switch name {
	case "value":
		return proto2cel(s.desc, s.value)
	case "parent":
		if s.parent == nil {
			return types.NewErr("request for parent in root scope")
		}
		return (*celScope)(s.parent)
	default:
		return types.NewErr("invalid method: %s", name)
	}
}

// proto2cel returns the CEL value for the specified protobuf value.
// fd must be non/nil unless the value represents a protobuf message.
func proto2cel(fd protoreflect.FieldDescriptor, pv protoreflect.Value) ref.Val {
	switch x := pv.Interface().(type) {
	case protoreflect.List:
		return types.NewProtoList(types.DefaultTypeAdapter, x)
	case protoreflect.Map:
		return types.NewProtoMap(types.DefaultTypeAdapter, &pb.Map{
			Map:       x,
			KeyType:   pb.NewFieldDescription(fd.MapKey()),
			ValueType: pb.NewFieldDescription(fd.MapValue()),
		})
	default:
		return types.DefaultTypeAdapter.NativeToValue(pv.Interface())
	}
}

// cel2go converts the given cel value to a go value.
func cel2go(v ref.Val) (interface{}, error) {
	switch x := v.Value().(type) {
	case bool, []byte, float64, time.Duration, int64, string, time.Time, uint64,
		proto.Message:
		return v.Value(), nil
	case protoreflect.List:
		// FIXME: cannot convert, descriptor missing
		return nil, errors.New("cannot convert CEL protoreflect.List to native Go")
	case protoreflect.Map:
		// FIXME: cannot convert, descriptor missing
		return nil, errors.New("cannot convert CEL protoreflect.Map to native Go")
	case protoreflect.Message:
		return x.Interface(), nil
	default:
		return nil, fmt.Errorf("cel2go: unable to handle type %T", v.Value())
	}
}
