// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.21.12
// source: topicUniqIdListReq.proto

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

// business向worker请求，获取网关中某个主题包含的uniqId
type TopicUniqIdListReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// worker会将该值赋给router.Cmd
	RouterCmd int32 `protobuf:"varint,1,opt,name=routerCmd,proto3" json:"routerCmd,omitempty"`
	// worker会原样回传给business
	CtxData []byte `protobuf:"bytes,2,opt,name=ctxData,proto3" json:"ctxData,omitempty"`
	// 目标主题
	Topic string `protobuf:"bytes,3,opt,name=topic,proto3" json:"topic,omitempty"`
}

func (x *TopicUniqIdListReq) Reset() {
	*x = TopicUniqIdListReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_topicUniqIdListReq_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TopicUniqIdListReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TopicUniqIdListReq) ProtoMessage() {}

func (x *TopicUniqIdListReq) ProtoReflect() protoreflect.Message {
	mi := &file_topicUniqIdListReq_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TopicUniqIdListReq.ProtoReflect.Descriptor instead.
func (*TopicUniqIdListReq) Descriptor() ([]byte, []int) {
	return file_topicUniqIdListReq_proto_rawDescGZIP(), []int{0}
}

func (x *TopicUniqIdListReq) GetRouterCmd() int32 {
	if x != nil {
		return x.RouterCmd
	}
	return 0
}

func (x *TopicUniqIdListReq) GetCtxData() []byte {
	if x != nil {
		return x.CtxData
	}
	return nil
}

func (x *TopicUniqIdListReq) GetTopic() string {
	if x != nil {
		return x.Topic
	}
	return ""
}

var File_topicUniqIdListReq_proto protoreflect.FileDescriptor

var file_topicUniqIdListReq_proto_rawDesc = []byte{
	0x0a, 0x18, 0x74, 0x6f, 0x70, 0x69, 0x63, 0x55, 0x6e, 0x69, 0x71, 0x49, 0x64, 0x4c, 0x69, 0x73,
	0x74, 0x52, 0x65, 0x71, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x19, 0x6e, 0x65, 0x74, 0x73,
	0x76, 0x72, 0x2e, 0x74, 0x6f, 0x70, 0x69, 0x63, 0x55, 0x6e, 0x69, 0x71, 0x49, 0x64, 0x4c, 0x69,
	0x73, 0x74, 0x52, 0x65, 0x71, 0x22, 0x62, 0x0a, 0x12, 0x54, 0x6f, 0x70, 0x69, 0x63, 0x55, 0x6e,
	0x69, 0x71, 0x49, 0x64, 0x4c, 0x69, 0x73, 0x74, 0x52, 0x65, 0x71, 0x12, 0x1c, 0x0a, 0x09, 0x72,
	0x6f, 0x75, 0x74, 0x65, 0x72, 0x43, 0x6d, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x09,
	0x72, 0x6f, 0x75, 0x74, 0x65, 0x72, 0x43, 0x6d, 0x64, 0x12, 0x18, 0x0a, 0x07, 0x63, 0x74, 0x78,
	0x44, 0x61, 0x74, 0x61, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x07, 0x63, 0x74, 0x78, 0x44,
	0x61, 0x74, 0x61, 0x12, 0x14, 0x0a, 0x05, 0x74, 0x6f, 0x70, 0x69, 0x63, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x05, 0x74, 0x6f, 0x70, 0x69, 0x63, 0x42, 0x17, 0x5a, 0x15, 0x70, 0x6b, 0x67,
	0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x3b, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63,
	0x6f, 0x6c, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_topicUniqIdListReq_proto_rawDescOnce sync.Once
	file_topicUniqIdListReq_proto_rawDescData = file_topicUniqIdListReq_proto_rawDesc
)

func file_topicUniqIdListReq_proto_rawDescGZIP() []byte {
	file_topicUniqIdListReq_proto_rawDescOnce.Do(func() {
		file_topicUniqIdListReq_proto_rawDescData = protoimpl.X.CompressGZIP(file_topicUniqIdListReq_proto_rawDescData)
	})
	return file_topicUniqIdListReq_proto_rawDescData
}

var file_topicUniqIdListReq_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_topicUniqIdListReq_proto_goTypes = []interface{}{
	(*TopicUniqIdListReq)(nil), // 0: netsvr.topicUniqIdListReq.TopicUniqIdListReq
}
var file_topicUniqIdListReq_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_topicUniqIdListReq_proto_init() }
func file_topicUniqIdListReq_proto_init() {
	if File_topicUniqIdListReq_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_topicUniqIdListReq_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TopicUniqIdListReq); i {
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
			RawDescriptor: file_topicUniqIdListReq_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_topicUniqIdListReq_proto_goTypes,
		DependencyIndexes: file_topicUniqIdListReq_proto_depIdxs,
		MessageInfos:      file_topicUniqIdListReq_proto_msgTypes,
	}.Build()
	File_topicUniqIdListReq_proto = out.File
	file_topicUniqIdListReq_proto_rawDesc = nil
	file_topicUniqIdListReq_proto_goTypes = nil
	file_topicUniqIdListReq_proto_depIdxs = nil
}
