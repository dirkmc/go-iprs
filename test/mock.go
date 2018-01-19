package iprs_test

import (
	"context"

	// c "github.com/dirkmc/go-iprs/certificate"
	ipns "github.com/ipfs/go-ipfs/namesys"
	mockrouting "github.com/ipfs/go-ipfs/routing/mock"
	dshelp "github.com/ipfs/go-ipfs/thirdparty/ds-help"
	routing "gx/ipfs/QmPCGUjMRuBcPybZFpjhzpifwPP9wPRoiy5geTQKU4vqWA/go-libp2p-routing"
	record "gx/ipfs/QmWGtsyPYEoiqTtWLpeUA2jpW4YSZgarKDD2zivYAFz7sR/go-libp2p-record"
	recordpb "gx/ipfs/QmWGtsyPYEoiqTtWLpeUA2jpW4YSZgarKDD2zivYAFz7sR/go-libp2p-record/pb"
	proto "gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
	ds "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore"
	testutil "gx/ipfs/QmeDA8gNhvRTsbrjEieay5wezupJDiky8xvCzDABbsGzmp/go-testutil"
)

type MockValueStore struct {
	dstore              ds.Datastore
	r                   routing.ValueStore
	Validator           record.Validator
	Selector            record.Selector
	mockEmptyLocalStore bool
}

func defaultValidator(k string, v []byte) error         { return nil }
func defaultSelector(k string, v [][]byte) (int, error) { return 0, nil }

func NewMockValueStore(ctx context.Context, id testutil.Identity, dstore ds.Datastore) *MockValueStore {
	serv := mockrouting.NewServer()
	r := serv.ClientWithDatastore(context.Background(), id, dstore)

	vs := &MockValueStore{
		dstore:              dstore,
		r:                   r,
		Validator:           make(record.Validator),
		Selector:            make(record.Selector),
		mockEmptyLocalStore: false,
	}

	vs.Validator["pk"] = record.PublicKeyValidator
	vs.Selector["pk"] = record.PublicKeySelector

	// vs.Validator[c.CertType] = c.CertificateValidator
	// vs.Selector[c.CertType] = c.CertificateSelector

	vs.Validator["ipns"] = ipns.IpnsRecordValidator
	vs.Selector["ipns"] = ipns.IpnsSelectorFunc

	vs.Validator["iprs"] = &record.ValidChecker{
		Func: defaultValidator,
		Sign: false,
	}
	vs.Selector["iprs"] = defaultSelector

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
