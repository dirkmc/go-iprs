package iprs_record_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"testing"
	"time"

	c "github.com/dirkmc/go-iprs/certificate"
	rsp "github.com/dirkmc/go-iprs/path"
	psh "github.com/dirkmc/go-iprs/publisher"
	rec "github.com/dirkmc/go-iprs/record"
	tu "github.com/dirkmc/go-iprs/test"
	dstest "github.com/ipfs/go-ipfs/merkledag/test"
	ds "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore"
	dssync "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore/sync"
	testutil "gx/ipfs/QmeDA8gNhvRTsbrjEieay5wezupJDiky8xvCzDABbsGzmp/go-testutil"
	cid "gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"
	// gologging "github.com/whyrusleeping/go-logging"
	// logging "github.com/ipfs/go-log"
)

func TestCertRecordVerification(t *testing.T) {
	//	logging.SetAllLoggers(gologging.DEBUG)

	ctx := context.Background()
	dag := dstest.Mock()
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	id := testutil.RandIdentityOrFatal(t)
	r := tu.NewMockValueStore(context.Background(), id, dstore)
	certManager := c.NewCertificateManager(dag)
	verifier := rec.NewCertRecordVerifier(certManager)
	publisher := psh.NewDHTPublisher(r, dag)

	// Simplifies creating a record and publishing it to routing
	var publishNewRecord = func(iprsKey rsp.IprsPath, pk *rsa.PrivateKey, cert *x509.Certificate, eol time.Time) *rec.Record {
		c, err := cid.Parse("/ipfs/QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN")
		if err != nil {
			t.Fatal(err)
		}
		vl := rec.NewEolRecordValidation(eol)
		s := rec.NewCertRecordSigner(cert, pk)
		rec, err := rec.NewRecord(vl, s, c.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		err = publisher.Publish(ctx, iprsKey, rec)
		if err != nil {
			t.Fatal(err)
		}
		return rec
	}

	// Setup: Create a CA certificate and a child of the CA certificate

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

	// Unrelated CA Certificate
	unrelatedCaCert, unrelatedCaPk, err := tu.GenerateCACertificate("unrelated ca cert")
	if err != nil {
		t.Fatal(err)
	}

	// ****** Crypto Tests ****** //
	ts := time.Now()

	// Sign record with CA cert signature
	caCertIprsKey := getIprsPathFromCert(t, caCert, caPk, "myIprsName")
	r1 := publishNewRecord(caCertIprsKey, caPk, caCert, ts.Add(time.Hour))

	// Record is valid if the key is prefixed with the CA cert cid
	// that signed the certificate
	// /iprs/<ca cert cid>/any name
	err = verifier.VerifyRecord(ctx, caCertIprsKey, r1)
	if err != nil {
		t.Fatal(err)
	}

	// Put the unrelated certificate onto the network by publishing a record
	// with that cert
	unrelatedCaCertIprsKey := getIprsPathFromCert(t, unrelatedCaCert, unrelatedCaPk, "myIprsName")
	publishNewRecord(unrelatedCaCertIprsKey, unrelatedCaPk, unrelatedCaCert, ts.Add(time.Hour))

	// Record is not valid if the key is prefixed with a different
	// CA cert cid (even though the unrelated cert is retrievable by the
	// CertificateManager, ie it's available on the network)
	err = verifier.VerifyRecord(ctx, unrelatedCaCertIprsKey, r1)
	if err == nil {
		t.Fatal("Failed to return error for validation with different cert")
	}

	// Sign record with CA child cert signature
	childCertIprsKey := getIprsPathFromCert(t, caCert, caPk, "myDelegatedFriendsIprsName")
	r2 := publishNewRecord(childCertIprsKey, childPk, childCert, ts.Add(time.Hour))

	// Record is valid if the key is prefixed with the CA cert cid
	// that issued the signing certificate
	// /iprs/<ca (issuing) cert cid>/any name
	err = verifier.VerifyRecord(ctx, childCertIprsKey, r2)
	if err != nil {
		t.Fatal(err)
	}

	// Record is not valid if the signing key is unrelated to the cert
	unrelatedPk, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	e3 := publishNewRecord(childCertIprsKey, unrelatedPk, childCert, ts.Add(time.Hour))

	err = verifier.VerifyRecord(ctx, childCertIprsKey, e3)
	if err == nil {
		t.Fatal("Failed to return error for signature with unrelated cert")
	}
	/*
		// Create a temporary CA certificate, sign a record with it and
		// publish the record (which will publish the cert to the network as well)
		tmpCaCert, tmpPk, err := tu.GenerateCACertificate("temporary ca cert")
		if err != nil {
			t.Fatal(err)
		}
		tmpCaCertIprsKey := getIprsPathFromCert(t, tmpCaCert, "/somePath")
		e4 := NewRecord(tmpCaCertIprsKey, tmpPk, tmpCaCert, ts.Add(time.Hour))

		// Record should verify correctly
		err = verifier.VerifyRecord(ctx, tmpCaCertIprsKey, e4)
		if err != nil {
			t.Fatal(err)
		}

		// Delete the certificate from the network
		deleteFromRouting(t, r, tmpCaCert)

		// Record should now fail to verify because CA cert cannot be retrieved
		// from the network
		err = verifier.VerifyRecord(ctx, tmpCaCertIprsKey, e4)
		if err == nil {
			t.Fatal("Failed to return error for record with cert that is not available on the network")
		}

		// Create a temporary child certificate issued by the CA certificate,
		// sign a record with it and publish the record (which will publish
		// the child cert to the network as well)
		tmpChildCert, tmpChildPk, err := tu.GenerateChildCertificate("tmp child cert", caCert, caPk)
		if err != nil {
			t.Fatal(err)
		}
		tmpChildCertIprsKey := getIprsPathFromCert(t, caCert, "/somePath")
		e5 := NewRecord(tmpChildCertIprsKey, tmpChildPk, tmpChildCert, ts.Add(time.Hour))

		// Record signed with child cert should verify correctly
		err = verifier.VerifyRecord(ctx, tmpChildCertIprsKey, e5)
		if err != nil {
			t.Fatal(err)
		}

		// Delete issuing certificate from the network
		deleteFromRouting(t, r, caCert)

		// Record should now fail to verify because issuing CA cert cannot
		// be retrieved from the network
		err = verifier.VerifyRecord(ctx, tmpChildCertIprsKey, e5)
		if err == nil {
			t.Fatal("Failed to return error for record with issuing cert that is not available on the network")
		}

		// Restore issuing CA certificate to the network
		_, err = certManager.PutCertificate(ctx, caCert)
		if err != nil {
			t.Fatal(err)
		}

		// Make sure record verifies correctly now that issuing
		// cert is available
		err = verifier.VerifyRecord(ctx, tmpChildCertIprsKey, e5)
		if err != nil {
			t.Fatal(err)
		}

		// Delete child certificate from the network
		deleteFromRouting(t, r, tmpChildCert)

		// Record should now fail to verify because child CA cert cannot
		// be retrieved from the network (even though issuing cert can be
		// retrieved from the network)
		err = verifier.VerifyRecord(ctx, tmpChildCertIprsKey, e5)
		if err == nil {
			t.Fatal("Failed to return error for record with cert that is not available on the network")
		}
	*/
}

/*
func deleteFromRouting(t *testing.T, r *vs.MockValueStore, cert *x509.Certificate) {
	certHash, err := c.GetCertificateHash(cert)
	if err != nil {
		t.Fatal(err)
	}
	err = r.DeleteValue(c.GetCertPath(certHash))
	if err != nil {
		t.Fatal(err)
	}
}
*/
func getIprsPathFromCert(t *testing.T, cert *x509.Certificate, certPk *rsa.PrivateKey, id string) rsp.IprsPath {
	s := rec.NewCertRecordSigner(cert, certPk)
	bp, err := s.BasePath(id)
	if err != nil {
		t.Fatal(err)
	}
	return bp
}
