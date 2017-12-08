package recordstore_record

import (
	"bytes"
	"context"
	"errors"
	"time"
	pb "github.com/dirkmc/go-iprs/pb"
	path "github.com/ipfs/go-ipfs/path"
	proto "gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
	routing "gx/ipfs/QmPR2JzfKd9poHx9XBhzoFeBBC31ZM3W5iUPKJZWyaoZZm/go-libp2p-routing"
	u "github.com/ipfs/go-ipfs-util"
	ci "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
)

// ***** EolRecordManager ***** //
type EolRecordManager struct {
	SignedRecordManager
}

func NewEolRecordManager(r routing.ValueStore, m *PublicKeyManager) *EolRecordManager {
	return &EolRecordManager{
		SignedRecordManager {
			routing: r,
			pubkManager: m,
		},
	}
}

func (m *EolRecordManager) NewRecord(pk ci.PrivKey, val path.Path, eol time.Time) *EolRecord {
	return &EolRecord{
		m: m,
		pk: pk,
		val: val,
		eol: eol,
	}
}

func (v *EolRecordManager) VerifyRecord(ctx context.Context, k string, entry *pb.IprsEntry) error {
	return v.CheckPublicKeySignature(ctx, k, entry)
}


// ***** EolRecord ***** //

type EolRecord struct {
	m *EolRecordManager
	pk ci.PrivKey
	val path.Path
	eol time.Time
}

func (r *EolRecord) Publish(ctx context.Context, iprsKey string, seq uint64) error {
	entry := new(pb.IprsEntry)

	entry.Value = []byte(r.val)
	typ := pb.IprsEntry_EOL
	entry.ValidityType = &typ
	entry.Sequence = proto.Uint64(seq)
	entry.Validity = []byte(u.FormatRFC3339(r.eol))

	return r.m.PublishRecord(ctx, iprsKey, entry, r.pk)
}


// ***** EolRecordValidator ***** //
type EolRecordValidator struct {}

func NewEolRecordValidator() *EolRecordValidator {
	return &EolRecordValidator{}
}

func (v *EolRecordValidator) SelectRecord(recs []*pb.IprsEntry, vals [][]byte) (int, error) {
	return EolSelectRecord(recs, vals, func(e *pb.IprsEntry) (string, error) {
		return string(e.GetValidity()), nil
	})
}

func (v *EolRecordValidator) ValidateRecord(k string, entry *pb.IprsEntry) error {
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

