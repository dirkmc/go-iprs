// Code generated by protoc-gen-go. DO NOT EDIT.
// source: iprs.proto

/*
Package recordset_pb is a generated protocol buffer package.

It is generated from these files:
	iprs.proto

It has these top-level messages:
	IprsEntry
*/
package recordset_pb

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type IprsEntry_ValidityType int32

const (
	// setting an EOL says "this record is valid until..."
	IprsEntry_EOL IprsEntry_ValidityType = 0
)

var IprsEntry_ValidityType_name = map[int32]string{
	0: "EOL",
}
var IprsEntry_ValidityType_value = map[string]int32{
	"EOL": 0,
}

func (x IprsEntry_ValidityType) Enum() *IprsEntry_ValidityType {
	p := new(IprsEntry_ValidityType)
	*p = x
	return p
}
func (x IprsEntry_ValidityType) String() string {
	return proto.EnumName(IprsEntry_ValidityType_name, int32(x))
}
func (x *IprsEntry_ValidityType) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(IprsEntry_ValidityType_value, data, "IprsEntry_ValidityType")
	if err != nil {
		return err
	}
	*x = IprsEntry_ValidityType(value)
	return nil
}
func (IprsEntry_ValidityType) EnumDescriptor() ([]byte, []int) { return fileDescriptor0, []int{0, 0} }

type IprsEntry struct {
	Value            []byte                  `protobuf:"bytes,1,req,name=value" json:"value,omitempty"`
	Signature        []byte                  `protobuf:"bytes,2,req,name=signature" json:"signature,omitempty"`
	ValidityType     *IprsEntry_ValidityType `protobuf:"varint,3,opt,name=validityType,enum=recordset.pb.IprsEntry_ValidityType" json:"validityType,omitempty"`
	Validity         []byte                  `protobuf:"bytes,4,opt,name=validity" json:"validity,omitempty"`
	Sequence         *uint64                 `protobuf:"varint,5,opt,name=sequence" json:"sequence,omitempty"`
	Ttl              *uint64                 `protobuf:"varint,6,opt,name=ttl" json:"ttl,omitempty"`
	XXX_unrecognized []byte                  `json:"-"`
}

func (m *IprsEntry) Reset()                    { *m = IprsEntry{} }
func (m *IprsEntry) String() string            { return proto.CompactTextString(m) }
func (*IprsEntry) ProtoMessage()               {}
func (*IprsEntry) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *IprsEntry) GetValue() []byte {
	if m != nil {
		return m.Value
	}
	return nil
}

func (m *IprsEntry) GetSignature() []byte {
	if m != nil {
		return m.Signature
	}
	return nil
}

func (m *IprsEntry) GetValidityType() IprsEntry_ValidityType {
	if m != nil && m.ValidityType != nil {
		return *m.ValidityType
	}
	return IprsEntry_EOL
}

func (m *IprsEntry) GetValidity() []byte {
	if m != nil {
		return m.Validity
	}
	return nil
}

func (m *IprsEntry) GetSequence() uint64 {
	if m != nil && m.Sequence != nil {
		return *m.Sequence
	}
	return 0
}

func (m *IprsEntry) GetTtl() uint64 {
	if m != nil && m.Ttl != nil {
		return *m.Ttl
	}
	return 0
}

func init() {
	proto.RegisterType((*IprsEntry)(nil), "recordset.pb.IprsEntry")
	proto.RegisterEnum("recordset.pb.IprsEntry_ValidityType", IprsEntry_ValidityType_name, IprsEntry_ValidityType_value)
}

func init() { proto.RegisterFile("iprs.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 199 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0xca, 0x2c, 0x28, 0x2a,
	0xd6, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0xe2, 0x29, 0x4a, 0x4d, 0xce, 0x2f, 0x4a, 0x29, 0x4e,
	0x2d, 0xd1, 0x2b, 0x48, 0x52, 0x7a, 0xc3, 0xc8, 0xc5, 0xe9, 0x59, 0x50, 0x54, 0xec, 0x9a, 0x57,
	0x52, 0x54, 0x29, 0x24, 0xc2, 0xc5, 0x5a, 0x96, 0x98, 0x53, 0x9a, 0x2a, 0xc1, 0xa8, 0xc0, 0xa4,
	0xc1, 0x13, 0x04, 0xe1, 0x08, 0xc9, 0x70, 0x71, 0x16, 0x67, 0xa6, 0xe7, 0x25, 0x96, 0x94, 0x16,
	0xa5, 0x4a, 0x30, 0x81, 0x65, 0x10, 0x02, 0x42, 0x1e, 0x5c, 0x3c, 0x65, 0x89, 0x39, 0x99, 0x29,
	0x99, 0x25, 0x95, 0x21, 0x95, 0x05, 0xa9, 0x12, 0xcc, 0x0a, 0x8c, 0x1a, 0x7c, 0x46, 0x2a, 0x7a,
	0xc8, 0xd6, 0xe8, 0xc1, 0xad, 0xd0, 0x0b, 0x43, 0x52, 0x1b, 0x84, 0xa2, 0x53, 0x48, 0x8a, 0x8b,
	0x03, 0xc6, 0x97, 0x60, 0x51, 0x60, 0xd4, 0xe0, 0x09, 0x82, 0xf3, 0x41, 0x72, 0xc5, 0xa9, 0x85,
	0xa5, 0xa9, 0x79, 0xc9, 0xa9, 0x12, 0xac, 0x0a, 0x8c, 0x1a, 0x2c, 0x41, 0x70, 0xbe, 0x90, 0x00,
	0x17, 0x73, 0x49, 0x49, 0x8e, 0x04, 0x1b, 0x58, 0x18, 0xc4, 0x54, 0x12, 0xe7, 0xe2, 0x41, 0xb6,
	0x47, 0x88, 0x9d, 0x8b, 0xd9, 0xd5, 0xdf, 0x47, 0x80, 0x01, 0x10, 0x00, 0x00, 0xff, 0xff, 0xd6,
	0x7c, 0xca, 0xc0, 0x09, 0x01, 0x00, 0x00,
}
