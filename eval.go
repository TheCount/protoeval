package protoeval

import (
	"errors"
	"fmt"
	reflect "reflect"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// Errors
var (
	// ErrEvalTooLong is returned when an evaluation is too complex.
	ErrEvalTooLong = errors.New("evaluation took too long")
)

// errBreak is a special error type to model a break statement. Its value is
// the number of while statements to still break out of.
type errBreak uint32

// Error implements error.Error.
func (e errBreak) Error() string {
	return fmt.Sprintf("break(%d)", e)
}

// Is ensures all errBreak instances are considered equivalent by the errors
// package.
func (errBreak) Is(err error) bool {
	_, ok := err.(errBreak)
	return ok
}

// errContinue is a special error type to model a continue statement. Its value
// is the number of the while statement (counted from innermost to outermost) to
// continue.
type errContinue uint32

// Error implements error.Error.
func (e errContinue) Error() string {
	return fmt.Sprintf("continue(%d)", e)
}

// Is ensures all errContinue instances are considered equivalent by the errors
// package.
func (errContinue) Is(err error) bool {
	_, ok := err.(errContinue)
	return ok
}

// Eval evaluates the given message within the given environment according
// to the specified value.
func Eval(env *Env, msg proto.Message, value *Value) (interface{}, error) {
	if env == nil {
		return nil, errors.New("env is nil")
	}
	if msg == nil {
		return nil, errors.New("msg is nil")
	}
	if value == nil {
		return nil, errors.New("value is nil")
	}
	rmsg := msg.ProtoReflect()
	env.scope.Init(rmsg)
	cyclesLeft := env.cyclesLeft
	return eval(env, &cyclesLeft, value)
}

// eval recursively evaluates msg in the given environment based on value.
func eval(env *Env, cyclesLeft *int, value *Value) (interface{}, error) {
	if *cyclesLeft <= 0 {
		return nil, ErrEvalTooLong
	}
	*cyclesLeft--
	// shift scope
	var err error
	switch x := value.Scope.(type) {
	case nil:
		// Scope remains unchanged
	case *Value_Name:
		env, err = env.shiftScopeByName(x.Name)
	case *Value_Index:
		env, err = env.shiftScopeByIndex(x.Index)
	case *Value_BoolKey:
		env, err = env.shiftScopeByBoolKey(x.BoolKey)
	case *Value_UintKey:
		env, err = env.shiftScopeByUintKey(x.UintKey)
	case *Value_IntKey:
		env, err = env.shiftScopeByIntKey(x.IntKey)
	default:
		panic(fmt.Sprintf("BUG: unsupported scope type %T", value.Scope))
	}
	if err != nil {
		return nil, err
	}
	// evaluate
	switch x := value.Value.(type) {
	case nil:
		return env.scope.Value(), nil
	case *Value_This:
		return eval(env, cyclesLeft, x.This)
	case *Value_Parent:
		env, err = env.shiftScopeToParent()
		if err != nil {
			return nil, err
		}
		return eval(env, cyclesLeft, x.Parent)
	case *Value_Default:
		return env.scope.DefaultValue(), nil
	case *Value_Nil:
		return nil, nil
	case *Value_Bool:
		return x.Bool, nil
	case *Value_Int:
		return x.Int, nil
	case *Value_Uint:
		return x.Uint, nil
	case *Value_Double:
		return x.Double, nil
	case *Value_String_:
		return x.String_, nil
	case *Value_Bytes:
		return x.Bytes, nil
	case *Value_Enum_:
		typeName := protoreflect.FullName(x.Enum.Type)
		et, err := protoregistry.GlobalTypes.FindEnumByName(typeName)
		if err != nil {
			return nil, fmt.Errorf("find enum '%s': %w", typeName, err)
		}
		descs := et.Descriptor().Values()
		var desc protoreflect.EnumValueDescriptor
		switch y := x.Enum.By.(type) {
		case nil:
			return nil, errors.New("Value.enum.by not set")
		case *Value_Enum_Number:
			desc = descs.ByNumber(protoreflect.EnumNumber(y.Number))
			if desc == nil {
				return nil, fmt.Errorf("enum %s number %d not found",
					typeName, y.Number)
			}
		case *Value_Enum_Name:
			desc = descs.ByName(protoreflect.Name(y.Name))
			if desc == nil {
				return nil, fmt.Errorf("enum %s name %s not found", typeName, y.Name)
			}
		default:
			panic(fmt.Sprintf("BUG: unsupported enum by type %T", x.Enum.By))
		}
		return desc, nil
	case *Value_List_:
		typ, err := getProtoType(x.List.Kind, x.List.Type)
		if err != nil {
			return nil, fmt.Errorf("determine protobuf type for '%s'/'%s': %w",
				x.List.Kind, x.List.Type, err)
		}
		startIndex := 0
		length := len(x.List.Values)
		var listValue reflect.Value
		if typ == nil {
			if length == 0 {
				return nil, errors.New("type must be specified for empty list")
			}
			firstValue, err := eval(env, cyclesLeft, x.List.Values[0])
			if err != nil {
				return firstValue, fmt.Errorf("eval list index 0: %w", err)
			}
			startIndex = 1
			typ = reflect.TypeOf(firstValue)
			listValue = reflect.MakeSlice(reflect.SliceOf(typ), 0, length)
			listValue = reflect.Append(listValue, reflect.ValueOf(firstValue))
		} else {
			listValue = reflect.MakeSlice(reflect.SliceOf(typ), 0, length)
		}
		for i := startIndex; i < length; i++ {
			value, err := eval(env, cyclesLeft, x.List.Values[i])
			if err != nil {
				return value, fmt.Errorf("eval list index %d: %w", i, err)
			}
			if reflect.TypeOf(value) != typ {
				return nil, fmt.Errorf(
					"eval list index %d: expected type '%s', got '%T'", i, typ, value)
			}
			listValue = reflect.Append(listValue, reflect.ValueOf(value))
		}
		return listValue.Interface(), nil
	case *Value_Map_:
		keyType, err := getProtoMapKeyType(x.Map.KeyKind)
		if err != nil {
			return nil, fmt.Errorf("determine protobuf map key type for '%s': %w",
				x.Map.KeyKind, err)
		}
		valueType, err := getProtoType(x.Map.ValueKind, x.Map.ValueType)
		if err != nil {
			return nil, fmt.Errorf(
				"determine protobuf map value type for '%s'/'%s': %w",
				x.Map.ValueKind, x.Map.ValueType, err)
		}
		startIndex := 0
		length := len(x.Map.Entries)
		var mapValue reflect.Value
		if keyType == nil || valueType == nil {
			if length == 0 {
				return nil, errors.New(
					"both key and value type must be specified for empty map")
			}
			firstEntry := x.Map.Entries[0]
			if firstEntry.Key == nil {
				return nil, errors.New("map entry 0 key missing")
			}
			if firstEntry.Value == nil {
				return nil, errors.New("map entry 0 value missing")
			}
			firstKey, err := eval(env, cyclesLeft, firstEntry.Key)
			if err != nil {
				return firstKey, fmt.Errorf("eval map entry 0 key: %w", err)
			}
			firstValue, err := eval(env, cyclesLeft, firstEntry.Value)
			if err != nil {
				return firstValue, fmt.Errorf("eval map entry 0 value: %w", err)
			}
			startIndex = 1
			if keyType == nil {
				keyType = reflect.TypeOf(firstKey)
				if !isValidMapKeyType(keyType) {
					return nil, fmt.Errorf("invalid map key type: %s", keyType)
				}
			} else if reflect.TypeOf(firstKey) != keyType {
				return nil, fmt.Errorf(
					"eval map entry 0 key: expected type '%s', got '%T'",
					keyType, firstKey)
			}
			if valueType == nil {
				valueType = reflect.TypeOf(firstValue)
			} else if reflect.TypeOf(firstValue) != valueType {
				return nil, fmt.Errorf(
					"eval map entry 0 value: expected type '%s', got '%T'",
					valueType, firstValue)
			}
			mapValue = reflect.MakeMapWithSize(
				reflect.MapOf(keyType, valueType), length)
			mapValue.SetMapIndex(
				reflect.ValueOf(firstKey), reflect.ValueOf(firstValue))
		} else {
			mapValue = reflect.MakeMapWithSize(
				reflect.MapOf(keyType, valueType), length)
		}
		for i := startIndex; i < length; i++ {
			entry := x.Map.Entries[i]
			if entry.Key == nil {
				return nil, fmt.Errorf("map entry %d key missing", i)
			}
			if entry.Value == nil {
				return nil, fmt.Errorf("map entry %d value missing", i)
			}
			key, err := eval(env, cyclesLeft, entry.Key)
			if err != nil {
				return key, fmt.Errorf("eval map entry %d key: %w", i, err)
			}
			keyValue := reflect.ValueOf(key)
			if keyValue.Type() != keyType {
				return nil, fmt.Errorf(
					"eval map entry %d key: expected type '%s', got '%T'",
					i, keyType, key)
			}
			if mapValue.MapIndex(keyValue).IsValid() {
				return nil, fmt.Errorf("duplicate map entry %d key: %v", i, key)
			}
			value, err := eval(env, cyclesLeft, entry.Value)
			if err != nil {
				return value, fmt.Errorf("eval map entry %d value: %w", i, err)
			}
			if reflect.TypeOf(value) != valueType {
				return nil, fmt.Errorf(
					"eval map entry %d value: expected type '%s', got '%T'",
					i, valueType, value)
			}
			mapValue.SetMapIndex(keyValue, reflect.ValueOf(value))
		}
		return mapValue.Interface(), nil
	case *Value_Message_:
		msgName := protoreflect.FullName(x.Message.Type)
		msgType, err := protoregistry.GlobalTypes.FindMessageByName(msgName)
		if err != nil {
			return nil, fmt.Errorf("find message type '%s': %w", msgName, err)
		}
		result := msgType.New()
		if result == nil {
			return nil, fmt.Errorf("message type '%s' is synthetic", msgName)
		}
		fields := result.Descriptor().Fields()
		for key, value := range x.Message.Fields {
			fd := fields.ByName(protoreflect.Name(key))
			if fd == nil {
				return nil, fmt.Errorf("field '%s' in message '%s' not found",
					key, msgName)
			}
			oneof := fd.ContainingOneof()
			if oneof != nil && result.WhichOneof(oneof) != nil {
				return nil, fmt.Errorf("multiple fields set in oneof '%s'",
					oneof.FullName())
			}
			rv, err := eval(env, cyclesLeft, value)
			if err != nil {
				return rv, fmt.Errorf("eval message field '%s': %w", key, err)
			}
			// enum needs special care
			if evd, ok := rv.(protoreflect.EnumValueDescriptor); ok {
				dvd := fd.DefaultEnumValue()
				if dvd == nil {
					return nil, fmt.Errorf("message field '%s' is not an enum", key)
				}
				if dvd.Parent().FullName() != evd.Parent().FullName() {
					return nil, fmt.Errorf("bad enum number '%s' for enum '%s'",
						evd.FullName(), dvd.Parent().FullName())
				}
				rv = evd.Number()
			}
			func() {
				defer func() {
					if r := recover(); r != nil {
						err = fmt.Errorf("message field '%s' value type %T: %v",
							msgName, rv, r)
					}
				}()
				result.Set(fd, protoreflect.ValueOf(rv))
			}()
			if err != nil {
				return nil, err
			}
		}
		return result.Interface(), nil
	case *Value_BasicMessage:
		result, err := x.BasicMessage.UnmarshalNew()
		if err != nil {
			return nil, fmt.Errorf("unmarshal basic message: %w", err)
		}
		return result, nil
	case *Value_Duration:
		if err := x.Duration.CheckValid(); err != nil {
			return nil, fmt.Errorf("duration: %w", err)
		}
		return x.Duration.AsDuration(), nil
	case *Value_Timestamp:
		if err := x.Timestamp.CheckValid(); err != nil {
			return nil, fmt.Errorf("timestamp: %w", err)
		}
		return x.Timestamp.AsTime(), nil
	case *Value_Not:
		rv, err := eval(env, cyclesLeft, x.Not)
		if err != nil {
			return rv, fmt.Errorf("eval not: %w", err)
		}
		if bv, ok := rv.(bool); ok {
			return !bv, nil
		}
		return nil, fmt.Errorf("eval not: expected bool, got %T", rv)
	case *Value_AllOf:
		for i, value := range x.AllOf.Values {
			rv, err := eval(env, cyclesLeft, value)
			if err != nil {
				return rv, fmt.Errorf("eval all_of index %d: %w", i, err)
			}
			if bv, ok := rv.(bool); ok {
				if !bv {
					return false, nil
				}
				continue
			}
			return nil, fmt.Errorf("eval all_of index %d: expected bool, got %T",
				i, rv)
		}
		return true, nil
	case *Value_AnyOf:
		for i, value := range x.AnyOf.Values {
			rv, err := eval(env, cyclesLeft, value)
			if err != nil {
				return rv, fmt.Errorf("eval any_of index %d: %w", i, err)
			}
			if bv, ok := rv.(bool); ok {
				if bv {
					return true, nil
				}
				continue
			}
			return nil, fmt.Errorf("eval any_of index %d: expected bool, got %T",
				i, rv)
		}
		return false, nil
	case *Value_Eq:
		if len(x.Eq.Values) == 0 {
			return true, nil
		}
		firstValue, err := eval(env, cyclesLeft, x.Eq.Values[0])
		if err != nil {
			return firstValue, fmt.Errorf("eval eq index 0: %w", err)
		}
		typ := reflect.TypeOf(firstValue)
		for i := 1; i < len(x.Eq.Values); i++ {
			value, err := eval(env, cyclesLeft, x.Eq.Values[i])
			if err != nil {
				return value, fmt.Errorf("eval eq index %d: %w", i, err)
			}
			if reflect.TypeOf(value) != typ {
				return nil, fmt.Errorf("eval eq index %d: expected type '%s', got %T",
					i, typ, value)
			}
			if !protoEqual(firstValue, value) {
				return false, nil
			}
		}
		return true, nil
	case *Value_Neq:
		if len(x.Neq.Values) == 0 {
			return true, nil
		}
		values := make([]interface{}, len(x.Neq.Values))
		var err error
		values[0], err = eval(env, cyclesLeft, x.Neq.Values[0])
		if err != nil {
			return values[0], fmt.Errorf("eval neq index 0: %w", err)
		}
		typ := reflect.TypeOf(values[0])
		for i := 1; i < len(x.Neq.Values); i++ {
			values[i], err = eval(env, cyclesLeft, x.Neq.Values[i])
			if err != nil {
				return values[i], fmt.Errorf("eval neq index %d: %w", i, err)
			}
			if reflect.TypeOf(values[i]) != typ {
				return nil, fmt.Errorf("eval neq index %d: expected type '%s', got %T",
					i, typ, values[i])
			}
			for j := 0; j < i; j++ {
				if protoEqual(values[j], values[i]) {
					return false, nil
				}
			}
		}
		return true, nil
	case *Value_Lt:
		if len(x.Lt.Values) == 0 {
			return true, nil
		}
		lastValue, err := eval(env, cyclesLeft, x.Lt.Values[0])
		if err != nil {
			return lastValue, fmt.Errorf("eval lt index 0: %w", err)
		}
		typ := reflect.TypeOf(lastValue)
		if !isComparable(typ) {
			return nil, fmt.Errorf("type %s is not comparable", typ)
		}
		if isNaN(lastValue) {
			return false, nil
		}
		for i := 1; i < len(x.Lt.Values); i++ {
			value, err := eval(env, cyclesLeft, x.Lt.Values[i])
			if err != nil {
				return value, fmt.Errorf("eval lt index %d: %w", i, err)
			}
			if reflect.TypeOf(value) != typ {
				return nil, fmt.Errorf("eval lt index %d: expected type '%s', got %T",
					i, typ, value)
			}
			if isNaN(value) || protoCompare(lastValue, value) >= 0 {
				return false, nil
			}
			lastValue = value
		}
		return true, nil
	case *Value_Lte:
		if len(x.Lte.Values) == 0 {
			return true, nil
		}
		lastValue, err := eval(env, cyclesLeft, x.Lte.Values[0])
		if err != nil {
			return lastValue, fmt.Errorf("eval lte index 0: %w", err)
		}
		typ := reflect.TypeOf(lastValue)
		if !isComparable(typ) {
			return nil, fmt.Errorf("type %s is not comparable", typ)
		}
		if isNaN(lastValue) {
			return false, nil
		}
		for i := 1; i < len(x.Lte.Values); i++ {
			value, err := eval(env, cyclesLeft, x.Lte.Values[i])
			if err != nil {
				return value, fmt.Errorf("eval lte index %d: %w", i, err)
			}
			if reflect.TypeOf(value) != typ {
				return nil, fmt.Errorf("eval lte index %d: expected type '%s', got %T",
					i, typ, value)
			}
			if isNaN(value) || protoCompare(lastValue, value) > 0 {
				return false, nil
			}
			lastValue = value
		}
		return true, nil
	case *Value_Gt:
		if len(x.Gt.Values) == 0 {
			return true, nil
		}
		lastValue, err := eval(env, cyclesLeft, x.Gt.Values[0])
		if err != nil {
			return lastValue, fmt.Errorf("eval gt index 0: %w", err)
		}
		typ := reflect.TypeOf(lastValue)
		if !isComparable(typ) {
			return nil, fmt.Errorf("type %s is not comparable", typ)
		}
		if isNaN(lastValue) {
			return false, nil
		}
		for i := 1; i < len(x.Gt.Values); i++ {
			value, err := eval(env, cyclesLeft, x.Gt.Values[i])
			if err != nil {
				return value, fmt.Errorf("eval gt index %d: %w", i, err)
			}
			if reflect.TypeOf(value) != typ {
				return nil, fmt.Errorf("eval gt index %d: expected type '%s', got %T",
					i, typ, value)
			}
			if isNaN(value) || protoCompare(lastValue, value) <= 0 {
				return false, nil
			}
			lastValue = value
		}
		return true, nil
	case *Value_Gte:
		if len(x.Gte.Values) == 0 {
			return true, nil
		}
		lastValue, err := eval(env, cyclesLeft, x.Gte.Values[0])
		if err != nil {
			return lastValue, fmt.Errorf("eval gte index 0: %w", err)
		}
		typ := reflect.TypeOf(lastValue)
		if !isComparable(typ) {
			return nil, fmt.Errorf("type %s is not comparable", typ)
		}
		if isNaN(lastValue) {
			return false, nil
		}
		for i := 1; i < len(x.Gte.Values); i++ {
			value, err := eval(env, cyclesLeft, x.Gte.Values[i])
			if err != nil {
				return value, fmt.Errorf("eval gte index %d: %w", i, err)
			}
			if reflect.TypeOf(value) != typ {
				return nil, fmt.Errorf("eval gte index %d: expected type '%s', got %T",
					i, typ, value)
			}
			if isNaN(value) || protoCompare(lastValue, value) < 0 {
				return false, nil
			}
			lastValue = value
		}
		return true, nil
	case *Value_Seq:
		var result interface{}
		for _, value := range x.Seq.Values {
			rv, err := eval(env, cyclesLeft, value)
			switch err.(type) {
			case nil:
				result = rv
			case errBreak, errContinue:
				return result, fmt.Errorf("seq interrupted: %w", err)
			default:
				return rv, err
			}
		}
		return result, nil
	case *Value_Switch_:
		for i, cse := range x.Switch.Cases {
			if cse.Case == nil {
				return nil, fmt.Errorf("switch case index %d condition missing", i)
			}
			if cse.Then == nil {
				return nil, fmt.Errorf("switch case index %d branch missing", i)
			}
			cond, err := eval(env, cyclesLeft, cse.Case)
			if err != nil {
				return cond, fmt.Errorf("eval switch case index %d condition: %w",
					i, err)
			}
			if bv, ok := cond.(bool); !ok {
				return nil, fmt.Errorf(
					"eval switch case index %d condition: expected bool, got %T", i, cond)
			} else if !bv {
				continue
			}
			value, err := eval(env, cyclesLeft, cse.Then)
			if err != nil {
				return value, fmt.Errorf("eval switch case index %d branch: %w", i, err)
			}
			return value, nil
		}
		if x.Switch.Default == nil {
			return nil, nil
		}
		value, err := eval(env, cyclesLeft, x.Switch.Default)
		if err != nil {
			return value, fmt.Errorf("eval switch default case: %w", err)
		}
		return value, nil
	case *Value_While:
		if x.While.Case == nil {
			return nil, errors.New("while condition missing")
		}
		if x.While.Then == nil {
			return nil, errors.New("while body missing")
		}
		var lastValue interface{}
		for {
			cond, err := eval(env, cyclesLeft, x.While.Case)
			if err != nil {
				return cond, fmt.Errorf("eval while condition: %w", err)
			}
			if bv, ok := cond.(bool); !ok {
				return nil, fmt.Errorf("eval while condition: expected bool, got %d",
					cond)
			} else if !bv {
				return lastValue, nil
			}
			value, err := eval(env, cyclesLeft, x.While.Then)
			var brk errBreak
			var cont errContinue
			switch {
			case err == nil:
				lastValue = value
			case errors.As(err, &brk):
				if brk == 1 {
					if value == nil {
						value = lastValue
					}
					return value, nil
				}
				return value, errBreak(brk - 1)
			case errors.As(err, &cont):
				if cont == 1 {
					if value != nil {
						lastValue = value
						continue
					}
				}
				return value, errContinue(cont - 1)
			default:
				return value, fmt.Errorf("eval while body: %w", err)
			}
		}
	case *Value_Break:
		if x.Break == 0 {
			return nil, errors.New("break must be positive")
		}
		return nil, errBreak(x.Break)
	case *Value_Continue:
		if x.Continue == 0 {
			return nil, errors.New("continue must be positive")
		}
		return nil, errContinue(x.Continue)
	case *Value_Store:
		if x.Store.Value == nil {
			return nil, errors.New("store value missing")
		}
		value, err := eval(env, cyclesLeft, x.Store.Value)
		if err != nil {
			return value, fmt.Errorf("eval store: %w", err)
		}
		env.values[x.Store.Key] = value
		return value, nil
	case *Value_Proc:
		if x.Proc.Value == nil {
			return nil, errors.New("proc value missing")
		}
		env.values[x.Proc.Key] = x.Proc.Value
		return nil, nil
	case *Value_Load:
		value := env.values[x.Load]
		if value == nil {
			return nil, nil
		}
		var err error
		if proc, ok := value.(*Value); ok {
			value, err = eval(env, cyclesLeft, proc)
			if err != nil {
				return value, fmt.Errorf("eval loaded proc: %w", err)
			}
		}
		return value, nil
	case *Value_Program:
		initCelTypes()
		celEnv, err := cel.NewEnv(
			cel.Types(celTypes...),
			cel.Declarations(
				decls.NewVar("scope",
					decls.NewObjectType("com.github.thecount.protoeval.Scope"),
				),
			),
		)
		if err != nil {
			return nil, fmt.Errorf("build CEL environment: %w", err)
		}
		ast, iss := celEnv.Compile(x.Program)
		if iss.Err() != nil {
			return nil, fmt.Errorf("compile CEL program source: %w", iss.Err())
		}
		prg, err := celEnv.Program(ast)
		if err != nil {
			return nil, fmt.Errorf("construct CEL program: %w", err)
		}
		celScope, err := scope2cel(&env.scope)
		if err != nil {
			return nil, fmt.Errorf("construct CEL scope: %w", err)
		}
		out, _, err := prg.Eval(map[string]interface{}{
			"scope": celScope,
		})
		if err != nil {
			return nil, fmt.Errorf("evaluate CEL program: %w", err)
		}
		result, err := cel2go(out)
		if err != nil {
			return nil, fmt.Errorf("CEL program output: %w", err)
		}
		return result, nil
	case *Value_ScopeIs:
		return env.scope.Matches(protoreflect.FullName(x.ScopeIs)), nil
	case *Value_ScopeHas:
		return env.scope.Has(x.ScopeHas), nil
	default:
		panic(fmt.Sprintf("BUG: unsupported value type %T", value.Value))
	}
}
