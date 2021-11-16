package protoeval

import (
	"errors"
	"fmt"
	"math"
	"strconv"

	"github.com/google/cel-go/common/types/pb"
	"github.com/google/cel-go/common/types/ref"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

// scope describes an evaluation scope.
// Each scope instance is a node in a scope tree.
type scope struct {
	// args is the current argument stack. From this scope's perspective,
	// if args is non-empty, argument 0 is the last element of args.
	args []ref.Val

	// desc is the field descriptor leading up to the scope value.
	// For the root scope, desc is nil.
	desc protoreflect.FieldDescriptor

	// value is the protobuf value of this evaluation scope.
	value protoreflect.Value

	// parent points to the parent scope. If this is the root scope,
	// parent is nil.
	parent *scope
}

// Init initialises this scope as a root scope for the specified message.
func (s *scope) Init(msg protoreflect.Message) {
	s.desc = nil
	s.value = protoreflect.ValueOfMessage(msg)
	s.parent = nil
}

// scope returns a child scope of this scope based on the given scope
// selection path, see Value.Scope.
func (s *scope) Shift(path *structpb.ListValue) (scope, error) {
	if len(path.Values) == 0 {
		return scope{
			desc:   s.desc,
			value:  s.value,
			parent: s,
		}, nil
	}
	var err error
	for i, step := range path.Values {
		s, err = s.shiftStep(step)
		if err != nil {
			return scope{}, fmt.Errorf("shift path index %d: %w", i, err)
		}
	}
	return *s, nil
}

// shiftStep shifts this scope by one step.
func (s *scope) shiftStep(step *structpb.Value) (*scope, error) {
	switch x := step.Kind.(type) {
	case *structpb.Value_StringValue:
		switch y := s.value.Interface().(type) {
		case protoreflect.Message:
			desc := y.Descriptor()
			if desc.FullName() == anypbName {
				msg, err := y.Interface().(*anypb.Any).UnmarshalNew()
				if err != nil {
					return nil, fmt.Errorf("unwrap Any: %w", err)
				}
				y = msg.ProtoReflect()
				desc = y.Descriptor()
			}
			fd := desc.Fields().ByName(protoreflect.Name(x.StringValue))
			if fd == nil {
				return nil, fmt.Errorf("no such message field: %s", x.StringValue)
			}
			if fd.HasPresence() && !y.Has(fd) {
				return nil, fmt.Errorf("message field %s not set", x.StringValue)
			}
			return &scope{
				desc:   fd,
				value:  y.Get(fd),
				parent: s,
			}, nil
		case protoreflect.Map:
			var key protoreflect.MapKey
			switch s.desc.MapKey().Kind() {
			case protoreflect.StringKind:
				key = protoreflect.ValueOfString(x.StringValue).MapKey()
			case protoreflect.Int32Kind, protoreflect.Sint32Kind,
				protoreflect.Sfixed32Kind:
				n, err := strconv.ParseInt(x.StringValue, 0, 32)
				if err != nil {
					return nil, fmt.Errorf("map key '%s' invalid for map key kind %s",
						x.StringValue, s.desc.MapKey().Kind())
				}
				key = protoreflect.ValueOfInt32(int32(n)).MapKey()
			case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
				n, err := strconv.ParseUint(x.StringValue, 0, 32)
				if err != nil {
					return nil, fmt.Errorf("map key '%s' invalid for map key kind %s",
						x.StringValue, s.desc.MapKey().Kind())
				}
				key = protoreflect.ValueOfUint32(uint32(n)).MapKey()
			case protoreflect.Int64Kind, protoreflect.Sint64Kind,
				protoreflect.Sfixed64Kind:
				n, err := strconv.ParseInt(x.StringValue, 0, 64)
				if err != nil {
					return nil, fmt.Errorf("map key '%s' invalid for map key kind %s",
						x.StringValue, s.desc.MapKey().Kind())
				}
				key = protoreflect.ValueOfInt64(n).MapKey()
			case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
				n, err := strconv.ParseUint(x.StringValue, 0, 64)
				if err != nil {
					return nil, fmt.Errorf("map key '%s' invalid for map key kind %s",
						x.StringValue, s.desc.MapKey().Kind())
				}
				key = protoreflect.ValueOfUint64(n).MapKey()
			case protoreflect.BoolKind:
				b, err := strconv.ParseBool(x.StringValue)
				if err != nil {
					return nil, fmt.Errorf("map key '%s' invalid for map key kind %s",
						x.StringValue, s.desc.MapKey().Kind())
				}
				key = protoreflect.ValueOfBool(b).MapKey()
			default:
				panic(fmt.Sprintf("BUG: unsupported map key kind %s",
					s.desc.MapKey().Kind()))
			}
			if !y.Has(key) {
				return nil, fmt.Errorf("map has no key '%s'", x.StringValue)
			}
			return &scope{
				desc:   s.desc,
				value:  y.Get(key),
				parent: s,
			}, nil
		case protoreflect.List:
			return nil, errors.New("cannot index list with string")
		default:
			return nil, fmt.Errorf("cannot index %s with string", s.desc.Kind())
		}
	case *structpb.Value_NumberValue:
		switch y := s.value.Interface().(type) {
		case protoreflect.Message:
			fn := protoreflect.FieldNumber(x.NumberValue)
			test := float64(fn)
			if test != x.NumberValue {
				return nil, fmt.Errorf("invalid field number: %f", x.NumberValue)
			}
			desc := y.Descriptor()
			if desc.FullName() == anypbName {
				msg, err := y.Interface().(*anypb.Any).UnmarshalNew()
				if err != nil {
					return nil, fmt.Errorf("unwrap Any: %w", err)
				}
				y = msg.ProtoReflect()
				desc = y.Descriptor()
			}
			fd := desc.Fields().ByNumber(fn)
			if fd == nil {
				return nil, fmt.Errorf("no such message field number: %d", fn)
			}
			if fd.HasPresence() && !y.Has(fd) {
				return nil, fmt.Errorf("message field %s (%d) not set", fd.Name(), fn)
			}
			return &scope{
				desc:   fd,
				value:  y.Get(fd),
				parent: s,
			}, nil
		case protoreflect.Map:
			var key protoreflect.MapKey
			switch s.desc.MapKey().Kind() {
			case protoreflect.StringKind:
				return nil, errors.New("cannot index string key map with number")
			case protoreflect.Int32Kind, protoreflect.Sint32Kind,
				protoreflect.Sfixed32Kind:
				n := int32(x.NumberValue)
				test := float64(n)
				if test != x.NumberValue {
					return nil, fmt.Errorf("cannot convert %f to int32", x.NumberValue)
				}
				key = protoreflect.ValueOfInt32(n).MapKey()
			case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
				n := uint32(x.NumberValue)
				test := float64(n)
				if test != x.NumberValue {
					return nil, fmt.Errorf("cannot convert %f to uint32", x.NumberValue)
				}
				key = protoreflect.ValueOfUint32(n).MapKey()
			case protoreflect.Int64Kind, protoreflect.Sint64Kind,
				protoreflect.Sfixed64Kind:
				n := int64(x.NumberValue)
				test := float64(n)
				if test != x.NumberValue {
					return nil, fmt.Errorf("cannot convert %f to int64", x.NumberValue)
				}
				key = protoreflect.ValueOfInt64(n).MapKey()
			case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
				n := uint64(x.NumberValue)
				test := float64(n)
				if test != x.NumberValue {
					return nil, fmt.Errorf("cannot convert %f to uint64", x.NumberValue)
				}
				key = protoreflect.ValueOfUint64(n).MapKey()
			case protoreflect.BoolKind:
				return nil, errors.New("cannot index bool key map with number")
			default:
				panic(fmt.Sprintf("BUG: unsupported map key kind %s",
					s.desc.MapKey().Kind()))
			}
			if !y.Has(key) {
				return nil, fmt.Errorf("map has no key %d", key.Interface())
			}
			return &scope{
				desc:   s.desc,
				value:  y.Get(key),
				parent: s,
			}, nil
		case protoreflect.List:
			idx := int(x.NumberValue)
			test := float64(idx)
			if test != x.NumberValue {
				return nil, fmt.Errorf("cannot convert %f to list index", x.NumberValue)
			}
			if idx < 0 || idx >= y.Len() {
				return nil, fmt.Errorf("list index %d out of bounds", idx)
			}
			return &scope{
				desc:   s.desc,
				value:  y.Get(idx),
				parent: s,
			}, nil
		default:
			return nil, fmt.Errorf("cannot index %s with number", s.desc.Kind())
		}
	case *structpb.Value_BoolValue:
		switch y := s.value.Interface().(type) {
		case protoreflect.Message:
			return nil, errors.New("cannot index message with bool")
		case protoreflect.Map:
			if s.desc.MapKey().Kind() != protoreflect.BoolKind {
				return nil, fmt.Errorf("cannot index %s with bool",
					s.desc.MapKey().Kind())
			}
			key := protoreflect.ValueOfBool(x.BoolValue).MapKey()
			if !y.Has(key) {
				return nil, fmt.Errorf("map has no key '%t'", x.BoolValue)
			}
			return &scope{
				desc:   s.desc,
				value:  y.Get(key),
				parent: s,
			}, nil
		case protoreflect.List:
			return nil, errors.New("cannot index list with bool")
		default:
			return nil, fmt.Errorf("cannot index %s with bool", s.desc.Kind())
		}
	default:
		return nil, fmt.Errorf("scope step kind %T not supported", step.Kind)
	}
}

// Value returns the value of this scope.
func (s *scope) Value() ref.Val {
	switch x := s.value.Interface().(type) {
	case protoreflect.Map:
		// CEL cannot deal with protoreflect.Map directly, need to wrap it into
		// a pb.Map.
		fd := pb.NewFieldDescription(s.desc)
		return celTypeRegistry.NativeToValue(&pb.Map{
			Map:       x,
			KeyType:   fd.KeyType,
			ValueType: fd.ValueType,
		})
	default:
		return celTypeRegistry.NativeToValue(s.value.Interface())
	}
}

// Value returns the default value of this scope.
func (s *scope) DefaultValue() ref.Val {
	switch x := s.value.Interface().(type) {
	case protoreflect.Message: // s.parent and s.desc may be nil in this case
		return celTypeRegistry.NativeToValue(x.Type().New())
	case protoreflect.List:
		return celTypeRegistry.NativeToValue(
			s.parent.value.Message().NewField(s.desc).Interface())
	case protoreflect.Map:
		// CEL cannot deal with protoreflect.Map directly, need to wrap it into
		// a pb.Map.
		fd := pb.NewFieldDescription(s.desc)
		return celTypeRegistry.NativeToValue(&pb.Map{
			Map:       s.parent.value.Message().NewField(s.desc).Map(),
			KeyType:   fd.KeyType,
			ValueType: fd.ValueType,
		})
	default:
		return celTypeRegistry.NativeToValue(s.desc.Default().Interface())
	}
}

// ShiftToParent returns a copy of the parent scope of this scope.
func (s *scope) ShiftToParent() (scope, error) {
	if s.parent == nil {
		return scope{}, errors.New("already at the root scope")
	}
	return *s.parent, nil
}

// PushArg pushes the given argument onto the argument stack of this scope.
func (s *scope) PushArg(arg ref.Val) {
	s.args = append(s.args, arg)
}

// DropArgs drops the given number of arguments from the argument stack.
func (s *scope) DropArgs(n uint32) error {
	if n > math.MaxInt32 {
		return errors.New("excess number of arguments to drop")
	}
	return s.dropArgs(int(n))
}

// dropArgs drops the given number of arguments from the argument stack.
func (s *scope) dropArgs(n int) error {
	if s == nil {
		return fmt.Errorf("%d arguments left to drop but stack empty", n)
	}
	if n > len(s.args) {
		if err := s.parent.dropArgs(n - len(s.args)); err != nil {
			return err
		}
		s.args = nil
		return nil
	}
	s.args = s.args[:len(s.args)-n]
	return nil
}

// Arg returns the n-th argument.
func (s *scope) Arg(n uint32) (ref.Val, error) {
	if n > math.MaxInt32 {
		return nil, errors.New("argument number out of bounds")
	}
	if a, ok := s.arg(int(n)); ok {
		return a, nil
	}
	return nil, fmt.Errorf("no such arg: %d", n)
}

// arg returns the n-th argument if it exists.
func (s *scope) arg(n int) (ref.Val, bool) {
	if s == nil {
		return nil, false
	}
	if n >= len(s.args) {
		return s.parent.arg(n - len(s.args))
	}
	return s.args[len(s.args)-n-1], true
}
