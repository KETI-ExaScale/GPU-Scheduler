// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.1
// 	protoc        v3.15.8
// source: proto/data.proto

package user

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

type NodeMessage struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	NodeName         string `protobuf:"bytes,1,opt,name=node_name,json=nodeName,proto3" json:"node_name,omitempty"`
	NodeTotalcpu     int64  `protobuf:"varint,2,opt,name=node_totalcpu,json=nodeTotalcpu,proto3" json:"node_totalcpu,omitempty"`
	NodeCpu          int64  `protobuf:"varint,3,opt,name=node_cpu,json=nodeCpu,proto3" json:"node_cpu,omitempty"`
	GpuCount         int64  `protobuf:"varint,4,opt,name=gpu_count,json=gpuCount,proto3" json:"gpu_count,omitempty"`
	NodeTotalmemory  int64  `protobuf:"varint,5,opt,name=node_totalmemory,json=nodeTotalmemory,proto3" json:"node_totalmemory,omitempty"`
	NodeMemory       int64  `protobuf:"varint,6,opt,name=node_memory,json=nodeMemory,proto3" json:"node_memory,omitempty"`
	NodeTotalstorage int64  `protobuf:"varint,7,opt,name=node_totalstorage,json=nodeTotalstorage,proto3" json:"node_totalstorage,omitempty"`
	NodeStorage      int64  `protobuf:"varint,8,opt,name=node_storage,json=nodeStorage,proto3" json:"node_storage,omitempty"`
	GpuUuid          string `protobuf:"bytes,9,opt,name=gpu_uuid,json=gpuUuid,proto3" json:"gpu_uuid,omitempty"`
	MaxGpuMemory     int64  `protobuf:"varint,10,opt,name=max_gpu_memory,json=maxGpuMemory,proto3" json:"max_gpu_memory,omitempty"`
}

