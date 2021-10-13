package protoeval

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"time"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// scope describes an evaluation scope.
// Each scope instance is a node in a scope tree.
type scope struct {
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
func (s *scope) Value() interface{} {
	return protoUnpackValue(s.desc, s.value)
}

// Value returns the default value of this scope.
func (s *scope) DefaultValue() interface{} {
	switch x := s.value.Interface().(type) {
	case protoreflect.EnumNumber:
		return s.desc.DefaultEnumValue()
	case protoreflect.Message:
		switch x.Descriptor().FullName() {
		case durationName:
			return time.Duration(0)
		case timestampName:
			return time.Time{}
		default:
			return x.Type().New().Interface()
		}
	case protoreflect.List:
		typ := reflect.TypeOf(protoUnpackValue(s.desc, x.NewElement()))
		return reflect.MakeSlice(reflect.SliceOf(typ), 0, 0).Interface()
	case protoreflect.Map:
		keyType, _ := getProtoMapKeyType(Value_Kind(s.desc.MapKey().Kind()))
		valueType := reflect.TypeOf(protoUnpackValue(s.desc, x.NewValue()))
		return reflect.MakeMap(reflect.MapOf(keyType, valueType)).Interface()
	default:
		return s.desc.Default().Interface()
	}
}

// ShiftToParent returns a copy of the parent scope of this scope.
func (s *scope) ShiftToParent() (scope, error) {
	if s.parent == nil {
		return scope{}, errors.New("already at the root scope")
	}
	return *s.parent, nil
}
