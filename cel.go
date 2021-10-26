package protoeval

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"sync"

	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
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

// celArgsList carries type information for celArgList
type celArgListType struct{}

var _ ref.Type = &celArgListType{}

// HasTrait implements ref.Type.HasTrait.
func (*celArgListType) HasTrait(tr int) bool {
	const availableTraits = traits.AdderType |
		traits.ContainerType |
		traits.IndexerType |
		traits.IterableType |
		traits.SizerType
	return tr&availableTraits != 0
}

// TypeName implements ref.Type.TypeName.
func (*celArgListType) TypeName() string {
	return "com.github.thecount.protoeval.arg_list"
}

// CelArgListType is the CEL type for an argument list.
var CelArgListType = &celArgListType{}

// celArgList uses a scope to provide a traits.Lister for the current scope
// arguments.
type celArgList scope

var _ ref.Val = &celArgList{}

// Add implements traits.Adder.Add for traits.Lister.
func (cal *celArgList) Add(ref.Val) ref.Val {
	return types.NewErr("argument lists are immutable within a CEL program")
}

// asRefValList returns this CEL arguments list as ref.Val list.
func (cal *celArgList) asRefValList() []ref.Val {
	if cal == nil {
		return nil
	}
	result := make([]ref.Val, 0, len(cal.args))
	for i := len(cal.args) - 1; i >= 0; i-- {
		result = append(result, cal.args[i])
	}
	tail := (*celArgList)(cal.parent).asRefValList()
	return append(result, tail...)
}

// Contains implements traits.Container.Contains for traits.Lister.
func (cal *celArgList) Contains(value ref.Val) ref.Val {
	for iter := cal.Iterator(); iter.HasNext() == types.True; {
		if iter.Next().Equal(value) == types.True {
			return types.True
		}
	}
	return types.False
}

// ConvertToNative implements ref.Val.ConvertToNative.
func (cal *celArgList) ConvertToNative(t reflect.Type) (interface{}, error) {
	result := cal.asRefValList()
	return types.NewRefValList(celTypeRegistry, result).ConvertToNative(t)
}

// ConvertToType implements ref.Val.ConvertToType
func (cal *celArgList) ConvertToType(t ref.Type) ref.Val {
	result := cal.asRefValList()
	return types.NewRefValList(celTypeRegistry, result).ConvertToType(t)
}

// Equal implements ref.Val.Equal.
func (cal *celArgList) Equal(other ref.Val) ref.Val {
	if cal.Type() != other.Type() {
		return types.False
	}
	otherList := other.(*celArgList)
	length := otherList.length()
	if cal.length() != length {
		return types.False
	}
	for i := 0; i != length; i++ {
		idx := types.Int(i)
		if cal.Get(idx).Equal(otherList.Get(idx)) != types.True {
			return types.False
		}
	}
	return types.True
}

// Get implements traits.Indexer.Get for traits.Lister.
func (cal *celArgList) Get(index ref.Val) ref.Val {
	indexValue := reflect.ValueOf(index.Value())
	intType := reflect.TypeOf(int(0))
	if !indexValue.Type().ConvertibleTo(intType) {
		return types.NewErr("type %T not convertible to integer index", index)
	}
	idx := indexValue.Convert(intType).Interface().(int)
	rv, ok := (*scope)(cal).arg(idx)
	if !ok {
		return types.NewErr("index %d out of bounds", idx)
	}
	return rv
}

// Iterator implements traits.Iterable for traits.Lister.
func (cal *celArgList) Iterator() traits.Iterator {
	return &celArgListIterator{
		scope: (*scope)(cal),
		pos:   len(cal.args) - 1,
	}
}

// length returns the length of this argument list.
func (cal *celArgList) length() int {
	if cal == nil {
		return 0
	}
	return len(cal.args) + (*celArgList)(cal.parent).length()
}

// Type implements ref.Val.Type
func (*celArgList) Type() ref.Type {
	return CelArgListType
}

// Value implements ref.Val.Value
func (cal *celArgList) Value() interface{} {
	return cal.asRefValList()
}

// celArgListIteratorType carries type information for celArgListIterator.
type celArgListIteratorType struct{}

var _ ref.Type = &celArgListIteratorType{}

// HasTrait implements ref.Type.HasTrait.
func (*celArgListIteratorType) HasTrait(tr int) bool {
	return tr&traits.IteratorType != 0
}

// TypeName implements ref.Type.TypeName.
func (*celArgListIteratorType) TypeName() string {
	return "com.github.thecount.protoeval.arg_list.iterator"
}

// celArgListIteratorTypeInstance is the CEL type for an argument list iterator.
var celArgListIteratorTypeInstance = &celArgListIteratorType{}

// celArgListIterator implements the traits.Iterator for celArgList.
type celArgListIterator struct {
	// scope is the current iterator scope.
	scope *scope

	// pos is the position of the next argument. If -1, scope is exhausted.
	pos int
}

// ConvertToNative implements ref.Value.ConvertToNative for traits.Iterator.
func (*celArgListIterator) ConvertToNative(t reflect.Type) (
	interface{}, error,
) {
	return nil, errors.New("internal iterator type not convertible")
}

// ConvertToType implements ref.Value.ConvertToType for traits.Iterator.
func (*celArgListIterator) ConvertToType(t ref.Type) ref.Val {
	return types.NewErr("internal iterator type not convertible")
}

// Equal implements ref.Value.Equal for traits.Iterator.
func (iter *celArgListIterator) Equal(other ref.Val) ref.Val {
	if iter.Type() != other.Type() {
		return types.False
	}
	otherIter := other.(*celArgListIterator)
	return types.Bool(*iter == *otherIter)
}

// HasNext implements traits.Iterator.HasNext.
func (iter *celArgListIterator) HasNext() ref.Val {
	if iter.pos >= 0 {
		return types.True
	}
	if iter.scope.parent == nil {
		return types.False
	}
	iter.scope = iter.scope.parent
	iter.pos = len(iter.scope.args) - 1
	return iter.HasNext()
}

// Next implements traits.Iterator.Next.
func (iter *celArgListIterator) Next() ref.Val {
	rv := iter.scope.args[iter.pos]
	iter.pos--
	return rv
}

// Type implements ref.Val.Type for traits.Iterator.
func (iter *celArgListIterator) Type() ref.Type {
	return celArgListIteratorTypeInstance
}

// Value implements ref.Val.Value for traits.Iterator.
func (iter *celArgListIterator) Value() interface{} {
	return iter
}
