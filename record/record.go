package iprs_record

import (
	"bytes"
	"context"
	"fmt"
	"time"
	logging "github.com/ipfs/go-log"
	path "github.com/ipfs/go-ipfs/path"
	pb "github.com/dirkmc/go-iprs/pb"
	proto "gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
	routing "gx/ipfs/QmPR2JzfKd9poHx9XBhzoFeBBC31ZM3W5iUPKJZWyaoZZm/go-libp2p-routing"
	rsp "github.com/dirkmc/go-iprs/path"
)

const PublishPutValTimeout = time.Second*10

var log = logging.Logger("iprs.record")

type RecordValidity interface {
	ValidityType() *pb.IprsEntry_ValidityType
	// Return the validity data for the record
	Validity() []byte
}

type RecordChecker interface {
	// Validates that the record has not expired etc
	ValidateRecord(iprsKey rsp.IprsPath, entry *pb.IprsEntry) error
	// Selects the best (most valid) record
	SelectRecord(recs []*pb.IprsEntry, vals [][]byte) (int, error)
}

type RecordSigner interface {
	VerificationType() *pb.IprsEntry_VerificationType
	// Return the verification data for the record
	Verification() []byte
	// Publish any data required for verification to the network
	// eg public key, certificate etc
	PublishVerification(ctx context.Context, iprsKey rsp.IprsPath, entry *pb.IprsEntry) error
	SignRecord(entry *pb.IprsEntry) error
}

type RecordVerifier interface {
	// Verifies cryptographic signatures etc
	VerifyRecord(ctx context.Context, iprsKey rsp.IprsPath, entry *pb.IprsEntry) error
}

type Record struct {
	routing routing.ValueStore
	vl RecordValidity
	s RecordSigner
	val path.Path
}

func NewRecord(r routing.ValueStore, vl RecordValidity, s RecordSigner, val path.Path) *Record {
	return &Record{
		routing: r,
		vl: vl,
		s: s,
		val: val,
	}
}

func (r *Record) Entry(seq uint64) (*pb.IprsEntry, error) {
	entry := new(pb.IprsEntry)

	entry.Sequence = proto.Uint64(seq)
	entry.Value = []byte(r.val)
	entry.ValidityType = r.vl.ValidityType()
	entry.Validity = r.vl.Validity()
	entry.VerificationType = r.s.VerificationType()
	entry.Verification = r.s.Verification()

	err := r.s.SignRecord(entry)
	if err != nil {
		return nil, err
	}

	return entry, nil
}

func (r *Record) Publish(ctx context.Context, iprsKey rsp.IprsPath, seq uint64) error {
	entry, err := r.Entry(seq)
	if err != nil {
		return err
	}

	// Put the verification data and the record itself to routing
	resp := make(chan error, 2)

	go func() {
		resp <- r.s.PublishVerification(ctx, iprsKey, entry)
	}()
	go func() {
		resp <- r.putEntryToRouting(ctx, iprsKey, entry)
	}()

	for i := 0; i < 2; i++ {
		err = <-resp
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Record) putEntryToRouting(ctx context.Context, iprsKey rsp.IprsPath, entry *pb.IprsEntry) error {
	data, err := proto.Marshal(entry)
	if err != nil {
		return err
	}

	timectx, cancel := context.WithTimeout(ctx, PublishPutValTimeout)
	defer cancel()

	log.Debugf("Storing iprs entry at %s", iprsKey)
	return r.routing.PutValue(timectx, iprsKey.String(), data)
}

func RecordDataForSig(r *pb.IprsEntry) []byte {
	return bytes.Join([][]byte{
		r.Value,
		[]byte(fmt.Sprint(r.GetValidityType())),
		r.Validity,
		[]byte(fmt.Sprint(r.GetVerificationType())),
		r.Verification,
	},
		[]byte{})
}