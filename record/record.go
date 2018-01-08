package iprs_record

import (
	"bytes"
	"context"
	"fmt"
	"time"

	ld "github.com/dirkmc/go-iprs/ipld"
	rsp "github.com/dirkmc/go-iprs/path"
	node "gx/ipfs/QmNwUEK7QbwSqyKBu3mMtToo8SUc6wQJ7gdZq4gGGJqfnf/go-ipld-format"
	logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"
	cid "gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"
)

const PublishPutValTimeout = time.Second * 10

var log = logging.Logger("iprs.record")

type RecordValidation interface {
	ValidationType() ld.IprsValidationType
	// Return the validation data for the record
	Validation() (interface{}, error)
	// Get any nodes of data required for validation
	Nodes() ([]node.Node, error)
}

type RecordChecker interface {
	// Validates that the record has not expired etc
	ValidateRecord(ctx context.Context, iprsKey rsp.IprsPath, record *Record) error
	// Selects the best (most valid) record
	SelectRecord(recs []*Record) (int, error)
}

type RecordSigner interface {
	// Get the base IPRS path, eg /iprs/<certificate cid>
	BasePath() (rsp.IprsPath, error)
	VerificationType() ld.IprsVerificationType
	// Return the verification data for the record
	Verification() (interface{}, error)
	// Get any nodes of data required for verification
	// eg public key, certificate etc
	Nodes() ([]node.Node, error)
	// Publish any data required for verification to the network
	// eg public key, certificate etc
	//PublishVerification(ctx context.Context, iprsKey rsp.IprsPath, entry *pb.IprsEntry) error
	SignRecord([]byte) ([]byte, error)
}

type RecordVerifier interface {
	// Verifies cryptographic signatures etc
	VerifyRecord(ctx context.Context, iprsKey rsp.IprsPath, record *Record) error
}

type Record struct {
	ld.Node
	nodes []node.Node
}

type PrepareSig func(interface{}) ([]byte, error)
type VfnSigPreparer map[ld.IprsVerificationType]PrepareSig

func (s VfnSigPreparer) PrepareSig(t ld.IprsVerificationType, v interface{}) ([]byte, error) {
	p, ok := s[t]
	if !ok {
		return nil, fmt.Errorf("Unrecognized verification type %d", t)
	}
	return p(v)
}

var VerificationSigPreparer = VfnSigPreparer(map[ld.IprsVerificationType]PrepareSig{})

type VdnSigPreparer map[ld.IprsValidationType]PrepareSig

func (s VdnSigPreparer) PrepareSig(t ld.IprsValidationType, v interface{}) ([]byte, error) {
	p, ok := s[t]
	if !ok {
		return nil, fmt.Errorf("Unrecognized validation type %d", t)
	}
	return p(v)
}

var ValidationSigPreparer = VdnSigPreparer(map[ld.IprsValidationType]PrepareSig{})

func NewRecord(vl RecordValidation, s RecordSigner, val *cid.Cid) (*Record, error) {
	vfn, err := s.Verification()
	if err != nil {
		return nil, err
	}
	vdn, err := vl.Validation()
	if err != nil {
		return nil, err
	}
	validity := &ld.Validity{
		VerificationType: s.VerificationType(),
		Verification:     vfn,
		ValidationType:   vl.ValidationType(),
		Validation:       vdn,
	}

	signable, err := dataForSig(val, validity)
	if err != nil {
		return nil, err
	}

	sig, err := s.SignRecord(signable)
	if err != nil {
		return nil, err
	}

	n, err := ld.NewIprsNode(val, validity, sig)
	if err != nil {
		return nil, err
	}

	sb, err := s.Nodes()
	if err != nil {
		return nil, err
	}
	vb, err := vl.Nodes()
	if err != nil {
		return nil, err
	}
	nodes := append(sb, vb...)

	return &Record{
		Node:  *n,
		nodes: nodes,
	}, nil
}

func NewRecordFromNode(n *ld.Node) *Record {
	return &Record{
		Node:  *n,
		nodes: []node.Node{},
	}
}

func (r *Record) DependencyNodes() []node.Node {
	return r.nodes
}

func dataForSig(val *cid.Cid, v *ld.Validity) ([]byte, error) {
	vfnb, err := VerificationSigPreparer.PrepareSig(v.VerificationType, v.Verification)
	if err != nil {
		return nil, err
	}
	vdnb, err := ValidationSigPreparer.PrepareSig(v.ValidationType, v.Validation)
	if err != nil {
		return nil, err
	}

	return bytes.Join([][]byte{
		val.Bytes(),
		[]byte(fmt.Sprint(v.VerificationType)),
		vfnb,
		[]byte(fmt.Sprint(v.ValidationType)),
		vdnb,
	}, []byte{}), nil
}
