package recordstore

import (
	"errors"
	pb "github.com/dirkmc/go-libp2p-kad-record-store/pb"
	eol "github.com/dirkmc/go-libp2p-kad-record-store/types/eol"
	timeRange "github.com/dirkmc/go-libp2p-kad-record-store/types/range"
	proto "github.com/gogo/protobuf/proto"
	record "github.com/libp2p/go-libp2p-record"
)

// ErrUnrecognizedValidity is returned when an IprsEntry has an
// unknown validity type.
var ErrUnrecognizedValidity = errors.New("unrecognized validity type")

// ValidateRecord implements ValidatorFunc and verifies that the
// given 'val' is an IprsEntry and that that entry is valid.
func ValidateRecord(k string, val []byte) error {
	entry := new(pb.IprsEntry)
	err := proto.Unmarshal(val, entry)
	if err != nil {
		return err
	}
	switch entry.GetValidityType() {
	case pb.IprsEntry_EOL:
		return eol.ValidateRecord(k, entry)
	case pb.IprsEntry_TimeRange:
		return timeRange.ValidateRecord(k, entry)
	default:
		return ErrUnrecognizedValidity
	}
	return nil
}

var RecordValidator = &record.ValidChecker{
	Func: ValidateRecord,
	Sign: true,
}
