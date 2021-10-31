package protoeval

import (
	"errors"
	"reflect"

	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
)

// celArgsListType carries type information for celArgList.
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

var _ traits.Lister = &celArgList{}

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

// ConvertToNative implements ref.Val.ConvertToNative for traits.Lister.
func (cal *celArgList) ConvertToNative(t reflect.Type) (interface{}, error) {
	result := cal.asRefValList()
	return types.NewRefValList(celTypeRegistry, result).ConvertToNative(t)
}

// ConvertToType implements ref.Val.ConvertToType for traits.Lister.
func (cal *celArgList) ConvertToType(t ref.Type) ref.Val {
	result := cal.asRefValList()
	return types.NewRefValList(celTypeRegistry, result).ConvertToType(t)
}

// Equal implements ref.Val.Equal for traits.Lister.
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

// Iterator implements traits.Iterable.Iterator for traits.Lister.
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

// Size implements traits.Sizer.Size for traits.Lister.
func (cal *celArgList) Size() ref.Val {
	return types.Int(cal.length())
}

// Type implements ref.Val.Type for traits.Lister.
func (*celArgList) Type() ref.Type {
	return CelArgListType
}

// Value implements ref.Val.Value for traits.Lister.
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

// celArgListIterator implements the traits.Iterator interface for celArgList.
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
