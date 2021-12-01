package protoeval

import (
	"reflect"

	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	// DefaultEvalMax is the default maximum number of sub-evaluations before
	// a call to Eval is aborted.
	DefaultEvalMax = 1000
)

// envValue describes an environment value.
type envValue struct {
	// origType is the original type of the value as supplied to the user.
	origType reflect.Type

	// value is the value in CEL format.
	value ref.Val
}

// Env describes an environment within which an evaluation can take place.
// Instances of this type are not safe for concurrent use. Clone your
// environment instead.
type Env struct {
	// values is the general value storage for this environment.
	values map[string]envValue

	// scope is the current scope.
	scope scope

	// cyclesLeft is the number of cycles (an evaluation cost measure) left
	// before we abort an evaluation.
	cyclesLeft int
}

// NewEnv creates a new, empty environment.
func NewEnv() *Env {
	initCel()
	return &Env{
		values:     make(map[string]envValue),
		cyclesLeft: DefaultEvalMax,
	}
}

// Set sets a value in this environment under the given key. If there already
// is a value, it is overwritten. If value is nil, it is deleted instead.
// If the value cannot be converted to a proper CEL value, an error is returned.
func (e *Env) Set(key string, value interface{}) error {
	if value == nil {
		delete(e.values, key)
		return nil
	}
	val := celTypeRegistry.NativeToValue(value)
	if types.IsError(val) {
		return val.Value().(error)
	}
	e.values[key] = envValue{
		origType: reflect.TypeOf(value),
		value:    val,
	}
	return nil
}

// Get gets a value from this environment for the given key. If no such value
// exists, ok == false is returned.
func (e *Env) Get(key string) (value interface{}, ok bool) {
	envValue, ok := e.values[key]
	if !ok {
		return nil, false
	}
	value, err := envValue.value.ConvertToNative(envValue.origType)
	if err != nil {
		// This should not happen since we obtained Val with NativeToValue.
		panic(err)
	}
	return value, true
}

// SetEvalMax sets the maximum number of sub-evaluations for an Eval call
// with this environment. A non-positive value will cause all evaluations to
// fail. This environment is returned.
func (e *Env) SetEvalMax(max int) *Env {
	e.cyclesLeft = max
	return e
}

// Clone creates a copy of this environment.
// Note that values set with Set or through previous evaluations are copied
// shallowly.
func (e *Env) Clone() *Env {
	result := &Env{
		values:     make(map[string]envValue, len(e.values)),
		cyclesLeft: e.cyclesLeft,
	}
	for k, v := range e.values {
		result.values[k] = v
	}
	return result
}

// shiftScope returns a shallow copy of this environment with the same scope.
func (e *Env) shiftScope(path *structpb.ListValue) (*Env, error) {
	newenv := *e
	var err error
	newenv.scope, err = e.scope.Shift(path)
	return &newenv, err
}

// shiftScopeToParent returns a shallow copy of this environment with the
// scope shifted to the parent scope.
func (e *Env) shiftScopeToParent() (*Env, error) {
	newenv := *e
	var err error
	newenv.scope, err = e.scope.ShiftToParent()
	if err != nil {
		return nil, err
	}
	return &newenv, nil
}
