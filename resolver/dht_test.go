package iprs_resolver

import (
	"context"
	//	"errors"
	"testing"
	"time"

	path "github.com/ipfs/go-ipfs/path"
	//mockrouting "github.com/ipfs/go-ipfs/routing/mock"
	testutil "gx/ipfs/QmQgLZP9haZheimMHqqAjJh2LhRmNfEoZDfbtkpeMhi9xK/go-testutil"
	ds "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore"
	dssync "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore/sync"
	//	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
	rsp "github.com/dirkmc/go-iprs/path"
	rec "github.com/dirkmc/go-iprs/record"
	psh "github.com/dirkmc/go-iprs/publisher"
	u "github.com/ipfs/go-ipfs-util"
	vs "github.com/dirkmc/go-iprs/vs"
)

func TestDHTResolve(t *testing.T) {
	ctx := context.Background()
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	id := testutil.RandIdentityOrFatal(t)
	r := vs.NewMockValueStore(context.Background(), id, dstore)
	factory := rec.NewRecordFactory(r)
	vstore := vs.NewCachedValueStore(r, 0, nil)
	kvstore := vs.NewKadValueStore(dstore, r)
	resolver := NewDHTResolver(vstore, factory)
	publisher := psh.NewDHTPublisher(psh.NewSeqManager(kvstore))

	pk, pubk, err := testutil.RandTestKeyPair(512)
	if err != nil {
		t.Fatal(err)
	}

	h := path.FromString("/ipfs/QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN")

	ts := time.Now().Add(time.Hour)
	pubkBytes, err := pubk.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	iprsKey, err := rsp.FromString("/iprs/" + u.Hash(pubkBytes).B58String())
	if err != nil {
		t.Fatal(err)
	}
	eolRecord := factory.NewEolKeyRecord(h, pk, ts.Add(time.Hour))
	publisher.Publish(ctx, iprsKey, eolRecord)
	/*
		pid, err := peer.IDFromPublicKey(pubk)
		if err != nil {
			t.Fatal(err)
		}
	*/
	//	res, err := resolver.Resolve(context.Background(), pid.Pretty())
	res, err := resolver.Resolve(context.Background(), iprsKey.String())
	if err != nil {
		t.Fatal(err)
	}

	if res != h {
		t.Fatal("Got back incorrect value.")
	}
}

/*
func TestPrexistingExpiredRecord(t *testing.T) {
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	d := mockrouting.NewServer().ClientWithDatastore(context.Background(), testutil.RandIdentityOrFatal(t), dstore)

	resolver := NewRoutingResolver(d, 0)
	publisher := NewRoutingPublisher(d, dstore)

	privk, pubk, err := testutil.RandTestKeyPair(512)
	if err != nil {
		t.Fatal(err)
	}

	id, err := peer.IDFromPublicKey(pubk)
	if err != nil {
		t.Fatal(err)
	}

	// Make an expired record and put it in the datastore
	h := path.FromString("/ipfs/QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN")
	eol := time.Now().Add(time.Hour * -1)
	err = PutRecordToRouting(context.Background(), privk, h, 0, eol, d, id)
	if err != nil {
		t.Fatal(err)
	}

	// Now, with an old record in the system already, try and publish a new one
	err = publisher.Publish(context.Background(), privk, h)
	if err != nil {
		t.Fatal(err)
	}

	err = verifyCanResolve(resolver, id.Pretty(), h)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPrexistingRecord(t *testing.T) {
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	d := mockrouting.NewServer().ClientWithDatastore(context.Background(), testutil.RandIdentityOrFatal(t), dstore)

	resolver := NewRoutingResolver(d, 0)
	publisher := NewRoutingPublisher(d, dstore)

	privk, pubk, err := testutil.RandTestKeyPair(512)
	if err != nil {
		t.Fatal(err)
	}

	id, err := peer.IDFromPublicKey(pubk)
	if err != nil {
		t.Fatal(err)
	}

	// Make a good record and put it in the datastore
	h := path.FromString("/ipfs/QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN")
	eol := time.Now().Add(time.Hour)
	err = PutRecordToRouting(context.Background(), privk, h, 0, eol, d, id)
	if err != nil {
		t.Fatal(err)
	}

	// Now, with an old record in the system already, try and publish a new one
	err = publisher.Publish(context.Background(), privk, h)
	if err != nil {
		t.Fatal(err)
	}

	err = verifyCanResolve(resolver, id.Pretty(), h)
	if err != nil {
		t.Fatal(err)
	}
}

func verifyCanResolve(r Resolver, name string, exp path.Path) error {
	res, err := r.Resolve(context.Background(), name)
	if err != nil {
		return err
	}

	if res != exp {
		return errors.New("got back wrong record!")
	}

	return nil
}
*/
