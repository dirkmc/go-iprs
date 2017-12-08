package recordstore_record

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"time"
	pb "github.com/dirkmc/go-iprs/pb"
	path "github.com/ipfs/go-ipfs/path"
	proto "gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
	routing "gx/ipfs/QmPR2JzfKd9poHx9XBhzoFeBBC31ZM3W5iUPKJZWyaoZZm/go-libp2p-routing"
	u "github.com/ipfs/go-ipfs-util"
	ci "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
)

// ErrRecordTimeRange should be returned when an attempt is made to
// construct an Iprs record with an end time before the start time
var ErrRecordTimeRange = errors.New("record end time before start time")
// ErrPendingRecord should be returned when an Iprs record is
// invalid due to not yet being valid
var ErrPendingRecord = errors.New("record not yet valid")

// ***** RangeRecordManager ***** //
type RangeRecordManager struct {
	SignedRecordManager
}

func NewRangeRecordManager(r routing.ValueStore, m *PublicKeyManager) *RangeRecordManager {
	return &RangeRecordManager{
		SignedRecordManager {
			routing: r,
			pubkManager: m,
		},
	}
}

func (m *RangeRecordManager) NewRecord(pk ci.PrivKey, val path.Path, start *time.Time, end *time.Time) (*RangeRecord, error) {
	if start != nil && end != nil && (*start).After(*end) {
		return nil, ErrRecordTimeRange
	}

	return &RangeRecord{
		m: m,
		pk: pk,
		val: val,
		start: start,
		end: end,
	}, nil
}

func (v *RangeRecordManager) VerifyRecord(ctx context.Context, k string, entry *pb.IprsEntry) error {
	return v.CheckPublicKeySignature(ctx, k, entry)
}


// ***** RangeRecord ***** //

type RangeRecord struct {
	m *RangeRecordManager
	pk ci.PrivKey
	val path.Path
	start *time.Time
	end *time.Time
}

func (r *RangeRecord) Publish(ctx context.Context, iprsKey string, seq uint64) error {
	startFmt := "-∞"
	if r.start != nil {
		startFmt = u.FormatRFC3339(*r.start)
	}
	endFmt := "∞"
	if r.end != nil {
		endFmt = u.FormatRFC3339(*r.end)
	}

	entry := new(pb.IprsEntry)

	entry.Value = []byte(r.val)
	typ := pb.IprsEntry_TimeRange
	entry.ValidityType = &typ
	entry.Sequence = proto.Uint64(seq)
	entry.Validity = []byte(startFmt + "~" + endFmt)

	return r.m.PublishRecord(ctx, iprsKey, entry, r.pk)
}

// ***** RangeRecordValidator ***** //
type RangeRecordValidator struct {}

func NewRangeRecordValidator() *RangeRecordValidator {
	return &RangeRecordValidator{}
}

func (v *RangeRecordValidator) SelectRecord(recs []*pb.IprsEntry, vals [][]byte) (int, error) {
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
			t, err := v.timeRange(r)
			if err != nil {
				continue
			}

			bestt, err := v.timeRange(recs[best_i])
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

func (v *RangeRecordValidator) timeRange(r *pb.IprsEntry) (*[2]*time.Time, error) {
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

// ValidateRecord verifies that the given entry is valid.
func (v *RangeRecordValidator) ValidateRecord(k string, entry *pb.IprsEntry) error {
	t, err := v.timeRange(entry)
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
