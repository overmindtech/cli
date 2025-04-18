// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.4
// 	protoc        (unknown)
// source: auth0support.proto

package sdp

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	_ "google.golang.org/protobuf/types/known/structpb"
	_ "google.golang.org/protobuf/types/known/timestamppb"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Auth0CreateUserRequest struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// The Auth0 User ID
	UserId string `protobuf:"bytes,1,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
	// The user's email address
	Email string `protobuf:"bytes,2,opt,name=email,proto3" json:"email,omitempty"`
	// The user's full name. This will be split and stored as first_name and
	// last_name internally. It is provided for convenience since some social
	// providers do not provide first_name and last_name fields. If `first_name`
	// and `last_name` are provided, this field will be ignored.
	Name string `protobuf:"bytes,3,opt,name=name,proto3" json:"name,omitempty"`
	// Whether the user's email address has been verified
	EmailVerified bool `protobuf:"varint,4,opt,name=email_verified,json=emailVerified,proto3" json:"email_verified,omitempty"`
	// The user's first name
	FirstName string `protobuf:"bytes,5,opt,name=first_name,json=firstName,proto3" json:"first_name,omitempty"`
	// The user's last name
	LastName string `protobuf:"bytes,6,opt,name=last_name,json=lastName,proto3" json:"last_name,omitempty"`
	// The user's connection id
	ConnectionId  string `protobuf:"bytes,7,opt,name=connection_id,json=connectionId,proto3" json:"connection_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Auth0CreateUserRequest) Reset() {
	*x = Auth0CreateUserRequest{}
	mi := &file_auth0support_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Auth0CreateUserRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Auth0CreateUserRequest) ProtoMessage() {}

func (x *Auth0CreateUserRequest) ProtoReflect() protoreflect.Message {
	mi := &file_auth0support_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Auth0CreateUserRequest.ProtoReflect.Descriptor instead.
func (*Auth0CreateUserRequest) Descriptor() ([]byte, []int) {
	return file_auth0support_proto_rawDescGZIP(), []int{0}
}

func (x *Auth0CreateUserRequest) GetUserId() string {
	if x != nil {
		return x.UserId
	}
	return ""
}

func (x *Auth0CreateUserRequest) GetEmail() string {
	if x != nil {
		return x.Email
	}
	return ""
}

func (x *Auth0CreateUserRequest) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Auth0CreateUserRequest) GetEmailVerified() bool {
	if x != nil {
		return x.EmailVerified
	}
	return false
}

func (x *Auth0CreateUserRequest) GetFirstName() string {
	if x != nil {
		return x.FirstName
	}
	return ""
}

func (x *Auth0CreateUserRequest) GetLastName() string {
	if x != nil {
		return x.LastName
	}
	return ""
}

func (x *Auth0CreateUserRequest) GetConnectionId() string {
	if x != nil {
		return x.ConnectionId
	}
	return ""
}

