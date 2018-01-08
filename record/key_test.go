package iprs_record_test

import (
	"context"
	"testing"
	"time"

	cid "gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"
	rec "github.com/dirkmc/go-iprs/record"
	psh "github.com/dirkmc/go-iprs/publisher"
	rsp "github.com/dirkmc/go-iprs/path"
	vs "github.com/dirkmc/go-iprs/vs"
	u "gx/ipfs/QmPsAfmDBnZN3kZGSuNwvCNDZiHneERSKmRcFyG3UkvcT3/go-ipfs-util"
	ci "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
	ds "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore"
	dssync "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore/sync"
	dstest "github.com/ipfs/go-ipfs/merkledag/test"
	testutil "gx/ipfs/QmeDA8gNhvRTsbrjEieay5wezupJDiky8xvCzDABbsGzmp/go-testutil"
	// gologging "github.com/whyrusleeping/go-logging"
	// logging "github.com/ipfs/go-log"
)

func TestKeyRecordVerification(t *testing.T) {
	ctx := context.Background()
	dag := dstest.Mock()
	id := testutil.RandIdentityOrFatal(t)
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	r := vs.NewMockValueStore(ctx, id, dstore)
	pubkManager := rec.NewPublicKeyManager(dag)
	verifier := rec.NewKeyRecordVerifier(pubkManager)
	publisher := psh.NewDHTPublisher(r, dag)

	// Helper function to create a record and publish it to routing
	var publishNewRecord = func(iprsKey rsp.IprsPath, pk ci.PrivKey, eol time.Time) *rec.Record {
		c, err := cid.Parse("/ipfs/QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN")
		if err != nil {
			t.Fatal(err)
		}
		vl := rec.NewEolRecordValidation(eol)
		s := rec.NewKeyRecordSigner(pk)
		rec, err := rec.NewRecord(vl, s, c)
		if err != nil {
			t.Fatal(err)
		}
		err = publisher.Publish(ctx, iprsKey, rec)
		if err != nil {
			t.Fatal(err)
		}
		return rec
	}

	// Setup: Create some keys
	sr := u.NewSeededRand(15) // generate deterministic keypair

	pk, _, err := ci.GenerateKeyPairWithReader(ci.RSA, 1024, sr)
	if err != nil {
		t.Fatal(err)
	}

	otherpk, _, err := ci.GenerateKeyPairWithReader(ci.RSA, 1024, sr)
	if err != nil {
		t.Fatal(err)
	}

	// ****** Crypto Tests ****** //
	ts := time.Now()

	// Sign record with signature
	iprsKey := getIprsPathFromKey(t, pk)
	r1 := publishNewRecord(iprsKey, pk, ts.Add(time.Hour))

	// Record is valid if the iprs path points to the cid
	// of the public key that signed the record
	// /iprs/<public key cid>
	err = verifier.VerifyRecord(ctx, iprsKey, r1)
	if err != nil {
		t.Fatal(err)
	}

	// Publish unrelated record so that it's key is available
	// on the network
	unrelatedIprsKey := getIprsPathFromKey(t, otherpk)
	publishNewRecord(unrelatedIprsKey, otherpk, ts.Add(time.Hour))
	
	// Record is not valid if the key is a different cid
	// (even though the unrelated cid is retrievable by the
	// PublicKeyManager, ie it's available on the network)
	err = verifier.VerifyRecord(ctx, unrelatedIprsKey, r1)
	if err == nil {
		t.Fatal("Failed to return error for verifification with different key")
	}

	// TODO: Use mocks to test what happens when a public key cannot
	// be retrieved from the network
}

func getIprsPathFromKey(t *testing.T, pk ci.PrivKey) rsp.IprsPath {
	s := rec.NewKeyRecordSigner(pk)
	bp, err := s.BasePath()
	if err != nil {
		t.Fatal(err)
	}
	return bp
}
