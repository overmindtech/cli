// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.4
// 	protoc        (unknown)
// source: bookmarks.proto

package sdp

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
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

// a complete Bookmark with user-supplied and machine-supplied values
type Bookmark struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Metadata      *BookmarkMetadata      `protobuf:"bytes,1,opt,name=metadata,proto3" json:"metadata,omitempty"`
	Properties    *BookmarkProperties    `protobuf:"bytes,2,opt,name=properties,proto3" json:"properties,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Bookmark) Reset() {
	*x = Bookmark{}
	mi := &file_bookmarks_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Bookmark) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Bookmark) ProtoMessage() {}

func (x *Bookmark) ProtoReflect() protoreflect.Message {
	mi := &file_bookmarks_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Bookmark.ProtoReflect.Descriptor instead.
func (*Bookmark) Descriptor() ([]byte, []int) {
	return file_bookmarks_proto_rawDescGZIP(), []int{0}
}

func (x *Bookmark) GetMetadata() *BookmarkMetadata {
	if x != nil {
		return x.Metadata
	}
	return nil
}

func (x *Bookmark) GetProperties() *BookmarkProperties {
	if x != nil {
		return x.Properties
	}
	return nil
}

// The user-editable parts of a Bookmark
type BookmarkProperties struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// user supplied name of this bookmark
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// user supplied description of this bookmark
	Description string `protobuf:"bytes,2,opt,name=description,proto3" json:"description,omitempty"`
	// queries that make up the bookmark
	Queries []*Query `protobuf:"bytes,3,rep,name=queries,proto3" json:"queries,omitempty"`
	// Items that should be excluded from the bookmark's results
	ExcludedItems []*Reference `protobuf:"bytes,4,rep,name=excludedItems,proto3" json:"excludedItems,omitempty"`
	// Whether this bookmark is a system bookmark. System bookmarks are hidden
	// from list results and can therefore only be accessed by their UUID.
	// Bookmarks created by users are not system bookmarks.
	IsSystem      bool `protobuf:"varint,5,opt,name=isSystem,proto3" json:"isSystem,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *BookmarkProperties) Reset() {
	*x = BookmarkProperties{}
	mi := &file_bookmarks_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *BookmarkProperties) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BookmarkProperties) ProtoMessage() {}

func (x *BookmarkProperties) ProtoReflect() protoreflect.Message {
	mi := &file_bookmarks_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BookmarkProperties.ProtoReflect.Descriptor instead.
func (*BookmarkProperties) Descriptor() ([]byte, []int) {
	return file_bookmarks_proto_rawDescGZIP(), []int{1}
}

func (x *BookmarkProperties) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *BookmarkProperties) GetDescription() string {
	if x != nil {
		return x.Description
	}
	return ""
}

func (x *BookmarkProperties) GetQueries() []*Query {
	if x != nil {
		return x.Queries
	}
	return nil
}

func (x *BookmarkProperties) GetExcludedItems() []*Reference {
	if x != nil {
		return x.ExcludedItems
	}
	return nil
}

func (x *BookmarkProperties) GetIsSystem() bool {
	if x != nil {
		return x.IsSystem
	}
	return false
}

// Descriptor for a bookmark
type BookmarkMetadata struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// unique id to identify this bookmark
	UUID []byte `protobuf:"bytes,1,opt,name=UUID,proto3" json:"UUID,omitempty"`
	// timestamp when this bookmark was created
	Created       *timestamppb.Timestamp `protobuf:"bytes,2,opt,name=created,proto3" json:"created,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *BookmarkMetadata) Reset() {
	*x = BookmarkMetadata{}
	mi := &file_bookmarks_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *BookmarkMetadata) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BookmarkMetadata) ProtoMessage() {}

func (x *BookmarkMetadata) ProtoReflect() protoreflect.Message {
	mi := &file_bookmarks_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BookmarkMetadata.ProtoReflect.Descriptor instead.
func (*BookmarkMetadata) Descriptor() ([]byte, []int) {
	return file_bookmarks_proto_rawDescGZIP(), []int{2}
}

func (x *BookmarkMetadata) GetUUID() []byte {
	if x != nil {
		return x.UUID
	}
	return nil
}

func (x *BookmarkMetadata) GetCreated() *timestamppb.Timestamp {
	if x != nil {
		return x.Created
	}
	return nil
}

// list all bookmarks
type ListBookmarksRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ListBookmarksRequest) Reset() {
	*x = ListBookmarksRequest{}
	mi := &file_bookmarks_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ListBookmarksRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListBookmarksRequest) ProtoMessage() {}

func (x *ListBookmarksRequest) ProtoReflect() protoreflect.Message {
	mi := &file_bookmarks_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListBookmarksRequest.ProtoReflect.Descriptor instead.
func (*ListBookmarksRequest) Descriptor() ([]byte, []int) {
	return file_bookmarks_proto_rawDescGZIP(), []int{3}
}

type ListBookmarkResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Bookmarks     []*Bookmark            `protobuf:"bytes,3,rep,name=bookmarks,proto3" json:"bookmarks,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ListBookmarkResponse) Reset() {
	*x = ListBookmarkResponse{}
	mi := &file_bookmarks_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ListBookmarkResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListBookmarkResponse) ProtoMessage() {}

