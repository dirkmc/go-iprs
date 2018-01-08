package iprs_record

import (
	"context"
	"errors"
	"fmt"

	ld "github.com/dirkmc/go-iprs/ipld"
	rsp "github.com/dirkmc/go-iprs/path"
)

var NoUsableRecordsError = errors.New("No usable records in given record set")

type masterRecordChecker struct {
	Checkers map[ld.IprsValidationType]RecordChecker
}

func NewMasterRecordChecker() *masterRecordChecker {
	checkers := make(map[ld.IprsValidationType]RecordChecker)
	checkers[ld.ValidationType_EOL] = EolRecordChecker
	checkers[ld.ValidationType_TimeRange] = RangeRecordChecker

	return &masterRecordChecker{checkers}
}

// Validates that the given record is valid (eg not expired)
func (m *masterRecordChecker) ValidateRecord(ctx context.Context, iprsKey rsp.IprsPath, record *Record) error {
	checker, ok := m.Checkers[record.Validity.ValidationType]
	if !ok {
		return fmt.Errorf("Unrecognized validation type %d", record.Validity.ValidationType)
	}
	return checker.ValidateRecord(ctx, iprsKey, record)
}

func (m *masterRecordChecker) SelectRecord(recs []*Record) (int, error) {
	var firstRec *Record
	var checkRecord = func(rec *Record, i int) bool {
		_, ok := m.Checkers[rec.Validity.ValidationType]
		if !ok {
			t := rec.Validity.ValidationType
			log.Warningf("No record checker found for record with validity type %d at index %d of %d", t, i, len(recs))
			return false
		}

		if firstRec == nil {
			firstRec = rec
			return true
		}

		if rec.Validity.ValidationType != firstRec.Validity.ValidationType {
			t1 := firstRec.Validity.ValidationType
			t2 := rec.Validity.ValidationType
			log.Warningf("Records have mixed validity types (%d and %d). Ignoring %d at index %d of %d", t1, t2, t2, i, len(recs))
			return false
		}

		return true
	}

	var usable []*Record
	for i, rec := range recs {
		if checkRecord(rec, i) {
			usable = append(usable, rec)
		}
	}

	if len(recs) == 0 {
		return 0, NoUsableRecordsError
	}

	checker, ok := m.Checkers[recs[0].Validity.ValidationType]
	if !ok {
		// In theory this is not actually possible
		return 0, NoUsableRecordsError
	}

	return checker.SelectRecord(usable)
}

var MasterRecordChecker = NewMasterRecordChecker()
