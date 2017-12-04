package recordstore

import (
	"context"
	"errors"
	iprscert "github.com/dirkmc/go-libp2p-kad-record-store/certificate"
	pb "github.com/dirkmc/go-libp2p-kad-record-store/pb"
	cert "github.com/dirkmc/go-libp2p-kad-record-store/types/cert"
	eol "github.com/dirkmc/go-libp2p-kad-record-store/types/eol"
	timeRange "github.com/dirkmc/go-libp2p-kad-record-store/types/range"
	proto "github.com/gogo/protobuf/proto"
	record "github.com/libp2p/go-libp2p-record"
	routing "gx/ipfs/QmPR2JzfKd9poHx9XBhzoFeBBC31ZM3W5iUPKJZWyaoZZm/go-libp2p-routing"
)

// ErrUnrecognizedValidity is returned when an IprsEntry has an
// unknown validity type.
var ErrUnrecognizedValidity = errors.New("unrecognized validity type")

func NewIprsRecordValidator(ctx context.Context, r routing.ValueStore) *record.ValidChecker {
	certManager := iprscert.NewCertificateManager(r)

	// Implements ValidatorFunc and verifies that the
	// given 'val' is an IprsEntry and that that entry is valid.
	validateRecord := func(k string, val []byte) error {
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
		case pb.IprsEntry_Cert:
			// TODO: Where should context come from?
			return cert.ValidateRecord(ctx, k, entry, certManager)
		default:
			return ErrUnrecognizedValidity
		}
		return nil
	}

	return &record.ValidChecker{
		Func: validateRecord,
		Sign: true,
	}
}
