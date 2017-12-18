package iprs_vs

import (
	"context"
	"fmt"

	dshelp "github.com/ipfs/go-ipfs/thirdparty/ds-help"
	logging "github.com/ipfs/go-log"
	routing "gx/ipfs/QmPCGUjMRuBcPybZFpjhzpifwPP9wPRoiy5geTQKU4vqWA/go-libp2p-routing"
	proto "gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
	dhtpb "gx/ipfs/QmbxkgUceEcuSZ4ZdBA3x74VUDSSYjHYmmeEqkjxbtZ6Jg/go-libp2p-record/pb"
	ds "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore"
)

var log = logging.Logger("iprs.vs")

type KadValueStore struct {
	ds  ds.Datastore
	rvs routing.ValueStore
}

func NewKadValueStore(ds ds.Datastore, routing routing.ValueStore) *KadValueStore {
	return &KadValueStore{ds, routing}
}

func (vs *KadValueStore) PutValue(ctx context.Context, k string, b []byte) error {
	return vs.rvs.PutValue(ctx, k, b)
}

func (vs *KadValueStore) GetValue(ctx context.Context, k string) ([]byte, error) {
	return vs.rvs.GetValue(ctx, k)
}

func (vs *KadValueStore) GetValues(ctx context.Context, k string, n int) ([]routing.RecvdVal, error) {
	return vs.rvs.GetValues(ctx, k, n)
}

func (vs *KadValueStore) GetLocalValue(ctx context.Context, k string) ([]byte, error) {
	// Get the data out of the DHT's local datastore
	r, err := vs.ds.Get(dshelp.NewKeyFromBinary([]byte(k)))
	if err != nil {
		return nil, err
	}

	b, ok := r.([]byte)
	if !ok {
		return nil, fmt.Errorf("Unexpected type returned from datastore: %#v", r)
	}

	// Unmarshall the local data into a DHT record and get its value
	dhtrec := new(dhtpb.Record)
	err = proto.Unmarshal(b, dhtrec)
	if err != nil {
		return nil, err
	}

	return dhtrec.GetValue(), nil
}
