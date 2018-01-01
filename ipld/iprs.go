package iprs_ipld

import (
	"errors"
	"math"

	blocks "github.com/ipfs/go-block-format"
	cid "github.com/ipfs/go-cid"
	node "github.com/ipfs/go-ipld-format"
	cborld "github.com/ipfs/go-ipld-cbor"
)

// TODO: Add to https://github.com/ipfs/go-cid/blob/master/cid.go
const CodecIprsCbor = 0xd0

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
	Sequence uint64
	VerificationType IprsVerificationType
	Verification interface{}
	ValidationType IprsValidationType
	Validation interface{}
}

func (v *Validity) Map() map[string]interface{} {
	return map[string]interface{}{
		"sequence": v.Sequence,
		"verificationType": v.VerificationType,
		"verification": v.Verification,
		"validationType": v.ValidationType,
		"validation": v.Validation,
	}
}

type Node struct {
	cborld.Node

	Version uint64
	Value *cid.Cid
	Validity *Validity
	Signature []byte
}

func (n *Node) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"node_type": "iprs",
		"cid": n.Cid(),
	}
}

var _ node.Node = (*Node)(nil)

func NewNode(value *cid.Cid, validity *Validity, signature []byte) (*Node, error) {
	// Store the fields as a CBOR map
	obj := map[string]interface{}{
		"version": 0,
		"value": value,
		"validity": validity.Map(),
		"signature": signature,
	}

	n, err := ipldCborNodeWithCodec(CodecIprsCbor, obj)
	if err != nil {
		return nil, err
	}

	return &Node{
		Version: 0,
		Node: *n,
		Value: value,
		Validity: validity,
		Signature: signature,
	}, nil
}

func DecodeIprsBlock(block blocks.Block) (node.Node, error) {
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
	seqi, _, err := n.Resolve([]string{"validity", "sequence"})
	seq, ok := seqi.(uint64)
	if err != nil || !ok {
		return nil, errors.New("incorrectly formatted sequence number")
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
		Node: *n,
		Version: version,
		Value: vall.Cid,
		Validity: &Validity{
			Sequence: seq,
			VerificationType: IprsVerificationType(vft),
			Verification: verificationi,
			ValidationType: IprsValidationType(vlt),
			Validation: validationi,
		},
		Signature: sig,
	}, nil
}

var _ node.DecodeBlockFunc = DecodeIprsBlock

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
