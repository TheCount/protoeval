package protoeval

import (
	"testing"
)

// TestRangeScopeList tests ranging over a scope list.
func TestRangeScopeList(t *testing.T) {
	testmsg := &ScopeTest{
		AList: []int32{1, 2, 3, 4},
	}
	env := NewEnv()
	if err := env.Set("sum", int64(0)); err != nil {
		t.Fatalf("set sum=0 in environment: %s", err)
	}
	result, err := evalJSON(env, testmsg, `
    {
      "scope": ["a_list"],
      "range": {
        "value": { "program": { "lines": [
          "(env.sum+args[1]).store('sum') == 0 ? null : null"
        ]}}
      }
    }
  `)
	if err != nil {
		t.Fatalf("eval scope list range: %s", err)
	}
	if result != nil {
		t.Error("expected nil result")
	}
	rv, ok := env.Get("sum")
	if !ok {
		t.Fatal("sum is missing")
	}
	sum, ok := rv.(int64)
	if !ok {
		t.Fatalf("sum is %T, expected int64", rv)
	}
	if sum != 10 {
		t.Errorf("expected sum=10, got %d", sum)
	}
}

// TestRangeScopeMap tests ranging over a scope map.
func TestRangeScopeMap(t *testing.T) {
	testmsg := &ScopeTest{
		AStringMap: map[string]int32{
			"one":   1,
			"two":   2,
			"three": 3,
			"four":  4,
		},
	}
	env := NewEnv()
	if err := env.Set("sum", int64(0)); err != nil {
		t.Fatalf("set sum=0 in environment: %s", err)
	}
	result, err := evalJSON(env, testmsg, `
    {
      "scope": ["a_string_map"],
      "range": {
        "value": { "program": { "lines": [
          "(env.sum+args[1]).store('sum') == 0 ? null : null"
        ]}}
      }
    }
  `)
	if err != nil {
		t.Fatalf("eval scope list range: %s", err)
	}
	if result != nil {
		t.Error("expected nil result")
	}
	rv, ok := env.Get("sum")
	if !ok {
		t.Fatal("sum is missing")
	}
	sum, ok := rv.(int64)
	if !ok {
		t.Fatalf("sum is %T, expected int64", rv)
	}
	if sum != 10 {
		t.Errorf("expected sum=10, got %d", sum)
	}
}
