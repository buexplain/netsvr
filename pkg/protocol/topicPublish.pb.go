//*
// Copyright 2022 buexplain@qq.com
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
// source: topicPublish.proto

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

// business向worker请求，给某个主题发布信息
type TopicPublish struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// 目标主题
	Topic string `protobuf:"bytes,1,opt,name=topic,proto3" json:"topic,omitempty"`
	// 需要发给客户的数据
	Data []byte `protobuf:"bytes,2,opt,name=data,proto3" json:"data,omitempty"`
}

func (x *TopicPublish) Reset() {
	*x = TopicPublish{}
	if protoimpl.UnsafeEnabled {
		mi := &file_topicPublish_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TopicPublish) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TopicPublish) ProtoMessage() {}

func (x *TopicPublish) ProtoReflect() protoreflect.Message {
	mi := &file_topicPublish_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TopicPublish.ProtoReflect.Descriptor instead.
func (*TopicPublish) Descriptor() ([]byte, []int) {
	return file_topicPublish_proto_rawDescGZIP(), []int{0}
}

func (x *TopicPublish) GetTopic() string {
	if x != nil {
		return x.Topic
	}
	return ""
}

func (x *TopicPublish) GetData() []byte {
	if x != nil {
		return x.Data
	}
	return nil
}

var File_topicPublish_proto protoreflect.FileDescriptor

var file_topicPublish_proto_rawDesc = []byte{
	0x0a, 0x12, 0x74, 0x6f, 0x70, 0x69, 0x63, 0x50, 0x75, 0x62, 0x6c, 0x69, 0x73, 0x68, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x12, 0x13, 0x6e, 0x65, 0x74, 0x73, 0x76, 0x72, 0x2e, 0x74, 0x6f, 0x70,
	0x69, 0x63, 0x50, 0x75, 0x62, 0x6c, 0x69, 0x73, 0x68, 0x22, 0x38, 0x0a, 0x0c, 0x54, 0x6f, 0x70,
	0x69, 0x63, 0x50, 0x75, 0x62, 0x6c, 0x69, 0x73, 0x68, 0x12, 0x14, 0x0a, 0x05, 0x74, 0x6f, 0x70,
	0x69, 0x63, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x74, 0x6f, 0x70, 0x69, 0x63, 0x12,
	0x12, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x04, 0x64,
	0x61, 0x74, 0x61, 0x42, 0x17, 0x5a, 0x15, 0x70, 0x6b, 0x67, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x63, 0x6f, 0x6c, 0x3b, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x62, 0x06, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_topicPublish_proto_rawDescOnce sync.Once
	file_topicPublish_proto_rawDescData = file_topicPublish_proto_rawDesc
)

func file_topicPublish_proto_rawDescGZIP() []byte {
	file_topicPublish_proto_rawDescOnce.Do(func() {
		file_topicPublish_proto_rawDescData = protoimpl.X.CompressGZIP(file_topicPublish_proto_rawDescData)
	})
	return file_topicPublish_proto_rawDescData
}

var file_topicPublish_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_topicPublish_proto_goTypes = []interface{}{
	(*TopicPublish)(nil), // 0: netsvr.topicPublish.TopicPublish
}
var file_topicPublish_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_topicPublish_proto_init() }
func file_topicPublish_proto_init() {
	if File_topicPublish_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_topicPublish_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TopicPublish); i {
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
			RawDescriptor: file_topicPublish_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_topicPublish_proto_goTypes,
		DependencyIndexes: file_topicPublish_proto_depIdxs,
		MessageInfos:      file_topicPublish_proto_msgTypes,
	}.Build()
	File_topicPublish_proto = out.File
	file_topicPublish_proto_rawDesc = nil
	file_topicPublish_proto_goTypes = nil
	file_topicPublish_proto_depIdxs = nil
}