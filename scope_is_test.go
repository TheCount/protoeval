package protoeval

import (
	"testing"

	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/emptypb"
)

// TestScopeIs tests the scope_is field of value.
func TestScopeIs(t *testing.T) {
	const (
		first  = "the first"
		second = "the second"
		other  = "none of the above"
	)
	proc := &Value{
		Value: &Value_Switch_{
			Switch: &Value_Switch{
				Cases: []*Value_Branch{
					&Value_Branch{
						Case: &Value{
							Value: &Value_ScopeIs{
								ScopeIs: "com.github.thecount.protoeval.ScopeTest",
							},
						},
						Then: &Value{
							Value: &Value_String_{
								String_: first,
							},
						},
					},
					&Value_Branch{
						Case: &Value{
							Value: &Value_ScopeIs{
								ScopeIs: "google.protobuf.Empty",
							},
						},
						Then: &Value{
							Value: &Value_String_{
								String_: second,
							},
						},
					},
				},
				Default: &Value{
					Value: &Value_String_{
						String_: other,
					},
				},
			},
		},
	}
	env := NewEnv()
	result, err := Eval(env, &ScopeTest{}, proc)
	if err != nil {
		t.Fatalf("scope_is ScopeTest: %s", err)
	}
	if result.(string) != first {
		t.Errorf("scope_is ScopeTest: expected '%s', got '%s'",
			first, result.(string))
	}
	result, err = Eval(env, &emptypb.Empty{}, proc)
	if err != nil {
		t.Fatalf("scope_is Empty: %s", err)
	}
	if result.(string) != second {
		t.Errorf("scope_is Empty: expected '%s', got '%s'",
			second, result.(string))
	}
	result, err = Eval(env, &anypb.Any{}, proc)
	if err != nil {
		t.Fatalf("scope_is Any: %s", err)
	}
	if result.(string) != other {
		t.Errorf("scope_is Any: expected '%s', got '%s'",
			other, result.(string))
	}
}
