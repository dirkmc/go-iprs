package iprs_record

import (
	"testing"
	"time"
	path "github.com/ipfs/go-ipfs/path"
	pb "github.com/dirkmc/go-iprs/pb"
	u "github.com/ipfs/go-ipfs-util"
	ci "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
	rsp "github.com/dirkmc/go-iprs/path"
	tu "github.com/dirkmc/go-iprs/test"
)

// This is just so we can get an IprsEntry for a given sequence number and timestamp
func setupNewEolRecordFunc(t *testing.T) (func(uint64, time.Time) *pb.IprsEntry) {
	// generate a key for signing the records
	sr := u.NewSeededRand(15) // generate deterministic keypair
	pk, _, err := ci.GenerateKeyPairWithReader(ci.RSA, 1024, sr)
	if err != nil {
		t.Fatal(err)
	}

	f := NewRecordFactory(nil)

	return func(seq uint64, eol time.Time) *pb.IprsEntry {
		e, err := f.NewEolKeyRecord(path.Path("foo"), pk, eol).Entry(seq)
		if err != nil {
			t.Fatal(err)
		}
		return e
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
	err := tu.AssertSelected(EolRecordChecker.SelectRecord, r, from)
	if err != nil {
		t.Fatal(err)
	}
}

func TestEolValidation(t *testing.T) {
	NewRecord := setupNewEolRecordFunc(t)
	ValidateRecord := EolRecordChecker.ValidateRecord
	iprsKey, err := rsp.FromString("/iprs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5")
	if err != nil {
		t.Fatal(err)
	}

	ts := time.Now()

	e1 := NewRecord(1, ts.Add(time.Hour * -1))
	e2 := NewRecord(1, ts.Add(time.Hour))

	err = ValidateRecord(iprsKey, e1)
	if err == nil {
		t.Fatal("Expected expired error")
	}

	err = ValidateRecord(iprsKey, e2)
	if err != nil {
		t.Fatal(err)
	}
}
