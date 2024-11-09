//*
// Copyright 2023 buexplain@qq.com
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.21.12
// source: singleCastBulkByCustomerId.proto

package netsvr

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

// business向worker请求，进行批量单播
// 网关必须实现以下三种处理：
// 1.当业务进程传递的customerIds的customerId数量与data的datum数量一致时，网关必须将同一下标的datum，发送给同一下标的customerId
// 2.当业务进程传递的customerIds的customerId数量只有一个，data的datum数量是一个以上时，网关必须将所有的datum都发送给这个customerId
// 3.除以上两种情况外，其它情况都丢弃不做处理
type SingleCastBulkByCustomerId struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// 目标customerId
	CustomerIds []string `protobuf:"bytes,1,rep,name=customerIds,proto3" json:"customerIds,omitempty"`
	// 目标customerId的数据
	Data [][]byte `protobuf:"bytes,2,rep,name=data,proto3" json:"data,omitempty"`
}

func (x *SingleCastBulkByCustomerId) Reset() {
	*x = SingleCastBulkByCustomerId{}
	if protoimpl.UnsafeEnabled {
		mi := &file_singleCastBulkByCustomerId_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SingleCastBulkByCustomerId) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SingleCastBulkByCustomerId) ProtoMessage() {}

func (x *SingleCastBulkByCustomerId) ProtoReflect() protoreflect.Message {
	mi := &file_singleCastBulkByCustomerId_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SingleCastBulkByCustomerId.ProtoReflect.Descriptor instead.
func (*SingleCastBulkByCustomerId) Descriptor() ([]byte, []int) {
	return file_singleCastBulkByCustomerId_proto_rawDescGZIP(), []int{0}
}

func (x *SingleCastBulkByCustomerId) GetCustomerIds() []string {
	if x != nil {
		return x.CustomerIds
	}
	return nil
}

func (x *SingleCastBulkByCustomerId) GetData() [][]byte {
	if x != nil {
		return x.Data
	}
	return nil
}

var File_singleCastBulkByCustomerId_proto protoreflect.FileDescriptor

var file_singleCastBulkByCustomerId_proto_rawDesc = []byte{
	0x0a, 0x20, 0x73, 0x69, 0x6e, 0x67, 0x6c, 0x65, 0x43, 0x61, 0x73, 0x74, 0x42, 0x75, 0x6c, 0x6b,
	0x42, 0x79, 0x43, 0x75, 0x73, 0x74, 0x6f, 0x6d, 0x65, 0x72, 0x49, 0x64, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x12, 0x21, 0x6e, 0x65, 0x74, 0x73, 0x76, 0x72, 0x2e, 0x73, 0x69, 0x6e, 0x67, 0x6c,
	0x65, 0x43, 0x61, 0x73, 0x74, 0x42, 0x75, 0x6c, 0x6b, 0x42, 0x79, 0x43, 0x75, 0x73, 0x74, 0x6f,
	0x6d, 0x65, 0x72, 0x49, 0x64, 0x22, 0x52, 0x0a, 0x1a, 0x53, 0x69, 0x6e, 0x67, 0x6c, 0x65, 0x43,
	0x61, 0x73, 0x74, 0x42, 0x75, 0x6c, 0x6b, 0x42, 0x79, 0x43, 0x75, 0x73, 0x74, 0x6f, 0x6d, 0x65,
	0x72, 0x49, 0x64, 0x12, 0x20, 0x0a, 0x0b, 0x63, 0x75, 0x73, 0x74, 0x6f, 0x6d, 0x65, 0x72, 0x49,
	0x64, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x09, 0x52, 0x0b, 0x63, 0x75, 0x73, 0x74, 0x6f, 0x6d,
	0x65, 0x72, 0x49, 0x64, 0x73, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x02, 0x20,
	0x03, 0x28, 0x0c, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x42, 0x27, 0x5a, 0x07, 0x6e, 0x65, 0x74,
	0x73, 0x76, 0x72, 0x2f, 0xca, 0x02, 0x06, 0x4e, 0x65, 0x74, 0x73, 0x76, 0x72, 0xe2, 0x02, 0x12,
	0x4e, 0x65, 0x74, 0x73, 0x76, 0x72, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61,
	0x74, 0x61, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_singleCastBulkByCustomerId_proto_rawDescOnce sync.Once
	file_singleCastBulkByCustomerId_proto_rawDescData = file_singleCastBulkByCustomerId_proto_rawDesc
)

func file_singleCastBulkByCustomerId_proto_rawDescGZIP() []byte {
	file_singleCastBulkByCustomerId_proto_rawDescOnce.Do(func() {
		file_singleCastBulkByCustomerId_proto_rawDescData = protoimpl.X.CompressGZIP(file_singleCastBulkByCustomerId_proto_rawDescData)
	})
	return file_singleCastBulkByCustomerId_proto_rawDescData
}

var file_singleCastBulkByCustomerId_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_singleCastBulkByCustomerId_proto_goTypes = []interface{}{
	(*SingleCastBulkByCustomerId)(nil), // 0: netsvr.singleCastBulkByCustomerId.SingleCastBulkByCustomerId
}
var file_singleCastBulkByCustomerId_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_singleCastBulkByCustomerId_proto_init() }
func file_singleCastBulkByCustomerId_proto_init() {
	if File_singleCastBulkByCustomerId_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_singleCastBulkByCustomerId_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SingleCastBulkByCustomerId); i {
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
			RawDescriptor: file_singleCastBulkByCustomerId_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_singleCastBulkByCustomerId_proto_goTypes,
		DependencyIndexes: file_singleCastBulkByCustomerId_proto_depIdxs,
		MessageInfos:      file_singleCastBulkByCustomerId_proto_msgTypes,
	}.Build()
	File_singleCastBulkByCustomerId_proto = out.File
	file_singleCastBulkByCustomerId_proto_rawDesc = nil
	file_singleCastBulkByCustomerId_proto_goTypes = nil
	file_singleCastBulkByCustomerId_proto_depIdxs = nil
}
