package iprs_publisher

import (
	"context"
	"errors"
	"time"

	rsp "github.com/dirkmc/go-iprs/path"
	r "github.com/dirkmc/go-iprs/record"
	routing "gx/ipfs/QmPR2JzfKd9poHx9XBhzoFeBBC31ZM3W5iUPKJZWyaoZZm/go-libp2p-routing"
	logging "github.com/ipfs/go-log"
)

var log = logging.Logger("iprs_publisher")

// ErrPublishFailed signals an error when attempting to publish.
var ErrPublishFailed = errors.New("Could not publish name.")

const PublishPutValTimeout = time.Minute
const DefaultRecordTTL = 24 * time.Hour

// iprsPublisher is capable of publishing and resolving names to the IPFS
// routing system.
type iprsPublisher struct {
	routing routing.ValueStore
	seqm *SeqManager
}

// NewDHTPublisher constructs a publisher for the IPFS Routing name system.
func NewDHTPublisher(route routing.ValueStore, s *SeqManager) *iprsPublisher {
	return &iprsPublisher{ route, s }
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