func (x *ListBookmarkResponse) ProtoReflect() protoreflect.Message {
	mi := &file_bookmarks_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListBookmarkResponse.ProtoReflect.Descriptor instead.
func (*ListBookmarkResponse) Descriptor() ([]byte, []int) {
	return file_bookmarks_proto_rawDescGZIP(), []int{4}
}

func (x *ListBookmarkResponse) GetBookmarks() []*Bookmark {
	if x != nil {
		return x.Bookmarks
	}
	return nil
}

// creates a new bookmark
type CreateBookmarkRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Properties    *BookmarkProperties    `protobuf:"bytes,1,opt,name=properties,proto3" json:"properties,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *CreateBookmarkRequest) Reset() {
	*x = CreateBookmarkRequest{}
	mi := &file_bookmarks_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CreateBookmarkRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateBookmarkRequest) ProtoMessage() {}

func (x *CreateBookmarkRequest) ProtoReflect() protoreflect.Message {
	mi := &file_bookmarks_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateBookmarkRequest.ProtoReflect.Descriptor instead.
func (*CreateBookmarkRequest) Descriptor() ([]byte, []int) {
	return file_bookmarks_proto_rawDescGZIP(), []int{5}
}

func (x *CreateBookmarkRequest) GetProperties() *BookmarkProperties {
	if x != nil {
		return x.Properties
	}
	return nil
}

type CreateBookmarkResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Bookmark      *Bookmark              `protobuf:"bytes,1,opt,name=bookmark,proto3" json:"bookmark,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *CreateBookmarkResponse) Reset() {
	*x = CreateBookmarkResponse{}
	mi := &file_bookmarks_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CreateBookmarkResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateBookmarkResponse) ProtoMessage() {}

func (x *CreateBookmarkResponse) ProtoReflect() protoreflect.Message {
	mi := &file_bookmarks_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateBookmarkResponse.ProtoReflect.Descriptor instead.
func (*CreateBookmarkResponse) Descriptor() ([]byte, []int) {
	return file_bookmarks_proto_rawDescGZIP(), []int{6}
}

func (x *CreateBookmarkResponse) GetBookmark() *Bookmark {
	if x != nil {
		return x.Bookmark
	}
	return nil
}

// gets a specific bookmark
type GetBookmarkRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	UUID          []byte                 `protobuf:"bytes,1,opt,name=UUID,proto3" json:"UUID,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetBookmarkRequest) Reset() {
	*x = GetBookmarkRequest{}
	mi := &file_bookmarks_proto_msgTypes[7]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetBookmarkRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetBookmarkRequest) ProtoMessage() {}

func (x *GetBookmarkRequest) ProtoReflect() protoreflect.Message {
	mi := &file_bookmarks_proto_msgTypes[7]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetBookmarkRequest.ProtoReflect.Descriptor instead.
func (*GetBookmarkRequest) Descriptor() ([]byte, []int) {
	return file_bookmarks_proto_rawDescGZIP(), []int{7}
}

func (x *GetBookmarkRequest) GetUUID() []byte {
	if x != nil {
		return x.UUID
	}
	return nil
}

type GetBookmarkResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Bookmark      *Bookmark              `protobuf:"bytes,1,opt,name=bookmark,proto3" json:"bookmark,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetBookmarkResponse) Reset() {
	*x = GetBookmarkResponse{}
	mi := &file_bookmarks_proto_msgTypes[8]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetBookmarkResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetBookmarkResponse) ProtoMessage() {}

