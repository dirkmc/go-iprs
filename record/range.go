package iprs_record

import (
	"bytes"
	"errors"
	"strings"
	"time"
	pb "github.com/dirkmc/go-iprs/pb"
	rsp "github.com/dirkmc/go-iprs/path"
	u "github.com/ipfs/go-ipfs-util"
)

// ErrRecordTimeRange should be returned when an attempt is made to
// construct an Iprs record with an end time before the start time
var ErrRecordTimeRange = errors.New("record end time before start time")
// ErrPendingRecord should be returned when an Iprs record is
// invalid due to not yet being valid
var ErrPendingRecord = errors.New("record not yet valid")


type RangeRecordValidity struct {
	start *time.Time
	end *time.Time
}

func NewRangeRecordValidity(start *time.Time, end *time.Time) (*RangeRecordValidity, error) {
	if start != nil && end != nil && (*start).After(*end) {
		return nil, ErrRecordTimeRange
	}

	return &RangeRecordValidity{ start, end }, nil
}

func (v *RangeRecordValidity) Validity() []byte {
	startFmt := "-∞"
	if v.start != nil {
		startFmt = u.FormatRFC3339(*v.start)
	}
	endFmt := "∞"
	if v.end != nil {
		endFmt = u.FormatRFC3339(*v.end)
	}

	return []byte(startFmt + "~" + endFmt)
}

func (v *RangeRecordValidity) ValidityType() *pb.IprsEntry_ValidityType {
	t := pb.IprsEntry_TimeRange
	return &t
}


//rangeRecordChecker

type rangeRecordChecker struct {}

func (v *rangeRecordChecker) SelectRecord(recs []*pb.IprsEntry, vals [][]byte) (int, error) {
	var best_seq uint64
	best_i := -1

	for i, r := range recs {
		// Best record is the one with the highest sequence number
		if r == nil || r.GetSequence() < best_seq {
			continue
		}

		if best_i == -1 || r.GetSequence() > best_seq {
			best_seq = r.GetSequence()
			best_i = i
		} else if r.GetSequence() == best_seq {
			// If sequence number is equal, look at time range
			t, err := RangeParseValidity(r)
			if err != nil {
				continue
			}

			bestt, err := RangeParseValidity(recs[best_i])
			if err != nil {
				continue
			}

			// Best record is the one that's valid to the latest possible moment
			if t[1] == nil && bestt[1] != nil || (t[1] != nil && bestt[1] != nil && (*t[1]).After(*bestt[1])) {
				best_i = i
			} else if t[1] == bestt[1] {
				// If records are valid until an equal time, best record
				// is the one that's valid since the longest time in the past
				if t[0] == nil && bestt[0] != nil || (t[0] != nil && bestt[0] != nil && (*t[0]).Before(*bestt[0])) {
					best_i = i
				} else if t[0] == bestt[0] {
					// This is just to make sure the selection is deterministic
					if bytes.Compare(vals[i], vals[best_i]) > 0 {
						best_i = i
					}
				}
			}
		}
	}
	if best_i == -1 {
		return 0, errors.New("no usable records in given set")
	}

	return best_i, nil
}

func RangeParseValidity(r *pb.IprsEntry) (*[2]*time.Time, error) {
	timeRange := strings.Split(string(r.GetValidity()), "~")
	if len(timeRange) != 2 {
		return nil, errors.New("Invalid TimeRange record")
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

func (v *rangeRecordChecker) ValidateRecord(iprsKey rsp.IprsPath, entry *pb.IprsEntry) error {
	t, err := RangeParseValidity(entry)
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
