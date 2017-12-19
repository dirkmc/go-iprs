package iprs

import (
	"context"
	"testing"
	"time"

	path "github.com/ipfs/go-ipfs/path"
	ds "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore"
	dssync "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore/sync"
	namesys "github.com/ipfs/go-ipfs/namesys"
	testutil "gx/ipfs/QmeDA8gNhvRTsbrjEieay5wezupJDiky8xvCzDABbsGzmp/go-testutil"
	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
	vs "github.com/dirkmc/go-iprs/vs"
)

func TestIpnsResolve(t *testing.T) {
	ctx := context.Background()
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	id := testutil.RandIdentityOrFatal(t)
	r := vs.NewMockValueStore(context.Background(), id, dstore)
	kvstore := vs.NewKadValueStore(dstore, r)
	ns := namesys.NewNameSystem(r, dstore, 0)
	rs := NewRecordSystem(kvstore, 0)

	pk, pubk, err := testutil.RandTestKeyPair(512)
	if err != nil {
		t.Fatal(err)
	}

	// Publish an IPNS record using IPFS namesys
	p := path.FromString("/ipfs/QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN")
	eol := time.Now().Add(time.Hour)
	err = ns.PublishWithEOL(ctx, pk, p, eol)
	if err != nil {
		t.Fatal(err)
	}

	pid, err := peer.IDFromPublicKey(pubk)
	if err != nil {
		t.Fatal(err)
	}

	// Retrieve the IPNS record value using IPRS
	res, err := rs.Resolve(context.Background(), "/ipns/" + pid.Pretty())
	if err != nil {
		t.Fatal(err)
	}

	if res != p {
		t.Fatal("Got back incorrect value")
	}
}
