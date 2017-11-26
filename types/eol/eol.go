package recordstore_types_eol

import (
	"bytes"
	"errors"
	"time"
	pb "github.com/dirkmc/go-libp2p-kad-record-store/pb"
	path "github.com/ipfs/go-ipfs/path"
	logging "github.com/ipfs/go-log"
	proto "github.com/gogo/protobuf/proto"
	u "github.com/ipfs/go-ipfs-util"
	ci "github.com/libp2p/go-libp2p-crypto"
	types "github.com/dirkmc/go-libp2p-kad-record-store/types"
)

// ErrExpiredRecord should be returned when an Iprs record is
// invalid due to being too old
var ErrExpiredRecord = errors.New("expired record")

var log = logging.Logger("recordstore.types.eol")

func NewRecord(pk ci.PrivKey, val path.Path, seq uint64, eol time.Time) (*pb.IprsEntry, error) {
	entry := new(pb.IprsEntry)

	entry.Value = []byte(val)
	typ := pb.IprsEntry_EOL
	entry.ValidityType = &typ
	entry.Sequence = proto.Uint64(seq)
	entry.Validity = []byte(u.FormatRFC3339(eol))

	sig, err := pk.Sign(types.RecordDataForSig(entry))
	if err != nil {
		return nil, err
	}
	entry.Signature = sig
	return entry, nil
}

func SelectorFunc(k string, vals [][]byte) (int, error) {
	return SelectRecord(types.UnmarshalRecords(vals), vals)
}

func SelectRecord(recs []*pb.IprsEntry, vals [][]byte) (int, error) {
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
			rt, err := u.ParseRFC3339(string(r.GetValidity()))
			if err != nil {
				continue
			}

			bestt, err := u.ParseRFC3339(string(recs[best_i].GetValidity()))
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
		return 0, errors.New("no usable records in given set")
	}

	return best_i, nil
}

// ValidateRecord implements ValidatorFunc and verifies that the
// given 'val' is an IprsEntry and that that entry is valid.
func ValidateRecord(k string, entry *pb.IprsEntry) error {
	t, err := u.ParseRFC3339(string(entry.GetValidity()))
	if err != nil {
		log.Debug("failed parsing time for Iprs record EOL")
		return err
	}
	if time.Now().After(t) {
		return ErrExpiredRecord
	}
	return nil
}
