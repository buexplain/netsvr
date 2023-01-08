// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.21.12
// source: toServer/setUserLoginStatusOk.proto

package setUserLoginStatusOk

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

// 设置用户登录状态为成功
type SetUserLoginStatusOk struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	SessionId uint32 `protobuf:"varint,1,opt,name=sessionId,proto3" json:"sessionId,omitempty"`
	// 存储到网关的用户信息
	UserInfo []byte `protobuf:"bytes,2,opt,name=userInfo,proto3" json:"userInfo,omitempty"`
	// 转发给用户的数据
	Data []byte `protobuf:"bytes,3,opt,name=data,proto3" json:"data,omitempty"`
}

func (x *SetUserLoginStatusOk) Reset() {
	*x = SetUserLoginStatusOk{}
	if protoimpl.UnsafeEnabled {
		mi := &file_toServer_setUserLoginStatusOk_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SetUserLoginStatusOk) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SetUserLoginStatusOk) ProtoMessage() {}

func (x *SetUserLoginStatusOk) ProtoReflect() protoreflect.Message {
	mi := &file_toServer_setUserLoginStatusOk_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SetUserLoginStatusOk.ProtoReflect.Descriptor instead.
func (*SetUserLoginStatusOk) Descriptor() ([]byte, []int) {
	return file_toServer_setUserLoginStatusOk_proto_rawDescGZIP(), []int{0}
}

func (x *SetUserLoginStatusOk) GetSessionId() uint32 {
	if x != nil {
		return x.SessionId
	}
	return 0
}

func (x *SetUserLoginStatusOk) GetUserInfo() []byte {
	if x != nil {
		return x.UserInfo
	}
	return nil
}

func (x *SetUserLoginStatusOk) GetData() []byte {
	if x != nil {
		return x.Data
	}
	return nil
}

var File_toServer_setUserLoginStatusOk_proto protoreflect.FileDescriptor

var file_toServer_setUserLoginStatusOk_proto_rawDesc = []byte{
	0x0a, 0x23, 0x74, 0x6f, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x2f, 0x73, 0x65, 0x74, 0x55, 0x73,
	0x65, 0x72, 0x4c, 0x6f, 0x67, 0x69, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x4f, 0x6b, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x1c, 0x74, 0x6f, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x53,
	0x65, 0x74, 0x55, 0x73, 0x65, 0x72, 0x4c, 0x6f, 0x67, 0x69, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x75,
	0x73, 0x4f, 0x6b, 0x22, 0x64, 0x0a, 0x14, 0x53, 0x65, 0x74, 0x55, 0x73, 0x65, 0x72, 0x4c, 0x6f,
	0x67, 0x69, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x4f, 0x6b, 0x12, 0x1c, 0x0a, 0x09, 0x73,
	0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x49, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x09,
	0x73, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x49, 0x64, 0x12, 0x1a, 0x0a, 0x08, 0x75, 0x73, 0x65,
	0x72, 0x49, 0x6e, 0x66, 0x6f, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x08, 0x75, 0x73, 0x65,
	0x72, 0x49, 0x6e, 0x66, 0x6f, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x0c, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x42, 0x3f, 0x5a, 0x3d, 0x2e, 0x2f, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x2f, 0x74, 0x6f, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72,
	0x2f, 0x73, 0x65, 0x74, 0x55, 0x73, 0x65, 0x72, 0x4c, 0x6f, 0x67, 0x69, 0x6e, 0x53, 0x74, 0x61,
	0x74, 0x75, 0x73, 0x4f, 0x6b, 0x3b, 0x73, 0x65, 0x74, 0x55, 0x73, 0x65, 0x72, 0x4c, 0x6f, 0x67,
	0x69, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x4f, 0x6b, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x33,
}

var (
	file_toServer_setUserLoginStatusOk_proto_rawDescOnce sync.Once
	file_toServer_setUserLoginStatusOk_proto_rawDescData = file_toServer_setUserLoginStatusOk_proto_rawDesc
)

func file_toServer_setUserLoginStatusOk_proto_rawDescGZIP() []byte {
	file_toServer_setUserLoginStatusOk_proto_rawDescOnce.Do(func() {
		file_toServer_setUserLoginStatusOk_proto_rawDescData = protoimpl.X.CompressGZIP(file_toServer_setUserLoginStatusOk_proto_rawDescData)
	})
	return file_toServer_setUserLoginStatusOk_proto_rawDescData
}

var file_toServer_setUserLoginStatusOk_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_toServer_setUserLoginStatusOk_proto_goTypes = []interface{}{
	(*SetUserLoginStatusOk)(nil), // 0: toServerSetUserLoginStatusOk.SetUserLoginStatusOk
}
var file_toServer_setUserLoginStatusOk_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_toServer_setUserLoginStatusOk_proto_init() }
func file_toServer_setUserLoginStatusOk_proto_init() {
	if File_toServer_setUserLoginStatusOk_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_toServer_setUserLoginStatusOk_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SetUserLoginStatusOk); i {
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
			RawDescriptor: file_toServer_setUserLoginStatusOk_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_toServer_setUserLoginStatusOk_proto_goTypes,
		DependencyIndexes: file_toServer_setUserLoginStatusOk_proto_depIdxs,
		MessageInfos:      file_toServer_setUserLoginStatusOk_proto_msgTypes,
	}.Build()
	File_toServer_setUserLoginStatusOk_proto = out.File
	file_toServer_setUserLoginStatusOk_proto_rawDesc = nil
	file_toServer_setUserLoginStatusOk_proto_goTypes = nil
	file_toServer_setUserLoginStatusOk_proto_depIdxs = nil
}