type Auth0CreateUserResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	OrgId         string                 `protobuf:"bytes,1,opt,name=org_id,json=orgId,proto3" json:"org_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Auth0CreateUserResponse) Reset() {
	*x = Auth0CreateUserResponse{}
	mi := &file_auth0support_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Auth0CreateUserResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Auth0CreateUserResponse) ProtoMessage() {}

func (x *Auth0CreateUserResponse) ProtoReflect() protoreflect.Message {
	mi := &file_auth0support_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Auth0CreateUserResponse.ProtoReflect.Descriptor instead.
func (*Auth0CreateUserResponse) Descriptor() ([]byte, []int) {
	return file_auth0support_proto_rawDescGZIP(), []int{1}
}

func (x *Auth0CreateUserResponse) GetOrgId() string {
	if x != nil {
		return x.OrgId
	}
	return ""
}

var File_auth0support_proto protoreflect.FileDescriptor

var file_auth0support_proto_rawDesc = string([]byte{
	0x0a, 0x12, 0x61, 0x75, 0x74, 0x68, 0x30, 0x73, 0x75, 0x70, 0x70, 0x6f, 0x72, 0x74, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0c, 0x61, 0x75, 0x74, 0x68, 0x30, 0x73, 0x75, 0x70, 0x70, 0x6f,
	0x72, 0x74, 0x1a, 0x1c, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2f, 0x73, 0x74, 0x72, 0x75, 0x63, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x1a, 0x0d, 0x61, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x22, 0xe3, 0x01, 0x0a, 0x16, 0x41, 0x75, 0x74, 0x68, 0x30, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65,
	0x55, 0x73, 0x65, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x17, 0x0a, 0x07, 0x75,
	0x73, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x75, 0x73,
	0x65, 0x72, 0x49, 0x64, 0x12, 0x14, 0x0a, 0x05, 0x65, 0x6d, 0x61, 0x69, 0x6c, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x05, 0x65, 0x6d, 0x61, 0x69, 0x6c, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61,
	0x6d, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x25,
	0x0a, 0x0e, 0x65, 0x6d, 0x61, 0x69, 0x6c, 0x5f, 0x76, 0x65, 0x72, 0x69, 0x66, 0x69, 0x65, 0x64,
	0x18, 0x04, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0d, 0x65, 0x6d, 0x61, 0x69, 0x6c, 0x56, 0x65, 0x72,
	0x69, 0x66, 0x69, 0x65, 0x64, 0x12, 0x1d, 0x0a, 0x0a, 0x66, 0x69, 0x72, 0x73, 0x74, 0x5f, 0x6e,
	0x61, 0x6d, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x66, 0x69, 0x72, 0x73, 0x74,
	0x4e, 0x61, 0x6d, 0x65, 0x12, 0x1b, 0x0a, 0x09, 0x6c, 0x61, 0x73, 0x74, 0x5f, 0x6e, 0x61, 0x6d,
	0x65, 0x18, 0x06, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x6c, 0x61, 0x73, 0x74, 0x4e, 0x61, 0x6d,
	0x65, 0x12, 0x23, 0x0a, 0x0d, 0x63, 0x6f, 0x6e, 0x6e, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x5f,
	0x69, 0x64, 0x18, 0x07, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x63, 0x6f, 0x6e, 0x6e, 0x65, 0x63,
	0x74, 0x69, 0x6f, 0x6e, 0x49, 0x64, 0x22, 0x30, 0x0a, 0x17, 0x41, 0x75, 0x74, 0x68, 0x30, 0x43,
	0x72, 0x65, 0x61, 0x74, 0x65, 0x55, 0x73, 0x65, 0x72, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x12, 0x15, 0x0a, 0x06, 0x6f, 0x72, 0x67, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x05, 0x6f, 0x72, 0x67, 0x49, 0x64, 0x32, 0xc7, 0x01, 0x0a, 0x0c, 0x41, 0x75, 0x74,
	0x68, 0x30, 0x53, 0x75, 0x70, 0x70, 0x6f, 0x72, 0x74, 0x12, 0x59, 0x0a, 0x0a, 0x43, 0x72, 0x65,
	0x61, 0x74, 0x65, 0x55, 0x73, 0x65, 0x72, 0x12, 0x24, 0x2e, 0x61, 0x75, 0x74, 0x68, 0x30, 0x73,
	0x75, 0x70, 0x70, 0x6f, 0x72, 0x74, 0x2e, 0x41, 0x75, 0x74, 0x68, 0x30, 0x43, 0x72, 0x65, 0x61,
	0x74, 0x65, 0x55, 0x73, 0x65, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x25, 0x2e,
	0x61, 0x75, 0x74, 0x68, 0x30, 0x73, 0x75, 0x70, 0x70, 0x6f, 0x72, 0x74, 0x2e, 0x41, 0x75, 0x74,
	0x68, 0x30, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x55, 0x73, 0x65, 0x72, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x12, 0x5c, 0x0a, 0x10, 0x4b, 0x65, 0x65, 0x70, 0x61, 0x6c, 0x69, 0x76,
	0x65, 0x53, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x73, 0x12, 0x25, 0x2e, 0x61, 0x63, 0x63, 0x6f, 0x75,
	0x6e, 0x74, 0x2e, 0x41, 0x64, 0x6d, 0x69, 0x6e, 0x4b, 0x65, 0x65, 0x70, 0x61, 0x6c, 0x69, 0x76,
	0x65, 0x53, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a,
	0x21, 0x2e, 0x61, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x2e, 0x4b, 0x65, 0x65, 0x70, 0x61, 0x6c,
	0x69, 0x76, 0x65, 0x53, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x42, 0x2e, 0x5a, 0x2c, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d,
	0x2f, 0x6f, 0x76, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x64, 0x74, 0x65, 0x63, 0x68, 0x2f, 0x77, 0x6f,
	0x72, 0x6b, 0x73, 0x70, 0x61, 0x63, 0x65, 0x2f, 0x73, 0x64, 0x70, 0x2d, 0x67, 0x6f, 0x3b, 0x73,
	0x64, 0x70, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
})

var (
	file_auth0support_proto_rawDescOnce sync.Once
	file_auth0support_proto_rawDescData []byte
)

func file_auth0support_proto_rawDescGZIP() []byte {
	file_auth0support_proto_rawDescOnce.Do(func() {
		file_auth0support_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_auth0support_proto_rawDesc), len(file_auth0support_proto_rawDesc)))
	})
	return file_auth0support_proto_rawDescData
}

var file_auth0support_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_auth0support_proto_goTypes = []any{
	(*Auth0CreateUserRequest)(nil),       // 0: auth0support.Auth0CreateUserRequest
	(*Auth0CreateUserResponse)(nil),      // 1: auth0support.Auth0CreateUserResponse
	(*AdminKeepaliveSourcesRequest)(nil), // 2: account.AdminKeepaliveSourcesRequest
	(*KeepaliveSourcesResponse)(nil),     // 3: account.KeepaliveSourcesResponse
}
var file_auth0support_proto_depIdxs = []int32{
	0, // 0: auth0support.Auth0Support.CreateUser:input_type -> auth0support.Auth0CreateUserRequest
	2, // 1: auth0support.Auth0Support.KeepaliveSources:input_type -> account.AdminKeepaliveSourcesRequest
	1, // 2: auth0support.Auth0Support.CreateUser:output_type -> auth0support.Auth0CreateUserResponse
	3, // 3: auth0support.Auth0Support.KeepaliveSources:output_type -> account.KeepaliveSourcesResponse
	2, // [2:4] is the sub-list for method output_type
	0, // [0:2] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_auth0support_proto_init() }
func file_auth0support_proto_init() {
	if File_auth0support_proto != nil {
		return
	}
	file_account_proto_init()
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_auth0support_proto_rawDesc), len(file_auth0support_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_auth0support_proto_goTypes,
		DependencyIndexes: file_auth0support_proto_depIdxs,
		MessageInfos:      file_auth0support_proto_msgTypes,
	}.Build()
	File_auth0support_proto = out.File
	file_auth0support_proto_goTypes = nil
	file_auth0support_proto_depIdxs = nil
}
