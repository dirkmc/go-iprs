package iprs

import (
	"context"
//	"fmt"
	"testing"
	"time"

	cid "gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"
	rec "github.com/dirkmc/go-iprs/record"
//	rsv "github.com/dirkmc/go-iprs/resolver"
	tu "github.com/dirkmc/go-iprs/test"
	vs "github.com/dirkmc/go-iprs/vs"
//	path "github.com/ipfs/go-ipfs/path"
	u "gx/ipfs/QmPsAfmDBnZN3kZGSuNwvCNDZiHneERSKmRcFyG3UkvcT3/go-ipfs-util"
	ci "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
	ds "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore"
	dssync "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore/sync"
	dstest "github.com/ipfs/go-ipfs/merkledag/test"
	testutil "gx/ipfs/QmeDA8gNhvRTsbrjEieay5wezupJDiky8xvCzDABbsGzmp/go-testutil"
	// gologging "gx/ipfs/QmQvJiADDe7JR4m968MwXobTCCzUqQkP87aRHe29MEBGHV/go-logging"
	// logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"
)
/*
type mockResolver struct {
	entries map[string]string
}

func testResolution(t *testing.T, resolver Resolver, name string, depth int, expected string, expError error) {
	p, err := resolver.ResolveN(context.Background(), name, depth)
	if err != expError {
		t.Fatal(fmt.Errorf(
			"Expected %s with a depth of %d to have a '%s' error, but got '%s'",
			name, depth, expError, err))
	}
	if p.String() != expected {
		t.Fatal(fmt.Errorf(
			"%s with depth %d resolved to %s != %s",
			name, depth, p.String(), expected))
	}
}

func (r *mockResolver) ResolveOnce(ctx context.Context, name string) (string, error) {
	return r.entries[name], nil
}

func mockResolverOne() *mockResolver {
	return &mockResolver{
		entries: map[string]string{
			"/ipns/QmatmE9msSfkKxoffpHwNLNKgwZG8eT9Bud6YoPab52vpy": "/ipfs/Qmcqtw8FfrVSBaRmbWwHxt3AuySBhJLcvmFYi3Lbc4xnwj",
			"/ipns/QmbCMUZw6JFeZ7Wp9jkzbye3Fzp2GGcPgC3nmeUjfVF87n": "/ipns/QmatmE9msSfkKxoffpHwNLNKgwZG8eT9Bud6YoPab52vpy",
			"/iprs/QmbCMUZw6JFeZ7Wp9jkzbye3Fzp2GGcPgC3nmeUjfVF87n": "/ipns/QmatmE9msSfkKxoffpHwNLNKgwZG8eT9Bud6YoPab52vpy",
			"/ipns/QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD": "/ipns/ipfs.io",
		},
	}
}

func mockResolverTwo() *mockResolver {
	return &mockResolver{
		entries: map[string]string{
			"/ipns/ipfs.io": "/ipns/QmbCMUZw6JFeZ7Wp9jkzbye3Fzp2GGcPgC3nmeUjfVF87n",
			"/ipns/iprs.io": "/iprs/QmbCMUZw6JFeZ7Wp9jkzbye3Fzp2GGcPgC3nmeUjfVF87n",
			"/iprs/iprs.io": "/iprs/QmbCMUZw6JFeZ7Wp9jkzbye3Fzp2GGcPgC3nmeUjfVF87n",
		},
	}
}

func TestRootResolution(t *testing.T) {
	// logging.SetAllLoggers(gologging.DEBUG)

	r := &mprs{
		resolvers: map[string]rsv.Lookup{
			"dht": mockResolverOne(),
			"dns": mockResolverTwo(),
		},
	}

	const DefaultDepth = rsv.DefaultDepthLimit
	ErrRecursion := rsv.ErrResolveRecursion
	testResolution(t, r, "Qmcqtw8FfrVSBaRmbWwHxt3AuySBhJLcvmFYi3Lbc4xnwj", DefaultDepth, "/ipfs/Qmcqtw8FfrVSBaRmbWwHxt3AuySBhJLcvmFYi3Lbc4xnwj", nil)
	testResolution(t, r, "/ipns/QmatmE9msSfkKxoffpHwNLNKgwZG8eT9Bud6YoPab52vpy", DefaultDepth, "/ipfs/Qmcqtw8FfrVSBaRmbWwHxt3AuySBhJLcvmFYi3Lbc4xnwj", nil)
	testResolution(t, r, "/ipns/QmbCMUZw6JFeZ7Wp9jkzbye3Fzp2GGcPgC3nmeUjfVF87n", DefaultDepth, "/ipfs/Qmcqtw8FfrVSBaRmbWwHxt3AuySBhJLcvmFYi3Lbc4xnwj", nil)
	testResolution(t, r, "/ipns/QmbCMUZw6JFeZ7Wp9jkzbye3Fzp2GGcPgC3nmeUjfVF87n", 1, "/ipns/QmatmE9msSfkKxoffpHwNLNKgwZG8eT9Bud6YoPab52vpy", ErrRecursion)
	testResolution(t, r, "/ipns/ipfs.io", DefaultDepth, "/ipfs/Qmcqtw8FfrVSBaRmbWwHxt3AuySBhJLcvmFYi3Lbc4xnwj", nil)
	testResolution(t, r, "/ipns/ipfs.io", 1, "/ipns/QmbCMUZw6JFeZ7Wp9jkzbye3Fzp2GGcPgC3nmeUjfVF87n", ErrRecursion)
	testResolution(t, r, "/ipns/ipfs.io", 2, "/ipns/QmatmE9msSfkKxoffpHwNLNKgwZG8eT9Bud6YoPab52vpy", ErrRecursion)
	testResolution(t, r, "/ipns/iprs.io", DefaultDepth, "/ipfs/Qmcqtw8FfrVSBaRmbWwHxt3AuySBhJLcvmFYi3Lbc4xnwj", nil)
	testResolution(t, r, "/iprs/iprs.io", DefaultDepth, "/ipfs/Qmcqtw8FfrVSBaRmbWwHxt3AuySBhJLcvmFYi3Lbc4xnwj", nil)
	testResolution(t, r, "/ipns/QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD", DefaultDepth, "/ipfs/Qmcqtw8FfrVSBaRmbWwHxt3AuySBhJLcvmFYi3Lbc4xnwj", nil)
	testResolution(t, r, "/ipns/QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD", 1, "/ipns/ipfs.io", ErrRecursion)
	testResolution(t, r, "/ipns/QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD", 2, "/ipns/QmbCMUZw6JFeZ7Wp9jkzbye3Fzp2GGcPgC3nmeUjfVF87n", ErrRecursion)
	testResolution(t, r, "/ipns/QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD", 3, "/ipns/QmatmE9msSfkKxoffpHwNLNKgwZG8eT9Bud6YoPab52vpy", ErrRecursion)
}
*/
func TestPublishAndResolve(t *testing.T) {
	// logging.SetAllLoggers(gologging.DEBUG)

	ctx := context.Background()
	dag := dstest.Mock()
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	id := testutil.RandIdentityOrFatal(t)
	r := vs.NewMockValueStore(ctx, id, dstore)
	kvstore := vs.NewKadValueStore(dstore, r)
	rs := NewRecordSystem(kvstore, dag, 0)

	// Generate a key for signing the records
	sr := u.NewSeededRand(15)
	pk, _, err := ci.GenerateKeyPairWithReader(ci.RSA, 1024, sr)
	if err != nil {
		t.Fatal(err)
	}

	// Create an EOL record
	p1, err := cid.Parse("/ipfs/QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN")
	if err != nil {
		t.Fatal(err)
	}
	eol := time.Now().Add(time.Hour)
	validation := rec.NewEolRecordValidation(eol)
	signer := rec.NewKeyRecordSigner(pk)
	record, err := rec.NewRecord(validation, signer, p1)
	if err != nil {
		t.Fatal(err)
	}
	iprsKey, err := signer.BasePath()
	if err != nil {
		t.Fatal(err)
	}

	// Publish the record
	err = rs.Publish(ctx, iprsKey, record)
	if err != nil {
		t.Fatal(err)
	}

	// Retrieve the record value
	res, _, err := rs.Resolve(ctx, iprsKey.String())
	if err != nil {
		t.Fatal(err)
	}

	// Should be the published record value
	if !res.Cid.Equals(p1) {
		t.Fatal("Got back incorrect value")
	}

	// Create a new EOL record
	p2, err := cid.Parse("/ipfs/QmatmE9msSfkKxoffpHwNLNKgwZG8eT9Bud6YoPab52vpy")
	if err != nil {
		t.Fatal(err)
	}
	eol = time.Now().Add(time.Minute * 10)
	validation = rec.NewEolRecordValidation(eol)
	record, err = rec.NewRecord(validation, signer, p2)
	if err != nil {
		t.Fatal(err)
	}

	// Publish the record to the same path
	err = rs.Publish(ctx, iprsKey, record)
	if err != nil {
		t.Fatal(err)
	}

	// Retrieve the record value
	res, _, err = rs.Resolve(ctx, iprsKey.String())
	if err != nil {
		t.Fatal(err)
	}

	// Should be the newly published record value
	if !res.Cid.Equals(p2) {
		t.Fatal("Got back incorrect value")
	}
}

