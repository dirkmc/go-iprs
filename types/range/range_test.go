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
	var BeginningOfTime *time.Time
	var EndOfTime *time.Time
	ts := time.Now()
	InOneHour := ts.Add(time.Hour)
	OneHourAgo := ts.Add(time.Hour * -1)

	// generate a key for signing the records
	r := u.NewSeededRand(15) // generate deterministic keypair
	priv, _, err := ci.GenerateKeyPairWithReader(ci.RSA, 1024, r)
	if err != nil {
		t.Fatal(err)
	}

	// Start before end OK
	_, err = NewRecord(priv, path.Path("foo"), 1, &ts, &InOneHour)
	if err != nil {
		t.Fatal(err)
	}

	_, err = NewRecord(priv, path.Path("foo"), 1, BeginningOfTime, &ts)
	if err != nil {
		t.Fatal(err)
	}

	_, err = NewRecord(priv, path.Path("foo"), 1, &ts, EndOfTime)
	if err != nil {
		t.Fatal(err)
	}

	_, err = NewRecord(priv, path.Path("foo"), 1, BeginningOfTime, EndOfTime)
	if err != nil {
		t.Fatal(err)
	}

	// End before start FAIL
	_, err = NewRecord(priv, path.Path("bar"), 1, &InOneHour, &OneHourAgo)
	if err == nil {
		t.Fatal("Expected end before start error")
	}

	// Start equals end OK
	_, err = NewRecord(priv, path.Path("baz"), 1, &ts, &ts)
	if err != nil {
		t.Fatal(err)
	}
}

func TestOrdering(t *testing.T) {
	var BeginningOfTime *time.Time
	var EndOfTime *time.Time
	// select timestamp so selection is deterministic
	ts := time.Unix(1000000, 0)
	InOneHour := ts.Add(time.Hour)
	OneHourAgo := ts.Add(time.Hour * -1)
	InTwoHours := ts.Add(time.Hour * 2)
	InThreeHours := ts.Add(time.Hour * 3)

	// generate a key for signing the records
	r := u.NewSeededRand(15) // generate deterministic keypair
	priv, _, err := ci.GenerateKeyPairWithReader(ci.RSA, 1024, r)
	if err != nil {
		t.Fatal(err)
	}

	e1, err := NewRecord(priv, path.Path("foo"), 1, &ts, &InOneHour)
	if err != nil {
		t.Fatal(err)
	}

	e2, err := NewRecord(priv, path.Path("bar"), 2, &ts, &InOneHour)
	if err != nil {
		t.Fatal(err)
	}

	e3, err := NewRecord(priv, path.Path("baz"), 3, &ts, &InOneHour)
	if err != nil {
		t.Fatal(err)
	}

	e4, err := NewRecord(priv, path.Path("cat"), 3, &ts, &InTwoHours)
	if err != nil {
		t.Fatal(err)
	}

	e5, err := NewRecord(priv, path.Path("dog"), 4, &ts, &InThreeHours)
	if err != nil {
		t.Fatal(err)
	}

	e6, err := NewRecord(priv, path.Path("fish"), 4, &OneHourAgo, &InThreeHours)
	if err != nil {
		t.Fatal(err)
	}

	e7, err := NewRecord(priv, path.Path("goat"), 4, &OneHourAgo, EndOfTime)
	if err != nil {
		t.Fatal(err)
	}

	e8, err := NewRecord(priv, path.Path("hornet"), 4, BeginningOfTime, EndOfTime)
	if err != nil {
		t.Fatal(err)
	}

	e9, err := NewRecord(priv, path.Path("impala"), 4, BeginningOfTime, EndOfTime)
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

	// e7 has the higest expiration and lowest start date
	err = AssertSelected(e7, e1, e2, e3, e4, e5, e6, e7)
	if err != nil {
		t.Fatal(err)
	}

	// e8 has the higest expiration and lowest start date
	err = AssertSelected(e8, e1, e2, e3, e4, e5, e6, e7, e8)
	if err != nil {
		t.Fatal(err)
	}

	// e9 should be selected as its signauture will win in the comparison
	err = AssertSelected(e9, e1, e2, e3, e4, e5, e6, e7, e8, e9)
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidation(t *testing.T) {
	var BeginningOfTime *time.Time
	var EndOfTime *time.Time
	ts := time.Now()
	OneHourAgo := ts.Add(time.Hour * -1)
	TwoHoursAgo := ts.Add(time.Hour * -2)
	InTwoHours := ts.Add(time.Hour * 2)
	InOneHour := ts.Add(time.Hour)

	// generate a key for signing the records
	r := u.NewSeededRand(15) // generate deterministic keypair
	priv, _, err := ci.GenerateKeyPairWithReader(ci.RSA, 1024, r)
	if err != nil {
		t.Fatal(err)
	}

	pendingA, err := NewRecord(priv, path.Path("foo"), 1, &TwoHoursAgo, &OneHourAgo)
	if err != nil {
		t.Fatal(err)
	}

	pendingB, err := NewRecord(priv, path.Path("foo2"), 1, BeginningOfTime, &OneHourAgo)
	if err != nil {
		t.Fatal(err)
	}

	okA, err := NewRecord(priv, path.Path("bar"), 1, &OneHourAgo, &InOneHour)
	if err != nil {
		t.Fatal(err)
	}

	okB, err := NewRecord(priv, path.Path("bar2"), 1, BeginningOfTime, &InOneHour)
	if err != nil {
		t.Fatal(err)
	}

	okC, err := NewRecord(priv, path.Path("bar3"), 1, &OneHourAgo, EndOfTime)
	if err != nil {
		t.Fatal(err)
	}

	okD, err := NewRecord(priv, path.Path("bar4"), 1, BeginningOfTime, EndOfTime)
	if err != nil {
		t.Fatal(err)
	}

	expiredA, err := NewRecord(priv, path.Path("baz"), 1, &InOneHour, &InTwoHours)
	if err != nil {
		t.Fatal(err)
	}

	expiredB, err := NewRecord(priv, path.Path("baz2"), 1, &InOneHour, EndOfTime)
	if err != nil {
		t.Fatal(err)
	}


	err = ValidateRecord("foo", pendingA)
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

func AssertSelected(r *pb.IprsEntry, from ...*pb.IprsEntry) error {
	return types.AssertSelected(SelectRecord, r, from)
}
