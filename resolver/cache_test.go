package iprs_resolver

import (
	"context"
	"testing"
	"time"

	rsp "github.com/dirkmc/go-iprs/path"
	psh "github.com/dirkmc/go-iprs/publisher"
	rec "github.com/dirkmc/go-iprs/record"
	tu "github.com/dirkmc/go-iprs/test"
	dstest "github.com/ipfs/go-ipfs/merkledag/test"
	routing "gx/ipfs/QmPCGUjMRuBcPybZFpjhzpifwPP9wPRoiy5geTQKU4vqWA/go-libp2p-routing"
	ds "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore"
	dssync "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore/sync"
	testutil "gx/ipfs/QmeDA8gNhvRTsbrjEieay5wezupJDiky8xvCzDABbsGzmp/go-testutil"
	cid "gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"
)

func getEolRecord(t *testing.T, c *cid.Cid, ts time.Time, r routing.ValueStore) (rsp.IprsPath, *rec.Record) {
	pk, _, err := testutil.RandTestKeyPair(512)
	if err != nil {
		t.Fatal(err)
	}
	vl := rec.NewEolRecordValidation(ts)
	s := rec.NewKeyRecordSigner(pk)
	record, err := rec.NewRecord(vl, s, c.Bytes())
	if err != nil {
		t.Fatal(err)
	}

	iprsKey, err := s.BasePath("myrec")
	if err != nil {
		t.Fatal(err)
	}
	return iprsKey, record
}

func TestCacheSizeZero(t *testing.T) {
	ctx := context.Background()
	dag := dstest.Mock()
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	id := testutil.RandIdentityOrFatal(t)
	r := tu.NewMockValueStore(context.Background(), id, dstore)
	publisher := psh.NewDHTPublisher(r, dag)

	ts := time.Now().Add(time.Hour)
	c, err := cid.Parse("/ipfs/QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN")
	if err != nil {
		t.Fatal(err)
	}
	iprsKey, eolRecord := getEolRecord(t, c, ts, r)

	// Publish record
	publisher.Publish(ctx, iprsKey, eolRecord)

	// Get the entry value (cache is size zero so it will be retrieved from routing)
	rs := NewIprsResolver(nil, r, dag, &CacheOpts{0, nil})
	res, _, err := rs.Resolve(ctx, iprsKey.String())
	if err != nil {
		t.Fatal(err)
	}
	resc, err := cid.Parse([]byte(res))
	if err != nil {
		t.Fatal(err)
	}
	if !resc.Equals(c) {
		t.Fatal("Got back incorrect value")
	}

	// Remove entry from routing. Cache is size zero so retrieving
	// it should fail
	err = r.DeleteValue(iprsKey.String())
	if err != nil {
		t.Fatal(err)
	}

	// Get the entry value again
	res, _, err = rs.Resolve(ctx, iprsKey.String())
	if err == nil {
		t.Fatal("Expected key not found error")
	}
}

func TestCacheSizeTen(t *testing.T) {
	ctx := context.Background()
	dag := dstest.Mock()
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	id := testutil.RandIdentityOrFatal(t)
	r := tu.NewMockValueStore(context.Background(), id, dstore)
	rs := NewIprsResolver(nil, r, dag, &CacheOpts{10, nil})
	publisher := psh.NewDHTPublisher(r, dag)

	ts := time.Now().Add(time.Hour)
	c, err := cid.Parse("/ipfs/QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN")
	if err != nil {
		t.Fatal(err)
	}
	iprsKey, eolRecord := getEolRecord(t, c, ts, r)

	// Publish record
	publisher.Publish(ctx, iprsKey, eolRecord)

	// Get the entry value
	res, _, err := rs.Resolve(ctx, iprsKey.String())
	if err != nil {
		t.Fatal(err)
	}
	resc, err := cid.Parse([]byte(res))
	if err != nil {
		t.Fatal(err)
	}
	if !resc.Equals(c) {
		t.Fatal("Got back incorrect value")
	}

	// Remove entry from routing. Should still be in cache.
	err = r.DeleteValue(iprsKey.String())
	if err != nil {
		t.Fatal(err)
	}

	// Get the entry value again
	res, _, err = rs.Resolve(ctx, iprsKey.String())
	if err != nil {
		t.Fatal(err)
	}
	resc, err = cid.Parse([]byte(res))
	if err != nil {
		t.Fatal(err)
	}
	if !resc.Equals(c) {
		t.Fatal("Got back incorrect value")
	}
}

func TestCacheEolExpired(t *testing.T) {
	ctx := context.Background()
	dag := dstest.Mock()
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	id := testutil.RandIdentityOrFatal(t)
	r := tu.NewMockValueStore(context.Background(), id, dstore)
	rs := NewIprsResolver(nil, r, dag, &CacheOpts{10, nil})
	publisher := psh.NewDHTPublisher(r, dag)

	ts := time.Now().Add(time.Millisecond * 100)
	c, err := cid.Parse("/ipfs/QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN")
	if err != nil {
		t.Fatal(err)
	}
	iprsKey, eolRecord := getEolRecord(t, c, ts, r)

	// Publish record
	publisher.Publish(ctx, iprsKey, eolRecord)

	// Get the entry value
	res, _, err := rs.Resolve(ctx, iprsKey.String())
	if err != nil {
		t.Fatal(err)
	}
	resc, err := cid.Parse([]byte(res))
	if err != nil {
		t.Fatal(err)
	}
	if !resc.Equals(c) {
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
	res, _, err = rs.Resolve(ctx, iprsKey.String())
	if err == nil {
		t.Fatal("Expected key not found error")
	}
}

func TestCacheTimeRangeExpired(t *testing.T) {
	ctx := context.Background()
	dag := dstest.Mock()
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	id := testutil.RandIdentityOrFatal(t)
	r := tu.NewMockValueStore(context.Background(), id, dstore)
	rs := NewIprsResolver(nil, r, dag, &CacheOpts{10, nil})
	publisher := psh.NewDHTPublisher(r, dag)

	pk, _, err := testutil.RandTestKeyPair(512)
	if err != nil {
		t.Fatal(err)
	}

	c, err := cid.Parse("/ipfs/QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN")
	if err != nil {
		t.Fatal(err)
	}
	ts := time.Now()
	end := ts.Add(time.Millisecond * 100)
	vl, err := rec.NewRangeRecordValidation(nil, &end)
	if err != nil {
		t.Fatal(err)
	}
	s := rec.NewKeyRecordSigner(pk)
	rangeRecord, err := rec.NewRecord(vl, s, c.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	iprsKey, err := s.BasePath("myrec")
	if err != nil {
		t.Fatal(err)
	}

	// Publish record
	publisher.Publish(ctx, iprsKey, rangeRecord)

	// Get the entry value
	res, _, err := rs.Resolve(ctx, iprsKey.String())
	if err != nil {
		t.Fatal(err)
	}
	resc, err := cid.Parse([]byte(res))
	if err != nil {
		t.Fatal(err)
	}
	if !resc.Equals(c) {
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
	res, _, err = rs.Resolve(ctx, iprsKey.String())
	if err == nil {
		t.Fatal("Expected key not found error")
	}
}
