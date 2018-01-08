package iprs_record

import (
	"context"
	"fmt"
	"time"

	ld "github.com/dirkmc/go-iprs/ipld"
	node "gx/ipfs/QmNwUEK7QbwSqyKBu3mMtToo8SUc6wQJ7gdZq4gGGJqfnf/go-ipld-format"
	ci "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
	cid "gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"
	// u "github.com/ipfs/go-ipfs-util"
)

//const PublicKeyPutTimeout = time.Second * 10
const PubKeyFetchTimeout = time.Second * 10

type PublicKeyManager struct {
	dag node.NodeGetter
}

func NewPublicKeyManager(dag node.NodeGetter) *PublicKeyManager {
	return &PublicKeyManager{dag}
}

/*
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
*/
func (m *PublicKeyManager) GetPublicKey(ctx context.Context, pubkCid *cid.Cid) (ci.PubKey, error) {
	log.Debugf("PublicKeyManager get public key %s", pubkCid)
	if pubkCid.Type() != ld.CodecPubKeyRaw {
		return nil, fmt.Errorf("Cid Codec %d is not CodecPubKeyRaw in Cid %s", ld.CodecPubKeyRaw, pubkCid)
	}

	log.Debugf("Fetching public key at %s", pubkCid)

	timectx, cancel := context.WithTimeout(ctx, PubKeyFetchTimeout)
	defer cancel()

	n, err := m.dag.Get(timectx, pubkCid)
	if err != nil {
		log.Warningf("Failed to fetch public key at %s: %s", pubkCid, err)
		return nil, err
	}

	pubk, err := ci.UnmarshalPublicKey(n.RawData())
	if err != nil {
		log.Warningf("Failed to unmarshal public key at %s: %s", pubkCid, err)
		return nil, err
	}

	return pubk, nil
}
