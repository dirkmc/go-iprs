package iprs_publisher

import (
	"context"
	"testing"
	"time"

	rsp "github.com/dirkmc/go-iprs/path"
	rec "github.com/dirkmc/go-iprs/record"
	vs "github.com/dirkmc/go-iprs/vs"
	u "github.com/ipfs/go-ipfs-util"
	path "github.com/ipfs/go-ipfs/path"
	ds "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore"
	dssync "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore/sync"
	testutil "gx/ipfs/QmeDA8gNhvRTsbrjEieay5wezupJDiky8xvCzDABbsGzmp/go-testutil"
)

func getEolRecord(t *testing.T, ts time.Time, r vs.ValueStore) (rsp.IprsPath, *rec.Record) {
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

func TestSeqFetch(t *testing.T) {
	ctx := context.Background()
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	id := testutil.RandIdentityOrFatal(t)
	r := vs.NewMockValueStore(context.Background(), id, dstore)
	kvs := vs.NewKadValueStore(dstore, r)
	seqm := NewSeqManager(kvs)

	ts := time.Now().Add(time.Hour)
	iprsKey, eolRecord := getEolRecord(t, ts, r)

	seq, err := seqm.GetPreviousSeqNo(ctx, iprsKey)
	if err != nil {
		t.Fatal(err)
	}
	if seq != 0 {
		t.Fatal("Sequence number on empty value store did not return 0")
	}

	// Put the entry
	eolRecord.Publish(ctx, iprsKey, 1)

	// Should now have new sequence number
	seq, err = seqm.GetPreviousSeqNo(ctx, iprsKey)
	if err != nil {
		t.Fatal(err)
	}
	if seq != 1 {
		t.Fatal("Unexpected sequence number %d", seq)
	}

	// Put the entry with a higher sequence number
	eolRecord.Publish(ctx, iprsKey, 2)

	// Should now have new sequence number
	seq, err = seqm.GetPreviousSeqNo(ctx, iprsKey)
	if err != nil {
		t.Fatal(err)
	}
	if seq != 2 {
		t.Fatal("Unexpected sequence number %d", seq)
	}

	// Simulate removing entry from local store
	r.MockEmptyLocalStore()

	// Should still retrieve it from the network
	seq, err = seqm.GetPreviousSeqNo(ctx, iprsKey)
	if err != nil {
		t.Fatal(err)
	}
	if seq != 2 {
		t.Fatal("Unexpected sequence number %d", seq)
	}

	// Remove entry from routing
	err = r.DeleteValue(iprsKey.String())
	if err != nil {
		t.Fatal(err)
	}

	// Should start again from zero
	seq, err = seqm.GetPreviousSeqNo(ctx, iprsKey)
	if err != nil {
		t.Fatal(err)
	}
	if seq != 0 {
		t.Fatal("Unexpected sequence number %d", seq)
	}
}
