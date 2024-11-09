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
// source: unRegisterReq.proto

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

// business向worker请求，取消一个已经被注册的连接
type UnRegisterReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// 连接id
	ConnId string `protobuf:"bytes,1,opt,name=connId,proto3" json:"connId,omitempty"`
}

func (x *UnRegisterReq) Reset() {
	*x = UnRegisterReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_unRegisterReq_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *UnRegisterReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UnRegisterReq) ProtoMessage() {}

func (x *UnRegisterReq) ProtoReflect() protoreflect.Message {
	mi := &file_unRegisterReq_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UnRegisterReq.ProtoReflect.Descriptor instead.
func (*UnRegisterReq) Descriptor() ([]byte, []int) {
	return file_unRegisterReq_proto_rawDescGZIP(), []int{0}
}

func (x *UnRegisterReq) GetConnId() string {
	if x != nil {
		return x.ConnId
	}
	return ""
}

var File_unRegisterReq_proto protoreflect.FileDescriptor

var file_unRegisterReq_proto_rawDesc = []byte{
	0x0a, 0x13, 0x75, 0x6e, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x65, 0x72, 0x52, 0x65, 0x71, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x14, 0x6e, 0x65, 0x74, 0x73, 0x76, 0x72, 0x2e, 0x75, 0x6e,
	0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x65, 0x72, 0x52, 0x65, 0x71, 0x22, 0x27, 0x0a, 0x0d, 0x55,
	0x6e, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x65, 0x72, 0x52, 0x65, 0x71, 0x12, 0x16, 0x0a, 0x06,
	0x63, 0x6f, 0x6e, 0x6e, 0x49, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x63, 0x6f,
	0x6e, 0x6e, 0x49, 0x64, 0x42, 0x27, 0x5a, 0x07, 0x6e, 0x65, 0x74, 0x73, 0x76, 0x72, 0x2f, 0xca,
	0x02, 0x06, 0x4e, 0x65, 0x74, 0x73, 0x76, 0x72, 0xe2, 0x02, 0x12, 0x4e, 0x65, 0x74, 0x73, 0x76,
	0x72, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_unRegisterReq_proto_rawDescOnce sync.Once
	file_unRegisterReq_proto_rawDescData = file_unRegisterReq_proto_rawDesc
)

func file_unRegisterReq_proto_rawDescGZIP() []byte {
	file_unRegisterReq_proto_rawDescOnce.Do(func() {
		file_unRegisterReq_proto_rawDescData = protoimpl.X.CompressGZIP(file_unRegisterReq_proto_rawDescData)
	})
	return file_unRegisterReq_proto_rawDescData
}

var file_unRegisterReq_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_unRegisterReq_proto_goTypes = []interface{}{
	(*UnRegisterReq)(nil), // 0: netsvr.unRegisterReq.UnRegisterReq
}
var file_unRegisterReq_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_unRegisterReq_proto_init() }
func file_unRegisterReq_proto_init() {
	if File_unRegisterReq_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_unRegisterReq_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*UnRegisterReq); i {
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
			RawDescriptor: file_unRegisterReq_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_unRegisterReq_proto_goTypes,
		DependencyIndexes: file_unRegisterReq_proto_depIdxs,
		MessageInfos:      file_unRegisterReq_proto_msgTypes,
	}.Build()
	File_unRegisterReq_proto = out.File
	file_unRegisterReq_proto_rawDesc = nil
	file_unRegisterReq_proto_goTypes = nil
	file_unRegisterReq_proto_depIdxs = nil
}
