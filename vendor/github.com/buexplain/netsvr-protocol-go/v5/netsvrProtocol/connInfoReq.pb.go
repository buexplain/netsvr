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
// source: connInfoReq.proto

package netsvrProtocol

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

// business向worker请求，获取目标uniqId在网关中存储的信息
type ConnInfoReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// 目标uniqId
	UniqIds []string `protobuf:"bytes,1,rep,name=uniqIds,proto3" json:"uniqIds,omitempty"`
	// 是否获取session
	ReqSession bool `protobuf:"varint,2,opt,name=reqSession,proto3" json:"reqSession,omitempty"`
	// 是否获取customerId
	ReqCustomerId bool `protobuf:"varint,3,opt,name=reqCustomerId,proto3" json:"reqCustomerId,omitempty"`
	// 是否获取topic
	ReqTopic bool `protobuf:"varint,4,opt,name=reqTopic,proto3" json:"reqTopic,omitempty"`
}

func (x *ConnInfoReq) Reset() {
	*x = ConnInfoReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_connInfoReq_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ConnInfoReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ConnInfoReq) ProtoMessage() {}

func (x *ConnInfoReq) ProtoReflect() protoreflect.Message {
	mi := &file_connInfoReq_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ConnInfoReq.ProtoReflect.Descriptor instead.
func (*ConnInfoReq) Descriptor() ([]byte, []int) {
	return file_connInfoReq_proto_rawDescGZIP(), []int{0}
}

func (x *ConnInfoReq) GetUniqIds() []string {
	if x != nil {
		return x.UniqIds
	}
	return nil
}

func (x *ConnInfoReq) GetReqSession() bool {
	if x != nil {
		return x.ReqSession
	}
	return false
}

func (x *ConnInfoReq) GetReqCustomerId() bool {
	if x != nil {
		return x.ReqCustomerId
	}
	return false
}

func (x *ConnInfoReq) GetReqTopic() bool {
	if x != nil {
		return x.ReqTopic
	}
	return false
}

var File_connInfoReq_proto protoreflect.FileDescriptor

var file_connInfoReq_proto_rawDesc = []byte{
	0x0a, 0x11, 0x63, 0x6f, 0x6e, 0x6e, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x65, 0x71, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x12, 0x6e, 0x65, 0x74, 0x73, 0x76, 0x72, 0x2e, 0x63, 0x6f, 0x6e, 0x6e,
	0x49, 0x6e, 0x66, 0x6f, 0x52, 0x65, 0x71, 0x22, 0x89, 0x01, 0x0a, 0x0b, 0x43, 0x6f, 0x6e, 0x6e,
	0x49, 0x6e, 0x66, 0x6f, 0x52, 0x65, 0x71, 0x12, 0x18, 0x0a, 0x07, 0x75, 0x6e, 0x69, 0x71, 0x49,
	0x64, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x09, 0x52, 0x07, 0x75, 0x6e, 0x69, 0x71, 0x49, 0x64,
	0x73, 0x12, 0x1e, 0x0a, 0x0a, 0x72, 0x65, 0x71, 0x53, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0a, 0x72, 0x65, 0x71, 0x53, 0x65, 0x73, 0x73, 0x69, 0x6f,
	0x6e, 0x12, 0x24, 0x0a, 0x0d, 0x72, 0x65, 0x71, 0x43, 0x75, 0x73, 0x74, 0x6f, 0x6d, 0x65, 0x72,
	0x49, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0d, 0x72, 0x65, 0x71, 0x43, 0x75, 0x73,
	0x74, 0x6f, 0x6d, 0x65, 0x72, 0x49, 0x64, 0x12, 0x1a, 0x0a, 0x08, 0x72, 0x65, 0x71, 0x54, 0x6f,
	0x70, 0x69, 0x63, 0x18, 0x04, 0x20, 0x01, 0x28, 0x08, 0x52, 0x08, 0x72, 0x65, 0x71, 0x54, 0x6f,
	0x70, 0x69, 0x63, 0x42, 0x3f, 0x5a, 0x0f, 0x6e, 0x65, 0x74, 0x73, 0x76, 0x72, 0x50, 0x72, 0x6f,
	0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x2f, 0xca, 0x02, 0x0e, 0x4e, 0x65, 0x74, 0x73, 0x76, 0x72, 0x50,
	0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0xe2, 0x02, 0x1a, 0x4e, 0x65, 0x74, 0x73, 0x76, 0x72,
	0x50, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74, 0x61,
	0x64, 0x61, 0x74, 0x61, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_connInfoReq_proto_rawDescOnce sync.Once
	file_connInfoReq_proto_rawDescData = file_connInfoReq_proto_rawDesc
)

func file_connInfoReq_proto_rawDescGZIP() []byte {
	file_connInfoReq_proto_rawDescOnce.Do(func() {
		file_connInfoReq_proto_rawDescData = protoimpl.X.CompressGZIP(file_connInfoReq_proto_rawDescData)
	})
	return file_connInfoReq_proto_rawDescData
}

var file_connInfoReq_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_connInfoReq_proto_goTypes = []interface{}{
	(*ConnInfoReq)(nil), // 0: netsvr.connInfoReq.ConnInfoReq
}
var file_connInfoReq_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_connInfoReq_proto_init() }
func file_connInfoReq_proto_init() {
	if File_connInfoReq_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_connInfoReq_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ConnInfoReq); i {
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
			RawDescriptor: file_connInfoReq_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_connInfoReq_proto_goTypes,
		DependencyIndexes: file_connInfoReq_proto_depIdxs,
		MessageInfos:      file_connInfoReq_proto_msgTypes,
	}.Build()
	File_connInfoReq_proto = out.File
	file_connInfoReq_proto_rawDesc = nil
	file_connInfoReq_proto_goTypes = nil
	file_connInfoReq_proto_depIdxs = nil
}
