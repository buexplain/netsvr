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
// source: metricsResp.proto

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

// worker响应business，返回网关统计的服务状态
type MetricsResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// key是统计的项，value是统计结果
	Items map[int32]*MetricsRespItem `protobuf:"bytes,1,rep,name=items,proto3" json:"items,omitempty" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *MetricsResp) Reset() {
	*x = MetricsResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_metricsResp_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MetricsResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MetricsResp) ProtoMessage() {}

func (x *MetricsResp) ProtoReflect() protoreflect.Message {
	mi := &file_metricsResp_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MetricsResp.ProtoReflect.Descriptor instead.
func (*MetricsResp) Descriptor() ([]byte, []int) {
	return file_metricsResp_proto_rawDescGZIP(), []int{0}
}

func (x *MetricsResp) GetItems() map[int32]*MetricsRespItem {
	if x != nil {
		return x.Items
	}
	return nil
}

type MetricsRespItem struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// 统计的项
	Item int32 `protobuf:"varint,1,opt,name=item,proto3" json:"item,omitempty"`
	// returns the count of events at the time the snapshot was taken.
	Count int64 `protobuf:"varint,2,opt,name=count,proto3" json:"count,omitempty"`
	// returns the meter's mean rate of events per second at the time the snapshot was taken.
	MeanRate float32 `protobuf:"fixed32,3,opt,name=meanRate,proto3" json:"meanRate,omitempty"`
	// Rate1 returns the one-minute moving average rate of events per second at the time the snapshot was taken.
	Rate1 float32 `protobuf:"fixed32,5,opt,name=rate1,proto3" json:"rate1,omitempty"`
	// Rate5 returns the five-minute moving average rate of events per second at the time the snapshot was taken.
	Rate5 float32 `protobuf:"fixed32,7,opt,name=rate5,proto3" json:"rate5,omitempty"`
	// Rate15 returns the fifteen-minute moving average rate of events per second at the time the snapshot was taken.
	Rate15 float32 `protobuf:"fixed32,9,opt,name=rate15,proto3" json:"rate15,omitempty"`
}

func (x *MetricsRespItem) Reset() {
	*x = MetricsRespItem{}
	if protoimpl.UnsafeEnabled {
		mi := &file_metricsResp_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MetricsRespItem) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MetricsRespItem) ProtoMessage() {}

func (x *MetricsRespItem) ProtoReflect() protoreflect.Message {
	mi := &file_metricsResp_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MetricsRespItem.ProtoReflect.Descriptor instead.
func (*MetricsRespItem) Descriptor() ([]byte, []int) {
	return file_metricsResp_proto_rawDescGZIP(), []int{1}
}

func (x *MetricsRespItem) GetItem() int32 {
	if x != nil {
		return x.Item
	}
	return 0
}

func (x *MetricsRespItem) GetCount() int64 {
	if x != nil {
		return x.Count
	}
	return 0
}

func (x *MetricsRespItem) GetMeanRate() float32 {
	if x != nil {
		return x.MeanRate
	}
	return 0
}

func (x *MetricsRespItem) GetRate1() float32 {
	if x != nil {
		return x.Rate1
	}
	return 0
}

func (x *MetricsRespItem) GetRate5() float32 {
	if x != nil {
		return x.Rate5
	}
	return 0
}

func (x *MetricsRespItem) GetRate15() float32 {
	if x != nil {
		return x.Rate15
	}
	return 0
}

var File_metricsResp_proto protoreflect.FileDescriptor

var file_metricsResp_proto_rawDesc = []byte{
	0x0a, 0x11, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x52, 0x65, 0x73, 0x70, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x12, 0x6e, 0x65, 0x74, 0x73, 0x76, 0x72, 0x2e, 0x6d, 0x65, 0x74, 0x72,
	0x69, 0x63, 0x73, 0x52, 0x65, 0x73, 0x70, 0x22, 0xae, 0x01, 0x0a, 0x0b, 0x4d, 0x65, 0x74, 0x72,
	0x69, 0x63, 0x73, 0x52, 0x65, 0x73, 0x70, 0x12, 0x40, 0x0a, 0x05, 0x69, 0x74, 0x65, 0x6d, 0x73,
	0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x2a, 0x2e, 0x6e, 0x65, 0x74, 0x73, 0x76, 0x72, 0x2e,
	0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x52, 0x65, 0x73, 0x70, 0x2e, 0x4d, 0x65, 0x74, 0x72,
	0x69, 0x63, 0x73, 0x52, 0x65, 0x73, 0x70, 0x2e, 0x49, 0x74, 0x65, 0x6d, 0x73, 0x45, 0x6e, 0x74,
	0x72, 0x79, 0x52, 0x05, 0x69, 0x74, 0x65, 0x6d, 0x73, 0x1a, 0x5d, 0x0a, 0x0a, 0x49, 0x74, 0x65,
	0x6d, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x05, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x39, 0x0a, 0x05, 0x76, 0x61, 0x6c,
	0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x23, 0x2e, 0x6e, 0x65, 0x74, 0x73, 0x76,
	0x72, 0x2e, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x52, 0x65, 0x73, 0x70, 0x2e, 0x4d, 0x65,
	0x74, 0x72, 0x69, 0x63, 0x73, 0x52, 0x65, 0x73, 0x70, 0x49, 0x74, 0x65, 0x6d, 0x52, 0x05, 0x76,
	0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0x9b, 0x01, 0x0a, 0x0f, 0x4d, 0x65, 0x74,
	0x72, 0x69, 0x63, 0x73, 0x52, 0x65, 0x73, 0x70, 0x49, 0x74, 0x65, 0x6d, 0x12, 0x12, 0x0a, 0x04,
	0x69, 0x74, 0x65, 0x6d, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x04, 0x69, 0x74, 0x65, 0x6d,
	0x12, 0x14, 0x0a, 0x05, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x52,
	0x05, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x1a, 0x0a, 0x08, 0x6d, 0x65, 0x61, 0x6e, 0x52, 0x61,
	0x74, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x02, 0x52, 0x08, 0x6d, 0x65, 0x61, 0x6e, 0x52, 0x61,
	0x74, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x72, 0x61, 0x74, 0x65, 0x31, 0x18, 0x05, 0x20, 0x01, 0x28,
	0x02, 0x52, 0x05, 0x72, 0x61, 0x74, 0x65, 0x31, 0x12, 0x14, 0x0a, 0x05, 0x72, 0x61, 0x74, 0x65,
	0x35, 0x18, 0x07, 0x20, 0x01, 0x28, 0x02, 0x52, 0x05, 0x72, 0x61, 0x74, 0x65, 0x35, 0x12, 0x16,
	0x0a, 0x06, 0x72, 0x61, 0x74, 0x65, 0x31, 0x35, 0x18, 0x09, 0x20, 0x01, 0x28, 0x02, 0x52, 0x06,
	0x72, 0x61, 0x74, 0x65, 0x31, 0x35, 0x42, 0x3f, 0x5a, 0x0f, 0x6e, 0x65, 0x74, 0x73, 0x76, 0x72,
	0x50, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x2f, 0xca, 0x02, 0x0e, 0x4e, 0x65, 0x74, 0x73,
	0x76, 0x72, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0xe2, 0x02, 0x1a, 0x4e, 0x65, 0x74,
	0x73, 0x76, 0x72, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x5c, 0x47, 0x50, 0x42, 0x4d,
	0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_metricsResp_proto_rawDescOnce sync.Once
	file_metricsResp_proto_rawDescData = file_metricsResp_proto_rawDesc
)

func file_metricsResp_proto_rawDescGZIP() []byte {
	file_metricsResp_proto_rawDescOnce.Do(func() {
		file_metricsResp_proto_rawDescData = protoimpl.X.CompressGZIP(file_metricsResp_proto_rawDescData)
	})
	return file_metricsResp_proto_rawDescData
}

var file_metricsResp_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_metricsResp_proto_goTypes = []interface{}{
	(*MetricsResp)(nil),     // 0: netsvr.metricsResp.MetricsResp
	(*MetricsRespItem)(nil), // 1: netsvr.metricsResp.MetricsRespItem
	nil,                     // 2: netsvr.metricsResp.MetricsResp.ItemsEntry
}
var file_metricsResp_proto_depIdxs = []int32{
	2, // 0: netsvr.metricsResp.MetricsResp.items:type_name -> netsvr.metricsResp.MetricsResp.ItemsEntry
	1, // 1: netsvr.metricsResp.MetricsResp.ItemsEntry.value:type_name -> netsvr.metricsResp.MetricsRespItem
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_metricsResp_proto_init() }
func file_metricsResp_proto_init() {
	if File_metricsResp_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_metricsResp_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*MetricsResp); i {
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
		file_metricsResp_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*MetricsRespItem); i {
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
			RawDescriptor: file_metricsResp_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_metricsResp_proto_goTypes,
		DependencyIndexes: file_metricsResp_proto_depIdxs,
		MessageInfos:      file_metricsResp_proto_msgTypes,
	}.Build()
	File_metricsResp_proto = out.File
	file_metricsResp_proto_rawDesc = nil
	file_metricsResp_proto_goTypes = nil
	file_metricsResp_proto_depIdxs = nil
}
