// Code generated by protoc-gen-go. DO NOT EDIT.
// source: videoService.proto

package videoService

import (
	context "context"
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	empty "github.com/golang/protobuf/ptypes/empty"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type Rectangle struct {
	X                    int32    `protobuf:"varint,1,opt,name=X,proto3" json:"X,omitempty"`
	Y                    int32    `protobuf:"varint,2,opt,name=Y,proto3" json:"Y,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Rectangle) Reset()         { *m = Rectangle{} }
func (m *Rectangle) String() string { return proto.CompactTextString(m) }
func (*Rectangle) ProtoMessage()    {}
func (*Rectangle) Descriptor() ([]byte, []int) {
	return fileDescriptor_c4ee3c3e3f13865b, []int{0}
}

func (m *Rectangle) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Rectangle.Unmarshal(m, b)
}
func (m *Rectangle) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Rectangle.Marshal(b, m, deterministic)
}
func (m *Rectangle) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Rectangle.Merge(m, src)
}
func (m *Rectangle) XXX_Size() int {
	return xxx_messageInfo_Rectangle.Size(m)
}
func (m *Rectangle) XXX_DiscardUnknown() {
	xxx_messageInfo_Rectangle.DiscardUnknown(m)
}

var xxx_messageInfo_Rectangle proto.InternalMessageInfo

func (m *Rectangle) GetX() int32 {
	if m != nil {
		return m.X
	}
	return 0
}

func (m *Rectangle) GetY() int32 {
	if m != nil {
		return m.Y
	}
	return 0
}

type VideoMetaData struct {
	Size                 *Rectangle `protobuf:"bytes,1,opt,name=Size,proto3" json:"Size,omitempty"`
	MacroBlocks          *Rectangle `protobuf:"bytes,2,opt,name=MacroBlocks,proto3" json:"MacroBlocks,omitempty"`
	BitRate              int32      `protobuf:"varint,3,opt,name=BitRate,proto3" json:"BitRate,omitempty"`
	FrameRate            int32      `protobuf:"varint,4,opt,name=FrameRate,proto3" json:"FrameRate,omitempty"`
	KeyFrameRate         int32      `protobuf:"varint,5,opt,name=KeyFrameRate,proto3" json:"KeyFrameRate,omitempty"`
	AnalogGain           float32    `protobuf:"fixed32,6,opt,name=AnalogGain,proto3" json:"AnalogGain,omitempty"`
	DigitalGain          float32    `protobuf:"fixed32,7,opt,name=DigitalGain,proto3" json:"DigitalGain,omitempty"`
	RefreshType          string     `protobuf:"bytes,8,opt,name=RefreshType,proto3" json:"RefreshType,omitempty"`
	H264Level            string     `protobuf:"bytes,9,opt,name=H264Level,proto3" json:"H264Level,omitempty"`
	H264Profile          string     `protobuf:"bytes,10,opt,name=H264Profile,proto3" json:"H264Profile,omitempty"`
	XXX_NoUnkeyedLiteral struct{}   `json:"-"`
	XXX_unrecognized     []byte     `json:"-"`
	XXX_sizecache        int32      `json:"-"`
}

func (m *VideoMetaData) Reset()         { *m = VideoMetaData{} }
func (m *VideoMetaData) String() string { return proto.CompactTextString(m) }
func (*VideoMetaData) ProtoMessage()    {}
func (*VideoMetaData) Descriptor() ([]byte, []int) {
	return fileDescriptor_c4ee3c3e3f13865b, []int{1}
}

func (m *VideoMetaData) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_VideoMetaData.Unmarshal(m, b)
}
func (m *VideoMetaData) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_VideoMetaData.Marshal(b, m, deterministic)
}
func (m *VideoMetaData) XXX_Merge(src proto.Message) {
	xxx_messageInfo_VideoMetaData.Merge(m, src)
}
func (m *VideoMetaData) XXX_Size() int {
	return xxx_messageInfo_VideoMetaData.Size(m)
}
func (m *VideoMetaData) XXX_DiscardUnknown() {
	xxx_messageInfo_VideoMetaData.DiscardUnknown(m)
}

var xxx_messageInfo_VideoMetaData proto.InternalMessageInfo

func (m *VideoMetaData) GetSize() *Rectangle {
	if m != nil {
		return m.Size
	}
	return nil
}

func (m *VideoMetaData) GetMacroBlocks() *Rectangle {
	if m != nil {
		return m.MacroBlocks
	}
	return nil
}

func (m *VideoMetaData) GetBitRate() int32 {
	if m != nil {
		return m.BitRate
	}
	return 0
}

func (m *VideoMetaData) GetFrameRate() int32 {
	if m != nil {
		return m.FrameRate
	}
	return 0
}

func (m *VideoMetaData) GetKeyFrameRate() int32 {
	if m != nil {
		return m.KeyFrameRate
	}
	return 0
}

func (m *VideoMetaData) GetAnalogGain() float32 {
	if m != nil {
		return m.AnalogGain
	}
	return 0
}

func (m *VideoMetaData) GetDigitalGain() float32 {
	if m != nil {
		return m.DigitalGain
	}
	return 0
}

func (m *VideoMetaData) GetRefreshType() string {
	if m != nil {
		return m.RefreshType
	}
	return ""
}

func (m *VideoMetaData) GetH264Level() string {
	if m != nil {
		return m.H264Level
	}
	return ""
}

func (m *VideoMetaData) GetH264Profile() string {
	if m != nil {
		return m.H264Profile
	}
	return ""
}

type Frame struct {
	Data                 []byte   `protobuf:"bytes,1,opt,name=Data,proto3" json:"Data,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Frame) Reset()         { *m = Frame{} }
func (m *Frame) String() string { return proto.CompactTextString(m) }
func (*Frame) ProtoMessage()    {}
func (*Frame) Descriptor() ([]byte, []int) {
	return fileDescriptor_c4ee3c3e3f13865b, []int{2}
}

func (m *Frame) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Frame.Unmarshal(m, b)
}
func (m *Frame) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Frame.Marshal(b, m, deterministic)
}
func (m *Frame) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Frame.Merge(m, src)
}
func (m *Frame) XXX_Size() int {
	return xxx_messageInfo_Frame.Size(m)
}
func (m *Frame) XXX_DiscardUnknown() {
	xxx_messageInfo_Frame.DiscardUnknown(m)
}

