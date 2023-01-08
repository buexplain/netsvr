// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.21.12
// source: toServer/subscribe.proto

package subscribe

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

// 订阅
type Subscribe struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	SessionId uint32 `protobuf:"varint,1,opt,name=sessionId,proto3" json:"sessionId,omitempty"`
	Topic     string `protobuf:"bytes,2,opt,name=topic,proto3" json:"topic,omitempty"`
}

func (x *Subscribe) Reset() {
	*x = Subscribe{}
	if protoimpl.UnsafeEnabled {
		mi := &file_toServer_subscribe_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Subscribe) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Subscribe) ProtoMessage() {}

func (x *Subscribe) ProtoReflect() protoreflect.Message {
	mi := &file_toServer_subscribe_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Subscribe.ProtoReflect.Descriptor instead.
func (*Subscribe) Descriptor() ([]byte, []int) {
	return file_toServer_subscribe_proto_rawDescGZIP(), []int{0}
}

func (x *Subscribe) GetSessionId() uint32 {
	if x != nil {
		return x.SessionId
	}
	return 0
}

func (x *Subscribe) GetTopic() string {
	if x != nil {
		return x.Topic
	}
	return ""
}

var File_toServer_subscribe_proto protoreflect.FileDescriptor

var file_toServer_subscribe_proto_rawDesc = []byte{
	0x0a, 0x18, 0x74, 0x6f, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x2f, 0x73, 0x75, 0x62, 0x73, 0x63,
	0x72, 0x69, 0x62, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x11, 0x74, 0x6f, 0x53, 0x65,
	0x72, 0x76, 0x65, 0x72, 0x53, 0x75, 0x62, 0x73, 0x63, 0x72, 0x69, 0x62, 0x65, 0x22, 0x3f, 0x0a,
	0x09, 0x53, 0x75, 0x62, 0x73, 0x63, 0x72, 0x69, 0x62, 0x65, 0x12, 0x1c, 0x0a, 0x09, 0x73, 0x65,
	0x73, 0x73, 0x69, 0x6f, 0x6e, 0x49, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x09, 0x73,
	0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x49, 0x64, 0x12, 0x14, 0x0a, 0x05, 0x74, 0x6f, 0x70, 0x69,
	0x63, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x74, 0x6f, 0x70, 0x69, 0x63, 0x42, 0x29,
	0x5a, 0x27, 0x2e, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x2f, 0x74, 0x6f, 0x53,
	0x65, 0x72, 0x76, 0x65, 0x72, 0x2f, 0x73, 0x75, 0x62, 0x73, 0x63, 0x72, 0x69, 0x62, 0x65, 0x3b,
	0x73, 0x75, 0x62, 0x73, 0x63, 0x72, 0x69, 0x62, 0x65, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x33,
}

var (
	file_toServer_subscribe_proto_rawDescOnce sync.Once
	file_toServer_subscribe_proto_rawDescData = file_toServer_subscribe_proto_rawDesc
)

func file_toServer_subscribe_proto_rawDescGZIP() []byte {
	file_toServer_subscribe_proto_rawDescOnce.Do(func() {
		file_toServer_subscribe_proto_rawDescData = protoimpl.X.CompressGZIP(file_toServer_subscribe_proto_rawDescData)
	})
	return file_toServer_subscribe_proto_rawDescData
}

var file_toServer_subscribe_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_toServer_subscribe_proto_goTypes = []interface{}{
	(*Subscribe)(nil), // 0: toServerSubscribe.Subscribe
}
var file_toServer_subscribe_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_toServer_subscribe_proto_init() }
func file_toServer_subscribe_proto_init() {
	if File_toServer_subscribe_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_toServer_subscribe_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Subscribe); i {
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
			RawDescriptor: file_toServer_subscribe_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_toServer_subscribe_proto_goTypes,
		DependencyIndexes: file_toServer_subscribe_proto_depIdxs,
		MessageInfos:      file_toServer_subscribe_proto_msgTypes,
	}.Build()
	File_toServer_subscribe_proto = out.File
	file_toServer_subscribe_proto_rawDesc = nil
	file_toServer_subscribe_proto_goTypes = nil
	file_toServer_subscribe_proto_depIdxs = nil
}