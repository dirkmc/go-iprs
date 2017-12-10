package recordstore_record

import (
	"context"
	"fmt"
	c "github.com/dirkmc/go-iprs/certificate"
	pb "github.com/dirkmc/go-iprs/pb"
	routing "gx/ipfs/QmPR2JzfKd9poHx9XBhzoFeBBC31ZM3W5iUPKJZWyaoZZm/go-libp2p-routing"
)

type RecordFactory struct {
	certManager *c.CertificateManager
	pubkManager *PublicKeyManager
	managers map[pb.IprsEntry_ValidityType]RecordManager
}

func NewRecordFactory(r routing.ValueStore) *RecordFactory {
	certManager := c.NewCertificateManager(r)
	pubkManager := NewPublicKeyManager(r)

	managers := map[pb.IprsEntry_ValidityType]RecordManager{
		pb.IprsEntry_EOL:       NewEolRecordManager(r, pubkManager),
		pb.IprsEntry_TimeRange: NewRangeRecordManager(r, pubkManager),
		pb.IprsEntry_Cert:      NewCertRecordManager(r, certManager),
	}

	return &RecordFactory{
		certManager: certManager,
		pubkManager: pubkManager,
		managers: managers,
	}
}

// Verifies that the given record is correctly signed etc
func (v *RecordFactory) Verify(ctx context.Context, iprsKey string, entry *pb.IprsEntry) error {
	manager, ok := v.managers[entry.GetValidityType()]
	if !ok {
		return fmt.Errorf("Unrecognized validity type %s", entry.GetValidityType().String())
	}
	return manager.VerifyRecord(ctx, iprsKey, entry)
}
