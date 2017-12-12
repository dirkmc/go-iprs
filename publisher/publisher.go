package iprs_publisher

import (
	"context"

	rsp "github.com/dirkmc/go-iprs/path"
	r "github.com/dirkmc/go-iprs/record"
	logging "github.com/ipfs/go-log"
)

var log = logging.Logger("iprs_publisher")

type iprsPublisher struct {
	seqm *SeqManager
}

// NewDHTPublisher constructs a publisher for the IPFS Routing name system.
func NewDHTPublisher(s *SeqManager) *iprsPublisher {
	return &iprsPublisher{ s }
}

// Publish implements Publisher. Accepts an IPRS path and a record,
// and publishes it out to the routing system
func (p *iprsPublisher) Publish(ctx context.Context, iprsKey rsp.IprsPath, record *r.Record) error {
	log.Debugf("Publish %s", iprsKey)

	// get previous records sequence number
	seqnum, err := p.seqm.GetPreviousSeqNo(ctx, iprsKey)
	if err != nil {
		return err
	}

	// increment it
	seqnum++

	log.Debugf("Putting record with new seq no %d for %s", seqnum, iprsKey)

	return record.Publish(ctx, iprsKey, seqnum)
}
