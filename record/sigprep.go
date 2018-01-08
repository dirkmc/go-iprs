package iprs_record

import (
	"bytes"
	"fmt"

	ld "github.com/dirkmc/go-iprs/ipld"
	cid "gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"
)

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
