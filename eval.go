package protoeval

import (
	"errors"
	"fmt"
	reflect "reflect"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
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
	result, err := eval(env, &cyclesLeft, value)
	if err != nil {
		return nil, err
	}
	if types.IsError(result) {
		return nil, fmt.Errorf("evaluation error: %w", result.Value().(error))
	}
	if result.Type() == types.NullType {
		return nil, nil
	}
	return result.Value(), nil
}

// eval recursively evaluates msg in the given environment based on value.
func eval(env *Env, cyclesLeft *int, value *Value) (ref.Val, error) {
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
		return types.NullValue, nil
	case *Value_Bool:
		return types.Bool(x.Bool), nil
	case *Value_Int:
		return types.Int(x.Int), nil
	case *Value_Uint:
		return types.Uint(x.Uint), nil
	case *Value_Double:
		return types.Double(x.Double), nil
	case *Value_String_:
		return types.String(x.String_), nil
	case *Value_Bytes:
		return types.Bytes(x.Bytes), nil
	case *Value_Enum_:
		typeName := protoreflect.FullName(x.Enum.Type)
		et, err := protoregistry.GlobalTypes.FindEnumByName(typeName)
		if err != nil {
			return nil, fmt.Errorf("find enum '%s': %w", typeName, err)
		}
		descs := et.Descriptor().Values()
		switch y := x.Enum.By.(type) {
		case nil:
			return nil, errors.New("Value.enum.by not set")
		case *Value_Enum_Number:
			enumNumber := protoreflect.EnumNumber(y.Number)
			if descs.ByNumber(enumNumber) == nil {
				return nil, fmt.Errorf("enum %s number %d not found",
					typeName, y.Number)
			}
			return celTypeRegistry.NativeToValue(enumNumber), nil
		case *Value_Enum_Name:
			desc := descs.ByName(protoreflect.Name(y.Name))
			if desc == nil {
				return nil, fmt.Errorf("enum %s name %s not found", typeName, y.Name)
			}
			return celTypeRegistry.NativeToValue(desc.Number()), nil
		default:
			panic(fmt.Sprintf("BUG: unsupported enum by type %T", x.Enum.By))
		}
	case *Value_List_:
		k, t := x.List.Kind, x.List.Type
		if k == Value_INVALID && t != "" {
			k = Value_MESSAGE
		}
		typ, err := getProtoType(k, t)
		if err != nil {
			return nil, fmt.Errorf("determine protobuf type for '%s'/'%s': %w",
				k, t, err)
		}
		length := len(x.List.Values)
		listValue := reflect.MakeSlice(reflect.SliceOf(typ), 0, length)
		for i := 0; i != length; i++ {
			val, err := eval(env, cyclesLeft, x.List.Values[i])
			if err != nil {
				return val, fmt.Errorf("eval list index %d: %w", i, err)
			}
			item := reflect.ValueOf(val.Value())
			if item.Type().ConvertibleTo(typ) {
				listValue = reflect.Append(listValue, item.Convert(typ))
			} else {
				return val, fmt.Errorf("cannot convert %T to %s", val.Value(), typ)
			}
		}
		return types.NewDynamicList(celTypeRegistry, listValue.Interface()), nil
	case *Value_Map_:
		keyType, err := getProtoMapKeyType(x.Map.KeyKind)
		if err != nil {
			return nil, fmt.Errorf("determine protobuf map key type for '%s': %w",
				x.Map.KeyKind, err)
		}
		k, t := x.Map.ValueKind, x.Map.ValueType
		if k == Value_INVALID && t != "" {
			k = Value_MESSAGE
		}
		valueType, err := getProtoType(k, t)
		if err != nil {
			return nil, fmt.Errorf(
				"determine protobuf map value type for '%s'/'%s': %w", k, t, err)
		}
		length := len(x.Map.Entries)
		mapValue := reflect.MakeMapWithSize(
			reflect.MapOf(keyType, valueType), length)
		for i := 0; i != length; i++ {
			entry := x.Map.Entries[i]
			if entry.Key == nil {
				return nil, fmt.Errorf("map entry %d key missing", i)
			}
			if entry.Value == nil {
				return nil, fmt.Errorf("map entry %d value missing", i)
			}
			keyVal, err := eval(env, cyclesLeft, entry.Key)
			if err != nil {
				return keyVal, fmt.Errorf("eval map entry %d key: %w", i, err)
			}
			keyItem := reflect.ValueOf(keyVal.Value())
			if !keyItem.Type().ConvertibleTo(keyType) {
				return nil, fmt.Errorf("cannot convert map entry %d key type %T to %s",
					i, keyVal.Value(), keyType)
			}
			convertedKey := keyItem.Convert(keyType)
			if mapValue.MapIndex(convertedKey).IsValid() {
				return nil, fmt.Errorf("duplicate map entry %d key: %v",
					i, keyVal.Value())
			}
			valueVal, err := eval(env, cyclesLeft, entry.Value)
			if err != nil {
				return valueVal, fmt.Errorf("eval map entry %d value: %w", i, err)
			}
			valueItem := reflect.ValueOf(valueVal.Value())
			if !valueItem.Type().ConvertibleTo(valueType) {
				return nil, fmt.Errorf(
					"cannot convert map entry %d value type %T to %s",
					i, valueVal.Value(), valueType)
			}
			convertedValue := valueItem.Convert(valueType)
			mapValue.SetMapIndex(convertedKey, convertedValue)
		}
		return types.NewDynamicMap(celTypeRegistry, mapValue.Interface()), nil
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
			fieldValue, err := go2protofd(rv.Value(), result, fd)
			if err != nil {
				return nil, fmt.Errorf("convert %T to field '%s' value: %w",
					rv.Value(), key, err)
			}
			result.Set(fd, fieldValue)
		}
		return celTypeRegistry.NativeToValue(result.Interface()), nil
	case *Value_BasicMessage:
		return celTypeRegistry.NativeToValue(x.BasicMessage), nil
	case *Value_Duration:
		return celTypeRegistry.NativeToValue(x.Duration), nil
	case *Value_Timestamp:
		return celTypeRegistry.NativeToValue(x.Timestamp), nil
	case *Value_Not:
		rv, err := eval(env, cyclesLeft, x.Not)
		if err != nil {
			return rv, fmt.Errorf("eval not: %w", err)
		}
		if bv, ok := rv.(types.Bool); ok {
			return !bv, nil
		}
		return nil, fmt.Errorf("eval not: expected bool, got %T", rv)
	case *Value_AllOf:
		for i, value := range x.AllOf.Values {
			rv, err := eval(env, cyclesLeft, value)
			if err != nil {
				return rv, fmt.Errorf("eval all_of index %d: %w", i, err)
			}
			if bv, ok := rv.(types.Bool); ok {
				if !bv {
					return types.False, nil
				}
				continue
			}
			return nil, fmt.Errorf("eval all_of index %d: expected bool, got %T",
				i, rv)
		}
		return types.True, nil
	case *Value_AnyOf:
		for i, value := range x.AnyOf.Values {
			rv, err := eval(env, cyclesLeft, value)
			if err != nil {
				return rv, fmt.Errorf("eval any_of index %d: %w", i, err)
			}
			if bv, ok := rv.(types.Bool); ok {
				if bv {
					return types.True, nil
				}
				continue
			}
			return nil, fmt.Errorf("eval any_of index %d: expected bool, got %T",
				i, rv)
		}
		return types.False, nil
	case *Value_Eq:
		if len(x.Eq.Values) == 0 {
			return types.True, nil
		}
		firstValue, err := eval(env, cyclesLeft, x.Eq.Values[0])
		if err != nil {
			return firstValue, fmt.Errorf("eval eq index 0: %w", err)
		}
		for i := 1; i < len(x.Eq.Values); i++ {
			value, err := eval(env, cyclesLeft, x.Eq.Values[i])
			if err != nil {
				return value, fmt.Errorf("eval eq index %d: %w", i, err)
			}
			check := firstValue.Equal(value)
			if check != types.True {
				return check, nil
			}
		}
		return types.True, nil
	case *Value_Neq:
		if len(x.Neq.Values) == 0 {
			return types.True, nil
		}
		values := make([]ref.Val, len(x.Neq.Values))
		var err error
		values[0], err = eval(env, cyclesLeft, x.Neq.Values[0])
		if err != nil {
			return values[0], fmt.Errorf("eval neq index 0: %w", err)
		}
		for i := 1; i < len(x.Neq.Values); i++ {
			values[i], err = eval(env, cyclesLeft, x.Neq.Values[i])
			if err != nil {
				return values[i], fmt.Errorf("eval neq index %d: %w", i, err)
			}
			for j := 0; j < i; j++ {
				check := values[j].Equal(values[i])
				if check != types.False {
					return check, nil
				}
			}
		}
		return types.True, nil
	case *Value_Seq:
		var result ref.Val
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
			if bv, ok := cond.(types.Bool); !ok {
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
			return types.NullValue, nil
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
		var lastValue ref.Val
		for {
			cond, err := eval(env, cyclesLeft, x.While.Case)
			if err != nil {
				return cond, fmt.Errorf("eval while condition: %w", err)
			}
			if bv, ok := cond.(types.Bool); !ok {
				return nil, fmt.Errorf("eval while condition: expected bool, got %T",
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
		env.values[x.Store.Key] = envValue{
			origType: reflect.TypeOf(value.Value()),
			value:    value,
		}
		return value, nil
	case *Value_Proc:
		if x.Proc.Value == nil {
			return nil, errors.New("proc value missing")
		}
		env.values[x.Proc.Key] = envValue{
			origType: reflect.TypeOf((*Value)(nil)),
			value:    celTypeRegistry.NativeToValue(x.Proc.Value),
		}
		return types.NullValue, nil
	case *Value_Load:
		envValue, ok := env.values[x.Load]
		if !ok {
			return types.NullValue, nil
		}
		if proc, ok := envValue.value.Value().(*Value); ok {
			value, err := eval(env, cyclesLeft, proc)
			if err != nil {
				return value, fmt.Errorf("eval loaded proc: %w", err)
			}
		}
		return envValue.value, nil
	case *Value_Program:
		celEnv, err := cel.NewEnv(
			cel.CustomTypeAdapter(celTypeRegistry),
			cel.CustomTypeProvider(celTypeRegistry),
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
		return out, nil
	case *Value_ScopeIs:
		return types.Bool(env.scope.Matches(protoreflect.FullName(x.ScopeIs))), nil
	case *Value_ScopeHas:
		return types.Bool(env.scope.Has(x.ScopeHas)), nil
	default:
		panic(fmt.Sprintf("BUG: unsupported value type %T", value.Value))
	}
}
