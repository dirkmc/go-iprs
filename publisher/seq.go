package iprs_publisher

import (
	"context"
	"time"

	pb "github.com/dirkmc/go-iprs/pb"
	rsp "github.com/dirkmc/go-iprs/path"
	proto "gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
	vs "github.com/dirkmc/go-iprs/vs"
)

type SeqManager struct {
	vstore vs.ValueStore
}

func NewSeqManager(vstore vs.ValueStore) *SeqManager {
	return &SeqManager{ vstore }
}

func (s *SeqManager) GetPreviousSeqNo(ctx context.Context, iprsKey rsp.IprsPath) (uint64, error) {
	log.Debugf("GetPreviousSeqNo %s", iprsKey)

	var val []byte
	val, err := s.vstore.GetLocalValue(ctx, iprsKey.String())
	if err != nil {
		ctxt, cancel := context.WithTimeout(ctx, time.Second*30)
		defer cancel()

		v, err := s.vstore.GetValue(ctxt, iprsKey.String())
		if err != nil {
			// No record found in the DHT either
			log.Debugf("No previous seq num found in DHT for %s, start at 0", iprsKey)
			return 0, nil
		}
		val = v
	}

	// Unmarshall the DHT record into an IprsEntry
	e := new(pb.IprsEntry)
	err = proto.Unmarshal(val, e)
	if err != nil {
		return 0, err
	}

	return e.GetSequence(), nil
}
