// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.34.2
// 	protoc        v4.24.4
// source: build.proto

package protos2

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	structpb "google.golang.org/protobuf/types/known/structpb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type EnvType int32

const (
	EnvType_local      EnvType = 0
	EnvType_build      EnvType = 1
	EnvType_production EnvType = 2
)

// Enum value maps for EnvType.
var (
	EnvType_name = map[int32]string{
		0: "local",
		1: "build",
		2: "production",
	}
	EnvType_value = map[string]int32{
		"local":      0,
		"build":      1,
		"production": 2,
	}
)

func (x EnvType) Enum() *EnvType {
	p := new(EnvType)
	*p = x
	return p
}

func (x EnvType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (EnvType) Descriptor() protoreflect.EnumDescriptor {
	return file_build_proto_enumTypes[0].Descriptor()
}

func (EnvType) Type() protoreflect.EnumType {
	return &file_build_proto_enumTypes[0]
}

func (x EnvType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use EnvType.Descriptor instead.
func (EnvType) EnumDescriptor() ([]byte, []int) {
	return file_build_proto_rawDescGZIP(), []int{0}
}

type BuildType int32

const (
	BuildType_GoLang  BuildType = 0
	BuildType_Maven   BuildType = 1
	BuildType_Python  BuildType = 2
	BuildType_Generic BuildType = 3
)

// Enum value maps for BuildType.
var (
	BuildType_name = map[int32]string{
		0: "GoLang",
		1: "Maven",
		2: "Python",
		3: "Generic",
	}
	BuildType_value = map[string]int32{
		"GoLang":  0,
		"Maven":   1,
		"Python":  2,
		"Generic": 3,
	}
)

func (x BuildType) Enum() *BuildType {
	p := new(BuildType)
	*p = x
	return p
}

func (x BuildType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (BuildType) Descriptor() protoreflect.EnumDescriptor {
	return file_build_proto_enumTypes[1].Descriptor()
}

func (BuildType) Type() protoreflect.EnumType {
	return &file_build_proto_enumTypes[1]
}

func (x BuildType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use BuildType.Descriptor instead.
func (BuildType) EnumDescriptor() ([]byte, []int) {
	return file_build_proto_rawDescGZIP(), []int{1}
}

type RuntimeType int32

const (
	RuntimeType_Docker RuntimeType = 0
	RuntimeType_Podman RuntimeType = 1
)

// Enum value maps for RuntimeType.
var (
	RuntimeType_name = map[int32]string{
		0: "Docker",
		1: "Podman",
	}
	RuntimeType_value = map[string]int32{
		"Docker": 0,
		"Podman": 1,
	}
)

func (x RuntimeType) Enum() *RuntimeType {
	p := new(RuntimeType)
	*p = x
	return p
}

func (x RuntimeType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (RuntimeType) Descriptor() protoreflect.EnumDescriptor {
	return file_build_proto_enumTypes[2].Descriptor()
}

func (RuntimeType) Type() protoreflect.EnumType {
	return &file_build_proto_enumTypes[2]
}

func (x RuntimeType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use RuntimeType.Descriptor instead.
func (RuntimeType) EnumDescriptor() ([]byte, []int) {
	return file_build_proto_rawDescGZIP(), []int{2}
}

type BuildArgs struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Application    string                         `protobuf:"bytes,1,opt,name=Application,proto3" json:"Application,omitempty"`
	Environment    EnvType                        `protobuf:"varint,2,opt,name=Environment,proto3,enum=protos2.EnvType" json:"Environment,omitempty"`
	Properties     map[string]*structpb.ListValue `protobuf:"bytes,3,rep,name=Properties,proto3" json:"Properties,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	File           string                         `protobuf:"bytes,4,opt,name=File,proto3" json:"File,omitempty"`
	Folder         string                         `protobuf:"bytes,5,opt,name=Folder,proto3" json:"Folder,omitempty"`
	Image          string                         `protobuf:"bytes,6,opt,name=Image,proto3" json:"Image,omitempty"`
	ImageTag       string                         `protobuf:"bytes,7,opt,name=ImageTag,proto3" json:"ImageTag,omitempty"`
	BuildType      BuildType                      `protobuf:"varint,8,opt,name=BuildType,proto3,enum=protos2.BuildType" json:"BuildType,omitempty"`
	RuntimeType    RuntimeType                    `protobuf:"varint,9,opt,name=RuntimeType,proto3,enum=protos2.RuntimeType" json:"RuntimeType,omitempty"`
	Organization   string                         `protobuf:"bytes,10,opt,name=Organization,proto3" json:"Organization,omitempty"`
	Platform       string                         `protobuf:"bytes,11,opt,name=Platform,proto3" json:"Platform,omitempty"`
	Repository     string                         `protobuf:"bytes,12,opt,name=Repository,proto3" json:"Repository,omitempty"`
	Registry       string                         `protobuf:"bytes,13,opt,name=Registry,proto3" json:"Registry,omitempty"`
	SourcePackages []string                       `protobuf:"bytes,14,rep,name=SourcePackages,proto3" json:"SourcePackages,omitempty"`
	SourceFiles    []string                       `protobuf:"bytes,15,rep,name=SourceFiles,proto3" json:"SourceFiles,omitempty"`
	Verbose        bool                           `protobuf:"varint,16,opt,name=Verbose,proto3" json:"Verbose,omitempty"`
}

func (x *BuildArgs) Reset() {
	*x = BuildArgs{}
	if protoimpl.UnsafeEnabled {
		mi := &file_build_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BuildArgs) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BuildArgs) ProtoMessage() {}

func (x *BuildArgs) ProtoReflect() protoreflect.Message {
	mi := &file_build_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BuildArgs.ProtoReflect.Descriptor instead.
func (*BuildArgs) Descriptor() ([]byte, []int) {
	return file_build_proto_rawDescGZIP(), []int{0}
}

func (x *BuildArgs) GetApplication() string {
	if x != nil {
		return x.Application
	}
	return ""
}

func (x *BuildArgs) GetEnvironment() EnvType {
	if x != nil {
		return x.Environment
	}
	return EnvType_local
}

func (x *BuildArgs) GetProperties() map[string]*structpb.ListValue {
	if x != nil {
		return x.Properties
	}
	return nil
}

func (x *BuildArgs) GetFile() string {
	if x != nil {
		return x.File
	}
	return ""
}

func (x *BuildArgs) GetFolder() string {
	if x != nil {
		return x.Folder
	}
	return ""
}

func (x *BuildArgs) GetImage() string {
	if x != nil {
		return x.Image
	}
	return ""
}

func (x *BuildArgs) GetImageTag() string {
	if x != nil {
		return x.ImageTag
	}
	return ""
}

func (x *BuildArgs) GetBuildType() BuildType {
	if x != nil {
		return x.BuildType
	}
	return BuildType_GoLang
}

func (x *BuildArgs) GetRuntimeType() RuntimeType {
	if x != nil {
		return x.RuntimeType
	}
	return RuntimeType_Docker
}

func (x *BuildArgs) GetOrganization() string {
	if x != nil {
		return x.Organization
	}
	return ""
}

func (x *BuildArgs) GetPlatform() string {
	if x != nil {
		return x.Platform
	}
	return ""
}

func (x *BuildArgs) GetRepository() string {
	if x != nil {
		return x.Repository
	}
	return ""
}

func (x *BuildArgs) GetRegistry() string {
	if x != nil {
		return x.Registry
	}
	return ""
}

func (x *BuildArgs) GetSourcePackages() []string {
	if x != nil {
		return x.SourcePackages
	}
	return nil
}

func (x *BuildArgs) GetSourceFiles() []string {
	if x != nil {
		return x.SourceFiles
	}
	return nil
}

func (x *BuildArgs) GetVerbose() bool {
	if x != nil {
		return x.Verbose
	}
	return false
}

type Empty struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *Empty) Reset() {
	*x = Empty{}
	if protoimpl.UnsafeEnabled {
		mi := &file_build_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Empty) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Empty) ProtoMessage() {}

func (x *Empty) ProtoReflect() protoreflect.Message {
	mi := &file_build_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Empty.ProtoReflect.Descriptor instead.
func (*Empty) Descriptor() ([]byte, []int) {
	return file_build_proto_rawDescGZIP(), []int{1}
}

type BuildArgsResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Args []*BuildArgs `protobuf:"bytes,1,rep,name=args,proto3" json:"args,omitempty"`
}

func (x *BuildArgsResponse) Reset() {
	*x = BuildArgsResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_build_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BuildArgsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BuildArgsResponse) ProtoMessage() {}

func (x *BuildArgsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_build_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BuildArgsResponse.ProtoReflect.Descriptor instead.
func (*BuildArgsResponse) Descriptor() ([]byte, []int) {
	return file_build_proto_rawDescGZIP(), []int{2}
}

func (x *BuildArgsResponse) GetArgs() []*BuildArgs {
	if x != nil {
		return x.Args
	}
	return nil
}

var File_build_proto protoreflect.FileDescriptor

var file_build_proto_rawDesc = []byte{
	0x0a, 0x0b, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x07, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x73, 0x32, 0x1a, 0x1c, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x73, 0x74, 0x72, 0x75, 0x63, 0x74, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x22, 0xa8, 0x05, 0x0a, 0x09, 0x42, 0x75, 0x69, 0x6c, 0x64, 0x41, 0x72,
	0x67, 0x73, 0x12, 0x20, 0x0a, 0x0b, 0x41, 0x70, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x41, 0x70, 0x70, 0x6c, 0x69, 0x63, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x12, 0x32, 0x0a, 0x0b, 0x45, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d,
	0x65, 0x6e, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x10, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x73, 0x32, 0x2e, 0x45, 0x6e, 0x76, 0x54, 0x79, 0x70, 0x65, 0x52, 0x0b, 0x45, 0x6e, 0x76,
	0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x12, 0x42, 0x0a, 0x0a, 0x50, 0x72, 0x6f, 0x70,
	0x65, 0x72, 0x74, 0x69, 0x65, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x22, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x73, 0x32, 0x2e, 0x42, 0x75, 0x69, 0x6c, 0x64, 0x41, 0x72, 0x67, 0x73,
	0x2e, 0x50, 0x72, 0x6f, 0x70, 0x65, 0x72, 0x74, 0x69, 0x65, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79,
	0x52, 0x0a, 0x50, 0x72, 0x6f, 0x70, 0x65, 0x72, 0x74, 0x69, 0x65, 0x73, 0x12, 0x12, 0x0a, 0x04,
	0x46, 0x69, 0x6c, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x46, 0x69, 0x6c, 0x65,
	0x12, 0x16, 0x0a, 0x06, 0x46, 0x6f, 0x6c, 0x64, 0x65, 0x72, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x06, 0x46, 0x6f, 0x6c, 0x64, 0x65, 0x72, 0x12, 0x14, 0x0a, 0x05, 0x49, 0x6d, 0x61, 0x67,
	0x65, 0x18, 0x06, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x49, 0x6d, 0x61, 0x67, 0x65, 0x12, 0x1a,
	0x0a, 0x08, 0x49, 0x6d, 0x61, 0x67, 0x65, 0x54, 0x61, 0x67, 0x18, 0x07, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x08, 0x49, 0x6d, 0x61, 0x67, 0x65, 0x54, 0x61, 0x67, 0x12, 0x30, 0x0a, 0x09, 0x42, 0x75,
	0x69, 0x6c, 0x64, 0x54, 0x79, 0x70, 0x65, 0x18, 0x08, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x12, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x32, 0x2e, 0x42, 0x75, 0x69, 0x6c, 0x64, 0x54, 0x79, 0x70,
	0x65, 0x52, 0x09, 0x42, 0x75, 0x69, 0x6c, 0x64, 0x54, 0x79, 0x70, 0x65, 0x12, 0x36, 0x0a, 0x0b,
	0x52, 0x75, 0x6e, 0x74, 0x69, 0x6d, 0x65, 0x54, 0x79, 0x70, 0x65, 0x18, 0x09, 0x20, 0x01, 0x28,
	0x0e, 0x32, 0x14, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x32, 0x2e, 0x52, 0x75, 0x6e, 0x74,
	0x69, 0x6d, 0x65, 0x54, 0x79, 0x70, 0x65, 0x52, 0x0b, 0x52, 0x75, 0x6e, 0x74, 0x69, 0x6d, 0x65,
	0x54, 0x79, 0x70, 0x65, 0x12, 0x22, 0x0a, 0x0c, 0x4f, 0x72, 0x67, 0x61, 0x6e, 0x69, 0x7a, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x18, 0x0a, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x4f, 0x72, 0x67, 0x61,
	0x6e, 0x69, 0x7a, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x1a, 0x0a, 0x08, 0x50, 0x6c, 0x61, 0x74,
	0x66, 0x6f, 0x72, 0x6d, 0x18, 0x0b, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x50, 0x6c, 0x61, 0x74,
	0x66, 0x6f, 0x72, 0x6d, 0x12, 0x1e, 0x0a, 0x0a, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f,
	0x72, 0x79, 0x18, 0x0c, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69,
	0x74, 0x6f, 0x72, 0x79, 0x12, 0x1a, 0x0a, 0x08, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79,
	0x18, 0x0d, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79,
	0x12, 0x26, 0x0a, 0x0e, 0x53, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x50, 0x61, 0x63, 0x6b, 0x61, 0x67,
	0x65, 0x73, 0x18, 0x0e, 0x20, 0x03, 0x28, 0x09, 0x52, 0x0e, 0x53, 0x6f, 0x75, 0x72, 0x63, 0x65,
	0x50, 0x61, 0x63, 0x6b, 0x61, 0x67, 0x65, 0x73, 0x12, 0x20, 0x0a, 0x0b, 0x53, 0x6f, 0x75, 0x72,
	0x63, 0x65, 0x46, 0x69, 0x6c, 0x65, 0x73, 0x18, 0x0f, 0x20, 0x03, 0x28, 0x09, 0x52, 0x0b, 0x53,
	0x6f, 0x75, 0x72, 0x63, 0x65, 0x46, 0x69, 0x6c, 0x65, 0x73, 0x12, 0x18, 0x0a, 0x07, 0x56, 0x65,
	0x72, 0x62, 0x6f, 0x73, 0x65, 0x18, 0x10, 0x20, 0x01, 0x28, 0x08, 0x52, 0x07, 0x56, 0x65, 0x72,
	0x62, 0x6f, 0x73, 0x65, 0x1a, 0x59, 0x0a, 0x0f, 0x50, 0x72, 0x6f, 0x70, 0x65, 0x72, 0x74, 0x69,
	0x65, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x30, 0x0a, 0x05, 0x76, 0x61, 0x6c,
	0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x4c, 0x69, 0x73, 0x74, 0x56,
	0x61, 0x6c, 0x75, 0x65, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22,
	0x07, 0x0a, 0x05, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x22, 0x3b, 0x0a, 0x11, 0x42, 0x75, 0x69, 0x6c,
	0x64, 0x41, 0x72, 0x67, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x26, 0x0a,
	0x04, 0x61, 0x72, 0x67, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x73, 0x32, 0x2e, 0x42, 0x75, 0x69, 0x6c, 0x64, 0x41, 0x72, 0x67, 0x73, 0x52,
	0x04, 0x61, 0x72, 0x67, 0x73, 0x2a, 0x2f, 0x0a, 0x07, 0x45, 0x6e, 0x76, 0x54, 0x79, 0x70, 0x65,
	0x12, 0x09, 0x0a, 0x05, 0x6c, 0x6f, 0x63, 0x61, 0x6c, 0x10, 0x00, 0x12, 0x09, 0x0a, 0x05, 0x62,
	0x75, 0x69, 0x6c, 0x64, 0x10, 0x01, 0x12, 0x0e, 0x0a, 0x0a, 0x70, 0x72, 0x6f, 0x64, 0x75, 0x63,
	0x74, 0x69, 0x6f, 0x6e, 0x10, 0x02, 0x2a, 0x3b, 0x0a, 0x09, 0x42, 0x75, 0x69, 0x6c, 0x64, 0x54,
	0x79, 0x70, 0x65, 0x12, 0x0a, 0x0a, 0x06, 0x47, 0x6f, 0x4c, 0x61, 0x6e, 0x67, 0x10, 0x00, 0x12,
	0x09, 0x0a, 0x05, 0x4d, 0x61, 0x76, 0x65, 0x6e, 0x10, 0x01, 0x12, 0x0a, 0x0a, 0x06, 0x50, 0x79,
	0x74, 0x68, 0x6f, 0x6e, 0x10, 0x02, 0x12, 0x0b, 0x0a, 0x07, 0x47, 0x65, 0x6e, 0x65, 0x72, 0x69,
	0x63, 0x10, 0x03, 0x2a, 0x25, 0x0a, 0x0b, 0x52, 0x75, 0x6e, 0x74, 0x69, 0x6d, 0x65, 0x54, 0x79,
	0x70, 0x65, 0x12, 0x0a, 0x0a, 0x06, 0x44, 0x6f, 0x63, 0x6b, 0x65, 0x72, 0x10, 0x00, 0x12, 0x0a,
	0x0a, 0x06, 0x50, 0x6f, 0x64, 0x6d, 0x61, 0x6e, 0x10, 0x01, 0x32, 0x4c, 0x0a, 0x12, 0x43, 0x6f,
	0x6e, 0x74, 0x61, 0x69, 0x6e, 0x69, 0x66, 0x79, 0x43, 0x49, 0x45, 0x6e, 0x67, 0x69, 0x6e, 0x65,
	0x12, 0x36, 0x0a, 0x08, 0x47, 0x65, 0x74, 0x42, 0x75, 0x69, 0x6c, 0x64, 0x12, 0x0e, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x73, 0x32, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x1a, 0x1a, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x73, 0x32, 0x2e, 0x42, 0x75, 0x69, 0x6c, 0x64, 0x41, 0x72, 0x67, 0x73,
	0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x42, 0x0c, 0x5a, 0x0a, 0x2e, 0x2e, 0x2f, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x73, 0x32, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_build_proto_rawDescOnce sync.Once
	file_build_proto_rawDescData = file_build_proto_rawDesc
)

func file_build_proto_rawDescGZIP() []byte {
	file_build_proto_rawDescOnce.Do(func() {
		file_build_proto_rawDescData = protoimpl.X.CompressGZIP(file_build_proto_rawDescData)
	})
	return file_build_proto_rawDescData
}

var file_build_proto_enumTypes = make([]protoimpl.EnumInfo, 3)
var file_build_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_build_proto_goTypes = []any{
	(EnvType)(0),               // 0: protos2.EnvType
	(BuildType)(0),             // 1: protos2.BuildType
	(RuntimeType)(0),           // 2: protos2.RuntimeType
	(*BuildArgs)(nil),          // 3: protos2.BuildArgs
	(*Empty)(nil),              // 4: protos2.Empty
	(*BuildArgsResponse)(nil),  // 5: protos2.BuildArgsResponse
	nil,                        // 6: protos2.BuildArgs.PropertiesEntry
	(*structpb.ListValue)(nil), // 7: google.protobuf.ListValue
}
var file_build_proto_depIdxs = []int32{
	0, // 0: protos2.BuildArgs.Environment:type_name -> protos2.EnvType
	6, // 1: protos2.BuildArgs.Properties:type_name -> protos2.BuildArgs.PropertiesEntry
	1, // 2: protos2.BuildArgs.BuildType:type_name -> protos2.BuildType
	2, // 3: protos2.BuildArgs.RuntimeType:type_name -> protos2.RuntimeType
	3, // 4: protos2.BuildArgsResponse.args:type_name -> protos2.BuildArgs
	7, // 5: protos2.BuildArgs.PropertiesEntry.value:type_name -> google.protobuf.ListValue
	4, // 6: protos2.ContainifyCIEngine.GetBuild:input_type -> protos2.Empty
	5, // 7: protos2.ContainifyCIEngine.GetBuild:output_type -> protos2.BuildArgsResponse
	7, // [7:8] is the sub-list for method output_type
	6, // [6:7] is the sub-list for method input_type
	6, // [6:6] is the sub-list for extension type_name
	6, // [6:6] is the sub-list for extension extendee
	0, // [0:6] is the sub-list for field type_name
}

func init() { file_build_proto_init() }
func file_build_proto_init() {
	if File_build_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_build_proto_msgTypes[0].Exporter = func(v any, i int) any {
			switch v := v.(*BuildArgs); i {
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
		file_build_proto_msgTypes[1].Exporter = func(v any, i int) any {
			switch v := v.(*Empty); i {
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
		file_build_proto_msgTypes[2].Exporter = func(v any, i int) any {
			switch v := v.(*BuildArgsResponse); i {
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
			RawDescriptor: file_build_proto_rawDesc,
			NumEnums:      3,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_build_proto_goTypes,
		DependencyIndexes: file_build_proto_depIdxs,
		EnumInfos:         file_build_proto_enumTypes,
		MessageInfos:      file_build_proto_msgTypes,
	}.Build()
	File_build_proto = out.File
	file_build_proto_rawDesc = nil
	file_build_proto_goTypes = nil
	file_build_proto_depIdxs = nil
}
