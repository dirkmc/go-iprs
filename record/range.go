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

// ErrRecordTimeRange should be returned when an attempt is made to
// construct an Iprs record with an end time before the start time
var ErrRecordTimeRange = errors.New("record end time before start time")

// ErrPendingRecord should be returned when an Iprs record is
// invalid due to not yet being valid
var ErrPendingRecord = errors.New("record not yet valid")

type RangeRecordValidation struct {
	start *time.Time
	end   *time.Time
}

func NewRangeRecordValidation(start *time.Time, end *time.Time) (*RangeRecordValidation, error) {
	if start != nil && end != nil && (*start).After(*end) {
		return nil, ErrRecordTimeRange
	}

	return &RangeRecordValidation{start, end}, nil
}

func (v *RangeRecordValidation) Nodes() ([]node.Node, error) {
	return []node.Node{}, nil
}

func (v *RangeRecordValidation) ValidationType() ld.IprsValidationType {
	return ld.ValidationType_TimeRange
}

func (v *RangeRecordValidation) Validation() (interface{}, error) {
	startFmt := "-∞"
	if v.start != nil {
		startFmt = u.FormatRFC3339(*v.start)
	}
	endFmt := "∞"
	if v.end != nil {
		endFmt = u.FormatRFC3339(*v.end)
	}
	return []string{startFmt, endFmt}, nil
}

func prepareRangeSig(o interface{}) ([]byte, error) {
	s, err := interfaceToStringTuple(o)
	if err != nil {
		return nil, err
	}
	return []byte(s[0] + s[1]), nil
}

func interfaceToStringTuple(o interface{}) ([]string, error) {
	s, ok := o.([]string)
	if !ok {
		a, ok := o.([]interface{})
		if !ok {
			return nil, fmt.Errorf("Unrecognized validation data type %T. Expected array", o)
		}
		s = make([]string, len(a))
		for i := range a {
		    s[i], ok = a[i].(string)
			if !ok {
				return nil, fmt.Errorf("Unrecognized validation data type []%T. Expected []string", a[i])
			}
		}
	}
	if len(s) != 2 {
		return nil, fmt.Errorf("Unexpected validation data length %d. Expected 2", len(s))
	}
	return s, nil
}


// rangeRecordChecker

type rangeRecordChecker struct{}

func (v *rangeRecordChecker) SelectRecord(recs []*Record) (int, error) {
	best_i := -1

	for i, r := range recs {
		if r == nil {
			continue
		}

		if best_i == -1 {
			best_i = i
			continue
		}

		// Compare time range
		t, err := RangeParseValidation(r)
		if err != nil {
			continue
		}

		bestt, err := RangeParseValidation(recs[best_i])
		if err != nil {
			continue
		}

		// Best record is the one that's valid to the latest possible moment
		if t[1] == nil && bestt[1] != nil || (t[1] != nil && bestt[1] != nil && (*t[1]).After(*bestt[1])) {
			best_i = i
			continue
		}

		if t[1] == bestt[1] {
			// If records are valid until an equal time, best record
			// is the one that's valid since the longest time in the past
			if t[0] == nil && bestt[0] != nil || (t[0] != nil && bestt[0] != nil && (*t[0]).Before(*bestt[0])) {
				best_i = i
				continue
			}

			if t[0] == bestt[0] {
				// This is just to make sure the selection is deterministic
				vi := recs[i].Cid().Bytes()
				vb := recs[best_i].Cid().Bytes()
				if bytes.Compare(vi, vb) > 0 {
					best_i = i
				}
			}
		}
	}
	if best_i == -1 {
		return 0, errors.New("no usable records in given set")
	}

	return best_i, nil
}

func RangeParseValidation(r *Record) (*[2]*time.Time, error) {
	timeRange, err := interfaceToStringTuple(r.Validity.Validation)
	if err != nil {
		return nil, err
	}

	var startPt *time.Time
	if timeRange[0] != "-∞" {
		start, err := u.ParseRFC3339(timeRange[0])
		if err != nil {
			return nil, err
		}
		startPt = &start
	}
	var endPt *time.Time
	if timeRange[1] != "∞" {
		end, err := u.ParseRFC3339(timeRange[1])
		if err != nil {
			return nil, err
		}
		endPt = &end
	}

	if startPt != nil && endPt != nil && (*startPt).After(*endPt) {
		return nil, ErrRecordTimeRange
	}

	return &[2]*time.Time{startPt, endPt}, nil
}

func (v *rangeRecordChecker) ValidateRecord(ctx context.Context, iprsKey rsp.IprsPath, record *Record) error {
	t, err := RangeParseValidation(record)
	if err != nil {
		log.Warning("Failed to parse IPRS Time Range record")
		return err
	}
	if t[0] != nil && time.Now().Before(*t[0]) {
		return ErrPendingRecord
	}
	if t[1] != nil && time.Now().After(*t[1]) {
		return ErrExpiredRecord
	}
	return nil
}

var RangeRecordChecker = &rangeRecordChecker{}

func init() {
	ValidationSigPreparer[ld.ValidationType_TimeRange] = prepareRangeSig
}
