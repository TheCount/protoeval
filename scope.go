package protoeval

import (
	"errors"
	"fmt"
	"math"

	"github.com/google/cel-go/common/types/ref"
	"google.golang.org/protobuf/reflect/protoreflect"
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

// Matches reports whether the given name matches this scope.
// This is the case if and only if at least one of the following applies:
//
// The current scope is a field with the given full protobuf name.
//
// The current scope is a message with the given full protobuf type name.
//
// The current scope is an enum value with the given full protobuf name,
// or the given name is the type name of the enum the enum value is from.
func (s *scope) Matches(name protoreflect.FullName) bool {
	if s.desc != nil && s.desc.FullName() == name {
		return true
	}
	switch x := s.value.Interface().(type) {
	case protoreflect.Message:
		return x.Descriptor().FullName() == name
	case protoreflect.Map:
		return s.desc.Message().FullName() == name
	case protoreflect.EnumNumber:
		enumDesc := s.desc.Enum()
		if enumDesc.FullName() == name {
			return true
		}
		if enumDesc.Values().ByNumber(x).FullName() == name {
			return true
		}
		return false
	default:
		return false
	}
}

// Has reports whether the value of this scope is either:
//
// A message and has a set field with name str. Fields without presence
// are always considered set.
//
// A map with key type string, with a value under the key str.
func (s *scope) Has(str string) bool {
	switch x := s.value.Interface().(type) {
	case protoreflect.Message:
		name := protoreflect.Name(str)
		md := x.Descriptor()
		fd := md.Fields().ByName(name)
		if fd == nil {
			ood := md.Oneofs().ByName(name)
			if ood == nil {
				return false
			}
			return x.WhichOneof(ood) != nil
		}
		if !fd.HasPresence() {
			return true
		}
		return x.Has(fd)
	case protoreflect.Map:
		if s.desc.MapKey().Kind() != protoreflect.StringKind {
			return false
		}
		key := protoreflect.ValueOfString(str).MapKey()
		return x.Has(key)
	default:
		return false
	}
}

// scope returns a child scope of this scope with identical content.
func (s *scope) Shift() scope {
	return scope{
		desc:   s.desc,
		value:  s.value,
		parent: s,
	}
}

// ShiftByName returns a child scope of this scope by indexing this scope
// by name.
func (s *scope) ShiftByName(name string) (scope, error) {
	switch x := s.value.Interface().(type) {
	case protoreflect.Map:
		if s.desc.MapKey().Kind() != protoreflect.StringKind {
			return scope{}, errors.New("map key type must be string")
		}
		key := protoreflect.ValueOf(name).MapKey()
		newValue := x.Get(key)
		if !newValue.IsValid() {
			return scope{}, fmt.Errorf("map entry '%s' does not exist", name)
		}
		return scope{
			desc:   s.desc.MapValue(),
			value:  newValue,
			parent: s,
		}, nil
	case protoreflect.Message:
		fd := x.Descriptor().Fields().ByName(protoreflect.Name(name))
		if fd == nil {
			return scope{}, fmt.Errorf("message has no field '%s'", name)
		}
		if fd.HasPresence() && !x.Has(fd) {
			return scope{}, fmt.Errorf("message field '%s' not set", name)
		}
		return scope{
			desc:   fd,
			value:  x.Get(fd),
			parent: s,
		}, nil
	default:
		return scope{}, fmt.Errorf("type %T cannot be indexed by name",
			s.value.Interface())
	}
}

// ShiftByIndex returns a child scope of this scope by indexing this scope.
func (s *scope) ShiftByIndex(index uint32) (scope, error) {
	list, ok := s.value.Interface().(protoreflect.List)
	if !ok {
		return scope{}, fmt.Errorf("type %T cannot be indexed",
			s.value.Interface())
	}
	intIndex := int(index)
	if index > math.MaxInt32 || intIndex >= list.Len() {
		// FIXME: once we have Go 1.17, we could use math.MaxInt
		return scope{}, fmt.Errorf("index %d out of bounds", index)
	}
	return scope{
		desc:   s.desc,
		value:  list.Get(intIndex),
		parent: s,
	}, nil
}

// ShiftByBoolKey returns a child scope of this scope by using the specified
// key as map index.
func (s *scope) ShiftByBoolKey(key bool) (scope, error) {
	m, ok := s.value.Interface().(protoreflect.Map)
	if !ok {
		return scope{}, fmt.Errorf("type %T cannot be indexed by bool key",
			s.value.Interface())
	}
	if s.desc.MapKey().Kind() != protoreflect.BoolKind {
		return scope{}, fmt.Errorf("map key type: expected bool, got %s",
			s.desc.MapKey().Kind())
	}
	newvalue := m.Get(protoreflect.ValueOf(key).MapKey())
	if !newvalue.IsValid() {
		return scope{}, fmt.Errorf("map entry '%t' does not exist", key)
	}
	return scope{
		desc:   s.desc.MapValue(),
		value:  newvalue,
		parent: s,
	}, nil
}

// ShiftByUintKey returns a child scope of this scope by using the specified
// key as map index.
func (s *scope) ShiftByUintKey(key uint64) (scope, error) {
	m, ok := s.value.Interface().(protoreflect.Map)
	if !ok {
		return scope{}, fmt.Errorf("type %T cannot be indexed by uint key",
			s.value.Interface())
	}
	var keyValue protoreflect.MapKey
	switch s.desc.MapKey().Kind() {
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		keyValue = protoreflect.ValueOfUint64(key).MapKey()
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		if key > math.MaxUint32 {
			return scope{}, fmt.Errorf("key %d out of bounds", key)
		}
		keyValue = protoreflect.ValueOfUint32(uint32(key)).MapKey()
	default:
		return scope{}, fmt.Errorf("map key type: expected uint, got %s",
			s.desc.MapKey().Kind())
	}
	newvalue := m.Get(keyValue)
	if !newvalue.IsValid() {
		return scope{}, fmt.Errorf("map entry %d does not exist", key)
	}
	return scope{
		desc:   s.desc.MapValue(),
		value:  newvalue,
		parent: s,
	}, nil
}

// ShiftByIntKey returns a child scope of this scope by using the specified
// key as map index.
func (s *scope) ShiftByIntKey(key int64) (scope, error) {
	m, ok := s.value.Interface().(protoreflect.Map)
	if !ok {
		return scope{}, fmt.Errorf("type %T cannot be indexed by int key",
			s.value.Interface())
	}
	var keyValue protoreflect.MapKey
	switch s.desc.MapKey().Kind() {
	case protoreflect.Int64Kind, protoreflect.Sint64Kind,
		protoreflect.Sfixed64Kind:
		keyValue = protoreflect.ValueOfInt64(key).MapKey()
	case protoreflect.Int32Kind, protoreflect.Sint32Kind,
		protoreflect.Sfixed32Kind:
		if key < math.MinInt32 || key > math.MaxInt32 {
			return scope{}, fmt.Errorf("key %d out of bounds", key)
		}
		keyValue = protoreflect.ValueOfInt32(int32(key)).MapKey()
	}
	newvalue := m.Get(keyValue)
	if !newvalue.IsValid() {
		return scope{}, fmt.Errorf("map entry %d does not exist", key)
	}
	return scope{
		desc:   s.desc.MapValue(),
		value:  newvalue,
		parent: s,
	}, nil
}

// Value returns the value of this scope.
func (s *scope) Value() ref.Val {
	return celTypeRegistry.NativeToValue(s.value.Interface())
}

// Value returns the default value of this scope.
func (s *scope) DefaultValue() ref.Val {
	switch x := s.value.Interface().(type) {
	case protoreflect.Message: // s.parent and s.desc may be nil in this case
		return celTypeRegistry.NativeToValue(x.Type().New())
	case protoreflect.List, protoreflect.Map:
		return celTypeRegistry.NativeToValue(
			s.parent.value.Message().NewField(s.desc).Interface())
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
