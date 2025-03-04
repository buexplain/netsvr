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
// source: connOpen.proto

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

// worker转发客户连接打开的信息到business
type ConnOpen struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// 网关分配给连接的唯一id，格式是：网关进程的worker服务监听的ip地址(4字节)+网关进程的worker服务监听的port(2字节)+时间戳(4字节)+自增id(4字节)，共14字节，28个16进制的字符
	// php解码uniqId示例：
	// $ret =  unpack('Nip/nport/Ntimestamp/NincrId', pack('H*', '7f00000117ad6621e43b8baa1b9a'));
	// $ret['ip'] = long2ip($ret['ip']);
	// var_dump($ret);
	UniqId string `protobuf:"bytes,1,opt,name=uniqId,proto3" json:"uniqId,omitempty"`
	// 连接携带的GET参数
	RawQuery string `protobuf:"bytes,2,opt,name=rawQuery,proto3" json:"rawQuery,omitempty"`
	// 连接携带的websocket子协议信息
	SubProtocol []string `protobuf:"bytes,3,rep,name=subProtocol,proto3" json:"subProtocol,omitempty"`
	// 读取的header：X-Forwarded-For
	XForwardedFor string `protobuf:"bytes,4,opt,name=xForwardedFor,proto3" json:"xForwardedFor,omitempty"`
	// 读取的header：X-Real-IP
	XRealIp string `protobuf:"bytes,5,opt,name=xRealIp,proto3" json:"xRealIp,omitempty"`
	// 连接的IP地址，这是直接连接到网关的IP地址
	RemoteAddr string `protobuf:"bytes,6,opt,name=remoteAddr,proto3" json:"remoteAddr,omitempty"`
}

func (x *ConnOpen) Reset() {
	*x = ConnOpen{}
	if protoimpl.UnsafeEnabled {
		mi := &file_connOpen_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ConnOpen) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ConnOpen) ProtoMessage() {}

func (x *ConnOpen) ProtoReflect() protoreflect.Message {
	mi := &file_connOpen_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ConnOpen.ProtoReflect.Descriptor instead.
func (*ConnOpen) Descriptor() ([]byte, []int) {
	return file_connOpen_proto_rawDescGZIP(), []int{0}
}

func (x *ConnOpen) GetUniqId() string {
	if x != nil {
		return x.UniqId
	}
	return ""
}

func (x *ConnOpen) GetRawQuery() string {
	if x != nil {
		return x.RawQuery
	}
	return ""
}

func (x *ConnOpen) GetSubProtocol() []string {
	if x != nil {
		return x.SubProtocol
	}
	return nil
}

func (x *ConnOpen) GetXForwardedFor() string {
	if x != nil {
		return x.XForwardedFor
	}
	return ""
}

func (x *ConnOpen) GetXRealIp() string {
	if x != nil {
		return x.XRealIp
	}
	return ""
}

func (x *ConnOpen) GetRemoteAddr() string {
	if x != nil {
		return x.RemoteAddr
	}
	return ""
}

var File_connOpen_proto protoreflect.FileDescriptor

var file_connOpen_proto_rawDesc = []byte{
	0x0a, 0x0e, 0x63, 0x6f, 0x6e, 0x6e, 0x4f, 0x70, 0x65, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x12, 0x0f, 0x6e, 0x65, 0x74, 0x73, 0x76, 0x72, 0x2e, 0x63, 0x6f, 0x6e, 0x6e, 0x4f, 0x70, 0x65,
	0x6e, 0x22, 0xc0, 0x01, 0x0a, 0x08, 0x43, 0x6f, 0x6e, 0x6e, 0x4f, 0x70, 0x65, 0x6e, 0x12, 0x16,
	0x0a, 0x06, 0x75, 0x6e, 0x69, 0x71, 0x49, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06,
	0x75, 0x6e, 0x69, 0x71, 0x49, 0x64, 0x12, 0x1a, 0x0a, 0x08, 0x72, 0x61, 0x77, 0x51, 0x75, 0x65,
	0x72, 0x79, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x72, 0x61, 0x77, 0x51, 0x75, 0x65,
	0x72, 0x79, 0x12, 0x20, 0x0a, 0x0b, 0x73, 0x75, 0x62, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f,
	0x6c, 0x18, 0x03, 0x20, 0x03, 0x28, 0x09, 0x52, 0x0b, 0x73, 0x75, 0x62, 0x50, 0x72, 0x6f, 0x74,
	0x6f, 0x63, 0x6f, 0x6c, 0x12, 0x24, 0x0a, 0x0d, 0x78, 0x46, 0x6f, 0x72, 0x77, 0x61, 0x72, 0x64,
	0x65, 0x64, 0x46, 0x6f, 0x72, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0d, 0x78, 0x46, 0x6f,
	0x72, 0x77, 0x61, 0x72, 0x64, 0x65, 0x64, 0x46, 0x6f, 0x72, 0x12, 0x18, 0x0a, 0x07, 0x78, 0x52,
	0x65, 0x61, 0x6c, 0x49, 0x70, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x78, 0x52, 0x65,
	0x61, 0x6c, 0x49, 0x70, 0x12, 0x1e, 0x0a, 0x0a, 0x72, 0x65, 0x6d, 0x6f, 0x74, 0x65, 0x41, 0x64,
	0x64, 0x72, 0x18, 0x06, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x72, 0x65, 0x6d, 0x6f, 0x74, 0x65,
	0x41, 0x64, 0x64, 0x72, 0x42, 0x3f, 0x5a, 0x0f, 0x6e, 0x65, 0x74, 0x73, 0x76, 0x72, 0x50, 0x72,
	0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x2f, 0xca, 0x02, 0x0e, 0x4e, 0x65, 0x74, 0x73, 0x76, 0x72,
	0x50, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0xe2, 0x02, 0x1a, 0x4e, 0x65, 0x74, 0x73, 0x76,
	0x72, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74,
	0x61, 0x64, 0x61, 0x74, 0x61, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_connOpen_proto_rawDescOnce sync.Once
	file_connOpen_proto_rawDescData = file_connOpen_proto_rawDesc
)

func file_connOpen_proto_rawDescGZIP() []byte {
	file_connOpen_proto_rawDescOnce.Do(func() {
		file_connOpen_proto_rawDescData = protoimpl.X.CompressGZIP(file_connOpen_proto_rawDescData)
	})
	return file_connOpen_proto_rawDescData
}

var file_connOpen_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_connOpen_proto_goTypes = []interface{}{
	(*ConnOpen)(nil), // 0: netsvr.connOpen.ConnOpen
}
var file_connOpen_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_connOpen_proto_init() }
func file_connOpen_proto_init() {
	if File_connOpen_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_connOpen_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ConnOpen); i {
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
			RawDescriptor: file_connOpen_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_connOpen_proto_goTypes,
		DependencyIndexes: file_connOpen_proto_depIdxs,
		MessageInfos:      file_connOpen_proto_msgTypes,
	}.Build()
	File_connOpen_proto = out.File
	file_connOpen_proto_rawDesc = nil
	file_connOpen_proto_goTypes = nil
	file_connOpen_proto_depIdxs = nil
}
