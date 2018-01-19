package iprs_record

import (
	"context"
	"testing"
	"time"

	rsp "github.com/dirkmc/go-iprs/path"
	u "gx/ipfs/QmPsAfmDBnZN3kZGSuNwvCNDZiHneERSKmRcFyG3UkvcT3/go-ipfs-util"
	ci "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
	cid "gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"
)

// Helper function to simplify creating a range record
func setupNewRangeRecordFunc(t *testing.T) func(*time.Time, *time.Time, string) *Record {
	// generate a key for signing the records
	sr := u.NewSeededRand(15) // generate deterministic keypair
	pk, _, err := ci.GenerateKeyPairWithReader(ci.RSA, 1024, sr)
	if err != nil {
		t.Fatal(err)
	}

	return func(start *time.Time, end *time.Time, p string) *Record {
		c, err := cid.Parse(p)
		if err != nil {
			t.Fatal(err)
		}
		vl, err := NewRangeRecordValidation(start, end)
		if err != nil {
			t.Fatal(err)
		}
		s := NewKeyRecordSigner(pk)
		r, err := NewRecord(vl, s, c.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
}

func TestNewRangeRecordValidation(t *testing.T) {
	var BeginningOfTime *time.Time
	var EndOfTime *time.Time
	ts := time.Now()
	InOneHour := ts.Add(time.Hour)
	OneHourAgo := ts.Add(time.Hour * -1)

	// Start before end OK
	_, err := NewRangeRecordValidation(&ts, &InOneHour)
	if err != nil {
		t.Fatal(err)
	}

	_, err = NewRangeRecordValidation(BeginningOfTime, &ts)
	if err != nil {
		t.Fatal(err)
	}

	_, err = NewRangeRecordValidation(&ts, EndOfTime)
	if err != nil {
		t.Fatal(err)
	}

	_, err = NewRangeRecordValidation(BeginningOfTime, EndOfTime)
	if err != nil {
		t.Fatal(err)
	}

	// End before start FAIL
	_, err = NewRangeRecordValidation(&InOneHour, &OneHourAgo)
	if err == nil {
		t.Fatal("Expected end before start error")
	}

	// Start equals end OK
	_, err = NewRangeRecordValidation(&ts, &ts)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRangeOrdering(t *testing.T) {
	NewRecord := setupNewRangeRecordFunc(t)

	var BeginningOfTime *time.Time
	var EndOfTime *time.Time
	// select timestamp so selection is deterministic
	ts := time.Unix(1000000, 0)
	InOneHour := ts.Add(time.Hour)
	OneHourAgo := ts.Add(time.Hour * -1)
	InTwoHours := ts.Add(time.Hour * 2)

	p1 := "/ipfs/QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN"
	p2 := "/ipfs/QmatmE9msSfkKxoffpHwNLNKgwZG8eT9Bud6YoPab52vpy"

	r1 := NewRecord(&ts, &InOneHour, p1)
	r2 := NewRecord(&ts, &InTwoHours, p1)
	r3 := NewRecord(&OneHourAgo, &InTwoHours, p1)
	r4 := NewRecord(&OneHourAgo, EndOfTime, p1)
	r5 := NewRecord(BeginningOfTime, EndOfTime, p1)
	r6 := NewRecord(BeginningOfTime, EndOfTime, p2)

	// r1 is the only record, I hope it gets this right
	assertRangeSelected(t, r1, r1)
	// r2 has a higher expiration
	assertRangeSelected(t, r2, r1, r2)
	// r3 has the highest expiration and lowest start date
	assertRangeSelected(t, r3, r1, r2, r3)
	// r4 has the highest expiration and lowest start date
	assertRangeSelected(t, r4, r1, r2, r3, r4)
	// r5 has the highest expiration and lowest start date
	assertRangeSelected(t, r5, r1, r2, r3, r4, r5)
	// r6 should be selected as its signature will win in the comparison
	assertRangeSelected(t, r6, r1, r2, r3, r4, r5, r6)
}

func assertRangeSelected(t *testing.T, expected *Record, from ...*Record) {
	err := AssertSelected(RangeRecordChecker.SelectRecord, expected, from)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRangeValidation(t *testing.T) {
	ctx := context.Background()
	NewRecord := setupNewRangeRecordFunc(t)
	ValidateRecord := RangeRecordChecker.ValidateRecord

	var BeginningOfTime *time.Time
	var EndOfTime *time.Time
	ts := time.Now()
	OneHourAgo := ts.Add(time.Hour * -1)
	TwoHoursAgo := ts.Add(time.Hour * -2)
	InTwoHours := ts.Add(time.Hour * 2)
	InOneHour := ts.Add(time.Hour)

	p1 := "/ipfs/QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN"
	pendingA := NewRecord(&TwoHoursAgo, &OneHourAgo, p1)
	pendingB := NewRecord(BeginningOfTime, &OneHourAgo, p1)
	okA := NewRecord(&OneHourAgo, &InOneHour, p1)
	okB := NewRecord(BeginningOfTime, &InOneHour, p1)
	okC := NewRecord(&OneHourAgo, EndOfTime, p1)
	okD := NewRecord(BeginningOfTime, EndOfTime, p1)
	expiredA := NewRecord(&InOneHour, &InTwoHours, p1)
	expiredB := NewRecord(&InOneHour, EndOfTime, p1)

	iprsKey, err := rsp.FromString("/iprs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/myrec")
	if err != nil {
		t.Fatal(err)
	}

	err = ValidateRecord(ctx, iprsKey, pendingA)
	if err == nil {
		t.Fatal("Expected pending error")
	}
	err = ValidateRecord(ctx, iprsKey, pendingB)
	if err == nil {
		t.Fatal("Expected pending error")
	}

	err = ValidateRecord(ctx, iprsKey, okA)
	if err != nil {
		t.Fatal(err)
	}
	err = ValidateRecord(ctx, iprsKey, okB)
	if err != nil {
		t.Fatal(err)
	}
	err = ValidateRecord(ctx, iprsKey, okC)
	if err != nil {
		t.Fatal(err)
	}
	err = ValidateRecord(ctx, iprsKey, okD)
	if err != nil {
		t.Fatal(err)
	}

	err = ValidateRecord(ctx, iprsKey, expiredA)
	if err == nil {
		t.Fatal("Expected expired error")
	}
	err = ValidateRecord(ctx, iprsKey, expiredB)
	if err == nil {
		t.Fatal("Expected expired error")
	}
}
