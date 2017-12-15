package iprs_vs

import (
	"bytes"
	"context"
	"testing"
	"time"

	path "github.com/ipfs/go-ipfs/path"
	testutil "gx/ipfs/QmQgLZP9haZheimMHqqAjJh2LhRmNfEoZDfbtkpeMhi9xK/go-testutil"
	ds "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore"
	dssync "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore/sync"
	proto "gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
	rsp "github.com/dirkmc/go-iprs/path"
	rec "github.com/dirkmc/go-iprs/record"
	u "github.com/ipfs/go-ipfs-util"
)

func getEolRecord(t *testing.T, ts time.Time, r ValueStore) (rsp.IprsPath, *rec.Record) {
	pk, pubk, err := testutil.RandTestKeyPair(512)
	if err != nil {
		t.Fatal(err)
	}

	p := path.FromString("/ipfs/QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN")

	pubkBytes, err := pubk.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	iprsKey, err := rsp.FromString("/iprs/" + u.Hash(pubkBytes).B58String())
	if err != nil {
		t.Fatal(err)
	}
	factory := rec.NewRecordFactory(r)
	return iprsKey, factory.NewEolKeyRecord(p, pk, ts)
}

func TestCacheSizeZero(t *testing.T) {
	ctx := context.Background()
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	id := testutil.RandIdentityOrFatal(t)
	r := NewMockValueStore(context.Background(), id, dstore)
	vstore := NewCachedValueStore(r, 0, nil)
	ts := time.Now().Add(time.Hour)
	iprsKey, eolRecord := getEolRecord(t, ts, r)

	// Put the entry
	e, err := eolRecord.Entry(1)
	if err != nil {
		t.Fatal(err)
	}
	err = vstore.PutEntry(ctx, iprsKey, e)
	if err != nil {
		t.Fatal(err)
	}
	ebytes, err := proto.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}

	// Get the entry
	res, err := vstore.GetEntry(ctx, iprsKey)
	if err != nil {
		t.Fatal(err)
	}
	resb, err := proto.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(resb, ebytes) != 0 {
		t.Fatal("Got back incorrect value")
	}
}

func TestCacheSizeTen(t *testing.T) {
	ctx := context.Background()
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	id := testutil.RandIdentityOrFatal(t)
	r := NewMockValueStore(context.Background(), id, dstore)
	vstore := NewCachedValueStore(r, 10, nil)
	ts := time.Now().Add(time.Hour)
	iprsKey, eolRecord := getEolRecord(t, ts, r)

	// Put the entry
	e, err := eolRecord.Entry(1)
	if err != nil {
		t.Fatal(err)
	}
	err = vstore.PutEntry(ctx, iprsKey, e)
	if err != nil {
		t.Fatal(err)
	}
	ebytes, err := proto.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}

	// Get the entry
	res, err := vstore.GetEntry(ctx, iprsKey)
	if err != nil {
		t.Fatal(err)
	}
	resb, err := proto.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(resb, ebytes) != 0 {
		t.Fatal("Got back incorrect value")
	}

	// Remove entry from routing. Should still be in cache.
	err = r.DeleteValue(iprsKey.String())
	if err != nil {
		t.Fatal(err)
	}

	// Get the entry again
	res, err = vstore.GetEntry(ctx, iprsKey)
	if err != nil {
		t.Fatal(err)
	}
	resb, err = proto.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(resb, ebytes) != 0 {
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
	iprsKey, eolRecord := getEolRecord(t, ts, r)

	// Put the entry
	e, err := eolRecord.Entry(1)
	if err != nil {
		t.Fatal(err)
	}
	err = vstore.PutEntry(ctx, iprsKey, e)
	if err != nil {
		t.Fatal(err)
	}
	ebytes, err := proto.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}

	// Get the entry
	res, err := vstore.GetEntry(ctx, iprsKey)
	if err != nil {
		t.Fatal(err)
	}
	resb, err := proto.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(resb, ebytes) != 0 {
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
	res, err = vstore.GetEntry(ctx, iprsKey)
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

	pk, pubk, err := testutil.RandTestKeyPair(512)
	if err != nil {
		t.Fatal(err)
	}

	p := path.FromString("/ipfs/QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN")
	ts := time.Now()
	pubkBytes, err := pubk.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	iprsKey, err := rsp.FromString("/iprs/" + u.Hash(pubkBytes).B58String())
	if err != nil {
		t.Fatal(err)
	}
	InTenMillis := ts.Add(time.Millisecond * 100)
	rangeRecord, err := factory.NewRangeKeyRecord(p, pk, nil, &InTenMillis)
	if err != nil {
		t.Fatal(err)
	}

	// Put the entry
	e, err := rangeRecord.Entry(1)
	if err != nil {
		t.Fatal(err)
	}
	err = vstore.PutEntry(ctx, iprsKey, e)
	if err != nil {
		t.Fatal(err)
	}
	ebytes, err := proto.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}

	// Get the entry
	res, err := vstore.GetEntry(ctx, iprsKey)
	if err != nil {
		t.Fatal(err)
	}
	resb, err := proto.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(resb, ebytes) != 0 {
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
	res, err = vstore.GetEntry(ctx, iprsKey)
	if err == nil {
		t.Fatal("Expected key not found error")
	}
}
