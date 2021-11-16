package protoeval

//go:generate protoc --go_out=. test.proto
//go:generate mv -f test.pb.go msg_test.go

import (
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// evalJSON is like Eval, except that the Value is specified as JSON.
func evalJSON(
	env *Env, msg proto.Message, jsonValue string,
) (interface{}, error) {
	var value Value
	err := protojson.UnmarshalOptions{}.Unmarshal([]byte(jsonValue), &value)
	if err != nil {
		return nil, err
	}
	return Eval(env, msg, &value)
}
