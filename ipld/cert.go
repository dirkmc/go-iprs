package iprs_ipld

import (
	"errors"

	blocks "github.com/ipfs/go-block-format"
	cid "github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
	node "github.com/ipfs/go-ipld-format"
)

// TODO: Add to https://github.com/ipfs/go-cid/blob/master/cid.go
const CodecCertRaw = 0xe0

// A PEM encoded x509 Certificate
type Certificate []byte

func DecodeBlock(block blocks.Block) (node.Node, error) {
	prefix := block.Cid().Prefix()

	if prefix.Codec != CodecCertRaw || prefix.MhType != mh.SHA2_256 || prefix.MhLength != mh.DefaultLengths[mh.SHA2_256] {
		return nil, errors.New("invalid CID prefix")
	}

	return Certificate(block.RawData()), nil
}

var _ node.DecodeBlockFunc = DecodeBlock

func (c Certificate) Cid() *cid.Cid {
	certCid, _ := cid.Prefix{
		MhType:   mh.SHA2_256,
		MhLength: -1,
		Codec:    CodecCertRaw,
		Version:  1,
	}.Sum([]byte(c))
	return certCid
}

func (c Certificate) Copy() node.Node {
	out := make([]byte, len(c))
	copy(out, c)
	return Certificate(out)
}

func (c Certificate) Links() []*node.Link {
	return nil
}

func (c Certificate) Resolve(_ []string) (interface{}, []string, error) {
	return nil, nil, errors.New("no such link")
}

func (c Certificate) ResolveLink(_ []string) (*node.Link, []string, error) {
	return nil, nil, errors.New("no such link")
}

func (c Certificate) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"type": "cert",
	}
}

func (c Certificate) RawData() []byte {
	return []byte(c)
}

func (c Certificate) Size() (uint64, error) {
	return uint64(len(c)), nil
}

func (c Certificate) Stat() (*node.NodeStat, error) {
	return &node.NodeStat{}, nil
}

func (c Certificate) String() string {
	return "[cert]"
}

func (c Certificate) Tree(p string, depth int) []string {
	return nil
}

var _ node.Node = (Certificate)(nil)
