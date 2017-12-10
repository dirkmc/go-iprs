package recordstore_record

import (
	"context"
	"crypto/x509"
	"crypto/rsa"
	"fmt"
	"time"
	c "github.com/dirkmc/go-iprs/certificate"
	ci "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
	path "github.com/ipfs/go-ipfs/path"
	pb "github.com/dirkmc/go-iprs/pb"
	routing "gx/ipfs/QmPR2JzfKd9poHx9XBhzoFeBBC31ZM3W5iUPKJZWyaoZZm/go-libp2p-routing"
)

type RecordFactory struct {
	eolManager *EolRecordManager
	rangeManager *RangeRecordManager
	certManager *CertRecordManager
	managers map[pb.IprsEntry_ValidityType]RecordManager
}

func NewRecordFactory(r routing.ValueStore) *RecordFactory {
	certManager := c.NewCertificateManager(r)
	pubkManager := NewPublicKeyManager(r)

	eolm := NewEolRecordManager(r, pubkManager)
	rangem := NewRangeRecordManager(r, pubkManager)
	certm := NewCertRecordManager(r, certManager)
	managers := map[pb.IprsEntry_ValidityType]RecordManager{
		pb.IprsEntry_EOL:       eolm,
		pb.IprsEntry_TimeRange: rangem,
		pb.IprsEntry_Cert:      certm,
	}

	return &RecordFactory{
		eolManager: eolm,
		rangeManager: rangem,
		certManager: certm,
		managers: managers,
	}
}

// Verifies that the given record is correctly signed etc
func (f *RecordFactory) Verify(ctx context.Context, iprsKey string, entry *pb.IprsEntry) error {
	manager, ok := f.managers[entry.GetValidityType()]
	if !ok {
		return fmt.Errorf("Unrecognized validity type %s", entry.GetValidityType().String())
	}
	return manager.VerifyRecord(ctx, iprsKey, entry)
}

func (f *RecordFactory) NewEolRecord(pk ci.PrivKey, p path.Path, eol time.Time) *EolRecord {
	return f.eolManager.NewRecord(pk, p, eol)
}

func (f *RecordFactory) NewRangeRecord(pk ci.PrivKey, p path.Path, start *time.Time, end *time.Time) (*RangeRecord, error) {
	return f.rangeManager.NewRecord(pk, p, start, end)
}

func (f *RecordFactory) NewCertRecord(pk *rsa.PrivateKey, cert *x509.Certificate, p path.Path, eol time.Time) *CertRecord {
	return f.certManager.NewRecord(pk, cert, p, eol)
}
