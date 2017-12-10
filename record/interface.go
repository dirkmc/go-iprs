package recordstore_record

import (
	"context"
	pb "github.com/dirkmc/go-iprs/pb"
	rsp "github.com/dirkmc/go-iprs/path"
)

type Record interface {
	Publish(ctx context.Context, iprsKey rsp.IprsPath, seq uint64) error
}

type RecordChecker interface {
	// Validates that the record has not expired etc
	ValidateRecord(iprsKey rsp.IprsPath, entry *pb.IprsEntry) error
	// Selects the best (most valid) record
	SelectRecord(recs []*pb.IprsEntry, vals [][]byte) (int, error)
}

type RecordManager interface {
	// Verifies cryptographic signatures etc
	VerifyRecord(ctx context.Context, iprsKey rsp.IprsPath, entry *pb.IprsEntry) error
}
