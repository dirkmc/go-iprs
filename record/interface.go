package recordstore_record

import (
	"context"
	pb "github.com/dirkmc/go-iprs/pb"
)

type Record interface {
	Publish(ctx context.Context, iprsKey string, seq uint64) error
}

type RecordValidator interface {
	// Validates that the record has not expired etc
	ValidateRecord(iprsKey string, entry *pb.IprsEntry) error
	// Selects the best (most valid) record
	SelectRecord(recs []*pb.IprsEntry, vals [][]byte) (int, error)
}

type RecordManager interface {
	// Verifies cryptographic signatures etc
	VerifyRecord(ctx context.Context, iprsKey string, entry *pb.IprsEntry) error
}
