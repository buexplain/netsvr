// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.21.12
// source: register.proto

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

// business向worker请求，注册自己，注册成功后，当网关收到客户发送的信息的时候，会转发客户发送的信息到business
// 如果不想接收来自客户的信息，只是与网关交互，可以不发起注册指令
type Register struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// 服务编号，取值范围是 1~999，包含1与999
	// 业务层可以自己随意安排，如果多个business共用一个服务编号，则网关在数据转发的过程中是轮询转发的
	Id int32 `protobuf:"varint,1,opt,name=Id,proto3" json:"Id,omitempty"`
	// 表示该business是否处理客户端连接关闭的信息
	ProcessConnClose bool `protobuf:"varint,2,opt,name=ProcessConnClose,proto3" json:"ProcessConnClose,omitempty"`
	// 表示该business是否处理客户端连接打开的信息
	ProcessConnOpen bool `protobuf:"varint,3,opt,name=ProcessConnOpen,proto3" json:"ProcessConnOpen,omitempty"`
	// 表示接下来，需要worker开启多少协程来处理本business的请求
	// 如果本business，非常频繁的与worker交互，可以考虑开大一点，但是也不能无限大，开太多也许不能解决问题，因为发送消息到客户连接是会被阻塞的，建议100~200条左右即可
	// 请注意worker默认已经开启了一条协程来处理本business的请求，所以该值只有在大于1的时候才会开启更多协程
	ProcessCmdGoroutineNum uint32 `protobuf:"varint,4,opt,name=ProcessCmdGoroutineNum,proto3" json:"ProcessCmdGoroutineNum,omitempty"`
}

func (x *Register) Reset() {
	*x = Register{}
	if protoimpl.UnsafeEnabled {
		mi := &file_register_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Register) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Register) ProtoMessage() {}

func (x *Register) ProtoReflect() protoreflect.Message {
	mi := &file_register_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Register.ProtoReflect.Descriptor instead.
func (*Register) Descriptor() ([]byte, []int) {
	return file_register_proto_rawDescGZIP(), []int{0}
}

func (x *Register) GetId() int32 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *Register) GetProcessConnClose() bool {
	if x != nil {
		return x.ProcessConnClose
	}
	return false
}

func (x *Register) GetProcessConnOpen() bool {
	if x != nil {
		return x.ProcessConnOpen
	}
	return false
}

func (x *Register) GetProcessCmdGoroutineNum() uint32 {
	if x != nil {
		return x.ProcessCmdGoroutineNum
	}
	return 0
}

var File_register_proto protoreflect.FileDescriptor

var file_register_proto_rawDesc = []byte{
	0x0a, 0x0e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x12, 0x0f, 0x6e, 0x65, 0x74, 0x73, 0x76, 0x72, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x65,
	0x72, 0x22, 0xa8, 0x01, 0x0a, 0x08, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x65, 0x72, 0x12, 0x0e,
	0x0a, 0x02, 0x49, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x02, 0x49, 0x64, 0x12, 0x2a,
	0x0a, 0x10, 0x50, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x43, 0x6f, 0x6e, 0x6e, 0x43, 0x6c, 0x6f,
	0x73, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x08, 0x52, 0x10, 0x50, 0x72, 0x6f, 0x63, 0x65, 0x73,
	0x73, 0x43, 0x6f, 0x6e, 0x6e, 0x43, 0x6c, 0x6f, 0x73, 0x65, 0x12, 0x28, 0x0a, 0x0f, 0x50, 0x72,
	0x6f, 0x63, 0x65, 0x73, 0x73, 0x43, 0x6f, 0x6e, 0x6e, 0x4f, 0x70, 0x65, 0x6e, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x08, 0x52, 0x0f, 0x50, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x43, 0x6f, 0x6e, 0x6e,
	0x4f, 0x70, 0x65, 0x6e, 0x12, 0x36, 0x0a, 0x16, 0x50, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x43,
	0x6d, 0x64, 0x47, 0x6f, 0x72, 0x6f, 0x75, 0x74, 0x69, 0x6e, 0x65, 0x4e, 0x75, 0x6d, 0x18, 0x04,
	0x20, 0x01, 0x28, 0x0d, 0x52, 0x16, 0x50, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x43, 0x6d, 0x64,
	0x47, 0x6f, 0x72, 0x6f, 0x75, 0x74, 0x69, 0x6e, 0x65, 0x4e, 0x75, 0x6d, 0x42, 0x17, 0x5a, 0x15,
	0x70, 0x6b, 0x67, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x3b, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_register_proto_rawDescOnce sync.Once
	file_register_proto_rawDescData = file_register_proto_rawDesc
)

func file_register_proto_rawDescGZIP() []byte {
	file_register_proto_rawDescOnce.Do(func() {
		file_register_proto_rawDescData = protoimpl.X.CompressGZIP(file_register_proto_rawDescData)
	})
	return file_register_proto_rawDescData
}

var file_register_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_register_proto_goTypes = []interface{}{
	(*Register)(nil), // 0: netsvr.register.Register
}
var file_register_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_register_proto_init() }
func file_register_proto_init() {
	if File_register_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_register_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Register); i {
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
			RawDescriptor: file_register_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_register_proto_goTypes,
		DependencyIndexes: file_register_proto_depIdxs,
		MessageInfos:      file_register_proto_msgTypes,
	}.Build()
	File_register_proto = out.File
	file_register_proto_rawDesc = nil
	file_register_proto_goTypes = nil
	file_register_proto_depIdxs = nil
}
