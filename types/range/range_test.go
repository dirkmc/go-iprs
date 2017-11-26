package recordstore_types_range

import (
	"testing"
	"time"
	path "github.com/ipfs/go-ipfs/path"
	pb "github.com/dirkmc/go-libp2p-kad-record-store/pb"
	types "github.com/dirkmc/go-libp2p-kad-record-store/types"
	u "github.com/ipfs/go-ipfs-util"
	ci "github.com/libp2p/go-libp2p-crypto"
)

func TestNewRecord(t *testing.T) {
	ts := time.Now()

	// generate a key for signing the records
	r := u.NewSeededRand(15) // generate deterministic keypair
	priv, _, err := ci.GenerateKeyPairWithReader(ci.RSA, 1024, r)
	if err != nil {
		t.Fatal(err)
	}

	// Start before end OK
	_, err = NewRecord(priv, path.Path("foo"), 1, ts, ts.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	// End before start FAIL
	_, err = NewRecord(priv, path.Path("bar"), 1, ts.Add(time.Hour), ts.Add(time.Hour * -1))
	if err == nil {
		t.Fatal("Expected end before start error")
	}

	// Start equals end OK
	_, err = NewRecord(priv, path.Path("baz"), 1, ts, ts)
	if err != nil {
		t.Fatal(err)
	}
}

func TestOrdering(t *testing.T) {
	// select timestamp so selection is deterministic
	ts := time.Unix(1000000, 0)

	// generate a key for signing the records
	r := u.NewSeededRand(15) // generate deterministic keypair
	priv, _, err := ci.GenerateKeyPairWithReader(ci.RSA, 1024, r)
	if err != nil {
		t.Fatal(err)
	}

	e1, err := NewRecord(priv, path.Path("foo"), 1, ts, ts.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	e2, err := NewRecord(priv, path.Path("bar"), 2, ts, ts.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	e3, err := NewRecord(priv, path.Path("baz"), 3, ts, ts.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	e4, err := NewRecord(priv, path.Path("cat"), 3, ts, ts.Add(time.Hour*2))
	if err != nil {
		t.Fatal(err)
	}

	e5, err := NewRecord(priv, path.Path("dog"), 4, ts, ts.Add(time.Hour*3))
	if err != nil {
		t.Fatal(err)
	}

	e6, err := NewRecord(priv, path.Path("fish"), 4, ts.Add(time.Hour*-1), ts.Add(time.Hour*3))
	if err != nil {
		t.Fatal(err)
	}

	e7, err := NewRecord(priv, path.Path("goat"), 4, ts.Add(time.Hour*-1), ts.Add(time.Hour*3))
	if err != nil {
		t.Fatal(err)
	}

	// e1 is the only record, i hope it gets this right
	err = AssertSelected(e1, e1)
	if err != nil {
		t.Fatal(err)
	}

	// e2 has the highest sequence number
	err = AssertSelected(e2, e1, e2)
	if err != nil {
		t.Fatal(err)
	}

	// e3 has the highest sequence number
	err = AssertSelected(e3, e1, e2, e3)
	if err != nil {
		t.Fatal(err)
	}

	// e4 has a higher expiration
	err = AssertSelected(e4, e1, e2, e3, e4)
	if err != nil {
		t.Fatal(err)
	}

	// e5 has the highest sequence number
	err = AssertSelected(e5, e1, e2, e3, e4, e5)
	if err != nil {
		t.Fatal(err)
	}

	// e6 has the higest expiration and lowest start date
	err = AssertSelected(e6, e1, e2, e3, e4, e5, e6)
	if err != nil {
		t.Fatal(err)
	}

	// e7 should be selected as its signauture will win in the comparison
	err = AssertSelected(e7, e1, e2, e3, e4, e5, e6, e7)
	if err != nil {
		t.Fatal(err)
	}

	_ = []interface{}{e1, e2, e3, e4, e5, e6, e7}
}

func TestValidation(t *testing.T) {
	ts := time.Now()

	// generate a key for signing the records
	r := u.NewSeededRand(15) // generate deterministic keypair
	priv, _, err := ci.GenerateKeyPairWithReader(ci.RSA, 1024, r)
	if err != nil {
		t.Fatal(err)
	}

	e1, err := NewRecord(priv, path.Path("foo"), 1, ts.Add(time.Hour * -2), ts.Add(time.Hour * -1))
	if err != nil {
		t.Fatal(err)
	}

	e2, err := NewRecord(priv, path.Path("bar"), 1, ts.Add(time.Hour * -1), ts.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	e3, err := NewRecord(priv, path.Path("baz"), 1, ts.Add(time.Hour * 1), ts.Add(time.Hour * 2))
	if err != nil {
		t.Fatal(err)
	}

	err = ValidateRecord("foo", e1)
	if err == nil {
		t.Fatal("Expected pending error")
	}

	err = ValidateRecord("bar", e2)
	if err != nil {
		t.Fatal(err)
	}

	err = ValidateRecord("foo", e3)
	if err == nil {
		t.Fatal("Expected expired error")
	}
}

func AssertSelected(r *pb.IprsEntry, from ...*pb.IprsEntry) error {
	return types.AssertSelected(SelectRecord, r, from)
}
