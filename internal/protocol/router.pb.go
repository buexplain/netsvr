// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.21.12
// source: router.proto

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

// worker能够处理的指令，分为三块：1、worker单向转发给business的指令，2、business单向请求worker的指令，3、business请求worker，worker再响应business的指令
type Cmd int32

const (
	// --------------------------------worker单向转发给business的指令 开始--------------------------------
	Cmd_Transfer  Cmd = 0 //透传客户数据
	Cmd_ConnClose Cmd = 1 //连接关闭
	Cmd_ConnOpen  Cmd = 2 //连接打开
	// --------------------------------business单向请求worker的指令 开始--------------------------------
	Cmd_Register     Cmd = 101 //注册business到worker，注册后，business会收到worker转发的客户信息
	Cmd_Unregister   Cmd = 102 //取消注册，取消后不会再收到客户信息
	Cmd_InfoUpdate   Cmd = 103 //更新连接的info信息
	Cmd_InfoDelete   Cmd = 104 //删除连接的info信息
	Cmd_Broadcast    Cmd = 105 //广播
	Cmd_Multicast    Cmd = 106 //组播
	Cmd_SingleCast   Cmd = 107 //单播
	Cmd_Subscribe    Cmd = 108 //订阅
	Cmd_Unsubscribe  Cmd = 109 //取消订阅
	Cmd_Publish      Cmd = 110 //发布
	Cmd_ForceOffline Cmd = 111 //强制关闭某个连接
	// --------------------------------business请求worker，worker再响应business的指令 开始--------------------------------
	Cmd_TotalUniqIdsRe      Cmd = 201 //获取网关中全部的uniqId
	Cmd_TopicUniqIdsRe      Cmd = 202 //获取网关中主题包含的uniqId
	Cmd_TopicsUniqIdCountRe Cmd = 203 //获取网关中的某几个主题的uniqId数量
	Cmd_InfoRe              Cmd = 204 //获取连接的信息
	Cmd_NetSvrStatusRe      Cmd = 205 //获取网关的状态
	Cmd_CheckOnlineRe       Cmd = 206 //判断uniqId是否在网关中
)

// Enum value maps for Cmd.
var (
	Cmd_name = map[int32]string{
		0:   "Transfer",
		1:   "ConnClose",
		2:   "ConnOpen",
		101: "Register",
		102: "Unregister",
		103: "InfoUpdate",
		104: "InfoDelete",
		105: "Broadcast",
		106: "Multicast",
		107: "SingleCast",
		108: "Subscribe",
		109: "Unsubscribe",
		110: "Publish",
		111: "ForceOffline",
		201: "TotalUniqIdsRe",
		202: "TopicUniqIdsRe",
		203: "TopicsUniqIdCountRe",
		204: "InfoRe",
		205: "NetSvrStatusRe",
		206: "CheckOnlineRe",
	}
	Cmd_value = map[string]int32{
		"Transfer":            0,
		"ConnClose":           1,
		"ConnOpen":            2,
		"Register":            101,
		"Unregister":          102,
		"InfoUpdate":          103,
		"InfoDelete":          104,
		"Broadcast":           105,
		"Multicast":           106,
		"SingleCast":          107,
		"Subscribe":           108,
		"Unsubscribe":         109,
		"Publish":             110,
		"ForceOffline":        111,
		"TotalUniqIdsRe":      201,
		"TopicUniqIdsRe":      202,
		"TopicsUniqIdCountRe": 203,
		"InfoRe":              204,
		"NetSvrStatusRe":      205,
		"CheckOnlineRe":       206,
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
	return file_router_proto_enumTypes[0].Descriptor()
}

func (Cmd) Type() protoreflect.EnumType {
	return &file_router_proto_enumTypes[0]
}

func (x Cmd) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Cmd.Descriptor instead.
func (Cmd) EnumDescriptor() ([]byte, []int) {
	return file_router_proto_rawDescGZIP(), []int{0}
}

// 路由
type Router struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// 命令
	Cmd Cmd `protobuf:"varint,1,opt,name=cmd,proto3,enum=netsvr.router.Cmd" json:"cmd,omitempty"`
	// 命令携带的数据
	Data []byte `protobuf:"bytes,2,opt,name=data,proto3" json:"data,omitempty"`
}

func (x *Router) Reset() {
	*x = Router{}
	if protoimpl.UnsafeEnabled {
		mi := &file_router_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Router) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Router) ProtoMessage() {}

