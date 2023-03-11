// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.21.12
// source: checkOnlineReq.proto

package protocol

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

// worker响应business，检查网关中是否包含某几个uniqId
type CheckOnlineReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// worker会将该值赋给router.Cmd
	RouterCmd int32 `protobuf:"varint,1,opt,name=routerCmd,proto3" json:"routerCmd,omitempty"`
	// worker会原样回传给business
	CtxData []byte `protobuf:"bytes,2,opt,name=ctxData,proto3" json:"ctxData,omitempty"`
	// 目标uniqId
	UniqIds []string `protobuf:"bytes,3,rep,name=uniqIds,proto3" json:"uniqIds,omitempty"`
}

func (x *CheckOnlineReq) Reset() {
	*x = CheckOnlineReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_checkOnlineReq_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CheckOnlineReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CheckOnlineReq) ProtoMessage() {}

func (x *CheckOnlineReq) ProtoReflect() protoreflect.Message {
	mi := &file_checkOnlineReq_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CheckOnlineReq.ProtoReflect.Descriptor instead.
func (*CheckOnlineReq) Descriptor() ([]byte, []int) {
	return file_checkOnlineReq_proto_rawDescGZIP(), []int{0}
}

func (x *CheckOnlineReq) GetRouterCmd() int32 {
	if x != nil {
		return x.RouterCmd
	}
	return 0
}

func (x *CheckOnlineReq) GetCtxData() []byte {
	if x != nil {
		return x.CtxData
	}
	return nil
}

func (x *CheckOnlineReq) GetUniqIds() []string {
	if x != nil {
		return x.UniqIds
	}
	return nil
}

var File_checkOnlineReq_proto protoreflect.FileDescriptor

var file_checkOnlineReq_proto_rawDesc = []byte{
	0x0a, 0x14, 0x63, 0x68, 0x65, 0x63, 0x6b, 0x4f, 0x6e, 0x6c, 0x69, 0x6e, 0x65, 0x52, 0x65, 0x71,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x15, 0x6e, 0x65, 0x74, 0x73, 0x76, 0x72, 0x2e, 0x63,
	0x68, 0x65, 0x63, 0x6b, 0x4f, 0x6e, 0x6c, 0x69, 0x6e, 0x65, 0x52, 0x65, 0x71, 0x22, 0x62, 0x0a,
	0x0e, 0x43, 0x68, 0x65, 0x63, 0x6b, 0x4f, 0x6e, 0x6c, 0x69, 0x6e, 0x65, 0x52, 0x65, 0x71, 0x12,
	0x1c, 0x0a, 0x09, 0x72, 0x6f, 0x75, 0x74, 0x65, 0x72, 0x43, 0x6d, 0x64, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x05, 0x52, 0x09, 0x72, 0x6f, 0x75, 0x74, 0x65, 0x72, 0x43, 0x6d, 0x64, 0x12, 0x18, 0x0a,
	0x07, 0x63, 0x74, 0x78, 0x44, 0x61, 0x74, 0x61, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x07,
	0x63, 0x74, 0x78, 0x44, 0x61, 0x74, 0x61, 0x12, 0x18, 0x0a, 0x07, 0x75, 0x6e, 0x69, 0x71, 0x49,
	0x64, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x09, 0x52, 0x07, 0x75, 0x6e, 0x69, 0x71, 0x49, 0x64,
	0x73, 0x42, 0x17, 0x5a, 0x15, 0x70, 0x6b, 0x67, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f,
	0x6c, 0x3b, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x33,
}

var (
	file_checkOnlineReq_proto_rawDescOnce sync.Once
	file_checkOnlineReq_proto_rawDescData = file_checkOnlineReq_proto_rawDesc
)

func file_checkOnlineReq_proto_rawDescGZIP() []byte {
	file_checkOnlineReq_proto_rawDescOnce.Do(func() {
		file_checkOnlineReq_proto_rawDescData = protoimpl.X.CompressGZIP(file_checkOnlineReq_proto_rawDescData)
	})
	return file_checkOnlineReq_proto_rawDescData
}

var file_checkOnlineReq_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_checkOnlineReq_proto_goTypes = []interface{}{
	(*CheckOnlineReq)(nil), // 0: netsvr.checkOnlineReq.CheckOnlineReq
}
var file_checkOnlineReq_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_checkOnlineReq_proto_init() }
func file_checkOnlineReq_proto_init() {
	if File_checkOnlineReq_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_checkOnlineReq_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CheckOnlineReq); i {
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
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_checkOnlineReq_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_checkOnlineReq_proto_goTypes,
		DependencyIndexes: file_checkOnlineReq_proto_depIdxs,
		MessageInfos:      file_checkOnlineReq_proto_msgTypes,
	}.Build()
	File_checkOnlineReq_proto = out.File
	file_checkOnlineReq_proto_rawDesc = nil
	file_checkOnlineReq_proto_goTypes = nil
	file_checkOnlineReq_proto_depIdxs = nil
}
