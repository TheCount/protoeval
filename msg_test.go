// File test.proto defines protobuf messages used for testing the protoeval
// package. The messages defined here will not be included in production
// builds.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.12.2
// source: test.proto

package protoeval

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// ScopeTest contains fields for scope selection testing.
type ScopeTest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	AScalar    int32            `protobuf:"varint,1,opt,name=a_scalar,json=aScalar,proto3" json:"a_scalar,omitempty"`
	AList      []int32          `protobuf:"varint,2,rep,packed,name=a_list,json=aList,proto3" json:"a_list,omitempty"`
	AStringMap map[string]int32 `protobuf:"bytes,3,rep,name=a_string_map,json=aStringMap,proto3" json:"a_string_map,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"varint,2,opt,name=value,proto3"`
	ABoolMap   map[bool]int32   `protobuf:"bytes,4,rep,name=a_bool_map,json=aBoolMap,proto3" json:"a_bool_map,omitempty" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"varint,2,opt,name=value,proto3"`
	AUint32Map map[uint32]int32 `protobuf:"bytes,5,rep,name=a_uint32_map,json=aUint32Map,proto3" json:"a_uint32_map,omitempty" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"varint,2,opt,name=value,proto3"`
	AUint64Map map[uint64]int32 `protobuf:"bytes,6,rep,name=a_uint64_map,json=aUint64Map,proto3" json:"a_uint64_map,omitempty" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"varint,2,opt,name=value,proto3"`
	AnInt32Map map[int32]int32  `protobuf:"bytes,7,rep,name=an_int32_map,json=anInt32Map,proto3" json:"an_int32_map,omitempty" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"varint,2,opt,name=value,proto3"`
	AnInt64Map map[int64]int32  `protobuf:"bytes,8,rep,name=an_int64_map,json=anInt64Map,proto3" json:"an_int64_map,omitempty" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"varint,2,opt,name=value,proto3"`
}

func (x *ScopeTest) Reset() {
	*x = ScopeTest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_test_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ScopeTest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ScopeTest) ProtoMessage() {}

func (x *ScopeTest) ProtoReflect() protoreflect.Message {
	mi := &file_test_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ScopeTest.ProtoReflect.Descriptor instead.
func (*ScopeTest) Descriptor() ([]byte, []int) {
	return file_test_proto_rawDescGZIP(), []int{0}
}

func (x *ScopeTest) GetAScalar() int32 {
	if x != nil {
		return x.AScalar
	}
	return 0
}

func (x *ScopeTest) GetAList() []int32 {
	if x != nil {
		return x.AList
	}
	return nil
}

func (x *ScopeTest) GetAStringMap() map[string]int32 {
	if x != nil {
		return x.AStringMap
	}
	return nil
}

func (x *ScopeTest) GetABoolMap() map[bool]int32 {
	if x != nil {
		return x.ABoolMap
	}
	return nil
}

func (x *ScopeTest) GetAUint32Map() map[uint32]int32 {
	if x != nil {
		return x.AUint32Map
	}
	return nil
}

func (x *ScopeTest) GetAUint64Map() map[uint64]int32 {
	if x != nil {
		return x.AUint64Map
	}
	return nil
}

func (x *ScopeTest) GetAnInt32Map() map[int32]int32 {
	if x != nil {
		return x.AnInt32Map
	}
	return nil
}

func (x *ScopeTest) GetAnInt64Map() map[int64]int32 {
	if x != nil {
		return x.AnInt64Map
	}
	return nil
}

// HasTest contains fields and a oneof for testing Value.has.
type HasTest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	FieldWithoutPresence int32   `protobuf:"varint,1,opt,name=field_without_presence,json=fieldWithoutPresence,proto3" json:"field_without_presence,omitempty"`
	ListWithoutPresence  []int32 `protobuf:"varint,2,rep,packed,name=list_without_presence,json=listWithoutPresence,proto3" json:"list_without_presence,omitempty"`
	// Types that are assignable to Test:
	//	*HasTest_FieldWithPresence
	Test       isHasTest_Test   `protobuf_oneof:"test"`
	AStringMap map[string]int32 `protobuf:"bytes,4,rep,name=a_string_map,json=aStringMap,proto3" json:"a_string_map,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"varint,2,opt,name=value,proto3"`
	AnIntMap   map[int32]int32  `protobuf:"bytes,5,rep,name=an_int_map,json=anIntMap,proto3" json:"an_int_map,omitempty" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"varint,2,opt,name=value,proto3"`
}

