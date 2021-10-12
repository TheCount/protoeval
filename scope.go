package protoeval

import (
	"errors"
	"fmt"
	"math"
	"reflect"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// scope describes an evaluation scope.
// Each scope instance is a node in a scope tree.
type scope struct {
	// current is the current evaluation scope.
	current protoreflect.Value

	// parent points to the parent scope. If current is the root scope,
	// parent is nil.
	parent *scope
}

// Init initialises this scope as a root scope for the specified message.
func (s *scope) Init(msg protoreflect.Message) {
	s.current = protoreflect.ValueOfMessage(msg)
	s.parent = nil
}

// ShiftByName returns a child scope of this scope by indexing this scope
// by name.
func (s *scope) ShiftByName(name string) (scope, error) {
	switch x := s.current.Interface().(type) {
	case protoreflect.Map:
		key := protoreflect.ValueOf(name).MapKey()
		var newValue protoreflect.Value
		var err error
		func() {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("unable to shift to map entry '%s': %v", name, r)
				}
				newValue = x.Get(key)
			}()
		}()
		if err != nil {
			return scope{}, err
		}
		if !newValue.IsValid() {
			return scope{}, fmt.Errorf("map entry '%s' does not exist", name)
		}
		return scope{
			current: newValue,
			parent:  s,
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
			current: x.Get(fd),
			parent:  s,
		}, nil
	default:
		return scope{}, fmt.Errorf("type %T cannot be indexed by name",
			s.current.Interface())
	}
}

// ShiftByIndex returns a child scope of this scope by indexing this scope.
func (s *scope) ShiftByIndex(index uint32) (scope, error) {
	list, ok := s.current.Interface().(protoreflect.List)
	if !ok {
		return scope{}, fmt.Errorf("type %T cannot be indexed",
			s.current.Interface())
	}
	intIndex := int(index)
	if index > math.MaxInt32 || intIndex >= list.Len() {
		// FIXME: once we have Go 1.17, we could use math.MaxInt
		return scope{}, fmt.Errorf("index %d out of bounds", index)
	}
	return scope{
		current: list.Get(intIndex),
		parent:  s,
	}, nil
}

// ShiftByBoolKey returns a child scope of this scope by using the specified
// key as map index.
func (s *scope) ShiftByBoolKey(key bool) (scope, error) {
	m, ok := s.current.Interface().(protoreflect.Map)
	if !ok {
		return scope{}, fmt.Errorf("type %T cannot be indexed by bool key",
			s.current.Interface())
	}
	var newvalue protoreflect.Value
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("index by bool key: %v", r)
			}
		}()
		newvalue = m.Get(protoreflect.ValueOf(key).MapKey())
	}()
	if err != nil {
		return scope{}, err
	}
	if !newvalue.IsValid() {
		return scope{}, fmt.Errorf("map entry '%t' does not exist", key)
	}
	return scope{
		current: newvalue,
		parent:  s,
	}, nil
}

// ShiftByUintKey returns a child scope of this scope by using the specified
// key as map index.
func (s *scope) ShiftByUintKey(key uint64) (scope, error) {
	m, ok := s.current.Interface().(protoreflect.Map)
	if !ok {
		return scope{}, fmt.Errorf("type %T cannot be indexed by uint key",
			s.current.Interface())
	}
	var newvalue protoreflect.Value
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("index by uint key: %v", r)
			}
		}()
		newvalue = m.Get(protoreflect.ValueOf(key).MapKey())
	}()
	if err != nil && key <= math.MaxUint32 {
		// Try again with uint32 key
		errmsg := err.Error()
		err = nil
		func() {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("index by uint key: %s/%v", errmsg, r)
				}
			}()
			newvalue = m.Get(protoreflect.ValueOf(uint32(key)).MapKey())
		}()
	}
	if err != nil {
		return scope{}, err
	}
	if !newvalue.IsValid() {
		return scope{}, fmt.Errorf("map entry %d does not exist", key)
	}
	return scope{
		current: newvalue,
		parent:  s,
	}, nil
}

// ShiftByIntKey returns a child scope of this scope by using the specified
// key as map index.
func (s *scope) ShiftByIntKey(key int64) (scope, error) {
	m, ok := s.current.Interface().(protoreflect.Map)
	if !ok {
		return scope{}, fmt.Errorf("type %T cannot be indexed by int key",
			s.current.Interface())
	}
	var newvalue protoreflect.Value
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("index by int key: %v", r)
			}
		}()
		newvalue = m.Get(protoreflect.ValueOf(key).MapKey())
	}()
	if err != nil && key >= math.MinInt32 && key <= math.MaxInt32 {
		// Try again with int32 key
		errmsg := err.Error()
		err = nil
		func() {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("index by int key: %s/%v", errmsg, r)
				}
			}()
			newvalue = m.Get(protoreflect.ValueOf(int32(key)).MapKey())
		}()
	}
	if err != nil {
		return scope{}, err
	}
	if !newvalue.IsValid() {
		return scope{}, fmt.Errorf("map entry %d does not exist", key)
	}
	return scope{
		current: newvalue,
		parent:  s,
	}, nil
}

// Value returns the value of this scope.
func (s *scope) Value() interface{} {
	return protoUnpackValue(s.current)
}

// Value returns the default value of this scope.
func (s *scope) DefaultValue() interface{} {
	// FIXME: returns proto3 default values only;
	// proto2 would require descriptor tracking.
	switch x := s.current.Interface().(type) {
	case bool:
		return false
	case int32:
		return int32(0)
	case int64:
		return int64(0)
	case uint32:
		return uint32(0)
	case uint64:
		return uint64(0)
	case float32:
		return float32(0)
	case float64:
		return float64(0)
	case string:
		return ""
	case []byte:
		return []byte{}
	case protoreflect.EnumNumber:
		return protoreflect.EnumNumber(0) // FIXME: should be EnumValueDescriptor
	case protoreflect.Message:
		return x.Type().New().Interface()
	case protoreflect.List:
		typ := reflect.TypeOf(protoUnpackValue(x.NewElement()))
		return reflect.MakeSlice(reflect.SliceOf(typ), 0, 0).Interface()
	case protoreflect.Map:
		keyType := getMapKeyType(x)
		valueType := reflect.TypeOf(protoUnpackValue(x.NewValue()))
		return reflect.MakeMap(reflect.MapOf(keyType, valueType)).Interface()
	default:
		panic(fmt.Sprintf(
			"BUG: unhandled proto value type %T", s.current.Interface()))
	}
}

// ShiftToParent returns a copy of the parent scope of this scope.
func (s *scope) ShiftToParent() (scope, error) {
	if s.parent == nil {
		return scope{}, errors.New("already at the root scope")
	}
	return *s.parent, nil
}
