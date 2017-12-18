package iprs_record

import (
	"context"
	"fmt"
	pb "github.com/dirkmc/go-iprs/pb"
	rsp "github.com/dirkmc/go-iprs/path"
	ci "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
)


type KeyRecordSigner struct {
	m *PublicKeyManager
	pk ci.PrivKey
}

func NewKeyRecordSigner(m *PublicKeyManager, pk ci.PrivKey) *KeyRecordSigner {
	return &KeyRecordSigner{ m, pk }
}

func (s *KeyRecordSigner) BasePath() (rsp.IprsPath, error) {
	h, err := GetPublicKeyHash(s.pk.GetPublic())
	if err != nil {
		return rsp.NilPath, err
	}
	return rsp.FromString("/iprs/" + h)
}

func (s *KeyRecordSigner) VerificationType() *pb.IprsEntry_VerificationType {
	t := pb.IprsEntry_Key
	return &t
}

func (s *KeyRecordSigner) Verification() ([]byte, error) {
	return []byte{}, nil
}

func (s *KeyRecordSigner) PublishVerification(ctx context.Context, iprsKey rsp.IprsPath, entry *pb.IprsEntry) error {
	// TODO: Check iprsKey is valid for this type of RecordSigner
	return s.m.PutPublicKey(ctx, s.pk.GetPublic())
}

func (s *KeyRecordSigner) SignRecord(entry *pb.IprsEntry) error {
	sig, err := s.pk.Sign(RecordDataForSig(entry))
	if err != nil {
		return err
	}
	entry.Signature = sig

	return nil
}


type KeyRecordVerifier struct {
	m *PublicKeyManager
}

func NewKeyRecordVerifier(m *PublicKeyManager) *KeyRecordVerifier {
	return &KeyRecordVerifier{ m }
}

func (v *KeyRecordVerifier) VerifyRecord(ctx context.Context, iprsKey rsp.IprsPath, entry *pb.IprsEntry) error {
	pubk, err := v.m.GetPublicKey(ctx, iprsKey)
	if err != nil {
		return err
	}

	if ok, err := pubk.Verify(RecordDataForSig(entry), entry.GetSignature()); err != nil || !ok {
		return fmt.Errorf("Invalid record value. Not signed by private key corresponding to public key %v", pubk)
	}

	return nil
}
