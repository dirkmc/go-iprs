package recordstore_record

import (
	"context"
	"fmt"
	pb "github.com/dirkmc/go-iprs/pb"
	routing "gx/ipfs/QmPR2JzfKd9poHx9XBhzoFeBBC31ZM3W5iUPKJZWyaoZZm/go-libp2p-routing"
	ci "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
)

type SignedRecordManager struct {
	routing routing.ValueStore
	pubkManager *PublicKeyManager
}

func NewSignedRecordManager(r routing.ValueStore, m *PublicKeyManager) *SignedRecordManager {
	return &SignedRecordManager{
		routing: r,
		pubkManager: m,
	}
}

func (m *SignedRecordManager) PublishRecord(ctx context.Context, iprsKey string, entry *pb.IprsEntry, pk ci.PrivKey) error {
	// Sign the record
	sig, err := pk.Sign(RecordDataForSig(entry))
	if err != nil {
		return err
	}
	entry.Signature = sig

	// Put the public key and the record itself to routing
	resp := make(chan error, 2)

	go func() {
		resp <- m.pubkManager.PutPublicKey(ctx, pk.GetPublic())
	}()
	go func() {
		resp <- PutEntryToRouting(ctx, m.routing, iprsKey, entry)
	}()

	for i := 0; i < 2; i++ {
		err = <-resp
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *SignedRecordManager) CheckPublicKeySignature(ctx context.Context, iprsKey string, entry *pb.IprsEntry) error {
	pubk, err := m.pubkManager.GetPublicKey(ctx, iprsKey)
	if err != nil {
		return err
	}

	if ok, err := pubk.Verify(RecordDataForSig(entry), entry.GetSignature()); err != nil || !ok {
		return fmt.Errorf("Invalid record value. Not signed by private key corresponding to public key %v", pubk)
	}

	return nil
}
