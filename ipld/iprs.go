package iprs_ipld

import (
	"errors"
	"math"

	node "gx/ipfs/QmNwUEK7QbwSqyKBu3mMtToo8SUc6wQJ7gdZq4gGGJqfnf/go-ipld-format"
	blocks "gx/ipfs/QmYsEQydGrsxNZfAiskvQ76N2xE9hDQtSAkRSynwMiUK3c/go-block-format"
	cid "gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"
	cborld "gx/ipfs/QmeZv9VXw2SfVbX55LV6kGTWASKBc9ZxAVqGBeJcDGdoXy/go-ipld-cbor"
)

const Version = 1

// TODO: Add to https://github.com/ipfs/go-cid/blob/master/cid.go
const CodecIprsCbor = 0xd0
const CodecIpns = 0xd8

type IprsVerificationType uint64

const (
	// Key verification verifies a record is signed with a private key
	VerificationType_Key IprsVerificationType = iota
	// Cert verification verifies a record is signed by a certificate issued by a CA
	VerificationType_Cert IprsVerificationType = iota
)

type IprsValidationType uint64

const (
	// Setting an EOL says "this record is valid until..."
	ValidationType_EOL IprsValidationType = iota
	// Setting a time range says "this record is valid between x and y"
	ValidationType_TimeRange IprsValidationType = iota
)

type Validity struct {
	VerificationType IprsVerificationType
	Verification     interface{}
	ValidationType   IprsValidationType
	Validation       interface{}
}

func (v *Validity) Map() map[string]interface{} {
	return map[string]interface{}{
		"verificationType": v.VerificationType,
		"verification":     v.Verification,
		"validationType":   v.ValidationType,
		"validation":       v.Validation,
	}
}

type Node struct {
	cborld.Node

	Version   uint64
	Value     *cid.Cid
	Validity  *Validity
	Signature []byte
}

func (n *Node) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"node_type": "iprs",
		"cid":       n.Cid(),
	}
}

var _ node.Node = (*Node)(nil)

func NewIprsNode(value *cid.Cid, validity *Validity, signature []byte) (*Node, error) {
	// Store the fields as a CBOR map
	obj := map[string]interface{}{
		"version":   Version,
		"value":     value,
		"validity":  validity.Map(),
		"signature": signature,
	}

	n, err := ipldCborNodeWithCodec(CodecIprsCbor, obj)
	if err != nil {
		return nil, err
	}

	return &Node{
		Version:   Version,
		Node:      *n,
		Value:     value,
		Validity:  validity,
		Signature: signature,
	}, nil
}

func DecodeIprsBlock(block blocks.Block) (*Node, error) {
	// Decode the block from CBOR format
	n, err := ipldCborNodeFromBlock(block)
	if err != nil {
		return nil, err
	}

	// Make sure it has the right structure
	versioni, _, err := n.Resolve([]string{"version"})
	version, ok := versioni.(uint64)
	if err != nil || !ok {
		return nil, errors.New("incorrectly formatted version")
	}
	vall, _, err := n.ResolveLink([]string{"value"})
	if err != nil {
		return nil, errors.New("incorrectly formatted value link")
	}
	_, _, err = n.Resolve([]string{"validity"})
	if err != nil {
		return nil, errors.New("incorrectly formatted validity")
	}
	vfti, _, err := n.Resolve([]string{"validity", "verificationType"})
	vft, ok := vfti.(uint64)
	if err != nil || !ok {
		return nil, errors.New("incorrectly formatted verificationType")
	}
	vlti, _, err := n.Resolve([]string{"validity", "validationType"})
	vlt, ok := vlti.(uint64)
	if err != nil || !ok {
		return nil, errors.New("incorrectly formatted validationType")
	}
	verificationi, _, err := n.Resolve([]string{"validity", "verification"})
	validationi, _, err := n.Resolve([]string{"validity", "validation"})
	sigi, _, err := n.Resolve([]string{"signature"})
	sig, ok := sigi.([]byte)
	if err != nil || !ok {
		return nil, errors.New("incorrectly formatted signature")
	}

	return &Node{
		Node:    *n,
		Version: version,
		Value:   vall.Cid,
		Validity: &Validity{
			VerificationType: IprsVerificationType(vft),
			Verification:     verificationi,
			ValidationType:   IprsValidationType(vlt),
			Validation:       validationi,
		},
		Signature: sig,
	}, nil
}

// Used by IPLD's block decoder to decode blocks into generic IPLD nodes
func DecodeIprsBlockGenericNode(block blocks.Block) (node.Node, error) {
	return DecodeIprsBlock(block)
}

var _ node.DecodeBlockFunc = DecodeIprsBlockGenericNode

func ipldCborNodeWithCodec(codec uint64, obj map[string]interface{}) (*cborld.Node, error) {
	nd, err := cborld.WrapObject(obj, math.MaxUint64, -1)
	if err != nil {
		return nil, err
	}

	// Use the same CID but with a codec specific to Iprs
	c := cid.NewCidV1(codec, nd.Cid().Hash())
	b, err := blocks.NewBlockWithCid(nd.RawData(), c)
	if err != nil {
		return nil, err
	}

	return ipldCborNodeFromBlock(b)
}

func ipldCborNodeFromBlock(block blocks.Block) (*cborld.Node, error) {
	// Decode the block from CBOR format
	nd, err := cborld.DecodeBlock(block)
	if err != nil {
		return nil, err
	}

	n, ok := nd.(*cborld.Node)
	if !ok {
		panic("Expected DecodeBlock to return go-ipld-cbor Node")
	}

	return n, nil
}

func init() {
	node.Register(CodecIprsCbor, DecodeIprsBlockGenericNode)
}
