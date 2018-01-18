package iprs_record

import (
	"context"
	"testing"
	"time"

	cid "gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"
	rsp "github.com/dirkmc/go-iprs/path"
	u "gx/ipfs/QmPsAfmDBnZN3kZGSuNwvCNDZiHneERSKmRcFyG3UkvcT3/go-ipfs-util"
	ci "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
)

// Helper function to simplify creating an EOL record
func setupNewEolRecordFunc(t *testing.T) func(time.Time, string) *Record {
	// generate a key for signing the records
	sr := u.NewSeededRand(15) // generate deterministic keypair
	pk, _, err := ci.GenerateKeyPairWithReader(ci.RSA, 1024, sr)
	if err != nil {
		t.Fatal(err)
	}

	return func(eol time.Time, p string) *Record {
		c, err := cid.Parse(p)
		if err != nil {
			t.Fatal(err)
		}
		vl := NewEolRecordValidation(eol)
		s := NewKeyRecordSigner(pk)
		r, err := NewRecord(vl, s, c.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
}

func TestEolOrdering(t *testing.T) {
	NewRecord := setupNewEolRecordFunc(t)

	// select timestamp so selection is deterministic
	ts := time.Unix(1000000, 0)

	p1 := "/ipfs/QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN"
	p2 := "/ipfs/QmatmE9msSfkKxoffpHwNLNKgwZG8eT9Bud6YoPab52vpy"

	r1 := NewRecord(ts.Add(time.Hour), p1)
	r2 := NewRecord(ts.Add(time.Hour*2), p2)
	r3 := NewRecord(ts.Add(time.Hour*2), p1)

	// r1 is the only record, I hope it gets this right
	assertEolSelected(t, r1, r1)

	// r2 has a higher timeout
	assertEolSelected(t, r2, r1, r2)

	// e3 should be selected as its signature will win in the comparison
	assertEolSelected(t, r3, r1, r2, r3)
}

func assertEolSelected(t *testing.T, expected *Record, from ...*Record) {
	err := AssertSelected(EolRecordChecker.SelectRecord, expected, from)
	if err != nil {
		t.Fatal(err)
	}
}

func TestEolValidation(t *testing.T) {
	ctx := context.Background()
	NewRecord := setupNewEolRecordFunc(t)
	ValidateRecord := EolRecordChecker.ValidateRecord
	iprsKey, err := rsp.FromString("/iprs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5")
	if err != nil {
		t.Fatal(err)
	}

	ts := time.Now()
	p1 := "/ipfs/QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN"

	r1 := NewRecord(ts.Add(time.Hour*-1), p1)
	r2 := NewRecord(ts.Add(time.Hour), p1)

	err = ValidateRecord(ctx, iprsKey, r1)
	if err == nil {
		t.Fatal("Expected expired error")
	}

	err = ValidateRecord(ctx, iprsKey, r2)
	if err != nil {
		t.Fatal(err)
	}
}
