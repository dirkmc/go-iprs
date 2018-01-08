package iprs_record

import (
	"context"
	"fmt"

	c "github.com/dirkmc/go-iprs/certificate"
	ld "github.com/dirkmc/go-iprs/ipld"
	rsp "github.com/dirkmc/go-iprs/path"
	node "gx/ipfs/QmNwUEK7QbwSqyKBu3mMtToo8SUc6wQJ7gdZq4gGGJqfnf/go-ipld-format"
)

type MasterRecordVerifier struct {
	Verifiers map[ld.IprsVerificationType]RecordVerifier
}

func NewMasterRecordVerifier(dag node.NodeGetter) *MasterRecordVerifier {
	pkm := NewPublicKeyManager(dag)
	certm := c.NewCertificateManager(dag)

	verifiers := make(map[ld.IprsVerificationType]RecordVerifier)
	verifiers[ld.VerificationType_Key] = NewKeyRecordVerifier(pkm)
	verifiers[ld.VerificationType_Cert] = NewCertRecordVerifier(certm)

	return &MasterRecordVerifier{verifiers}
}

// Verifies that the given record is correctly signed etc
func (m *MasterRecordVerifier) Verify(ctx context.Context, iprsKey rsp.IprsPath, record *Record) error {
	verifier, ok := m.Verifiers[record.Validity.VerificationType]
	if !ok {
		return fmt.Errorf("Unrecognized verification type %d", record.Validity.VerificationType)
	}
	return verifier.VerifyRecord(ctx, iprsKey, record)
}
