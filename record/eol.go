package recordstore_record

import (
	"bytes"
	"errors"
	"time"
	pb "github.com/dirkmc/go-iprs/pb"
	rsp "github.com/dirkmc/go-iprs/path"
	u "github.com/ipfs/go-ipfs-util"
)

// ErrExpiredRecord should be returned when an Iprs record is
// invalid due to being too old
var ErrExpiredRecord = errors.New("expired record")

type EolRecordValidity struct {
	eol time.Time
}

func NewEolRecordValidity(eol time.Time) *EolRecordValidity {
	return &EolRecordValidity{ eol }
}

func (v *EolRecordValidity) Validity() []byte {
	return []byte(u.FormatRFC3339(v.eol))
}

func (v *EolRecordValidity) ValidityType() *pb.IprsEntry_ValidityType {
	var t = pb.IprsEntry_EOL
	return &t
}

type eolRecordChecker struct {}

func (v *eolRecordChecker) SelectRecord(recs []*pb.IprsEntry, vals [][]byte) (int, error) {
	return EolSelectRecord(recs, vals, func(e *pb.IprsEntry) (string, error) {
		return string(e.GetValidity()), nil
	})
}

func (v *eolRecordChecker) ValidateRecord(iprsKey rsp.IprsPath, entry *pb.IprsEntry) error {
	return EolValidityCheck(string(entry.GetValidity()))
}

func EolValidityCheck(eolStr string) error {
	t, err := u.ParseRFC3339(eolStr)
	if err != nil {
		log.Warningf("Failed to parse time from IPRS record EOL [%s]", eolStr)
		return err
	}
	if time.Now().After(t) {
		return ErrExpiredRecord
	}
	return nil
}

type GetEolFunc func(*pb.IprsEntry) (string, error)

func EolSelectRecord(recs []*pb.IprsEntry, vals [][]byte, getEol GetEolFunc) (int, error) {
	var best_seq uint64
	best_i := -1

	for i, r := range recs {
		if r == nil || r.GetSequence() < best_seq {
			continue
		}

		if best_i == -1 || r.GetSequence() > best_seq {
			best_seq = r.GetSequence()
			best_i = i
		} else if r.GetSequence() == best_seq {
			eols, err := getEol(r)
			if err != nil {
				continue
			}

			rt, err := u.ParseRFC3339(eols)
			if err != nil {
				continue
			}

			beols, err := getEol(recs[best_i])
			if err != nil {
				continue
			}

			bestt, err := u.ParseRFC3339(beols)
			if err != nil {
				continue
			}

			if rt.After(bestt) {
				best_i = i
			} else if rt == bestt {
				if bytes.Compare(vals[i], vals[best_i]) > 0 {
					best_i = i
				}
			}
		}
	}
	if best_i == -1 {
		return 0, errors.New("No usable records in given record set")
	}

	return best_i, nil
}

var EolRecordChecker = &eolRecordChecker{}
