package iprs_vs

import (
	"context"
	"testing"
	"time"

	rsp "github.com/dirkmc/go-iprs/path"
	rec "github.com/dirkmc/go-iprs/record"
	path "github.com/ipfs/go-ipfs/path"
	ds "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore"
	dssync "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore/sync"
	testutil "gx/ipfs/QmeDA8gNhvRTsbrjEieay5wezupJDiky8xvCzDABbsGzmp/go-testutil"
)

func getEolRecord(t *testing.T, p path.Path, ts time.Time, r ValueStore) (rsp.IprsPath, *rec.Record) {
	pk, _, err := testutil.RandTestKeyPair(512)
	if err != nil {
		t.Fatal(err)
	}
	factory := rec.NewRecordFactory(r)
	eolRecord := factory.NewEolKeyRecord(p, pk, ts)
	iprsKey, err := eolRecord.BasePath()
	if err != nil {
		t.Fatal(err)
	}
	return iprsKey, eolRecord
}

func TestCacheSizeZero(t *testing.T) {
	ctx := context.Background()
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	id := testutil.RandIdentityOrFatal(t)
	r := NewMockValueStore(context.Background(), id, dstore)
	ts := time.Now().Add(time.Hour)
	p := path.FromString("/ipfs/QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN")
	iprsKey, eolRecord := getEolRecord(t, p, ts, r)

	// Publish record
	eolRecord.Publish(ctx, iprsKey, 1)

	// Get the entry value (cache is size zero so it will be retrieved from routing)
	vstore := NewCachedValueStore(r, 0, nil)
	res, err := vstore.GetValue(ctx, iprsKey)
	if err != nil {
		t.Fatal(err)
	}
	if string(res) != string(p) {
		t.Fatal("Got back incorrect value")
	}

	// Remove entry from routing. Cache is size zero so retrieving
	// it should fail
	err = r.DeleteValue(iprsKey.String())
	if err != nil {
		t.Fatal(err)
	}

	// Get the entry value again
	res, err = vstore.GetValue(ctx, iprsKey)
	if err == nil {
		t.Fatal("Expected key not found error")
	}
}

func TestCacheSizeTen(t *testing.T) {
	ctx := context.Background()
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	id := testutil.RandIdentityOrFatal(t)
	r := NewMockValueStore(context.Background(), id, dstore)
	vstore := NewCachedValueStore(r, 10, nil)
	ts := time.Now().Add(time.Hour)
	p := path.FromString("/ipfs/QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN")
	iprsKey, eolRecord := getEolRecord(t, p, ts, r)

	// Publish record
	eolRecord.Publish(ctx, iprsKey, 1)

	// Get the entry value
	res, err := vstore.GetValue(ctx, iprsKey)
	if err != nil {
		t.Fatal(err)
	}
	if string(res) != string(p) {
		t.Fatal("Got back incorrect value")
	}

	// Remove entry from routing. Should still be in cache.
	err = r.DeleteValue(iprsKey.String())
	if err != nil {
		t.Fatal(err)
	}

	// Get the entry value again
	res, err = vstore.GetValue(ctx, iprsKey)
	if err != nil {
		t.Fatal(err)
	}
	if string(res) != string(p) {
		t.Fatal("Got back incorrect value")
	}
}

func TestCacheEolExpired(t *testing.T) {
	ctx := context.Background()
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	id := testutil.RandIdentityOrFatal(t)
	r := NewMockValueStore(context.Background(), id, dstore)
	vstore := NewCachedValueStore(r, 10, nil)
	ts := time.Now().Add(time.Millisecond * 100)
 	p := path.FromString("/ipfs/QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN")
	iprsKey, eolRecord := getEolRecord(t, p, ts, r)

	// Publish record
	eolRecord.Publish(ctx, iprsKey, 1)

	// Get the entry value
	res, err := vstore.GetValue(ctx, iprsKey)
	if err != nil {
		t.Fatal(err)
	}
	if string(res) != string(p) {
		t.Fatal("Got back incorrect value")
	}

	// Sleep beyond the entry's EOL
	time.Sleep(time.Millisecond * 101)

	// Remove entry from routing
	err = r.DeleteValue(iprsKey.String())
	if err != nil {
		t.Fatal(err)
	}

	// Get the entry value again. Should have expired
	res, err = vstore.GetValue(ctx, iprsKey)
	if err == nil {
		t.Fatal("Expected key not found error")
	}
}

func TestCacheTimeRangeExpired(t *testing.T) {
	ctx := context.Background()
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	id := testutil.RandIdentityOrFatal(t)
	r := NewMockValueStore(context.Background(), id, dstore)
	factory := rec.NewRecordFactory(r)
	vstore := NewCachedValueStore(r, 10, nil)

	pk, _, err := testutil.RandTestKeyPair(512)
	if err != nil {
		t.Fatal(err)
	}

	p := path.FromString("/ipfs/QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN")
	ts := time.Now()
	InTenMillis := ts.Add(time.Millisecond * 100)
	rangeRecord, err := factory.NewRangeKeyRecord(p, pk, nil, &InTenMillis)
	if err != nil {
		t.Fatal(err)
	}
	iprsKey, err := rangeRecord.BasePath()
	if err != nil {
		t.Fatal(err)
	}

	// Publish record
	rangeRecord.Publish(ctx, iprsKey, 1)

	// Get the entry value
	res, err := vstore.GetValue(ctx, iprsKey)
	if err != nil {
		t.Fatal(err)
	}
	if string(res) != string(p) {
		t.Fatal("Got back incorrect value")
	}

	// Sleep beyond the entry's EOL
	time.Sleep(time.Millisecond * 101)

	// Remove entry from routing
	err = r.DeleteValue(iprsKey.String())
	if err != nil {
		t.Fatal(err)
	}

	// Get the entry again. Should have expired
	res, err = vstore.GetValue(ctx, iprsKey)
	if err == nil {
		t.Fatal("Expected key not found error")
	}
}
