package iprs_record

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	ld "github.com/dirkmc/go-iprs/ipld"
	rsp "github.com/dirkmc/go-iprs/path"
	node "gx/ipfs/QmNwUEK7QbwSqyKBu3mMtToo8SUc6wQJ7gdZq4gGGJqfnf/go-ipld-format"
	u "gx/ipfs/QmPsAfmDBnZN3kZGSuNwvCNDZiHneERSKmRcFyG3UkvcT3/go-ipfs-util"
)

// ErrExpiredRecord should be returned when an Iprs record is
// invalid due to being too old
var ErrExpiredRecord = errors.New("expired record")

type EolRecordValidation struct {
	eol time.Time
}

func NewEolRecordValidation(eol time.Time) *EolRecordValidation {
	return &EolRecordValidation{eol}
}

func (v *EolRecordValidation) Nodes() ([]node.Node, error) {
	return []node.Node{}, nil
}

func (v *EolRecordValidation) ValidationType() ld.IprsValidationType {
	return ld.ValidationType_EOL
}

func (v *EolRecordValidation) Validation() (interface{}, error) {
	return u.FormatRFC3339(v.eol), nil
}

func prepareEolSig(o interface{}) ([]byte, error) {
	s, ok := o.(string)
	if !ok {
		return nil, fmt.Errorf("Unrecognized validation data type %T. Expected string", o)
	}
	return []byte(s), nil
}

type eolRecordChecker struct{}

func (v *eolRecordChecker) ValidateRecord(ctx context.Context, iprsKey rsp.IprsPath, record *Record) error {
	t, err := EolParseValidation(record)
	if err != nil {
		log.Warningf("Failed to parse time from IPRS record EOL [%s]", record.Validity.Validation)
		return err
	}
	if time.Now().After(t) {
		return ErrExpiredRecord
	}
	return nil
}

func EolParseValidation(record *Record) (time.Time, error) {
	s, ok := record.Validity.Validation.(string)
	if !ok {
		return time.Time{}, fmt.Errorf("Unrecognized validation data type %T. Expected string", record.Validity.Validation)
	}
	return u.ParseRFC3339(s)
}

func (v *eolRecordChecker) SelectRecord(recs []*Record) (int, error) {
	best_i := -1

	for i, r := range recs {
		if r == nil {
			continue
		}

		if best_i == -1 {
			best_i = i
			continue
		}

		rt, err := EolParseValidation(r)
		if err != nil {
			continue
		}

		bestt, err := EolParseValidation(recs[best_i])
		if err != nil {
			continue
		}

		if rt.After(bestt) {
			best_i = i
			continue
		}

		if rt == bestt {
			// Neither is better so just compare bytes to
			// make sure its deterministic
			vi := recs[i].Cid().Bytes()
			vb := recs[best_i].Cid().Bytes()
			if bytes.Compare(vi, vb) > 0 {
				best_i = i
			}
		}
	}
	if best_i == -1 {
		return 0, errors.New("No usable records in given record set")
	}

	return best_i, nil
}

var EolRecordChecker = &eolRecordChecker{}

func init() {
	ValidationSigPreparer[ld.ValidationType_EOL] = prepareEolSig
}
