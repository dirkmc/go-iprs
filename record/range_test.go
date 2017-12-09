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

// All this is just so we can get an IprsEntry for a given sequence number
// and start/end timestamp
func setupNewRangeRecordFunc(t *testing.T) (func(uint64, *time.Time, *time.Time) *pb.IprsEntry) {
	ctx := context.Background()
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	r := mockrouting.NewServer().ClientWithDatastore(ctx, testutil.RandIdentityOrFatal(t), dstore)
	pubkManager := NewPublicKeyManager(r)
	rangeRecordManager := NewRangeRecordManager(r, pubkManager)

	// generate a key for signing the records
	sr := u.NewSeededRand(15) // generate deterministic keypair
	pk, _, err := ci.GenerateKeyPairWithReader(ci.RSA, 1024, sr)
	if err != nil {
		t.Fatal(err)
	}

	return func(seq uint64, start *time.Time, end *time.Time) *pb.IprsEntry {
		iprsKey := "/iprs/somehash"
		rangeRec, err := rangeRecordManager.NewRecord(pk, path.Path("foo"), start, end)
		if err != nil {
			t.Fatal(err)
		}
		err = rangeRec.Publish(ctx, iprsKey, seq)
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

// Simmilar to the above but just invoke the NewRecord function
// and return the record / error
func setupNewRangeRecordFuncWithError(t *testing.T) (func(uint64, *time.Time, *time.Time) (*RangeRecord, error)) {
	ctx := context.Background()
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	r := mockrouting.NewServer().ClientWithDatastore(ctx, testutil.RandIdentityOrFatal(t), dstore)
	pubkManager := NewPublicKeyManager(r)
	rangeRecordManager := NewRangeRecordManager(r, pubkManager)

	// generate a key for signing the records
	sr := u.NewSeededRand(15) // generate deterministic keypair
	pk, _, err := ci.GenerateKeyPairWithReader(ci.RSA, 1024, sr)
	if err != nil {
		t.Fatal(err)
	}

	return func(seq uint64, start *time.Time, end *time.Time) (*RangeRecord, error) {
		return rangeRecordManager.NewRecord(pk, path.Path("foo"), start, end)
	}
}


func TestNewRangeRecord(t *testing.T) {
	NewRecord := setupNewRangeRecordFuncWithError(t)

	var BeginningOfTime *time.Time
	var EndOfTime *time.Time
	ts := time.Now()
	InOneHour := ts.Add(time.Hour)
	OneHourAgo := ts.Add(time.Hour * -1)

	// Start before end OK
	_, err := NewRecord(1, &ts, &InOneHour)
	if err != nil {
		t.Fatal(err)
	}

	_, err = NewRecord(1, BeginningOfTime, &ts)
	if err != nil {
		t.Fatal(err)
	}

	_, err = NewRecord(1, &ts, EndOfTime)
	if err != nil {
		t.Fatal(err)
	}

	_, err = NewRecord(1, BeginningOfTime, EndOfTime)
	if err != nil {
		t.Fatal(err)
	}

	// End before start FAIL
	_, err = NewRecord(1, &InOneHour, &OneHourAgo)
	if err == nil {
		t.Fatal("Expected end before start error")
	}

	// Start equals end OK
	_, err = NewRecord(1, &ts, &ts)
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
	InThreeHours := ts.Add(time.Hour * 3)

	e1 := NewRecord(1, &ts, &InOneHour)
	e2 := NewRecord(2, &ts, &InOneHour)
	e3 := NewRecord(3, &ts, &InOneHour)
	e4 := NewRecord(3, &ts, &InTwoHours)
	e5 := NewRecord(4, &ts, &InThreeHours)
	e6 := NewRecord(4, &OneHourAgo, &InThreeHours)
	e7 := NewRecord(4, &OneHourAgo, EndOfTime)
	e8 := NewRecord(4, BeginningOfTime, EndOfTime)
	e9 := NewRecord(4, BeginningOfTime, EndOfTime)

	// e1 is the only record, i hope it gets this right
	assertRangeSelected(t, e1, e1)
	// e2 has the highest sequence number
	assertRangeSelected(t, e2, e1, e2)
	// e3 has the highest sequence number
	assertRangeSelected(t, e3, e1, e2, e3)
	// e4 has a higher expiration
	assertRangeSelected(t, e4, e1, e2, e3, e4)
	// e5 has the highest sequence number
	assertRangeSelected(t, e5, e1, e2, e3, e4, e5)
	// e6 has the higest expiration and lowest start date
	assertRangeSelected(t, e6, e1, e2, e3, e4, e5, e6)
	// e7 has the higest expiration and lowest start date
	assertRangeSelected(t, e7, e1, e2, e3, e4, e5, e6, e7)
	// e8 has the higest expiration and lowest start date
	assertRangeSelected(t, e8, e1, e2, e3, e4, e5, e6, e7, e8)
	// e9 should be selected as its signature will win in the comparison
	assertRangeSelected(t, e9, e1, e2, e3, e4, e5, e6, e7, e8, e9)
}

func assertRangeSelected(t *testing.T, r *pb.IprsEntry, from ...*pb.IprsEntry) {
	err := AssertSelected(NewRangeRecordChecker().SelectRecord, r, from)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRangeValidation(t *testing.T) {
	NewRecord := setupNewRangeRecordFunc(t)
	ValidateRecord := NewRangeRecordChecker().ValidateRecord

	var BeginningOfTime *time.Time
	var EndOfTime *time.Time
	ts := time.Now()
	OneHourAgo := ts.Add(time.Hour * -1)
	TwoHoursAgo := ts.Add(time.Hour * -2)
	InTwoHours := ts.Add(time.Hour * 2)
	InOneHour := ts.Add(time.Hour)

	pendingA := NewRecord(1, &TwoHoursAgo, &OneHourAgo)
	pendingB := NewRecord(1, BeginningOfTime, &OneHourAgo)
	okA := NewRecord(1, &OneHourAgo, &InOneHour)
	okB := NewRecord(1, BeginningOfTime, &InOneHour)
	okC := NewRecord(1, &OneHourAgo, EndOfTime)
	okD := NewRecord(1, BeginningOfTime, EndOfTime)
	expiredA := NewRecord(1, &InOneHour, &InTwoHours)
	expiredB := NewRecord(1, &InOneHour, EndOfTime)

	err := ValidateRecord("foo", pendingA)
	if err == nil {
		t.Fatal("Expected pending error")
	}
	err = ValidateRecord("foo2", pendingB)
	if err == nil {
		t.Fatal("Expected pending error")
	}

	err = ValidateRecord("bar", okA)
	if err != nil {
		t.Fatal(err)
	}
	err = ValidateRecord("bar2", okB)
	if err != nil {
		t.Fatal(err)
	}
	err = ValidateRecord("bar3", okC)
	if err != nil {
		t.Fatal(err)
	}
	err = ValidateRecord("bar4", okD)
	if err != nil {
		t.Fatal(err)
	}

	err = ValidateRecord("baz", expiredA)
	if err == nil {
		t.Fatal("Expected expired error")
	}
	err = ValidateRecord("baz2", expiredB)
	if err == nil {
		t.Fatal("Expected expired error")
	}
}
