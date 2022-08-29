package protoeval

import (
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// TestEmptyScope tests a value with empty scope.
func TestEmptyScope(t *testing.T) {
	testmsg := &ScopeTest{
		AScalar: 42,
	}
	env := NewEnv()
	result, err := Eval(env, testmsg, &Value{})
	if err != nil {
		t.Fatalf("eval: %s", err)
	}
	msg, ok := result.(proto.Message)
	if !ok {
		t.Fatalf("result type %T is not proto.Message", result)
	}
	if !proto.Equal(testmsg, msg) {
		t.Errorf("expected %v, got %v", testmsg, msg)
	}
}

// TestFieldScope tests scoping into a field.
func TestFieldScope(t *testing.T) {
	testmsg := &ScopeTest{
		AScalar: 42,
	}
	env := NewEnv()
	if _, err := Eval(env, testmsg, &Value{
		Scope: &structpb.ListValue{
			Values: []*structpb.Value{
				{
					Kind: &structpb.Value_StringValue{
						StringValue: "does_not_exist",
					},
				},
			},
		},
	}); err == nil {
		t.Error("expected error selecting nonexistent field")
	}
	result, err := Eval(env, testmsg, &Value{
		Scope: &structpb.ListValue{
			Values: []*structpb.Value{
				{
					Kind: &structpb.Value_StringValue{
						StringValue: "a_scalar",
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("eval: %s", err)
	}
	scalar, ok := result.(int64) // cel type
	if !ok {
		t.Fatalf("result type %T is not int64", result)
	}
	if scalar != 42 {
		t.Errorf("expected 42, got %d", scalar)
	}
}

// TestStringMapScope tests scoping into a map with string keys.
func TestStringMapScope(t *testing.T) {
	testmsg := &ScopeTest{
		AStringMap: map[string]int32{
			"MapKey": 42,
		},
	}
	env := NewEnv()
	if _, err := Eval(env, testmsg, &Value{
		Scope: &structpb.ListValue{
			Values: []*structpb.Value{
				{
					Kind: &structpb.Value_StringValue{
						StringValue: "a_string_map",
					},
				},
				{
					Kind: &structpb.Value_StringValue{
						StringValue: "does_not_exist",
					},
				},
			},
		},
	}); err == nil {
		t.Error("expected error selecting nonexistent map entry")
	}
	result, err := Eval(env, testmsg, &Value{
		Scope: &structpb.ListValue{
			Values: []*structpb.Value{
				{
					Kind: &structpb.Value_StringValue{
						StringValue: "a_string_map",
					},
				},
				{
					Kind: &structpb.Value_StringValue{
						StringValue: "MapKey",
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("eval: %s", err)
	}
	scalar, ok := result.(int64) // cel type
	if !ok {
		t.Fatalf("result type %T is not int64", result)
	}
	if scalar != 42 {
		t.Errorf("expected 42, got %d", scalar)
	}
}

// TestBoolMapScope tests scoping into a map with boolean keys.
func TestBoolMapScope(t *testing.T) {
	testmsg := &ScopeTest{
		ABoolMap: map[bool]int32{
			true: 42,
		},
	}
	env := NewEnv()
	if _, err := Eval(env, testmsg, &Value{
		Scope: &structpb.ListValue{
			Values: []*structpb.Value{
				{
					Kind: &structpb.Value_StringValue{
						StringValue: "a_bool_map",
					},
				},
				{
					Kind: &structpb.Value_BoolValue{
						BoolValue: false,
					},
				},
			},
		},
	}); err == nil {
		t.Error("expected error selecting nonexistent map entry")
	}
	result, err := Eval(env, testmsg, &Value{
		Scope: &structpb.ListValue{
			Values: []*structpb.Value{
				{
					Kind: &structpb.Value_StringValue{
						StringValue: "a_bool_map",
					},
				},
				{
					Kind: &structpb.Value_BoolValue{
						BoolValue: true,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("eval: %s", err)
	}
	scalar, ok := result.(int64) // cel type
	if !ok {
		t.Fatalf("result type %T is not int64", result)
	}
	if scalar != 42 {
		t.Errorf("expected 42, got %d", scalar)
	}
}
