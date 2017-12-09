package recordstore_record

import (
	"context"
	"testing"
	"time"
	path "github.com/ipfs/go-ipfs/path"
	pb "github.com/dirkmc/go-iprs/pb"
	u "github.com/ipfs/go-ipfs-util"
	ci "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
	ds "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore"
	dssync "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore/sync"
	mockrouting "github.com/ipfs/go-ipfs/routing/mock"
	proto "gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
	testutil "gx/ipfs/QmQgLZP9haZheimMHqqAjJh2LhRmNfEoZDfbtkpeMhi9xK/go-testutil"
)

// All this is just so we can get an IprsEntry for a given sequence number and timestamp
func setupNewEolRecordFunc(t *testing.T) (func(uint64, time.Time) *pb.IprsEntry) {
	ctx := context.Background()
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	r := mockrouting.NewServer().ClientWithDatastore(ctx, testutil.RandIdentityOrFatal(t), dstore)
	pubkManager := NewPublicKeyManager(r)
	eolRecordManager := NewEolRecordManager(r, pubkManager)

	// generate a key for signing the records
	sr := u.NewSeededRand(15) // generate deterministic keypair
	pk, _, err := ci.GenerateKeyPairWithReader(ci.RSA, 1024, sr)
	if err != nil {
		t.Fatal(err)
	}

	return func(seq uint64, eol time.Time) *pb.IprsEntry {
		iprsKey := "/iprs/somehash"
		err := eolRecordManager.NewRecord(pk, path.Path("foo"), eol).Publish(ctx, iprsKey, seq)
		if err != nil {
			t.Fatal(err)
		}
		eBytes, err := r.GetValue(ctx, iprsKey)
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
}

func TestEolOrdering(t *testing.T) {
	NewRecord := setupNewEolRecordFunc(t)

	// select timestamp so selection is deterministic
	ts := time.Unix(1000000, 0)

	e1 := NewRecord(1, ts.Add(time.Hour))
	e2 := NewRecord(2, ts.Add(time.Hour))
	e3 := NewRecord(3, ts.Add(time.Hour))
	e4 := NewRecord(3, ts.Add(time.Hour*2))
	e5 := NewRecord(4, ts.Add(time.Hour*3))
	e6 := NewRecord(4, ts.Add(time.Hour*3))

	// e1 is the only record, i hope it gets this right
	assertEolSelected(t, e1, e1)
	// e2 has the highest sequence number
	assertEolSelected(t, e2, e1, e2)
	// e3 has the highest sequence number
	assertEolSelected(t, e3, e1, e2, e3)
	// e4 has a higher timeout
	assertEolSelected(t, e4, e1, e2, e3, e4)
	// e5 has the highest sequence number
	assertEolSelected(t, e5, e1, e2, e3, e4, e5)
	// e6 should be selected as its signauture will win in the comparison
	assertEolSelected(t, e6, e1, e2, e3, e4, e5, e6)
}

func assertEolSelected(t *testing.T, r *pb.IprsEntry, from ...*pb.IprsEntry) {
	err := AssertSelected(NewEolRecordChecker().SelectRecord, r, from)
	if err != nil {
		t.Fatal(err)
	}
}

func TestEolValidation(t *testing.T) {
	NewRecord := setupNewEolRecordFunc(t)
	ValidateRecord := NewEolRecordChecker().ValidateRecord

	ts := time.Now()

	e1 := NewRecord(1, ts.Add(time.Hour * -1))
	e2 := NewRecord(1, ts.Add(time.Hour))

	err := ValidateRecord("foo", e1)
	if err == nil {
		t.Fatal("Expected expired error")
	}

	err = ValidateRecord("foo", e2)
	if err != nil {
		t.Fatal(err)
	}
}
