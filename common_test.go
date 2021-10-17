package protoeval

//go:generate protoc --go_out=. test.proto
//go:generate mv -f test.pb.go msg_test.go
