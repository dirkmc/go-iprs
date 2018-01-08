package iprs_ipld

import (
	"errors"

	node "gx/ipfs/QmNwUEK7QbwSqyKBu3mMtToo8SUc6wQJ7gdZq4gGGJqfnf/go-ipld-format"
	mh "gx/ipfs/QmYeKnKpubCMRiq3PGZcTREErthbb5Q9cXsCoSkD9bjEBd/go-multihash"
	blocks "gx/ipfs/QmYsEQydGrsxNZfAiskvQ76N2xE9hDQtSAkRSynwMiUK3c/go-block-format"
	cid "gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"
)

// TODO: Add to https://github.com/ipfs/go-cid/blob/master/cid.go
const CodecPubKeyRaw = 0xf0

// A PEM encoded Public Key
type PublicKey []byte

func DecodePublicKeyBlock(block blocks.Block) (node.Node, error) {
	prefix := block.Cid().Prefix()

	if prefix.Codec != CodecPubKeyRaw || prefix.MhType != mh.SHA2_256 || prefix.MhLength != mh.DefaultLengths[mh.SHA2_256] {
		return nil, errors.New("invalid CID prefix for PublicKey block")
	}

	return PublicKey(block.RawData()), nil
}

var _ node.DecodeBlockFunc = DecodePublicKeyBlock

func (c PublicKey) Cid() *cid.Cid {
	certCid, _ := cid.Prefix{
		MhType:   mh.SHA2_256,
		MhLength: -1,
		Codec:    CodecPubKeyRaw,
		Version:  1,
	}.Sum([]byte(c))
	return certCid
}

func (c PublicKey) Copy() node.Node {
	out := make([]byte, len(c))
	copy(out, c)
	return PublicKey(out)
}

func (c PublicKey) Links() []*node.Link {
	return nil
}

func (c PublicKey) Resolve(_ []string) (interface{}, []string, error) {
	return nil, nil, errors.New("no such link")
}

func (c PublicKey) ResolveLink(_ []string) (*node.Link, []string, error) {
	return nil, nil, errors.New("no such link")
}

func (c PublicKey) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"type": "pubkey",
	}
}

func (c PublicKey) RawData() []byte {
	return []byte(c)
}

func (c PublicKey) Size() (uint64, error) {
	return uint64(len(c)), nil
}

func (c PublicKey) Stat() (*node.NodeStat, error) {
	return &node.NodeStat{}, nil
}

func (c PublicKey) String() string {
	return "[pubk]"
}

func (c PublicKey) Tree(p string, depth int) []string {
	return nil
}

var _ node.Node = (PublicKey)(nil)

func init() {
	node.Register(CodecPubKeyRaw, DecodePublicKeyBlock)
}
