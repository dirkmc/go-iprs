package iprs_record

import (
	"context"
	"testing"
	"time"

	rsp "github.com/dirkmc/go-iprs/path"
	pb "github.com/dirkmc/go-iprs/pb"
	u "github.com/ipfs/go-ipfs-util"
	path "github.com/ipfs/go-ipfs/path"
	mockrouting "github.com/ipfs/go-ipfs/routing/mock"
	proto "gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
	ci "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
	ds "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore"
	dssync "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore/sync"
	testutil "gx/ipfs/QmeDA8gNhvRTsbrjEieay5wezupJDiky8xvCzDABbsGzmp/go-testutil"
	// gologging "github.com/whyrusleeping/go-logging"
	// logging "github.com/ipfs/go-log"
)

func TestKeyRecordVerification(t *testing.T) {
	ctx := context.Background()
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	r := mockrouting.NewServer().ClientWithDatastore(ctx, testutil.RandIdentityOrFatal(t), dstore)
	pubkManager := NewPublicKeyManager(r)
	verifier := NewKeyRecordVerifier(pubkManager)

	// Simplifies creating a record and publishing it to routing
	NewRecord := func() func(rsp.IprsPath, ci.PrivKey, uint64, time.Time) *pb.IprsEntry {
		return func(iprsKey rsp.IprsPath, pk ci.PrivKey, seq uint64, eol time.Time) *pb.IprsEntry {
			vl := NewEolRecordValidity(eol)
			s := NewKeyRecordSigner(pubkManager, pk)
			rec := NewRecord(r, vl, s, path.Path("foo"))
			err := rec.Publish(ctx, iprsKey, seq)
			if err != nil {
				t.Fatal(err)
			}
			eBytes, err := r.GetValue(ctx, iprsKey.String())
			if err != nil {
				t.Fatal(err)
			}
			entry := new(pb.IprsEntry)
			err = proto.Unmarshal(eBytes, entry)
			if err != nil {
				t.Fatal(err)
			}
			return entry
		}
	}()

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

	// Put the unrelated key onto the network
	// so it's available to the verifier
	err = pubkManager.PutPublicKey(ctx, otherpk.GetPublic())
	if err != nil {
		t.Fatal(err)
	}

	// ****** Crypto Tests ****** //
	ts := time.Now()

	// Sign record with signature
	iprsKey := getIprsPathFromKey(t, pk)
	e1 := NewRecord(iprsKey, pk, 1, ts.Add(time.Hour))

	// Record is valid if the iprs path points to the hash
	// of the public key that signed the record
	// /iprs/<public key hash>
	err = verifier.VerifyRecord(ctx, iprsKey, e1)
	if err != nil {
		t.Fatal(err)
	}

	// Record is not valid if the key is a different hash
	// (even though the unrelated hash is retrievable by the
	// PublicKeyManager, ie it's available on the network)
	unrelatedIprsKey := getIprsPathFromKey(t, otherpk)
	err = verifier.VerifyRecord(ctx, unrelatedIprsKey, e1)
	if err == nil {
		t.Fatal("Failed to return error for verifification with different key")
	}

	// TODO: Use mocks to test what happens when a public key cannot
	// be retrieved from the network
}

func getIprsPathFromKey(t *testing.T, pk ci.PrivKey) rsp.IprsPath {
	b, err := pk.GetPublic().Bytes()
	if err != nil {
		t.Fatal(err)
	}
	iprsKeyStr := "/iprs/" + u.Hash(b).B58String()
	iprsKey, err := rsp.FromString(iprsKeyStr)
	if err != nil {
		t.Fatal(err)
	}
	return iprsKey
}
