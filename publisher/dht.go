package recordstore_publisher

import (
	"context"
	"errors"
	"fmt"
	"time"

	pb "github.com/dirkmc/go-iprs/pb"
	rsp "github.com/dirkmc/go-iprs/path"
	r "github.com/dirkmc/go-iprs/record"
	routing "gx/ipfs/QmPR2JzfKd9poHx9XBhzoFeBBC31ZM3W5iUPKJZWyaoZZm/go-libp2p-routing"
	proto "gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
	dhtpb "gx/ipfs/QmbxkgUceEcuSZ4ZdBA3x74VUDSSYjHYmmeEqkjxbtZ6Jg/go-libp2p-record/pb"
	ds "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore"
	base32 "gx/ipfs/QmfVj3x4D6Jkq9SEoi5n2NmoUomLwoeiwnYz2KQa15wRw6/base32"
	logging "github.com/ipfs/go-log"
)

var log = logging.Logger("recordstore_publisher")

// ErrPublishFailed signals an error when attempting to publish.
var ErrPublishFailed = errors.New("Could not publish name.")

const PublishPutValTimeout = time.Minute
const DefaultRecordTTL = 24 * time.Hour

// iprsPublisher is capable of publishing and resolving names to the IPFS
// routing system.
type iprsPublisher struct {
	routing routing.ValueStore
	ds      ds.Datastore
}

// NewDHTPublisher constructs a publisher for the IPFS Routing name system.
func NewDHTPublisher(route routing.ValueStore, ds ds.Datastore) *iprsPublisher {
	if ds == nil {
		panic("nil datastore")
	}
	return &iprsPublisher{routing: route, ds: ds}
}

// Publish implements Publisher. Accepts an IPRS path and a record,
// and publishes it out to the routing system
func (p *iprsPublisher) Publish(ctx context.Context, iprsKey rsp.IprsPath, record *r.Record) error {
	log.Debugf("Publish %s", iprsKey)

	// get previous records sequence number
	seqnum, err := p.getPreviousSeqNo(ctx, iprsKey)
	if err != nil {
		return err
	}

	// increment it
	seqnum++

	log.Debugf("Putting record with new seq no %d for %s", seqnum, iprsKey)

	return record.Publish(ctx, iprsKey, seqnum)
}

func (p *iprsPublisher) getPreviousSeqNo(ctx context.Context, iprsKey rsp.IprsPath) (uint64, error) {
	log.Debugf("getPreviousSeqNo %s", iprsKey)
	prevrec, err := p.ds.Get(NewKeyFromBinary(iprsKey.Bytes()))
	if err != nil && err != ds.ErrNotFound {
		// None found, lets start at zero!
		log.Debugf("No previous seq no found for %s, start at 0", iprsKey)
		return 0, err
	}
	var val []byte
	if err == nil {
		prbytes, ok := prevrec.([]byte)
		if !ok {
			return 0, fmt.Errorf("unexpected type returned from datastore: %#v", prevrec)
		}
		dhtrec := new(dhtpb.Record)
		err := proto.Unmarshal(prbytes, dhtrec)
		if err != nil {
			return 0, err
		}

		val = dhtrec.GetValue()
	} else {
		log.Debugf("Checking DHT for seq no for %s", iprsKey)

		// try and check the dht for a record
		ctx, cancel := context.WithTimeout(ctx, time.Second*30)
		defer cancel()

		rv, err := p.routing.GetValue(ctx, iprsKey.String())
		if err != nil {
			log.Debugf("No previous seq no found for %s, start at 0", iprsKey)
			// no such record found, start at zero!
			return 0, nil
		}

		val = rv
	}

	e := new(pb.IprsEntry)
	err = proto.Unmarshal(val, e)
	if err != nil {
		return 0, err
	}

	return e.GetSequence(), nil
}

// TODO: Figure out gx resolution so we can import this instead of copying it
// Copied from https://github.com/ipfs/go-ipfs/blob/master/thirdparty/ds-help/key.go
func NewKeyFromBinary(rawKey []byte) ds.Key {
	buf := make([]byte, 1+base32.RawStdEncoding.EncodedLen(len(rawKey)))
	buf[0] = '/'
	base32.RawStdEncoding.Encode(buf[1:], rawKey)
	return ds.RawKey(string(buf))
}
