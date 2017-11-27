package recordstore

import (
	pb "github.com/libp2p/go-libp2p-record/pb"
)

// TODO: Move this to github.com/libp2p/interface-record-store
type RecordStore interface {
	Get(key string) ([]pb.Record, error)
	Put(key string, recordSigMultihash []byte) error
}
