package recordstore

import (
	"context"
	"errors"
	c "github.com/dirkmc/go-iprs/certificate"
	pb "github.com/dirkmc/go-iprs/pb"
	rec "github.com/dirkmc/go-iprs/record"
	proto "github.com/gogo/protobuf/proto"
	record "github.com/libp2p/go-libp2p-record"
	routing "gx/ipfs/QmPR2JzfKd9poHx9XBhzoFeBBC31ZM3W5iUPKJZWyaoZZm/go-libp2p-routing"
)

// ErrUnrecognizedValidityType is returned when an IprsEntry has an
// unknown record type.
var ErrUnrecognizedValidityType = errors.New("unrecognized validity type")

func NewRecordValidator() *record.ValidChecker {
	validators := map[pb.IprsEntry_ValidityType]rec.RecordValidator{
		pb.IprsEntry_EOL: rec.NewEolRecordValidator(),
		pb.IprsEntry_TimeRange: rec.NewRangeRecordValidator(),
		pb.IprsEntry_Cert: rec.NewCertRecordValidator(),
	}

	// Implements ValidatorFunc and verifies that the
	// given 'val' is an IprsEntry and that that entry is valid.
	validateRecord := func(k string, val []byte) error {
		entry := new(pb.IprsEntry)
		err := proto.Unmarshal(val, entry)
		if err != nil {
			return err
		}

		validator, ok := validators[entry.GetValidityType()]
		if !ok {
			return ErrUnrecognizedValidityType
		}
		return validator.ValidateRecord(k, entry)
	}

	return &record.ValidChecker{
		Func: validateRecord,
		Sign: true,
	}
}

type RecordFactory struct {
	managers map[pb.IprsEntry_ValidityType]rec.RecordManager
}

func NewRecordFactory(r routing.ValueStore) *RecordFactory {
	certManager := c.NewCertificateManager(r)
	pubkManager := rec.NewPublicKeyManager(r)

	managers := map[pb.IprsEntry_ValidityType]rec.RecordManager{
		pb.IprsEntry_EOL: rec.NewEolRecordManager(r, pubkManager),
		pb.IprsEntry_TimeRange: rec.NewRangeRecordManager(r, pubkManager),
		pb.IprsEntry_Cert: rec.NewCertRecordManager(r, certManager),
	}

	return &RecordFactory{
		managers: managers,
	}
}

// Verifies that the given record is correctly signed etc
func (v *RecordFactory) Verify(ctx context.Context, iprsKey string, entry *pb.IprsEntry) error {
	manager, ok := v.managers[entry.GetValidityType()]
	if !ok {
		return ErrUnrecognizedValidityType
	}
	return manager.VerifyRecord(ctx, iprsKey, entry)
}