func (x *NodeMessage) Reset() {
	*x = NodeMessage{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_data_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *NodeMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NodeMessage) ProtoMessage() {}

func (x *NodeMessage) ProtoReflect() protoreflect.Message {
	mi := &file_proto_data_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NodeMessage.ProtoReflect.Descriptor instead.
func (*NodeMessage) Descriptor() ([]byte, []int) {
	return file_proto_data_proto_rawDescGZIP(), []int{0}
}

func (x *NodeMessage) GetNodeName() string {
	if x != nil {
		return x.NodeName
	}
	return ""
}

func (x *NodeMessage) GetNodeTotalcpu() int64 {
	if x != nil {
		return x.NodeTotalcpu
	}
	return 0
}

func (x *NodeMessage) GetNodeCpu() int64 {
	if x != nil {
		return x.NodeCpu
	}
	return 0
}

func (x *NodeMessage) GetGpuCount() int64 {
	if x != nil {
		return x.GpuCount
	}
	return 0
}

func (x *NodeMessage) GetNodeTotalmemory() int64 {
	if x != nil {
		return x.NodeTotalmemory
	}
	return 0
}

func (x *NodeMessage) GetNodeMemory() int64 {
	if x != nil {
		return x.NodeMemory
	}
	return 0
}

func (x *NodeMessage) GetNodeTotalstorage() int64 {
	if x != nil {
		return x.NodeTotalstorage
	}
	return 0
}

func (x *NodeMessage) GetNodeStorage() int64 {
	if x != nil {
		return x.NodeStorage
	}
	return 0
}

func (x *NodeMessage) GetGpuUuid() string {
	if x != nil {
		return x.GpuUuid
	}
	return ""
}

func (x *NodeMessage) GetMaxGpuMemory() int64 {
	if x != nil {
		return x.MaxGpuMemory
	}
	return 0
}

type GetNodeRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *GetNodeRequest) Reset() {
	*x = GetNodeRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_data_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetNodeRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetNodeRequest) ProtoMessage() {}

func (x *GetNodeRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_data_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetNodeRequest.ProtoReflect.Descriptor instead.
func (*GetNodeRequest) Descriptor() ([]byte, []int) {
	return file_proto_data_proto_rawDescGZIP(), []int{1}
}

type GetNodeResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	NodeMessage *NodeMessage `protobuf:"bytes,1,opt,name=node_message,json=nodeMessage,proto3" json:"node_message,omitempty"`
}

func (x *GetNodeResponse) Reset() {
	*x = GetNodeResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_data_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetNodeResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetNodeResponse) ProtoMessage() {}

func (x *GetNodeResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_data_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetNodeResponse.ProtoReflect.Descriptor instead.
func (*GetNodeResponse) Descriptor() ([]byte, []int) {
	return file_proto_data_proto_rawDescGZIP(), []int{2}
}

func (x *GetNodeResponse) GetNodeMessage() *NodeMessage {
	if x != nil {
		return x.NodeMessage
	}
	return nil
}

type GPUMessage struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	GpuUuid  string `protobuf:"bytes,1,opt,name=gpu_uuid,json=gpuUuid,proto3" json:"gpu_uuid,omitempty"`
	GpuTotal uint64 `protobuf:"varint,2,opt,name=gpu_total,json=gpuTotal,proto3" json:"gpu_total,omitempty"`
	GpuUsed  uint64 `protobuf:"varint,3,opt,name=gpu_used,json=gpuUsed,proto3" json:"gpu_used,omitempty"`
	GpuFree  uint64 `protobuf:"varint,4,opt,name=gpu_free,json=gpuFree,proto3" json:"gpu_free,omitempty"`
	GpuName  string `protobuf:"bytes,5,opt,name=gpu_name,json=gpuName,proto3" json:"gpu_name,omitempty"`
	GpuIndex int64  `protobuf:"varint,6,opt,name=gpu_index,json=gpuIndex,proto3" json:"gpu_index,omitempty"`
	GpuTemp  int64  `protobuf:"varint,7,opt,name=gpu_temp,json=gpuTemp,proto3" json:"gpu_temp,omitempty"`
	GpuPower int64  `protobuf:"varint,8,opt,name=gpu_power,json=gpuPower,proto3" json:"gpu_power,omitempty"`
	MpsCount int64  `protobuf:"varint,9,opt,name=mps_count,json=mpsCount,proto3" json:"mps_count,omitempty"`
}

func (x *GPUMessage) Reset() {
	*x = GPUMessage{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_data_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GPUMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GPUMessage) ProtoMessage() {}

func (x *GPUMessage) ProtoReflect() protoreflect.Message {
	mi := &file_proto_data_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GPUMessage.ProtoReflect.Descriptor instead.
func (*GPUMessage) Descriptor() ([]byte, []int) {
	return file_proto_data_proto_rawDescGZIP(), []int{3}
}

func (x *GPUMessage) GetGpuUuid() string {
	if x != nil {
		return x.GpuUuid
	}
	return ""
}

func (x *GPUMessage) GetGpuTotal() uint64 {
	if x != nil {
		return x.GpuTotal
	}
	return 0
}

func (x *GPUMessage) GetGpuUsed() uint64 {
	if x != nil {
		return x.GpuUsed
	}
	return 0
}

func (x *GPUMessage) GetGpuFree() uint64 {
	if x != nil {
		return x.GpuFree
	}
	return 0
}

func (x *GPUMessage) GetGpuName() string {
	if x != nil {
		return x.GpuName
	}
	return ""
}

func (x *GPUMessage) GetGpuIndex() int64 {
	if x != nil {
		return x.GpuIndex
	}
	return 0
}

func (x *GPUMessage) GetGpuTemp() int64 {
	if x != nil {
		return x.GpuTemp
	}
	return 0
}

func (x *GPUMessage) GetGpuPower() int64 {
	if x != nil {
		return x.GpuPower
	}
	return 0
}

func (x *GPUMessage) GetMpsCount() int64 {
	if x != nil {
		return x.MpsCount
	}
	return 0
}

type GetGPURequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	GpuUuid string `protobuf:"bytes,1,opt,name=gpu_uuid,json=gpuUuid,proto3" json:"gpu_uuid,omitempty"`
}

func (x *GetGPURequest) Reset() {
	*x = GetGPURequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_data_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetGPURequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetGPURequest) ProtoMessage() {}

func (x *GetGPURequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_data_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetGPURequest.ProtoReflect.Descriptor instead.
func (*GetGPURequest) Descriptor() ([]byte, []int) {
	return file_proto_data_proto_rawDescGZIP(), []int{4}
}

func (x *GetGPURequest) GetGpuUuid() string {
	if x != nil {
		return x.GpuUuid
	}
	return ""
}

type GetGPUResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	GpuMessage *GPUMessage `protobuf:"bytes,1,opt,name=gpu_message,json=gpuMessage,proto3" json:"gpu_message,omitempty"`
}

func (x *GetGPUResponse) Reset() {
	*x = GetGPUResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_data_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetGPUResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetGPUResponse) ProtoMessage() {}

func (x *GetGPUResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_data_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetGPUResponse.ProtoReflect.Descriptor instead.
func (*GetGPUResponse) Descriptor() ([]byte, []int) {
	return file_proto_data_proto_rawDescGZIP(), []int{5}
}

func (x *GetGPUResponse) GetGpuMessage() *GPUMessage {
	if x != nil {
		return x.GpuMessage
	}
	return nil
}

var File_proto_data_proto protoreflect.FileDescriptor

var file_proto_data_proto_rawDesc = []byte{
	0x0a, 0x10, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x64, 0x61, 0x74, 0x61, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x12, 0x07, 0x76, 0x31, 0x2e, 0x75, 0x73, 0x65, 0x72, 0x22, 0xe4, 0x02, 0x0a, 0x0b,
	0x4e, 0x6f, 0x64, 0x65, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x1b, 0x0a, 0x09, 0x6e,
	0x6f, 0x64, 0x65, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08,
	0x6e, 0x6f, 0x64, 0x65, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x23, 0x0a, 0x0d, 0x6e, 0x6f, 0x64, 0x65,
	0x5f, 0x74, 0x6f, 0x74, 0x61, 0x6c, 0x63, 0x70, 0x75, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x52,
	0x0c, 0x6e, 0x6f, 0x64, 0x65, 0x54, 0x6f, 0x74, 0x61, 0x6c, 0x63, 0x70, 0x75, 0x12, 0x19, 0x0a,
	0x08, 0x6e, 0x6f, 0x64, 0x65, 0x5f, 0x63, 0x70, 0x75, 0x18, 0x03, 0x20, 0x01, 0x28, 0x03, 0x52,
	0x07, 0x6e, 0x6f, 0x64, 0x65, 0x43, 0x70, 0x75, 0x12, 0x1b, 0x0a, 0x09, 0x67, 0x70, 0x75, 0x5f,
	0x63, 0x6f, 0x75, 0x6e, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x03, 0x52, 0x08, 0x67, 0x70, 0x75,
	0x43, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x29, 0x0a, 0x10, 0x6e, 0x6f, 0x64, 0x65, 0x5f, 0x74, 0x6f,
	0x74, 0x61, 0x6c, 0x6d, 0x65, 0x6d, 0x6f, 0x72, 0x79, 0x18, 0x05, 0x20, 0x01, 0x28, 0x03, 0x52,
	0x0f, 0x6e, 0x6f, 0x64, 0x65, 0x54, 0x6f, 0x74, 0x61, 0x6c, 0x6d, 0x65, 0x6d, 0x6f, 0x72, 0x79,
	0x12, 0x1f, 0x0a, 0x0b, 0x6e, 0x6f, 0x64, 0x65, 0x5f, 0x6d, 0x65, 0x6d, 0x6f, 0x72, 0x79, 0x18,
	0x06, 0x20, 0x01, 0x28, 0x03, 0x52, 0x0a, 0x6e, 0x6f, 0x64, 0x65, 0x4d, 0x65, 0x6d, 0x6f, 0x72,
	0x79, 0x12, 0x2b, 0x0a, 0x11, 0x6e, 0x6f, 0x64, 0x65, 0x5f, 0x74, 0x6f, 0x74, 0x61, 0x6c, 0x73,
	0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x18, 0x07, 0x20, 0x01, 0x28, 0x03, 0x52, 0x10, 0x6e, 0x6f,
	0x64, 0x65, 0x54, 0x6f, 0x74, 0x61, 0x6c, 0x73, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x12, 0x21,
	0x0a, 0x0c, 0x6e, 0x6f, 0x64, 0x65, 0x5f, 0x73, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x18, 0x08,
	0x20, 0x01, 0x28, 0x03, 0x52, 0x0b, 0x6e, 0x6f, 0x64, 0x65, 0x53, 0x74, 0x6f, 0x72, 0x61, 0x67,
	0x65, 0x12, 0x19, 0x0a, 0x08, 0x67, 0x70, 0x75, 0x5f, 0x75, 0x75, 0x69, 0x64, 0x18, 0x09, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x07, 0x67, 0x70, 0x75, 0x55, 0x75, 0x69, 0x64, 0x12, 0x24, 0x0a, 0x0e,
	0x6d, 0x61, 0x78, 0x5f, 0x67, 0x70, 0x75, 0x5f, 0x6d, 0x65, 0x6d, 0x6f, 0x72, 0x79, 0x18, 0x0a,
	0x20, 0x01, 0x28, 0x03, 0x52, 0x0c, 0x6d, 0x61, 0x78, 0x47, 0x70, 0x75, 0x4d, 0x65, 0x6d, 0x6f,
	0x72, 0x79, 0x22, 0x10, 0x0a, 0x0e, 0x47, 0x65, 0x74, 0x4e, 0x6f, 0x64, 0x65, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x22, 0x4a, 0x0a, 0x0f, 0x47, 0x65, 0x74, 0x4e, 0x6f, 0x64, 0x65, 0x52,
	0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x37, 0x0a, 0x0c, 0x6e, 0x6f, 0x64, 0x65, 0x5f,
	0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x14, 0x2e,
	0x76, 0x31, 0x2e, 0x75, 0x73, 0x65, 0x72, 0x2e, 0x4e, 0x6f, 0x64, 0x65, 0x4d, 0x65, 0x73, 0x73,
	0x61, 0x67, 0x65, 0x52, 0x0b, 0x6e, 0x6f, 0x64, 0x65, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65,
	0x22, 0x87, 0x02, 0x0a, 0x0a, 0x47, 0x50, 0x55, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12,
	0x19, 0x0a, 0x08, 0x67, 0x70, 0x75, 0x5f, 0x75, 0x75, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x07, 0x67, 0x70, 0x75, 0x55, 0x75, 0x69, 0x64, 0x12, 0x1b, 0x0a, 0x09, 0x67, 0x70,
	0x75, 0x5f, 0x74, 0x6f, 0x74, 0x61, 0x6c, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04, 0x52, 0x08, 0x67,
	0x70, 0x75, 0x54, 0x6f, 0x74, 0x61, 0x6c, 0x12, 0x19, 0x0a, 0x08, 0x67, 0x70, 0x75, 0x5f, 0x75,
	0x73, 0x65, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x04, 0x52, 0x07, 0x67, 0x70, 0x75, 0x55, 0x73,
	0x65, 0x64, 0x12, 0x19, 0x0a, 0x08, 0x67, 0x70, 0x75, 0x5f, 0x66, 0x72, 0x65, 0x65, 0x18, 0x04,
	0x20, 0x01, 0x28, 0x04, 0x52, 0x07, 0x67, 0x70, 0x75, 0x46, 0x72, 0x65, 0x65, 0x12, 0x19, 0x0a,
	0x08, 0x67, 0x70, 0x75, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x07, 0x67, 0x70, 0x75, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x1b, 0x0a, 0x09, 0x67, 0x70, 0x75, 0x5f,
	0x69, 0x6e, 0x64, 0x65, 0x78, 0x18, 0x06, 0x20, 0x01, 0x28, 0x03, 0x52, 0x08, 0x67, 0x70, 0x75,
	0x49, 0x6e, 0x64, 0x65, 0x78, 0x12, 0x19, 0x0a, 0x08, 0x67, 0x70, 0x75, 0x5f, 0x74, 0x65, 0x6d,
	0x70, 0x18, 0x07, 0x20, 0x01, 0x28, 0x03, 0x52, 0x07, 0x67, 0x70, 0x75, 0x54, 0x65, 0x6d, 0x70,
	0x12, 0x1b, 0x0a, 0x09, 0x67, 0x70, 0x75, 0x5f, 0x70, 0x6f, 0x77, 0x65, 0x72, 0x18, 0x08, 0x20,
	0x01, 0x28, 0x03, 0x52, 0x08, 0x67, 0x70, 0x75, 0x50, 0x6f, 0x77, 0x65, 0x72, 0x12, 0x1b, 0x0a,
	0x09, 0x6d, 0x70, 0x73, 0x5f, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x18, 0x09, 0x20, 0x01, 0x28, 0x03,
	0x52, 0x08, 0x6d, 0x70, 0x73, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x22, 0x2a, 0x0a, 0x0d, 0x47, 0x65,
	0x74, 0x47, 0x50, 0x55, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x19, 0x0a, 0x08, 0x67,
	0x70, 0x75, 0x5f, 0x75, 0x75, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x67,
	0x70, 0x75, 0x55, 0x75, 0x69, 0x64, 0x22, 0x46, 0x0a, 0x0e, 0x47, 0x65, 0x74, 0x47, 0x50, 0x55,
	0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x34, 0x0a, 0x0b, 0x67, 0x70, 0x75, 0x5f,
	0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x13, 0x2e,
	0x76, 0x31, 0x2e, 0x75, 0x73, 0x65, 0x72, 0x2e, 0x47, 0x50, 0x55, 0x4d, 0x65, 0x73, 0x73, 0x61,
	0x67, 0x65, 0x52, 0x0a, 0x67, 0x70, 0x75, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x32, 0x7f,
	0x0a, 0x04, 0x55, 0x73, 0x65, 0x72, 0x12, 0x3c, 0x0a, 0x07, 0x47, 0x65, 0x74, 0x4e, 0x6f, 0x64,
	0x65, 0x12, 0x17, 0x2e, 0x76, 0x31, 0x2e, 0x75, 0x73, 0x65, 0x72, 0x2e, 0x47, 0x65, 0x74, 0x4e,
	0x6f, 0x64, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x18, 0x2e, 0x76, 0x31, 0x2e,
	0x75, 0x73, 0x65, 0x72, 0x2e, 0x47, 0x65, 0x74, 0x4e, 0x6f, 0x64, 0x65, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x12, 0x39, 0x0a, 0x06, 0x47, 0x65, 0x74, 0x47, 0x50, 0x55, 0x12, 0x16,
	0x2e, 0x76, 0x31, 0x2e, 0x75, 0x73, 0x65, 0x72, 0x2e, 0x47, 0x65, 0x74, 0x47, 0x50, 0x55, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x17, 0x2e, 0x76, 0x31, 0x2e, 0x75, 0x73, 0x65, 0x72,
	0x2e, 0x47, 0x65, 0x74, 0x47, 0x50, 0x55, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x42,
	0x1b, 0x5a, 0x19, 0x67, 0x72, 0x70, 0x63, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x2f, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x73, 0x2f, 0x76, 0x31, 0x2f, 0x75, 0x73, 0x65, 0x72, 0x62, 0x06, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_proto_data_proto_rawDescOnce sync.Once
	file_proto_data_proto_rawDescData = file_proto_data_proto_rawDesc
)

func file_proto_data_proto_rawDescGZIP() []byte {
	file_proto_data_proto_rawDescOnce.Do(func() {
		file_proto_data_proto_rawDescData = protoimpl.X.CompressGZIP(file_proto_data_proto_rawDescData)
	})
	return file_proto_data_proto_rawDescData
}

var file_proto_data_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_proto_data_proto_goTypes = []interface{}{
	(*NodeMessage)(nil),     // 0: v1.user.NodeMessage
	(*GetNodeRequest)(nil),  // 1: v1.user.GetNodeRequest
	(*GetNodeResponse)(nil), // 2: v1.user.GetNodeResponse
	(*GPUMessage)(nil),      // 3: v1.user.GPUMessage
	(*GetGPURequest)(nil),   // 4: v1.user.GetGPURequest
	(*GetGPUResponse)(nil),  // 5: v1.user.GetGPUResponse
}
var file_proto_data_proto_depIdxs = []int32{
	0, // 0: v1.user.GetNodeResponse.node_message:type_name -> v1.user.NodeMessage
	3, // 1: v1.user.GetGPUResponse.gpu_message:type_name -> v1.user.GPUMessage
	1, // 2: v1.user.User.GetNode:input_type -> v1.user.GetNodeRequest
	4, // 3: v1.user.User.GetGPU:input_type -> v1.user.GetGPURequest
	2, // 4: v1.user.User.GetNode:output_type -> v1.user.GetNodeResponse
	5, // 5: v1.user.User.GetGPU:output_type -> v1.user.GetGPUResponse
	4, // [4:6] is the sub-list for method output_type
	2, // [2:4] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_proto_data_proto_init() }
func file_proto_data_proto_init() {
	if File_proto_data_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_proto_data_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*NodeMessage); i {
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
		file_proto_data_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetNodeRequest); i {
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
		file_proto_data_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetNodeResponse); i {
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
		file_proto_data_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GPUMessage); i {
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
		file_proto_data_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetGPURequest); i {
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
		file_proto_data_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetGPUResponse); i {
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
			RawDescriptor: file_proto_data_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_proto_data_proto_goTypes,
		DependencyIndexes: file_proto_data_proto_depIdxs,
		MessageInfos:      file_proto_data_proto_msgTypes,
	}.Build()
	File_proto_data_proto = out.File
	file_proto_data_proto_rawDesc = nil
	file_proto_data_proto_goTypes = nil
	file_proto_data_proto_depIdxs = nil
}
