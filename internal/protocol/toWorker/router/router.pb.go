// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.21.12
// source: toWorker/router.proto

package router

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

// 服务器发给工作进程的各种指令
type Cmd int32

const (
	// 透传用户数据
	Cmd_Transfer Cmd = 0
	// 连接关闭
	Cmd_ConnClose Cmd = 1
	// 连接打开
	Cmd_ConnOpen Cmd = 2
	// 返回网关中全部的session id
	Cmd_RespTotalSessionId Cmd = 3
	// 返回某几个主题的session id到工作进程
	Cmd_RespTopicsSessionId Cmd = 4
	// 返回session信息到工作进程
	Cmd_RespSessionInfo Cmd = 5
)

// Enum value maps for Cmd.
var (
	Cmd_name = map[int32]string{
		0: "Transfer",
		1: "ConnClose",
		2: "ConnOpen",
		3: "RespTotalSessionId",
		4: "RespTopicsSessionId",
		5: "RespSessionInfo",
	}
	Cmd_value = map[string]int32{
		"Transfer":            0,
		"ConnClose":           1,
		"ConnOpen":            2,
		"RespTotalSessionId":  3,
		"RespTopicsSessionId": 4,
		"RespSessionInfo":     5,
	}
)

func (x Cmd) Enum() *Cmd {
	p := new(Cmd)
	*p = x
	return p
}

func (x Cmd) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Cmd) Descriptor() protoreflect.EnumDescriptor {
	return file_toWorker_router_proto_enumTypes[0].Descriptor()
}

func (Cmd) Type() protoreflect.EnumType {
	return &file_toWorker_router_proto_enumTypes[0]
}

func (x Cmd) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Cmd.Descriptor instead.
func (Cmd) EnumDescriptor() ([]byte, []int) {
	return file_toWorker_router_proto_rawDescGZIP(), []int{0}
}

// 路由
type Router struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// 命令
	Cmd Cmd `protobuf:"varint,1,opt,name=cmd,proto3,enum=toWorkerRouter.Cmd" json:"cmd,omitempty"`
	// 命令携带的数据
	Data []byte `protobuf:"bytes,2,opt,name=data,proto3" json:"data,omitempty"`
}

func (x *Router) Reset() {
	*x = Router{}
	if protoimpl.UnsafeEnabled {
		mi := &file_toWorker_router_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Router) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Router) ProtoMessage() {}

func (x *Router) ProtoReflect() protoreflect.Message {
	mi := &file_toWorker_router_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Router.ProtoReflect.Descriptor instead.
func (*Router) Descriptor() ([]byte, []int) {
	return file_toWorker_router_proto_rawDescGZIP(), []int{0}
}

func (x *Router) GetCmd() Cmd {
	if x != nil {
		return x.Cmd
	}
	return Cmd_Transfer
}

func (x *Router) GetData() []byte {
	if x != nil {
		return x.Data
	}
	return nil
}

var File_toWorker_router_proto protoreflect.FileDescriptor

var file_toWorker_router_proto_rawDesc = []byte{
	0x0a, 0x15, 0x74, 0x6f, 0x57, 0x6f, 0x72, 0x6b, 0x65, 0x72, 0x2f, 0x72, 0x6f, 0x75, 0x74, 0x65,
	0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0e, 0x74, 0x6f, 0x57, 0x6f, 0x72, 0x6b, 0x65,
	0x72, 0x52, 0x6f, 0x75, 0x74, 0x65, 0x72, 0x22, 0x43, 0x0a, 0x06, 0x52, 0x6f, 0x75, 0x74, 0x65,
	0x72, 0x12, 0x25, 0x0a, 0x03, 0x63, 0x6d, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x13,
	0x2e, 0x74, 0x6f, 0x57, 0x6f, 0x72, 0x6b, 0x65, 0x72, 0x52, 0x6f, 0x75, 0x74, 0x65, 0x72, 0x2e,
	0x43, 0x6d, 0x64, 0x52, 0x03, 0x63, 0x6d, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x2a, 0x76, 0x0a, 0x03,
	0x43, 0x6d, 0x64, 0x12, 0x0c, 0x0a, 0x08, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x66, 0x65, 0x72, 0x10,
	0x00, 0x12, 0x0d, 0x0a, 0x09, 0x43, 0x6f, 0x6e, 0x6e, 0x43, 0x6c, 0x6f, 0x73, 0x65, 0x10, 0x01,
	0x12, 0x0c, 0x0a, 0x08, 0x43, 0x6f, 0x6e, 0x6e, 0x4f, 0x70, 0x65, 0x6e, 0x10, 0x02, 0x12, 0x16,
	0x0a, 0x12, 0x52, 0x65, 0x73, 0x70, 0x54, 0x6f, 0x74, 0x61, 0x6c, 0x53, 0x65, 0x73, 0x73, 0x69,
	0x6f, 0x6e, 0x49, 0x64, 0x10, 0x03, 0x12, 0x17, 0x0a, 0x13, 0x52, 0x65, 0x73, 0x70, 0x54, 0x6f,
	0x70, 0x69, 0x63, 0x73, 0x53, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x49, 0x64, 0x10, 0x04, 0x12,
	0x13, 0x0a, 0x0f, 0x52, 0x65, 0x73, 0x70, 0x53, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x49, 0x6e,
	0x66, 0x6f, 0x10, 0x05, 0x42, 0x23, 0x5a, 0x21, 0x2e, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63,
	0x6f, 0x6c, 0x2f, 0x74, 0x6f, 0x57, 0x6f, 0x72, 0x6b, 0x65, 0x72, 0x2f, 0x72, 0x6f, 0x75, 0x74,
	0x65, 0x72, 0x3b, 0x72, 0x6f, 0x75, 0x74, 0x65, 0x72, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x33,
}

var (
	file_toWorker_router_proto_rawDescOnce sync.Once
	file_toWorker_router_proto_rawDescData = file_toWorker_router_proto_rawDesc
)

func file_toWorker_router_proto_rawDescGZIP() []byte {
	file_toWorker_router_proto_rawDescOnce.Do(func() {
		file_toWorker_router_proto_rawDescData = protoimpl.X.CompressGZIP(file_toWorker_router_proto_rawDescData)
	})
	return file_toWorker_router_proto_rawDescData
}

var file_toWorker_router_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_toWorker_router_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_toWorker_router_proto_goTypes = []interface{}{
	(Cmd)(0),       // 0: toWorkerRouter.Cmd
	(*Router)(nil), // 1: toWorkerRouter.Router
}
var file_toWorker_router_proto_depIdxs = []int32{
	0, // 0: toWorkerRouter.Router.cmd:type_name -> toWorkerRouter.Cmd
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_toWorker_router_proto_init() }
func file_toWorker_router_proto_init() {
	if File_toWorker_router_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_toWorker_router_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Router); i {
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
			RawDescriptor: file_toWorker_router_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_toWorker_router_proto_goTypes,
		DependencyIndexes: file_toWorker_router_proto_depIdxs,
		EnumInfos:         file_toWorker_router_proto_enumTypes,
		MessageInfos:      file_toWorker_router_proto_msgTypes,
	}.Build()
	File_toWorker_router_proto = out.File
	file_toWorker_router_proto_rawDesc = nil
	file_toWorker_router_proto_goTypes = nil
	file_toWorker_router_proto_depIdxs = nil
}
