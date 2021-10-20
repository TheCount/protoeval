package protoeval

import (
	"testing"
)

// TestScopeHas tests Value.scope_has.
func TestScopeHas(t *testing.T) {
	env := NewEnv()
	getresult := func(msg *HasTest, scope isValue_Scope, has string) bool {
		result, err := Eval(env, msg, &Value{
			Scope: scope,
			Value: &Value_ScopeHas{
				ScopeHas: has,
			},
		})
		if err != nil {
			t.Fatalf("scope_has '%s': %s", has, err)
		}
		return result.(bool)
	}
	testmsg := &HasTest{
		AStringMap: map[string]int32{
			"a_key": 42,
		},
		AnIntMap: map[int32]int32{
			42: 43,
		},
	}
	if getresult(testmsg, nil, "does_not_exist") {
		t.Error("expected false result for nonexistent field")
	}
	if !getresult(testmsg, nil, "field_without_presence") {
		t.Error("expected true result for zero field without presence")
	}
	if !getresult(testmsg, nil, "list_without_presence") {
		t.Error("expected true result for empty list without presence")
	}
	if getresult(testmsg, nil, "oneof_field") {
		t.Error("expected false result for unset field with presence")
	}
	if getresult(testmsg, nil, "test") {
		t.Error("expected false result for unset oneof")
	}
	testmsg.Test = &HasTest_FieldWithPresence{
		FieldWithPresence: 0,
	}
	if !getresult(testmsg, nil, "test") {
		t.Error("expected true result for set oneof")
	}
	if !getresult(testmsg, nil, "field_with_presence") {
		t.Error("expected true result for zero field with presence")
	}
	testmsg.ListWithoutPresence = []int32{42}
	if getresult(testmsg, &Value_Name{"list_without_presence"}, "0") {
		t.Error("expected false result for wrong scope value type")
	}
	if getresult(testmsg, &Value_Name{"a_string_map"}, "no_such_key") {
		t.Error("expected false result for map scope with no such key")
	}
	if !getresult(testmsg, &Value_Name{"a_string_map"}, "a_key") {
		t.Error("expected true result for existing map key")
	}
	if getresult(testmsg, &Value_Name{"an_int_map"}, "42") {
		t.Error("expected false result for map scope with wrong key type")
	}
}