func (x *GetBookmarkResponse) ProtoReflect() protoreflect.Message {
	mi := &file_bookmarks_proto_msgTypes[8]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetBookmarkResponse.ProtoReflect.Descriptor instead.
func (*GetBookmarkResponse) Descriptor() ([]byte, []int) {
	return file_bookmarks_proto_rawDescGZIP(), []int{8}
}

func (x *GetBookmarkResponse) GetBookmark() *Bookmark {
	if x != nil {
		return x.Bookmark
	}
	return nil
}

// updates an existing bookmark
type UpdateBookmarkRequest struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// unique id to identify this bookmark
	UUID []byte `protobuf:"bytes,1,opt,name=UUID,proto3" json:"UUID,omitempty"`
	// new attributes for this bookmark
	Properties    *BookmarkProperties `protobuf:"bytes,2,opt,name=properties,proto3" json:"properties,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UpdateBookmarkRequest) Reset() {
	*x = UpdateBookmarkRequest{}
	mi := &file_bookmarks_proto_msgTypes[9]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UpdateBookmarkRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UpdateBookmarkRequest) ProtoMessage() {}

func (x *UpdateBookmarkRequest) ProtoReflect() protoreflect.Message {
	mi := &file_bookmarks_proto_msgTypes[9]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UpdateBookmarkRequest.ProtoReflect.Descriptor instead.
func (*UpdateBookmarkRequest) Descriptor() ([]byte, []int) {
	return file_bookmarks_proto_rawDescGZIP(), []int{9}
}

func (x *UpdateBookmarkRequest) GetUUID() []byte {
	if x != nil {
		return x.UUID
	}
	return nil
}

func (x *UpdateBookmarkRequest) GetProperties() *BookmarkProperties {
	if x != nil {
		return x.Properties
	}
	return nil
}

type UpdateBookmarkResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Bookmark      *Bookmark              `protobuf:"bytes,3,opt,name=bookmark,proto3" json:"bookmark,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UpdateBookmarkResponse) Reset() {
	*x = UpdateBookmarkResponse{}
	mi := &file_bookmarks_proto_msgTypes[10]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UpdateBookmarkResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UpdateBookmarkResponse) ProtoMessage() {}

func (x *UpdateBookmarkResponse) ProtoReflect() protoreflect.Message {
	mi := &file_bookmarks_proto_msgTypes[10]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UpdateBookmarkResponse.ProtoReflect.Descriptor instead.
func (*UpdateBookmarkResponse) Descriptor() ([]byte, []int) {
	return file_bookmarks_proto_rawDescGZIP(), []int{10}
}

func (x *UpdateBookmarkResponse) GetBookmark() *Bookmark {
	if x != nil {
		return x.Bookmark
	}
	return nil
}

// Delete the bookmark with the specified ID.
type DeleteBookmarkRequest struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// unique id of the bookmark to delete
	UUID          []byte `protobuf:"bytes,1,opt,name=UUID,proto3" json:"UUID,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DeleteBookmarkRequest) Reset() {
	*x = DeleteBookmarkRequest{}
	mi := &file_bookmarks_proto_msgTypes[11]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DeleteBookmarkRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeleteBookmarkRequest) ProtoMessage() {}

func (x *DeleteBookmarkRequest) ProtoReflect() protoreflect.Message {
	mi := &file_bookmarks_proto_msgTypes[11]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeleteBookmarkRequest.ProtoReflect.Descriptor instead.
func (*DeleteBookmarkRequest) Descriptor() ([]byte, []int) {
	return file_bookmarks_proto_rawDescGZIP(), []int{11}
}

func (x *DeleteBookmarkRequest) GetUUID() []byte {
	if x != nil {
		return x.UUID
	}
	return nil
}

type DeleteBookmarkResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DeleteBookmarkResponse) Reset() {
	*x = DeleteBookmarkResponse{}
	mi := &file_bookmarks_proto_msgTypes[12]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DeleteBookmarkResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeleteBookmarkResponse) ProtoMessage() {}

func (x *DeleteBookmarkResponse) ProtoReflect() protoreflect.Message {
	mi := &file_bookmarks_proto_msgTypes[12]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeleteBookmarkResponse.ProtoReflect.Descriptor instead.
func (*DeleteBookmarkResponse) Descriptor() ([]byte, []int) {
	return file_bookmarks_proto_rawDescGZIP(), []int{12}
}

type GetAffectedBookmarksRequest struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// the snapshot to consider
	SnapshotUUID []byte `protobuf:"bytes,1,opt,name=snapshotUUID,proto3" json:"snapshotUUID,omitempty"`
	// the bookmarks to filter
	BookmarkUUIDs [][]byte `protobuf:"bytes,2,rep,name=bookmarkUUIDs,proto3" json:"bookmarkUUIDs,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetAffectedBookmarksRequest) Reset() {
	*x = GetAffectedBookmarksRequest{}
	mi := &file_bookmarks_proto_msgTypes[13]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetAffectedBookmarksRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetAffectedBookmarksRequest) ProtoMessage() {}

