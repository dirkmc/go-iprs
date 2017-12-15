package iprs_vs

import (
	"context"

	c "github.com/dirkmc/go-iprs/certificate"
	v "github.com/dirkmc/go-iprs/validation"
	mockrouting "github.com/ipfs/go-ipfs/routing/mock"
	record "github.com/libp2p/go-libp2p-record"
	recordpb "github.com/libp2p/go-libp2p-record/pb"
	routing "gx/ipfs/QmPR2JzfKd9poHx9XBhzoFeBBC31ZM3W5iUPKJZWyaoZZm/go-libp2p-routing"
	testutil "gx/ipfs/QmQgLZP9haZheimMHqqAjJh2LhRmNfEoZDfbtkpeMhi9xK/go-testutil"
	proto "gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
	ds "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore"
	dshelp "github.com/ipfs/go-ipfs/thirdparty/ds-help"
)

type MockValueStore struct {
	dstore    ds.Datastore
	r         routing.ValueStore
	Validator record.Validator
	Selector  record.Selector
}

func NewMockValueStore(ctx context.Context, id testutil.Identity, dstore ds.Datastore) *MockValueStore {
	serv := mockrouting.NewServer()
	r := serv.ClientWithDatastore(context.Background(), id, dstore)

	vs := &MockValueStore{
		dstore:    dstore,
		r:         r,
		Validator: make(record.Validator),
		Selector:  make(record.Selector),
	}

	vs.Validator["pk"] = record.PublicKeyValidator
	vs.Selector["pk"] = record.PublicKeySelector

	vs.Validator[c.CertType] = c.CertificateValidator
	vs.Selector[c.CertType] = c.CertificateSelector

	vs.Validator["iprs"] = v.RecordChecker.ValidChecker
	vs.Selector["iprs"] = v.RecordChecker.Selector

	return vs
}

func (m *MockValueStore) PutValue(ctx context.Context, k string, d []byte) error {
	return m.r.PutValue(ctx, k, d)
}

func (m *MockValueStore) GetLocalValue(ctx context.Context, k string) ([]byte, error) {
	return m.r.GetValue(ctx, k)
}

func (m *MockValueStore) GetValue(ctx context.Context, k string) ([]byte, error) {
	data, err := m.r.GetValue(ctx, k)
	if err != nil {
		return data, err
	}

	rec := new(recordpb.Record)
	rec.Key = proto.String(k)
	rec.Value = data
	if err = m.Validator.VerifyRecord(rec); err != nil {
		return nil, err
	}

	return data, err
}

func (m *MockValueStore) GetValues(ctx context.Context, k string, count int) ([]routing.RecvdVal, error) {
	return m.r.GetValues(ctx, k, count)
}

func (m *MockValueStore) DeleteValue(k string) error {
	return m.dstore.Delete(dshelp.NewKeyFromBinary([]byte(k)))
}
