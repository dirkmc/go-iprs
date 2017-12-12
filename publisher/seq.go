package iprs_publisher

import (
	"context"
	"fmt"
	"time"

	pb "github.com/dirkmc/go-iprs/pb"
	rsp "github.com/dirkmc/go-iprs/path"
	proto "gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
	dhtpb "gx/ipfs/QmbxkgUceEcuSZ4ZdBA3x74VUDSSYjHYmmeEqkjxbtZ6Jg/go-libp2p-record/pb"
	ds "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore"
	dshelp "github.com/ipfs/go-ipfs/thirdparty/ds-help"
	routing "gx/ipfs/QmPR2JzfKd9poHx9XBhzoFeBBC31ZM3W5iUPKJZWyaoZZm/go-libp2p-routing"
)

type SeqManager struct {
	ds ds.Datastore
	routing routing.ValueStore
}

// The ValueStore is optional
// If not provided, SeqManager will only search locally for an existing
// record at that path
func NewSeqManager(ds ds.Datastore, routing routing.ValueStore) *SeqManager {
	return &SeqManager{ ds, routing }
}

func (s *SeqManager) GetPreviousSeqNo(ctx context.Context, iprsKey rsp.IprsPath) (uint64, error) {
	log.Debugf("GetPreviousSeqNo %s", iprsKey)

	// Search for the record in the local datastore
	prevrec, err := s.getLocalRecord(ctx, iprsKey)
	if err != nil {
		return 0, err
	}

	var val []byte
	// If there was nothing locally
	if prevrec == nil {
		// If we don't have a ValueStore to search, just start from zero
		if s.routing == nil {
			log.Debugf("No previous seq num found for %s, start at 0", iprsKey)
			return 0, nil
		}

		// Try and check the dht for a record
		log.Debugf("Checking DHT for seq num for %s", iprsKey)

		ctx, cancel := context.WithTimeout(ctx, time.Second*30)
		defer cancel()

		rv, err := s.routing.GetValue(ctx, iprsKey.String())
		if err != nil {
			// No record found in the DHT either
			log.Debugf("No previous seq num found in DHT for %s, start at 0", iprsKey)
			return 0, nil
		}

		val = rv
	} else {
		// Unmarshall the local data into a DHT record and get its value
		dhtrec := new(dhtpb.Record)
		err := proto.Unmarshal(prevrec, dhtrec)
		if err != nil {
			return 0, err
		}

		val = dhtrec.GetValue()
	}

	// Unmarshall the DHT record into an IprsEntry
	e := new(pb.IprsEntry)
	err = proto.Unmarshal(val, e)
	if err != nil {
		return 0, err
	}

	return e.GetSequence(), nil
}

// Get the record from local storage, returning nil if it's not found
func (s *SeqManager) getLocalRecord(ctx context.Context, iprsKey rsp.IprsPath) ([]byte, error) {
	r, err := s.ds.Get(dshelp.NewKeyFromBinary(iprsKey.Bytes()))
	if err == ds.ErrNotFound {
		return nil, nil
	}
	rbytes, ok := r.([]byte)
	if !ok {
		return nil, fmt.Errorf("unexpected type returned from datastore: %#v", r)
	}
	return rbytes, err
}