func TestPublishAndResolveSharedKey(t *testing.T) {
	ctx := context.Background()
	dag := dstest.Mock()
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	id := testutil.RandIdentityOrFatal(t)
	r := vs.NewMockValueStore(ctx, id, dstore)
	kvstore := vs.NewKadValueStore(dstore, r)
	rs := NewRecordSystem(kvstore, dag, 0)

	// CA Certificate
	caCert, caPk, err := tu.GenerateCACertificate("ca cert")
	if err != nil {
		t.Fatal(err)
	}
	// Child of CA Certificate
	childCert, childPk, err := tu.GenerateChildCertificate("child cert", caCert, caPk)
	if err != nil {
		t.Fatal(err)
	}

	// Create an EOL record
	p1, err := cid.Parse("/ipfs/QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN")
	if err != nil {
		t.Fatal(err)
	}
	eol := time.Now().Add(time.Hour)
	validation := rec.NewEolRecordValidation(eol)
	signer := rec.NewCertRecordSigner(caCert, caPk)
	record, err := rec.NewRecord(validation, signer, p1)
	if err != nil {
		t.Fatal(err)
	}
	iprsBasePath, err := signer.BasePath()
	if err != nil {
		t.Fatal(err)
	}

	// Publish the record
	err = rs.Publish(ctx, iprsBasePath, record)
	if err != nil {
		t.Fatal(err)
	}

	// Retrieve the record value
	res, p, err := rs.Resolve(ctx, iprsBasePath.String() + "/my/path")
	if err != nil {
		t.Fatal(err)
	}

	// Should be the published record value
	if !res.Cid.Equals(p1) {
		t.Fatal("Got back incorrect value")
	}

	if len(p) != 2 || p[0] != "my" || p[1] != "path" {
		t.Fatal("Got back incorrect path value")
	}

	// Create a new EOL record with the child certificate
	p2, err := cid.Parse("/ipfs/QmatmE9msSfkKxoffpHwNLNKgwZG8eT9Bud6YoPab52vpy")
	if err != nil {
		t.Fatal(err)
	}	
	eol = time.Now().Add(time.Minute * 10)
	validation = rec.NewEolRecordValidation(eol)
	signer = rec.NewCertRecordSigner(childCert, childPk)
	record, err = rec.NewRecord(validation, signer, p2)
	if err != nil {
		t.Fatal(err)
	}

	// Publish the record to the same path
	err = rs.Publish(ctx, iprsBasePath, record)
	if err != nil {
		t.Fatal(err)
	}

	// Retrieve the record value
	res, _, err = rs.Resolve(ctx, iprsBasePath.String())
	if err != nil {
		t.Fatal(err)
	}

	// Should be the newly published record value
	if !res.Cid.Equals(p2) {
		t.Fatal("Got back incorrect value")
	}
}
