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
// source: forceOfflineGuest.proto

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

// business向worker请求，将某几个没有session值的连接强制关闭
type ForceOfflineGuest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// 目标uniqId
	UniqIds []string `protobuf:"bytes,1,rep,name=uniqIds,proto3" json:"uniqIds,omitempty"`
	// 延迟多少秒执行，如果是0，立刻执行，否则就等待该秒数后，再根据uniqId获取连接，并判断连接是否存在session，没有就关闭连接，有就忽略
	Delay int32 `protobuf:"varint,2,opt,name=delay,proto3" json:"delay,omitempty"`
	// 需要发给客户的数据，有这个数据，则转发给该连接，并在3秒倒计时后强制关闭连接，反之，立马关闭连接
	Data []byte `protobuf:"bytes,3,opt,name=data,proto3" json:"data,omitempty"`
}

func (x *ForceOfflineGuest) Reset() {
	*x = ForceOfflineGuest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_forceOfflineGuest_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ForceOfflineGuest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ForceOfflineGuest) ProtoMessage() {}

func (x *ForceOfflineGuest) ProtoReflect() protoreflect.Message {
	mi := &file_forceOfflineGuest_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ForceOfflineGuest.ProtoReflect.Descriptor instead.
func (*ForceOfflineGuest) Descriptor() ([]byte, []int) {
	return file_forceOfflineGuest_proto_rawDescGZIP(), []int{0}
}

func (x *ForceOfflineGuest) GetUniqIds() []string {
	if x != nil {
		return x.UniqIds
	}
	return nil
}

func (x *ForceOfflineGuest) GetDelay() int32 {
	if x != nil {
		return x.Delay
	}
	return 0
}

func (x *ForceOfflineGuest) GetData() []byte {
	if x != nil {
		return x.Data
	}
	return nil
}

var File_forceOfflineGuest_proto protoreflect.FileDescriptor

var file_forceOfflineGuest_proto_rawDesc = []byte{
	0x0a, 0x17, 0x66, 0x6f, 0x72, 0x63, 0x65, 0x4f, 0x66, 0x66, 0x6c, 0x69, 0x6e, 0x65, 0x47, 0x75,
	0x65, 0x73, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x18, 0x6e, 0x65, 0x74, 0x73, 0x76,
	0x72, 0x2e, 0x66, 0x6f, 0x72, 0x63, 0x65, 0x4f, 0x66, 0x66, 0x6c, 0x69, 0x6e, 0x65, 0x47, 0x75,
	0x65, 0x73, 0x74, 0x22, 0x57, 0x0a, 0x11, 0x46, 0x6f, 0x72, 0x63, 0x65, 0x4f, 0x66, 0x66, 0x6c,
	0x69, 0x6e, 0x65, 0x47, 0x75, 0x65, 0x73, 0x74, 0x12, 0x18, 0x0a, 0x07, 0x75, 0x6e, 0x69, 0x71,
	0x49, 0x64, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x09, 0x52, 0x07, 0x75, 0x6e, 0x69, 0x71, 0x49,
	0x64, 0x73, 0x12, 0x14, 0x0a, 0x05, 0x64, 0x65, 0x6c, 0x61, 0x79, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x05, 0x52, 0x05, 0x64, 0x65, 0x6c, 0x61, 0x79, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x42, 0x27, 0x5a, 0x07,
	0x6e, 0x65, 0x74, 0x73, 0x76, 0x72, 0x2f, 0xca, 0x02, 0x06, 0x4e, 0x65, 0x74, 0x73, 0x76, 0x72,
	0xe2, 0x02, 0x12, 0x4e, 0x65, 0x74, 0x73, 0x76, 0x72, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74,
	0x61, 0x64, 0x61, 0x74, 0x61, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_forceOfflineGuest_proto_rawDescOnce sync.Once
	file_forceOfflineGuest_proto_rawDescData = file_forceOfflineGuest_proto_rawDesc
)

func file_forceOfflineGuest_proto_rawDescGZIP() []byte {
	file_forceOfflineGuest_proto_rawDescOnce.Do(func() {
		file_forceOfflineGuest_proto_rawDescData = protoimpl.X.CompressGZIP(file_forceOfflineGuest_proto_rawDescData)
	})
	return file_forceOfflineGuest_proto_rawDescData
}

var file_forceOfflineGuest_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_forceOfflineGuest_proto_goTypes = []interface{}{
	(*ForceOfflineGuest)(nil), // 0: netsvr.forceOfflineGuest.ForceOfflineGuest
}
var file_forceOfflineGuest_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_forceOfflineGuest_proto_init() }
func file_forceOfflineGuest_proto_init() {
	if File_forceOfflineGuest_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_forceOfflineGuest_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ForceOfflineGuest); i {
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
			RawDescriptor: file_forceOfflineGuest_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_forceOfflineGuest_proto_goTypes,
		DependencyIndexes: file_forceOfflineGuest_proto_depIdxs,
		MessageInfos:      file_forceOfflineGuest_proto_msgTypes,
	}.Build()
	File_forceOfflineGuest_proto = out.File
	file_forceOfflineGuest_proto_rawDesc = nil
	file_forceOfflineGuest_proto_goTypes = nil
	file_forceOfflineGuest_proto_depIdxs = nil
}
