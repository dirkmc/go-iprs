package iprs

import (
	"bytes"
	"context"
	"testing"
	"time"

	rec "github.com/dirkmc/go-iprs/record"
	rsv "github.com/dirkmc/go-iprs/resolver"
	tu "github.com/dirkmc/go-iprs/test"
	dstest "github.com/ipfs/go-ipfs/merkledag/test"
	b58 "github.com/mr-tron/base58/base58"
	u "gx/ipfs/QmPsAfmDBnZN3kZGSuNwvCNDZiHneERSKmRcFyG3UkvcT3/go-ipfs-util"
	ci "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
	gipld "gx/ipfs/Qmajzb6i5uwyfzcBtdqHYx94qSAMKZHBFouGV1xVbAKES9/go-ipld-git"
	ds "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore"
	dssync "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore/sync"
	testutil "gx/ipfs/QmeDA8gNhvRTsbrjEieay5wezupJDiky8xvCzDABbsGzmp/go-testutil"
	cid "gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"
	// gologging "gx/ipfs/QmQvJiADDe7JR4m968MwXobTCCzUqQkP87aRHe29MEBGHV/go-logging"
	// logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"
)

func TestPublishAndResolve(t *testing.T) {
	// logging.SetAllLoggers(gologging.DEBUG)

	ctx := context.Background()
	dag := dstest.Mock()
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	id := testutil.RandIdentityOrFatal(t)
	r := tu.NewMockValueStore(ctx, id, dstore)
	rs := NewRecordSystem(r, dag, rsv.NoCacheOpts)

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
	record, err := rec.NewRecord(validation, signer, p1.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	iprsKey, err := signer.BasePath("myrec")
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
	record, err = rec.NewRecord(validation, signer, p2.Bytes())
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
	// logging.SetAllLoggers(gologging.DEBUG)

	ctx := context.Background()
	dag := dstest.Mock()
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	id := testutil.RandIdentityOrFatal(t)
	r := tu.NewMockValueStore(ctx, id, dstore)
	rs := NewRecordSystem(r, dag, rsv.NoCacheOpts)

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
	record, err := rec.NewRecord(validation, signer, p1.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	iprsKey, err := signer.BasePath("myrec")
	if err != nil {
		t.Fatal(err)
	}

	// Publish the record
	err = rs.Publish(ctx, iprsKey, record)
	if err != nil {
		t.Fatal(err)
	}

	// Retrieve the record value with an appended path
	res, p, err := rs.Resolve(ctx, iprsKey.String()+"/my/path")
	if err != nil {
		t.Fatal(err)
	}

	// Should be the published record value
	if !res.Cid.Equals(p1) {
		t.Fatal("Got back incorrect value")
	}

	// Should be the appended path
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
	record, err = rec.NewRecord(validation, signer, p2.Bytes())
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

const gitBlob = "Ai5Bz5C6XUCZ7S2mjBUka9zd4VjbrJ1oZGxHmUw48Ar2aLoynCpcenY7CF9f8heTKcwK8a15UgiuTTcUmh5jMpLF" +
	"utMTY7qmbR8ANPM92sRJtsHXgP6okz2GGuUJwuLkBWv5SjFgutNhbGrvgsEXNNYszxAeTPDEfcHfPJpJu5tGn7skggY5yRzJoFGXj55e" +
	"VDJ9NxsVfTsaHZs1qrmPjuJeMHpXNWzd5cE2JuPsA68eRCcXc2qfNyFLjvWHmcyuxh5uTDtRbw3kx7Nkvdzkg45VrgBvyuvYv6PFcDBm" +
	"fcLoQTgtuSssbotvZr6ongYjH4tLoKTY7tP4E9WpKvvXsmodJTcJgHXyJBA4svKZpyva9aQi4KdsQZ6nZKkvWEvND6rHWGEmQ5zHf3Ef" +
	"sD2V5yPoKbe12t4ZBnv8VKac8eupqQLJsnXGx5jW78hD9s2pYR9eyh4xyZosh7VfthM26jhhSm9N6hmAWbDRv6F7pTbQkagMWQ5bkje9" +
	"jyyMNeX3ouiWg1uCVx59xEV4MB4AZVXi4RtxJL3nqJ4C9A4fcUHMUsG93VoPCMHWvkpjNRkBb2U47roPGuf17u7EbbYpKcD1UJoAU2eW" +
	"vRfCGA5y6U4vPDjf1DydWAmupgdUNMCh1TuSVeGZ65LZEJev3G1dcJsbqgKubrbRn3UX947S4LmkPd1WXiXHaiGCgLMktgxe7Tbzp5rF" +
	"vx9L1vDEwEogEndTvWt1KhS36jspGqKPom2aba"

func TestPublishAndResolveGitNode(t *testing.T) {
	// logging.SetAllLoggers(gologging.DEBUG)

	ctx := context.Background()
	dag := dstest.Mock()
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	id := testutil.RandIdentityOrFatal(t)
	r := tu.NewMockValueStore(ctx, id, dstore)
	rs := NewRecordSystem(r, dag, rsv.NoCacheOpts)

	// Generate a key for signing the records
	sr := u.NewSeededRand(15)
	pk, _, err := ci.GenerateKeyPairWithReader(ci.RSA, 1024, sr)
	if err != nil {
		t.Fatal(err)
	}

	// Create an EOL record with a GitRaw CID
	blob, err := b58.Decode(gitBlob)
	if err != nil {
		t.Fatal(err)
	}

	obj, err := gipld.ParseCompressedObject(bytes.NewReader(blob))
	if err != nil {
		t.Fatal(err)
	}

	p1 := obj.Cid()
	eol := time.Now().Add(time.Hour)
	validation := rec.NewEolRecordValidation(eol)
	signer := rec.NewKeyRecordSigner(pk)
	record, err := rec.NewRecord(validation, signer, p1.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	iprsKey, err := signer.BasePath("myrec")
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
}
