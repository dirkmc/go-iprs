package iprs_publisher

import (
	"context"
	"time"

	rsp "github.com/dirkmc/go-iprs/path"
	rec "github.com/dirkmc/go-iprs/record"
	mdag "github.com/ipfs/go-ipfs/merkledag"
	routing "gx/ipfs/QmPCGUjMRuBcPybZFpjhzpifwPP9wPRoiy5geTQKU4vqWA/go-libp2p-routing"
	logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"
)

var log = logging.Logger("iprs_publisher")

const PublishTimeout = time.Second * 10

type iprsPublisher struct {
	vs  routing.ValueStore
	dag mdag.DAGService
}

// NewDHTPublisher constructs a publisher for the IPFS Routing name system.
func NewDHTPublisher(vs routing.ValueStore, dag mdag.DAGService) *iprsPublisher {
	return &iprsPublisher{vs, dag}
}

// Publish implements Publisher. Accepts an IPRS path and a record,
// and publishes it out to the routing system
func (p *iprsPublisher) Publish(ctx context.Context, iprsKey rsp.IprsPath, record *rec.Record) error {
	log.Debugf("Publish %s", iprsKey)

	timectx, cancel := context.WithTimeout(ctx, PublishTimeout)
	defer cancel()

	// Publish the record and associated nodes
	err := p.publishRecord(timectx, record)
	if err != nil {
		return err
	}

	// Update the reference to the record
	return p.publishRecordRef(timectx, iprsKey, record)
}

func (p *iprsPublisher) publishRecord(ctx context.Context, record *rec.Record) error {
	nodes := append(record.DependencyNodes(), record)
	batch := p.dag.Batch()
	for _, n := range nodes {
		if _, err := batch.Add(n); err != nil {
			return err
		}
	}
	if err := batch.Commit(); err != nil {
		return err
	}

	return nil
}

func (p *iprsPublisher) publishRecordRef(ctx context.Context, iprsKey rsp.IprsPath, record *rec.Record) error {
	log.Debugf("Updating IPRS entry %s to %s", iprsKey, record.Cid())
	return p.vs.PutValue(ctx, iprsKey.String(), record.Cid().Bytes())
}
