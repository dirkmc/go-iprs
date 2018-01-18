package iprs_record

import (
	"context"
	"fmt"

	ld "github.com/dirkmc/go-iprs/ipld"
	rsp "github.com/dirkmc/go-iprs/path"
	node "gx/ipfs/QmNwUEK7QbwSqyKBu3mMtToo8SUc6wQJ7gdZq4gGGJqfnf/go-ipld-format"
	ci "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
)

type KeyRecordSigner struct {
	pk       ci.PrivKey
	pubkNode node.Node
}

func NewKeyRecordSigner(pk ci.PrivKey) *KeyRecordSigner {
	return &KeyRecordSigner{pk, nil}
}

func (s *KeyRecordSigner) VerificationType() ld.IprsVerificationType {
	return ld.VerificationType_Key
}

// Cache the Public Key node
func (s *KeyRecordSigner) getPubkNode() (node.Node, error) {
	if s.pubkNode != nil {
		return s.pubkNode, nil
	}

	b, err := s.pk.GetPublic().Bytes()
	if err != nil {
		return nil, err
	}
	s.pubkNode = ld.PublicKey(b)

	return s.pubkNode, nil
}

func (s *KeyRecordSigner) Nodes() ([]node.Node, error) {
	n, err := s.getPubkNode()
	if err != nil {
		return nil, err
	}
	return []node.Node{n}, nil
}

func (s *KeyRecordSigner) BasePath(id string) (rsp.IprsPath, error) {
	n, err := s.getPubkNode()
	if err != nil {
		return rsp.NilPath, err
	}
	return rsp.FromString("/iprs/" + n.Cid().String() + "/" + id)
}

func (s *KeyRecordSigner) SignRecord(data []byte) ([]byte, error) {
	return s.pk.Sign(data)
}

func (s *KeyRecordSigner) Verification() (interface{}, error) {
	return nil, nil
}

func prepareKeySig(o interface{}) ([]byte, error) {
	return nil, nil
}

type KeyRecordVerifier struct {
	m *PublicKeyManager
}

func NewKeyRecordVerifier(m *PublicKeyManager) *KeyRecordVerifier {
	return &KeyRecordVerifier{m}
}

func (v *KeyRecordVerifier) VerifyRecord(ctx context.Context, iprsKey rsp.IprsPath, record *Record) error {
	pubk, err := v.m.GetPublicKey(ctx, iprsKey.Cid())
	if err != nil {
		return err
	}

	// Check signature
	sigd, err := dataForSig(record.Value, record.Validity)
	if err != nil {
		return fmt.Errorf("Failed to marshall data for signature for path [%s]: %v", iprsKey, err)
	}
	if ok, err := pubk.Verify(sigd, record.Signature); err != nil || !ok {
		return fmt.Errorf("Invalid record value. Not signed by private key corresponding to public key %v", pubk)
	}

	return nil
}

func init() {
	VerificationSigPreparer[ld.VerificationType_Key] = prepareKeySig
}
