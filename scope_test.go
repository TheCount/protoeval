package protoeval

import (
	"testing"

	"google.golang.org/protobuf/proto"
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
		Scope: &Value_Name{
			Name: "does_not_exist",
		},
	}); err == nil {
		t.Error("expected error selecting nonexistent field")
	}
	result, err := Eval(env, testmsg, &Value{
		Scope: &Value_Name{
			Name: "a_scalar",
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
		Scope: &Value_Name{
			Name: "a_string_map",
		},
		Value: &Value_This{
			This: &Value{
				Scope: &Value_Name{
					Name: "does_not_exist",
				},
			},
		},
	}); err == nil {
		t.Error("expected error selecting nonexistent map entry")
	}
	result, err := Eval(env, testmsg, &Value{
		Scope: &Value_Name{
			Name: "a_string_map",
		},
		Value: &Value_This{
			This: &Value{
				Scope: &Value_Name{
					Name: "MapKey",
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
		Scope: &Value_Name{
			Name: "a_bool_map",
		},
		Value: &Value_This{
			This: &Value{
				Scope: &Value_BoolKey{
					BoolKey: false,
				},
			},
		},
	}); err == nil {
		t.Error("expected error selecting nonexistent map entry")
	}
	result, err := Eval(env, testmsg, &Value{
		Scope: &Value_Name{
			Name: "a_bool_map",
		},
		Value: &Value_This{
			This: &Value{
				Scope: &Value_BoolKey{
					BoolKey: true,
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
