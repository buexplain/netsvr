// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.21.12
// source: totalUniqIdsResp.proto

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

// worker响应business，返回网关中全部的uniqId
type TotalUniqIdsResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// req（request）、resp（response）时候的上下文，worker会原样回传给business
	ReCtx *ReCtx `protobuf:"bytes,1,opt,name=reCtx,proto3" json:"reCtx,omitempty"`
	// 包含的uniqId
	UniqIds []string `protobuf:"bytes,3,rep,name=uniqIds,proto3" json:"uniqIds,omitempty"`
}

func (x *TotalUniqIdsResp) Reset() {
	*x = TotalUniqIdsResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_totalUniqIdsResp_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TotalUniqIdsResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TotalUniqIdsResp) ProtoMessage() {}

func (x *TotalUniqIdsResp) ProtoReflect() protoreflect.Message {
	mi := &file_totalUniqIdsResp_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TotalUniqIdsResp.ProtoReflect.Descriptor instead.
func (*TotalUniqIdsResp) Descriptor() ([]byte, []int) {
	return file_totalUniqIdsResp_proto_rawDescGZIP(), []int{0}
}

func (x *TotalUniqIdsResp) GetReCtx() *ReCtx {
	if x != nil {
		return x.ReCtx
	}
	return nil
}

func (x *TotalUniqIdsResp) GetUniqIds() []string {
	if x != nil {
		return x.UniqIds
	}
	return nil
}

var File_totalUniqIdsResp_proto protoreflect.FileDescriptor

var file_totalUniqIdsResp_proto_rawDesc = []byte{
	0x0a, 0x16, 0x74, 0x6f, 0x74, 0x61, 0x6c, 0x55, 0x6e, 0x69, 0x71, 0x49, 0x64, 0x73, 0x52, 0x65,
	0x73, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x17, 0x6e, 0x65, 0x74, 0x73, 0x76, 0x72,
	0x2e, 0x74, 0x6f, 0x74, 0x61, 0x6c, 0x55, 0x6e, 0x69, 0x71, 0x49, 0x64, 0x73, 0x52, 0x65, 0x73,
	0x70, 0x1a, 0x0b, 0x72, 0x65, 0x43, 0x74, 0x78, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x57,
	0x0a, 0x10, 0x54, 0x6f, 0x74, 0x61, 0x6c, 0x55, 0x6e, 0x69, 0x71, 0x49, 0x64, 0x73, 0x52, 0x65,
	0x73, 0x70, 0x12, 0x29, 0x0a, 0x05, 0x72, 0x65, 0x43, 0x74, 0x78, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x13, 0x2e, 0x6e, 0x65, 0x74, 0x73, 0x76, 0x72, 0x2e, 0x72, 0x65, 0x43, 0x74, 0x78,
	0x2e, 0x52, 0x65, 0x43, 0x74, 0x78, 0x52, 0x05, 0x72, 0x65, 0x43, 0x74, 0x78, 0x12, 0x18, 0x0a,
	0x07, 0x75, 0x6e, 0x69, 0x71, 0x49, 0x64, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x09, 0x52, 0x07,
	0x75, 0x6e, 0x69, 0x71, 0x49, 0x64, 0x73, 0x42, 0x1c, 0x5a, 0x1a, 0x69, 0x6e, 0x74, 0x65, 0x72,
	0x6e, 0x61, 0x6c, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x3b, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_totalUniqIdsResp_proto_rawDescOnce sync.Once
	file_totalUniqIdsResp_proto_rawDescData = file_totalUniqIdsResp_proto_rawDesc
)

func file_totalUniqIdsResp_proto_rawDescGZIP() []byte {
	file_totalUniqIdsResp_proto_rawDescOnce.Do(func() {
		file_totalUniqIdsResp_proto_rawDescData = protoimpl.X.CompressGZIP(file_totalUniqIdsResp_proto_rawDescData)
	})
	return file_totalUniqIdsResp_proto_rawDescData
}

var file_totalUniqIdsResp_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_totalUniqIdsResp_proto_goTypes = []interface{}{
	(*TotalUniqIdsResp)(nil), // 0: netsvr.totalUniqIdsResp.TotalUniqIdsResp
	(*ReCtx)(nil),            // 1: netsvr.reCtx.ReCtx
}
var file_totalUniqIdsResp_proto_depIdxs = []int32{
	1, // 0: netsvr.totalUniqIdsResp.TotalUniqIdsResp.reCtx:type_name -> netsvr.reCtx.ReCtx
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_totalUniqIdsResp_proto_init() }
func file_totalUniqIdsResp_proto_init() {
	if File_totalUniqIdsResp_proto != nil {
		return
	}
	file_reCtx_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_totalUniqIdsResp_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TotalUniqIdsResp); i {
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
			RawDescriptor: file_totalUniqIdsResp_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_totalUniqIdsResp_proto_goTypes,
		DependencyIndexes: file_totalUniqIdsResp_proto_depIdxs,
		MessageInfos:      file_totalUniqIdsResp_proto_msgTypes,
	}.Build()
	File_totalUniqIdsResp_proto = out.File
	file_totalUniqIdsResp_proto_rawDesc = nil
	file_totalUniqIdsResp_proto_goTypes = nil
	file_totalUniqIdsResp_proto_depIdxs = nil
}
