package iprs

import (
	"context"
	"testing"
	"time"

	rsv "github.com/dirkmc/go-iprs/resolver"
	tu "github.com/dirkmc/go-iprs/test"
	dstest "github.com/ipfs/go-ipfs/merkledag/test"
	namesys "github.com/ipfs/go-ipfs/namesys"
	path "github.com/ipfs/go-ipfs/path"
	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
	ds "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore"
	dssync "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore/sync"
	testutil "gx/ipfs/QmeDA8gNhvRTsbrjEieay5wezupJDiky8xvCzDABbsGzmp/go-testutil"
	cid "gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"
	// gologging "gx/ipfs/QmQvJiADDe7JR4m968MwXobTCCzUqQkP87aRHe29MEBGHV/go-logging"
	// logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"
)

func TestIpnsResolve(t *testing.T) {
	// logging.SetAllLoggers(gologging.DEBUG)
	ctx := context.Background()
	dag := dstest.Mock()
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	id := testutil.RandIdentityOrFatal(t)
	r := tu.NewMockValueStore(context.Background(), id, dstore)
	ns := namesys.NewNameSystem(r, dstore, 0)
	rs := NewRecordSystem(r, dag, rsv.NoCacheOpts)

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
	res, _, err := rs.Resolve(context.Background(), "/ipns/"+pid.Pretty())
	if err != nil {
		t.Fatal(err)
	}

	pcid, err := cid.Parse(p.String())
	if err != nil {
		t.Fatal(err)
	}
	if !res.Cid.Equals(pcid) {
		t.Fatal("Got back incorrect value", res.Cid, pcid)
	}
}
