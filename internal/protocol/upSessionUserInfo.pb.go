// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.21.12
// source: upSessionUserInfo.proto

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

// business向worker请求，将客户信息存储到网关连接的session中，只有登录成功的连接才会进行设置，因为没登录的，在后续登录成功后也会设置一次
type UpSessionUserInfo struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// 连接的session id
	SessionId uint32 `protobuf:"varint,1,opt,name=sessionId,proto3" json:"sessionId,omitempty"`
	// 存储到网关的客户信息
	UserInfo string `protobuf:"bytes,3,opt,name=userInfo,proto3" json:"userInfo,omitempty"`
	// 客户在业务系统中的唯一id，当该值与网关中现有的值不一致的时候，网关中的客户信息是不会被更改的
	// 如此设定是为了避免，session id被顶号的问题，如果没有，可以不传
	UserId string `protobuf:"bytes,4,opt,name=userId,proto3" json:"userId,omitempty"`
	// 需要发给客户的数据，如果没有，可以不传
	Data []byte `protobuf:"bytes,5,opt,name=data,proto3" json:"data,omitempty"`
}

func (x *UpSessionUserInfo) Reset() {
	*x = UpSessionUserInfo{}
	if protoimpl.UnsafeEnabled {
		mi := &file_upSessionUserInfo_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *UpSessionUserInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UpSessionUserInfo) ProtoMessage() {}

func (x *UpSessionUserInfo) ProtoReflect() protoreflect.Message {
	mi := &file_upSessionUserInfo_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UpSessionUserInfo.ProtoReflect.Descriptor instead.
func (*UpSessionUserInfo) Descriptor() ([]byte, []int) {
	return file_upSessionUserInfo_proto_rawDescGZIP(), []int{0}
}

func (x *UpSessionUserInfo) GetSessionId() uint32 {
	if x != nil {
		return x.SessionId
	}
	return 0
}

func (x *UpSessionUserInfo) GetUserInfo() string {
	if x != nil {
		return x.UserInfo
	}
	return ""
}

func (x *UpSessionUserInfo) GetUserId() string {
	if x != nil {
		return x.UserId
	}
	return ""
}

func (x *UpSessionUserInfo) GetData() []byte {
	if x != nil {
		return x.Data
	}
	return nil
}

var File_upSessionUserInfo_proto protoreflect.FileDescriptor

var file_upSessionUserInfo_proto_rawDesc = []byte{
	0x0a, 0x17, 0x75, 0x70, 0x53, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x55, 0x73, 0x65, 0x72, 0x49,
	0x6e, 0x66, 0x6f, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x15, 0x6e, 0x65, 0x74, 0x73, 0x76,
	0x72, 0x2e, 0x73, 0x65, 0x74, 0x53, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x55, 0x73, 0x65, 0x72,
	0x22, 0x79, 0x0a, 0x11, 0x55, 0x70, 0x53, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x55, 0x73, 0x65,
	0x72, 0x49, 0x6e, 0x66, 0x6f, 0x12, 0x1c, 0x0a, 0x09, 0x73, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e,
	0x49, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x09, 0x73, 0x65, 0x73, 0x73, 0x69, 0x6f,
	0x6e, 0x49, 0x64, 0x12, 0x1a, 0x0a, 0x08, 0x75, 0x73, 0x65, 0x72, 0x49, 0x6e, 0x66, 0x6f, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x75, 0x73, 0x65, 0x72, 0x49, 0x6e, 0x66, 0x6f, 0x12,
	0x16, 0x0a, 0x06, 0x75, 0x73, 0x65, 0x72, 0x49, 0x64, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x06, 0x75, 0x73, 0x65, 0x72, 0x49, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18,
	0x05, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x42, 0x1c, 0x5a, 0x1a, 0x69,
	0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c,
	0x3b, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x33,
}

var (
	file_upSessionUserInfo_proto_rawDescOnce sync.Once
	file_upSessionUserInfo_proto_rawDescData = file_upSessionUserInfo_proto_rawDesc
)

func file_upSessionUserInfo_proto_rawDescGZIP() []byte {
	file_upSessionUserInfo_proto_rawDescOnce.Do(func() {
		file_upSessionUserInfo_proto_rawDescData = protoimpl.X.CompressGZIP(file_upSessionUserInfo_proto_rawDescData)
	})
	return file_upSessionUserInfo_proto_rawDescData
}

var file_upSessionUserInfo_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_upSessionUserInfo_proto_goTypes = []interface{}{
	(*UpSessionUserInfo)(nil), // 0: netsvr.setSessionUser.UpSessionUserInfo
}
var file_upSessionUserInfo_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_upSessionUserInfo_proto_init() }
func file_upSessionUserInfo_proto_init() {
	if File_upSessionUserInfo_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_upSessionUserInfo_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*UpSessionUserInfo); i {
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
			RawDescriptor: file_upSessionUserInfo_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_upSessionUserInfo_proto_goTypes,
		DependencyIndexes: file_upSessionUserInfo_proto_depIdxs,
		MessageInfos:      file_upSessionUserInfo_proto_msgTypes,
	}.Build()
	File_upSessionUserInfo_proto = out.File
	file_upSessionUserInfo_proto_rawDesc = nil
	file_upSessionUserInfo_proto_goTypes = nil
	file_upSessionUserInfo_proto_depIdxs = nil
}
