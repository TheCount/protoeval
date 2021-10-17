package protoeval

import (
	"testing"
)

// TestProgram tests running a simple CEL program.
func TestProgram(t *testing.T) {
	testmsg := &ScopeTest{
		AScalar: 42,
	}
	env := NewEnv()
	result, err := Eval(env, testmsg, &Value{
		Value: &Value_Program{
			Program: "scope.value.a_scalar",
		},
	})
	if err != nil {
		t.Fatalf("eval program: %s", err)
	}
	scalar, ok := result.(int64) // CEL converts a_scalar to int64
	if !ok {
		t.Fatalf("result type %T is not int64", result)
	}
	if scalar != 42 {
		t.Errorf("expected 42, got %d", scalar)
	}
}
