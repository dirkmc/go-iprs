package recordstore

import (
	"fmt"
	pb "github.com/libp2p/go-libp2p-record/pb"
)

// TODO: Move this to github.com/libp2p/interface-record-store
type RecordStore interface {
	Get(key string) ([]pb.Record, error)
	Put(key string, recordSigMultihash []byte) error
}

func main() {
	fmt.Println("go-libp2p-kad-record-store")
}