var xxx_messageInfo_Frame proto.InternalMessageInfo

func (m *Frame) GetData() []byte {
	if m != nil {
		return m.Data
	}
	return nil
}

func init() {
	proto.RegisterType((*Rectangle)(nil), "Rectangle")
	proto.RegisterType((*VideoMetaData)(nil), "VideoMetaData")
	proto.RegisterType((*Frame)(nil), "Frame")
}

func init() { proto.RegisterFile("videoService.proto", fileDescriptor_c4ee3c3e3f13865b) }

var fileDescriptor_c4ee3c3e3f13865b = []byte{
	// 383 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x51, 0xcd, 0x8e, 0xd3, 0x30,
	0x10, 0x96, 0x43, 0xd3, 0x4d, 0xa6, 0x85, 0x83, 0x0f, 0xc8, 0xda, 0x45, 0xab, 0x28, 0x17, 0x2a,
	0x58, 0xb2, 0xab, 0x80, 0xb8, 0xb3, 0x6a, 0x29, 0x02, 0x2a, 0x55, 0x2e, 0x42, 0xed, 0xd1, 0x0d,
	0xd3, 0x60, 0x91, 0xc6, 0x55, 0x6a, 0x8a, 0xca, 0xd3, 0xf0, 0x62, 0xbc, 0x0b, 0xf2, 0xa4, 0x3f,
	0xe9, 0x01, 0x6d, 0x6f, 0x9e, 0xef, 0xcf, 0xa3, 0xf9, 0x80, 0x6f, 0xf4, 0x37, 0x34, 0x13, 0xac,
	0x36, 0x3a, 0xc3, 0x64, 0x55, 0x19, 0x6b, 0x2e, 0xaf, 0x72, 0x63, 0xf2, 0x02, 0x6f, 0x69, 0x9a,
	0xff, 0x5c, 0xdc, 0xe2, 0x72, 0x65, 0xb7, 0x35, 0x19, 0x3f, 0x87, 0x50, 0x62, 0x66, 0x55, 0x99,
	0x17, 0xc8, 0xbb, 0xc0, 0xa6, 0x82, 0x45, 0xac, 0xe7, 0x4b, 0x36, 0x75, 0xd3, 0x4c, 0x78, 0xf5,
	0x34, 0x8b, 0xff, 0x7a, 0xf0, 0xf8, 0xab, 0x0b, 0x1f, 0xa1, 0x55, 0x7d, 0x65, 0x15, 0xbf, 0x86,
	0xd6, 0x44, 0xff, 0x46, 0x32, 0x74, 0x52, 0x48, 0x0e, 0x39, 0x92, 0x70, 0x7e, 0x03, 0x9d, 0x91,
	0xca, 0x2a, 0x73, 0x5f, 0x98, 0xec, 0xc7, 0x9a, 0x92, 0x4e, 0x65, 0x4d, 0x9a, 0x0b, 0xb8, 0xb8,
	0xd7, 0x56, 0x2a, 0x8b, 0xe2, 0x11, 0xfd, 0xb9, 0x1f, 0xf9, 0x33, 0x08, 0xdf, 0x57, 0x6a, 0x89,
	0xc4, 0xb5, 0x88, 0x3b, 0x02, 0x3c, 0x86, 0xee, 0x27, 0xdc, 0x1e, 0x05, 0x3e, 0x09, 0x4e, 0x30,
	0x7e, 0x0d, 0xf0, 0xae, 0x54, 0x85, 0xc9, 0x87, 0x4a, 0x97, 0xa2, 0x1d, 0xb1, 0x9e, 0x27, 0x1b,
	0x08, 0x8f, 0xa0, 0xd3, 0xd7, 0xb9, 0xb6, 0xaa, 0x20, 0xc1, 0x05, 0x09, 0x9a, 0x90, 0x53, 0x48,
	0x5c, 0x54, 0xb8, 0xfe, 0xfe, 0x65, 0xbb, 0x42, 0x11, 0x44, 0xac, 0x17, 0xca, 0x26, 0xe4, 0xb6,
	0xfc, 0x90, 0xbe, 0x7d, 0xf3, 0x19, 0x37, 0x58, 0x88, 0x90, 0xf8, 0x23, 0xe0, 0xfc, 0x6e, 0x18,
	0x57, 0x66, 0xa1, 0x0b, 0x14, 0x50, 0xfb, 0x1b, 0x50, 0x7c, 0x05, 0x3e, 0x2d, 0xcc, 0x39, 0xb4,
	0xdc, 0x79, 0xe9, 0xac, 0x5d, 0x49, 0xef, 0xf4, 0x8f, 0x07, 0x3e, 0x1d, 0x9f, 0xa7, 0x10, 0x1c,
	0x0a, 0x78, 0x9a, 0xd4, 0xcd, 0x26, 0xfb, 0x66, 0x93, 0x81, 0x6b, 0xf6, 0xf2, 0x49, 0x72, 0x5a,
	0xd4, 0x0b, 0x08, 0x76, 0xb7, 0xf8, 0xf5, 0x5f, 0x4f, 0x3b, 0xa9, 0x7f, 0xbf, 0x81, 0x80, 0xcc,
	0x67, 0x68, 0xef, 0x18, 0x7f, 0xb9, 0xab, 0xe6, 0xe3, 0x78, 0x30, 0x7c, 0x30, 0xfa, 0x15, 0x84,
	0x14, 0x7d, 0x8e, 0xf8, 0x8e, 0x39, 0xf9, 0xc8, 0x58, 0x6d, 0xca, 0xb3, 0x56, 0x99, 0xb7, 0x89,
	0x79, 0xfd, 0x2f, 0x00, 0x00, 0xff, 0xff, 0xe6, 0x16, 0xc8, 0xd8, 0x02, 0x03, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// VideoClient is the client API for Video service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type VideoClient interface {
	MetaData(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (*VideoMetaData, error)
	FrameRaw(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (*Frame, error)
	VideoRaw(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (Video_VideoRawClient, error)
	FrameJPEG(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (*Frame, error)
	VideoJPEG(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (Video_VideoJPEGClient, error)
	MotionRaw(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (Video_MotionRawClient, error)
}

type videoClient struct {
	cc *grpc.ClientConn
}

func NewVideoClient(cc *grpc.ClientConn) VideoClient {
	return &videoClient{cc}
}

func (c *videoClient) MetaData(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (*VideoMetaData, error) {
	out := new(VideoMetaData)
	err := c.cc.Invoke(ctx, "/Video/MetaData", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *videoClient) FrameRaw(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (*Frame, error) {
	out := new(Frame)
	err := c.cc.Invoke(ctx, "/Video/FrameRaw", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *videoClient) VideoRaw(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (Video_VideoRawClient, error) {
	stream, err := c.cc.NewStream(ctx, &_Video_serviceDesc.Streams[0], "/Video/VideoRaw", opts...)
	if err != nil {
		return nil, err
	}
	x := &videoVideoRawClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Video_VideoRawClient interface {
	Recv() (*Frame, error)
	grpc.ClientStream
}

type videoVideoRawClient struct {
	grpc.ClientStream
}

func (x *videoVideoRawClient) Recv() (*Frame, error) {
	m := new(Frame)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *videoClient) FrameJPEG(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (*Frame, error) {
	out := new(Frame)
	err := c.cc.Invoke(ctx, "/Video/FrameJPEG", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *videoClient) VideoJPEG(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (Video_VideoJPEGClient, error) {
	stream, err := c.cc.NewStream(ctx, &_Video_serviceDesc.Streams[1], "/Video/VideoJPEG", opts...)
	if err != nil {
		return nil, err
	}
	x := &videoVideoJPEGClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Video_VideoJPEGClient interface {
	Recv() (*Frame, error)
	grpc.ClientStream
}

type videoVideoJPEGClient struct {
	grpc.ClientStream
}

func (x *videoVideoJPEGClient) Recv() (*Frame, error) {
	m := new(Frame)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *videoClient) MotionRaw(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (Video_MotionRawClient, error) {
	stream, err := c.cc.NewStream(ctx, &_Video_serviceDesc.Streams[2], "/Video/MotionRaw", opts...)
	if err != nil {
		return nil, err
	}
	x := &videoMotionRawClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Video_MotionRawClient interface {
	Recv() (*Frame, error)
	grpc.ClientStream
}

type videoMotionRawClient struct {
	grpc.ClientStream
}

func (x *videoMotionRawClient) Recv() (*Frame, error) {
	m := new(Frame)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// VideoServer is the server API for Video service.
type VideoServer interface {
	MetaData(context.Context, *empty.Empty) (*VideoMetaData, error)
	FrameRaw(context.Context, *empty.Empty) (*Frame, error)
	VideoRaw(*empty.Empty, Video_VideoRawServer) error
	FrameJPEG(context.Context, *empty.Empty) (*Frame, error)
	VideoJPEG(*empty.Empty, Video_VideoJPEGServer) error
	MotionRaw(*empty.Empty, Video_MotionRawServer) error
}

// UnimplementedVideoServer can be embedded to have forward compatible implementations.
type UnimplementedVideoServer struct {
}

func (*UnimplementedVideoServer) MetaData(ctx context.Context, req *empty.Empty) (*VideoMetaData, error) {
	return nil, status.Errorf(codes.Unimplemented, "method MetaData not implemented")
}
func (*UnimplementedVideoServer) FrameRaw(ctx context.Context, req *empty.Empty) (*Frame, error) {
	return nil, status.Errorf(codes.Unimplemented, "method FrameRaw not implemented")
}
func (*UnimplementedVideoServer) VideoRaw(req *empty.Empty, srv Video_VideoRawServer) error {
	return status.Errorf(codes.Unimplemented, "method VideoRaw not implemented")
}
func (*UnimplementedVideoServer) FrameJPEG(ctx context.Context, req *empty.Empty) (*Frame, error) {
	return nil, status.Errorf(codes.Unimplemented, "method FrameJPEG not implemented")
}
func (*UnimplementedVideoServer) VideoJPEG(req *empty.Empty, srv Video_VideoJPEGServer) error {
	return status.Errorf(codes.Unimplemented, "method VideoJPEG not implemented")
}
func (*UnimplementedVideoServer) MotionRaw(req *empty.Empty, srv Video_MotionRawServer) error {
	return status.Errorf(codes.Unimplemented, "method MotionRaw not implemented")
}

func RegisterVideoServer(s *grpc.Server, srv VideoServer) {
	s.RegisterService(&_Video_serviceDesc, srv)
}

func _Video_MetaData_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(empty.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VideoServer).MetaData(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Video/MetaData",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VideoServer).MetaData(ctx, req.(*empty.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _Video_FrameRaw_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(empty.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VideoServer).FrameRaw(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Video/FrameRaw",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VideoServer).FrameRaw(ctx, req.(*empty.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _Video_VideoRaw_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(empty.Empty)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(VideoServer).VideoRaw(m, &videoVideoRawServer{stream})
}

type Video_VideoRawServer interface {
	Send(*Frame) error
	grpc.ServerStream
}

type videoVideoRawServer struct {
	grpc.ServerStream
}

func (x *videoVideoRawServer) Send(m *Frame) error {
	return x.ServerStream.SendMsg(m)
}

func _Video_FrameJPEG_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(empty.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VideoServer).FrameJPEG(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Video/FrameJPEG",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VideoServer).FrameJPEG(ctx, req.(*empty.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _Video_VideoJPEG_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(empty.Empty)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(VideoServer).VideoJPEG(m, &videoVideoJPEGServer{stream})
}

type Video_VideoJPEGServer interface {
	Send(*Frame) error
	grpc.ServerStream
}

type videoVideoJPEGServer struct {
	grpc.ServerStream
}

func (x *videoVideoJPEGServer) Send(m *Frame) error {
	return x.ServerStream.SendMsg(m)
}

func _Video_MotionRaw_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(empty.Empty)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(VideoServer).MotionRaw(m, &videoMotionRawServer{stream})
}

type Video_MotionRawServer interface {
	Send(*Frame) error
	grpc.ServerStream
}

type videoMotionRawServer struct {
	grpc.ServerStream
}

func (x *videoMotionRawServer) Send(m *Frame) error {
	return x.ServerStream.SendMsg(m)
}

var _Video_serviceDesc = grpc.ServiceDesc{
	ServiceName: "Video",
	HandlerType: (*VideoServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "MetaData",
			Handler:    _Video_MetaData_Handler,
		},
		{
			MethodName: "FrameRaw",
			Handler:    _Video_FrameRaw_Handler,
		},
		{
			MethodName: "FrameJPEG",
			Handler:    _Video_FrameJPEG_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "VideoRaw",
			Handler:       _Video_VideoRaw_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "VideoJPEG",
			Handler:       _Video_VideoJPEG_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "MotionRaw",
			Handler:       _Video_MotionRaw_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "videoService.proto",
}
