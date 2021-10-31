package protoeval

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
)

// celEnvType carries type information for celEnv.
type celEnvType struct{}

var _ ref.Type = &celEnvType{}

// HasTrait implements ref.Type.HasTrait.
func (*celEnvType) HasTrait(tr int) bool {
	const availableTraits = traits.ContainerType |
		traits.IndexerType |
		traits.IterableType |
		traits.SizerType
	return tr&availableTraits != 0
}

// TypeName implements ref.Type.TypeName.
func (*celEnvType) TypeName() string {
	return "com.github.thecount.protoeval.env"
}

// CelEnvType is the CEL type for the protoeval environment.
var CelEnvType = &celEnvType{}

// celEnv uses Env to provide a traits.Mapper for the current environment.
type celEnv Env

var _ traits.Mapper = &celEnv{}

// asMapStringInterface returns this CEL environment as a
// map[string]interface{}. If a back conversion fails, an error is returned
// (should not happen if ConvertToNative reverses NativeToValue, as it should).
func (ce *celEnv) asMapStringInterface() (map[string]interface{}, error) {
	result := make(map[string]interface{}, len(ce.values))
	for k, v := range ce.values {
		origValue, err := v.value.ConvertToNative(v.origType)
		if err != nil {
			return nil, fmt.Errorf(
				"unable to convert key '%s' value to native type %s", k, v.origType)
		}
		result[k] = origValue
	}
	return result, nil
}

// Contains implements traits.Container.Contains for traits.Mapper.
func (ce *celEnv) Contains(value ref.Val) ref.Val {
	key, ok := value.Value().(string)
	if !ok {
		return types.False
	}
	_, ok = (*Env)(ce).values[key]
	return types.Bool(ok)
}

// ConvertToNative implements ref.Val.ConvertToNative for traits.Mapper.
func (ce *celEnv) ConvertToNative(t reflect.Type) (interface{}, error) {
	if t == reflect.TypeOf(map[string]interface{}(nil)) {
		return ce.asMapStringInterface()
	}
	return nil, fmt.Errorf("conversion to type %s not supported", t)
}

// ConvertToType implements ref.Val.ConvertToType for traits.Mapper.
func (ce *celEnv) ConvertToType(t ref.Type) ref.Val {
	result, err := ce.asMapStringInterface()
	if err != nil {
		return types.NewErr(err.Error())
	}
	return types.NewStringInterfaceMap(celTypeRegistry, result).ConvertToType(t)
}

// Equal implements ref.Val.Equal for traits.Mapper.
func (ce *celEnv) Equal(other ref.Val) ref.Val {
	if ce.Type() != other.Type() {
		return types.False
	}
	otherEnv := other.(*celEnv)
	if len(ce.values) != len(otherEnv.values) {
		return types.False
	}
	for k, v := range ce.values {
		otherV, ok := otherEnv.values[k]
		if !ok {
			return types.False
		}
		if v.origType != otherV.origType {
			return types.False
		}
		if v.value.Equal(otherV.value) != types.True {
			return types.False
		}
	}
	return types.True
}

// Find implements traits.Mapper.Find.
func (ce *celEnv) Find(key ref.Val) (ref.Val, bool) {
	k, ok := key.Value().(string)
	if !ok {
		return types.ValOrErr(key, "invalid key type %s",
			key.Type().TypeName()), false
	}
	v, ok := ce.values[k]
	if !ok {
		return nil, false
	}
	return v.value, true
}

// Get implements traits.Indexer.Get for traits.Mapper.
func (ce *celEnv) Get(key ref.Val) ref.Val {
	k, ok := key.Value().(string)
	if !ok {
		return types.ValOrErr(key, "invalid key type %s", key.Type().TypeName())
	}
	v, ok := ce.values[k]
	if !ok {
		return types.NewErr("key '%s' not found", k)
	}
	return v.value
}

// Iterator implements traits.Iterable.Iterator for traits.Mapper.
func (ce *celEnv) Iterator() traits.Iterator {
	keys := make([]string, 0, len(ce.values))
	for k := range ce.values {
		keys = append(keys, k)
	}
	return &celEnvIterator{
		keys:   keys,
		values: ce.values,
		pos:    0,
	}
}

// Size implements traits.Sizer.Size for traits.Mapper.
func (ce *celEnv) Size() ref.Val {
	return types.Int(len(ce.values))
}

// Type implements ref.Val.Type for traits.Mapper.
func (ce *celEnv) Type() ref.Type {
	return CelEnvType
}

// Value implements ref.Val.Value for traits.Mapper.
func (ce *celEnv) Value() interface{} {
	result, err := ce.asMapStringInterface()
	if err != nil {
		panic(err) // should not happen
	}
	return result
}

// celEnvIteratorType carries type information for celEnvIterator.
type celEnvIteratorType struct{}

var _ ref.Type = &celEnvIteratorType{}

// HasTrait implements ref.Type.HasTrait.
func (*celEnvIteratorType) HasTrait(tr int) bool {
	return tr&traits.IteratorType != 0
}

// TypeName implements ref.Type.TypeName.
func (*celEnvIteratorType) TypeName() string {
	return "com.github.thecount.protoeval.env.iterator"
}

// celEnvIteratorTypeInstance is the CEL type for an environment iterator.
var celEnvIteratorTypeInstance = &celEnvIteratorType{}

// celEnvIterator implements the traits.Iterator interface for celEnv.
type celEnvIterator struct {
	// keys is the list of environment keys, in unspecified order.
	keys []string

	// values maps keys to their respective values.
	values map[string]envValue

	// pos is the current iterator position into keys. Once the iteration has
	// finished, pos == len(keys).
	pos int
}

// ConvertToNative implements ref.Value.ConvertToNative for traits.Iterator.
func (*celEnvIterator) ConvertToNative(t reflect.Type) (
	interface{}, error,
) {
	return nil, errors.New("internal iterator type not convertible")
}

// ConvertToType implements ref.Value.ConvertToType for traits.Iterator.
func (*celEnvIterator) ConvertToType(t ref.Type) ref.Val {
	return types.NewErr("internal iterator type not convertible")
}

// Equal implements ref.Value.Equal for traits.Iterator.
func (iter *celEnvIterator) Equal(other ref.Val) ref.Val {
	if iter.Type() != other.Type() {
		return types.False
	}
	otherIter := other.(*celEnvIterator)
	return types.Bool(reflect.DeepEqual(iter.keys, otherIter.keys) &&
		iter.pos == otherIter.pos)
}

// HasNext implements traits.Iterator.HasNext.
func (iter *celEnvIterator) HasNext() ref.Val {
	return types.Bool(iter.pos < len(iter.keys))
}

// Next implements traits.Iterator.Next.
func (iter *celEnvIterator) Next() ref.Val {
	rv := iter.values[iter.keys[iter.pos]]
	iter.pos++
	return rv.value
}

// Type implements ref.Val.Type for traits.Iterator.
func (iter *celEnvIterator) Type() ref.Type {
	return celEnvIteratorTypeInstance
}

// Value implements ref.Val.Value for traits.Iterator.
func (iter *celEnvIterator) Value() interface{} {
	return iter
}
