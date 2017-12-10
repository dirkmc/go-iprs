package recordstore_record

import (
	"context"
	"time"
	ci "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
	routing "gx/ipfs/QmPR2JzfKd9poHx9XBhzoFeBBC31ZM3W5iUPKJZWyaoZZm/go-libp2p-routing"
	rsp "github.com/dirkmc/go-iprs/path"
	u "github.com/ipfs/go-ipfs-util"
)

const PublicKeyPutTimeout = time.Second*10

type PublicKeyManager struct {
	routing routing.ValueStore
}

func NewPublicKeyManager(r routing.ValueStore) *PublicKeyManager {
	return &PublicKeyManager{
		routing: r,
	}
}

func (m *PublicKeyManager) PutPublicKey(ctx context.Context, pubk ci.PubKey) error {
	pubkBytes, err := pubk.Bytes()
	if err != nil {
		return err
	}

	timectx, cancel := context.WithTimeout(ctx, PublicKeyPutTimeout)
	defer cancel()

	// Note: IPFS stores public keys internally at string(hash)
	// not at B58String(hash)
	pubkPath := "/pk/" + string(u.Hash(pubkBytes))
	log.Debugf("Storing public key at %s", pubkPath)
	if err := m.routing.PutValue(timectx, pubkPath, pubkBytes); err != nil {
		log.Warningf("Failed to put public key at %s: %s", pubkPath, err)
		return err
	}

	return nil
}

func (m *PublicKeyManager) GetPublicKey(ctx context.Context, iprsKey rsp.IprsPath) (ci.PubKey, error) {
	pkHash := iprsKey.GetHash()
	pubk, err := routing.GetPublicKey(m.routing, ctx, pkHash)
	if err != nil {
		log.Warningf("Failed to get public key %s", string(pkHash))
		return nil, err
	}

	return pubk, nil
}
