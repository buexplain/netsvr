// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.21.12
// source: infoDelete.proto

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

// 删除连接的info信息
type InfoDelete struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// 目标uniqId
	UniqId string `protobuf:"bytes,1,opt,name=uniqId,proto3" json:"uniqId,omitempty"`
	// 是否删除uniqId，true：重新随机生成一个uniqId，false：不处理
	DelUniqId bool `protobuf:"varint,2,opt,name=delUniqId,proto3" json:"delUniqId,omitempty"`
	// 是否删除session，true：设置session为空字符串，false：不处理
	DelSession bool `protobuf:"varint,3,opt,name=delSession,proto3" json:"delSession,omitempty"`
	// 是否删除topic，true：设置topic为空[]string，false：不处理
	DelTopic bool `protobuf:"varint,4,opt,name=delTopic,proto3" json:"delTopic,omitempty"`
	// 需要发给客户的数据，传递了则转发给客户
	Data []byte `protobuf:"bytes,5,opt,name=data,proto3" json:"data,omitempty"`
}

func (x *InfoDelete) Reset() {
	*x = InfoDelete{}
	if protoimpl.UnsafeEnabled {
		mi := &file_infoDelete_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *InfoDelete) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*InfoDelete) ProtoMessage() {}

func (x *InfoDelete) ProtoReflect() protoreflect.Message {
	mi := &file_infoDelete_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use InfoDelete.ProtoReflect.Descriptor instead.
func (*InfoDelete) Descriptor() ([]byte, []int) {
	return file_infoDelete_proto_rawDescGZIP(), []int{0}
}

func (x *InfoDelete) GetUniqId() string {
	if x != nil {
		return x.UniqId
	}
	return ""
}

func (x *InfoDelete) GetDelUniqId() bool {
	if x != nil {
		return x.DelUniqId
	}
	return false
}

func (x *InfoDelete) GetDelSession() bool {
	if x != nil {
		return x.DelSession
	}
	return false
}

func (x *InfoDelete) GetDelTopic() bool {
	if x != nil {
		return x.DelTopic
	}
	return false
}

func (x *InfoDelete) GetData() []byte {
	if x != nil {
		return x.Data
	}
	return nil
}

var File_infoDelete_proto protoreflect.FileDescriptor

var file_infoDelete_proto_rawDesc = []byte{
	0x0a, 0x10, 0x69, 0x6e, 0x66, 0x6f, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x12, 0x11, 0x6e, 0x65, 0x74, 0x73, 0x76, 0x72, 0x2e, 0x69, 0x6e, 0x66, 0x6f, 0x44,
	0x65, 0x6c, 0x65, 0x74, 0x65, 0x22, 0x92, 0x01, 0x0a, 0x0a, 0x49, 0x6e, 0x66, 0x6f, 0x44, 0x65,
	0x6c, 0x65, 0x74, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x75, 0x6e, 0x69, 0x71, 0x49, 0x64, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x75, 0x6e, 0x69, 0x71, 0x49, 0x64, 0x12, 0x1c, 0x0a, 0x09,
	0x64, 0x65, 0x6c, 0x55, 0x6e, 0x69, 0x71, 0x49, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x08, 0x52,
	0x09, 0x64, 0x65, 0x6c, 0x55, 0x6e, 0x69, 0x71, 0x49, 0x64, 0x12, 0x1e, 0x0a, 0x0a, 0x64, 0x65,
	0x6c, 0x53, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x03, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0a,
	0x64, 0x65, 0x6c, 0x53, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x1a, 0x0a, 0x08, 0x64, 0x65,
	0x6c, 0x54, 0x6f, 0x70, 0x69, 0x63, 0x18, 0x04, 0x20, 0x01, 0x28, 0x08, 0x52, 0x08, 0x64, 0x65,
	0x6c, 0x54, 0x6f, 0x70, 0x69, 0x63, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x05,
	0x20, 0x01, 0x28, 0x0c, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x42, 0x17, 0x5a, 0x15, 0x70, 0x6b,
	0x67, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x3b, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x63, 0x6f, 0x6c, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_infoDelete_proto_rawDescOnce sync.Once
	file_infoDelete_proto_rawDescData = file_infoDelete_proto_rawDesc
)

func file_infoDelete_proto_rawDescGZIP() []byte {
	file_infoDelete_proto_rawDescOnce.Do(func() {
		file_infoDelete_proto_rawDescData = protoimpl.X.CompressGZIP(file_infoDelete_proto_rawDescData)
	})
	return file_infoDelete_proto_rawDescData
}

var file_infoDelete_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_infoDelete_proto_goTypes = []interface{}{
	(*InfoDelete)(nil), // 0: netsvr.infoDelete.InfoDelete
}
var file_infoDelete_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_infoDelete_proto_init() }
func file_infoDelete_proto_init() {
	if File_infoDelete_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_infoDelete_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*InfoDelete); i {
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
			RawDescriptor: file_infoDelete_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_infoDelete_proto_goTypes,
		DependencyIndexes: file_infoDelete_proto_depIdxs,
		MessageInfos:      file_infoDelete_proto_msgTypes,
	}.Build()
	File_infoDelete_proto = out.File
	file_infoDelete_proto_rawDesc = nil
	file_infoDelete_proto_goTypes = nil
	file_infoDelete_proto_depIdxs = nil
}
