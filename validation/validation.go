package iprs_validation

import (
	"errors"
	"fmt"
	rsp "github.com/dirkmc/go-iprs/path"
	pb "github.com/dirkmc/go-iprs/pb"
	rec "github.com/dirkmc/go-iprs/record"
	proto "github.com/gogo/protobuf/proto"
	record "github.com/libp2p/go-libp2p-record"
	logging "github.com/ipfs/go-log"
)

var log = logging.Logger("iprs.validation")

// ErrUnrecognizedValidityType is returned when an IprsEntry has an
// unknown record type.
var ErrUnrecognizedValidityType = errors.New("unrecognized validity type")

type recordChecker struct {
	ValidChecker *record.ValidChecker
	Selector     record.SelectorFunc
}

func newRecordChecker() *recordChecker {
	validators := map[pb.IprsEntry_ValidityType]rec.RecordChecker{
		pb.IprsEntry_EOL:       rec.EolRecordChecker,
		pb.IprsEntry_TimeRange: rec.RangeRecordChecker,
	}

	// Implements ValidatorFunc and verifies that the
	// given 'val' is an IprsEntry and that that entry is valid.
	validateRecord := func(k string, val []byte) error {
		entry := new(pb.IprsEntry)
		err := proto.Unmarshal(val, entry)
		if err != nil {
			return err
		}

		validator, ok := validators[entry.GetValidityType()]
		if !ok {
			return ErrUnrecognizedValidityType
		}

		p, err := rsp.FromString(k)
		if err != nil {
			return fmt.Errorf("Invalid IPRS string passed to ValidateFunc [%s]", k)
		}
		return validator.ValidateRecord(p, entry)
	}

	getEntry := func(i int, length int, val []byte, firstEntry *pb.IprsEntry) *pb.IprsEntry {
		entry := new(pb.IprsEntry)
		err := proto.Unmarshal(val, entry)
		if err != nil {
			log.Warningf("Could not parse IprsEntry from []byte at index %d of %d", i, length)
			return nil
		}

		_, ok := validators[entry.GetValidityType()]
		if !ok {
			t := entry.GetValidityType().String()
			log.Warningf("No validator found for record with validity type %s at index %d of %d", t, i, length)
			return nil
		}

		if firstEntry == nil {
			return entry
		}

		if entry.GetValidityType() != firstEntry.GetValidityType() {
			t1 := firstEntry.GetValidityType().String()
			t2 := entry.GetValidityType().String()
			log.Warningf("Records have mixed validity types (%s and %s). Ignoring %s at index %d of %d", t1, t2, t2, i, length)
			return nil
		}

		return entry
	}

	// Implements SelectorFunc
	selectRecord := func(k string, vals [][]byte) (int, error) {
		NoUsableRecordsError := errors.New("No usable records in given record set")
		if len(vals) == 0 {
			return 0, NoUsableRecordsError
		}

		var firstEntry *pb.IprsEntry
		var entries []*pb.IprsEntry
		for i, val := range vals {
			entry := getEntry(i, len(vals), val, firstEntry)
			if firstEntry == nil && entry != nil {
				firstEntry = entry
			}
			entries = append(entries, entry)
		}

		if firstEntry == nil {
			return 0, NoUsableRecordsError
		}

		validator, ok := validators[firstEntry.GetValidityType()]
		if !ok {
			return 0, NoUsableRecordsError
		}

		return validator.SelectRecord(entries, vals)
	}

	return &recordChecker{
		ValidChecker: &record.ValidChecker{
			Func: validateRecord,
			Sign: true,
		},
		Selector: selectRecord,
	}
}

var RecordChecker = newRecordChecker()