func (x *HasTest) Reset() {
	*x = HasTest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_test_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *HasTest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HasTest) ProtoMessage() {}

func (x *HasTest) ProtoReflect() protoreflect.Message {
	mi := &file_test_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HasTest.ProtoReflect.Descriptor instead.
func (*HasTest) Descriptor() ([]byte, []int) {
	return file_test_proto_rawDescGZIP(), []int{1}
}

func (x *HasTest) GetFieldWithoutPresence() int32 {
	if x != nil {
		return x.FieldWithoutPresence
	}
	return 0
}

func (x *HasTest) GetListWithoutPresence() []int32 {
	if x != nil {
		return x.ListWithoutPresence
	}
	return nil
}

func (m *HasTest) GetTest() isHasTest_Test {
	if m != nil {
		return m.Test
	}
	return nil
}

func (x *HasTest) GetFieldWithPresence() int32 {
	if x, ok := x.GetTest().(*HasTest_FieldWithPresence); ok {
		return x.FieldWithPresence
	}
	return 0
}

func (x *HasTest) GetAStringMap() map[string]int32 {
	if x != nil {
		return x.AStringMap
	}
	return nil
}

func (x *HasTest) GetAnIntMap() map[int32]int32 {
	if x != nil {
		return x.AnIntMap
	}
	return nil
}

type isHasTest_Test interface {
	isHasTest_Test()
}

type HasTest_FieldWithPresence struct {
	FieldWithPresence int32 `protobuf:"varint,3,opt,name=field_with_presence,json=fieldWithPresence,proto3,oneof"`
}

func (*HasTest_FieldWithPresence) isHasTest_Test() {}

var File_test_proto protoreflect.FileDescriptor

var file_test_proto_rawDesc = []byte{
	0x0a, 0x0a, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x1d, 0x63, 0x6f,
	0x6d, 0x2e, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x74, 0x68, 0x65, 0x63, 0x6f, 0x75, 0x6e,
	0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x65, 0x76, 0x61, 0x6c, 0x22, 0xd7, 0x07, 0x0a, 0x09,
	0x53, 0x63, 0x6f, 0x70, 0x65, 0x54, 0x65, 0x73, 0x74, 0x12, 0x19, 0x0a, 0x08, 0x61, 0x5f, 0x73,
	0x63, 0x61, 0x6c, 0x61, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x07, 0x61, 0x53, 0x63,
	0x61, 0x6c, 0x61, 0x72, 0x12, 0x15, 0x0a, 0x06, 0x61, 0x5f, 0x6c, 0x69, 0x73, 0x74, 0x18, 0x02,
	0x20, 0x03, 0x28, 0x05, 0x52, 0x05, 0x61, 0x4c, 0x69, 0x73, 0x74, 0x12, 0x5a, 0x0a, 0x0c, 0x61,
	0x5f, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x5f, 0x6d, 0x61, 0x70, 0x18, 0x03, 0x20, 0x03, 0x28,
	0x0b, 0x32, 0x38, 0x2e, 0x63, 0x6f, 0x6d, 0x2e, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x74,
	0x68, 0x65, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x65, 0x76, 0x61,
	0x6c, 0x2e, 0x53, 0x63, 0x6f, 0x70, 0x65, 0x54, 0x65, 0x73, 0x74, 0x2e, 0x41, 0x53, 0x74, 0x72,
	0x69, 0x6e, 0x67, 0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x0a, 0x61, 0x53, 0x74,
	0x72, 0x69, 0x6e, 0x67, 0x4d, 0x61, 0x70, 0x12, 0x54, 0x0a, 0x0a, 0x61, 0x5f, 0x62, 0x6f, 0x6f,
	0x6c, 0x5f, 0x6d, 0x61, 0x70, 0x18, 0x04, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x36, 0x2e, 0x63, 0x6f,
	0x6d, 0x2e, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x74, 0x68, 0x65, 0x63, 0x6f, 0x75, 0x6e,
	0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x65, 0x76, 0x61, 0x6c, 0x2e, 0x53, 0x63, 0x6f, 0x70,
	0x65, 0x54, 0x65, 0x73, 0x74, 0x2e, 0x41, 0x42, 0x6f, 0x6f, 0x6c, 0x4d, 0x61, 0x70, 0x45, 0x6e,
	0x74, 0x72, 0x79, 0x52, 0x08, 0x61, 0x42, 0x6f, 0x6f, 0x6c, 0x4d, 0x61, 0x70, 0x12, 0x5a, 0x0a,
	0x0c, 0x61, 0x5f, 0x75, 0x69, 0x6e, 0x74, 0x33, 0x32, 0x5f, 0x6d, 0x61, 0x70, 0x18, 0x05, 0x20,
	0x03, 0x28, 0x0b, 0x32, 0x38, 0x2e, 0x63, 0x6f, 0x6d, 0x2e, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62,
	0x2e, 0x74, 0x68, 0x65, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x65,
	0x76, 0x61, 0x6c, 0x2e, 0x53, 0x63, 0x6f, 0x70, 0x65, 0x54, 0x65, 0x73, 0x74, 0x2e, 0x41, 0x55,
	0x69, 0x6e, 0x74, 0x33, 0x32, 0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x0a, 0x61,
	0x55, 0x69, 0x6e, 0x74, 0x33, 0x32, 0x4d, 0x61, 0x70, 0x12, 0x5a, 0x0a, 0x0c, 0x61, 0x5f, 0x75,
	0x69, 0x6e, 0x74, 0x36, 0x34, 0x5f, 0x6d, 0x61, 0x70, 0x18, 0x06, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x38, 0x2e, 0x63, 0x6f, 0x6d, 0x2e, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x74, 0x68, 0x65,
	0x63, 0x6f, 0x75, 0x6e, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x65, 0x76, 0x61, 0x6c, 0x2e,
	0x53, 0x63, 0x6f, 0x70, 0x65, 0x54, 0x65, 0x73, 0x74, 0x2e, 0x41, 0x55, 0x69, 0x6e, 0x74, 0x36,
	0x34, 0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x0a, 0x61, 0x55, 0x69, 0x6e, 0x74,
	0x36, 0x34, 0x4d, 0x61, 0x70, 0x12, 0x5a, 0x0a, 0x0c, 0x61, 0x6e, 0x5f, 0x69, 0x6e, 0x74, 0x33,
	0x32, 0x5f, 0x6d, 0x61, 0x70, 0x18, 0x07, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x38, 0x2e, 0x63, 0x6f,
	0x6d, 0x2e, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x74, 0x68, 0x65, 0x63, 0x6f, 0x75, 0x6e,
	0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x65, 0x76, 0x61, 0x6c, 0x2e, 0x53, 0x63, 0x6f, 0x70,
	0x65, 0x54, 0x65, 0x73, 0x74, 0x2e, 0x41, 0x6e, 0x49, 0x6e, 0x74, 0x33, 0x32, 0x4d, 0x61, 0x70,
	0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x0a, 0x61, 0x6e, 0x49, 0x6e, 0x74, 0x33, 0x32, 0x4d, 0x61,
	0x70, 0x12, 0x5a, 0x0a, 0x0c, 0x61, 0x6e, 0x5f, 0x69, 0x6e, 0x74, 0x36, 0x34, 0x5f, 0x6d, 0x61,
	0x70, 0x18, 0x08, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x38, 0x2e, 0x63, 0x6f, 0x6d, 0x2e, 0x67, 0x69,
	0x74, 0x68, 0x75, 0x62, 0x2e, 0x74, 0x68, 0x65, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x65, 0x76, 0x61, 0x6c, 0x2e, 0x53, 0x63, 0x6f, 0x70, 0x65, 0x54, 0x65, 0x73,
	0x74, 0x2e, 0x41, 0x6e, 0x49, 0x6e, 0x74, 0x36, 0x34, 0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74, 0x72,
	0x79, 0x52, 0x0a, 0x61, 0x6e, 0x49, 0x6e, 0x74, 0x36, 0x34, 0x4d, 0x61, 0x70, 0x1a, 0x3d, 0x0a,
	0x0f, 0x41, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74, 0x72, 0x79,
	0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b,
	0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x05, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x1a, 0x3b, 0x0a, 0x0d,
	0x41, 0x42, 0x6f, 0x6f, 0x6c, 0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a,
	0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12,
	0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x05,
	0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x1a, 0x3d, 0x0a, 0x0f, 0x41, 0x55, 0x69,
	0x6e, 0x74, 0x33, 0x32, 0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03,
	0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14,
	0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x05, 0x76,
	0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x1a, 0x3d, 0x0a, 0x0f, 0x41, 0x55, 0x69, 0x6e,
	0x74, 0x36, 0x34, 0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b,
	0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a,
	0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x05, 0x76, 0x61,
	0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x1a, 0x3d, 0x0a, 0x0f, 0x41, 0x6e, 0x49, 0x6e, 0x74,
	0x33, 0x32, 0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65,
	0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05,
	0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x05, 0x76, 0x61, 0x6c,
	0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x1a, 0x3d, 0x0a, 0x0f, 0x41, 0x6e, 0x49, 0x6e, 0x74, 0x36,
	0x34, 0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76,
	0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75,
	0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0xd7, 0x03, 0x0a, 0x07, 0x48, 0x61, 0x73, 0x54, 0x65, 0x73,
	0x74, 0x12, 0x34, 0x0a, 0x16, 0x66, 0x69, 0x65, 0x6c, 0x64, 0x5f, 0x77, 0x69, 0x74, 0x68, 0x6f,
	0x75, 0x74, 0x5f, 0x70, 0x72, 0x65, 0x73, 0x65, 0x6e, 0x63, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x05, 0x52, 0x14, 0x66, 0x69, 0x65, 0x6c, 0x64, 0x57, 0x69, 0x74, 0x68, 0x6f, 0x75, 0x74, 0x50,
	0x72, 0x65, 0x73, 0x65, 0x6e, 0x63, 0x65, 0x12, 0x32, 0x0a, 0x15, 0x6c, 0x69, 0x73, 0x74, 0x5f,
	0x77, 0x69, 0x74, 0x68, 0x6f, 0x75, 0x74, 0x5f, 0x70, 0x72, 0x65, 0x73, 0x65, 0x6e, 0x63, 0x65,
	0x18, 0x02, 0x20, 0x03, 0x28, 0x05, 0x52, 0x13, 0x6c, 0x69, 0x73, 0x74, 0x57, 0x69, 0x74, 0x68,
	0x6f, 0x75, 0x74, 0x50, 0x72, 0x65, 0x73, 0x65, 0x6e, 0x63, 0x65, 0x12, 0x30, 0x0a, 0x13, 0x66,
	0x69, 0x65, 0x6c, 0x64, 0x5f, 0x77, 0x69, 0x74, 0x68, 0x5f, 0x70, 0x72, 0x65, 0x73, 0x65, 0x6e,
	0x63, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x05, 0x48, 0x00, 0x52, 0x11, 0x66, 0x69, 0x65, 0x6c,
	0x64, 0x57, 0x69, 0x74, 0x68, 0x50, 0x72, 0x65, 0x73, 0x65, 0x6e, 0x63, 0x65, 0x12, 0x58, 0x0a,
	0x0c, 0x61, 0x5f, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x5f, 0x6d, 0x61, 0x70, 0x18, 0x04, 0x20,
	0x03, 0x28, 0x0b, 0x32, 0x36, 0x2e, 0x63, 0x6f, 0x6d, 0x2e, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62,
	0x2e, 0x74, 0x68, 0x65, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x65,
	0x76, 0x61, 0x6c, 0x2e, 0x48, 0x61, 0x73, 0x54, 0x65, 0x73, 0x74, 0x2e, 0x41, 0x53, 0x74, 0x72,
	0x69, 0x6e, 0x67, 0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x0a, 0x61, 0x53, 0x74,
	0x72, 0x69, 0x6e, 0x67, 0x4d, 0x61, 0x70, 0x12, 0x52, 0x0a, 0x0a, 0x61, 0x6e, 0x5f, 0x69, 0x6e,
	0x74, 0x5f, 0x6d, 0x61, 0x70, 0x18, 0x05, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x34, 0x2e, 0x63, 0x6f,
	0x6d, 0x2e, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x74, 0x68, 0x65, 0x63, 0x6f, 0x75, 0x6e,
	0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x65, 0x76, 0x61, 0x6c, 0x2e, 0x48, 0x61, 0x73, 0x54,
	0x65, 0x73, 0x74, 0x2e, 0x41, 0x6e, 0x49, 0x6e, 0x74, 0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74, 0x72,
	0x79, 0x52, 0x08, 0x61, 0x6e, 0x49, 0x6e, 0x74, 0x4d, 0x61, 0x70, 0x1a, 0x3d, 0x0a, 0x0f, 0x41,
	0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10,
	0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79,
	0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52,
	0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x1a, 0x3b, 0x0a, 0x0d, 0x41, 0x6e,
	0x49, 0x6e, 0x74, 0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b,
	0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a,
	0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x05, 0x76, 0x61,
	0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x42, 0x06, 0x0a, 0x04, 0x74, 0x65, 0x73, 0x74, 0x42,
	0x1f, 0x5a, 0x1d, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x54, 0x68,
	0x65, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x65, 0x76, 0x61, 0x6c,
	0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_test_proto_rawDescOnce sync.Once
	file_test_proto_rawDescData = file_test_proto_rawDesc
)

func file_test_proto_rawDescGZIP() []byte {
	file_test_proto_rawDescOnce.Do(func() {
		file_test_proto_rawDescData = protoimpl.X.CompressGZIP(file_test_proto_rawDescData)
	})
	return file_test_proto_rawDescData
}

var file_test_proto_msgTypes = make([]protoimpl.MessageInfo, 10)
var file_test_proto_goTypes = []interface{}{
	(*ScopeTest)(nil), // 0: com.github.thecount.protoeval.ScopeTest
	(*HasTest)(nil),   // 1: com.github.thecount.protoeval.HasTest
	nil,               // 2: com.github.thecount.protoeval.ScopeTest.AStringMapEntry
	nil,               // 3: com.github.thecount.protoeval.ScopeTest.ABoolMapEntry
	nil,               // 4: com.github.thecount.protoeval.ScopeTest.AUint32MapEntry
	nil,               // 5: com.github.thecount.protoeval.ScopeTest.AUint64MapEntry
	nil,               // 6: com.github.thecount.protoeval.ScopeTest.AnInt32MapEntry
	nil,               // 7: com.github.thecount.protoeval.ScopeTest.AnInt64MapEntry
	nil,               // 8: com.github.thecount.protoeval.HasTest.AStringMapEntry
	nil,               // 9: com.github.thecount.protoeval.HasTest.AnIntMapEntry
}
var file_test_proto_depIdxs = []int32{
	2, // 0: com.github.thecount.protoeval.ScopeTest.a_string_map:type_name -> com.github.thecount.protoeval.ScopeTest.AStringMapEntry
	3, // 1: com.github.thecount.protoeval.ScopeTest.a_bool_map:type_name -> com.github.thecount.protoeval.ScopeTest.ABoolMapEntry
	4, // 2: com.github.thecount.protoeval.ScopeTest.a_uint32_map:type_name -> com.github.thecount.protoeval.ScopeTest.AUint32MapEntry
	5, // 3: com.github.thecount.protoeval.ScopeTest.a_uint64_map:type_name -> com.github.thecount.protoeval.ScopeTest.AUint64MapEntry
	6, // 4: com.github.thecount.protoeval.ScopeTest.an_int32_map:type_name -> com.github.thecount.protoeval.ScopeTest.AnInt32MapEntry
	7, // 5: com.github.thecount.protoeval.ScopeTest.an_int64_map:type_name -> com.github.thecount.protoeval.ScopeTest.AnInt64MapEntry
	8, // 6: com.github.thecount.protoeval.HasTest.a_string_map:type_name -> com.github.thecount.protoeval.HasTest.AStringMapEntry
	9, // 7: com.github.thecount.protoeval.HasTest.an_int_map:type_name -> com.github.thecount.protoeval.HasTest.AnIntMapEntry
	8, // [8:8] is the sub-list for method output_type
	8, // [8:8] is the sub-list for method input_type
	8, // [8:8] is the sub-list for extension type_name
	8, // [8:8] is the sub-list for extension extendee
	0, // [0:8] is the sub-list for field type_name
}

func init() { file_test_proto_init() }
func file_test_proto_init() {
	if File_test_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_test_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ScopeTest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_test_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*HasTest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	file_test_proto_msgTypes[1].OneofWrappers = []interface{}{
		(*HasTest_FieldWithPresence)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_test_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   10,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_test_proto_goTypes,
		DependencyIndexes: file_test_proto_depIdxs,
		MessageInfos:      file_test_proto_msgTypes,
	}.Build()
	File_test_proto = out.File
	file_test_proto_rawDesc = nil
	file_test_proto_goTypes = nil
	file_test_proto_depIdxs = nil
}