func (x *Router) ProtoReflect() protoreflect.Message {
	mi := &file_router_proto_msgTypes[0]
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
	return file_router_proto_rawDescGZIP(), []int{0}
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

var File_router_proto protoreflect.FileDescriptor

var file_router_proto_rawDesc = []byte{
	0x0a, 0x0c, 0x72, 0x6f, 0x75, 0x74, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0d,
	0x6e, 0x65, 0x74, 0x73, 0x76, 0x72, 0x2e, 0x72, 0x6f, 0x75, 0x74, 0x65, 0x72, 0x22, 0x42, 0x0a,
	0x06, 0x52, 0x6f, 0x75, 0x74, 0x65, 0x72, 0x12, 0x24, 0x0a, 0x03, 0x63, 0x6d, 0x64, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0e, 0x32, 0x12, 0x2e, 0x6e, 0x65, 0x74, 0x73, 0x76, 0x72, 0x2e, 0x72, 0x6f,
	0x75, 0x74, 0x65, 0x72, 0x2e, 0x43, 0x6d, 0x64, 0x52, 0x03, 0x63, 0x6d, 0x64, 0x12, 0x12, 0x0a,
	0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x04, 0x64, 0x61, 0x74,
	0x61, 0x2a, 0xd5, 0x02, 0x0a, 0x03, 0x43, 0x6d, 0x64, 0x12, 0x0c, 0x0a, 0x08, 0x54, 0x72, 0x61,
	0x6e, 0x73, 0x66, 0x65, 0x72, 0x10, 0x00, 0x12, 0x0d, 0x0a, 0x09, 0x43, 0x6f, 0x6e, 0x6e, 0x43,
	0x6c, 0x6f, 0x73, 0x65, 0x10, 0x01, 0x12, 0x0c, 0x0a, 0x08, 0x43, 0x6f, 0x6e, 0x6e, 0x4f, 0x70,
	0x65, 0x6e, 0x10, 0x02, 0x12, 0x0c, 0x0a, 0x08, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x65, 0x72,
	0x10, 0x65, 0x12, 0x0e, 0x0a, 0x0a, 0x55, 0x6e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x65, 0x72,
	0x10, 0x66, 0x12, 0x0e, 0x0a, 0x0a, 0x49, 0x6e, 0x66, 0x6f, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65,
	0x10, 0x67, 0x12, 0x0e, 0x0a, 0x0a, 0x49, 0x6e, 0x66, 0x6f, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65,
	0x10, 0x68, 0x12, 0x0d, 0x0a, 0x09, 0x42, 0x72, 0x6f, 0x61, 0x64, 0x63, 0x61, 0x73, 0x74, 0x10,
	0x69, 0x12, 0x0d, 0x0a, 0x09, 0x4d, 0x75, 0x6c, 0x74, 0x69, 0x63, 0x61, 0x73, 0x74, 0x10, 0x6a,
	0x12, 0x0e, 0x0a, 0x0a, 0x53, 0x69, 0x6e, 0x67, 0x6c, 0x65, 0x43, 0x61, 0x73, 0x74, 0x10, 0x6b,
	0x12, 0x0d, 0x0a, 0x09, 0x53, 0x75, 0x62, 0x73, 0x63, 0x72, 0x69, 0x62, 0x65, 0x10, 0x6c, 0x12,
	0x0f, 0x0a, 0x0b, 0x55, 0x6e, 0x73, 0x75, 0x62, 0x73, 0x63, 0x72, 0x69, 0x62, 0x65, 0x10, 0x6d,
	0x12, 0x0b, 0x0a, 0x07, 0x50, 0x75, 0x62, 0x6c, 0x69, 0x73, 0x68, 0x10, 0x6e, 0x12, 0x10, 0x0a,
	0x0c, 0x46, 0x6f, 0x72, 0x63, 0x65, 0x4f, 0x66, 0x66, 0x6c, 0x69, 0x6e, 0x65, 0x10, 0x6f, 0x12,
	0x13, 0x0a, 0x0e, 0x54, 0x6f, 0x74, 0x61, 0x6c, 0x55, 0x6e, 0x69, 0x71, 0x49, 0x64, 0x73, 0x52,
	0x65, 0x10, 0xc9, 0x01, 0x12, 0x13, 0x0a, 0x0e, 0x54, 0x6f, 0x70, 0x69, 0x63, 0x55, 0x6e, 0x69,
	0x71, 0x49, 0x64, 0x73, 0x52, 0x65, 0x10, 0xca, 0x01, 0x12, 0x18, 0x0a, 0x13, 0x54, 0x6f, 0x70,
	0x69, 0x63, 0x73, 0x55, 0x6e, 0x69, 0x71, 0x49, 0x64, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x52, 0x65,
	0x10, 0xcb, 0x01, 0x12, 0x0b, 0x0a, 0x06, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x65, 0x10, 0xcc, 0x01,
	0x12, 0x13, 0x0a, 0x0e, 0x4e, 0x65, 0x74, 0x53, 0x76, 0x72, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73,
	0x52, 0x65, 0x10, 0xcd, 0x01, 0x12, 0x12, 0x0a, 0x0d, 0x43, 0x68, 0x65, 0x63, 0x6b, 0x4f, 0x6e,
	0x6c, 0x69, 0x6e, 0x65, 0x52, 0x65, 0x10, 0xce, 0x01, 0x42, 0x1c, 0x5a, 0x1a, 0x69, 0x6e, 0x74,
	0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x3b, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_router_proto_rawDescOnce sync.Once
	file_router_proto_rawDescData = file_router_proto_rawDesc
)

func file_router_proto_rawDescGZIP() []byte {
	file_router_proto_rawDescOnce.Do(func() {
		file_router_proto_rawDescData = protoimpl.X.CompressGZIP(file_router_proto_rawDescData)
	})
	return file_router_proto_rawDescData
}

var file_router_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_router_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_router_proto_goTypes = []interface{}{
	(Cmd)(0),       // 0: netsvr.router.Cmd
	(*Router)(nil), // 1: netsvr.router.Router
}
var file_router_proto_depIdxs = []int32{
	0, // 0: netsvr.router.Router.cmd:type_name -> netsvr.router.Cmd
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_router_proto_init() }
func file_router_proto_init() {
	if File_router_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_router_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
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
			RawDescriptor: file_router_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_router_proto_goTypes,
		DependencyIndexes: file_router_proto_depIdxs,
		EnumInfos:         file_router_proto_enumTypes,
		MessageInfos:      file_router_proto_msgTypes,
	}.Build()
	File_router_proto = out.File
	file_router_proto_rawDesc = nil
	file_router_proto_goTypes = nil
	file_router_proto_depIdxs = nil
}
