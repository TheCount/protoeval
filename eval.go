package protoeval

import (
	"errors"
	"fmt"
	reflect "reflect"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
	"github.com/google/cel-go/interpreter/functions"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/structpb"
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
// to the specified value, with the given arguments.
func Eval(
	env *Env, msg proto.Message, value *Value, args ...interface{},
) (interface{}, error) {
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
	for i := len(args) - 1; i >= 0; i-- {
		argVal := celTypeRegistry.NativeToValue(args[i])
		if types.IsError(argVal) {
			return nil, fmt.Errorf("arg %d: %w", i, argVal.Value().(error))
		}
		env.scope.PushArg(argVal)
	}
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
	if value.Scope != nil {
		env, err = env.shiftScope(value.Scope)
		if err != nil {
			return nil, err
		}
	}
	// handle args
	if err = env.scope.DropArgs(value.DropArgs); err != nil {
		return nil, err
	}
	for i := len(value.Args) - 1; i >= 0; i-- {
		rv, err := eval(env, cyclesLeft, value.Args[i])
		if err != nil {
			return nil, fmt.Errorf("arg %d: %w", i, err)
		}
		env.scope.PushArg(rv)
	}
	// evaluate
	switch x := value.Value.(type) {
	case nil:
		return env.scope.Value(), nil
	case *Value_Arg:
		return env.scope.Arg(x.Arg)
	case *Value_Parent:
		env, err = env.shiftScopeToParent()
		if err != nil {
			return nil, err
		}
		return eval(env, cyclesLeft, x.Parent)
	case *Value_Default:
		return env.scope.DefaultValue(), nil
	case *Value_BasicValue:
		switch y := x.BasicValue.Kind.(type) {
		case *structpb.Value_NullValue:
			return types.NullValue, nil
		case *structpb.Value_NumberValue:
			return types.Double(y.NumberValue), nil
		case *structpb.Value_StringValue:
			return types.String(y.StringValue), nil
		case *structpb.Value_BoolValue:
			return types.Bool(y.BoolValue), nil
		case *structpb.Value_StructValue:
			return celTypeRegistry.NativeToValue(y.StructValue), nil
		case *structpb.Value_ListValue:
			return celTypeRegistry.NativeToValue(y.ListValue), nil
		default:
			panic(fmt.Sprintf("BUG: unhandled structpb Value kind %T",
				x.BasicValue.Kind))
		}
	case *Value_Int:
		return types.Int(x.Int), nil
	case *Value_Uint:
		return types.Uint(x.Uint), nil
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
			if types.IsError(val) {
				return val, nil
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
			if types.IsError(keyVal) {
				return keyVal, nil
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
			if types.IsError(valueVal) {
				return valueVal, nil
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
			if types.IsError(rv) {
				return rv, nil
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
		if types.IsError(rv) {
			return rv, nil
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
			if types.IsError(rv) {
				return rv, nil
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
			if types.IsError(rv) {
				return rv, nil
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
	case *Value_Seq:
		var result ref.Val
		for _, value := range x.Seq.Values {
			rv, err := eval(env, cyclesLeft, value)
			switch err.(type) {
			case nil:
				if types.IsError(rv) {
					return rv, nil
				}
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
			if types.IsError(cond) {
				return cond, nil
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
			if types.IsError(cond) {
				return cond, nil
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
					if value == types.NullValue {
						value = lastValue
					}
					return value, nil
				}
				return value, errBreak(brk - 1)
			case errors.As(err, &cont):
				if cont == 1 {
					if types.IsError(value) {
						return value, nil
					}
					if value != types.NullValue {
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
		return types.NullValue, errBreak(x.Break)
	case *Value_Continue:
		if x.Continue == 0 {
			return nil, errors.New("continue must be positive")
		}
		return types.NullValue, errContinue(x.Continue)
	case *Value_Store:
		if x.Store.Value == nil {
			return nil, errors.New("store value missing")
		}
		value, err := eval(env, cyclesLeft, x.Store.Value)
		if err != nil {
			return value, fmt.Errorf("eval store: %w", err)
		}
		if types.IsError(value) {
			return value, nil
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
	case *Value_Program_:
		code := x.Program.Code
		if code == "" {
			code = strings.Join(x.Program.Lines, "\n")
			if code == "" {
				return nil, errors.New("no code in program")
			}
		} else if len(x.Program.Lines) != 0 {
			return nil, errors.New("lines must not be set if code is non-empty")
		}
		cEnv, err := cel.NewEnv(
			cel.CustomTypeAdapter(celTypeRegistry),
			cel.CustomTypeProvider(celTypeRegistry),
			cel.Declarations(
				decls.NewVar("env", decls.NewMapType(decls.String, decls.Dyn)),
				decls.NewVar("scope",
					decls.NewObjectType("com.github.thecount.protoeval.Scope"),
				),
				decls.NewVar("args", decls.NewListType(decls.Dyn)),
				decls.NewFunction("nix", decls.NewInstanceOverload("dyn_nix",
					[]*exprpb.Type{decls.Dyn}, decls.Null)),
				decls.NewFunction("store",
					decls.NewInstanceOverload("dyn_store_string",
						[]*exprpb.Type{decls.Dyn, decls.String}, decls.Dyn)),
			),
		)
		if err != nil {
			return nil, fmt.Errorf("build CEL environment: %w", err)
		}
		ast, iss := cEnv.Compile(code)
		if iss.Err() != nil {
			return nil, fmt.Errorf("compile CEL program source: %w", iss.Err())
		}
		prg, err := cEnv.Program(ast, cel.Functions(&functions.Overload{
			Operator: "dyn_nix",
			Unary: func(ref.Val) ref.Val {
				return types.NullValue
			},
		}, &functions.Overload{
			Operator: "dyn_store_string",
			Binary: func(lhs, rhs ref.Val) ref.Val {
				v := lhs
				if types.IsError(v) {
					return v
				}
				k, ok := rhs.Value().(string)
				if !ok {
					return types.MaybeNoSuchOverloadErr(rhs)
				}
				env.values[k] = envValue{
					origType: reflect.TypeOf(v.Value()),
					value:    v,
				}
				return v
			},
		}))
		if err != nil {
			return nil, fmt.Errorf("construct CEL program: %w", err)
		}
		celScope, err := scope2cel(&env.scope)
		if err != nil {
			return nil, fmt.Errorf("construct CEL scope: %w", err)
		}
		out, _, err := prg.Eval(map[string]interface{}{
			"env":   (*celEnv)(env),
			"scope": celScope,
			"args":  (*celArgList)(&env.scope),
		})
		if err != nil {
			return nil, fmt.Errorf("evaluate CEL program: %w", err)
		}
		return out, nil
	case *Value_Range_:
		if x.Range.Value == nil {
			return nil, errors.New("range value missing")
		}
		var sv ref.Val
		if x.Range.Iterable == nil {
			sv = env.scope.Value()
		} else {
			var err error
			sv, err = eval(env, cyclesLeft, x.Range.Iterable)
			if err != nil {
				return sv, fmt.Errorf("eval range iterable: %w", err)
			}
		}
		switch y := sv.(type) {
		case traits.Lister:
			for i, iter := 0, y.Iterator(); iter.HasNext() == types.True; i++ {
				env.scope.PushArg(iter.Next())
				env.scope.PushArg(types.Int(i))
				rv, err := eval(env, cyclesLeft, x.Range.Value)
				if err2 := env.scope.DropArgs(2); err2 != nil {
					return nil,
						fmt.Errorf("list range element %d drop index/value: %w", i, err2)
				}
				if err != nil {
					return rv, fmt.Errorf("eval list range element %d: %w", i, err)
				}
				if rv != types.NullValue {
					return rv, nil
				}
			}
			return types.NullValue, nil
		case traits.Mapper:
			for iter := y.Iterator(); iter.HasNext() == types.True; {
				key := iter.Next()
				value := y.Get(key)
				env.scope.PushArg(value)
				env.scope.PushArg(key)
				rv, err := eval(env, cyclesLeft, x.Range.Value)
				if err2 := env.scope.DropArgs(2); err2 != nil {
					return nil, fmt.Errorf("map range key %v drop index/value: %w",
						key.Value(), err2)
				}
				if err != nil {
					return rv, fmt.Errorf("eval map range key %v: %w", key.Value(), err)
				}
				if rv != types.NullValue {
					return rv, nil
				}
			}
			return types.NullValue, nil
		default:
			if types.IsError(sv) {
				return sv, nil
			}
			return nil, fmt.Errorf("type %T not iterable", sv.Value())
		}
	default:
		panic(fmt.Sprintf("BUG: unsupported value type %T", value.Value))
	}
}
