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
// source: topicUniqIdListResp.proto

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

// worker响应business，返回网关中某个主题包含的uniqId
type TopicUniqIdListResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// key是topic，value是该主题包含的uniqId
	// 如果请求的topic没找到，则items中不会有该topic
	Items map[string]*TopicUniqIdListRespItem `protobuf:"bytes,2,rep,name=items,proto3" json:"items,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *TopicUniqIdListResp) Reset() {
	*x = TopicUniqIdListResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_topicUniqIdListResp_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TopicUniqIdListResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TopicUniqIdListResp) ProtoMessage() {}

func (x *TopicUniqIdListResp) ProtoReflect() protoreflect.Message {
	mi := &file_topicUniqIdListResp_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TopicUniqIdListResp.ProtoReflect.Descriptor instead.
func (*TopicUniqIdListResp) Descriptor() ([]byte, []int) {
	return file_topicUniqIdListResp_proto_rawDescGZIP(), []int{0}
}

func (x *TopicUniqIdListResp) GetItems() map[string]*TopicUniqIdListRespItem {
	if x != nil {
		return x.Items
	}
	return nil
}

type TopicUniqIdListRespItem struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// 当前主题包含的uniqId
	UniqIds []string `protobuf:"bytes,1,rep,name=uniqIds,proto3" json:"uniqIds,omitempty"`
}

func (x *TopicUniqIdListRespItem) Reset() {
	*x = TopicUniqIdListRespItem{}
	if protoimpl.UnsafeEnabled {
		mi := &file_topicUniqIdListResp_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TopicUniqIdListRespItem) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TopicUniqIdListRespItem) ProtoMessage() {}

func (x *TopicUniqIdListRespItem) ProtoReflect() protoreflect.Message {
	mi := &file_topicUniqIdListResp_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TopicUniqIdListRespItem.ProtoReflect.Descriptor instead.
func (*TopicUniqIdListRespItem) Descriptor() ([]byte, []int) {
	return file_topicUniqIdListResp_proto_rawDescGZIP(), []int{1}
}

func (x *TopicUniqIdListRespItem) GetUniqIds() []string {
	if x != nil {
		return x.UniqIds
	}
	return nil
}

var File_topicUniqIdListResp_proto protoreflect.FileDescriptor

var file_topicUniqIdListResp_proto_rawDesc = []byte{
	0x0a, 0x19, 0x74, 0x6f, 0x70, 0x69, 0x63, 0x55, 0x6e, 0x69, 0x71, 0x49, 0x64, 0x4c, 0x69, 0x73,
	0x74, 0x52, 0x65, 0x73, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x1a, 0x6e, 0x65, 0x74,
	0x73, 0x76, 0x72, 0x2e, 0x74, 0x6f, 0x70, 0x69, 0x63, 0x55, 0x6e, 0x69, 0x71, 0x49, 0x64, 0x4c,
	0x69, 0x73, 0x74, 0x52, 0x65, 0x73, 0x70, 0x22, 0xd6, 0x01, 0x0a, 0x13, 0x54, 0x6f, 0x70, 0x69,
	0x63, 0x55, 0x6e, 0x69, 0x71, 0x49, 0x64, 0x4c, 0x69, 0x73, 0x74, 0x52, 0x65, 0x73, 0x70, 0x12,
	0x50, 0x0a, 0x05, 0x69, 0x74, 0x65, 0x6d, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x3a,
	0x2e, 0x6e, 0x65, 0x74, 0x73, 0x76, 0x72, 0x2e, 0x74, 0x6f, 0x70, 0x69, 0x63, 0x55, 0x6e, 0x69,
	0x71, 0x49, 0x64, 0x4c, 0x69, 0x73, 0x74, 0x52, 0x65, 0x73, 0x70, 0x2e, 0x54, 0x6f, 0x70, 0x69,
	0x63, 0x55, 0x6e, 0x69, 0x71, 0x49, 0x64, 0x4c, 0x69, 0x73, 0x74, 0x52, 0x65, 0x73, 0x70, 0x2e,
	0x49, 0x74, 0x65, 0x6d, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x05, 0x69, 0x74, 0x65, 0x6d,
	0x73, 0x1a, 0x6d, 0x0a, 0x0a, 0x49, 0x74, 0x65, 0x6d, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12,
	0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65,
	0x79, 0x12, 0x49, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x33, 0x2e, 0x6e, 0x65, 0x74, 0x73, 0x76, 0x72, 0x2e, 0x74, 0x6f, 0x70, 0x69, 0x63, 0x55,
	0x6e, 0x69, 0x71, 0x49, 0x64, 0x4c, 0x69, 0x73, 0x74, 0x52, 0x65, 0x73, 0x70, 0x2e, 0x54, 0x6f,
	0x70, 0x69, 0x63, 0x55, 0x6e, 0x69, 0x71, 0x49, 0x64, 0x4c, 0x69, 0x73, 0x74, 0x52, 0x65, 0x73,
	0x70, 0x49, 0x74, 0x65, 0x6d, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01,
	0x22, 0x33, 0x0a, 0x17, 0x54, 0x6f, 0x70, 0x69, 0x63, 0x55, 0x6e, 0x69, 0x71, 0x49, 0x64, 0x4c,
	0x69, 0x73, 0x74, 0x52, 0x65, 0x73, 0x70, 0x49, 0x74, 0x65, 0x6d, 0x12, 0x18, 0x0a, 0x07, 0x75,
	0x6e, 0x69, 0x71, 0x49, 0x64, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x09, 0x52, 0x07, 0x75, 0x6e,
	0x69, 0x71, 0x49, 0x64, 0x73, 0x42, 0x27, 0x5a, 0x07, 0x6e, 0x65, 0x74, 0x73, 0x76, 0x72, 0x2f,
	0xca, 0x02, 0x06, 0x4e, 0x65, 0x74, 0x73, 0x76, 0x72, 0xe2, 0x02, 0x12, 0x4e, 0x65, 0x74, 0x73,
	0x76, 0x72, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_topicUniqIdListResp_proto_rawDescOnce sync.Once
	file_topicUniqIdListResp_proto_rawDescData = file_topicUniqIdListResp_proto_rawDesc
)

func file_topicUniqIdListResp_proto_rawDescGZIP() []byte {
	file_topicUniqIdListResp_proto_rawDescOnce.Do(func() {
		file_topicUniqIdListResp_proto_rawDescData = protoimpl.X.CompressGZIP(file_topicUniqIdListResp_proto_rawDescData)
	})
	return file_topicUniqIdListResp_proto_rawDescData
}

var file_topicUniqIdListResp_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_topicUniqIdListResp_proto_goTypes = []interface{}{
	(*TopicUniqIdListResp)(nil),     // 0: netsvr.topicUniqIdListResp.TopicUniqIdListResp
	(*TopicUniqIdListRespItem)(nil), // 1: netsvr.topicUniqIdListResp.TopicUniqIdListRespItem
	nil,                             // 2: netsvr.topicUniqIdListResp.TopicUniqIdListResp.ItemsEntry
}
var file_topicUniqIdListResp_proto_depIdxs = []int32{
	2, // 0: netsvr.topicUniqIdListResp.TopicUniqIdListResp.items:type_name -> netsvr.topicUniqIdListResp.TopicUniqIdListResp.ItemsEntry
	1, // 1: netsvr.topicUniqIdListResp.TopicUniqIdListResp.ItemsEntry.value:type_name -> netsvr.topicUniqIdListResp.TopicUniqIdListRespItem
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_topicUniqIdListResp_proto_init() }
func file_topicUniqIdListResp_proto_init() {
	if File_topicUniqIdListResp_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_topicUniqIdListResp_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TopicUniqIdListResp); i {
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
		file_topicUniqIdListResp_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TopicUniqIdListRespItem); i {
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
			RawDescriptor: file_topicUniqIdListResp_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_topicUniqIdListResp_proto_goTypes,
		DependencyIndexes: file_topicUniqIdListResp_proto_depIdxs,
		MessageInfos:      file_topicUniqIdListResp_proto_msgTypes,
	}.Build()
	File_topicUniqIdListResp_proto = out.File
	file_topicUniqIdListResp_proto_rawDesc = nil
	file_topicUniqIdListResp_proto_goTypes = nil
	file_topicUniqIdListResp_proto_depIdxs = nil
}