func (x *GetAffectedBookmarksRequest) ProtoReflect() protoreflect.Message {
	mi := &file_bookmarks_proto_msgTypes[13]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetAffectedBookmarksRequest.ProtoReflect.Descriptor instead.
func (*GetAffectedBookmarksRequest) Descriptor() ([]byte, []int) {
	return file_bookmarks_proto_rawDescGZIP(), []int{13}
}

func (x *GetAffectedBookmarksRequest) GetSnapshotUUID() []byte {
	if x != nil {
		return x.SnapshotUUID
	}
	return nil
}

func (x *GetAffectedBookmarksRequest) GetBookmarkUUIDs() [][]byte {
	if x != nil {
		return x.BookmarkUUIDs
	}
	return nil
}

type GetAffectedBookmarksResponse struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// the bookmarks that intersected with the snapshot
	BookmarkUUIDs [][]byte `protobuf:"bytes,1,rep,name=bookmarkUUIDs,proto3" json:"bookmarkUUIDs,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetAffectedBookmarksResponse) Reset() {
	*x = GetAffectedBookmarksResponse{}
	mi := &file_bookmarks_proto_msgTypes[14]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetAffectedBookmarksResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetAffectedBookmarksResponse) ProtoMessage() {}

func (x *GetAffectedBookmarksResponse) ProtoReflect() protoreflect.Message {
	mi := &file_bookmarks_proto_msgTypes[14]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetAffectedBookmarksResponse.ProtoReflect.Descriptor instead.
func (*GetAffectedBookmarksResponse) Descriptor() ([]byte, []int) {
	return file_bookmarks_proto_rawDescGZIP(), []int{14}
}

func (x *GetAffectedBookmarksResponse) GetBookmarkUUIDs() [][]byte {
	if x != nil {
		return x.BookmarkUUIDs
	}
	return nil
}

var File_bookmarks_proto protoreflect.FileDescriptor

var file_bookmarks_proto_rawDesc = string([]byte{
	0x0a, 0x0f, 0x62, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x12, 0x09, 0x62, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x73, 0x1a, 0x1f, 0x67, 0x6f,
	0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69,
	0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x0b, 0x69,
	0x74, 0x65, 0x6d, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x82, 0x01, 0x0a, 0x08, 0x42,
	0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x12, 0x37, 0x0a, 0x08, 0x6d, 0x65, 0x74, 0x61, 0x64,
	0x61, 0x74, 0x61, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1b, 0x2e, 0x62, 0x6f, 0x6f, 0x6b,
	0x6d, 0x61, 0x72, 0x6b, 0x73, 0x2e, 0x42, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x4d, 0x65,
	0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x52, 0x08, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61,
	0x12, 0x3d, 0x0a, 0x0a, 0x70, 0x72, 0x6f, 0x70, 0x65, 0x72, 0x74, 0x69, 0x65, 0x73, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x1d, 0x2e, 0x62, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x73,
	0x2e, 0x42, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x50, 0x72, 0x6f, 0x70, 0x65, 0x72, 0x74,
	0x69, 0x65, 0x73, 0x52, 0x0a, 0x70, 0x72, 0x6f, 0x70, 0x65, 0x72, 0x74, 0x69, 0x65, 0x73, 0x22,
	0xba, 0x01, 0x0a, 0x12, 0x42, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x50, 0x72, 0x6f, 0x70,
	0x65, 0x72, 0x74, 0x69, 0x65, 0x73, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x20, 0x0a, 0x0b, 0x64, 0x65,
	0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x20, 0x0a, 0x07,
	0x71, 0x75, 0x65, 0x72, 0x69, 0x65, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x06, 0x2e,
	0x51, 0x75, 0x65, 0x72, 0x79, 0x52, 0x07, 0x71, 0x75, 0x65, 0x72, 0x69, 0x65, 0x73, 0x12, 0x30,
	0x0a, 0x0d, 0x65, 0x78, 0x63, 0x6c, 0x75, 0x64, 0x65, 0x64, 0x49, 0x74, 0x65, 0x6d, 0x73, 0x18,
	0x04, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x0a, 0x2e, 0x52, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63,
	0x65, 0x52, 0x0d, 0x65, 0x78, 0x63, 0x6c, 0x75, 0x64, 0x65, 0x64, 0x49, 0x74, 0x65, 0x6d, 0x73,
	0x12, 0x1a, 0x0a, 0x08, 0x69, 0x73, 0x53, 0x79, 0x73, 0x74, 0x65, 0x6d, 0x18, 0x05, 0x20, 0x01,
	0x28, 0x08, 0x52, 0x08, 0x69, 0x73, 0x53, 0x79, 0x73, 0x74, 0x65, 0x6d, 0x22, 0x5c, 0x0a, 0x10,
	0x42, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61,
	0x12, 0x12, 0x0a, 0x04, 0x55, 0x55, 0x49, 0x44, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x04,
	0x55, 0x55, 0x49, 0x44, 0x12, 0x34, 0x0a, 0x07, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x64, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d,
	0x70, 0x52, 0x07, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x64, 0x22, 0x16, 0x0a, 0x14, 0x4c, 0x69,
	0x73, 0x74, 0x42, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x22, 0x49, 0x0a, 0x14, 0x4c, 0x69, 0x73, 0x74, 0x42, 0x6f, 0x6f, 0x6b, 0x6d, 0x61,
	0x72, 0x6b, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x31, 0x0a, 0x09, 0x62, 0x6f,
	0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x13, 0x2e,
	0x62, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x73, 0x2e, 0x42, 0x6f, 0x6f, 0x6b, 0x6d, 0x61,
	0x72, 0x6b, 0x52, 0x09, 0x62, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x73, 0x22, 0x56, 0x0a,
	0x15, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x42, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x3d, 0x0a, 0x0a, 0x70, 0x72, 0x6f, 0x70, 0x65, 0x72,
	0x74, 0x69, 0x65, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1d, 0x2e, 0x62, 0x6f, 0x6f,
	0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x73, 0x2e, 0x42, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x50,
	0x72, 0x6f, 0x70, 0x65, 0x72, 0x74, 0x69, 0x65, 0x73, 0x52, 0x0a, 0x70, 0x72, 0x6f, 0x70, 0x65,
	0x72, 0x74, 0x69, 0x65, 0x73, 0x22, 0x49, 0x0a, 0x16, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x42,
	0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12,
	0x2f, 0x0a, 0x08, 0x62, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x13, 0x2e, 0x62, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x73, 0x2e, 0x42, 0x6f,
	0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x52, 0x08, 0x62, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b,
	0x22, 0x28, 0x0a, 0x12, 0x47, 0x65, 0x74, 0x42, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x55, 0x55, 0x49, 0x44, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0c, 0x52, 0x04, 0x55, 0x55, 0x49, 0x44, 0x22, 0x46, 0x0a, 0x13, 0x47, 0x65,
	0x74, 0x42, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x12, 0x2f, 0x0a, 0x08, 0x62, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x13, 0x2e, 0x62, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x73, 0x2e,
	0x42, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x52, 0x08, 0x62, 0x6f, 0x6f, 0x6b, 0x6d, 0x61,
	0x72, 0x6b, 0x22, 0x6a, 0x0a, 0x15, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x42, 0x6f, 0x6f, 0x6b,
	0x6d, 0x61, 0x72, 0x6b, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x55,
	0x55, 0x49, 0x44, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x04, 0x55, 0x55, 0x49, 0x44, 0x12,
	0x3d, 0x0a, 0x0a, 0x70, 0x72, 0x6f, 0x70, 0x65, 0x72, 0x74, 0x69, 0x65, 0x73, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x1d, 0x2e, 0x62, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x73, 0x2e,
	0x42, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x50, 0x72, 0x6f, 0x70, 0x65, 0x72, 0x74, 0x69,
	0x65, 0x73, 0x52, 0x0a, 0x70, 0x72, 0x6f, 0x70, 0x65, 0x72, 0x74, 0x69, 0x65, 0x73, 0x22, 0x49,
	0x0a, 0x16, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x42, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b,
	0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x2f, 0x0a, 0x08, 0x62, 0x6f, 0x6f, 0x6b,
	0x6d, 0x61, 0x72, 0x6b, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x13, 0x2e, 0x62, 0x6f, 0x6f,
	0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x73, 0x2e, 0x42, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x52,
	0x08, 0x62, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x22, 0x2b, 0x0a, 0x15, 0x44, 0x65, 0x6c,
	0x65, 0x74, 0x65, 0x42, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x55, 0x55, 0x49, 0x44, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c,
	0x52, 0x04, 0x55, 0x55, 0x49, 0x44, 0x22, 0x18, 0x0a, 0x16, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65,
	0x42, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x22, 0x67, 0x0a, 0x1b, 0x47, 0x65, 0x74, 0x41, 0x66, 0x66, 0x65, 0x63, 0x74, 0x65, 0x64, 0x42,
	0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12,
	0x22, 0x0a, 0x0c, 0x73, 0x6e, 0x61, 0x70, 0x73, 0x68, 0x6f, 0x74, 0x55, 0x55, 0x49, 0x44, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x0c, 0x73, 0x6e, 0x61, 0x70, 0x73, 0x68, 0x6f, 0x74, 0x55,
	0x55, 0x49, 0x44, 0x12, 0x24, 0x0a, 0x0d, 0x62, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x55,
	0x55, 0x49, 0x44, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0c, 0x52, 0x0d, 0x62, 0x6f, 0x6f, 0x6b,
	0x6d, 0x61, 0x72, 0x6b, 0x55, 0x55, 0x49, 0x44, 0x73, 0x22, 0x44, 0x0a, 0x1c, 0x47, 0x65, 0x74,
	0x41, 0x66, 0x66, 0x65, 0x63, 0x74, 0x65, 0x64, 0x42, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b,
	0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x24, 0x0a, 0x0d, 0x62, 0x6f, 0x6f,
	0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x55, 0x55, 0x49, 0x44, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0c,
	0x52, 0x0d, 0x62, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x55, 0x55, 0x49, 0x44, 0x73, 0x32,
	0xa1, 0x04, 0x0a, 0x10, 0x42, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x73, 0x53, 0x65, 0x72,
	0x76, 0x69, 0x63, 0x65, 0x12, 0x51, 0x0a, 0x0d, 0x4c, 0x69, 0x73, 0x74, 0x42, 0x6f, 0x6f, 0x6b,
	0x6d, 0x61, 0x72, 0x6b, 0x73, 0x12, 0x1f, 0x2e, 0x62, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b,
	0x73, 0x2e, 0x4c, 0x69, 0x73, 0x74, 0x42, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x73, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x1f, 0x2e, 0x62, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72,
	0x6b, 0x73, 0x2e, 0x4c, 0x69, 0x73, 0x74, 0x42, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x52,
	0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x55, 0x0a, 0x0e, 0x43, 0x72, 0x65, 0x61, 0x74,
	0x65, 0x42, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x12, 0x20, 0x2e, 0x62, 0x6f, 0x6f, 0x6b,
	0x6d, 0x61, 0x72, 0x6b, 0x73, 0x2e, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x42, 0x6f, 0x6f, 0x6b,
	0x6d, 0x61, 0x72, 0x6b, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x21, 0x2e, 0x62, 0x6f,
	0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x73, 0x2e, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x42, 0x6f,
	0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x4c,
	0x0a, 0x0b, 0x47, 0x65, 0x74, 0x42, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x12, 0x1d, 0x2e,
	0x62, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x73, 0x2e, 0x47, 0x65, 0x74, 0x42, 0x6f, 0x6f,
	0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x1e, 0x2e, 0x62,
	0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x73, 0x2e, 0x47, 0x65, 0x74, 0x42, 0x6f, 0x6f, 0x6b,
	0x6d, 0x61, 0x72, 0x6b, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x55, 0x0a, 0x0e,
	0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x42, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x12, 0x20,
	0x2e, 0x62, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x73, 0x2e, 0x55, 0x70, 0x64, 0x61, 0x74,
	0x65, 0x42, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x1a, 0x21, 0x2e, 0x62, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x73, 0x2e, 0x55, 0x70, 0x64,
	0x61, 0x74, 0x65, 0x42, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x12, 0x55, 0x0a, 0x0e, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x42, 0x6f, 0x6f,
	0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x12, 0x20, 0x2e, 0x62, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b,
	0x73, 0x2e, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x42, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x21, 0x2e, 0x62, 0x6f, 0x6f, 0x6b, 0x6d, 0x61,
	0x72, 0x6b, 0x73, 0x2e, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x42, 0x6f, 0x6f, 0x6b, 0x6d, 0x61,
	0x72, 0x6b, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x67, 0x0a, 0x14, 0x47, 0x65,
	0x74, 0x41, 0x66, 0x66, 0x65, 0x63, 0x74, 0x65, 0x64, 0x42, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72,
	0x6b, 0x73, 0x12, 0x26, 0x2e, 0x62, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x73, 0x2e, 0x47,
	0x65, 0x74, 0x41, 0x66, 0x66, 0x65, 0x63, 0x74, 0x65, 0x64, 0x42, 0x6f, 0x6f, 0x6b, 0x6d, 0x61,
	0x72, 0x6b, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x27, 0x2e, 0x62, 0x6f, 0x6f,
	0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x73, 0x2e, 0x47, 0x65, 0x74, 0x41, 0x66, 0x66, 0x65, 0x63, 0x74,
	0x65, 0x64, 0x42, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x42, 0x2e, 0x5a, 0x2c, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f,
	0x6d, 0x2f, 0x6f, 0x76, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x64, 0x74, 0x65, 0x63, 0x68, 0x2f, 0x77,
	0x6f, 0x72, 0x6b, 0x73, 0x70, 0x61, 0x63, 0x65, 0x2f, 0x73, 0x64, 0x70, 0x2d, 0x67, 0x6f, 0x3b,
	0x73, 0x64, 0x70, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
})

var (
	file_bookmarks_proto_rawDescOnce sync.Once
	file_bookmarks_proto_rawDescData []byte
)

func file_bookmarks_proto_rawDescGZIP() []byte {
	file_bookmarks_proto_rawDescOnce.Do(func() {
		file_bookmarks_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_bookmarks_proto_rawDesc), len(file_bookmarks_proto_rawDesc)))
	})
	return file_bookmarks_proto_rawDescData
}

var file_bookmarks_proto_msgTypes = make([]protoimpl.MessageInfo, 15)
var file_bookmarks_proto_goTypes = []any{
	(*Bookmark)(nil),                     // 0: bookmarks.Bookmark
	(*BookmarkProperties)(nil),           // 1: bookmarks.BookmarkProperties
	(*BookmarkMetadata)(nil),             // 2: bookmarks.BookmarkMetadata
	(*ListBookmarksRequest)(nil),         // 3: bookmarks.ListBookmarksRequest
	(*ListBookmarkResponse)(nil),         // 4: bookmarks.ListBookmarkResponse
	(*CreateBookmarkRequest)(nil),        // 5: bookmarks.CreateBookmarkRequest
	(*CreateBookmarkResponse)(nil),       // 6: bookmarks.CreateBookmarkResponse
	(*GetBookmarkRequest)(nil),           // 7: bookmarks.GetBookmarkRequest
	(*GetBookmarkResponse)(nil),          // 8: bookmarks.GetBookmarkResponse
	(*UpdateBookmarkRequest)(nil),        // 9: bookmarks.UpdateBookmarkRequest
	(*UpdateBookmarkResponse)(nil),       // 10: bookmarks.UpdateBookmarkResponse
	(*DeleteBookmarkRequest)(nil),        // 11: bookmarks.DeleteBookmarkRequest
	(*DeleteBookmarkResponse)(nil),       // 12: bookmarks.DeleteBookmarkResponse
	(*GetAffectedBookmarksRequest)(nil),  // 13: bookmarks.GetAffectedBookmarksRequest
	(*GetAffectedBookmarksResponse)(nil), // 14: bookmarks.GetAffectedBookmarksResponse
	(*Query)(nil),                        // 15: Query
	(*Reference)(nil),                    // 16: Reference
	(*timestamppb.Timestamp)(nil),        // 17: google.protobuf.Timestamp
}
var file_bookmarks_proto_depIdxs = []int32{
	2,  // 0: bookmarks.Bookmark.metadata:type_name -> bookmarks.BookmarkMetadata
	1,  // 1: bookmarks.Bookmark.properties:type_name -> bookmarks.BookmarkProperties
	15, // 2: bookmarks.BookmarkProperties.queries:type_name -> Query
	16, // 3: bookmarks.BookmarkProperties.excludedItems:type_name -> Reference
	17, // 4: bookmarks.BookmarkMetadata.created:type_name -> google.protobuf.Timestamp
	0,  // 5: bookmarks.ListBookmarkResponse.bookmarks:type_name -> bookmarks.Bookmark
	1,  // 6: bookmarks.CreateBookmarkRequest.properties:type_name -> bookmarks.BookmarkProperties
	0,  // 7: bookmarks.CreateBookmarkResponse.bookmark:type_name -> bookmarks.Bookmark
	0,  // 8: bookmarks.GetBookmarkResponse.bookmark:type_name -> bookmarks.Bookmark
	1,  // 9: bookmarks.UpdateBookmarkRequest.properties:type_name -> bookmarks.BookmarkProperties
	0,  // 10: bookmarks.UpdateBookmarkResponse.bookmark:type_name -> bookmarks.Bookmark
	3,  // 11: bookmarks.BookmarksService.ListBookmarks:input_type -> bookmarks.ListBookmarksRequest
	5,  // 12: bookmarks.BookmarksService.CreateBookmark:input_type -> bookmarks.CreateBookmarkRequest
	7,  // 13: bookmarks.BookmarksService.GetBookmark:input_type -> bookmarks.GetBookmarkRequest
	9,  // 14: bookmarks.BookmarksService.UpdateBookmark:input_type -> bookmarks.UpdateBookmarkRequest
	11, // 15: bookmarks.BookmarksService.DeleteBookmark:input_type -> bookmarks.DeleteBookmarkRequest
	13, // 16: bookmarks.BookmarksService.GetAffectedBookmarks:input_type -> bookmarks.GetAffectedBookmarksRequest
	4,  // 17: bookmarks.BookmarksService.ListBookmarks:output_type -> bookmarks.ListBookmarkResponse
	6,  // 18: bookmarks.BookmarksService.CreateBookmark:output_type -> bookmarks.CreateBookmarkResponse
	8,  // 19: bookmarks.BookmarksService.GetBookmark:output_type -> bookmarks.GetBookmarkResponse
	10, // 20: bookmarks.BookmarksService.UpdateBookmark:output_type -> bookmarks.UpdateBookmarkResponse
	12, // 21: bookmarks.BookmarksService.DeleteBookmark:output_type -> bookmarks.DeleteBookmarkResponse
	14, // 22: bookmarks.BookmarksService.GetAffectedBookmarks:output_type -> bookmarks.GetAffectedBookmarksResponse
	17, // [17:23] is the sub-list for method output_type
	11, // [11:17] is the sub-list for method input_type
	11, // [11:11] is the sub-list for extension type_name
	11, // [11:11] is the sub-list for extension extendee
	0,  // [0:11] is the sub-list for field type_name
}

func init() { file_bookmarks_proto_init() }
func file_bookmarks_proto_init() {
	if File_bookmarks_proto != nil {
		return
	}
	file_items_proto_init()
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_bookmarks_proto_rawDesc), len(file_bookmarks_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   15,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_bookmarks_proto_goTypes,
		DependencyIndexes: file_bookmarks_proto_depIdxs,
		MessageInfos:      file_bookmarks_proto_msgTypes,
	}.Build()
	File_bookmarks_proto = out.File
	file_bookmarks_proto_goTypes = nil
	file_bookmarks_proto_depIdxs = nil
}
